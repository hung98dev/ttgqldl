package simulation

import (
	"errors"
	"reflect"
	"testing"

	"github.com/hung98dev/ttgqldl/backend/internal/canonical"
	"github.com/hung98dev/ttgqldl/backend/internal/domain"
)

func TestCappedXPStillPersistsBookkeeping(t *testing.T) {
	state := lockedAt(t, 200, domain.PhasePrep)
	state.Members = []domain.WaveMember{{ActorKey: uniqueActor(1), Role: domain.RoleBuilder, JoinedTick: 0}}
	state.XP = []domain.PlayerXP{{ActorKey: uniqueActor(1), XP: domain.CapSessionXP}}
	next, err := awardXP(state)
	if err != nil {
		t.Fatal(err)
	}
	if !next.Members[0].XPAwarded {
		t.Fatal("capped member bookkeeping was not persisted")
	}
	if next.XP[0].XP != domain.CapSessionXP {
		t.Fatalf("XP exceeded the cap: %d", next.XP[0].XP)
	}
	again, err := awardXP(next)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(again.XP, next.XP) || len(again.XP) != 1 {
		t.Fatal("repeated award changed XP")
	}
}

func TestSettlementRejectsInvalidTiming(t *testing.T) {
	for _, tick := range []int{1600, 1800, 2600, 2799} {
		state := lockedAt(t, tick, phaseForTick(t, tick))
		before := state.Clone()
		if _, _, err := settleWave(state); !errors.Is(err, ErrInvalidState) {
			t.Fatalf("tick %d accepted settlement", tick)
		}
		if !reflect.DeepEqual(state, before) {
			t.Fatalf("rejected settlement at %d mutated the state", tick)
		}
	}
	waiting := newState(t)
	if _, _, err := settleWave(waiting); !errors.Is(err, ErrInvalidState) {
		t.Fatal("waiting wave accepted settlement")
	}
}

func phaseForTick(t *testing.T, tick int) domain.Phase {
	t.Helper()
	phase, err := domain.PhaseForTick(tick)
	if err != nil {
		t.Fatalf("phase for %d: %v", tick, err)
	}
	return phase
}

func TestCrossKindInputSequenceUniqueness(t *testing.T) {
	state, queue := evenWave(t)
	state, queue = advanceWithRoles(t, state, queue, 899)

	role, roleQueue, _ := acceptRoles(t, state, queue, roleCommand(7, actorKey('b'), domain.RoleGuard, 2, 900))
	if roleQueue.Len() != 1 {
		t.Fatal("role command not queued")
	}
	if _, _, _, err := AcceptVoteCommands(role, roleQueue, []VoteCommand{ballot(7, 'a', domain.VoteOptionA, 2, 900)}); !errors.Is(err, ErrDuplicateInputSeq) {
		t.Fatalf("vote reused a role sequence: %v", err)
	}

	// Reverse order: vote first, role second.
	state2, queue2 := evenWave(t)
	state2, queue2 = advanceWithRoles(t, state2, queue2, 899)
	voteState, voteQueue, _, err := AcceptVoteCommands(state2, queue2, []VoteCommand{ballot(9, 'a', domain.VoteOptionA, 2, 900)})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, _, err := AcceptRoleCommands(voteState, voteQueue, []RoleCommand{{InputSeq: 9, ActorKey: actorKey('b'), Role: domain.RoleGuard, TargetWave: 2, ScheduledTick: 900, AcceptedTick: AcceptedTickUnspecified}}); !errors.Is(err, ErrDuplicateInputSeq) {
		t.Fatalf("role reused a vote sequence: %v", err)
	}
}

func TestFinalBoundaryEventSequence(t *testing.T) {
	state, _ := realWave(t)
	state = advanceTo(t, state, 2799)
	next, _, events, _, err := AdvanceWithRoles(state, RoleQueue{}, Input{Kind: KindAdvanceTick})
	if err != nil {
		t.Fatal(err)
	}
	if next.Wave.Result == domain.WaveResultPending {
		t.Fatal("no settlement at 2800")
	}
	// Exact observable order: boundaries, final attack, wave_result, phase change.
	types := make([]EventType, 0, len(events))
	for _, event := range events {
		types = append(types, event.Type)
	}
	want := []EventType{EventScheduleBoundary, EventScheduleBoundary, EventAttackResolved, EventWaveResult, EventPhaseChanged}
	if !reflect.DeepEqual(types, want) {
		t.Fatalf("event order = %v, want %v", types, want)
	}
	for _, event := range events {
		if event.Type == EventSettlementRequired {
			t.Fatal("completed settlement must not be followed by a settlement_required marker")
		}
	}
}

// --- deterministic replay harness -------------------------------------------

// replayScenario records the canonical hash after every transition.
type replayTrace struct {
	hashes []string
	states []domain.SessionState
	events []Event
}

func runTrace(t *testing.T, state domain.SessionState, queue RoleQueue, ticks int) replayTrace {
	t.Helper()
	trace := replayTrace{}
	for step := 0; step < ticks; step++ {
		next, nextQueue, events, _, err := AdvanceWithRoles(state, queue, Input{Kind: KindAdvanceTick})
		if err != nil {
			t.Fatalf("replay step %d at tick %d: %v", step, state.Tick(), err)
		}
		state, queue = next, nextQueue
		hash, err := canonical.Hash(state)
		if err != nil {
			t.Fatalf("hash at tick %d: %v", state.Tick(), err)
		}
		trace.hashes = append(trace.hashes, hash)
		trace.states = append(trace.states, state.Clone())
		trace.events = append(trace.events, events...)
	}
	return trace
}

func TestReplayCheckpointRestartMatchesUninterruptedRun(t *testing.T) {
	const checkpoints = 110
	positions := 0
	for scenario := 0; scenario < 2; scenario++ {
		state, queue := realWave(t)
		if scenario > 0 {
			state.Wave.Number = 4 + int64(scenario*2)
			// Keep the fixture valid: synthetic wave jump with no vote state.
			state.Difficulty = domain.Difficulty{Level: 1}
		}
		full := runTrace(t, state, queue, 60)
		for position := 5; position < 60 && positions < checkpoints; position++ {
			// Reconstruct by replaying the prefix, then resume from the captured
			// state and queue (both deep copies, not aliases).
			replayState, replayQueue := state.Clone(), queue.Clone()
			for step := 0; step < position; step++ {
				next, nextQueue, _, _, err := AdvanceWithRoles(replayState, replayQueue, Input{Kind: KindAdvanceTick})
				if err != nil {
					t.Fatalf("prefix %d: %v", step, err)
				}
				replayState, replayQueue = next, nextQueue
			}
			if replayState.Tick() != state.Tick()+position {
				t.Fatalf("prefix ended at %d, want %d", replayState.Tick(), state.Tick()+position)
			}
			checkpointHash, err := canonical.Hash(replayState)
			if err != nil {
				t.Fatal(err)
			}
			if checkpointHash != full.hashes[position-1] {
				t.Fatalf("checkpoint %d hash mismatch", position)
			}
			resumed := runTrace(t, replayState.Clone(), replayQueue.Clone(), 60-position)
			for offset, hash := range resumed.hashes {
				if hash != full.hashes[position+offset] {
					t.Fatalf("resume mismatch at checkpoint %d offset %d", position, offset)
				}
			}
			if !reflect.DeepEqual(resumed.states[len(resumed.states)-1], full.states[len(full.states)-1]) {
				t.Fatalf("final state mismatch after resume at %d", position)
			}
			positions++
		}
	}
	if positions < checkpoints {
		t.Fatalf("only %d replay checkpoints executed", positions)
	}
}

func TestReplayRejectsCorruptedCheckpointHash(t *testing.T) {
	state, _ := realWave(t)
	state = advanceTo(t, state, 25)
	hash, err := canonical.Hash(state)
	if err != nil {
		t.Fatal(err)
	}
	corrupted := state.Clone()
	corrupted.Village.Resources.Wood++
	otherHash, err := canonical.Hash(corrupted)
	if err != nil {
		t.Fatal(err)
	}
	if otherHash == hash {
		t.Fatal("corrupted state produced the same hash")
	}
	// Rules/schema corruption fails closed.
	bad := state.Clone()
	bad.Revision.RulesVersion = "ttg-rules-v2"
	if _, err := canonical.Hash(bad); err == nil {
		t.Fatal("foreign rules version was hashed")
	}
}
