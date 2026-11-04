package simulation

import (
	"testing"

	"github.com/hung98dev/ttgqldl/backend/internal/canonical"
	"github.com/hung98dev/ttgqldl/backend/internal/domain"
)

// multiWaveStart builds a real wave-1 session from WAITING: one accepted start
// command, then additional accepted registrations effective at tick 1.
func multiWaveStart(t *testing.T, participants int) domain.SessionState {
	t.Helper()
	state := newState(t)
	next, queue, _ := acceptRoles(t, state, RoleQueue{}, roleCommand(1, uniqueActor(1), domain.RoleBuilder, 1, 0))
	started, queue, _, _, err := AdvanceWithRoles(next, queue, Input{
		Kind:         KindStartWave,
		Registration: &Registration{ActorKey: uniqueActor(1), Role: domain.RoleBuilder},
	})
	if err != nil {
		t.Fatal(err)
	}
	commands := make([]RoleCommand, 0, participants-1)
	roles := []domain.Role{domain.RoleBuilder, domain.RoleGuard, domain.RoleFighter}
	for i := 2; i <= participants; i++ {
		commands = append(commands, RoleCommand{
			InputSeq: int64(i), ActorKey: uniqueActor(i), Role: roles[i%3],
			TargetWave: 1, ScheduledTick: 1, AcceptedTick: AcceptedTickUnspecified,
		})
	}
	accepted, queue, _, err := AcceptRoleCommands(started, queue, commands)
	if err != nil {
		t.Fatal(err)
	}
	at1, _ := advanceWithRoles(t, accepted, queue, 1)
	return at1
}

// runMultiWave executes wave 1, queues next-wave members, rolls over, votes on
// wave 2, and returns hashes/events per committed tick.
func runMultiWave(t *testing.T) ([]string, [][]Event) {
	t.Helper()
	state := multiWaveStart(t, 10)
	hashes := make([]string, 0, 6000)
	eventsAll := make([][]Event, 0, 6000)
	advance := func(next domain.SessionState, ev []Event) {
		state = next
		hash, err := canonical.Hash(state)
		if err != nil {
			t.Fatalf("hash at %d/%d: %v", state.Wave.Number, state.Tick(), err)
		}
		hashes = append(hashes, hash)
		eventsAll = append(eventsAll, ev)
	}
	for state.Wave.Number == 1 && state.Tick() < domain.RolloverTick {
		next, queue, ev, _, err := AdvanceWithRoles(state, RoleQueue{}, Input{Kind: KindAdvanceTick})
		if err != nil {
			t.Fatalf("wave1 advance %d: %v", state.Tick(), err)
		}
		state = next
		_ = queue
		advance(state, ev)
	}
	// Queue three NEXT-wave registrations during RESULTS.
	queue := RoleQueue{}
	for i := 50; i < 53; i++ {
		next, nextQueue, _, err := AcceptRoleCommands(state, queue, []RoleCommand{{
			InputSeq: int64(i), ActorKey: uniqueActor(i), Role: domain.RoleFighter,
			TargetWave: 2, ScheduledTick: 0, AcceptedTick: AcceptedTickUnspecified,
		}})
		if err != nil {
			t.Fatalf("queue next wave: %v", err)
		}
		state, queue = next, nextQueue
	}
	if state.Tick() < domain.RolloverTick {
		state, queue = advanceWithRoles(t, state, queue, domain.RolloverTick)
	}
	rolled, _, err := Next(state, Input{Kind: KindRollover})
	if err != nil {
		t.Fatalf("rollover: %v", err)
	}
	state = rolled
	if state.Wave.Number != 2 || state.Tick() != 0 {
		t.Fatalf("wave 2 did not auto-start: wave=%d tick=%d", state.Wave.Number, state.Tick())
	}
	advance(state, nil)
	for state.Wave.Number == 2 && state.Tick() < domain.RolloverTick {
		next, nextQueue, ev, _, err := AdvanceWithRoles(state, queue, Input{Kind: KindAdvanceTick})
		if err != nil {
			t.Fatalf("wave2 advance %d: %v", state.Tick(), err)
		}
		// Accept one ballot for the next tick at 899, then let it apply.
		if state.Tick() == 899 && len(queue.Ballots) == 0 {
			voted, voteQueue, _, err := AcceptVoteCommands(state, nextQueue, []VoteCommand{{
				InputSeq: 900, ActorKey: uniqueActor(50), TargetWave: 2,
				ScheduledTick: 900, Option: domain.VoteOptionA, AcceptedTick: AcceptedTickUnspecified,
			}})
			if err != nil {
				t.Fatalf("ballot: %v", err)
			}
			nextQueue = voteQueue
			_ = voted
		}
		state, queue = next, nextQueue
		advance(state, ev)
	}
	return hashes, eventsAll
}

func TestMultiWaveReplayFromWaiting(t *testing.T) {
	hashes, events := runMultiWave(t)
	if len(hashes) < 6000 {
		t.Fatalf("unexpected trace length %d", len(hashes))
	}
	if len(events) != len(hashes) {
		t.Fatal("event trace length mismatch")
	}
}

func TestCheckpointSemanticValidation(t *testing.T) {
	state, queue := queuedCheckpointFixture(t)
	envelope, err := BuildCheckpointEnvelope(state, queue)
	if err != nil {
		t.Fatal(err)
	}
	// A syntactically valid but target-wave-illegal command must be rejected only
	// when it is part of the queue being verified; the envelope covers the queue.
	if err := VerifyCheckpointEnvelope(envelope, state, queue); err != nil {
		t.Fatalf("valid envelope: %v", err)
	}
	bad := queue.Clone()
	bad.Entries[0].TargetWave = state.Wave.Number + 5
	if err := VerifyCheckpointEnvelope(envelope, state, bad); err == nil {
		t.Fatal("illegal target wave accepted")
	}
}
