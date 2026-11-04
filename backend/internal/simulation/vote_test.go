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
	// Start wave 2 from an accepted registration.
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

func ballot(seq int64, actor byte, option domain.VoteOption, tick int) VoteCommand {
	return VoteCommand{
		InputSeq: seq, ActorKey: actorKey(actor), TargetWave: 2,
		AcceptedTick: AcceptedTickUnspecified, ScheduledTick: tick, Option: option,
	}
}

func acceptVotes(t *testing.T, state domain.SessionState, ballots ...VoteCommand) domain.SessionState {
	t.Helper()
	next, _, err := AcceptVoteCommands(state, ballots)
	if err != nil {
		t.Fatalf("AcceptVoteCommands: %v", err)
	}
	return next
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

func TestBallotWindowAndValidation(t *testing.T) {
	state, queue := evenWave(t)
	state, queue = advanceWithRoles(t, state, queue, 800)
	// Non-member, late, invalid option and out-of-window ballots are rejected.
	cases := []VoteCommand{
		ballot(1, 'b', domain.VoteOptionA, 800),  // non-member
		ballot(2, 'a', domain.VoteOptionA, 799),  // before open
		ballot(3, 'a', domain.VoteOptionA, 1200), // window closed
		ballot(4, 'a', domain.VoteOptionNone, 900),
	}
	for _, command := range cases {
		if _, _, err := AcceptVoteCommands(state, []VoteCommand{command}); err == nil {
			t.Fatalf("ballot %+v was accepted", command)
		}
	}
	// A member ballot inside the window is accepted and stays one row.
	state = acceptVotes(t, state, ballot(5, 'a', domain.VoteOptionA, 800))
	state = acceptVotes(t, state, ballot(6, 'a', domain.VoteOptionA, 800))
	if len(state.Wave.Ballots) != 1 {
		t.Fatalf("ballots = %d", len(state.Wave.Ballots))
	}
}

func TestLatestBallotWinsAndFloodCollapses(t *testing.T) {
	state, queue := evenWave(t)
	state, queue = advanceWithRoles(t, state, queue, 800)
	state = acceptVotes(t, state, ballot(1, 'a', domain.VoteOptionA, 900))
	state = acceptVotes(t, state, ballot(2, 'a', domain.VoteOptionB, 950))
	if len(state.Wave.Ballots) != 1 || state.Wave.Ballots[0].Option != domain.VoteOptionB {
		t.Fatalf("latest ballot lost: %+v", state.Wave.Ballots)
	}
	for i := 1; i <= 1000; i++ {
		state = acceptVotes(t, state, ballot(int64(1000+i), 'a', domain.VoteOptionC, 1000))
	}
	if len(state.Wave.Ballots) != 1 {
		t.Fatalf("flood created %d ballots", len(state.Wave.Ballots))
	}
}

func TestVoteClosesOnceBeforeEconomy(t *testing.T) {
	state, queue := evenWave(t)
	state, queue = advanceWithRoles(t, state, queue, 1199)
	woodBefore := state.Village.Resources.Wood
	state = acceptVotes(t, state, ballot(1, 'a', domain.VoteOptionA, 1199))
	state, _ = advanceWithRoles(t, state, queue, 1200)
	if !state.Wave.Vote.Closed || state.Wave.Vote.Open || state.Wave.Vote.Selected != domain.VoteOptionA {
		t.Fatalf("vote at 1200 = %+v", state.Wave.Vote)
	}
	// A costs 50 wood at close; the tick-1200 builder economy then adds 2 (B=1).
	if state.Village.Resources.Wood != woodBefore-50+2 {
		t.Fatalf("wood at 1200 = %d, want %d", state.Village.Resources.Wood, woodBefore-50+2)
	}
	// Closing again must not deduct a second time.
	var events []Event
	again, againEvents, err := closeVote(state)
	if err != nil {
		t.Fatal(err)
	}
	if len(againEvents) != 0 || !reflect.DeepEqual(again, state) {
		t.Fatal("repeated close changed state or emitted events")
	}
	_ = events
}

func TestZeroBallotFallbacks(t *testing.T) {
	// Enough wood: A is selected and charged exactly once at close.
	state, queue := evenWave(t)
	state.Village.Resources.Wood = 80
	state, queue = advanceWithRoles(t, state, queue, 1199)
	woodBefore := state.Village.Resources.Wood
	state, _ = advanceWithRoles(t, state, queue, 1200)
	if state.Wave.Vote.Selected != domain.VoteOptionA {
		t.Fatalf("zero-ballot selection = %s, want A", state.Wave.Vote.Selected)
	}
	if state.Village.Resources.Wood != woodBefore-50+2 {
		t.Fatalf("zero-ballot A wood = %d, want %d", state.Village.Resources.Wood, woodBefore-50+2)
	}

	// Not enough wood: C, no charge.
	state2, queue2 := evenWave(t)
	state2.Village.Resources.Wood = 20
	state2, queue2 = advanceWithRoles(t, state2, queue2, 1199)
	woodBefore = state2.Village.Resources.Wood
	state2, _ = advanceWithRoles(t, state2, queue2, 1200)
	if state2.Wave.Vote.Selected != domain.VoteOptionC {
		t.Fatalf("zero-ballot C fallback = %s", state2.Wave.Vote.Selected)
	}
	if state2.Village.Resources.Wood <= woodBefore-50 {
		t.Fatalf("C charged wood: before=%d after=%d", woodBefore, state2.Village.Resources.Wood)
	}
}
func TestVoteModifiersAffectCombat(t *testing.T) {
	base := lockedAt(t, 1800, domain.PhaseAssault)
	base.Members = []domain.WaveMember{{ActorKey: uniqueActor(1), Role: domain.RoleGuard, JoinedTick: 0}}
	base.Wave.Number = 2 // even wave so vote modifiers are authorized
	base.Wave.Vote = domain.VoteState{}
	base.Village.WallHP = 1000

	cases := []struct {
		name   string
		option domain.VoteOption
		want   int64
	}{
		{"no vote", domain.VoteOptionNone, 4}, // raw 12 - block 8
		{"A guard", domain.VoteOptionA, 3},    // block 9
		{"C guard", domain.VoteOptionC, 5},    // block 7
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

	// Vote B multiplies the complete attack before lane splitting.
	withB := base
	withB.Wave.Vote = domain.VoteState{Closed: true, Selected: domain.VoteOptionB}
	var events []Event
	next, err := resolveCombat(withB, &events)
	if err != nil {
		t.Fatal(err)
	}
	attack := int64(0)
	for _, event := range events {
		if event.Type == EventAttackResolved {
			attack = event.Amount
		}
	}
	// No fighters: base attack = militia 40; B -> floor(40*115/100) = 46.
	if attack != 46 {
		t.Fatalf("B attack = %d, want 46", attack)
	}
	_ = next
}

func TestVoteDeferredPausedAndResetAtRollover(t *testing.T) {
	state, queue := evenWave(t)
	state, queue = advanceWithRoles(t, state, queue, 800)
	paused, _, err := Next(state, Input{Kind: KindPause, Reason: domain.PauseOperatorPaused})
	if err != nil {
		t.Fatal(err)
	}
	snapshot := paused.Clone()
	_, _, err = AcceptVoteCommands(paused, []VoteCommand{ballot(1, 'a', domain.VoteOptionA, 900)})
	var deferred *VoteDeferredError
	if !errors.As(err, &deferred) || !errors.Is(err, ErrInputDeferred) {
		t.Fatalf("paused ballot error = %v", err)
	}
	if !reflect.DeepEqual(paused, snapshot) {
		t.Fatal("paused ballot mutated state")
	}

	settled, _ := advanceWithRoles(t, state, queue, 3000)
	settled.Wave.Result = domain.WaveResultVictory
	rolled, _, err := Next(settled, Input{Kind: KindRollover})
	if err != nil {
		t.Fatalf("rollover: %v", err)
	}
	if rolled.Wave.Vote.Open || rolled.Wave.Vote.Closed || rolled.Wave.Vote.Selected != domain.VoteOptionNone || len(rolled.Wave.Ballots) != 0 {
		t.Fatalf("vote state leaked into wave 3: %+v", rolled.Wave.Vote)
	}
}
