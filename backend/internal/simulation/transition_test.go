package simulation

import (
	"errors"
	"reflect"
	"testing"

	"github.com/hung98dev/ttgqldl/backend/internal/domain"
)

const (
	villageID = domain.VillageID("11111111-1111-4111-8111-111111111111")
	sessionID = domain.SessionID("22222222-2222-4222-8222-222222222222")
)

// actorKey returns a valid scoped actor key: 64 copies of one lowercase hex
// digit. The input byte selects the digit, so arbitrary test bytes stay valid.
func actorKey(ch byte) domain.ActorKey {
	digit := "0123456789abcdef"[int(ch)%16]
	raw := make([]byte, 64)
	for i := range raw {
		raw[i] = digit
	}
	return domain.ActorKey(raw)
}

// findMember returns the index of an actor in the canonical member slice.
func findMember(t *testing.T, state domain.SessionState, key domain.ActorKey) int {
	t.Helper()
	for i, member := range state.Members {
		if member.ActorKey == key {
			return i
		}
	}
	t.Fatalf("member %s not found", key)
	return -1
}

func newState(t *testing.T) domain.SessionState {
	t.Helper()
	state, err := domain.NewInitialState(villageID, sessionID, domain.SourceModeMock)
	if err != nil {
		t.Fatalf("NewInitialState: %v", err)
	}
	return state
}

func mustStart(t *testing.T, state domain.SessionState) (domain.SessionState, []Event) {
	t.Helper()
	next, events, err := Next(state, Input{
		Kind:         KindStartWave,
		Registration: &Registration{ActorKey: actorKey('a'), Role: domain.RoleBuilder},
	})
	if err != nil {
		t.Fatalf("start wave: %v", err)
	}
	return next, events
}

func advanceTo(t *testing.T, state domain.SessionState, target int) domain.SessionState {
	t.Helper()
	for state.Tick() < target {
		next, _, err := Next(state, Input{Kind: KindAdvanceTick})
		if err != nil {
			t.Fatalf("advance to %d at tick %d: %v", target, state.Tick(), err)
		}
		state = next
	}
	return state
}

func eventTypes(events []Event) []EventType {
	types := make([]EventType, len(events))
	for i, event := range events {
		types[i] = event.Type
	}
	return types
}

func hasBoundary(events []Event, boundary Boundary) bool {
	for _, event := range events {
		if event.Type == EventScheduleBoundary && event.Boundary == boundary {
			return true
		}
	}
	return false
}

func TestWaitingStateIsFrozen(t *testing.T) {
	state := newState(t)
	next, events, err := Next(state, Input{Kind: KindAdvanceTick})
	if !errors.Is(err, ErrNotRunning) {
		t.Fatalf("advance in waiting: error = %v, want ErrNotRunning", err)
	}
	if len(events) != 0 || !reflect.DeepEqual(next, state) {
		t.Fatal("a rejected transition must not change state or emit events")
	}
}

func TestOperatorStartAloneCannotStartAWave(t *testing.T) {
	state := newState(t)
	next, _, err := Next(state, Input{Kind: KindStartWave})
	if !errors.Is(err, ErrNoRegistration) {
		t.Fatalf("start without registration: error = %v, want ErrNoRegistration", err)
	}
	if !reflect.DeepEqual(next, state) {
		t.Fatal("state changed on a rejected start")
	}
}

func TestFirstRegistrationStartsTickZero(t *testing.T) {
	state := newState(t)
	next, events := mustStart(t, state)

	if next.Tick() != 0 || next.Wave.Phase != domain.PhasePrep {
		t.Fatalf("start: tick %d phase %s", next.Tick(), next.Wave.Phase)
	}
	if next.Lifecycle != domain.LifecycleRunning {
		t.Fatalf("lifecycle = %s", next.Lifecycle)
	}
	if len(next.Members) != 1 || next.Members[0].JoinedTick != 0 {
		t.Fatalf("members = %+v", next.Members)
	}
	if next.Members[0].LastChangeTick != nil || next.Members[0].FirstSwitchUsed {
		t.Fatal("a fresh member must not have a recorded role change")
	}
	if !reflect.DeepEqual(eventTypes(events), []EventType{EventWaveStarted}) {
		t.Fatalf("events = %v", eventTypes(events))
	}
	// Tick 0 performs no gameplay work and starts nothing else.
	if next.Wave.Result != domain.WaveResultPending || next.Wave.Boss.Exists {
		t.Fatal("tick 0 must not settle or spawn")
	}
}

func TestStartRejectsInvalidRegistrationAndSecondStart(t *testing.T) {
	state := newState(t)
	if _, _, err := Next(state, Input{Kind: KindStartWave, Registration: &Registration{ActorKey: "short", Role: domain.RoleBuilder}}); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("invalid actor key: %v", err)
	}
	if _, _, err := Next(state, Input{Kind: KindStartWave, Registration: &Registration{ActorKey: actorKey('a'), Role: domain.Role(9)}}); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("invalid role: %v", err)
	}

	started, _ := mustStart(t, state)
	if _, _, err := Next(started, Input{Kind: KindStartWave, Registration: &Registration{ActorKey: actorKey('b'), Role: domain.RoleGuard}}); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("second start: %v", err)
	}
}

func TestAdvanceIsExactlyOneTickAndMonotonic(t *testing.T) {
	state, _ := mustStart(t, newState(t))
	for expected := 1; expected <= 20; expected++ {
		next, _, err := Next(state, Input{Kind: KindAdvanceTick})
		if err != nil {
			t.Fatalf("advance %d: %v", expected, err)
		}
		if next.Tick() != state.Tick()+1 {
			t.Fatalf("tick jumped from %d to %d", state.Tick(), next.Tick())
		}
		state = next
	}
}

func TestStaleTickRequestIsRejected(t *testing.T) {
	state, _ := mustStart(t, newState(t))
	state = advanceTo(t, state, 10)
	stale := 5
	if _, _, err := Next(state, Input{Kind: KindAdvanceTick, FromTick: &stale}); !errors.Is(err, ErrOutOfOrder) {
		t.Fatalf("stale guard: error = %v, want ErrOutOfOrder", err)
	}
	current := state.Tick()
	next, _, err := Next(state, Input{Kind: KindAdvanceTick, FromTick: &current})
	if err != nil || next.Tick() != current+1 {
		t.Fatalf("current guard must be accepted: %v", err)
	}
}

func TestPhaseBoundariesEmitSinglePhaseChange(t *testing.T) {
	cases := []struct {
		tick  int
		phase domain.Phase
	}{
		{1599, domain.PhasePrep},
		{1600, domain.PhaseAssault},
		{2599, domain.PhaseAssault},
		{2600, domain.PhaseFinale},
		{2799, domain.PhaseFinale},
		{2800, domain.PhaseResults},
		{2999, domain.PhaseResults},
	}
	for _, tc := range cases {
		state, _ := mustStart(t, newState(t))
		state = advanceTo(t, state, tc.tick-1)
		next, events, err := Next(state, Input{Kind: KindAdvanceTick})
		if err != nil {
			t.Fatalf("tick %d: %v", tc.tick, err)
		}
		if next.Wave.Phase != tc.phase {
			t.Fatalf("tick %d phase = %s, want %s", tc.tick, next.Wave.Phase, tc.phase)
		}
		phaseChanges := 0
		for _, event := range events {
			if event.Type == EventPhaseChanged {
				phaseChanges++
			}
		}
		// A phase change is emitted only when the phase actually changes.
		wantChanges := 0
		if tc.tick == 1600 || tc.tick == 2600 || tc.tick == 2800 {
			wantChanges = 1
		}
		if phaseChanges != wantChanges {
			t.Fatalf("tick %d phase_changed events = %d, want %d", tc.tick, phaseChanges, wantChanges)
		}
	}
}

func TestBoundarySignalsAtCriticalTicks(t *testing.T) {
	state, _ := mustStart(t, newState(t))
	state = advanceTo(t, state, 1199)
	next, events, err := Next(state, Input{Kind: KindAdvanceTick})
	if err != nil {
		t.Fatal(err)
	}
	if next.Tick() != 1200 || !hasBoundary(events, BoundaryVoteClose) {
		t.Fatalf("tick 1200 must signal vote close: %v", eventTypes(events))
	}

	state = advanceTo(t, next, 1599)
	next, events, err = Next(state, Input{Kind: KindAdvanceTick})
	if err != nil {
		t.Fatal(err)
	}
	if !hasBoundary(events, BoundaryPopulationLock) || !hasBoundary(events, BoundaryRoleResolution) {
		t.Fatalf("tick 1600 boundaries = %v", eventTypes(events))
	}

	state = advanceTo(t, next, 1799)
	next, events, err = Next(state, Input{Kind: KindAdvanceTick})
	if err != nil {
		t.Fatal(err)
	}
	if !hasBoundary(events, BoundaryCombat) {
		t.Fatalf("tick 1800 must signal combat: %v", eventTypes(events))
	}

	state = advanceTo(t, next, 2800)
	if state.Tick() != 2800 {
		t.Fatalf("tick = %d", state.Tick())
	}
	// The final boundary must require settlement and must not settle by itself.
	foundSettlement := false
	_, events, err = Next(state, Input{Kind: KindAdvanceTick})
	if err != nil {
		t.Fatal(err)
	}
	for _, event := range events {
		if event.Type == EventSettlementRequired {
			foundSettlement = true
		}
	}
	if foundSettlement {
		t.Fatal("settlement_required belongs to the tick that reaches 2800, not 2801")
	}
	if state.Wave.Result != domain.WaveResultPending {
		t.Fatalf("W0202 must not settle: result = %s", state.Wave.Result)
	}
}

func TestSettlementIsRequiredWhenEntering2800(t *testing.T) {
	state, _ := mustStart(t, newState(t))
	state = advanceTo(t, state, 2799)
	next, events, err := Next(state, Input{Kind: KindAdvanceTick})
	if err != nil {
		t.Fatal(err)
	}
	if next.Tick() != 2800 || next.Wave.Phase != domain.PhaseResults {
		t.Fatalf("tick %d phase %s", next.Tick(), next.Wave.Phase)
	}
	// The complete ordered sequence is contractual: final contribution, final
	// combat, settlement, and only then RESULTS publication.
	want := []struct {
		Type     EventType
		Boundary Boundary
		Stage    Stage
	}{
		{EventScheduleBoundary, BoundaryRoleResolution, StageFinalContribution},
		{EventScheduleBoundary, BoundaryFinalCombat, StageFinalCombat},
		{EventAttackResolved, "", ""},
		{EventSettlementRequired, "", StageSettlement},
		{EventPhaseChanged, "", StageEnterResults},
	}
	if len(events) != len(want) {
		t.Fatalf("tick 2800 events = %v, want %d", eventTypes(events), len(want))
	}
	for i, expected := range want {
		if events[i].Type != expected.Type || events[i].Boundary != expected.Boundary || events[i].Stage != expected.Stage {
			t.Fatalf("event %d = %+v, want type=%s boundary=%s stage=%s", i, events[i], expected.Type, expected.Boundary, expected.Stage)
		}
	}
	if next.Wave.Result != domain.WaveResultPending {
		t.Fatalf("no result may be produced in W0202: %s", next.Wave.Result)
	}
}

func TestTick3000IsControlOnly(t *testing.T) {
	state, _ := mustStart(t, newState(t))
	state = advanceTo(t, state, 2999)
	next, events, err := Next(state, Input{Kind: KindAdvanceTick})
	if err != nil {
		t.Fatal(err)
	}
	if next.Tick() != 3000 {
		t.Fatalf("tick = %d", next.Tick())
	}
	if !hasBoundary(events, BoundaryRollover) {
		t.Fatalf("tick 3000 must signal rollover: %v", eventTypes(events))
	}
	for _, event := range events {
		switch event.Type {
		case EventSettlementRequired, EventPhaseChanged:
			t.Fatalf("tick 3000 must not resolve gameplay: %v", eventTypes(events))
		}
	}

	// No gameplay work is possible on the control tick.
	if _, _, err := Next(next, Input{Kind: KindAdvanceTick}); !errors.Is(err, ErrControlBoundary) {
		t.Fatalf("advance at 3000: %v", err)
	}
}

func TestPureProgressionChangesNoGameplayValues(t *testing.T) {
	state, _ := mustStart(t, newState(t))
	before := state
	state = advanceTo(t, state, 2999)

	// W0205 now resolves economy and combat at 200-tick boundaries, so village
	// values legitimately change; only participant identity and XP must not.
	if len(state.XP) != len(before.XP) {
		t.Fatal("pure tick progression must not award XP")
	}
	if len(state.Members) != len(before.Members) {
		t.Fatal("pure tick progression must not create or remove players")
	}
	if state.Wave.Result != domain.WaveResultPending || state.Wave.Defeated {
		t.Fatal("pure tick progression must not settle the wave")
	}
}

func TestInputStateIsNeverMutated(t *testing.T) {
	state, _ := mustStart(t, newState(t))
	state = advanceTo(t, state, 50)
	snapshot := state.Clone()

	next, _, err := Next(state, Input{Kind: KindAdvanceTick})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(state, snapshot) {
		t.Fatal("input state was mutated by the transition")
	}
	// Mutating the returned state must not affect the caller's state.
	next.Members[0].Role = domain.RoleFighter
	if state.Members[0].Role != domain.RoleBuilder {
		t.Fatal("returned state shares member storage with the input")
	}
}

func TestDeterministicRepeatedExecution(t *testing.T) {
	state, _ := mustStart(t, newState(t))
	state = advanceTo(t, state, 1599)

	firstState, firstEvents, err := Next(state, Input{Kind: KindAdvanceTick})
	if err != nil {
		t.Fatal(err)
	}
	secondState, secondEvents, err := Next(state, Input{Kind: KindAdvanceTick})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(firstState, secondState) || !reflect.DeepEqual(firstEvents, secondEvents) {
		t.Fatal("identical inputs produced different transitions")
	}
}

func TestInvalidInputKindIsRejected(t *testing.T) {
	state, _ := mustStart(t, newState(t))
	if _, _, err := Next(state, Input{Kind: Kind(99)}); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("unknown kind: %v", err)
	}
}

func TestInvalidInputStateIsRejectedBeforeMutation(t *testing.T) {
	state := newState(t)
	state.Village.Level = 0
	next, _, err := Next(state, Input{Kind: KindStartWave, Registration: &Registration{ActorKey: actorKey('a'), Role: domain.RoleBuilder}})
	if !errors.Is(err, ErrInvalidState) {
		t.Fatalf("invalid input state: %v", err)
	}
	if next.Village.Level != 0 {
		t.Fatal("rejected transition modified the state")
	}
}
