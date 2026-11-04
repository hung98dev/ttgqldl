package domain

import (
	"sort"
	"strconv"
)

// EnemyPool is one (lane, archetype) ordinary HP subpool.
type EnemyPool struct {
	Lane      Lane
	Archetype Archetype
	HP        int64
}

// BossPool is the boss HP pool. It does not exist before BossSpawnTick; only the
// derivable maximum is known before that.
type BossPool struct {
	Exists bool
	HP     int64
	MaxHP  int64
}

// SortPools orders pools canonically: west, center, east, then runner, grunt,
// brute. Map iteration order is never trusted.
func SortPools(pools []EnemyPool) {
	sort.SliceStable(pools, func(i, j int) bool {
		if pools[i].Lane != pools[j].Lane {
			return pools[i].Lane < pools[j].Lane
		}
		return pools[i].Archetype < pools[j].Archetype
	})
}

// IsBossWave reports whether waveNo is divisible by five.
func IsBossWave(waveNo int64) bool { return waveNo > 0 && waveNo%5 == 0 }

// ValidatePools checks lane/archetype values, canonical order, non-negative HP
// and boss invariants. Ordinary pools must never carry the boss archetype and a
// boss pool may not exist before its spawn tick.
func ValidatePools(pools []EnemyPool, boss BossPool, waveNo int64, tick int) *ValidationError {
	errs := &ValidationError{}
	var previousLane Lane
	var previousArchetype Archetype
	for index, pool := range pools {
		field := "wave.pools[" + strconv.Itoa(index) + "]"
		if !pool.Lane.Valid() {
			errs.add(field+".lane", errLaneInvalid)
		}
		if !pool.Archetype.Valid() {
			errs.add(field+".archetype", errArchetypeInvalid)
		}
		if pool.Archetype == ArchetypeBoss {
			errs.add(field+".archetype", errOrdinaryPoolBoss)
		}
		if pool.HP < 0 {
			errs.add(field+".hp", errNegativeHP)
		}
		if index > 0 {
			if pool.Lane < previousLane || (pool.Lane == previousLane && pool.Archetype <= previousArchetype) {
				errs.add(field, errPoolOrder)
			}
		}
		previousLane, previousArchetype = pool.Lane, pool.Archetype
	}

	if boss.Exists {
		if !IsBossWave(waveNo) {
			errs.add("wave.boss", errBossWaveMismatch)
		}
		if tick < BossSpawnTick {
			errs.add("wave.boss", errBossBeforeSpawn)
		}
		if boss.HP <= 0 || boss.MaxHP <= 0 || boss.HP > boss.MaxHP {
			errs.add("wave.boss", errBossHP)
		}
	} else {
		if boss.HP != 0 {
			errs.add("wave.boss.hp", "must be zero while the boss pool does not exist")
		}
		if boss.MaxHP < 0 {
			errs.add("wave.boss.max_hp", errNegativeHP)
		}
	}

	if len(errs.Problems) == 0 {
		return nil
	}
	return errs
}

func itoa(value int) string { return strconv.Itoa(value) }
