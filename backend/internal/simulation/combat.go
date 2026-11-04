package simulation

import (
	"fmt"

	"github.com/hung98dev/ttgqldl/backend/internal/domain"
)

// combatIndex is the zero-based index of a combat boundary (1800..2800).
func combatIndex(tick int) (int, error) {
	if tick < domain.CombatStartTick || tick > domain.FinalCombatTick || (tick-domain.CombatStartTick)%200 != 0 {
		return 0, fmt.Errorf("%w: tick %d is not a combat boundary", ErrInvalidState, tick)
	}
	return (tick - domain.CombatStartTick) / 200, nil
}

// resolveContribution runs the builder economy, wall repair and at most one wall
// upgrade for one contribution boundary. It never awards XP (W0207) and never
// runs for a defeated wave.
func resolveContribution(state domain.SessionState, events *[]Event) (domain.SessionState, error) {
	// Builder work is independent of the enemy lock: it runs through PREP.
	if state.Wave.Defeated {
		return state, nil
	}
	if state.Lifecycle != domain.LifecycleRunning || len(state.PauseReasons) > 0 {
		return state, nil
	}
	next := state.Clone()
	counts := ActiveCounts(next, next.Tick())
	builders, err := capped(counts.Builder)
	if err != nil {
		return state, err
	}
	root, err := domain.Isqrt(builders)
	if err != nil {
		return state, err
	}
	twiceRoot, err := domain.MulInt64(2, root)
	if err != nil {
		return state, err
	}
	perFifty, err := domain.SafeRatio(builders, 50, 1)
	if err != nil {
		return state, err
	}
	woodGain, err := domain.AddInt64(twiceRoot, perFifty)
	if err != nil {
		return state, err
	}
	perFiveHundred, err := domain.SafeRatio(builders, 500, 1)
	if err != nil {
		return state, err
	}
	stoneGain, err := domain.AddInt64(root, perFiveHundred)
	if err != nil {
		return state, err
	}
	if next.Village.Resources.Wood, err = clampResource(next.Village.Resources.Wood, woodGain); err != nil {
		return state, err
	}
	if next.Village.Resources.Stone, err = clampResource(next.Village.Resources.Stone, stoneGain); err != nil {
		return state, err
	}

	wallMax, err := next.Village.WallMax()
	if err != nil {
		return state, err
	}
	capacity, err := domain.MulInt64(8, root)
	if err != nil {
		return state, err
	}
	repairable, err := domain.SubInt64(wallMax, next.Village.WallHP)
	if err != nil {
		return state, err
	}
	affordable, err := domain.MulInt64(5, next.Village.Resources.Wood)
	if err != nil {
		return state, err
	}
	repair, err := minInt64(capacity, repairable)
	if err != nil {
		return state, err
	}
	repair, err = minInt64(repair, affordable)
	if err != nil {
		return state, err
	}
	woodCost, err := domain.CeilDiv(repair, 5)
	if err != nil {
		return state, err
	}
	if woodCost > next.Village.Resources.Wood {
		return state, fmt.Errorf("%w: repair wood cost exceeds stock", ErrInvalidState)
	}
	next.Village.Resources.Wood -= woodCost
	next.Village.WallHP += repair

	unused, err := domain.SubInt64(capacity, repair)
	if err != nil {
		return state, err
	}
	if counts.Builder > 0 && next.Village.WallTier < domain.MaxWallTier {
		threshold, err := domain.MulInt64(200, int64(next.Village.WallTier))
		if err != nil {
			return state, err
		}
		stoneCost, err := domain.MulInt64(100, int64(next.Village.WallTier))
		if err != nil {
			return state, err
		}
		accumulated, err := domain.AddInt64(next.Village.ConstructionWork, unused)
		if err != nil {
			return state, err
		}
		if accumulated > threshold {
			accumulated = threshold
		}
		next.Village.ConstructionWork = accumulated
		if accumulated >= threshold && next.Village.Resources.Stone >= stoneCost {
			next.Village.ConstructionWork -= threshold
			next.Village.Resources.Stone -= stoneCost
			next.Village.WallTier++
			newMax, err := next.Village.WallMax()
			if err != nil {
				return state, err
			}
			next.Village.WallHP += 200
			if next.Village.WallHP > newMax {
				next.Village.WallHP = newMax
			}
			emitEvent(events, Event{
				Type:   EventWallUpgraded,
				WaveNo: next.Wave.Number,
				Tick:   next.Wave.Tick,
				Phase:  next.Wave.Phase,
				Amount: int64(next.Village.WallTier),
			})
		}
	}
	// Clamp against the tier in effect after any upgrade, never the stale maximum.
	finalMax, err := next.Village.WallMax()
	if err != nil {
		return state, err
	}
	if next.Village.WallHP > finalMax {
		next.Village.WallHP = finalMax
	}
	if err := next.Validate(); err != nil {
		return state, fmt.Errorf("%w: economy produced an invalid state: %v", ErrInvalidState, err)
	}
	return next, nil
}

// resolveCombat runs one combat boundary: fighter+militia attack distributed over
// the three lanes, retaliation from surviving HP, guard mitigation, wall then
// townhall damage, and immediate defeat. It never settles the wave or awards XP.
func resolveCombat(state domain.SessionState, events *[]Event) (domain.SessionState, error) {
	if state.Wave.Defeated || !state.Difficulty.Locked {
		return state, nil
	}
	if state.Lifecycle != domain.LifecycleRunning || len(state.PauseReasons) > 0 {
		return state, nil
	}
	index, err := combatIndex(state.Wave.Tick)
	if err != nil {
		return state, err
	}
	next := state.Clone()
	counts := ActiveCounts(next, next.Tick())
	fighters, err := capped(counts.Fighter)
	if err != nil {
		return state, err
	}
	guards, err := capped(counts.Guard)
	if err != nil {
		return state, err
	}
	fighterRoot, err := domain.Isqrt(fighters)
	if err != nil {
		return state, err
	}
	fighterAttack, err := domain.MulInt64(18, fighterRoot)
	if err != nil {
		return state, err
	}
	totalAttack, err := domain.AddInt64(fighterAttack, next.Difficulty.MilitiaAttack)
	if err != nil {
		return state, err
	}
	// Vote B raises the complete attack before lane splitting; no other option
	// changes attack. Odd waves and unselected votes keep 100%.
	attackPercent, guardPercent := VoteModifiers(next)
	if attackPercent != 100 {
		totalAttack, err = domain.ApplyVotePercent(totalAttack, attackPercent)
		if err != nil {
			return state, err
		}
	}

	distributed, err := distributeAttack(next, totalAttack, index)
	if err != nil {
		return state, err
	}
	next = distributed

	ordinaryRemaining := int64(0)
	for _, pool := range next.Wave.Pools {
		ordinaryRemaining += pool.HP
	}
	population, err := domain.Clamp(int64(next.Difficulty.PopulationAnchor), 1, domain.MaxPopulationAnchor)
	if err != nil {
		return state, err
	}
	populationRoot, err := domain.Isqrt(population)
	if err != nil {
		return state, err
	}
	nineRoot, err := domain.MulInt64(9, populationRoot)
	if err != nil {
		return state, err
	}
	threeD, err := domain.MulInt64(3, int64(next.Difficulty.Level))
	if err != nil {
		return state, err
	}
	baseRaw, err := domain.AddInt64(10, threeD)
	if err != nil {
		return state, err
	}
	baseRaw, err = domain.AddInt64(baseRaw, nineRoot)
	if err != nil {
		return state, err
	}

	var raw int64
	if ordinaryRemaining > 0 {
		scaled, err := domain.SafeRatio(ordinaryRemaining, next.Difficulty.BaseHP, 1)
		if err != nil {
			return state, err
		}
		if scaled == 0 {
			scaled = 1
		}
		numerator, err := domain.MulInt64(baseRaw, ordinaryRemaining)
		if err != nil {
			return state, err
		}
		// ceil(baseRaw * remaining / baseHP) = SafeRatio rounded up.
		ordinaryRaw, err := domain.SafeRatio(numerator, next.Difficulty.BaseHP, 1)
		if err != nil {
			return state, err
		}
		if numerator%next.Difficulty.BaseHP != 0 {
			ordinaryRaw++
		}
		raw = ordinaryRaw
	}
	if next.Wave.Boss.Exists && next.Wave.Boss.HP > 0 {
		bossRaw, err := domain.CeilDiv(baseRaw, 5)
		if err != nil {
			return state, err
		}
		raw, err = domain.AddInt64(raw, bossRaw)
		if err != nil {
			return state, err
		}
	}

	guardRoot, err := domain.Isqrt(guards)
	if err != nil {
		return state, err
	}
	guardBlock, err := domain.MulInt64(8, guardRoot)
	if err != nil {
		return state, err
	}
	// Vote A raises and vote C lowers the guard block after the base block is
	// computed; the modifier never changes the guard player count.
	if guardPercent != 100 {
		guardBlock, err = domain.ApplyVotePercent(guardBlock, guardPercent)
		if err != nil {
			return state, err
		}
	}
	damage := int64(0)
	if raw > guardBlock {
		damage = raw - guardBlock
	}

	wallDamage, err := minInt64(damage, next.Village.WallHP)
	if err != nil {
		return state, err
	}
	next.Village.WallHP -= wallDamage
	overflow, err := domain.SubInt64(damage, wallDamage)
	if err != nil {
		return state, err
	}
	townhallDamage, err := minInt64(overflow, next.Village.TownhallHP)
	if err != nil {
		return state, err
	}
	next.Village.TownhallHP -= townhallDamage
	if next.Village.TownhallHP == 0 {
		next.Wave.Defeated = true
	}

	emitEvent(events, Event{
		Type:   EventAttackResolved,
		WaveNo: next.Wave.Number,
		Tick:   next.Wave.Tick,
		Phase:  next.Wave.Phase,
		Amount: totalAttack,
	})
	if damage > 0 {
		emitEvent(events, Event{
			Type:   EventDamageApplied,
			WaveNo: next.Wave.Number,
			Tick:   next.Wave.Tick,
			Phase:  next.Wave.Phase,
			Amount: damage,
		})
	}
	if err := next.Validate(); err != nil {
		return state, fmt.Errorf("%w: combat produced an invalid state: %v", ErrInvalidState, err)
	}
	return next, nil
}

// distributeAttack applies attack to the three lanes and then redistributes any
// unused damage across all living pools, boss last in the center lane.
func distributeAttack(state domain.SessionState, attack int64, index int) (domain.SessionState, error) {
	if attack <= 0 {
		return state, nil
	}
	next := state.Clone()
	shares := [domain.LaneCount]int64{}
	baseShare := attack / 3
	remainder := attack % 3
	for lane := 0; lane < domain.LaneCount; lane++ {
		shares[lane] = baseShare
	}
	start := index % domain.LaneCount
	for i := int64(0); i < remainder; i++ {
		shares[(start+int(i))%domain.LaneCount]++
	}

	// Index the canonical pools per lane: runner, grunt, brute.
	pools := [domain.LaneCount][3]int{}
	for lane := 0; lane < domain.LaneCount; lane++ {
		pools[lane] = [3]int{-1, -1, -1}
	}
	for i, pool := range next.Wave.Pools {
		if int(pool.Lane) < domain.LaneCount {
			pools[int(pool.Lane)][int(pool.Archetype)] = i
		}
	}

	leftover := int64(0)
	for lane := 0; lane < domain.LaneCount; lane++ {
		remaining := shares[lane]
		for _, poolIndex := range pools[lane] {
			if remaining == 0 {
				break
			}
			if poolIndex < 0 {
				continue
			}
			consumed := min64(remaining, next.Wave.Pools[poolIndex].HP)
			next.Wave.Pools[poolIndex].HP -= consumed
			remaining -= consumed
		}
		// Boss is attacked after all ordinary center-lane pools, only if spawned.
		if lane == int(domain.LaneCenter) && remaining > 0 && next.Wave.Boss.Exists && next.Wave.Boss.HP > 0 {
			consumed := min64(remaining, next.Wave.Boss.HP)
			next.Wave.Boss.HP -= consumed
			remaining -= consumed
		}
		leftover += remaining
	}

	// Round two: visit all living pools cyclically from the same start lane.
	if leftover > 0 {
		visited := 0
		for leftover > 0 && visited < 2 {
			visited++
			for offset := 0; offset < domain.LaneCount && leftover > 0; offset++ {
				lane := (start + offset) % domain.LaneCount
				for _, poolIndex := range pools[lane] {
					if leftover == 0 {
						break
					}
					if poolIndex < 0 || next.Wave.Pools[poolIndex].HP == 0 {
						continue
					}
					consumed := min64(leftover, next.Wave.Pools[poolIndex].HP)
					next.Wave.Pools[poolIndex].HP -= consumed
					leftover -= consumed
				}
				if lane == int(domain.LaneCenter) && leftover > 0 && next.Wave.Boss.Exists && next.Wave.Boss.HP > 0 {
					consumed := min64(leftover, next.Wave.Boss.HP)
					next.Wave.Boss.HP -= consumed
					leftover -= consumed
				}
			}
		}
	}
	return next, nil
}

// emitEvent appends an event when the caller provided a sink.
func emitEvent(events *[]Event, event Event) {
	if events == nil {
		return
	}
	*events = append(*events, event)
}

func min64(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

func minInt64(a, b int64) (int64, error) {
	if a < 0 || b < 0 {
		return 0, domain.ErrNegative
	}
	return min64(a, b), nil
}

func capped(value int) (int64, error) {
	if value < 0 {
		return 0, domain.ErrNegative
	}
	if value > domain.MaxPopulationAnchor {
		return domain.MaxPopulationAnchor, nil
	}
	return int64(value), nil
}

func clampResource(current, gain int64) (int64, error) {
	sum, err := domain.AddInt64(current, gain)
	if err != nil {
		return 0, err
	}
	if sum > domain.ResourceCap {
		sum = domain.ResourceCap
	}
	return sum, nil
}
