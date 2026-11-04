package domain

import "testing"

func TestLockedDifficulty(t *testing.T) {
	cases := map[int64]int{0: 1, 1: 2, 3: 4, 9: 10, 10: 10, 50: 10}
	for wins, want := range cases {
		got, err := LockedDifficulty(wins)
		if err != nil || got != want {
			t.Fatalf("LockedDifficulty(%d) = %d, %v; want %d", wins, got, err, want)
		}
	}
	if _, err := LockedDifficulty(-1); err == nil {
		t.Fatal("negative wins must fail")
	}
}

func TestRecoveryPercent(t *testing.T) {
	cases := map[int64]int{0: 100, 1: 85, 2: 70, 3: 55, 4: 55}
	for losses, want := range cases {
		got, err := RecoveryPercent(losses)
		if err != nil || got != want {
			t.Fatalf("RecoveryPercent(%d) = %d, %v; want %d", losses, got, err, want)
		}
	}
}

func TestPopulationForDifficulty(t *testing.T) {
	cases := map[int]int{0: 1, 1: 1, 3: 3, 10_000: 10_000, 10_001: 10_000, 50_000: 10_000}
	for actual, want := range cases {
		if got := PopulationForDifficulty(actual); got != want {
			t.Fatalf("PopulationForDifficulty(%d) = %d, want %d", actual, got, want)
		}
	}
}

func TestMilitiaAttackReferenceValues(t *testing.T) {
	cases := map[int]int64{0: 0, 1: 40, 3: 32, 4: 28, 5: 24, 10: 4, 11: 0, 30: 0}
	for population, want := range cases {
		if got := MilitiaAttack(population); got != want {
			t.Fatalf("MilitiaAttack(%d) = %d, want %d", population, got, want)
		}
	}
}

func TestBaseEnemyHPReferenceValues(t *testing.T) {
	cases := []struct {
		d, p   int
		losses int64
		want   int64
	}{
		{1, 1, 0, 86},     // 40+8+38
		{1, 3, 0, 86},     // isqrt(3)=1
		{1, 4, 0, 124},    // isqrt(4)=2 -> 40+8+76
		{1, 10, 0, 162},   // isqrt(10)=3 -> 40+8+114
		{4, 10, 0, 186},   // 40+32+114
		{10, 100, 0, 500}, // 40+80+380, isqrt(100)=10
	}
	for _, tc := range cases {
		got, err := BaseEnemyHP(tc.d, tc.p, tc.losses)
		if err != nil {
			t.Fatalf("BaseEnemyHP(%d,%d,%d): %v", tc.d, tc.p, tc.losses, err)
		}
		if got != tc.want {
			t.Fatalf("BaseEnemyHP(%d,%d,%d) = %d, want %d", tc.d, tc.p, tc.losses, got, tc.want)
		}
	}
}

func TestBaseEnemyHPRecoveryAndMinimum(t *testing.T) {
	full, _ := BaseEnemyHP(1, 1, 0)
	for _, losses := range []int64{1, 2, 3, 4} {
		got, err := BaseEnemyHP(1, 1, losses)
		if err != nil {
			t.Fatal(err)
		}
		recovery, _ := RecoveryPercent(losses)
		want := int64(86 * int64(recovery) / 100)
		if want < 1 {
			want = 1
		}
		if got != want {
			t.Fatalf("losses=%d hp=%d want %d", losses, got, want)
		}
	}
	if full < 1 {
		t.Fatal("hp must be at least 1")
	}
	if _, err := BaseEnemyHP(0, 1, 0); err == nil {
		t.Fatal("difficulty 0 must fail")
	}
	if _, err := BaseEnemyHP(1, 0, 0); err == nil {
		t.Fatal("population 0 must fail")
	}
}

func TestLaneAndSubpoolConservation(t *testing.T) {
	for _, base := range []int64{1, 2, 3, 10, 99, 100, 1000, 1_000_000} {
		west, center, east, err := LaneHPs(base)
		if err != nil {
			t.Fatal(err)
		}
		if west+center+east != base {
			t.Fatalf("lane sum %d != %d", west+center+east, base)
		}
		for _, laneHP := range []int64{west, center, east} {
			runner, grunt, brute, err := SubpoolHPs(laneHP)
			if err != nil {
				t.Fatal(err)
			}
			if runner < 0 || grunt < 0 || brute < 0 {
				t.Fatal("negative subpool")
			}
			if runner+grunt+brute != laneHP {
				t.Fatalf("subpool sum %d != lane %d", runner+grunt+brute, laneHP)
			}
		}
	}
}

func TestBuildEnemyPools(t *testing.T) {
	pools, err := BuildEnemyPools(1000)
	if err != nil {
		t.Fatal(err)
	}
	if len(pools) != 9 {
		t.Fatalf("pools = %d", len(pools))
	}
	total := int64(0)
	for i, pool := range pools {
		if wantLane := Lane(i / 3); pool.Lane != wantLane {
			t.Fatalf("pool %d lane %s", i, pool.Lane)
		}
		if wantArch := Archetype(i % 3); pool.Archetype != wantArch {
			t.Fatalf("pool %d archetype %s", i, pool.Archetype)
		}
		total += pool.HP
	}
	if total != 1000 {
		t.Fatalf("total = %d", total)
	}
}

func TestBossMaxHP(t *testing.T) {
	cases := map[int64]int64{0: 0, 1: 1, 4: 1, 5: 1, 6: 2, 100: 20}
	for base, want := range cases {
		got, err := BossMaxHP(base)
		if err != nil {
			t.Fatal(err)
		}
		if got != want {
			t.Fatalf("BossMaxHP(%d) = %d, want %d", base, got, want)
		}
	}
}

func TestEnemyStatusProjection(t *testing.T) {
	wave := WaveState{Number: 5, Phase: PhaseFinale, Pools: []EnemyPool{{Lane: LaneWest, Archetype: ArchetypeRunner, HP: 0}}}
	if got := wave.EnemyStatus(true); got != EnemyStatusBossIncoming {
		t.Fatalf("boss wave with empty pools = %s", got)
	}
	wave.Boss = BossPool{Exists: true, HP: 0, MaxHP: 10}
	if got := wave.EnemyStatus(true); got != EnemyStatusClear {
		t.Fatalf("dead boss with empty pools = %s", got)
	}
	ordinary := WaveState{Number: 3, Phase: PhaseAssault, Pools: []EnemyPool{{Lane: LaneWest, Archetype: ArchetypeRunner, HP: 0}}}
	if got := ordinary.EnemyStatus(true); got != EnemyStatusClear {
		t.Fatalf("ordinary wave with empty pools = %s", got)
	}
}
