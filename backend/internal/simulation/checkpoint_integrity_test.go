package simulation

import (
	"errors"
	"testing"

	"github.com/hung98dev/ttgqldl/backend/internal/canonical"
	"github.com/hung98dev/ttgqldl/backend/internal/domain"
)

func queuedCheckpointFixture(t *testing.T) (domain.SessionState, RoleQueue) {
	t.Helper()
	state, queue := evenWave(t)
	state, queue = advanceWithRoles(t, state, queue, 899)
	accepted, queue, _, err := AcceptRoleCommands(state, queue, []RoleCommand{{
		InputSeq: 11, ActorKey: actorKey('b'), Role: domain.RoleGuard,
		TargetWave: 2, ScheduledTick: 900, AcceptedTick: AcceptedTickUnspecified,
	}})
	if err != nil {
		t.Fatal(err)
	}
	state, queue, _, err = AcceptVoteCommands(accepted, queue, []VoteCommand{{
		InputSeq: 12, ActorKey: actorKey('a'), TargetWave: 2,
		ScheduledTick: 900, Option: domain.VoteOptionB, AcceptedTick: AcceptedTickUnspecified,
	}})
	if err != nil {
		t.Fatal(err)
	}
	return state, queue
}

func TestCheckpointEnvelopeDetectsQueueTampering(t *testing.T) {
	state, queue := queuedCheckpointFixture(t)
	envelope, err := BuildCheckpointEnvelope(state, queue)
	if err != nil {
		t.Fatal(err)
	}
	if err := VerifyCheckpointEnvelope(envelope, state, queue); err != nil {
		t.Fatalf("valid checkpoint rejected: %v", err)
	}

	tampered := envelope
	tampered.RoleQueue = append([]RoleCommand(nil), envelope.RoleQueue...)
	tampered.RoleQueue[0].Role = domain.RoleFighter
	if err := VerifyCheckpointEnvelope(tampered, state, queue); !errors.Is(err, ErrInvalidState) {
		t.Fatalf("tampered role was accepted: %v", err)
	}

	tampered = envelope
	tampered.RoleQueue = append([]RoleCommand(nil), envelope.RoleQueue...)
	tampered.RoleQueue[0].ScheduledTick = 950
	if err := VerifyCheckpointEnvelope(tampered, state, queue); err == nil {
		t.Fatal("tampered scheduled tick was accepted")
	}

	tampered = envelope
	tampered.BallotQueue = append([]VoteCommand(nil), envelope.BallotQueue...)
	tampered.BallotQueue[0].Option = domain.VoteOptionC
	if err := VerifyCheckpointEnvelope(tampered, state, queue); err == nil {
		t.Fatal("tampered ballot was accepted")
	}

	tampered = envelope
	tampered.RoleQueue = envelope.RoleQueue[:0]
	if err := VerifyCheckpointEnvelope(tampered, state, queue); err == nil {
		t.Fatal("missing queued command was accepted")
	}

	tampered = envelope
	tampered.IntegrityHash = "deadbeef"
	if err := VerifyCheckpointEnvelope(tampered, state, queue); err == nil {
		t.Fatal("tampered integrity hash was accepted")
	}

	tampered = envelope
	tampered.RulesVersion = "ttg-rules-v2"
	if err := VerifyCheckpointEnvelope(tampered, state, queue); err == nil {
		t.Fatal("foreign rules version was accepted")
	}

	// The same state with a different pending command must not verify against the
	// original envelope.
	otherQueue := queue.Clone()
	otherQueue.Entries[0].Role = domain.RoleFighter
	if err := VerifyCheckpointEnvelope(envelope, state, otherQueue); err == nil {
		t.Fatal("restored queue mismatch was accepted")
	}
}
func TestCheckpointDetectsRetainedXPTampering(t *testing.T) {
	state, queue := queuedCheckpointFixture(t)
	// A retired actor keeps session XP but is not a current-wave member; canonical
	// v1 does not hash this record.
	state.XP = append(state.XP, domain.PlayerXP{ActorKey: actorKey('z'), XP: 50})
	domain.SortPlayerXP(state.XP)
	envelope, err := BuildCheckpointEnvelope(state, queue)
	if err != nil {
		t.Fatal(err)
	}
	canonicalBefore, err := canonical.Hash(state)
	if err != nil {
		t.Fatal(err)
	}
	tampered := state.Clone()
	for i := range tampered.XP {
		if tampered.XP[i].ActorKey == actorKey('z') {
			tampered.XP[i].XP = 60
		}
	}
	canonicalAfter, err := canonical.Hash(tampered)
	if err != nil {
		t.Fatal(err)
	}
	if canonicalBefore != canonicalAfter {
		t.Fatal("canonical v1 unexpectedly covers non-member XP; update the contract notes")
	}
	if err := VerifyCheckpointEnvelope(envelope, tampered, queue); err == nil {
		t.Fatal("private checkpoint did not detect retained-XP tampering")
	}
	// An attacker who also edits the envelope copy still fails the digest.
	rewritten := envelope
	rewritten.SessionXP = append([]domain.PlayerXP(nil), envelope.SessionXP...)
	for i := range rewritten.SessionXP {
		if rewritten.SessionXP[i].ActorKey == actorKey('z') {
			rewritten.SessionXP[i].XP = 60
		}
	}
	if err := VerifyCheckpointEnvelope(rewritten, tampered, queue); err == nil {
		t.Fatal("tampered envelope passed its integrity digest")
	}
}
