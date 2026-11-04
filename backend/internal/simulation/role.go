package simulation

import (
	"errors"
	"fmt"
	"sort"

	"github.com/hung98dev/ttgqldl/backend/internal/domain"
)

// Sentinel errors for role processing. A batch is rejected atomically: any
// validation error leaves the caller's state untouched.
var (
	ErrInvalidCommand     = errors.New("simulation: invalid role command")
	ErrDuplicateInputSeq  = errors.New("simulation: duplicate input sequence")
	ErrRetroactiveInput   = errors.New("simulation: role command targets a committed tick")
	ErrSourceDisconnected = errors.New("simulation: source is disconnected")
	ErrStorageUnavailable = errors.New("simulation: storage is unavailable")
)

// RoleCommand is one already-canonicalized, authorized role selection. Text
// parsing, Unicode normalization and provider identity mapping belong to M03; the
// reducer accepts only this validated shape and never raw comments.
type RoleCommand struct {
	InputSeq      int64
	ActorKey      domain.ActorKey
	Role          domain.Role
	TargetWave    int64
	ScheduledTick int
}

// RoleOutcome is the internal processing result for one command.
type RoleOutcome string

const (
	OutcomeRegistered          RoleOutcome = "registered"
	OutcomeRoleChanged         RoleOutcome = "role_changed"
	OutcomeDuplicateSameRole   RoleOutcome = "duplicate_same_role"
	OutcomeCooldownRejected    RoleOutcome = "cooldown_rejected"
	OutcomeNextWaveQueued      RoleOutcome = "next_wave_queued"
	OutcomeDeferredWhilePaused RoleOutcome = "deferred_while_paused"
)

// RoleDecision is the internal outcome for one command. It may carry the private
// actor key for durable acknowledgement; it is never serialized into public
// events.
type RoleDecision struct {
	Outcome      RoleOutcome
	InputSeq     int64
	ActorKey     domain.ActorKey
	Role         domain.Role
	PreviousRole domain.Role
	TargetWave   int64
	Tick         int
}

// RoleEventType is the stable public signal name.
type RoleEventType string

const (
	EventRoleRegistered       RoleEventType = "role_registered"
	EventRoleChanged          RoleEventType = "role_changed"
	EventNextWaveQueued       RoleEventType = "next_wave_queued"
	EventRoleCooldownRejected RoleEventType = "role_cooldown_rejected"
)

// RoleEvent is public-safe: it never contains an actor key, provider identifier,
// comment or alias.
type RoleEvent struct {
	Type         RoleEventType `json:"type"`
	WaveNo       int64         `json:"wave_no"`
	Tick         int           `json:"tick"`
	Role         domain.Role   `json:"role"`
	PreviousRole domain.Role   `json:"previous_role,omitempty"`
	TargetWave   int64         `json:"target_wave,omitempty"`
}

// Counts are the active participant counts per role.
type Counts struct {
	Builder int
	Guard   int
	Fighter int
}

// Total is the number of active participants.
func (c Counts) Total() int { return c.Builder + c.Guard + c.Fighter }

// IsContributionBoundary reports the 20-second resolution boundaries.
func IsContributionBoundary(tick int) bool {
	return tick > 0 && tick <= domain.FinalCombatTick && tick%200 == 0
}

// ActiveCounts counts the members whose registration is effective at boundaryTick.
// A member joining at tick 200 contributes at tick 200; one joining at 201 does
// not. Roles are read from the state produced by the applied commands, so the
// latest accepted role in effect at that tick is what counts.
func ActiveCounts(state domain.SessionState, boundaryTick int) Counts {
	counts := Counts{}
	for _, member := range state.Members {
		if member.JoinedTick > boundaryTick {
			continue
		}
		switch member.Role {
		case domain.RoleBuilder:
			counts.Builder++
		case domain.RoleGuard:
			counts.Guard++
		case domain.RoleFighter:
			counts.Fighter++
		}
	}
	return counts
}

// ApplyRoleCommands applies a batch of canonical role commands in deterministic
// input-seq order and returns the new state, public events and per-command
// decisions. The input state is never mutated; any validation error rejects the
// whole batch without partial effects.
func ApplyRoleCommands(state domain.SessionState, commands []RoleCommand) (domain.SessionState, []RoleEvent, []RoleDecision, error) {
	if err := state.Validate(); err != nil {
		return state, nil, nil, fmt.Errorf("%w: %v", ErrInvalidState, err)
	}
	switch state.Lifecycle {
	case domain.LifecycleStopped:
		return state, nil, nil, ErrStopped
	case domain.LifecycleStorageUnavailable:
		// A storage outage must never acknowledge input that cannot be durable.
		return state, nil, nil, ErrStorageUnavailable
	case domain.LifecycleSourceDisconnected:
		// No new source events may be fabricated while the source is down.
		return state, nil, nil, ErrSourceDisconnected
	}
	if state.Lifecycle == domain.LifecyclePaused {
		// Operator pause freezes logical time: commands are acknowledged as
		// deferred and must be re-submitted after resume. Nothing is applied.
		decisions := make([]RoleDecision, 0, len(commands))
		for _, command := range commands {
			decisions = append(decisions, RoleDecision{
				Outcome:  OutcomeDeferredWhilePaused,
				InputSeq: command.InputSeq,
				ActorKey: command.ActorKey,
				Role:     command.Role,
				Tick:     state.Tick(),
			})
		}
		return state.Clone(), nil, decisions, nil
	}

	ordered, err := validateRoleCommands(state, commands)
	if err != nil {
		return state, nil, nil, err
	}

	work := state.Clone()
	events := make([]RoleEvent, 0, len(ordered))
	decisions := make([]RoleDecision, 0, len(ordered))
	index := make(map[domain.ActorKey]int, len(work.Members))
	for i, member := range work.Members {
		index[member.ActorKey] = i
	}
	started := work.Lifecycle != domain.LifecycleWaiting

	for _, command := range ordered {
		decision := RoleDecision{
			InputSeq:   command.InputSeq,
			ActorKey:   command.ActorKey,
			Role:       command.Role,
			TargetWave: command.TargetWave,
			Tick:       command.ScheduledTick,
		}

		// Wave start: the first accepted registration is the only way out of
		// WAITING, and it reuses the W0202 transition rather than a second clock.
		if !started {
			nextState, startEvents, startErr := startWave(work, Input{
				Kind:         KindStartWave,
				Registration: &Registration{ActorKey: command.ActorKey, Role: command.Role},
			})
			if startErr != nil {
				return state, nil, nil, startErr
			}
			for _, event := range startEvents {
				if event.Type == EventWaveStarted {
					events = append(events, RoleEvent{
						Type:       EventRoleRegistered,
						WaveNo:     nextState.Wave.Number,
						Tick:       0,
						Role:       command.Role,
						TargetWave: nextState.Wave.Number,
					})
				}
			}
			work = nextState
			started = true
			index[command.ActorKey] = 0
			decision.Outcome = OutcomeRegistered
			decision.Tick = 0
			decisions = append(decisions, decision)
			continue
		}

		// RESULTS queues the NEXT wave and must never touch current membership.
		if command.ScheduledTick >= domain.FinalCombatTick {
			outcome := queueNextWave(&work, command, &events)
			decision.Outcome = outcome
			decisions = append(decisions, decision)
			continue
		}

		position, exists := index[command.ActorKey]
		if !exists {
			work.Members = append(work.Members, domain.WaveMember{
				ActorKey:   command.ActorKey,
				Role:       command.Role,
				JoinedTick: command.ScheduledTick,
			})
			domain.SortWaveMembers(work.Members)
			index = make(map[domain.ActorKey]int, len(work.Members))
			for i, member := range work.Members {
				index[member.ActorKey] = i
			}
			events = append(events, RoleEvent{
				Type:       EventRoleRegistered,
				WaveNo:     work.Wave.Number,
				Tick:       command.ScheduledTick,
				Role:       command.Role,
				TargetWave: work.Wave.Number,
			})
			decision.Outcome = OutcomeRegistered
			decisions = append(decisions, decision)
			continue
		}

		member := &work.Members[position]
		switch {
		case member.Role == command.Role:
			decision.Outcome = OutcomeDuplicateSameRole
		case !member.FirstSwitchUsed:
			decision.PreviousRole = member.Role
			member.Role = command.Role
			member.FirstSwitchUsed = true
			tick := command.ScheduledTick
			member.LastChangeTick = &tick
			events = append(events, RoleEvent{
				Type:         EventRoleChanged,
				WaveNo:       work.Wave.Number,
				Tick:         command.ScheduledTick,
				Role:         command.Role,
				PreviousRole: decision.PreviousRole,
			})
			decision.Outcome = OutcomeRoleChanged
		default:
			elapsed, overflowErr := domain.SubInt64(int64(command.ScheduledTick), int64(*member.LastChangeTick))
			if overflowErr != nil {
				return state, nil, nil, fmt.Errorf("%w: cooldown arithmetic: %v", ErrInvalidCommand, overflowErr)
			}
			if elapsed < domain.RoleChangeCooldownTick {
				decision.PreviousRole = member.Role
				events = append(events, RoleEvent{
					Type:   EventRoleCooldownRejected,
					WaveNo: work.Wave.Number,
					Tick:   command.ScheduledTick,
					Role:   member.Role,
				})
				decision.Outcome = OutcomeCooldownRejected
				break
			}
			decision.PreviousRole = member.Role
			member.Role = command.Role
			tick := command.ScheduledTick
			member.LastChangeTick = &tick
			events = append(events, RoleEvent{
				Type:         EventRoleChanged,
				WaveNo:       work.Wave.Number,
				Tick:         command.ScheduledTick,
				Role:         command.Role,
				PreviousRole: decision.PreviousRole,
			})
			decision.Outcome = OutcomeRoleChanged
		}
		decisions = append(decisions, decision)
	}

	domain.SortWaveMembers(work.Members)
	domain.SortPendingMembers(work.Pending)
	if err := work.Validate(); err != nil {
		return state, nil, nil, fmt.Errorf("%w: role batch produced an invalid state: %v", ErrInvalidState, err)
	}
	return work, events, decisions, nil
}

// validateRoleCommands validates the whole batch before any state is touched and
// returns it in deterministic input-seq order.
func validateRoleCommands(state domain.SessionState, commands []RoleCommand) ([]RoleCommand, error) {
	ordered := make([]RoleCommand, len(commands))
	copy(ordered, commands)
	sort.SliceStable(ordered, func(i, j int) bool { return ordered[i].InputSeq < ordered[j].InputSeq })

	seen := make(map[int64]bool, len(ordered))
	waiting := state.Lifecycle == domain.LifecycleWaiting
	first := true
	for _, command := range ordered {
		if command.InputSeq <= 0 {
			return nil, fmt.Errorf("%w: input_seq must be positive", ErrInvalidCommand)
		}
		if seen[command.InputSeq] {
			return nil, fmt.Errorf("%w: %d", ErrDuplicateInputSeq, command.InputSeq)
		}
		seen[command.InputSeq] = true
		if !command.ActorKey.Valid() {
			return nil, fmt.Errorf("%w: invalid actor key", ErrInvalidCommand)
		}
		if !command.Role.Valid() {
			return nil, fmt.Errorf("%w: invalid role", ErrInvalidCommand)
		}
		if command.ScheduledTick < 0 || command.ScheduledTick >= domain.WaveTicks {
			return nil, fmt.Errorf("%w: scheduled tick %d outside the wave", ErrInvalidCommand, command.ScheduledTick)
		}
		if waiting && first {
			// The first registration starts wave 1 at tick 0.
			if command.ScheduledTick != 0 {
				return nil, fmt.Errorf("%w: the first registration must be effective at tick 0", ErrInvalidCommand)
			}
			if command.TargetWave != state.Wave.Number {
				return nil, fmt.Errorf("%w: first registration targets wave %d", ErrInvalidCommand, command.TargetWave)
			}
			first = false
			continue
		}
		first = false
		if command.ScheduledTick <= state.Tick() {
			return nil, fmt.Errorf("%w: tick %d already committed at %d", ErrRetroactiveInput, command.ScheduledTick, state.Tick())
		}
		if command.ScheduledTick >= domain.FinalCombatTick {
			// RESULTS (2800..2999) queues the NEXT wave.
			if command.TargetWave != state.Wave.Number+1 {
				return nil, fmt.Errorf("%w: RESULTS registration must target wave %d", ErrInvalidCommand, state.Wave.Number+1)
			}
			continue
		}
		if command.TargetWave != state.Wave.Number {
			return nil, fmt.Errorf("%w: command targets wave %d while the session is on %d", ErrInvalidCommand, command.TargetWave, state.Wave.Number)
		}
	}
	return ordered, nil
}

// queueNextWave applies a role choice to the pending NEXT-wave queue, preserving
// the same first-switch and cooldown contract while the current wave finishes.
func queueNextWave(state *domain.SessionState, command RoleCommand, events *[]RoleEvent) RoleOutcome {
	for i := range state.Pending {
		pending := &state.Pending[i]
		if pending.ActorKey != command.ActorKey {
			continue
		}
		if pending.Role == command.Role {
			return OutcomeDuplicateSameRole
		}
		if !pending.FirstSwitchUsed {
			pending.Role = command.Role
			pending.FirstSwitchUsed = true
			tick := command.ScheduledTick
			pending.LastChangeTick = &tick
			*events = append(*events, RoleEvent{
				Type:       EventNextWaveQueued,
				WaveNo:     pending.TargetWave,
				Tick:       command.ScheduledTick,
				Role:       command.Role,
				TargetWave: pending.TargetWave,
			})
			return OutcomeNextWaveQueued
		}
		if command.ScheduledTick-*pending.LastChangeTick < domain.RoleChangeCooldownTick {
			*events = append(*events, RoleEvent{
				Type:   EventRoleCooldownRejected,
				WaveNo: pending.TargetWave,
				Tick:   command.ScheduledTick,
				Role:   pending.Role,
			})
			return OutcomeCooldownRejected
		}
		pending.Role = command.Role
		tick := command.ScheduledTick
		pending.LastChangeTick = &tick
		*events = append(*events, RoleEvent{
			Type:       EventNextWaveQueued,
			WaveNo:     pending.TargetWave,
			Tick:       command.ScheduledTick,
			Role:       command.Role,
			TargetWave: pending.TargetWave,
		})
		return OutcomeNextWaveQueued
	}

	state.Pending = append(state.Pending, domain.PendingMember{
		ActorKey:   command.ActorKey,
		Role:       command.Role,
		TargetWave: state.Wave.Number + 1,
	})
	*events = append(*events, RoleEvent{
		Type:       EventNextWaveQueued,
		WaveNo:     state.Wave.Number + 1,
		Tick:       command.ScheduledTick,
		Role:       command.Role,
		TargetWave: state.Wave.Number + 1,
	})
	return OutcomeNextWaveQueued
}
