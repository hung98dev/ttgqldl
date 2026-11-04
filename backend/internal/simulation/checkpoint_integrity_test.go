package simulation

import (
	"encoding/json"
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

// reseal recomputes the private digest for a deliberately tampered payload, so a
// structural test proves the queue contract is enforced independently of the
// checksum.
func reseal(t *testing.T, envelope CheckpointEnvelope, state domain.SessionState, queue RoleQueue) CheckpointEnvelope {
	t.Helper()
	digest, err := privateRestorationDigest(state, queue)
	if err != nil {
		t.Fatal(err)
	}
	envelope.PrivateDigest = digest
	return envelope
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
	tampered.PrivateDigest = "deadbeef"
	if err := VerifyCheckpointEnvelope(tampered, state, queue); err == nil {
		t.Fatal("tampered private digest was accepted")
	}

	tampered = envelope
	tampered.StateHash = "deadbeef"
	if err := VerifyCheckpointEnvelope(tampered, state, queue); err == nil {
		t.Fatal("tampered canonical state hash was accepted")
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

// TestCheckpointRejectsMalformedQueuedCommands rewrites both the payload and the
// digest, so only the semantic queue contract can reject these restores.
func TestCheckpointRejectsMalformedQueuedCommands(t *testing.T) {
	state, queue := queuedCheckpointFixture(t)
	envelope, err := BuildCheckpointEnvelope(state, queue)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("invalid target wave", func(t *testing.T) {
		badQueue := queue.Clone()
		badQueue.Entries[0].TargetWave = 99
		badEnvelope := envelope
		badEnvelope.RoleQueue = append([]RoleCommand(nil), badQueue.Entries...)
		badEnvelope = reseal(t, badEnvelope, state, badQueue)
		if err := VerifyCheckpointEnvelope(badEnvelope, state, badQueue); err == nil {
			t.Fatal("queued command with an invalid target wave was accepted")
		}
	})

	t.Run("retroactive queued command", func(t *testing.T) {
		badQueue := queue.Clone()
		badQueue.Entries[0].AcceptedTick = 900
		badEnvelope := envelope
		badEnvelope.RoleQueue = append([]RoleCommand(nil), badQueue.Entries...)
		badEnvelope = reseal(t, badEnvelope, state, badQueue)
		if err := VerifyCheckpointEnvelope(badEnvelope, state, badQueue); err == nil {
			t.Fatal("retroactive queued command was accepted")
		}
	})

	t.Run("VoteOptionNone queue entry", func(t *testing.T) {
		badQueue := queue.Clone()
		badQueue.Ballots[0].Option = domain.VoteOptionNone
		badEnvelope := envelope
		badEnvelope.BallotQueue = append([]VoteCommand(nil), badQueue.Ballots...)
		badEnvelope = reseal(t, badEnvelope, state, badQueue)
		if err := VerifyCheckpointEnvelope(badEnvelope, state, badQueue); err == nil {
			t.Fatal("a queued ballot without an option was accepted")
		}
	})

	t.Run("duplicate cross-kind input sequence", func(t *testing.T) {
		badQueue := queue.Clone()
		badQueue.Ballots[0].InputSeq = badQueue.Entries[0].InputSeq
		badEnvelope := envelope
		badEnvelope.BallotQueue = append([]VoteCommand(nil), badQueue.Ballots...)
		badEnvelope = reseal(t, badEnvelope, state, badQueue)
		if err := VerifyCheckpointEnvelope(badEnvelope, state, badQueue); err == nil {
			t.Fatal("a cross-kind duplicate input sequence was accepted")
		}
	})
}

func TestCheckpointDetectsRetainedXPTampering(t *testing.T) {
	state, queue := queuedCheckpointFixture(t)
	// A retired actor keeps session XP but is not a current-wave member.
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
	if canonicalBefore == canonicalAfter {
		t.Fatal("canonical v2 must cover retained non-member XP")
	}
	if err := VerifyCheckpointEnvelope(envelope, tampered, queue); err == nil {
		t.Fatal("private checkpoint did not detect retained-XP tampering")
	}

	// The historical v1 compatibility path keeps the documented gap, so old v1
	// hashes remain meaningful and are never silently reinterpreted.
	v1Before := state.Clone()
	v1Before.Revision.SchemaVersion = domain.LegacyCanonicalSchemaVersion
	v1After := tampered.Clone()
	v1After.Revision.SchemaVersion = domain.LegacyCanonicalSchemaVersion
	hashV1Before, err := canonical.Hash(v1Before)
	if err != nil {
		t.Fatal(err)
	}
	hashV1After, err := canonical.Hash(v1After)
	if err != nil {
		t.Fatal(err)
	}
	if hashV1Before != hashV1After {
		t.Fatal("historical v1 hashing unexpectedly covers non-member XP")
	}
}

func TestLegacyEnvelopeVersionRejected(t *testing.T) {
	state, queue := queuedCheckpointFixture(t)
	envelope, err := BuildCheckpointEnvelope(state, queue)
	if err != nil {
		t.Fatal(err)
	}
	if envelope.SchemaVersion != checkpointEnvelopeVersion {
		t.Fatalf("envelope version = %d", envelope.SchemaVersion)
	}
	for _, version := range []int{1, 2, 3, checkpointEnvelopeVersion + 1} {
		legacy := envelope
		legacy.SchemaVersion = version
		if err := VerifyCheckpointEnvelope(legacy, state, queue); err == nil {
			t.Fatalf("envelope version %d was accepted", version)
		}
	}
}

func TestCheckpointRejectsMalformedJSON(t *testing.T) {
	state, queue := queuedCheckpointFixture(t)
	envelope, err := BuildCheckpointEnvelope(state, queue)
	if err != nil {
		t.Fatal(err)
	}
	payload, err := json.Marshal(envelope)
	if err != nil {
		t.Fatal(err)
	}
	broken := append([]byte(nil), payload[:len(payload)/2]...)
	var restored CheckpointEnvelope
	if err := json.Unmarshal(broken, &restored); err == nil {
		t.Fatal("truncated checkpoint JSON was decoded")
	}
}

func TestCheckpointRejectsInvalidSourceMode(t *testing.T) {
	state, queue := queuedCheckpointFixture(t)
	envelope, err := BuildCheckpointEnvelope(state, queue)
	if err != nil {
		t.Fatal(err)
	}
	tampered := state.Clone()
	tampered.Mode = domain.SourceMode(99)
	if err := VerifyCheckpointEnvelope(envelope, tampered, queue); err == nil {
		t.Fatal("checkpoint with an invalid source mode was accepted")
	}
}

// TestCheckpointRejectsAppliedBallotTampering covers the applied-ballot half of
// the restore contract. Both variants keep the state internally valid (ballots and
// tally agree), so only the integrity values can reject them.
func TestCheckpointRejectsAppliedBallotTampering(t *testing.T) {
	base := collisionAssaultBase(t)
	envelope, err := BuildCheckpointEnvelope(base, RoleQueue{})
	if err != nil {
		t.Fatal(err)
	}
	if len(base.Wave.Ballots) != 2 {
		t.Fatalf("fixture has %d applied ballots, want 2", len(base.Wave.Ballots))
	}

	t.Run("modified applied ballot", func(t *testing.T) {
		tampered := base.Clone()
		tampered.Wave.Ballots[0].Option = domain.VoteOptionB
		tampered.Wave.Vote.Counts = domain.VoteCounts{A: 0, B: 1, C: 1}
		if err := tampered.Validate(); err != nil {
			t.Fatalf("modified-ballot fixture is invalid: %v", err)
		}
		if err := VerifyCheckpointEnvelope(envelope, tampered, RoleQueue{}); err == nil {
			t.Fatal("checkpoint accepted a state whose applied ballot was modified")
		}
	})

	t.Run("missing applied ballot", func(t *testing.T) {
		tampered := base.Clone()
		removed := tampered.Wave.Ballots[0].Option
		tampered.Wave.Ballots = append([]domain.Ballot(nil), tampered.Wave.Ballots[1:]...)
		if removed == domain.VoteOptionA {
			tampered.Wave.Vote.Counts = domain.VoteCounts{C: 1}
		} else {
			tampered.Wave.Vote.Counts = domain.VoteCounts{A: 1}
		}
		if err := tampered.Validate(); err != nil {
			t.Fatalf("missing-ballot fixture is invalid: %v", err)
		}
		if err := VerifyCheckpointEnvelope(envelope, tampered, RoleQueue{}); err == nil {
			t.Fatal("checkpoint accepted a state with a missing applied ballot")
		}
	})
}

// The canonical hash covers lifecycle but not the independent pause-reason set, so two
// paused states with different blockers can share a canonical hash while resuming
// differently. The private envelope must separate them.
func TestPauseReasonsAffectPrivateCheckpoint(t *testing.T) {
	state, queue := queuedCheckpointFixture(t)
	first := state.Clone()
	first.Lifecycle = domain.LifecyclePaused
	first.PauseReasons = []domain.PauseReason{domain.PauseOperatorPaused}
	second := state.Clone()
	second.Lifecycle = domain.LifecyclePaused
	second.PauseReasons = []domain.PauseReason{domain.PauseOperatorPaused, domain.PauseStorageUnavailable}
	if err := first.Validate(); err != nil {
		t.Fatal(err)
	}
	if err := second.Validate(); err != nil {
		t.Fatal(err)
	}
	hashFirst, err := canonical.Hash(first)
	if err != nil {
		t.Fatal(err)
	}
	hashSecond, err := canonical.Hash(second)
	if err != nil {
		t.Fatal(err)
	}
	if hashFirst != hashSecond {
		t.Fatal("the canonical hash unexpectedly covers pause reasons; update the contract notes")
	}
	envelopeFirst, err := BuildCheckpointEnvelope(first, queue)
	if err != nil {
		t.Fatal(err)
	}
	envelopeSecond, err := BuildCheckpointEnvelope(second, queue)
	if err != nil {
		t.Fatal(err)
	}
	if envelopeFirst.PrivateDigest == envelopeSecond.PrivateDigest {
		t.Fatal("private checkpoint does not separate pause reason sets")
	}
	if err := VerifyCheckpointEnvelope(envelopeFirst, second, queue); err == nil {
		t.Fatal("pause reason tampering was accepted")
	}
}
