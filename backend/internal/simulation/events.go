// Package simulation implements the pure logical progression layer of the
// authoritative simulation (M02-W0202): tick advancement, lifecycle transitions,
// pause reasons, phase boundaries and rollover control.
//
// It is intentionally effects-free: no database, filesystem, clock, network,
// randomness or global state. Every transition takes an explicit state and an
// explicit input and returns a new state plus deterministic events. Gameplay
// resolution (roles, economy, combat, votes, settlement, XP) belongs to later
// work packages; this package only provides the scheduling structure they will
// fill in.
package simulation

import "github.com/hung98dev/ttgqldl/backend/internal/domain"

// Boundary names a scheduling boundary at a specific tick. It is a signal for a
// later package, never gameplay by itself.
type Boundary string

const (
	BoundaryVoteClose      Boundary = "vote_close"
	BoundaryPopulationLock Boundary = "population_lock"
	BoundaryRoleResolution Boundary = "role_resolution"
	BoundaryCombat         Boundary = "combat"
	BoundaryBossSpawn      Boundary = "boss_spawn"
	BoundaryFinalCombat    Boundary = "final_combat"
	BoundaryRollover       Boundary = "rollover"
)

// Stage is one step of the ordered work at a boundary. Tick 2800 must run them
// in exactly this order once W0203-W0207 exist.
type Stage string

const (
	StageConsumeInputs     Stage = "consume_inputs"
	StageFinalContribution Stage = "final_contribution"
	StageFinalCombat       Stage = "final_combat"
	StageSettlement        Stage = "settlement"
	StageEnterResults      Stage = "enter_results"
	StageControlRollover   Stage = "control_rollover"
)

// EventType is a stable event identifier.
type EventType string

const (
	EventWaveStarted        EventType = "wave_started"
	EventPhaseChanged       EventType = "phase_changed"
	EventPaused             EventType = "paused"
	EventResumed            EventType = "resumed"
	EventStopped            EventType = "stopped"
	EventRolledOver         EventType = "wave_rollover"
	EventScheduleBoundary   EventType = "schedule_boundary"
	EventSettlementRequired EventType = "settlement_required"
	// Canonical W0204 enemy signals.
	EventBossIncoming EventType = "boss_incoming"
	EventBossSpawned  EventType = "boss_spawned"
	// Canonical W0205 gameplay signals.
	EventAttackResolved EventType = "attack_resolved"
	EventDamageApplied  EventType = "damage_applied"
	EventWallUpgraded   EventType = "wall_upgraded"
)

// Event is one deterministic transition signal. It carries no participant
// identity, provider identifier, comment or secret.
type Event struct {
	Type     EventType          `json:"type"`
	WaveNo   int64              `json:"wave_no"`
	Tick     int                `json:"tick"`
	Phase    domain.Phase       `json:"phase"`
	Amount   int64              `json:"amount,omitempty"`
	Boundary Boundary           `json:"boundary,omitempty"`
	Reason   domain.PauseReason `json:"reason,omitempty"`
	// Stage marks the ordered work item this event schedules. It is what lets a
	// later reducer prove that final contribution and combat ran before
	// settlement, and that RESULTS was published only afterwards.
	Stage Stage `json:"stage,omitempty"`
}

// BoundariesAt returns the scheduling boundaries that fall on a tick, in the
// canonical execution order. It never decides gameplay: a boundary is a signal.
func BoundariesAt(tick int, bossWave bool) []Boundary {
	result := make([]Boundary, 0, 4)
	if tick == domain.VoteCloseTick {
		result = append(result, BoundaryVoteClose)
	}
	if tick == domain.PopulationLockTick {
		result = append(result, BoundaryPopulationLock)
	}
	if tick > 0 && tick <= domain.FinalCombatTick && tick%200 == 0 {
		result = append(result, BoundaryRoleResolution)
	}
	// The boss spawns after role contribution and before the attack phase.
	if tick == domain.BossSpawnTick && bossWave {
		result = append(result, BoundaryBossSpawn)
	}
	// Ordinary combat boundaries stop at 2780; tick 2800 is the named final
	// combat, so it is never reported twice.
	if tick >= domain.CombatStartTick && tick < domain.FinalCombatTick && tick%200 == 0 {
		result = append(result, BoundaryCombat)
	}
	if tick == domain.FinalCombatTick {
		result = append(result, BoundaryFinalCombat)
	}
	if tick == domain.RolloverTick {
		result = append(result, BoundaryRollover)
	}
	return result
}

// StagesAt returns the ordered stages a tick requires. Tick 2800 is the critical
// boundary: final contribution and combat must complete before settlement, and
// RESULTS is entered only after settlement.
func StagesAt(tick int) []Stage {
	switch tick {
	case domain.FinalCombatTick:
		return []Stage{StageConsumeInputs, StageFinalContribution, StageFinalCombat, StageSettlement, StageEnterResults}
	case domain.RolloverTick:
		return []Stage{StageControlRollover}
	default:
		return nil
	}
}
