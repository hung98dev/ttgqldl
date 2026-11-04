package simulation

import (
	"errors"
	"reflect"
	"testing"

	"github.com/hung98dev/ttgqldl/backend/internal/domain"
)

// envelopeWithQueue builds a checkpoint envelope whose queues are exactly the
// supplied ones. It is used by the semantic negative tests: the digest is
// recomputed deliberately, so a rejection can only come from the queue contract,
// never from a checksum mismatch.
func envelopeWithQueue(t *testing.T, state domain.SessionState, queue RoleQueue) CheckpointEnvelope {
	t.Helper()
	if err := state.Validate(); err != nil {
		t.Fatalf("negative fixture state is invalid: %v", err)
	}
	envelope, err := BuildCheckpointEnvelope(state, queue)
	if err != nil {
		t.Fatalf("build checkpoint envelope: %v", err)
	}
	return envelope
}

// corruptQueue rewrites the queues, then reseals the digest over the corrupted
// payload. The state is untouched, so only validateCheckpointQueue can reject it.
func corruptQueue(t *testing.T, state domain.SessionState, queue RoleQueue, mutate func(*RoleQueue)) (CheckpointEnvelope, RoleQueue) {
	t.Helper()
	corrupted := queue.Clone()
	mutate(&corrupted)
	return envelopeWithQueue(t, state, corrupted), corrupted
}

// TestCheckpointQueueStaleCommands proves that a pending command which is already
// due at the committed tick cannot be restored. The digest is recomputed, so this
// is a semantic rejection, not a checksum rejection.
func TestCheckpointQueueStaleCommands(t *testing.T) {
	state, queue := queuedCheckpointFixture(t)
	baseline := envelopeWithQueue(t, state, queue)
	if err := VerifyCheckpointEnvelope(baseline, state, queue); err != nil {
		t.Fatalf("legitimate tick-899 -> tick-900 checkpoint was rejected: %v", err)
	}

	t.Run("ballot already due at the committed tick", func(t *testing.T) {
		envelope, corrupted := corruptQueue(t, state, queue, func(corrupted *RoleQueue) {
			corrupted.Ballots[0].AcceptedTick = 898
			corrupted.Ballots[0].ScheduledTick = 899
		})
		before := state.Clone()
		if err := VerifyCheckpointEnvelope(envelope, state, corrupted); !errors.Is(err, ErrInvalidState) {
			t.Fatalf("stale ballot checkpoint was accepted (err=%v); a ballot effective at the committed tick must be rejected", err)
		}
		if !reflect.DeepEqual(state, before) {
			t.Fatal("rejected restore mutated the state")
		}
	})

	t.Run("role already due at the committed tick", func(t *testing.T) {
		envelope, corrupted := corruptQueue(t, state, queue, func(corrupted *RoleQueue) {
			corrupted.Entries[0].AcceptedTick = 898
			corrupted.Entries[0].ScheduledTick = 899
		})
		before := state.Clone()
		if err := VerifyCheckpointEnvelope(envelope, state, corrupted); !errors.Is(err, ErrInvalidState) {
			t.Fatalf("stale role checkpoint was accepted (err=%v); a command effective at the committed tick must be rejected", err)
		}
		if !reflect.DeepEqual(state, before) {
			t.Fatal("rejected restore mutated the state")
		}
	})

	t.Run("role far in the past", func(t *testing.T) {
		envelope, corrupted := corruptQueue(t, state, queue, func(corrupted *RoleQueue) {
			corrupted.Entries[0].AcceptedTick = 799
			corrupted.Entries[0].ScheduledTick = 800
		})
		if err := VerifyCheckpointEnvelope(envelope, state, corrupted); !errors.Is(err, ErrInvalidState) {
			t.Fatalf("role command scheduled long before the committed tick was accepted (err=%v)", err)
		}
	})
}

// TestCheckpointQueueBallotEligibility applies the same voter-authorization rules
// the production acceptance path uses: the actor must be a current-wave member and
// must already be effective at the ballot's scheduled tick.
func TestCheckpointQueueBallotEligibility(t *testing.T) {
	state, queue := queuedCheckpointFixture(t)

	t.Run("actor is not a current-wave participant", func(t *testing.T) {
		envelope, corrupted := corruptQueue(t, state, queue, func(corrupted *RoleQueue) {
			corrupted.Ballots[0].ActorKey = actorKey('c')
		})
		if err := VerifyCheckpointEnvelope(envelope, state, corrupted); !errors.Is(err, ErrInvalidState) {
			t.Fatalf("ballot from a non-participant was accepted (err=%v)", err)
		}
	})

	t.Run("actor joins after the ballot tick", func(t *testing.T) {
		late := state.Clone()
		late.Members = append(late.Members, domain.WaveMember{
			ActorKey: actorKey('d'), Role: domain.RoleFighter, JoinedTick: 950,
		})
		domain.SortWaveMembers(late.Members)
		if err := late.Validate(); err != nil {
			t.Fatalf("late-join fixture is invalid: %v", err)
		}
		envelope, corrupted := corruptQueue(t, late, queue, func(corrupted *RoleQueue) {
			corrupted.Ballots[0].ActorKey = actorKey('d')
		})
		if err := VerifyCheckpointEnvelope(envelope, late, corrupted); !errors.Is(err, ErrInvalidState) {
			t.Fatalf("ballot from an actor not effective at the ballot tick was accepted (err=%v)", err)
		}
	})
}

// TestCheckpointQueueOwnership proves that NEXT-wave registrations live in the
// authoritative pending-member state, never as ordinary current-wave role queue
// entries. A production fixture demonstrates the legal lifecycle; the corrupted
// queue must be rejected.
func TestCheckpointQueueOwnership(t *testing.T) {
	state, queue := evenWave(t)
	state, queue = advanceWithRoles(t, state, queue, domain.FinalCombatTick)
	nextWave := RoleCommand{
		InputSeq: 40, ActorKey: actorKey('e'), Role: domain.RoleBuilder,
		TargetWave: state.Wave.Number + 1, ScheduledTick: 0, AcceptedTick: AcceptedTickUnspecified,
	}
	acceptedState, acceptedQueue, decisions, err := AcceptRoleCommands(state, queue, []RoleCommand{nextWave})
	if err != nil {
		t.Fatalf("legitimate RESULTS registration was rejected: %v", err)
	}
	if decisions[0].Outcome != OutcomeNextWaveQueued {
		t.Fatalf("RESULTS registration outcome = %s", decisions[0].Outcome)
	}
	if acceptedQueue.Len() != 0 {
		t.Fatalf("NEXT-wave registration stayed in the role queue: %d entries", acceptedQueue.Len())
	}
	if len(acceptedState.Pending) != 1 {
		t.Fatalf("NEXT-wave registration was not written to authoritative pending state: %d", len(acceptedState.Pending))
	}
	legitimate := envelopeWithQueue(t, acceptedState, acceptedQueue)
	if err := VerifyCheckpointEnvelope(legitimate, acceptedState, acceptedQueue); err != nil {
		t.Fatalf("legitimate pending-state checkpoint was rejected: %v", err)
	}

	corruptedCommand := nextWave
	corruptedCommand.AcceptedTick = state.Tick()
	envelope, corrupted := corruptQueue(t, acceptedState, acceptedQueue, func(corrupted *RoleQueue) {
		corrupted.Entries = []RoleCommand{corruptedCommand}
	})
	if err := VerifyCheckpointEnvelope(envelope, acceptedState, corrupted); !errors.Is(err, ErrInvalidState) {
		t.Fatalf("NEXT-wave command in the ordinary role queue was accepted (err=%v); production never keeps one there", err)
	}
}

// TestCheckpointQueueOrdering enforces the persisted deterministic order contract:
// scheduled_tick ascending, then input_seq ascending on ties.
func TestCheckpointQueueOrdering(t *testing.T) {
	state, queue := queuedCheckpointFixture(t)

	t.Run("role queue order", func(t *testing.T) {
		envelope, corrupted := corruptQueue(t, state, queue, func(corrupted *RoleQueue) {
			corrupted.Entries = []RoleCommand{
				{InputSeq: 20, ActorKey: actorKey('c'), Role: domain.RoleBuilder, TargetWave: state.Wave.Number, ScheduledTick: 950, AcceptedTick: 899},
				{InputSeq: 21, ActorKey: actorKey('d'), Role: domain.RoleGuard, TargetWave: state.Wave.Number, ScheduledTick: 900, AcceptedTick: 899},
			}
		})
		if err := VerifyCheckpointEnvelope(envelope, state, corrupted); !errors.Is(err, ErrInvalidState) {
			t.Fatalf("noncanonical role queue order was accepted (err=%v)", err)
		}
	})

	t.Run("ballot queue order", func(t *testing.T) {
		envelope, corrupted := corruptQueue(t, state, queue, func(corrupted *RoleQueue) {
			corrupted.Ballots = []VoteCommand{
				{InputSeq: 30, ActorKey: actorKey('a'), TargetWave: state.Wave.Number, ScheduledTick: 900, Option: domain.VoteOptionA, AcceptedTick: 899},
				{InputSeq: 29, ActorKey: actorKey('a'), TargetWave: state.Wave.Number, ScheduledTick: 900, Option: domain.VoteOptionB, AcceptedTick: 899},
			}
		})
		if err := VerifyCheckpointEnvelope(envelope, state, corrupted); !errors.Is(err, ErrInvalidState) {
			t.Fatalf("noncanonical ballot queue order was accepted (err=%v)", err)
		}
	})
}

// TestCheckpointQueueAlreadyAppliedBallot rejects a queue entry that duplicates an
// input sequence already recorded among the applied ballots.
func TestCheckpointQueueAlreadyAppliedBallot(t *testing.T) {
	state, queue := evenWave(t)
	state, queue = advanceWithRoles(t, state, queue, 899)
	accepted, queue, _, err := AcceptVoteCommands(state, queue, []VoteCommand{{
		InputSeq: 12, ActorKey: actorKey('a'), TargetWave: state.Wave.Number,
		ScheduledTick: 900, Option: domain.VoteOptionA, AcceptedTick: AcceptedTickUnspecified,
	}})
	if err != nil {
		t.Fatal(err)
	}
	state, queue = advanceWithRoles(t, accepted, queue, 1000)
	if len(state.Wave.Ballots) != 1 || state.Wave.Ballots[0].InputSeq != 12 {
		t.Fatalf("fixture applied ballots = %+v", state.Wave.Ballots)
	}
	accepted, queue, _, err = AcceptVoteCommands(state, queue, []VoteCommand{{
		InputSeq: 13, ActorKey: actorKey('a'), TargetWave: state.Wave.Number,
		ScheduledTick: 1001, Option: domain.VoteOptionB, AcceptedTick: AcceptedTickUnspecified,
	}})
	if err != nil {
		t.Fatal(err)
	}
	envelope, corrupted := corruptQueue(t, accepted, queue, func(corrupted *RoleQueue) {
		corrupted.Ballots[0].InputSeq = 12
	})
	if err := VerifyCheckpointEnvelope(envelope, accepted, corrupted); !errors.Is(err, ErrInvalidState) {
		t.Fatalf("queued ballot reusing an applied input sequence was accepted (err=%v)", err)
	}
}

// TestCheckpointQueueAcceptsLegitimateStates guards against over-rejection. A
// queued wave-start registration exists in production between AcceptRoleCommands and
// AdvanceWithRoles(KindStartWave), and a paused session legitimately keeps future
// commands queued.
func TestCheckpointQueueAcceptsLegitimateStates(t *testing.T) {
	t.Run("waiting start registration", func(t *testing.T) {
		state := newState(t)
		next, queue, _ := acceptRoles(t, state, RoleQueue{}, roleCommand(1, actorKey('a'), domain.RoleBuilder, 1, 0))
		if queue.Len() != 1 {
			t.Fatalf("waiting queue = %d entries", queue.Len())
		}
		envelope := envelopeWithQueue(t, next, queue)
		if err := VerifyCheckpointEnvelope(envelope, next, queue); err != nil {
			t.Fatalf("legitimate WAITING checkpoint with a queued start registration was rejected: %v", err)
		}
	})

	t.Run("paused session with future inputs", func(t *testing.T) {
		state, queue := queuedCheckpointFixture(t)
		paused := state.Clone()
		paused.Lifecycle = domain.LifecyclePaused
		paused.PauseReasons = []domain.PauseReason{domain.PauseOperatorPaused}
		envelope := envelopeWithQueue(t, paused, queue)
		if err := VerifyCheckpointEnvelope(envelope, paused, queue); err != nil {
			t.Fatalf("legitimate paused checkpoint with future queued inputs was rejected: %v", err)
		}
	})
}
