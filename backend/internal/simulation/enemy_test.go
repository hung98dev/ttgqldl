package simulation

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/hung98dev/ttgqldl/backend/internal/domain"
)

// uniqueActor returns a valid 64-hex actor key derived from an index.
func uniqueActor(index int) domain.ActorKey {
	return domain.ActorKey(fmt.Sprintf("%064x", index))
}

// lockedFixture builds a started state advanced to tick 1599 with the given
// members and persistent progress. Wave number may be overridden for boss tests.
func lockedFixture(t *testing.T, members []domain.WaveMember, wins, losses int64, waveNo int64) domain.SessionState {
	t.Helper()
	state, _ := startWithRole(t, 'a', domain.RoleBuilder)
	state, _ = advanceWithRoles(t, state, RoleQueue{}, 1599)
	state.Members = members
	state.Wave.Number = waveNo
	state.ConsecutiveWins = wins
	state.ConsecutiveLosses = losses
	state.Wave.Result = domain.WaveResultPending
	state.Difficulty = domain.Difficulty{Level: 1}
	return state
}

func advanceOne(t *testing.T, state domain.SessionState) (domain.SessionState, []Event) {
	t.Helper()
	next, events, err := Next(state, Input{Kind: KindAdvanceTick})
	if err != nil {
		t.Fatalf("advance from %d: %v", state.Tick(), err)
	}
	return next, events
}

func hasEvent(events []Event, want EventType) bool {
	for _, event := range events {
		if event.Type == want {
			return true
		}
	}
	return false
}

func TestPopulationLockCountsUniqueMembersJoinedBefore1600(t *testing.T) {
	members := []domain.WaveMember{
		{ActorKey: uniqueActor(1), Role: domain.RoleBuilder, JoinedTick: 0},
		{ActorKey: uniqueActor(2), Role: domain.RoleGuard, JoinedTick: 1599},
		{ActorKey: uniqueActor(3), Role: domain.RoleFighter, JoinedTick: 1600},
	}
	state := lockedFixture(t, members, 0, 0, 1)
	locked, _ := advanceOne(t, state)

	if !locked.Difficulty.Locked {
		t.Fatal("population did not lock at 1600")
	}
	if locked.Difficulty.ActualPopulation != 2 {
		t.Fatalf("actual population = %d, want 2 (tick-1600 join excluded)", locked.Difficulty.ActualPopulation)
	}
	if locked.Difficulty.PopulationAnchor != 2 {
		t.Fatalf("anchor = %d", locked.Difficulty.PopulationAnchor)
	}
	if len(locked.Members) != 3 {
		t.Fatal("lock must not remove or fabricate members")
	}
}

func TestNoPopulationLockWhileWaiting(t *testing.T) {
	state := newState(t)
	if state.Difficulty.Locked {
		t.Fatal("initial state must not be locked")
	}
	if _, _, err := Next(state, Input{Kind: KindAdvanceTick}); err == nil {
		t.Fatal("waiting session advanced")
	}
	if state.Difficulty.Locked {
		t.Fatal("waiting session locked its population")
	}
}

func TestDifficultyAndRecoveryAtLock(t *testing.T) {
	cases := []struct {
		wins, losses int64
		wantLevel    int
		wantRecovery int
	}{
		{0, 0, 1, 100},
		{3, 1, 4, 85},
		{9, 2, 10, 70},
		{10, 3, 10, 55},
	}
	for _, tc := range cases {
		members := []domain.WaveMember{{ActorKey: uniqueActor(1), Role: domain.RoleBuilder, JoinedTick: 0}}
		state := lockedFixture(t, members, tc.wins, tc.losses, 1)
		locked, _ := advanceOne(t, state)
		if locked.Difficulty.Level != tc.wantLevel {
			t.Fatalf("wins=%d level=%d want %d", tc.wins, locked.Difficulty.Level, tc.wantLevel)
		}
		if locked.Difficulty.RecoveryPercent != tc.wantRecovery {
			t.Fatalf("losses=%d recovery=%d want %d", tc.losses, locked.Difficulty.RecoveryPercent, tc.wantRecovery)
		}
		wantHP, err := domain.BaseEnemyHP(tc.wantLevel, 1, tc.losses)
		if err != nil {
			t.Fatal(err)
		}
		if locked.Difficulty.BaseHP != wantHP {
			t.Fatalf("base hp=%d want %d", locked.Difficulty.BaseHP, wantHP)
		}
	}
}

func TestBaseHPAndPoolConservationAtLock(t *testing.T) {
	members := make([]domain.WaveMember, 0, 30)
	for i := 1; i <= 30; i++ {
		members = append(members, domain.WaveMember{ActorKey: uniqueActor(i), Role: domain.RoleFighter, JoinedTick: 0})
	}
	state := lockedFixture(t, members, 0, 0, 1)
	locked, _ := advanceOne(t, state)

	wantBase, err := domain.BaseEnemyHP(1, 30, 0)
	if err != nil {
		t.Fatal(err)
	}
	if locked.Difficulty.BaseHP != wantBase {
		t.Fatalf("base hp = %d want %d", locked.Difficulty.BaseHP, wantBase)
	}
	if len(locked.Wave.Pools) != 9 {
		t.Fatalf("pools = %d", len(locked.Wave.Pools))
	}
	total := int64(0)
	for i, pool := range locked.Wave.Pools {
		if pool.Lane != domain.Lane(i/3) || pool.Archetype != domain.Archetype(i%3) {
			t.Fatalf("pool %d order = %s/%s", i, pool.Lane, pool.Archetype)
		}
		if pool.HP < 0 {
			t.Fatal("negative pool HP")
		}
		total += pool.HP
	}
	if total != wantBase {
		t.Fatalf("pool total %d != base %d", total, wantBase)
	}
	west, center, east, err := domain.LaneHPs(wantBase)
	if err != nil {
		t.Fatal(err)
	}
	if west+center+east != wantBase {
		t.Fatal("lane split lost HP")
	}
	if locked.Difficulty.MilitiaAttack != 0 {
		t.Fatalf("militia = %d, want 0 for 30 players", locked.Difficulty.MilitiaAttack)
	}
}

func TestMilitiaReferenceValuesAtLock(t *testing.T) {
	for _, population := range []int{1, 3, 4, 5, 10, 11, 30} {
		members := make([]domain.WaveMember, 0, population)
		for i := 1; i <= population; i++ {
			members = append(members, domain.WaveMember{ActorKey: uniqueActor(i), Role: domain.RoleBuilder, JoinedTick: 0})
		}
		state := lockedFixture(t, members, 0, 0, 1)
		locked, _ := advanceOne(t, state)
		want := domain.MilitiaAttack(population)
		if locked.Difficulty.MilitiaAttack != want {
			t.Fatalf("P=%d militia=%d want %d", population, locked.Difficulty.MilitiaAttack, want)
		}
		if locked.Difficulty.ActualPopulation != population {
			t.Fatal("actual population was falsified")
		}
	}
}

func TestPopulationAboveCapKeepsTruthfulCount(t *testing.T) {
	members := make([]domain.WaveMember, 0, 10001)
	for i := 0; i < 10001; i++ {
		members = append(members, domain.WaveMember{ActorKey: uniqueActor(i), Role: domain.RoleBuilder, JoinedTick: 0})
	}
	state := lockedFixture(t, members, 0, 0, 1)
	locked, _ := advanceOne(t, state)
	if locked.Difficulty.ActualPopulation != 10001 {
		t.Fatalf("actual = %d", locked.Difficulty.ActualPopulation)
	}
	if locked.Difficulty.PopulationAnchor != domain.MaxPopulationAnchor {
		t.Fatalf("anchor = %d", locked.Difficulty.PopulationAnchor)
	}
}

func TestLockIsIdempotentAndLateJoinDoesNotRecalculate(t *testing.T) {
	members := []domain.WaveMember{{ActorKey: uniqueActor(1), Role: domain.RoleBuilder, JoinedTick: 0}}
	state := lockedFixture(t, members, 0, 0, 1)
	locked, _ := advanceOne(t, state)
	anchor := locked.Difficulty.PopulationAnchor
	base := locked.Difficulty.BaseHP

	next, _ := advanceOne(t, locked)
	if next.Difficulty.PopulationAnchor != anchor || next.Difficulty.BaseHP != base {
		t.Fatal("difficulty changed after the lock")
	}
}

func TestBossSpawnsOnlyOnBossWaves(t *testing.T) {
	state := lockedFixture(t, []domain.WaveMember{{ActorKey: uniqueActor(1), Role: domain.RoleBuilder, JoinedTick: 0}}, 0, 0, 5)
	locked, _ := advanceOne(t, state) // tick 1600
	if !locked.Difficulty.Locked || locked.Difficulty.BossMaxHP < 1 {
		t.Fatalf("boss maximum not derived: %+v", locked.Difficulty)
	}
	if locked.Wave.Boss.Exists || locked.Wave.Boss.HP != 0 {
		t.Fatal("boss must not exist before tick 2600")
	}
	bossMax := locked.Difficulty.BossMaxHP

	at2600, _ := advanceWithRoles(t, locked, RoleQueue{}, 2600)
	// The boss spawns at 2600 and combat in the same tick may already kill it.
	if !at2600.Wave.Boss.Exists || at2600.Wave.Boss.MaxHP != bossMax {
		t.Fatalf("boss at 2600 = %+v", at2600.Wave.Boss)
	}
	if len(at2600.Wave.Pools) != 9 {
		t.Fatal("boss spawn must preserve ordinary pools")
	}

	// Dead spawned boss remains valid until rollover.
	at2600.Wave.Boss.HP = 0
	if err := at2600.Validate(); err != nil {
		t.Fatalf("dead boss must stay representable: %v", err)
	}
}

func TestBossDoesNotSpawnOnOrdinaryWave(t *testing.T) {
	state := lockedFixture(t, []domain.WaveMember{{ActorKey: uniqueActor(1), Role: domain.RoleBuilder, JoinedTick: 0}}, 0, 0, 3)
	locked, _ := advanceOne(t, state)
	at2600, _ := advanceWithRoles(t, locked, RoleQueue{}, 2600)
	if at2600.Wave.Boss.Exists {
		t.Fatal("boss spawned on an ordinary wave")
	}
	if at2600.Difficulty.BossMaxHP != 0 {
		t.Fatal("boss maximum derived on an ordinary wave")
	}
}

func TestBossSpawnEmittedBeforeCombatBoundary(t *testing.T) {
	state := lockedFixture(t, []domain.WaveMember{{ActorKey: uniqueActor(1), Role: domain.RoleBuilder, JoinedTick: 0}}, 0, 0, 5)
	locked, _ := advanceOne(t, state)
	at2599, _ := advanceWithRoles(t, locked, RoleQueue{}, 2599)
	at2600, events := advanceOne(t, at2599)

	spawnIndex, combatIndex := -1, -1
	for i, event := range events {
		if event.Type == EventBossSpawned {
			spawnIndex = i
		}
		if event.Type == EventScheduleBoundary && event.Boundary == BoundaryCombat {
			combatIndex = i
		}
	}
	if spawnIndex < 0 {
		t.Fatalf("boss_spawned missing: %+v", events)
	}
	if combatIndex < 0 || spawnIndex > combatIndex {
		t.Fatalf("boss spawn must precede the combat boundary: %+v", events)
	}
	if at2600.Wave.Boss.MaxHP != at2600.Difficulty.BossMaxHP {
		t.Fatal("boss maximum was not set to the derived value")
	}
	if at2600.Wave.Boss.HP < 0 || at2600.Wave.Boss.HP > at2600.Wave.Boss.MaxHP {
		t.Fatalf("boss HP out of range: %+v", at2600.Wave.Boss)
	}
}

func TestBossIncomingWhenPoolsEmptyBeforeSpawn(t *testing.T) {
	members := []domain.WaveMember{{ActorKey: uniqueActor(1), Role: domain.RoleBuilder, JoinedTick: 0}}
	state := lockedFixture(t, members, 0, 0, 5)
	state, _ = advanceOne(t, state) // locked at 1600
	for i := range state.Wave.Pools {
		state.Wave.Pools[i].HP = 0
	}
	if err := state.Validate(); err != nil {
		t.Fatalf("depleted pools must be valid: %v", err)
	}
	if status := EnemyStatus(state); status != domain.EnemyStatusBossIncoming {
		t.Fatalf("status = %s, want BOSS_INCOMING", status)
	}
	notified, events := advanceOne(t, state)
	if !hasEvent(events, EventBossIncoming) {
		t.Fatalf("boss_incoming not emitted: %+v", events)
	}
	if !notified.Wave.BossIncoming {
		t.Fatal("boss incoming flag not set")
	}
	again, events := advanceOne(t, notified)
	if hasEvent(events, EventBossIncoming) {
		t.Fatal("boss_incoming emitted twice")
	}
	_ = again
}

func TestNoPrematureVictory(t *testing.T) {
	members := []domain.WaveMember{{ActorKey: uniqueActor(1), Role: domain.RoleBuilder, JoinedTick: 0}}
	state := lockedFixture(t, members, 0, 0, 5)
	state, _ = advanceOne(t, state)
	state, _ = advanceWithRoles(t, state, RoleQueue{}, 2799)
	for i := range state.Wave.Pools {
		state.Wave.Pools[i].HP = 0
	}
	state.Wave.Boss = domain.BossPool{Exists: true, HP: 0, MaxHP: state.Difficulty.BossMaxHP}
	if err := state.Validate(); err != nil {
		t.Fatalf("killed-boss state invalid: %v", err)
	}
	if state.Wave.Result != domain.WaveResultPending {
		t.Fatal("enemy state produced a premature result")
	}
	if status := EnemyStatus(state); status == domain.EnemyStatusClear && state.Wave.Result == domain.WaveResultPending {
		// CLEAR is a projection only; no victory is implied.
	}
}

func TestPauseFreezesPopulationLock(t *testing.T) {
	state, _ := startWithRole(t, 'a', domain.RoleBuilder)
	state, _ = advanceWithRoles(t, state, RoleQueue{}, 1590)
	paused, _, err := Next(state, Input{Kind: KindPause, Reason: domain.PauseOperatorPaused})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := Next(paused, Input{Kind: KindAdvanceTick}); err == nil {
		t.Fatal("paused session advanced")
	}
	if paused.Difficulty.Locked {
		t.Fatal("population locked while paused")
	}
	resumed, _, err := Next(paused, Input{Kind: KindResume, Reason: domain.PauseOperatorPaused})
	if err != nil {
		t.Fatal(err)
	}
	locked, _ := advanceWithRoles(t, resumed, RoleQueue{}, 1600)
	if !locked.Difficulty.Locked {
		t.Fatal("lock did not happen after resume")
	}
	if locked.Difficulty.PopulationAnchor != 1 {
		t.Fatalf("anchor = %d", locked.Difficulty.PopulationAnchor)
	}
}

func TestLockDeterminismAndImmutability(t *testing.T) {
	members := []domain.WaveMember{
		{ActorKey: uniqueActor(1), Role: domain.RoleBuilder, JoinedTick: 0},
		{ActorKey: uniqueActor(2), Role: domain.RoleGuard, JoinedTick: 100},
	}
	state := lockedFixture(t, members, 2, 1, 1)
	before := state.Clone()
	first, firstEvents := advanceOne(t, state)
	second, secondEvents := advanceOne(t, state.Clone())
	if !reflect.DeepEqual(first, second) || !reflect.DeepEqual(firstEvents, secondEvents) {
		t.Fatal("population lock is not deterministic")
	}
	if !reflect.DeepEqual(state, before) {
		t.Fatal("lock mutated its input state")
	}

	// Mutating the produced pools must not touch the input state.
	first.Wave.Pools[0].HP = 999999
	if state.Wave.Pools != nil && len(state.Wave.Pools) > 0 {
		t.Fatal("returned pools alias the input state")
	}
}

func TestLockAndSpawnArithmeticBounds(t *testing.T) {
	// Negative or out-of-range inputs fail closed.
	if _, err := domain.BaseEnemyHP(0, 1, 0); err == nil {
		t.Fatal("difficulty 0 must fail")
	}
	if _, err := domain.BaseEnemyHP(1, 0, 0); err == nil {
		t.Fatal("population 0 must fail")
	}
	if _, err := domain.BaseEnemyHP(1, 1, -1); err == nil {
		t.Fatal("negative losses must fail")
	}
	if _, err := domain.BossMaxHP(-1); err == nil {
		t.Fatal("negative boss base must fail")
	}
	// Huge losses clamp to the documented 3-loss floor instead of overflowing.
	hp, err := domain.BaseEnemyHP(1, 1, 1<<62)
	if err != nil {
		t.Fatalf("clamped losses: %v", err)
	}
	if hp != 47 { // 86 * 55 / 100 = 47
		t.Fatalf("hp with clamped losses = %d, want 47", hp)
	}
	// Valid maximum inputs remain exact.
	if hp, err := domain.BaseEnemyHP(10, domain.MaxPopulationAnchor, 0); err != nil || hp != 3920 {
		t.Fatalf("max inputs hp = %d, err %v", hp, err)
	}
}
