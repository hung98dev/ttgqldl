package domain

import (
	"errors"
	"math"
	"math/big"
	"testing"
)

// Regression tests for the W0201 remediation. Each case failed or would have
// panicked before the fix.

func TestSafeRatioZeroScaleDoesNotPanic(t *testing.T) {
	cases := []struct {
		num, den, scale int64
	}{
		{1, 2, 0},
		{0, 2, 0},
		{math.MaxInt64, math.MaxInt64, 0},
		{math.MaxInt64, 1, 0},
	}
	for _, tc := range cases {
		got, err := SafeRatio(tc.num, tc.den, tc.scale)
		if err != nil {
			t.Fatalf("SafeRatio(%d,%d,%d) error = %v, want nil", tc.num, tc.den, tc.scale, err)
		}
		if got != 0 {
			t.Fatalf("SafeRatio(%d,%d,%d) = %d, want 0", tc.num, tc.den, tc.scale, got)
		}
	}
}

func TestSafeRatioZeroDenominatorIsAnErrorNotAPanic(t *testing.T) {
	values := []int64{0, 1, math.MaxInt64}
	for _, num := range values {
		for _, scale := range values {
			got, err := SafeRatio(num, 0, scale)
			if !errors.Is(err, ErrDivisionByZero) {
				t.Fatalf("SafeRatio(%d,0,%d) error = %v, want ErrDivisionByZero", num, scale, err)
			}
			if got != 0 {
				t.Fatalf("SafeRatio(%d,0,%d) = %d, want 0", num, scale, got)
			}
		}
	}
}

func TestSafePercentNearMaximumIsExact(t *testing.T) {
	// floor((MaxInt64-1)*100/MaxInt64) = 99: the product overflows int64 but the
	// true result is representable.
	got, err := SafePercent(math.MaxInt64-1, math.MaxInt64)
	if err != nil {
		t.Fatalf("SafePercent near max: %v", err)
	}
	if got != 99 {
		t.Fatalf("SafePercent(MaxInt64-1, MaxInt64) = %d, want 99", got)
	}

	got, err = SafeRatio(math.MaxInt64-1, math.MaxInt64, 100)
	if err != nil {
		t.Fatalf("SafeRatio near max: %v", err)
	}
	if got != 99 {
		t.Fatalf("SafeRatio(MaxInt64-1, MaxInt64, 100) = %d, want 99", got)
	}
}

func TestSafeRatioGenuineOverflowIsReported(t *testing.T) {
	if _, err := SafeRatio(math.MaxInt64, 1, 2); !errors.Is(err, ErrOverflow) {
		t.Fatalf("expected ErrOverflow, got %v", err)
	}
	if _, err := SafeRatio(math.MaxInt64, 3, math.MaxInt64); !errors.Is(err, ErrOverflow) {
		t.Fatalf("expected ErrOverflow for a product exceeding int64, got %v", err)
	}
}

func TestSafeRatioMatchesBigIntOracle(t *testing.T) {
	values := []int64{0, 1, 2, 3, 99, 100, 115, 1000, math.MaxInt64 - 1, math.MaxInt64}
	for _, num := range values {
		for _, den := range values {
			if den == 0 {
				continue
			}
			for _, scale := range []int64{0, 1, 100, 115} {
				wantBig := new(big.Int).Mul(big.NewInt(num), big.NewInt(scale))
				wantBig.Quo(wantBig, big.NewInt(den))
				got, err := SafeRatio(num, den, scale)
				if !wantBig.IsInt64() {
					if !errors.Is(err, ErrOverflow) {
						t.Fatalf("SafeRatio(%d,%d,%d) = %d, %v; want overflow", num, den, scale, got, err)
					}
					continue
				}
				if err != nil {
					t.Fatalf("SafeRatio(%d,%d,%d): %v", num, den, scale, err)
				}
				if got != wantBig.Int64() {
					t.Fatalf("SafeRatio(%d,%d,%d) = %d, want %d", num, den, scale, got, wantBig.Int64())
				}
			}
		}
	}
}

func TestDefeatIsValidFromFirstCombatBoundary(t *testing.T) {
	cases := []struct {
		tick  int
		phase Phase
	}{
		{1800, PhaseAssault},
		{2000, PhaseAssault},
		{2200, PhaseAssault},
		{2400, PhaseAssault},
		{2600, PhaseFinale},
		{2700, PhaseFinale},
		{2800, PhaseResults},
		{2900, PhaseResults},
	}
	for _, tc := range cases {
		state := validState(t)
		state.Wave.Number = 3
		state.Pending = nil
		state.Wave.Tick = tc.tick
		state.Wave.Phase = tc.phase
		state.Wave.Defeated = true
		if err := state.Validate(); err != nil {
			t.Fatalf("defeat at tick %d must be representable: %v", tc.tick, err)
		}
	}
}

func TestDefeatDuringPrepOrBeforeCombatIsRejected(t *testing.T) {
	for _, tc := range []struct {
		tick  int
		phase Phase
	}{{0, PhasePrep}, {100, PhasePrep}, {1600, PhaseAssault}, {1799, PhaseAssault}} {
		state := validState(t)
		state.Wave.Tick = tc.tick
		state.Wave.Phase = tc.phase
		state.Wave.Defeated = true
		if err := state.Validate(); err == nil {
			t.Fatalf("defeat at tick %d must be rejected", tc.tick)
		}
	}
}

func TestLossVersusDefeat(t *testing.T) {
	// Loss with the town hall still standing (enemy pools survived settlement).
	lossOnly := validState(t)
	lossOnly.Wave.Tick = 2900
	lossOnly.Wave.Phase = PhaseResults
	lossOnly.Wave.Defeated = false
	lossOnly.Village.TownhallHP = 500
	lossOnly.Wave.Pools = []EnemyPool{{Lane: LaneWest, Archetype: ArchetypeRunner, HP: 10}}
	lossOnly.Wave.Result = WaveResultDefeat
	if err := lossOnly.Validate(); err != nil {
		t.Fatalf("loss with a standing town hall must be valid: %v", err)
	}

	// Loss after the town hall fell.
	lossDefeated := validState(t)
	lossDefeated.Wave.Tick = 2900
	lossDefeated.Wave.Phase = PhaseResults
	lossDefeated.Village.TownhallHP = 0
	lossDefeated.Wave.Defeated = true
	lossDefeated.Wave.Result = WaveResultDefeat
	if err := lossDefeated.Validate(); err != nil {
		t.Fatalf("loss after defeat must be valid: %v", err)
	}

	// Victory may never coexist with the defeated flag.
	victoryDefeated := validState(t)
	victoryDefeated.Wave.Tick = 2900
	victoryDefeated.Wave.Phase = PhaseResults
	victoryDefeated.Wave.Defeated = true
	victoryDefeated.Wave.Result = WaveResultVictory
	if err := victoryDefeated.Validate(); err == nil {
		t.Fatal("victory plus defeated must be rejected")
	}
}

func TestBossLifecycleStates(t *testing.T) {
	bossWave := func(t *testing.T, tick int, phase Phase, boss BossPool) SessionState {
		t.Helper()
		state := validState(t)
		state.Pending = nil
		state.Wave.Number = 5
		state.Wave.Tick = tick
		state.Wave.Phase = phase
		state.Wave.Boss = boss
		return state
	}

	// Before the spawn tick the boss does not exist; only the maximum may be known.
	preSpawn := validState(t)
	preSpawn.Pending = nil
	preSpawn.Wave.Number = 5
	preSpawn.Wave.Tick = 2000
	preSpawn.Wave.Phase = PhaseAssault
	preSpawn.Wave.Boss = BossPool{Exists: false, HP: 0, MaxHP: 20}
	if err := preSpawn.Validate(); err != nil {
		t.Fatalf("pre-spawn boss state must be valid: %v", err)
	}

	if err := bossWave(t, 2600, PhaseFinale, BossPool{Exists: true, HP: 20, MaxHP: 20}).Validate(); err != nil {
		t.Fatalf("newly spawned boss must be valid: %v", err)
	}
	if err := bossWave(t, 2600, PhaseFinale, BossPool{Exists: true, HP: 0, MaxHP: 20}).Validate(); err != nil {
		t.Fatalf("boss killed at its spawn tick must be valid: %v", err)
	}
	if err := bossWave(t, 2800, PhaseResults, BossPool{Exists: true, HP: 0, MaxHP: 20}).Validate(); err != nil {
		t.Fatalf("boss killed at final combat must be valid: %v", err)
	}
	if err := bossWave(t, 2900, PhaseResults, BossPool{Exists: true, HP: 0, MaxHP: 20}).Validate(); err != nil {
		t.Fatalf("dead boss during results must be valid: %v", err)
	}
	if err := bossWave(t, 2700, PhaseFinale, BossPool{Exists: true, HP: 21, MaxHP: 20}).Validate(); err == nil {
		t.Fatal("boss HP above maximum must be rejected")
	}
	if err := bossWave(t, 2700, PhaseFinale, BossPool{Exists: true, HP: 0, MaxHP: 0}).Validate(); err == nil {
		t.Fatal("spawned boss without a positive maximum must be rejected")
	}

	nonBoss := validState(t)
	nonBoss.Pending = nil
	nonBoss.Wave.Tick = 2700
	nonBoss.Wave.Phase = PhaseFinale
	nonBoss.Wave.Boss = BossPool{Exists: true, HP: 5, MaxHP: 10}
	if err := nonBoss.Validate(); err == nil {
		t.Fatal("boss on a non-boss wave must be rejected")
	}
}

func TestWaveMemberFirstSwitchSemantics(t *testing.T) {
	base := validState(t)

	// No change yet: nil last change and the free switch still available.
	fresh := base
	fresh.Members = []WaveMember{{ActorKey: actorOne, Role: RoleBuilder, JoinedTick: 0}}
	if err := fresh.Validate(); err != nil {
		t.Fatalf("fresh member must be valid: %v", err)
	}

	// First accepted change: timestamp plus used flag.
	switched := base
	switched.Members = []WaveMember{{
		ActorKey: actorOne, Role: RoleGuard, JoinedTick: 0,
		LastChangeTick: intPtr(50), FirstSwitchUsed: true,
	}}
	if err := switched.Validate(); err != nil {
		t.Fatalf("member after first switch must be valid: %v", err)
	}

	// Joined tick is preserved independently from the change tick.
	if switched.Members[0].JoinedTick != 0 {
		t.Fatal("join tick must survive a role change")
	}

	// Invalid combinations.
	flagWithoutChange := base
	flagWithoutChange.Members = []WaveMember{{ActorKey: actorOne, Role: RoleBuilder, JoinedTick: 0, FirstSwitchUsed: true}}
	if err := flagWithoutChange.Validate(); err == nil {
		t.Fatal("first_switch_used without a recorded change must be rejected")
	}

	changeWithoutFlag := base
	changeWithoutFlag.Members = []WaveMember{{
		ActorKey: actorOne, Role: RoleBuilder, JoinedTick: 0, LastChangeTick: intPtr(10),
	}}
	if err := changeWithoutFlag.Validate(); err == nil {
		t.Fatal("a recorded change without first_switch_used must be rejected")
	}
}

func TestPendingTargetMustBeExactlyNextWave(t *testing.T) {
	base := validState(t)
	for _, target := range []int64{3, 5, 10} {
		state := base
		state.Pending = []PendingMember{{ActorKey: actorKeyFor(1), Role: RoleGuard, TargetWave: target}}
		if err := state.Validate(); err == nil {
			t.Fatalf("pending target wave %d must be rejected when the current wave is 3", target)
		}
	}
}

func TestZeroValueStateIsRejected(t *testing.T) {
	var state SessionState
	if err := state.Validate(); err == nil {
		t.Fatal("the zero-value state must never validate")
	}
}

func FuzzSafeRatioNoPanic(f *testing.F) {
	f.Add(int64(1), int64(2), int64(100))
	f.Add(int64(0), int64(0), int64(0))
	f.Add(int64(math.MaxInt64), int64(math.MaxInt64), int64(115))
	f.Fuzz(func(t *testing.T, num, den, scale int64) {
		got, err := SafeRatio(num, den, scale)
		if den == 0 {
			if !errors.Is(err, ErrDivisionByZero) {
				t.Fatalf("den=0 must return ErrDivisionByZero, got %v", err)
			}
			return
		}
		if den < 0 || num < 0 || scale < 0 {
			if !errors.Is(err, ErrNegative) {
				t.Fatalf("negative operands must return ErrNegative, got %v", err)
			}
			return
		}
		want := new(big.Int).Mul(big.NewInt(num), big.NewInt(scale))
		want.Quo(want, big.NewInt(den))
		if !want.IsInt64() {
			if !errors.Is(err, ErrOverflow) {
				t.Fatalf("expected overflow for %d*%d/%d", num, scale, den)
			}
			return
		}
		if err != nil {
			t.Fatalf("unexpected error %v for %d*%d/%d", err, num, scale, den)
		}
		if got != want.Int64() {
			t.Fatalf("SafeRatio(%d,%d,%d) = %d, want %d", num, den, scale, got, want.Int64())
		}
	})
}
