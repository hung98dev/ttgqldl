package simulation

import (
	"fmt"

	"github.com/hung98dev/ttgqldl/backend/internal/domain"
)

// resolveBoundary applies all authoritative gameplay effects that belong to the
// committed logical tick, in canonical order. It runs only for a running,
// unpaused session and never awards XP (W0207) or settles the wave.
//
// Order: population lock (1600), vote open (800) / close (1200) on even waves,
// builder economy and repair, at most one wall upgrade, boss spawn (2600), combat,
// BOSS_INCOMING notification.
func resolveBoundary(state domain.SessionState) (domain.SessionState, []Event, error) {
	if state.Lifecycle != domain.LifecycleRunning || len(state.PauseReasons) > 0 {
		return state, nil, nil
	}
	tick := state.Wave.Tick
	var events []Event
	next := state.Clone()

	if tick == domain.PopulationLockTick && !next.Difficulty.Locked {
		locked, err := lockPopulation(next)
		if err != nil {
			return state, nil, err
		}
		next = locked
	}

	if tick == domain.VoteOpenTick {
		opened, openedEvents, err := openVote(next)
		if err != nil {
			return state, nil, err
		}
		next = opened
		events = append(events, openedEvents...)
	}
	if tick == domain.VoteCloseTick {
		closed, closedEvents, err := closeVote(next)
		if err != nil {
			return state, nil, err
		}
		next = closed
		events = append(events, closedEvents...)
	}

	isBoundary := tick > 0 && tick%200 == 0 && tick <= domain.FinalCombatTick
	if isBoundary && !next.Wave.Defeated {
		resolved, err := resolveContribution(next, &events)
		if err != nil {
			return state, nil, err
		}
		next = resolved
	}

	// The boss spawns after contribution/repair and before the attack stage.
	if tick == domain.BossSpawnTick && domain.IsBossWave(next.Wave.Number) && !next.Wave.Boss.Exists && !next.Wave.Defeated {
		spawned, spawnEvents, err := spawnBoss(next)
		if err != nil {
			return state, nil, err
		}
		next = spawned
		events = append(events, spawnEvents...)
	}

	if isBoundary && tick >= domain.CombatStartTick && !next.Wave.Defeated {
		resolved, err := resolveCombat(next, &events)
		if err != nil {
			return state, nil, err
		}
		next = resolved
	}

	if next.Difficulty.Locked && domain.IsBossWave(next.Wave.Number) && !next.Wave.Boss.Exists &&
		tick < domain.BossSpawnTick && !next.Wave.BossIncoming {
		var ordinary int64
		for _, pool := range next.Wave.Pools {
			ordinary += pool.HP
		}
		if ordinary == 0 {
			next.Wave.BossIncoming = true
			events = append(events, Event{
				Type:   EventBossIncoming,
				WaveNo: next.Wave.Number,
				Tick:   next.Wave.Tick,
				Phase:  next.Wave.Phase,
			})
		}
	}

	if err := next.Validate(); err != nil {
		return state, nil, fmt.Errorf("%w: boundary produced an invalid state: %v", ErrInvalidState, err)
	}
	return next, events, nil
}

// mergeBoundaryEvents composes clock scheduling signals, role acceptances and
// gameplay effects into one deterministic order: scheduling boundaries, role
// acceptances, gameplay effects, the settlement requirement and finally the phase
// notification. A phase change can therefore never appear to precede the final
// combat that belongs to it.
func mergeBoundaryEvents(state domain.SessionState, clockEvents []Event, roleEvents []RoleEvent, gameplayEvents []Event) []Event {
	merged := make([]Event, 0, len(clockEvents)+len(roleEvents)+len(gameplayEvents))
	var settlement []Event
	var phase []Event
	for _, event := range clockEvents {
		switch event.Type {
		case EventSettlementRequired:
			settlement = append(settlement, event)
		case EventPhaseChanged:
			phase = append(phase, event)
		default:
			merged = append(merged, event)
		}
	}
	for _, event := range roleEvents {
		merged = append(merged, Event{
			Type:   EventType(event.Type),
			WaveNo: event.WaveNo,
			Tick:   event.Tick,
			Phase:  state.Wave.Phase,
			Amount: int64(event.Role),
		})
	}
	// Boss spawn signal must sit after the boss_spawn boundary and before the
	// combat boundary that follows it.
	spawnIndex := -1
	for index, event := range merged {
		if event.Type == EventScheduleBoundary && event.Boundary == BoundaryBossSpawn {
			spawnIndex = index + 1
		}
	}
	if spawnIndex >= 0 {
		for _, event := range gameplayEvents {
			if event.Type == EventBossSpawned {
				merged = append(merged, Event{})
				copy(merged[spawnIndex+1:], merged[spawnIndex:])
				merged[spawnIndex] = event
				gameplayEvents = removeEvent(gameplayEvents, EventBossSpawned)
				break
			}
		}
	}
	merged = append(merged, gameplayEvents...)
	merged = append(merged, settlement...)
	merged = append(merged, phase...)
	return merged
}

func removeEvent(events []Event, eventType EventType) []Event {
	result := events[:0]
	for _, event := range events {
		if event.Type != eventType {
			result = append(result, event)
		}
	}
	return result
}
