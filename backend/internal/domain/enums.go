package domain

import "fmt"

// Role is one participant role. The zero value is invalid on purpose: a
// participant always has exactly one documented role.
type Role int8

const (
	RoleBuilder Role = 1
	RoleGuard   Role = 2
	RoleFighter Role = 3
)

func (r Role) Valid() bool { return r >= RoleBuilder && r <= RoleFighter }

func (r Role) String() string {
	switch r {
	case RoleBuilder:
		return "builder"
	case RoleGuard:
		return "guard"
	case RoleFighter:
		return "fighter"
	default:
		return "invalid"
	}
}

// ParseRole accepts only the canonical lowercase role names.
func ParseRole(name string) (Role, error) {
	switch name {
	case "builder":
		return RoleBuilder, nil
	case "guard":
		return RoleGuard, nil
	case "fighter":
		return RoleFighter, nil
	default:
		return 0, fmt.Errorf("domain: unknown role %q", name)
	}
}

// Lifecycle is the session lifecycle, matching the sessions.lifecycle CHECK.
type Lifecycle int8

const (
	LifecycleWaiting            Lifecycle = 1
	LifecycleRunning            Lifecycle = 2
	LifecyclePaused             Lifecycle = 3
	LifecycleSourceDisconnected Lifecycle = 4
	LifecycleStorageUnavailable Lifecycle = 5
	LifecycleStopped            Lifecycle = 6
)

func (l Lifecycle) Valid() bool { return l >= LifecycleWaiting && l <= LifecycleStopped }

func (l Lifecycle) String() string {
	switch l {
	case LifecycleWaiting:
		return "waiting"
	case LifecycleRunning:
		return "running"
	case LifecyclePaused:
		return "paused"
	case LifecycleSourceDisconnected:
		return "source_disconnected"
	case LifecycleStorageUnavailable:
		return "storage_unavailable"
	case LifecycleStopped:
		return "stopped"
	default:
		return "invalid"
	}
}

// Phase is derived from the local wave tick only.
type Phase int8

const (
	PhaseWaiting Phase = 1
	PhasePrep    Phase = 2
	PhaseAssault Phase = 3
	PhaseFinale  Phase = 4
	PhaseResults Phase = 5
)

func (p Phase) Valid() bool { return p >= PhaseWaiting && p <= PhaseResults }

func (p Phase) String() string {
	switch p {
	case PhaseWaiting:
		return "waiting"
	case PhasePrep:
		return "prep"
	case PhaseAssault:
		return "assault"
	case PhaseFinale:
		return "finale"
	case PhaseResults:
		return "results"
	default:
		return "invalid"
	}
}

// PauseReason is one independent pause cause. Several may be active at once and
// all must clear before time advances again.
type PauseReason int8

const (
	PauseOperatorPaused     PauseReason = 1
	PauseSourceDisconnected PauseReason = 2
	PauseStorageUnavailable PauseReason = 3
)

func (p PauseReason) Valid() bool { return p >= PauseOperatorPaused && p <= PauseStorageUnavailable }

func (p PauseReason) String() string {
	switch p {
	case PauseOperatorPaused:
		return "operator_paused"
	case PauseSourceDisconnected:
		return "source_disconnected"
	case PauseStorageUnavailable:
		return "storage_unavailable"
	default:
		return "invalid"
	}
}

// WaveResult is the settlement outcome of a wave. The zero value means the wave
// has not been settled.
type WaveResult int8

const (
	WaveResultPending WaveResult = 0
	WaveResultVictory WaveResult = 1
	WaveResultDefeat  WaveResult = 2
)

func (w WaveResult) Valid() bool { return w >= WaveResultPending && w <= WaveResultDefeat }

func (w WaveResult) String() string {
	switch w {
	case WaveResultPending:
		return "pending"
	case WaveResultVictory:
		return "victory"
	case WaveResultDefeat:
		return "defeat"
	default:
		return "invalid"
	}
}

// Lane is one approach lane in canonical order.
type Lane int8

const (
	LaneWest   Lane = 0
	LaneCenter Lane = 1
	LaneEast   Lane = 2
)

func (l Lane) Valid() bool { return l >= LaneWest && l <= LaneEast }

func (l Lane) String() string {
	switch l {
	case LaneWest:
		return "west"
	case LaneCenter:
		return "center"
	case LaneEast:
		return "east"
	default:
		return "invalid"
	}
}

// Archetype is an ordinary enemy archetype. Boss is a separate pool and may not
// appear among ordinary subpools.
type Archetype int8

const (
	ArchetypeRunner Archetype = 0
	ArchetypeGrunt  Archetype = 1
	ArchetypeBrute  Archetype = 2
	ArchetypeBoss   Archetype = 3
)

func (a Archetype) Valid() bool { return a >= ArchetypeRunner && a <= ArchetypeBoss }

func (a Archetype) String() string {
	switch a {
	case ArchetypeRunner:
		return "runner"
	case ArchetypeGrunt:
		return "grunt"
	case ArchetypeBrute:
		return "brute"
	case ArchetypeBoss:
		return "boss"
	default:
		return "invalid"
	}
}

// SourceMode is the session input source, matching sessions.mode.
type SourceMode int8

const (
	SourceModeMock           SourceMode = 1
	SourceModeReplay         SourceMode = 2
	SourceModeAuthorizedLive SourceMode = 3
)

func (m SourceMode) Valid() bool {
	return m >= SourceModeMock && m <= SourceModeAuthorizedLive
}

func (m SourceMode) String() string {
	switch m {
	case SourceModeMock:
		return "mock"
	case SourceModeReplay:
		return "replay"
	case SourceModeAuthorizedLive:
		return "authorized_live"
	default:
		return "invalid"
	}
}

// VoteOption is one ballot option. The zero value means "no selection".
type VoteOption int8

const (
	VoteOptionNone VoteOption = 0
	VoteOptionA    VoteOption = 1
	VoteOptionB    VoteOption = 2
	VoteOptionC    VoteOption = 3
)

func (v VoteOption) Valid() bool { return v >= VoteOptionNone && v <= VoteOptionC }

func (v VoteOption) String() string {
	switch v {
	case VoteOptionNone:
		return ""
	case VoteOptionA:
		return "A"
	case VoteOptionB:
		return "B"
	case VoteOptionC:
		return "C"
	default:
		return "invalid"
	}
}

// TickClass separates the waiting sentinel, gameplay ticks and the single
// control rollover boundary.
type TickClass int8

const (
	TickClassWaiting  TickClass = 1
	TickClassGameplay TickClass = 2
	TickClassRollover TickClass = 3
)

func (c TickClass) String() string {
	switch c {
	case TickClassWaiting:
		return "waiting"
	case TickClassGameplay:
		return "gameplay"
	case TickClassRollover:
		return "rollover"
	default:
		return "invalid"
	}
}
