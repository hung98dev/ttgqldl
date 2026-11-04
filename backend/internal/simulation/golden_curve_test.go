package simulation

import (
	"strconv"
	"testing"

	"github.com/hung98dev/ttgqldl/backend/internal/domain"
)

// qaBal05Curve is the independent minimum-fighter table for boss wave 5 with
// difficulty 1, no streaks and no vote. Composition convention: the remaining
// players split into guards (floor/2) and builders.
var qaBal05Curve = []struct {
	population  int
	minFighters int
}{
	{1, 0}, {3, 0}, {4, 0}, {5, 1}, {10, 4}, {30, 9}, {100, 25}, {1000, 196}, {10000, 1849},
}

func qaBal05Fixture(t *testing.T, population, fighters int) domain.SessionState {
	t.Helper()
	remaining := population - fighters
	guards := remaining / 2
	builders := remaining - guards
	return goldenWaveFixture(t, builders, guards, fighters)
}

func qaBal05Outcome(t *testing.T, population, fighters int) domain.WaveResult {
	t.Helper()
	state := qaBal05Fixture(t, population, fighters)
	final, _ := advanceWithRoles(t, state, RoleQueue{}, domain.RolloverTick)
	return final.Wave.Result
}

func TestQABal05MinimumFighterCurve(t *testing.T) {
	if raceDetectorEnabled {
		t.Skip("QA-BAL-05 runs the full 3,000-tick pipeline; it is executed by the non-race unit gate")
	}
	for _, tc := range qaBal05Curve {
		tc := tc
		t.Run("population_"+strconv.Itoa(tc.population), func(t *testing.T) {
			result := qaBal05Outcome(t, tc.population, tc.minFighters)
			if result != domain.WaveResultVictory {
				t.Fatalf("P=%d F=%d: result=%s, want victory", tc.population, tc.minFighters, result)
			}
			if tc.minFighters > 0 {
				fewer := qaBal05Outcome(t, tc.population, tc.minFighters-1)
				if fewer != domain.WaveResultDefeat {
					t.Fatalf("P=%d F=%d: result=%s, want defeat", tc.population, tc.minFighters-1, fewer)
				}
			}
		})
	}
}
