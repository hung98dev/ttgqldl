package simulation

import (
	"errors"
	"math"
	"reflect"
	"testing"

	"github.com/hung98dev/ttgqldl/backend/internal/domain"
)

// stateAtRollover drives a started wave to the control tick and applies the
// explicit synthetic settlement fixture required by the fail-closed gate. W0202
// does not implement settlement; tests identify it clearly as a fixture.
func stateAtRollover(t *testing.T) domain.SessionState {
	t.Helper()
	state, _ := mustStart(t, newState(t))
	state = advanceTo(t, state, domain.RolloverTick)
	state.Wave.Result = domain.WaveResultVictory
	return state
}

func TestRolloverRejectsUnsettledWave(t *testing.T) {
	state, _ := mustStart(t, newState(t))
	state = advanceTo(t, state, domain.RolloverTick)
	snapshot := state.Clone()
	if state.Wave.Result != domain.WaveResultPending {
		t.Fatal("fixture must be unsettled")
	}
	next, events, err := Next(state, Input{Kind: KindRollover})
	if !errors.Is(err, ErrUnsettledWave) {
		t.Fatalf("unsettled rollover: %v", err)
	}
	if len(events) != 0 {
		t.Fatal("a rejected rollover must not emit events")
	}
	if !reflect.DeepEqual(next, state) || !reflect.DeepEqual(state, snapshot) {
		t.Fatal("a rejected rollover changed the state")
	}
}

func TestRolloverRejectsWaveNumberOverflow(t *testing.T) {
	state := stateAtRollover(t)
	state.Wave.Number = math.MaxInt64
	state.Pending = nil
	next, _, err := Next(state, Input{Kind: KindRollover})
	if !errors.Is(err, ErrOverflow) {
		t.Fatalf("overflow rollover: %v", err)
	}
	if next.Wave.Number != math.MaxInt64 {
		t.Fatal("overflow rejection changed the wave number")
	}
}

func TestRolloverWithoutPendingReturnsToWaiting(t *testing.T) {
	state := stateAtRollover(t)
	next, events, err := Next(state, Input{Kind: KindRollover})
	if err != nil {
		t.Fatalf("rollover: %v", err)
	}

	if next.Wave.Number != state.Wave.Number+1 {
		t.Fatalf("wave number = %d, want %d", next.Wave.Number, state.Wave.Number+1)
	}
	if next.Tick() != domain.WaitingTick || next.Wave.Phase != domain.PhaseWaiting {
		t.Fatalf("tick/phase = %d/%s", next.Tick(), next.Wave.Phase)
	}
	if next.Lifecycle != domain.LifecycleWaiting {
		t.Fatalf("lifecycle = %s", next.Lifecycle)
	}
	if len(next.Members) != 0 || len(next.Pending) != 0 {
		t.Fatalf("participants must expire at rollover: %d/%d", len(next.Members), len(next.Pending))
	}
	if !reflect.DeepEqual(next.Village, state.Village) {
		t.Fatal("village progress must survive rollover")
	}
	if len(next.XP) != len(state.XP) {
		t.Fatal("rollover must not award XP")
	}
	if len(events) != 1 || events[0].Type != EventRolledOver {
		t.Fatalf("events = %v", eventTypes(events))
	}
}

func TestRolloverPromotesPendingRegistrations(t *testing.T) {
	state := stateAtRollover(t)
	state.Pending = []domain.PendingMember{
		{ActorKey: actorKey('b'), Role: domain.RoleGuard, TargetWave: state.Wave.Number + 1},
		{ActorKey: actorKey('c'), Role: domain.RoleFighter, TargetWave: state.Wave.Number + 1},
	}
	// Keep the canonical order invariant explicit.
	domain.SortPendingMembers(state.Pending)

	next, events, err := Next(state, Input{Kind: KindRollover})
	if err != nil {
		t.Fatalf("rollover: %v", err)
	}
	if next.Wave.Number != state.Wave.Number+1 {
		t.Fatalf("wave = %d", next.Wave.Number)
	}
	if next.Tick() != 0 || next.Wave.Phase != domain.PhasePrep {
		t.Fatalf("promoted wave must start at tick 0: %d/%s", next.Tick(), next.Wave.Phase)
	}
	if next.Lifecycle != domain.LifecycleRunning {
		t.Fatalf("lifecycle = %s", next.Lifecycle)
	}
	if len(next.Pending) != 0 {
		t.Fatal("pending queue must be consumed")
	}
	if len(next.Members) != 2 {
		t.Fatalf("members = %d", len(next.Members))
	}
	for _, member := range next.Members {
		if member.JoinedTick != 0 {
			t.Fatalf("promoted member joined tick = %d", member.JoinedTick)
		}
		if member.LastChangeTick != nil || member.FirstSwitchUsed {
			t.Fatal("a promoted member starts with no role change")
		}
	}
	// Roles survive; viewers never need to comment again.
	if next.Members[0].Role != domain.RoleGuard || next.Members[1].Role != domain.RoleFighter {
		t.Fatalf("roles after promotion = %s/%s", next.Members[0].Role, next.Members[1].Role)
	}
	if !reflect.DeepEqual(eventTypes(events), []EventType{EventRolledOver, EventWaveStarted}) {
		t.Fatalf("events = %v", eventTypes(events))
	}
}

func TestRepeatedRolloverIsRejected(t *testing.T) {
	state := stateAtRollover(t)
	next, _, err := Next(state, Input{Kind: KindRollover})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := Next(next, Input{Kind: KindRollover}); !errors.Is(err, ErrNotAtRollover) {
		t.Fatalf("second rollover at tick 0: %v", err)
	}
	// The wave number must not have been incremented twice.
	if next.Wave.Number != state.Wave.Number+1 {
		t.Fatal("wave number changed on the rejected rollover")
	}
}

func TestRolloverRequiresTheControlTick(t *testing.T) {
	state, _ := mustStart(t, newState(t))
	state = advanceTo(t, state, 2999)
	if _, _, err := Next(state, Input{Kind: KindRollover}); !errors.Is(err, ErrNotAtRollover) {
		t.Fatalf("rollover before 3000: %v", err)
	}
	// Advancing reaches the control tick but must not roll over by itself.
	state, _, err := Next(state, Input{Kind: KindAdvanceTick})
	if err != nil {
		t.Fatal(err)
	}
	if state.Wave.Number != 1 || state.Tick() != 3000 {
		t.Fatalf("unexpected state %d/%d", state.Wave.Number, state.Tick())
	}
}

func TestRolloverResetsWaveStateButKeepsProgress(t *testing.T) {
	state := stateAtRollover(t)
	// Use a boss wave so a spawned (and killed) boss is representable.
	state.Wave.Number = 5
	state.Pending = nil
	state.Wave.Pools = []domain.EnemyPool{{Lane: domain.LaneWest, Archetype: domain.ArchetypeRunner, HP: 5}}
	state.Wave.Boss = domain.BossPool{Exists: true, HP: 0, MaxHP: 10}
	state.Wave.Ballots = []domain.Ballot{{ActorKey: actorKey('a'), Option: domain.VoteOptionA, Tick: 10}}
	state.Wave.Vote = domain.VoteState{Open: false, Selected: domain.VoteOptionA, Counts: domain.VoteCounts{A: 1}}
	state.Wave.Defeated = true
	state.Wave.Result = domain.WaveResultDefeat
	state.Village.Resources.Wood = 1234
	state.Village.TotalWins = 7
	state.XP = []domain.PlayerXP{{ActorKey: actorKey('a'), XP: 20}}

	// The defeated/lossed wave is now at the control tick; rollover must reset the
	// wave but keep persistent progress and XP.
	next, _, err := Next(state, Input{Kind: KindRollover})
	if err != nil {
		t.Fatalf("rollover: %v", err)
	}
	if len(next.Wave.Pools) != 0 || next.Wave.Boss.Exists || len(next.Wave.Ballots) != 0 {
		t.Fatal("wave state was not reset")
	}
	if next.Wave.Defeated || next.Wave.Result != domain.WaveResultPending || next.Wave.Vote.Open {
		t.Fatal("wave flags were not reset")
	}
	if next.Village.Resources.Wood != 1234 || next.Village.TotalWins != 7 {
		t.Fatal("village progress changed at rollover")
	}
	if len(next.XP) != 1 || next.XP[0].XP != 20 {
		t.Fatal("session XP changed at rollover")
	}
	if next.ConsecutiveWins != state.ConsecutiveWins || next.ConsecutiveLosses != state.ConsecutiveLosses {
		t.Fatal("streak counters changed at rollover")
	}
	if next.Difficulty.PopulationAnchor != 0 {
		t.Fatal("the population anchor must reset for the new wave")
	}
}

func TestPauseFreezesRolloverUntilResumed(t *testing.T) {
	state := stateAtRollover(t)
	state.Pending = []domain.PendingMember{{ActorKey: actorKey('b'), Role: domain.RoleGuard, TargetWave: state.Wave.Number + 1}}
	state, _, err := Next(state, Input{Kind: KindPause, Reason: domain.PauseSourceDisconnected})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := Next(state, Input{Kind: KindRollover}); !errors.Is(err, ErrTickFrozen) {
		t.Fatal("a paused session performed a rollover")
	}

	resumed, _, err := Next(state, Input{Kind: KindResume, Reason: domain.PauseSourceDisconnected})
	if err != nil {
		t.Fatal(err)
	}
	next, _, err := Next(resumed, Input{Kind: KindRollover})
	if err != nil {
		t.Fatalf("rollover after resume: %v", err)
	}
	if next.Tick() != 0 || next.Lifecycle != domain.LifecycleRunning {
		t.Fatalf("promoted wave = tick %d lifecycle %s", next.Tick(), next.Lifecycle)
	}
	if len(next.Members) != 1 || next.Members[0].ActorKey != actorKey('b') {
		t.Fatalf("pending registration was not promoted: %+v", next.Members)
	}
}
