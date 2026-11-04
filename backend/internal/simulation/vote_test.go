package simulation

import (
	"errors"
	"reflect"
	"testing"

	"github.com/hung98dev/ttgqldl/backend/internal/domain"
)

// evenWave builds a genuine wave 2 (voting enabled) from WAITING.
func evenWave(t *testing.T) (domain.SessionState, RoleQueue) {
	t.Helper()
	state := newState(t)
	next, queue, _ := acceptRoles(t, state, RoleQueue{}, roleCommand(1, actorKey('a'), domain.RoleBuilder, 1, 0))
	started, _, _, _, err := AdvanceWithRoles(next, queue, Input{
		Kind:         KindStartWave,
		Registration: &Registration{ActorKey: actorKey('a'), Role: domain.RoleBuilder},
	})
	if err != nil {
		t.Fatal(err)
	}
	at3000, _ := advanceWithRoles(t, started, RoleQueue{}, 3000)
	at3000.Wave.Result = domain.WaveResultVictory
	rolled, _, err := Next(at3000, Input{Kind: KindRollover})
	if err != nil {
		t.Fatalf("rollover: %v", err)
	}
	accepted, queue, _ := acceptRoles(t, rolled, RoleQueue{}, roleCommand(2, actorKey('a'), domain.RoleBuilder, 2, 0))
	started2, remaining, _, _, err := AdvanceWithRoles(accepted, queue, Input{
		Kind:         KindStartWave,
		Registration: &Registration{ActorKey: actorKey('a'), Role: domain.RoleBuilder},
	})
	if err != nil {
		t.Fatalf("start wave 2: %v", err)
	}
	return started2, remaining
}

func ballot(seq int64, actor byte, option domain.VoteOption, wave int64, tick int) VoteCommand {
	return VoteCommand{
		InputSeq: seq, ActorKey: actorKey(actor), TargetWave: wave,
		AcceptedTick: AcceptedTickUnspecified, ScheduledTick: tick, Option: option,
	}
}

func acceptVotes(t *testing.T, state domain.SessionState, queue RoleQueue, ballots ...VoteCommand) (domain.SessionState, RoleQueue) {
	t.Helper()
	next, nextQueue, _, err := AcceptVoteCommands(state, queue, ballots)
	if err != nil {
		t.Fatalf("AcceptVoteCommands: %v", err)
	}
	if len(next.Wave.Ballots) != len(state.Wave.Ballots) {
		t.Fatal("acceptance prematurely mutated authoritative ballots")
	}
	return next, nextQueue
}

func TestVoteOpensOnlyOnEvenWaveAt800(t *testing.T) {
	state, queue := evenWave(t)
	state, queue = advanceWithRoles(t, state, queue, 799)
	if state.Wave.Vote.Open {
		t.Fatal("vote opened before 800")
	}
	state, queue = advanceWithRoles(t, state, queue, 800)
	if !state.Wave.Vote.Open || state.Wave.Vote.Selected != domain.VoteOptionNone {
		t.Fatalf("vote at 800 = %+v", state.Wave.Vote)
	}

	odd, _ := realWave(t)
	odd, _ = advanceWithRoles(t, odd, RoleQueue{}, 800)
	if odd.Wave.Vote.Open {
		t.Fatal("odd wave opened a vote")
	}
}

func TestBallotQueueDefersUntilEffectiveTick(t *testing.T) {
	state, queue := evenWave(t)
	state, queue = advanceWithRoles(t, state, queue, 899)

	accepted, queue := acceptVotes(t, state, queue, ballot(1, 'a', domain.VoteOptionA, 2, 900))
	if len(accepted.Wave.Ballots) != 0 {
		t.Fatal("future ballot became authoritative immediately")
	}
	if len(queue.Ballots) != 1 {
		t.Fatalf("queue = %d", len(queue.Ballots))
	}

	at900, queue := advanceWithRoles(t, accepted, queue, 900)
	if len(at900.Wave.Ballots) != 1 || at900.Wave.Ballots[0].Option != domain.VoteOptionA {
		t.Fatalf("applied ballots = %+v", at900.Wave.Ballots)
	}
	if len(queue.Ballots) != 0 {
		t.Fatalf("queue after application = %d", len(queue.Ballots))
	}
}

func TestRetroactiveAndInvalidBallotsRejected(t *testing.T) {
	state, queue := evenWave(t)
	state, queue = advanceWithRoles(t, state, queue, 1100)

	cases := []VoteCommand{
		ballot(1, 'a', domain.VoteOptionB, 2, 900),  // committed tick
		ballot(2, 'a', domain.VoteOptionB, 2, 1100), // committed tick
		ballot(3, 'a', domain.VoteOptionB, 2, 1102), // not committed+1
		ballot(4, 'a', domain.VoteOptionB, 2, 1200), // window closed
		ballot(5, 'b', domain.VoteOptionB, 2, 1101), // non-member
		ballot(6, 'a', domain.VoteOptionNone, 2, 1101),
		ballot(7, 'a', domain.VoteOptionA, 3, 1101), // wrong wave
	}
	for _, command := range cases {
		if _, _, _, err := AcceptVoteCommands(state, queue, []VoteCommand{command}); err == nil {
			t.Fatalf("invalid ballot %+v accepted", command)
		}
	}
	if len(state.Wave.Ballots) != 0 || len(queue.Ballots) != 0 {
		t.Fatal("rejected ballots mutated state or queue")
	}
}

func TestLatestBallotWinsAndFloodCollapses(t *testing.T) {
	state, queue := evenWave(t)
	state, queue = advanceWithRoles(t, state, queue, 899)
	state, queue = acceptVotes(t, state, queue, ballot(1, 'a', domain.VoteOptionA, 2, 900))
	state, queue = advanceWithRoles(t, state, queue, 900)
	state, queue = advanceWithRoles(t, state, queue, 949)
	state, queue = acceptVotes(t, state, queue, ballot(2, 'a', domain.VoteOptionB, 2, 950))
	state, queue = advanceWithRoles(t, state, queue, 950)
	if len(state.Wave.Ballots) != 1 || state.Wave.Ballots[0].Option != domain.VoteOptionB {
		t.Fatalf("latest ballot = %+v", state.Wave.Ballots)
	}

	// A flood for one effective tick still yields exactly one ballot.
	state, queue = advanceWithRoles(t, state, queue, 999)
	commands := make([]VoteCommand, 0, 1000)
	for i := 1; i <= 1000; i++ {
		commands = append(commands, ballot(int64(1000+i), 'a', domain.VoteOptionC, 2, 1000))
	}
	state, queue = acceptVotes(t, state, queue, commands...)
	state, queue = advanceWithRoles(t, state, queue, 1000)
	if len(state.Wave.Ballots) != 1 {
		t.Fatalf("flood ballots = %d", len(state.Wave.Ballots))
	}
	_ = queue
}

func TestVoteClosesOnceBeforeEconomy(t *testing.T) {
	state, queue := evenWave(t)
	state, queue = advanceWithRoles(t, state, queue, 1198)
	state, queue = acceptVotes(t, state, queue, ballot(1, 'a', domain.VoteOptionA, 2, 1199))
	state, queue = advanceWithRoles(t, state, queue, 1199)
	if len(state.Wave.Ballots) != 1 {
		t.Fatal("ballot not applied at 1199")
	}
	woodBefore := state.Village.Resources.Wood
	state, queue = advanceWithRoles(t, state, queue, 1200)
	if !state.Wave.Vote.Closed || state.Wave.Vote.Selected != domain.VoteOptionA {
		t.Fatalf("vote at 1200 = %+v", state.Wave.Vote)
	}
	if state.Village.Resources.Wood != woodBefore-50+2 {
		t.Fatalf("wood at 1200 = %d, want %d", state.Village.Resources.Wood, woodBefore-50+2)
	}
	if _, events, err := closeVote(state); err != nil || len(events) != 0 {
		t.Fatal("repeated close was not a no-op")
	}
}

func TestVoteModifiersAffectCombat(t *testing.T) {
	base := lockedAt(t, 1800, domain.PhaseAssault)
	base.Members = []domain.WaveMember{{ActorKey: uniqueActor(1), Role: domain.RoleGuard, JoinedTick: 0}}
	base.Wave.Number = 2
	base.Wave.Vote = domain.VoteState{}
	base.Village.WallHP = 1000

	cases := []struct {
		name   string
		option domain.VoteOption
		want   int64
	}{
		{"no vote", domain.VoteOptionNone, 4},
		{"A guard", domain.VoteOptionA, 3},
		{"C guard", domain.VoteOptionC, 5},
	}
	for _, tc := range cases {
		state := base
		if tc.option != domain.VoteOptionNone {
			state.Wave.Vote = domain.VoteState{Closed: true, Selected: tc.option}
		}
		var events []Event
		next, err := resolveCombat(state, &events)
		if err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}
		if got := state.Village.WallHP - next.Village.WallHP; got != tc.want {
			t.Fatalf("%s: wall damage = %d, want %d", tc.name, got, tc.want)
		}
	}

	withB := base
	withB.Wave.Vote = domain.VoteState{Closed: true, Selected: domain.VoteOptionB}
	var events []Event
	if _, err := resolveCombat(withB, &events); err != nil {
		t.Fatal(err)
	}
	attack := int64(0)
	for _, event := range events {
		if event.Type == EventAttackResolved {
			attack = event.Amount
		}
	}
	if attack != 46 {
		t.Fatalf("B attack = %d, want 46", attack)
	}
}

func TestPausedBallotsDeferred(t *testing.T) {
	state, queue := evenWave(t)
	state, queue = advanceWithRoles(t, state, queue, 899)
	paused, _, err := Next(state, Input{Kind: KindPause, Reason: domain.PauseOperatorPaused})
	if err != nil {
		t.Fatal(err)
	}
	snapshot := paused.Clone()
	_, _, _, err = AcceptVoteCommands(paused, queue, []VoteCommand{ballot(1, 'a', domain.VoteOptionA, 2, 900)})
	var deferred *VoteDeferredError
	if !errors.As(err, &deferred) || !errors.Is(err, ErrInputDeferred) {
		t.Fatalf("paused ballot error = %v", err)
	}
	if !reflect.DeepEqual(paused, snapshot) {
		t.Fatal("paused ballot mutated state")
	}
}

func TestVoteStateResetsAtRollover(t *testing.T) {
	state, queue := evenWave(t)
	state, _ = advanceWithRoles(t, state, queue, 3000)
	state.Wave.Result = domain.WaveResultVictory
	rolled, _, err := Next(state, Input{Kind: KindRollover})
	if err != nil {
		t.Fatalf("rollover: %v", err)
	}
	if rolled.Wave.Vote.Open || rolled.Wave.Vote.Closed || rolled.Wave.Vote.Selected != domain.VoteOptionNone || len(rolled.Wave.Ballots) != 0 {
		t.Fatalf("vote leaked into wave 3: %+v", rolled.Wave.Vote)
	}
}
