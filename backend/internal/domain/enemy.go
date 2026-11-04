package domain

import "fmt"

// LockedDifficulty derives the wave difficulty from the consecutive-win streak.
func LockedDifficulty(consecutiveWins int64) (int, error) {
	if consecutiveWins < 0 {
		return 0, ErrNegative
	}
	if consecutiveWins >= MaxConsecutiveWins {
		return MaxConsecutiveWins, nil
	}
	return int(consecutiveWins) + 1, nil
}

// RecoveryPercent returns the recovery modifier applied to ordinary enemy HP:
// 100 - 15*min(consecutiveLosses,3).
func RecoveryPercent(consecutiveLosses int64) (int, error) {
	if consecutiveLosses < 0 {
		return 0, ErrNegative
	}
	losses, err := Clamp(consecutiveLosses, 0, MaxConsecutiveLosses)
	if err != nil {
		return 0, err
	}
	penalty, err := MulInt64(15, losses)
	if err != nil {
		return 0, err
	}
	value, err := SubInt64(100, penalty)
	if err != nil {
		return 0, err
	}
	return int(value), nil
}

// PopulationForDifficulty caps the anchor for formulas: P = max(1, min(count,10000)).
// It never changes the truthful participant count itself.
func PopulationForDifficulty(actual int) int {
	if actual < 1 {
		return 1
	}
	if actual > MaxPopulationAnchor {
		return MaxPopulationAnchor
	}
	return actual
}

// BaseEnemyHP computes the ordinary enemy HP for one wave:
//
//	max(1, floor((40 + 8*d + 38*isqrt(P)) * recovery_percent / 100))
func BaseEnemyHP(difficulty int, population int, consecutiveLosses int64) (int64, error) {
	if difficulty < 1 || difficulty > 10 {
		return 0, newError("difficulty", "must be within 1..10")
	}
	if population < 1 || population > MaxPopulationAnchor {
		return 0, newError("population", "must be within 1..10000")
	}
	root, err := Isqrt(int64(population))
	if err != nil {
		return 0, err
	}
	scaledRoot, err := MulInt64(38, root)
	if err != nil {
		return 0, err
	}
	scaledDifficulty, err := MulInt64(8, int64(difficulty))
	if err != nil {
		return 0, err
	}
	core, err := AddInt64(40, scaledDifficulty)
	if err != nil {
		return 0, err
	}
	core, err = AddInt64(core, scaledRoot)
	if err != nil {
		return 0, err
	}
	recovery, err := RecoveryPercent(consecutiveLosses)
	if err != nil {
		return 0, err
	}
	hp, err := SafeRatio(core, 100, int64(recovery))
	if err != nil {
		return 0, err
	}
	if hp < 1 {
		return 1, nil
	}
	return hp, nil
}

// LaneHPs splits ordinary HP across the three lanes with exact conservation:
// west = floor(base*34/100), center = floor(base*32/100), east = remainder.
func LaneHPs(baseHP int64) (west, center, east int64, err error) {
	if baseHP < 0 {
		return 0, 0, 0, ErrNegative
	}
	west, err = SafeRatio(baseHP, 100, 34)
	if err != nil {
		return 0, 0, 0, err
	}
	center, err = SafeRatio(baseHP, 100, 32)
	if err != nil {
		return 0, 0, 0, err
	}
	east, err = SubInt64(baseHP, west)
	if err != nil {
		return 0, 0, 0, err
	}
	east, err = SubInt64(east, center)
	if err != nil {
		return 0, 0, 0, err
	}
	return west, center, east, nil
}

// SubpoolHPs splits one lane with exact conservation: runner = floor(lane*25/100),
// grunt = floor(lane*55/100), brute = remainder.
func SubpoolHPs(laneHP int64) (runner, grunt, brute int64, err error) {
	if laneHP < 0 {
		return 0, 0, 0, ErrNegative
	}
	runner, err = SafeRatio(laneHP, 100, 25)
	if err != nil {
		return 0, 0, 0, err
	}
	grunt, err = SafeRatio(laneHP, 100, 55)
	if err != nil {
		return 0, 0, 0, err
	}
	brute, err = SubInt64(laneHP, runner)
	if err != nil {
		return 0, 0, 0, err
	}
	brute, err = SubInt64(brute, grunt)
	if err != nil {
		return 0, 0, 0, err
	}
	return runner, grunt, brute, nil
}

// MilitiaAttack returns the beginner militia support for the capped population.
// It is system assistance, never a registered player and never an XP source.
func MilitiaAttack(population int) int64 {
	if population < 1 || population > 10 {
		return 0
	}
	return int64(40 - 4*(population-1))
}

// BossMaxHP returns ceil(baseHP/5) for a boss wave.
func BossMaxHP(baseHP int64) (int64, error) {
	return CeilDiv(baseHP, 5)
}

// BuildEnemyPools derives the nine canonical ordinary subpools for a locked wave.
// Pools are ordered west/center/east, then runner/grunt/brute, and their sum is
// exactly baseHP.
func BuildEnemyPools(baseHP int64) ([]EnemyPool, error) {
	west, center, east, err := LaneHPs(baseHP)
	if err != nil {
		return nil, err
	}
	laneHP := []int64{west, center, east}
	pools := make([]EnemyPool, 0, LaneCount*3)
	for lane := LaneWest; lane <= LaneEast; lane++ {
		runner, grunt, brute, splitErr := SubpoolHPs(laneHP[int(lane)])
		if splitErr != nil {
			return nil, splitErr
		}
		pools = append(pools,
			EnemyPool{Lane: lane, Archetype: ArchetypeRunner, HP: runner},
			EnemyPool{Lane: lane, Archetype: ArchetypeGrunt, HP: grunt},
			EnemyPool{Lane: lane, Archetype: ArchetypeBrute, HP: brute},
		)
	}
	return pools, nil
}

// EnemyStatus is the pure enemy-state projection. CLEAR is never reported while a
// boss spawn is still pending.
type EnemyStatus string

const (
	EnemyStatusWaiting      EnemyStatus = "waiting"
	EnemyStatusPreparing    EnemyStatus = "preparing"
	EnemyStatusEngaged      EnemyStatus = "engaged"
	EnemyStatusBossIncoming EnemyStatus = "BOSS_INCOMING"
	EnemyStatusClear        EnemyStatus = "clear"
)

// Status projects the enemy state from authoritative values only.
func (w WaveState) EnemyStatus(locked bool) EnemyStatus {
	if !locked {
		if w.Phase == PhaseWaiting {
			return EnemyStatusWaiting
		}
		return EnemyStatusPreparing
	}
	remaining := int64(0)
	for _, pool := range w.Pools {
		remaining += pool.HP
	}
	bossWave := IsBossWave(w.Number)
	switch {
	case remaining > 0:
		return EnemyStatusEngaged
	case bossWave && !w.Boss.Exists:
		// Ordinary pools are empty before the boss spawn: never CLEAR.
		return EnemyStatusBossIncoming
	case bossWave && w.Boss.Exists && w.Boss.HP > 0:
		return EnemyStatusEngaged
	default:
		return EnemyStatusClear
	}
}

func (e EnemyStatus) String() string { return string(e) }

// validateEnemyLock checks the derived difficulty and pool conservation.
func validateEnemyLock(w WaveState, d Difficulty) *ValidationError {
	errs := &ValidationError{}
	if !d.Locked {
		if len(w.Pools) > 0 && len(w.Pools) != LaneCount*3 {
			errs.add("wave.pools", "unlocked pools must be canonical")
		}
		return nil
	}
	if d.Level < 1 || d.Level > 10 {
		errs.add("session.difficulty.level", "must be within 1..10")
	}
	if d.PopulationAnchor < 1 || d.PopulationAnchor > MaxPopulationAnchor {
		errs.add("session.difficulty.population_anchor", "must be within 1..10000 when locked")
	}
	if d.ActualPopulation < 0 {
		errs.add("session.difficulty.actual_population", "must not be negative")
	} else if d.PopulationAnchor != PopulationForDifficulty(d.ActualPopulation) {
		errs.add("session.difficulty.population_anchor", "must equal max(1,min(actual,10000))")
	}
	switch d.RecoveryPercent {
	case 100, 85, 70, 55:
	default:
		errs.add("session.difficulty.recovery_percent", "must be 100, 85, 70 or 55")
	}
	if d.BaseHP < 1 {
		errs.add("session.difficulty.base_hp", "must be at least 1")
	}
	if d.MilitiaAttack < 0 || d.MilitiaAttack > 40 {
		errs.add("session.difficulty.militia_attack", "must be within 0..40")
	}
	if IsBossWave(w.Number) {
		if d.BossMaxHP < 1 {
			errs.add("session.difficulty.boss_max_hp", "must be positive on a boss wave")
		}
	} else if d.BossMaxHP != 0 {
		errs.add("session.difficulty.boss_max_hp", "must be zero on an ordinary wave")
	}
	if len(w.Pools) != LaneCount*3 {
		errs.add("wave.pools", fmt.Sprintf("locked wave needs %d pools, got %d", LaneCount*3, len(w.Pools)))
	} else {
		total := int64(0)
		for _, pool := range w.Pools {
			total += pool.HP
		}
		// Combat only depletes HP, so the live total can never exceed base_hp.
		if total > d.BaseHP {
			errs.add("wave.pools", fmt.Sprintf("HP total %d exceeds base_hp %d", total, d.BaseHP))
		}
	}
	if w.Boss.Exists {
		if d.BossMaxHP > 0 && w.Boss.MaxHP != d.BossMaxHP {
			errs.add("wave.boss.max_hp", "must equal the derived boss maximum")
		}
	}
	return errs
}
