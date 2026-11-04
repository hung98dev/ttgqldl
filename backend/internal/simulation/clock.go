package simulation

import (
	"fmt"
	"math"

	"github.com/hung98dev/ttgqldl/backend/internal/domain"
)

// startWave performs the WAITING -> RUNNING transition. It requires one explicit,
// already-validated accepted registration: an operator START alone, a timer, or a
// fabricated participant can never start a wave.
func startWave(state domain.SessionState, input Input) (domain.SessionState, []Event, error) {
	if len(state.PauseReasons) > 0 {
		return state, nil, fmt.Errorf("%w: a paused session cannot start a wave", ErrTickFrozen)
	}
	if state.Lifecycle != domain.LifecycleWaiting || tickOf(state) != domain.WaitingTick {
		return state, nil, fmt.Errorf("%w: wave already started (lifecycle %s, tick %d)",
			ErrInvalidInput, state.Lifecycle, tickOf(state))
	}
	if len(state.Members) != 0 {
		return state, nil, fmt.Errorf("%w: waiting state must not carry members", ErrInvalidState)
	}
	if input.Registration == nil {
		return state, nil, fmt.Errorf("%w: no registration supplied", ErrNoRegistration)
	}
	registration := *input.Registration
	if !registration.ActorKey.Valid() {
		return state, nil, fmt.Errorf("%w: invalid actor key", ErrInvalidInput)
	}
	if !registration.Role.Valid() {
		return state, nil, fmt.Errorf("%w: invalid role", ErrInvalidInput)
	}

	next := state.Clone()
	next.Lifecycle = domain.LifecycleRunning
	next.Wave.Tick = 0
	next.Wave.Phase = domain.PhasePrep
	next.Members = []domain.WaveMember{{
		ActorKey:   registration.ActorKey,
		Role:       registration.Role,
		JoinedTick: 0,
	}}

	events := []Event{{
		Type:   EventWaveStarted,
		WaveNo: next.Wave.Number,
		Tick:   next.Wave.Tick,
		Phase:  next.Wave.Phase,
	}}
	return commit(state, next, events)
}

// advanceTick advances the logical clock by exactly one tick. Tick 0 is the start
// tick and performs no gameplay work; ticks 1..2999 are ordinary logical ticks;
// tick 3000 is a control boundary that only a rollover may consume.
func advanceTick(state domain.SessionState, _ Input) (domain.SessionState, []Event, error) {
	switch state.Lifecycle {
	case domain.LifecycleStopped:
		return state, nil, ErrStopped
	case domain.LifecycleWaiting:
		return state, nil, fmt.Errorf("%w: the wave has not started", ErrNotRunning)
	}
	if len(state.PauseReasons) > 0 || state.Lifecycle != domain.LifecycleRunning {
		return state, nil, fmt.Errorf("%w: lifecycle %s with %d pause reason(s)", ErrTickFrozen, state.Lifecycle, len(state.PauseReasons))
	}

	current := tickOf(state)
	if current == domain.RolloverTick {
		return state, nil, fmt.Errorf("%w: roll the wave over instead of advancing", ErrControlBoundary)
	}
	if current < 0 || current >= domain.WaveTicks {
		return state, nil, fmt.Errorf("%w: tick %d cannot advance", ErrInvalidState, current)
	}

	next := state.Clone()
	next.Wave.Tick = current + 1

	// Tick 3000 is never a gameplay tick: it is the single control transition.
	if next.Wave.Tick == domain.RolloverTick {
		next.Wave.Phase = domain.PhaseResults
		events := []Event{{
			Type:     EventScheduleBoundary,
			WaveNo:   next.Wave.Number,
			Tick:     next.Wave.Tick,
			Phase:    next.Wave.Phase,
			Boundary: BoundaryRollover,
		}}
		return commit(state, next, events)
	}

	phase, err := domain.PhaseForTick(next.Wave.Tick)
	if err != nil {
		return state, nil, fmt.Errorf("%w: %v", ErrInvalidState, err)
	}
	phaseChanged := phase != next.Wave.Phase
	next.Wave.Phase = phase

	// W0204 effects run inside the winning tick, before any later stage: lock the
	// population and derive the enemy pools at 1600, then spawn the boss at 2600.
	var bossSpawnEvent *Event
	if next.Wave.Tick == domain.PopulationLockTick && !next.Difficulty.Locked {
		locked, lockErr := lockPopulation(next)
		if lockErr != nil {
			return state, nil, lockErr
		}
		next = locked
	}
	if next.Wave.Tick == domain.BossSpawnTick && domain.IsBossWave(next.Wave.Number) && !next.Wave.Boss.Exists {
		spawned, spawnEvents, spawnErr := spawnBoss(next)
		if spawnErr != nil {
			return state, nil, spawnErr
		}
		next = spawned
		if len(spawnEvents) == 1 {
			bossSpawnEvent = &spawnEvents[0]
		}
	}
	// W0205: contribution and combat run on 200-tick boundaries in the canonical
	// order (economy, repair, upgrade, attack, retaliation, guard, damage).
	var combatEvents []Event
	if next.Wave.Tick > 0 && next.Wave.Tick%200 == 0 && next.Wave.Tick <= domain.FinalCombatTick {
		if !next.Wave.Defeated && next.Difficulty.Locked {
			resolved, resolveErr := resolveContribution(next, &combatEvents)
			if resolveErr != nil {
				return state, nil, resolveErr
			}
			next = resolved
			if next.Wave.Tick >= domain.CombatStartTick {
				resolvedCombat, combatErr := resolveCombat(next, &combatEvents)
				if combatErr != nil {
					return state, nil, combatErr
				}
				next = resolvedCombat
			}
		}
	}

	bossIncoming := false
	if next.Difficulty.Locked && domain.IsBossWave(next.Wave.Number) && !next.Wave.Boss.Exists &&
		next.Wave.Tick < domain.BossSpawnTick && !next.Wave.BossIncoming {
		var ordinary int64
		for _, pool := range next.Wave.Pools {
			ordinary += pool.HP
		}
		if ordinary == 0 {
			next.Wave.BossIncoming = true
			bossIncoming = true
		}
	}

	// Deterministic event order: scheduling boundaries in canonical order, then
	// the settlement requirement, then the phase notification. A later reducer can
	// therefore never observe RESULTS before the settlement it belongs to.
	events := make([]Event, 0, 5)
	final := next.Wave.Tick == domain.FinalCombatTick
	bossWave := domain.IsBossWave(next.Wave.Number)
	for _, boundary := range BoundariesAt(next.Wave.Tick, bossWave) {
		events = append(events, Event{
			Type:     EventScheduleBoundary,
			WaveNo:   next.Wave.Number,
			Tick:     next.Wave.Tick,
			Phase:    phase,
			Boundary: boundary,
			Stage:    stageForBoundary(boundary, final),
		})
		// The spawn signal belongs after the boss_spawn boundary and before the
		// combat boundary that follows it.
		if boundary == BoundaryBossSpawn && bossSpawnEvent != nil {
			events = append(events, *bossSpawnEvent)
		}
	}
	events = append(events, combatEvents...)
	if bossIncoming {
		events = append(events, Event{
			Type:   EventBossIncoming,
			WaveNo: next.Wave.Number,
			Tick:   next.Wave.Tick,
			Phase:  phase,
		})
	}
	// W0202 cannot settle a wave: the final boundary is reported as required work
	// so W0205/W0207 can run the stages in order before RESULTS is treated as
	// settled. The wave result deliberately stays pending.
	if final {
		events = append(events, Event{
			Type:   EventSettlementRequired,
			WaveNo: next.Wave.Number,
			Tick:   next.Wave.Tick,
			Phase:  phase,
			Stage:  StageSettlement,
		})
	}
	if phaseChanged {
		events = append(events, Event{
			Type:   EventPhaseChanged,
			WaveNo: next.Wave.Number,
			Tick:   next.Wave.Tick,
			Phase:  phase,
			Stage:  stageForPhaseChange(final),
		})
	}
	return commit(state, next, events)
}

// stageForBoundary labels a scheduling boundary with the stage that must run it.
func stageForBoundary(boundary Boundary, final bool) Stage {
	switch boundary {
	case BoundaryRoleResolution:
		if final {
			return StageFinalContribution
		}
		return StageConsumeInputs
	case BoundaryCombat:
		return StageConsumeInputs
	case BoundaryBossSpawn:
		return StageConsumeInputs
	case BoundaryFinalCombat:
		return StageFinalCombat
	default:
		return ""
	}
}

// stageForPhaseChange labels the phase notification at the final boundary.
func stageForPhaseChange(final bool) Stage {
	if final {
		return StageEnterResults
	}
	return ""
}

// rollover performs the single tick-3000 control transition: one wave increment,
// previous-wave gameplay state reset, pending registrations promoted, and either a
// running next wave at tick 0 or a return to WAITING at tick -1.
func rollover(state domain.SessionState, _ Input) (domain.SessionState, []Event, error) {
	if state.Lifecycle == domain.LifecycleStopped {
		return state, nil, ErrStopped
	}
	if tickOf(state) != domain.RolloverTick {
		return state, nil, fmt.Errorf("%w: state is at tick %d", ErrNotAtRollover, tickOf(state))
	}
	// Fail closed: an unsettled wave must never be rolled over. Settlement itself
	// belongs to W0207; W0202 only requires that it already happened.
	if state.Wave.Result == domain.WaveResultPending {
		return state, nil, fmt.Errorf("%w: wave %d has no completed settlement", ErrUnsettledWave, state.Wave.Number)
	}
	if state.Wave.Number == math.MaxInt64 {
		return state, nil, fmt.Errorf("%w: wave number overflow", ErrOverflow)
	}
	if len(state.PauseReasons) > 0 {
		return state, nil, fmt.Errorf("%w: rollover requires clear pause reasons", ErrTickFrozen)
	}

	next := state.Clone()
	next.Wave.Number = state.Wave.Number + 1
	next.Wave.Pools = []domain.EnemyPool{}
	next.Wave.Boss = domain.BossPool{}
	next.Wave.Vote = domain.VoteState{}
	next.Wave.Ballots = []domain.Ballot{}
	next.Wave.Defeated = false
	next.Wave.BossIncoming = false
	next.Wave.Result = domain.WaveResultPending
	// Per-wave enemy state and derived difficulty reset; the next wave derives its
	// own lock at 1600 from persistent progress.
	next.Difficulty = domain.Difficulty{Level: 1}

	promoted := make([]domain.WaveMember, 0, len(state.Pending))
	for _, pending := range state.Pending {
		promoted = append(promoted, domain.WaveMember{
			ActorKey:   pending.ActorKey,
			Role:       pending.Role,
			JoinedTick: 0,
		})
	}
	domain.SortWaveMembers(promoted)
	next.Members = promoted
	next.Pending = []domain.PendingMember{}

	switch {
	case len(promoted) > 0:
		// Pending registrations are promoted and the next wave starts at tick 0.
		// If a pause is still active the wave is started but its clock stays frozen.
		next.Wave.Tick = 0
		next.Wave.Phase = domain.PhasePrep
		if len(state.PauseReasons) > 0 {
			next.Lifecycle = lifecycleFor(state.PauseReasons)
		} else {
			next.Lifecycle = domain.LifecycleRunning
		}
	case state.Lifecycle == domain.LifecyclePaused:
		// A paused empty queue returns to WAITING but keeps its pause reasons.
		next.Wave.Tick = domain.WaitingTick
		next.Wave.Phase = domain.PhaseWaiting
		next.Lifecycle = domain.LifecyclePaused
	default:
		next.Wave.Tick = domain.WaitingTick
		next.Wave.Phase = domain.PhaseWaiting
		next.Lifecycle = domain.LifecycleWaiting
	}

	events := []Event{{
		Type:   EventRolledOver,
		WaveNo: next.Wave.Number,
		Tick:   next.Wave.Tick,
		Phase:  next.Wave.Phase,
	}}
	if next.Lifecycle == domain.LifecycleRunning {
		events = append(events, Event{
			Type:   EventWaveStarted,
			WaveNo: next.Wave.Number,
			Tick:   next.Wave.Tick,
			Phase:  next.Wave.Phase,
		})
	}
	return commit(state, next, events)
}
