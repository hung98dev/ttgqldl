package simulation

import (
	"fmt"
	"sort"

	"github.com/hung98dev/ttgqldl/backend/internal/domain"
)

// lifecycleFor derives the single lifecycle value from the independent pause
// reasons. Operator pause is the most visible blocking state; a source or storage
// problem is reported only when no operator pause is active.
func lifecycleFor(reasons []domain.PauseReason) domain.Lifecycle {
	var hasOperator, hasSource, hasStorage bool
	for _, reason := range reasons {
		switch reason {
		case domain.PauseOperatorPaused:
			hasOperator = true
		case domain.PauseSourceDisconnected:
			hasSource = true
		case domain.PauseStorageUnavailable:
			hasStorage = true
		}
	}
	switch {
	case hasOperator:
		return domain.LifecyclePaused
	case hasSource:
		return domain.LifecycleSourceDisconnected
	case hasStorage:
		return domain.LifecycleStorageUnavailable
	default:
		return domain.LifecycleRunning
	}
}

func sortPauseReasons(reasons []domain.PauseReason) {
	sort.SliceStable(reasons, func(i, j int) bool { return reasons[i] < reasons[j] })
}

func hasReason(reasons []domain.PauseReason, reason domain.PauseReason) bool {
	for _, existing := range reasons {
		if existing == reason {
			return true
		}
	}
	return false
}

// pause adds one independent pause reason. The logical tick is frozen; adding an
// already active reason is an idempotent no-op.
func pause(state domain.SessionState, input Input) (domain.SessionState, []Event, error) {
	if state.Lifecycle == domain.LifecycleStopped {
		return state, nil, ErrStopped
	}
	if !input.Reason.Valid() {
		return state, nil, fmt.Errorf("%w: invalid pause reason", ErrInvalidInput)
	}
	if hasReason(state.PauseReasons, input.Reason) {
		return state.Clone(), nil, nil
	}

	next := state.Clone()
	next.PauseReasons = append(next.PauseReasons, input.Reason)
	sortPauseReasons(next.PauseReasons)
	next.Lifecycle = lifecycleFor(next.PauseReasons)

	events := []Event{{
		Type:   EventPaused,
		WaveNo: next.Wave.Number,
		Tick:   next.Wave.Tick,
		Phase:  next.Wave.Phase,
		Reason: input.Reason,
	}}
	return commit(state, next, events)
}

// resume clears one pause reason. Simulation resumes only when every reason is
// gone; no ticks are replayed or caught up.
func resume(state domain.SessionState, input Input) (domain.SessionState, []Event, error) {
	if state.Lifecycle == domain.LifecycleStopped {
		return state, nil, ErrStopped
	}
	if !input.Reason.Valid() {
		return state, nil, fmt.Errorf("%w: invalid pause reason", ErrInvalidInput)
	}
	if !hasReason(state.PauseReasons, input.Reason) {
		// Clearing an inactive reason is an idempotent no-op.
		return state.Clone(), nil, nil
	}

	next := state.Clone()
	remaining := make([]domain.PauseReason, 0, len(state.PauseReasons))
	for _, reason := range state.PauseReasons {
		if reason != input.Reason {
			remaining = append(remaining, reason)
		}
	}
	sortPauseReasons(remaining)
	next.PauseReasons = remaining

	if len(remaining) > 0 {
		next.Lifecycle = lifecycleFor(remaining)
		return commit(state, next, nil)
	}

	// All reasons cleared: resume running, or return to waiting before a start.
	if tickOf(next) == domain.WaitingTick {
		next.Lifecycle = domain.LifecycleWaiting
	} else {
		next.Lifecycle = domain.LifecycleRunning
	}
	events := []Event{{
		Type:   EventResumed,
		WaveNo: next.Wave.Number,
		Tick:   next.Wave.Tick,
		Phase:  next.Wave.Phase,
	}}
	return commit(state, next, events)
}

// stop freezes the session permanently for this run. Stopping twice is an
// idempotent no-op.
func stop(state domain.SessionState, _ Input) (domain.SessionState, []Event, error) {
	if state.Lifecycle == domain.LifecycleStopped {
		return state.Clone(), nil, nil
	}
	next := state.Clone()
	next.PauseReasons = []domain.PauseReason{}
	next.Lifecycle = domain.LifecycleStopped

	events := []Event{{
		Type:   EventStopped,
		WaveNo: next.Wave.Number,
		Tick:   next.Wave.Tick,
		Phase:  next.Wave.Phase,
	}}
	return commit(state, next, events)
}
