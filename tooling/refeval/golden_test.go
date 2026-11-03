package main

import (
	"bytes"
	"os"
	"testing"
)

// TestReferenceAssertions re-runs every hand-derived expectation. This is the
// gate M02 must keep green while the authoritative reducer is implemented.
func TestReferenceAssertions(t *testing.T) {
	for _, e := range assertions() {
		t.Error(e)
	}
}

// TestGoldenVectorsAreCurrent fails when the checked-in golden vectors drift
// from the reference evaluator output, so a rules_version change cannot land
// without regenerating and reviewing the executable vectors.
func TestGoldenVectorsAreCurrent(t *testing.T) {
	const path = "testdata/golden_vectors.json"
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v (run: go run ./refeval -write)", path, err)
	}
	got := marshalGolden(buildGolden())
	if !bytes.Equal(want, got) {
		t.Fatalf("%s is stale: run 'go run ./refeval -write' from tooling/ and review the diff", path)
	}
}

// TestInitialStateIsDocumented pins the documented initial village/session
// state so an implementation cannot silently change the baseline.
func TestInitialStateIsDocumented(t *testing.T) {
	s := NewSim()
	if s.WaveNo != 1 || s.Tick != -1 || s.Lifecycle != "waiting" {
		t.Fatalf("session init = wave %d tick %d lifecycle %s", s.WaveNo, s.Tick, s.Lifecycle)
	}
	if s.VillageLevel != 1 || s.WallTier != 1 || s.wallMax() != 1000 || s.TownhallHP != 500 {
		t.Fatalf("village init level=%d tier=%d wallMax=%d townhall=%d", s.VillageLevel, s.WallTier, s.wallMax(), s.TownhallHP)
	}
	if s.Wood != 400 || s.Stone != 250 || s.Food != 300 || s.ConstructionWork != 0 {
		t.Fatalf("resources init wood=%d stone=%d food=%d work=%d", s.Wood, s.Stone, s.Food, s.ConstructionWork)
	}
	if s.ConsecutiveWins != 0 || s.ConsecutiveLosses != 0 || s.TotalWins != 0 {
		t.Fatalf("progress init cw=%d cl=%d tw=%d", s.ConsecutiveWins, s.ConsecutiveLosses, s.TotalWins)
	}
}

// TestPhaseBoundaries pins the documented phase derivation.
func TestPhaseBoundaries(t *testing.T) {
	cases := []struct {
		tick  int
		phase string
	}{
		{0, "prep"}, {1599, "prep"}, {1600, "assault"}, {1799, "assault"},
		{2599, "assault"}, {2600, "finale"}, {2799, "finale"},
		{2800, "results"}, {2999, "results"}, {3000, "rollover"},
	}
	s := NewSim()
	s.Lifecycle = "running"
	for _, c := range cases {
		s.Tick = c.tick
		if got := s.Phase(); got != c.phase {
			t.Errorf("tick %d phase=%s want %s", c.tick, got, c.phase)
		}
	}
}

// TestBossNeverAttackableBefore2600 verifies the scheduling contract.
func TestBossNeverAttackableBefore2600(t *testing.T) {
	in := defaultInputs()
	in.WaveNo = 5
	in.Fighters = 30
	s := buildSim(in)
	for s.Tick < BossSpawnTick-1 {
		s.Tick++
		s.ProcessTick()
		if s.BossSpawned || s.BossHP != 0 {
			t.Fatalf("boss pool exists before 2600: hp=%d spawned=%t at tick %d", s.BossHP, s.BossSpawned, s.Tick)
		}
	}
	s.Tick = BossSpawnTick
	s.ProcessTick()
	if !s.BossSpawned || s.BossMaxHP == 0 {
		t.Fatalf("boss pool must be created at 2600: max=%d spawned=%t", s.BossMaxHP, s.BossSpawned)
	}
}
