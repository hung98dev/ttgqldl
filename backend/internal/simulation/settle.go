package simulation

import (
	"fmt"

	"github.com/hung98dev/ttgqldl/backend/internal/domain"
)

// awardXP grants at most 10 session XP to each active participant who has not yet
// been awarded this wave, respecting the session cap. It runs before economy and
// combat at every contribution boundary and never awards to pending members or
// after the wave is defeated.
func awardXP(state domain.SessionState) (domain.SessionState, error) {
	next := state.Clone()
	xpByActor := make(map[domain.ActorKey]int64, len(next.XP))
	for _, row := range next.XP {
		xpByActor[row.ActorKey] = row.XP
	}
	changed := false
	for index := range next.Members {
		member := &next.Members[index]
		if member.XPAwarded || member.JoinedTick > next.Tick() {
			continue
		}
		current := xpByActor[member.ActorKey]
		if current >= domain.CapSessionXP {
			member.XPAwarded = true
			continue
		}
		gain := int64(domain.XPPerWave)
		if current+gain > domain.CapSessionXP {
			gain = domain.CapSessionXP - current
		}
		updated, err := domain.AddInt64(current, gain)
		if err != nil {
			return state, fmt.Errorf("%w: xp overflow", ErrInvalidState)
		}
		if updated > domain.CapSessionXP {
			updated = domain.CapSessionXP
		}
		xpByActor[member.ActorKey] = updated
		member.XPAwarded = true
		changed = true
	}
	if !changed {
		return state, nil
	}
	rows := make([]domain.PlayerXP, 0, len(xpByActor))
	for actor, xp := range xpByActor {
		rows = append(rows, domain.PlayerXP{ActorKey: actor, XP: xp})
	}
	domain.SortPlayerXP(rows)
	next.XP = rows
	if err := next.Validate(); err != nil {
		return state, fmt.Errorf("%w: xp produced an invalid state: %v", ErrInvalidState, err)
	}
	return next, nil
}

// victoryReached applies the binding victory condition: the town hall stands, all
// ordinary pools are empty, and every boss on a boss wave spawned and died.
func victoryReached(state domain.SessionState) bool {
	if state.Village.TownhallHP <= 0 {
		return false
	}
	for _, pool := range state.Wave.Pools {
		if pool.HP > 0 {
			return false
		}
	}
	if domain.IsBossWave(state.Wave.Number) {
		if !state.Wave.Boss.Exists || state.Wave.Boss.HP > 0 {
			return false
		}
	}
	return true
}

// settleWave performs the exactly-once tick-2800 settlement: victory rewards or
// loss penalties, streaks, lifetime wins, village level and (on actual defeat)
// recovery floors. A second call is a no-op.
func settleWave(state domain.SessionState) (domain.SessionState, []Event, error) {
	if state.Wave.Result != domain.WaveResultPending {
		return state, nil, nil
	}
	next := state.Clone()
	if victoryReached(next) {
		wood, err := domain.AddInt64(next.Village.Resources.Wood, 100)
		if err != nil {
			return state, nil, err
		}
		stone, err := domain.AddInt64(next.Village.Resources.Stone, 60)
		if err != nil {
			return state, nil, err
		}
		food, err := domain.AddInt64(next.Village.Resources.Food, 50)
		if err != nil {
			return state, nil, err
		}
		next.Village.Resources = domain.Resources{
			Wood:  clampCap(wood),
			Stone: clampCap(stone),
			Food:  clampCap(food),
		}
		next.ConsecutiveWins, err = domain.AddInt64(next.ConsecutiveWins, 1)
		if err != nil {
			return state, nil, err
		}
		if next.ConsecutiveWins > domain.MaxConsecutiveWins {
			next.ConsecutiveWins = domain.MaxConsecutiveWins
		}
		next.ConsecutiveLosses = 0
		next.Village.TotalWins, err = domain.AddInt64(next.Village.TotalWins, 1)
		if err != nil {
			return state, nil, err
		}
		next.Village.Level = villageLevel(next.Village.TotalWins)
		next.Wave.Result = domain.WaveResultVictory
	} else {
		next.ConsecutiveWins = 0
		next.ConsecutiveLosses++
		if next.ConsecutiveLosses > domain.MaxConsecutiveLosses {
			next.ConsecutiveLosses = domain.MaxConsecutiveLosses
		}
		woodPenalty := next.Village.Resources.Wood / 5
		stonePenalty := next.Village.Resources.Stone / 5
		foodPenalty := next.Village.Resources.Food / 5
		if next.Wave.Vote.Closed && next.Wave.Vote.Selected == domain.VoteOptionC && domain.VoteAvailable(next.Wave.Number) {
			var err error
			if woodPenalty, err = domain.LossPenaltyAfterC(woodPenalty); err != nil {
				return state, nil, err
			}
			if stonePenalty, err = domain.LossPenaltyAfterC(stonePenalty); err != nil {
				return state, nil, err
			}
			if foodPenalty, err = domain.LossPenaltyAfterC(foodPenalty); err != nil {
				return state, nil, err
			}
		}
		next.Village.Resources.Wood = clampZero(next.Village.Resources.Wood - woodPenalty)
		next.Village.Resources.Stone = clampZero(next.Village.Resources.Stone - stonePenalty)
		next.Village.Resources.Food = clampZero(next.Village.Resources.Food - foodPenalty)
		if next.Wave.Defeated {
			wallMax, err := next.Village.WallMax()
			if err != nil {
				return state, nil, err
			}
			if next.Village.WallHP < 250 {
				next.Village.WallHP = 250
			}
			if next.Village.WallHP > wallMax {
				next.Village.WallHP = wallMax
			}
			if next.Village.TownhallHP < 250 {
				next.Village.TownhallHP = 250
			}
			if next.Village.TownhallHP > domain.TownhallMax {
				next.Village.TownhallHP = domain.TownhallMax
			}
			if next.Village.Resources.Wood < 100 {
				next.Village.Resources.Wood = 100
			}
			if next.Village.Resources.Wood > domain.ResourceCap {
				next.Village.Resources.Wood = domain.ResourceCap
			}
		}
		next.Wave.Result = domain.WaveResultDefeat
	}
	if err := next.Validate(); err != nil {
		return state, nil, fmt.Errorf("%w: settlement produced an invalid state: %v", ErrInvalidState, err)
	}
	events := []Event{{
		Type:   EventWaveResult,
		WaveNo: next.Wave.Number,
		Tick:   next.Wave.Tick,
		Phase:  next.Wave.Phase,
		Amount: int64(next.Wave.Result),
	}}
	return next, events, nil
}

func villageLevel(totalWins int64) int {
	if totalWins < 0 {
		return 1
	}
	level := int64(1) + totalWins/3
	if level > domain.MaxVillageLevel {
		return domain.MaxVillageLevel
	}
	return int(level)
}

func clampCap(value int64) int64 {
	if value > domain.ResourceCap {
		return domain.ResourceCap
	}
	if value < 0 {
		return 0
	}
	return value
}

func clampZero(value int64) int64 {
	if value < 0 {
		return 0
	}
	if value > domain.ResourceCap {
		return domain.ResourceCap
	}
	return value
}
