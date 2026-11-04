package simulation

import (
	"fmt"

	"github.com/hung98dev/ttgqldl/backend/internal/domain"
)

// lockPopulation applies the tick-1600 population lock: unique current-wave
// members joined before the lock tick, the derived difficulty, the recovery
// modifier, ordinary HP pools and the derivable boss maximum. It is idempotent and
// never fabricates participants.
func lockPopulation(state domain.SessionState) (domain.SessionState, error) {
	if state.Difficulty.Locked {
		return state, nil
	}
	actual := 0
	for _, member := range state.Members {
		if member.JoinedTick <= domain.PopulationLockTick-1 {
			actual++
		}
	}
	population := domain.PopulationForDifficulty(actual)

	difficulty, err := domain.LockedDifficulty(state.ConsecutiveWins)
	if err != nil {
		return state, fmt.Errorf("%w: %v", ErrInvalidState, err)
	}
	recovery, err := domain.RecoveryPercent(state.ConsecutiveLosses)
	if err != nil {
		return state, fmt.Errorf("%w: %v", ErrInvalidState, err)
	}
	baseHP, err := domain.BaseEnemyHP(difficulty, population, state.ConsecutiveLosses)
	if err != nil {
		return state, fmt.Errorf("%w: %v", ErrInvalidState, err)
	}
	pools, err := domain.BuildEnemyPools(baseHP)
	if err != nil {
		return state, fmt.Errorf("%w: %v", ErrInvalidState, err)
	}

	bossMax := int64(0)
	if domain.IsBossWave(state.Wave.Number) {
		bossMax, err = domain.BossMaxHP(baseHP)
		if err != nil {
			return state, fmt.Errorf("%w: %v", ErrInvalidState, err)
		}
	}

	next := state.Clone()
	next.Difficulty = domain.Difficulty{
		Level:            difficulty,
		PopulationAnchor: population,
		Locked:           true,
		ActualPopulation: actual,
		RecoveryPercent:  recovery,
		BaseHP:           baseHP,
		MilitiaAttack:    domain.MilitiaAttack(population),
		BossMaxHP:        bossMax,
	}
	next.Wave.Pools = pools
	return next, nil
}

// spawnBoss creates the boss pool exactly once at tick 2600 on a boss wave. It
// preserves the ordinary pools and never spawns on an ordinary wave.
func spawnBoss(state domain.SessionState) (domain.SessionState, []Event, error) {
	if !domain.IsBossWave(state.Wave.Number) {
		return state, nil, fmt.Errorf("%w: boss spawn on a non-boss wave", ErrInvalidState)
	}
	if !state.Difficulty.Locked || state.Difficulty.BossMaxHP < 1 {
		return state, nil, fmt.Errorf("%w: boss maximum is not derived", ErrInvalidState)
	}
	if state.Wave.Boss.Exists {
		return state, nil, nil
	}
	next := state.Clone()
	next.Wave.Boss = domain.BossPool{
		Exists: true,
		HP:     state.Difficulty.BossMaxHP,
		MaxHP:  state.Difficulty.BossMaxHP,
	}
	events := []Event{{
		Type:   EventBossSpawned,
		WaveNo: next.Wave.Number,
		Tick:   next.Wave.Tick,
		Phase:  next.Wave.Phase,
	}}
	return next, events, nil
}

// EnemyStatus projects the enemy state for public consumption.
func EnemyStatus(state domain.SessionState) domain.EnemyStatus {
	return state.Wave.EnemyStatus(state.Difficulty.Locked)
}
