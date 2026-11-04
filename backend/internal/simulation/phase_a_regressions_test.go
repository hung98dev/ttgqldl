package simulation

import (
	"errors"
	"reflect"
	"testing"

	"github.com/hung98dev/ttgqldl/backend/internal/domain"
)

// realWave starts a genuine wave 1 from WAITING with one accepted builder.
func realWave(t *testing.T) (domain.SessionState, RoleQueue) {
	t.Helper()
	state := newState(t)
	next, queue, _ := acceptRoles(t, state, RoleQueue{}, roleCommand(1, actorKey('a'), domain.RoleBuilder, 1, 0))
	started, remaining, _, _, err := AdvanceWithRoles(next, queue, Input{
		Kind:         KindStartWave,
		Registration: &Registration{ActorKey: actorKey('a'), Role: domain.RoleBuilder},
	})
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	return started, remaining
}

func TestPrepEconomyFromRealWave(t *testing.T) {
	state, queue := realWave(t)
	// A real progression: no pre-locked fixture and no manual tick edits.
	state, queue = advanceWithRoles(t, state, queue, 199)
	if state.Village.Resources.Wood != 400 || state.Village.Resources.Stone != 250 || state.Village.ConstructionWork != 0 {
		t.Fatalf("economy ran before tick 200: %+v", state.Village)
	}
	if state.Difficulty.Locked {
		t.Fatal("population locked before 1600")
	}

	state, queue = advanceWithRoles(t, state, queue, 200)
	if state.Village.Resources.Wood != 402 {
		t.Fatalf("wood at 200 = %d, want 402", state.Village.Resources.Wood)
	}
	if state.Village.Resources.Stone != 251 {
		t.Fatalf("stone at 200 = %d, want 251", state.Village.Resources.Stone)
	}
	if state.Village.ConstructionWork != 8 {
		t.Fatalf("work at 200 = %d, want 8", state.Village.ConstructionWork)
	}

	state, queue = advanceWithRoles(t, state, queue, 1400)
	if state.Village.Resources.Wood != 414 {
		t.Fatalf("wood at 1400 = %d, want 414 (seven contributions)", state.Village.Resources.Wood)
	}
	if state.Village.Resources.Stone != 257 {
		t.Fatalf("stone at 1400 = %d, want 257", state.Village.Resources.Stone)
	}
	if state.Village.ConstructionWork != 56 {
		t.Fatalf("work at 1400 = %d, want 56", state.Village.ConstructionWork)
	}

	state, queue = advanceWithRoles(t, state, queue, 1600)
	if !state.Difficulty.Locked || state.Difficulty.ActualPopulation != 1 {
		t.Fatalf("population lock at 1600 = %+v", state.Difficulty)
	}
	if state.Village.Resources.Wood != 416 || state.Village.Resources.Stone != 258 {
		t.Fatalf("eighth contribution at 1600 missing: %+v", state.Village.Resources)
	}
	if len(state.Wave.Pools) != 9 {
		t.Fatalf("enemy pools not created at 1600: %d", len(state.Wave.Pools))
	}
}

func TestScheduledRoleAppliesBeforeBoundaryResolution(t *testing.T) {
	// Case A: a builder effective at 200 contributes at 200, not at 199.
	state, _ := realWave(t)
	state.Members = append(state.Members, domain.WaveMember{ActorKey: actorKey('f'), Role: domain.RoleFighter, JoinedTick: 0})
	domain.SortWaveMembers(state.Members)
	accepted, queue, _ := acceptRoles(t, state, RoleQueue{}, roleCommand(2, actorKey('b'), domain.RoleBuilder, 1, 200))
	at199, _ := advanceWithRoles(t, accepted, queue.Clone(), 199)
	if at199.Village.Resources.Wood != 400 {
		t.Fatalf("wood at 199 = %d, want 400", at199.Village.Resources.Wood)
	}
	at200, _ := advanceWithRoles(t, accepted, queue, 200)
	// Two builders (a and b) effective at 200: isqrt(2)=1 -> +2 wood, +1 stone.
	if at200.Village.Resources.Wood != 402 {
		t.Fatalf("wood at 200 = %d, want 402", at200.Village.Resources.Wood)
	}
}

func TestFighterSwitchAffectsTick1800Attack(t *testing.T) {
	// Case B: builder active in PREP, switch to fighter effective at 1800.
	state, _ := realWave(t)
	accepted, queue, _ := acceptRoles(t, state, RoleQueue{}, roleCommand(2, actorKey('a'), domain.RoleFighter, 1, 1800))
	at1800, _ := advanceWithRoles(t, accepted, queue, 1800)
	if at1800.Members[0].Role != domain.RoleFighter {
		t.Fatalf("role at 1800 = %s", at1800.Members[0].Role)
	}
	// Reconstruct the boundary to inspect its attack amount: fighter 18 + militia 40.
	probe := at1800
	probe.Wave.Tick = 1800
	probe.Wave.Phase = domain.PhaseAssault
	var events []Event
	if _, err := resolveCombat(probe, &events); err != nil {
		t.Fatal(err)
	}
	attack := int64(0)
	for _, event := range events {
		if event.Type == EventAttackResolved {
			attack = event.Amount
		}
	}
	if attack != 58 {
		t.Fatalf("tick-1800 attack = %d, want 58 (fighter, not builder)", attack)
	}
}

func TestFinalSwitchAffectsTick2800(t *testing.T) {
	// Case C: accepted at 2799, effective at 2800, used by final combat.
	state, _ := realWave(t)
	state, _ = advanceWithRoles(t, state, RoleQueue{}, 2799)
	accepted, queue, decisions := acceptRoles(t, state, RoleQueue{}, roleCommand(2, actorKey('a'), domain.RoleFighter, 1, 2800))
	if decisions[0].Outcome != OutcomeScheduled {
		t.Fatalf("final-boundary outcome = %s", decisions[0].Outcome)
	}
	at2800, remaining := advanceWithRoles(t, accepted, queue, 2800)
	if remaining.Len() != 0 || at2800.Members[0].Role != domain.RoleFighter {
		t.Fatalf("final boundary not applied: queue=%d role=%s", remaining.Len(), at2800.Members[0].Role)
	}
	probe := at2800
	probe.Wave.Tick = 2800
	probe.Wave.Phase = domain.PhaseResults
	var events []Event
	if _, err := resolveCombat(probe, &events); err != nil {
		t.Fatal(err)
	}
	if len(events) == 0 || events[0].Type != EventAttackResolved || events[0].Amount != 58 {
		t.Fatalf("final combat events = %+v", events)
	}
}

func TestFailedBoundaryLeavesStateAndQueueUntouched(t *testing.T) {
	state, _ := realWave(t)
	state, _ = advanceWithRoles(t, state, RoleQueue{}, 10)
	badQueue := RoleQueue{Entries: []RoleCommand{{
		InputSeq: 7, ActorKey: actorKey('z'), Role: domain.RoleBuilder,
		TargetWave: 2, ScheduledTick: 20, AcceptedTick: 10,
	}}}
	snapshot := state.Clone()
	_, _, _, _, err := AdvanceWithRoles(state, badQueue, Input{Kind: KindAdvanceTick})
	if !errors.Is(err, ErrStaleSchedule) {
		t.Fatalf("invalid queued command error = %v", err)
	}
	if !reflect.DeepEqual(state, snapshot) {
		t.Fatal("failed boundary mutated the original state")
	}
}

func TestTick2600BossAfterEconomyBeforeCombat(t *testing.T) {
	state, _ := realWave(t)
	state.Wave.Number = 5
	state.Difficulty = domain.Difficulty{Level: 1}
	state, _ = advanceWithRoles(t, state, RoleQueue{}, 1600)
	before := state.Village.Resources.Wood
	at2599, _ := advanceWithRoles(t, state, RoleQueue{}, 2599)
	at2600, events := advanceOne(t, at2599)
	if at2600.Village.Resources.Wood < before {
		t.Fatal("economy did not run at 2600")
	}
	if !at2600.Wave.Boss.Exists {
		t.Fatalf("boss missing at 2600: %+v", at2600.Wave.Boss)
	}
	spawnIndex, combatIndex := -1, -1
	for i, event := range events {
		switch {
		case event.Type == EventBossSpawned:
			spawnIndex = i
		case event.Type == EventScheduleBoundary && event.Boundary == BoundaryCombat:
			combatIndex = i
		}
	}
	if spawnIndex < 0 || combatIndex < 0 || spawnIndex > combatIndex {
		t.Fatalf("boss spawn order = %+v", events)
	}
}
