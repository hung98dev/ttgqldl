// Package domain holds the deterministic, pure-game simulation domain for
// TTGQLDL. It has no dependency on the database, HTTP, the filesystem, the wall
// clock or randomness: every value is derived from explicit inputs.
//
// The package implements M02-W0201 only: value types, the deterministic initial
// state, invariants, rules constants, checked integer arithmetic, phase
// derivation and immutable-copy boundaries. Combat, votes settlement, wave
// rollover, persistence and replay belong to later work packages.
package domain

// RulesVersion identifies the binding rule set in docs/plan/02_GAME_RULES.md.
// Changing any gameplay constant requires a new rules version and regenerated
// golden vectors; it must never be bumped for convenience.
const RulesVersion = "ttg-rules-v1"

// CanonicalSchemaVersion is the version of the canonical state serialization
// covered by the RFC 8785 hash.
const CanonicalSchemaVersion = 1

// Simulation cadence and wave geometry.
const (
	TicksPerSecond = 10
	WaveTicks      = 3000 // 300 seconds at 10 Hz: gameplay ticks 0..2999

	// WaitingTick is the lifecycle value before a wave starts.
	WaitingTick = -1
	// RolloverTick is a control transition, never a gameplay tick.
	RolloverTick = 3000
)

// Phase boundaries in local wave ticks, half-open [start, end).
const (
	PrepEndTick    = 1600
	AssaultEndTick = 2600
	FinaleEndTick  = 2800
	ResultsEndTick = 3000
)

// Scheduling boundaries from the rules.
const (
	VoteCloseTick          = 1200
	PopulationLockTick     = 1600
	CombatStartTick        = 1800
	BossSpawnTick          = 2600
	FinalCombatTick        = 2800
	RoleChangeCooldownTick = 600
)

// Village and wall bounds.
const (
	MaxVillageLevel = 50
	MaxWallTier     = 20

	BaseWallMax = 1000
	WallMaxStep = 200

	TownhallMax = 500

	// ResourceCap bounds each of wood, stone and food.
	ResourceCap = 1_000_000
)

// Progression bounds.
const (
	CapSessionXP         = 300
	XPPerWave            = 10
	MaxConsecutiveWins   = 10
	MaxConsecutiveLosses = 3
)

// Initial village values for a brand new village.
const (
	InitialVillageLevel     = 1
	InitialWallTier         = 1
	InitialTownhallHP       = TownhallMax
	InitialWood             = 400
	InitialStone            = 250
	InitialFood             = 300
	InitialConstructionWork = 0
	InitialTotalWins        = 0
	InitialWaveNo           = 1
)

// LaneCount is the number of enemy approach lanes (west, center, east).
const LaneCount = 3

// WallMax returns the maximum wall hit points for a wall tier, using the
// binding formula from 02_GAME_RULES.md and the villages table CHECK:
//
//	wall_hp <= 1000 + 200 * (wall_tier - 1)
func WallMax(tier int) (int64, error) {
	if tier < 1 || tier > MaxWallTier {
		return 0, newError("wall_tier", errWallTierRange)
	}
	return int64(BaseWallMax) + int64(WallMaxStep)*int64(tier-1), nil
}
