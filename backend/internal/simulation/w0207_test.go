package simulation

import (
	"reflect"
	"testing"

	"github.com/hung98dev/ttgqldl/backend/internal/domain"
)

func TestXPAwardBoundariesAndCap(t *testing.T) {
	state := lockedAt(t, 200, domain.PhasePrep)
	state.Members = []domain.WaveMember{{ActorKey: uniqueActor(1), Role: domain.RoleBuilder, JoinedTick: 200}}
	state.XP = []domain.PlayerXP{{ActorKey: uniqueActor(1), XP: 295}}
	next, err := awardXP(state)
	if err != nil {
		t.Fatalf("award: %v", err)
	}
	if len(next.XP) != 1 || next.XP[0].XP != domain.CapSessionXP {
		t.Fatalf("xp = %+v, want cap 300", next.XP)
	}
	if !next.Members[0].XPAwarded {
		t.Fatal("member not marked awarded")
	}
	// A second call in the same wave awards nothing.
	again, err := awardXP(next)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(again.XP, next.XP) {
		t.Fatal("duplicate XP award")
	}

	// A member joining at 201 is not eligible at 200 but is at 400.
	late := lockedAt(t, 200, domain.PhasePrep)
	late.Members = []domain.WaveMember{{ActorKey: uniqueActor(2), Role: domain.RoleBuilder, JoinedTick: 201}}
	beforeXP := late.XP
	awarded, err := awardXP(late)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(awarded.XP, beforeXP) {
		t.Fatal("member joined at 201 awarded at 200")
	}
}

func TestVictoryRewardsAndProgression(t *testing.T) {
	state := lockedAt(t, 2800, domain.PhaseResults)
	for i := range state.Wave.Pools {
		state.Wave.Pools[i].HP = 0
	}
	state.Village.Resources = domain.Resources{Wood: 400, Stone: 250, Food: 300}
	state.Village.TotalWins = 2
	state.ConsecutiveLosses = 2
	next, events, err := settleWave(state)
	if err != nil {
		t.Fatal(err)
	}
	if next.Wave.Result != domain.WaveResultVictory {
		t.Fatalf("result = %s", next.Wave.Result)
	}
	if next.Village.Resources.Wood != 500 || next.Village.Resources.Stone != 310 || next.Village.Resources.Food != 350 {
		t.Fatalf("resources = %+v", next.Village.Resources)
	}
	if next.Village.TotalWins != 3 || next.Village.Level != 2 {
		t.Fatalf("wins/level = %d/%d", next.Village.TotalWins, next.Village.Level)
	}
	if next.ConsecutiveWins != 1 || next.ConsecutiveLosses != 0 {
		t.Fatalf("streaks = %d/%d", next.ConsecutiveWins, next.ConsecutiveLosses)
	}
	if len(events) != 1 || events[0].Type != EventWaveResult {
		t.Fatalf("events = %+v", events)
	}
	// Idempotent: a second settlement changes nothing and emits nothing.
	again, againEvents, err := settleWave(next)
	if err != nil || len(againEvents) != 0 || !reflect.DeepEqual(again, next) {
		t.Fatal("settlement is not idempotent")
	}
}

func TestBossWaveVictoryRequiresDeadBoss(t *testing.T) {
	state := lockedAt(t, 2800, domain.PhaseResults)
	state.Wave.Number = 5
	state.Wave.Ballots = nil
	state.Difficulty = domain.Difficulty{Level: 1}
	for i := range state.Wave.Pools {
		state.Wave.Pools[i].HP = 0
	}
	state.Wave.Boss = domain.BossPool{Exists: true, HP: 5, MaxHP: 10}
	next, _, err := settleWave(state)
	if err != nil {
		t.Fatal(err)
	}
	if next.Wave.Result != domain.WaveResultDefeat {
		t.Fatalf("boss alive but result = %s", next.Wave.Result)
	}
	state.Wave.Boss.HP = 0
	won, _, err := settleWave(state)
	if err != nil {
		t.Fatal(err)
	}
	if won.Wave.Result != domain.WaveResultVictory {
		t.Fatalf("dead boss result = %s", won.Wave.Result)
	}
}

func TestLossPenaltiesAndVoteCHalving(t *testing.T) {
	base := lockedAt(t, 2800, domain.PhaseResults)
	for i := range base.Wave.Pools {
		base.Wave.Pools[i].HP = 0
	}
	base.Wave.Boss = domain.BossPool{}
	base.Wave.Pools[0].HP = 1 // enemies remain -> loss
	base.Village.Resources = domain.Resources{Wood: 500, Stone: 250, Food: 300}
	base.Wave.Number = 2

	plain, _, err := settleWave(base)
	if err != nil {
		t.Fatal(err)
	}
	// floor(500/5)=100, 250/5=50, 300/5=60.
	if plain.Village.Resources.Wood != 400 || plain.Village.Resources.Stone != 200 || plain.Village.Resources.Food != 240 {
		t.Fatalf("penalties = %+v", plain.Village.Resources)
	}
	if plain.ConsecutiveLosses != base.ConsecutiveLosses+1 || plain.ConsecutiveWins != 0 {
		t.Fatalf("streaks = %d/%d", plain.ConsecutiveWins, plain.ConsecutiveLosses)
	}
	// No recovery floors without actual defeat.
	if plain.Village.TownhallHP != base.Village.TownhallHP {
		t.Fatal("recovery floor applied to a non-defeat loss")
	}

	withC := base
	withC.Wave.Vote = domain.VoteState{Closed: true, Selected: domain.VoteOptionC}
	halved, _, err := settleWave(withC)
	if err != nil {
		t.Fatal(err)
	}
	if halved.Village.Resources.Wood != 500-50 || halved.Village.Resources.Stone != 250-25 || halved.Village.Resources.Food != 300-30 {
		t.Fatalf("C penalties = %+v", halved.Village.Resources)
	}
}

func TestRecoveryFloorsOnlyAfterDefeat(t *testing.T) {
	state := lockedAt(t, 2800, domain.PhaseResults)
	for i := range state.Wave.Pools {
		state.Wave.Pools[i].HP = 1
	}
	state.Village.WallHP = 100
	state.Village.TownhallHP = 0
	state.Village.Resources = domain.Resources{Wood: 0, Stone: 10, Food: 10}
	state.Wave.Defeated = true
	next, _, err := settleWave(state)
	if err != nil {
		t.Fatal(err)
	}
	wallMax, _ := next.Village.WallMax()
	if next.Village.WallHP != 250 && next.Village.WallHP != wallMax {
		t.Fatalf("wall = %d, want at least 250", next.Village.WallHP)
	}
	if next.Village.TownhallHP != 250 {
		t.Fatalf("townhall = %d, want 250", next.Village.TownhallHP)
	}
	if next.Village.Resources.Wood != 100 {
		t.Fatalf("wood = %d, want 100", next.Village.Resources.Wood)
	}
	if next.Wave.Result != domain.WaveResultDefeat {
		t.Fatalf("result = %s", next.Wave.Result)
	}
}

func TestDefeatStopsFutureXPAndEconomy(t *testing.T) {
	state := lockedAt(t, 1799, domain.PhaseAssault)
	state.Village.WallHP = 0
	state.Village.TownhallHP = 1
	state, _, _, _, err := AdvanceWithRoles(state, RoleQueue{}, Input{Kind: KindAdvanceTick})
	if err != nil {
		t.Fatal(err)
	}
	if !state.Wave.Defeated {
		t.Fatalf("expected defeat, townhall=%d", state.Village.TownhallHP)
	}
	wood := state.Village.Resources.Wood
	xpBefore := state.XP
	at2799, _ := advanceWithRoles(t, state, RoleQueue{}, 2799)
	if at2799.Village.Resources.Wood != wood {
		t.Fatal("economy ran after defeat")
	}
	if !reflect.DeepEqual(at2799.XP, xpBefore) {
		t.Fatal("XP changed after defeat")
	}
	at3000, _ := advanceWithRoles(t, at2799, RoleQueue{}, 3000)
	if at3000.Wave.Result != domain.WaveResultDefeat {
		t.Fatalf("settled result = %s", at3000.Wave.Result)
	}
}

func TestVillageLevelProgression(t *testing.T) {
	cases := map[int64]int{0: 1, 2: 1, 3: 2, 6: 3, 147: 50, 150: 50}
	for wins, want := range cases {
		if got := villageLevel(wins); got != want {
			t.Fatalf("villageLevel(%d) = %d, want %d", wins, got, want)
		}
	}
}

func TestRolloverPreservesXPAndStreaks(t *testing.T) {
	state := lockedAt(t, 3000, domain.PhaseResults)
	state.Wave.Result = domain.WaveResultDefeat
	state.ConsecutiveWins = 0
	state.ConsecutiveLosses = 2
	state.XP = []domain.PlayerXP{{ActorKey: uniqueActor(1), XP: 30}}
	rolled, _, err := Next(state, Input{Kind: KindRollover})
	if err != nil {
		t.Fatalf("rollover: %v", err)
	}
	if len(rolled.XP) != 1 || rolled.XP[0].XP != 30 {
		t.Fatalf("xp lost at rollover: %+v", rolled.XP)
	}
	if rolled.ConsecutiveLosses != 2 || rolled.Village.TotalWins != state.Village.TotalWins {
		t.Fatal("streaks or lifetime wins changed at rollover")
	}
}

func TestSettlementRunsOnceAtFinalBoundary(t *testing.T) {
	state, _ := realWave(t)
	state = advanceTo(t, state, 2799)
	next, _, events, _, err := AdvanceWithRoles(state, RoleQueue{}, Input{Kind: KindAdvanceTick})
	if err != nil {
		t.Fatal(err)
	}
	if next.Tick() != 2800 {
		t.Fatalf("tick = %d", next.Tick())
	}
	results := 0
	for _, event := range events {
		if event.Type == EventWaveResult {
			results++
		}
	}
	if results != 1 {
		t.Fatalf("wave_result count = %d, events=%+v", results, eventTypes(events))
	}
	if next.Wave.Result == domain.WaveResultPending {
		t.Fatal("no settlement at 2800")
	}
}
