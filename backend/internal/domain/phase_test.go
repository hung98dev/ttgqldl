package domain

import "testing"

func TestPhaseBoundaries(t *testing.T) {
	cases := []struct {
		tick  int
		phase Phase
	}{
		{-1, PhaseWaiting},
		{0, PhasePrep},
		{1, PhasePrep},
		{1599, PhasePrep},
		{1600, PhaseAssault},
		{1799, PhaseAssault},
		{2599, PhaseAssault},
		{2600, PhaseFinale},
		{2799, PhaseFinale},
		{2800, PhaseResults},
		{2999, PhaseResults},
	}
	for _, tc := range cases {
		got, err := PhaseForTick(tc.tick)
		if err != nil {
			t.Fatalf("PhaseForTick(%d): %v", tc.tick, err)
		}
		if got != tc.phase {
			t.Errorf("PhaseForTick(%d) = %s, want %s", tc.tick, got, tc.phase)
		}
	}
}

func TestPhaseForTickRejectsControlAndOutOfRangeTicks(t *testing.T) {
	for _, tick := range []int{3000, 3001, -2, 100000} {
		if _, err := PhaseForTick(tick); err == nil {
			t.Errorf("PhaseForTick(%d) accepted an invalid tick", tick)
		}
	}
	if _, err := PhaseForTick(RolloverTick); err == nil {
		t.Fatal("tick 3000 must be handled as a control boundary, not a phase")
	}
}

func TestClassifyTick(t *testing.T) {
	cases := []struct {
		tick  int
		class TickClass
	}{
		{-1, TickClassWaiting},
		{0, TickClassGameplay},
		{2999, TickClassGameplay},
		{3000, TickClassRollover},
	}
	for _, tc := range cases {
		got, err := ClassifyTick(tc.tick)
		if err != nil {
			t.Fatalf("ClassifyTick(%d): %v", tc.tick, err)
		}
		if got != tc.class {
			t.Errorf("ClassifyTick(%d) = %s, want %s", tc.tick, got, tc.class)
		}
	}
	for _, tick := range []int{-2, 3001} {
		if _, err := ClassifyTick(tick); err == nil {
			t.Errorf("ClassifyTick(%d) accepted an out-of-range tick", tick)
		}
	}
}

func TestTickPredicates(t *testing.T) {
	if !IsRolloverTick(3000) || IsRolloverTick(2999) || IsRolloverTick(-1) {
		t.Fatal("IsRolloverTick must match only tick 3000")
	}
	if IsGameplayTick(-1) || !IsGameplayTick(0) || !IsGameplayTick(2999) || IsGameplayTick(3000) {
		t.Fatal("IsGameplayTick must accept 0..2999 only")
	}
}

func FuzzPhaseForTick(f *testing.F) {
	f.Add(0)
	f.Add(-1)
	f.Add(3000)
	f.Fuzz(func(t *testing.T, tick int) {
		phase, err := PhaseForTick(tick)
		if err != nil {
			return
		}
		if !phase.Valid() {
			t.Fatalf("PhaseForTick(%d) returned an invalid phase", tick)
		}
		if tick != WaitingTick && (tick < 0 || tick >= WaveTicks) {
			t.Fatalf("PhaseForTick(%d) accepted an out-of-range tick", tick)
		}
	})
}
