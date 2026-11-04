package simulation

import (
	"reflect"
	"testing"

	"github.com/hung98dev/ttgqldl/backend/internal/domain"
)

func TestBoundariesAt(t *testing.T) {
	cases := []struct {
		tick     int
		bossWave bool
		want     []Boundary
	}{
		{0, false, nil},
		{1, false, nil},
		{199, false, nil},
		{200, false, []Boundary{BoundaryRoleResolution}},
		{1199, false, nil},
		{1200, false, []Boundary{BoundaryVoteClose, BoundaryRoleResolution}},
		{1600, false, []Boundary{BoundaryPopulationLock, BoundaryRoleResolution}},
		{1799, false, nil},
		{1800, false, []Boundary{BoundaryRoleResolution, BoundaryCombat}},
		{2400, false, []Boundary{BoundaryRoleResolution, BoundaryCombat}},
		{2600, false, []Boundary{BoundaryRoleResolution, BoundaryCombat}},
		{2600, true, []Boundary{BoundaryRoleResolution, BoundaryBossSpawn, BoundaryCombat}},
		{2800, true, []Boundary{BoundaryRoleResolution, BoundaryFinalCombat}},
		{2999, false, nil},
		{3000, false, []Boundary{BoundaryRollover}},
	}
	for _, tc := range cases {
		got := BoundariesAt(tc.tick, tc.bossWave)
		if len(got) == 0 && len(tc.want) == 0 {
			continue
		}
		if !reflect.DeepEqual(got, tc.want) {
			t.Fatalf("BoundariesAt(%d, boss=%v) = %v, want %v", tc.tick, tc.bossWave, got, tc.want)
		}
	}
}

func TestStagesAtFinalCombatAreOrdered(t *testing.T) {
	want := []Stage{StageConsumeInputs, StageFinalContribution, StageFinalCombat, StageSettlement, StageEnterResults}
	got := StagesAt(domain.FinalCombatTick)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("StagesAt(2800) = %v, want %v", got, want)
	}
	if got := StagesAt(domain.RolloverTick); !reflect.DeepEqual(got, []Stage{StageControlRollover}) {
		t.Fatalf("StagesAt(3000) = %v", got)
	}
	if got := StagesAt(100); got != nil {
		t.Fatalf("StagesAt(100) = %v, want nil", got)
	}
}

func TestBossSpawnBoundaryOnlyOnBossWaves(t *testing.T) {
	if !hasBoundaryEventsForTick(t, 2600, true, BoundaryBossSpawn) {
		t.Fatal("boss spawn boundary missing on a boss wave")
	}
	if hasBoundaryEventsForTick(t, 2600, false, BoundaryBossSpawn) {
		t.Fatal("boss spawn boundary emitted on a non-boss wave")
	}
}

// hasBoundaryEventsForTick checks the static boundary table, not a transition.
func hasBoundaryEventsForTick(t *testing.T, tick int, bossWave bool, boundary Boundary) bool {
	t.Helper()
	for _, candidate := range BoundariesAt(tick, bossWave) {
		if candidate == boundary {
			return true
		}
	}
	return false
}

func TestNoGameplayEventsAreInvented(t *testing.T) {
	state, _ := mustStart(t, newState(t))
	state = advanceTo(t, state, 2799)
	_, events, err := Next(state, Input{Kind: KindAdvanceTick})
	if err != nil {
		t.Fatal(err)
	}
	// W0202 may only emit lifecycle/scheduling signals; gameplay events belong to
	// W0203-W0207 and must not be fabricated here.
	allowed := map[EventType]bool{
		EventPhaseChanged:       true,
		EventScheduleBoundary:   true,
		EventSettlementRequired: true,
	}
	for _, event := range events {
		if !allowed[event.Type] {
			t.Fatalf("unexpected event type %s", event.Type)
		}
	}
}
