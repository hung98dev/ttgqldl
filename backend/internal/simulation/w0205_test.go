package simulation

import (
	"reflect"
	"testing"

	"github.com/hung98dev/ttgqldl/backend/internal/domain"
)

// lockedAt returns a locked state positioned at an arbitrary tick for direct
// economy/combat tests. The derived difficulty comes from the real lock at 1600.
func lockedAt(t *testing.T, tick int, phase domain.Phase) domain.SessionState {
	t.Helper()
	state, _ := startWithRole(t, 'a', domain.RoleBuilder)
	state, _ = advanceWithRoles(t, state, RoleQueue{}, domain.PopulationLockTick)
	state.Wave.Tick = tick
	state.Wave.Phase = phase
	if err := state.Validate(); err != nil {
		t.Fatalf("locked fixture invalid: %v", err)
	}
	return state
}

func TestEconomyGeneratesExactResources(t *testing.T) {
	state := lockedAt(t, 200, domain.PhasePrep)
	state.Members = []domain.WaveMember{
		{ActorKey: uniqueActor(1), Role: domain.RoleBuilder, JoinedTick: 0},
		{ActorKey: uniqueActor(2), Role: domain.RoleBuilder, JoinedTick: 0},
		{ActorKey: uniqueActor(3), Role: domain.RoleBuilder, JoinedTick: 0},
		{ActorKey: uniqueActor(4), Role: domain.RoleBuilder, JoinedTick: 0},
	}
	state.Village.WallHP = 900
	state.Village.ConstructionWork = 0
	state.Village.Resources = domain.Resources{Wood: 400, Stone: 250, Food: 300}
	var events []Event
	next, err := resolveContribution(state, &events)
	if err != nil {
		t.Fatalf("contribution: %v", err)
	}
	// B=4: wood +4, stone +2; capacity 16, repairable 100, affordable 2020.
	if next.Village.Resources.Wood != 400 {
		t.Fatalf("wood = %d, want 400 (400+4 gain-4 repair cost)", next.Village.Resources.Wood)
	}
	if next.Village.Resources.Stone != 252 {
		t.Fatalf("stone = %d, want 252", next.Village.Resources.Stone)
	}
	if next.Village.WallHP != 916 {
		t.Fatalf("wall hp = %d, want 916", next.Village.WallHP)
	}
	if next.Village.ConstructionWork != 0 {
		t.Fatalf("construction work = %d, want 0", next.Village.ConstructionWork)
	}
}

func TestZeroBuildersGenerateNothing(t *testing.T) {
	state := lockedAt(t, 200, domain.PhasePrep)
	state.Members = []domain.WaveMember{{ActorKey: uniqueActor(1), Role: domain.RoleFighter, JoinedTick: 0}}
	state.Village.Resources = domain.Resources{Wood: 10, Stone: 10, Food: 10}
	next, err := resolveContribution(state, nil)
	if err != nil {
		t.Fatal(err)
	}
	if next.Village.Resources != state.Village.Resources || next.Village.WallHP != state.Village.WallHP {
		t.Fatal("zero builders produced resources or repair")
	}
}

func TestUpgradeConsumesWorkAndStoneOnce(t *testing.T) {
	state := lockedAt(t, 200, domain.PhasePrep)
	members := make([]domain.WaveMember, 0, 100)
	for i := 1; i <= 100; i++ {
		members = append(members, domain.WaveMember{ActorKey: uniqueActor(i), Role: domain.RoleBuilder, JoinedTick: 0})
	}
	state.Members = members
	state.Village.WallHP = 1000
	state.Village.ConstructionWork = 120
	state.Village.Resources = domain.Resources{Wood: 500, Stone: 250, Food: 0}
	var events []Event
	next, err := resolveContribution(state, &events)
	if err != nil {
		t.Fatal(err)
	}
	// B=100: capacity 80, repairable 0 -> unused 80; work 120+80=200 = threshold.
	if next.Village.WallTier != 2 {
		t.Fatalf("tier = %d, want 2", next.Village.WallTier)
	}
	if next.Village.Resources.Stone != 160 {
		t.Fatalf("stone = %d, want 160 (250 + 10 gain - 100 upgrade)", next.Village.Resources.Stone)
	}
	if next.Village.ConstructionWork != 0 {
		t.Fatalf("work = %d", next.Village.ConstructionWork)
	}
	if next.Village.WallHP != 1200 {
		t.Fatalf("wall hp = %d, want 1200", next.Village.WallHP)
	}
	if len(events) != 1 || events[0].Type != EventWallUpgraded {
		t.Fatalf("events = %+v", events)
	}
}

func TestNoUpgradeWithoutStoneOrAtMaxTier(t *testing.T) {
	state := lockedAt(t, 200, domain.PhasePrep)
	members := make([]domain.WaveMember, 0, 100)
	for i := 1; i <= 100; i++ {
		members = append(members, domain.WaveMember{ActorKey: uniqueActor(i), Role: domain.RoleBuilder, JoinedTick: 0})
	}
	state.Members = members
	state.Village.ConstructionWork = 199
	state.Village.Resources = domain.Resources{Wood: 500, Stone: 0, Food: 0}
	next, err := resolveContribution(state, nil)
	if err != nil {
		t.Fatal(err)
	}
	if next.Village.WallTier != 1 {
		t.Fatal("upgraded without stone")
	}

	maxed := lockedAt(t, 200, domain.PhasePrep)
	maxed.Members = members
	maxed.Village.WallTier = domain.MaxWallTier
	maxed.Village.WallHP = 4800
	maxed.Village.Resources = domain.Resources{Wood: 100, Stone: 100000, Food: 0}
	maxed2, err := resolveContribution(maxed, nil)
	if err != nil {
		t.Fatalf("tier20 contribution: %v", err)
	}
	if maxed2.Village.WallTier != domain.MaxWallTier {
		t.Fatal("invalid tier 21")
	}
}

func TestAttackDistributionRemainderRotation(t *testing.T) {
	for _, tc := range []struct {
		index      int
		wantWest   int64
		wantCenter int64
		wantEast   int64
	}{
		{0, 34, 33, 33},
		{1, 33, 34, 33},
		{2, 33, 33, 34},
	} {
		state := lockedAt(t, 1800, domain.PhaseAssault)
		for i := range state.Wave.Pools {
			state.Wave.Pools[i].HP = 1000
		}
		next, err := distributeAttack(state, 100, tc.index)
		if err != nil {
			t.Fatal(err)
		}
		damage := map[int64]int64{}
		for i, pool := range state.Wave.Pools {
			damage[int64(pool.Lane)] += state.Wave.Pools[i].HP - next.Wave.Pools[i].HP
		}
		if damage[int64(domain.LaneWest)] != tc.wantWest ||
			damage[int64(domain.LaneCenter)] != tc.wantCenter ||
			damage[int64(domain.LaneEast)] != tc.wantEast {
			t.Fatalf("index %d damage = %v", tc.index, damage)
		}
	}
}

func TestAttackPriorityAndRoundTwoRedistribution(t *testing.T) {
	state := lockedAt(t, 1800, domain.PhaseAssault)
	for i := range state.Wave.Pools {
		state.Wave.Pools[i].HP = 100
	}
	// Empty the west lane so its share must be redistributed.
	for i := range state.Wave.Pools {
		if state.Wave.Pools[i].Lane == domain.LaneWest {
			state.Wave.Pools[i].HP = 0
		}
	}
	next, err := distributeAttack(state, 99, 0)
	if err != nil {
		t.Fatal(err)
	}
	// Lane order in Pools: west runner/grunt/brute, center runner/grunt/brute.
	centerRunner := 3
	if next.Wave.Pools[centerRunner].HP != 34 {
		t.Fatalf("center runner = %d, want 34 (33 own share + 33 redistributed)", next.Wave.Pools[centerRunner].HP)
	}
	if next.Wave.Pools[4].HP != 100 {
		t.Fatalf("center grunt = %d, want untouched at 100", next.Wave.Pools[4].HP)
	}
	// Within-lane priority: runner before grunt before brute.
	if next.Wave.Pools[5].HP != 100 {
		t.Fatalf("center brute = %d, want untouched", next.Wave.Pools[5].HP)
	}
}

func TestBossKillableByRedistributedDamage(t *testing.T) {
	state := lockedAt(t, 2600, domain.PhaseFinale)
	state.Wave.Number = 5
	for i := range state.Wave.Pools {
		state.Wave.Pools[i].HP = 0
	}
	state.Wave.Boss = domain.BossPool{Exists: true, HP: 10, MaxHP: 10}
	next, err := distributeAttack(state, 99, 4)
	if err != nil {
		t.Fatal(err)
	}
	if next.Wave.Boss.HP != 0 {
		t.Fatalf("boss hp = %d, want 0", next.Wave.Boss.HP)
	}
}

func TestRetaliationGuardAndWallFirstDamage(t *testing.T) {
	state := lockedAt(t, 1800, domain.PhaseAssault)
	state.Members = []domain.WaveMember{
		{ActorKey: uniqueActor(1), Role: domain.RoleFighter, JoinedTick: 0},
		{ActorKey: uniqueActor(2), Role: domain.RoleGuard, JoinedTick: 0},
	}
	state.Village.WallHP = 1000
	state.Village.TownhallHP = 500
	var events []Event
	next, err := resolveCombat(state, &events)
	if err != nil {
		t.Fatal(err)
	}
	if next.Wave.Defeated {
		t.Fatal("wave defeated unexpectedly")
	}
	if next.Village.WallHP > 1000 || next.Village.TownhallHP > 500 {
		t.Fatal("combat increased defensive HP")
	}
	if next.Village.TownhallHP < 500 && next.Village.WallHP != 0 {
		t.Fatal("townhall damaged while wall stood")
	}
	// Attack 18: isqrt(1)=1; militia 40 -> total 58 recorded.
	if len(events) == 0 || events[0].Type != EventAttackResolved || events[0].Amount != 58 {
		t.Fatalf("attack event = %+v", events)
	}
}

func TestEarlyDefeatStopsAllFurtherEffects(t *testing.T) {
	state := lockedAt(t, 1800, domain.PhaseAssault)
	state.Village.WallHP = 0
	state.Village.TownhallHP = 5
	next, err := resolveCombat(state, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !next.Wave.Defeated || next.Village.TownhallHP != 0 {
		t.Fatalf("defeat state = defeated=%v townhall=%d", next.Wave.Defeated, next.Village.TownhallHP)
	}
	// After defeat, economy and combat are inert.
	var events []Event
	afterEconomy, err := resolveContribution(next, &events)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 0 || !reflect.DeepEqual(afterEconomy.Village, next.Village) {
		t.Fatal("economy ran after defeat")
	}
	afterCombat, err := resolveCombat(next, &events)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 0 || !reflect.DeepEqual(afterCombat.Village, next.Village) {
		t.Fatal("combat ran after defeat")
	}
}

func TestFinalCombatRunsOnceAndLeavesResultPending(t *testing.T) {
	state := lockedAt(t, 2799, domain.PhaseFinale)
	next, _, events, _, err := AdvanceWithRoles(state, RoleQueue{}, Input{Kind: KindAdvanceTick})
	if err != nil {
		t.Fatal(err)
	}
	if next.Tick() != 2800 {
		t.Fatalf("tick = %d", next.Tick())
	}
	attackEvents := 0
	for _, event := range events {
		if event.Type == EventAttackResolved {
			attackEvents++
		}
	}
	if attackEvents != 1 {
		t.Fatalf("attack_resolved count = %d", attackEvents)
	}
	if next.Wave.Result != domain.WaveResultPending {
		t.Fatal("W0205 must not settle the wave")
	}
	// Tick 3000 is control only.
	at3000, _ := advanceWithRoles(t, next, RoleQueue{}, 3000)
	control, events, err := Next(at3000, Input{Kind: KindAdvanceTick})
	if err == nil {
		t.Fatal("tick 3000 must not advance")
	}
	_ = control
	_ = events
}

func TestPausedWaveCannotResolveGameplay(t *testing.T) {
	state := lockedAt(t, 1800, domain.PhaseAssault)
	paused, _, err := Next(state, Input{Kind: KindPause, Reason: domain.PauseOperatorPaused})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := Next(paused, Input{Kind: KindAdvanceTick}); err == nil {
		t.Fatal("paused wave advanced")
	}
	var events []Event
	unchanged, err := resolveCombat(paused, &events)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 0 || !reflect.DeepEqual(unchanged.Village, paused.Village) {
		t.Fatal("combat ran while paused")
	}
}
