package simulation

import (
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	"github.com/hung98dev/ttgqldl/backend/internal/canonical"
	"github.com/hung98dev/ttgqldl/backend/internal/domain"
)

// TestInactiveCanonicalRevisionRejected proves the version split is enforced:
// schema-v1 states stay hashable through the canonical compatibility path but can
// never be advanced, accepted into, restored as live checkpoints, or queued.
func TestInactiveCanonicalRevisionRejected(t *testing.T) {
	active, queue := queuedCheckpointFixture(t)
	legacy := active.Clone()
	legacy.Revision.SchemaVersion = domain.LegacyCanonicalSchemaVersion
	if err := legacy.Validate(); err != nil {
		t.Fatalf("historical v1 revision must remain structurally valid: %v", err)
	}

	if _, _, err := Next(legacy, Input{Kind: KindAdvanceTick}); !errors.Is(err, ErrInvalidState) {
		t.Fatalf("advance of a v1 state: %v", err)
	}
	if _, _, _, err := AcceptRoleCommands(legacy, RoleQueue{}, []RoleCommand{{
		InputSeq: 30, ActorKey: actorKey('b'), Role: domain.RoleGuard,
		TargetWave: 2, ScheduledTick: 901, AcceptedTick: AcceptedTickUnspecified,
	}}); !errors.Is(err, ErrInvalidState) {
		t.Fatalf("role admission into a v1 state: %v", err)
	}
	if _, _, _, err := AcceptVoteCommands(legacy, RoleQueue{}, []VoteCommand{{
		InputSeq: 31, ActorKey: actorKey('a'), TargetWave: 2,
		ScheduledTick: 900, Option: domain.VoteOptionA, AcceptedTick: AcceptedTickUnspecified,
	}}); !errors.Is(err, ErrInvalidState) {
		t.Fatalf("ballot admission into a v1 state: %v", err)
	}
	if _, _, _, _, err := ApplyDueRoles(legacy, queue); !errors.Is(err, ErrInvalidState) {
		t.Fatalf("due role application on a v1 state: %v", err)
	}
	if _, _, err := ApplyDueVotes(legacy, queue); !errors.Is(err, ErrInvalidState) {
		t.Fatalf("due ballot application on a v1 state: %v", err)
	}
	if _, err := BuildCheckpointEnvelope(legacy, queue); !errors.Is(err, ErrInvalidState) {
		t.Fatalf("checkpoint creation from a v1 state: %v", err)
	}
	activeEnvelope, err := BuildCheckpointEnvelope(active, queue)
	if err != nil {
		t.Fatal(err)
	}
	if err := VerifyCheckpointEnvelope(activeEnvelope, legacy, queue); err == nil {
		t.Fatal("a v2 checkpoint verified against a v1 state")
	}

	// An unknown schema is neither active nor historically verifiable.
	unknown := active.Clone()
	unknown.Revision.SchemaVersion = 3
	if err := unknown.Validate(); err == nil {
		t.Fatal("unknown canonical schema must fail domain validation")
	}
	if _, err := BuildCheckpointEnvelope(unknown, queue); err == nil {
		t.Fatal("checkpoint creation accepted an unknown canonical schema")
	}

	// The active revision still works after all rejections.
	if _, _, _, _, err := AdvanceWithRoles(active, queue, Input{Kind: KindAdvanceTick}); err != nil {
		t.Fatalf("active v2 state stopped advancing: %v", err)
	}
}

// TestCanonicalV2CheckpointRoundTrip covers the v2 envelope lifecycle: creation,
// verification, JSON round trip, restore, and the retained-XP/private-digest
// rejection paths under the active schema.
func TestCanonicalV2CheckpointRoundTrip(t *testing.T) {
	state, queue := queuedCheckpointFixture(t)
	state.XP = append(state.XP, domain.PlayerXP{ActorKey: actorKey('z'), XP: 50})
	domain.SortPlayerXP(state.XP)

	envelope, err := BuildCheckpointEnvelope(state, queue)
	if err != nil {
		t.Fatalf("v2 checkpoint creation: %v", err)
	}
	if envelope.SchemaVersion != checkpointEnvelopeVersion {
		t.Fatalf("private envelope version = %d", envelope.SchemaVersion)
	}
	if err := VerifyCheckpointEnvelope(envelope, state, queue); err != nil {
		t.Fatalf("v2 checkpoint verification: %v", err)
	}
	hashV2, err := canonical.Hash(state)
	if err != nil {
		t.Fatal(err)
	}
	legacy := state.Clone()
	legacy.Revision.SchemaVersion = domain.LegacyCanonicalSchemaVersion
	hashV1, err := canonical.Hash(legacy)
	if err != nil {
		t.Fatal(err)
	}
	if hashV1 == hashV2 {
		t.Fatal("v1 and v2 state hashes must differ")
	}

	restored := restoreCheckpointRoundTrip(t, state, queue)
	if err := VerifyCheckpointEnvelope(restored.Envelope, restored.State, restored.Queue); err != nil {
		t.Fatalf("restored v2 checkpoint: %v", err)
	}
	if differences := stateDifferences(restored.State, state); len(differences) > 0 {
		t.Fatalf("v2 checkpoint round trip changed the state: %v", differences)
	}
	if !reflect.DeepEqual(restored.Queue, queue) {
		t.Fatal("v2 checkpoint round trip changed the queue")
	}
}

func restoreCheckpointRoundTrip(t *testing.T, state domain.SessionState, queue RoleQueue) replayCheckpoint {
	t.Helper()
	envelope, err := BuildCheckpointEnvelope(state, queue)
	if err != nil {
		t.Fatal(err)
	}
	restored := replayCheckpoint{State: state.Clone(), Queue: queue.Clone(), Envelope: envelope}
	payload, err := json.Marshal(restored)
	if err != nil {
		t.Fatal(err)
	}
	var decoded replayCheckpoint
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatal(err)
	}
	decoded.State = decoded.State.Clone()
	decoded.Queue = decoded.Queue.Clone()
	return decoded
}

// TestHashingDoesNotAlterGameplay is acceptance Test F: running the same scripted
// scenario with per-step canonical hashing enabled and disabled must produce
// identical authoritative states, queues and event streams.
func TestHashingDoesNotAlterGameplay(t *testing.T) {
	scenario := scenarioDefeatRecovery(t)
	initialState, initialQueue := scenario.Initial(t)
	withoutIntegrity := runScenario(t, scenario, resumePoint{state: initialState.Clone(), queue: initialQueue.Clone()}, false)
	withIntegrity := runScenario(t, scenario, resumePoint{state: initialState.Clone(), queue: initialQueue.Clone()}, true)
	if len(withoutIntegrity) != len(withIntegrity) {
		t.Fatalf("hashing changed the step count: %d vs %d", len(withoutIntegrity), len(withIntegrity))
	}
	for index := range withoutIntegrity {
		left, right := withoutIntegrity[index], withIntegrity[index]
		if differences := stateDifferences(left.State, right.State); len(differences) > 0 {
			t.Fatalf("step %d: hashing changed the state: %v", index, differences)
		}
		if !reflect.DeepEqual(left.Queue, right.Queue) {
			t.Fatalf("step %d: hashing changed the queue", index)
		}
		if !reflect.DeepEqual(left.Events, right.Events) {
			t.Fatalf("step %d: hashing changed the event stream", index)
		}
	}
}
