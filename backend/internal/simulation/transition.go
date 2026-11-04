package simulation

import (
	"errors"
	"fmt"

	"github.com/hung98dev/ttgqldl/backend/internal/domain"
)

// Sentinel errors. Every transition error wraps exactly one of them so callers
// can branch without parsing prose.
var (
	ErrInvalidState    = errors.New("simulation: invalid state")
	ErrInvalidInput    = errors.New("simulation: invalid input")
	ErrNotRunning      = errors.New("simulation: wave is not running")
	ErrTickFrozen      = errors.New("simulation: logical tick is frozen")
	ErrStopped         = errors.New("simulation: session is stopped")
	ErrNotAtRollover   = errors.New("simulation: rollover requires tick 3000")
	ErrControlBoundary = errors.New("simulation: tick 3000 is a control boundary")
	ErrOutOfOrder      = errors.New("simulation: out-of-order transition")
	ErrNoRegistration  = errors.New("simulation: start requires an accepted registration")
	ErrUnsupported     = errors.New("simulation: transition not supported in W0202")
)

// Kind identifies a transition request.
type Kind int8

const (
	KindStartWave   Kind = 1
	KindAdvanceTick Kind = 2
	KindRollover    Kind = 3
	KindPause       Kind = 4
	KindResume      Kind = 5
	KindStop        Kind = 6
)

func (k Kind) String() string {
	switch k {
	case KindStartWave:
		return "start_wave"
	case KindAdvanceTick:
		return "advance_tick"
	case KindRollover:
		return "rollover"
	case KindPause:
		return "pause"
	case KindResume:
		return "resume"
	case KindStop:
		return "stop"
	default:
		return "invalid"
	}
}

// Registration is one explicit accepted role registration. Parsing commands and
// admitting players belongs to W0203; W0202 requires the caller to supply a
// already-validated registration and never fabricates one.
type Registration struct {
	ActorKey domain.ActorKey
	Role     domain.Role
}

// Input is one explicit transition request. All context is passed in; nothing is
// read from a clock or from the environment.
type Input struct {
	Kind Kind

	// Reason is required for KindPause and KindResume.
	Reason domain.PauseReason

	// Registration is required for KindStartWave.
	Registration *Registration

	// FromTick, when set, asserts the caller's expected current tick. A stale
	// request is rejected instead of being applied out of order.
	FromTick *int
}

// Next applies one transition. The input state is never mutated: on error the
// caller's state is untouched, and on success a deep copy is returned.
func Next(state domain.SessionState, input Input) (domain.SessionState, []Event, error) {
	if err := state.Validate(); err != nil {
		return state, nil, fmt.Errorf("%w: %v", ErrInvalidState, err)
	}
	if input.FromTick != nil && *input.FromTick != state.Tick() {
		return state, nil, fmt.Errorf("%w: expected tick %d, state is at %d", ErrOutOfOrder, *input.FromTick, state.Tick())
	}

	switch input.Kind {
	case KindStartWave:
		return startWave(state, input)
	case KindAdvanceTick:
		return advanceTick(state, input)
	case KindRollover:
		return rollover(state, input)
	case KindPause:
		return pause(state, input)
	case KindResume:
		return resume(state, input)
	case KindStop:
		return stop(state, input)
	default:
		return state, nil, fmt.Errorf("%w: unknown kind %d", ErrInvalidInput, input.Kind)
	}
}

// commit validates the produced state and returns it. An internal invariant
// violation is reported as a bug rather than persisted.
func commit(original, next domain.SessionState, events []Event) (domain.SessionState, []Event, error) {
	if err := next.Validate(); err != nil {
		return original, nil, fmt.Errorf("%w: transition produced an invalid state: %v", ErrInvalidState, err)
	}
	return next, events, nil
}

// tick returns the current logical tick.
func tickOf(state domain.SessionState) int { return state.Wave.Tick }
