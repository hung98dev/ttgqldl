package simulation

import (
	"errors"
	"testing"

	"github.com/hung98dev/ttgqldl/backend/internal/domain"
)

func pausedState(t *testing.T, reasons ...domain.PauseReason) domain.SessionState {
	t.Helper()
	state, _ := mustStart(t, newState(t))
	for _, reason := range reasons {
		next, _, err := Next(state, Input{Kind: KindPause, Reason: reason})
		if err != nil {
			t.Fatalf("pause %s: %v", reason, err)
		}
		state = next
	}
	return state
}

func TestPauseFreezesTheTick(t *testing.T) {
	for _, reason := range []domain.PauseReason{domain.PauseOperatorPaused, domain.PauseSourceDisconnected, domain.PauseStorageUnavailable} {
		state := pausedState(t, reason)
		tick := state.Tick()
		if _, _, err := Next(state, Input{Kind: KindAdvanceTick}); !errors.Is(err, ErrTickFrozen) {
			t.Fatalf("%s: advance while paused: %v", reason, err)
		}
		if state.Tick() != tick {
			t.Fatal("paused tick moved")
		}
	}
}

func TestPauseReasonMapsToLifecycle(t *testing.T) {
	cases := map[domain.PauseReason]domain.Lifecycle{
		domain.PauseOperatorPaused:     domain.LifecyclePaused,
		domain.PauseSourceDisconnected: domain.LifecycleSourceDisconnected,
		domain.PauseStorageUnavailable: domain.LifecycleStorageUnavailable,
	}
	for reason, want := range cases {
		state, _, err := Next(pausedStateWithout(t), Input{Kind: KindPause, Reason: reason})
		if err != nil {
			t.Fatalf("%s: %v", reason, err)
		}
		if state.Lifecycle != want {
			t.Fatalf("%s: lifecycle = %s, want %s", reason, state.Lifecycle, want)
		}
	}
}

// pausedStateWithout returns a started state with no pause applied yet.
func pausedStateWithout(t *testing.T) domain.SessionState {
	t.Helper()
	state, _ := mustStart(t, newState(t))
	return state
}

func TestIndependentPauseReasonsRequireAllToClear(t *testing.T) {
	state := pausedState(t, domain.PauseOperatorPaused)
	state, events, err := Next(state, Input{Kind: KindPause, Reason: domain.PauseSourceDisconnected})
	if err != nil {
		t.Fatal(err)
	}
	if len(state.PauseReasons) != 2 {
		t.Fatalf("pause reasons = %v", state.PauseReasons)
	}
	if len(events) != 1 || events[0].Type != EventPaused {
		t.Fatalf("events = %v", eventTypes(events))
	}

	// Clearing the operator pause leaves the source reason: still frozen.
	resumed, events, err := Next(state, Input{Kind: KindResume, Reason: domain.PauseOperatorPaused})
	if err != nil {
		t.Fatal(err)
	}
	if len(resumed.PauseReasons) != 1 || resumed.PauseReasons[0] != domain.PauseSourceDisconnected {
		t.Fatalf("reasons after clearing one = %v", resumed.PauseReasons)
	}
	if resumed.Lifecycle != domain.LifecycleSourceDisconnected {
		t.Fatalf("lifecycle = %s", resumed.Lifecycle)
	}
	if len(events) != 0 {
		t.Fatalf("clearing one reason must not emit a resume event: %v", eventTypes(events))
	}
	if _, _, err := Next(resumed, Input{Kind: KindAdvanceTick}); !errors.Is(err, ErrTickFrozen) {
		t.Fatal("simulation resumed while a reason was active")
	}

	// Clearing the last reason resumes without any catch-up.
	tick := resumed.Tick()
	final, events, err := Next(resumed, Input{Kind: KindResume, Reason: domain.PauseSourceDisconnected})
	if err != nil {
		t.Fatal(err)
	}
	if final.Lifecycle != domain.LifecycleRunning || len(final.PauseReasons) != 0 {
		t.Fatalf("state after full resume: %s %v", final.Lifecycle, final.PauseReasons)
	}
	if final.Tick() != tick {
		t.Fatalf("resume changed the tick from %d to %d", tick, final.Tick())
	}
	if len(events) != 1 || events[0].Type != EventResumed {
		t.Fatalf("events = %v", eventTypes(events))
	}

	advanced, _, err := Next(final, Input{Kind: KindAdvanceTick})
	if err != nil {
		t.Fatal(err)
	}
	if advanced.Tick() != tick+1 {
		t.Fatalf("first tick after resume = %d, want %d", advanced.Tick(), tick+1)
	}
}

func TestPauseIsIdempotent(t *testing.T) {
	state := pausedState(t, domain.PauseOperatorPaused)
	next, events, err := Next(state, Input{Kind: KindPause, Reason: domain.PauseOperatorPaused})
	if err != nil {
		t.Fatal(err)
	}
	if len(next.PauseReasons) != 1 {
		t.Fatalf("duplicate pause reason stored: %v", next.PauseReasons)
	}
	if len(events) != 0 {
		t.Fatalf("idempotent pause emitted events: %v", eventTypes(events))
	}
}

func TestResumeWithoutActiveReasonIsNoOp(t *testing.T) {
	state, _ := mustStart(t, newState(t))
	next, events, err := Next(state, Input{Kind: KindResume, Reason: domain.PauseOperatorPaused})
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 0 || next.Lifecycle != domain.LifecycleRunning {
		t.Fatalf("no-op resume changed state: %s %v", next.Lifecycle, eventTypes(events))
	}
}

func TestInvalidPauseReasonIsRejected(t *testing.T) {
	state, _ := mustStart(t, newState(t))
	if _, _, err := Next(state, Input{Kind: KindPause, Reason: domain.PauseReason(9)}); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("invalid pause reason: %v", err)
	}
}

func TestStoppedRejectsGameplayAndPause(t *testing.T) {
	state, _ := mustStart(t, newState(t))
	state, events, err := Next(state, Input{Kind: KindStop})
	if err != nil {
		t.Fatal(err)
	}
	if state.Lifecycle != domain.LifecycleStopped || len(events) != 1 || events[0].Type != EventStopped {
		t.Fatalf("stop state/events: %s %v", state.Lifecycle, eventTypes(events))
	}
	if _, _, err := Next(state, Input{Kind: KindAdvanceTick}); !errors.Is(err, ErrStopped) {
		t.Fatal("stopped session advanced")
	}
	if _, _, err := Next(state, Input{Kind: KindPause, Reason: domain.PauseOperatorPaused}); !errors.Is(err, ErrStopped) {
		t.Fatal("stopped session accepted a pause")
	}
	if _, _, err := Next(state, Input{Kind: KindRollover}); !errors.Is(err, ErrStopped) {
		t.Fatal("stopped session accepted a rollover")
	}
}

func TestStopClearsPauseReasonsAndIsIdempotent(t *testing.T) {
	state := pausedState(t, domain.PauseOperatorPaused)
	state, _, err := Next(state, Input{Kind: KindStop})
	if err != nil {
		t.Fatal(err)
	}
	if len(state.PauseReasons) != 0 {
		t.Fatalf("stopped state retained pause reasons: %v", state.PauseReasons)
	}
	next, events, err := Next(state, Input{Kind: KindStop})
	if err != nil || len(events) != 0 {
		t.Fatalf("second stop: %v %v", err, eventTypes(events))
	}
	if next.Lifecycle != domain.LifecycleStopped {
		t.Fatal("second stop changed the lifecycle")
	}
}

func TestPauseBeforeStartKeepsWaitingTick(t *testing.T) {
	state := newState(t)
	state, _, err := Next(state, Input{Kind: KindPause, Reason: domain.PauseOperatorPaused})
	if err != nil {
		t.Fatal(err)
	}
	if state.Tick() != domain.WaitingTick || len(state.Members) != 0 {
		t.Fatal("pausing a waiting session must not start it")
	}
	// A paused waiting session cannot be started until the pause clears.
	if _, _, err := Next(state, Input{Kind: KindStartWave, Registration: &Registration{ActorKey: actorKey('a'), Role: domain.RoleBuilder}}); !errors.Is(err, ErrTickFrozen) {
		t.Fatalf("start while paused: %v", err)
	}
}
