package simulation

import (
	"errors"
	"fmt"
	"math"
	"sort"

	"github.com/hung98dev/ttgqldl/backend/internal/domain"
)

// Sentinel errors for role processing. A batch is rejected atomically: any
// validation error leaves the caller's state and queue untouched.
var (
	ErrInvalidCommand     = errors.New("simulation: invalid role command")
	ErrDuplicateInputSeq  = errors.New("simulation: duplicate input sequence")
	ErrRetroactiveInput   = errors.New("simulation: role command targets a committed tick")
	ErrSourceDisconnected = errors.New("simulation: source is disconnected")
	ErrStorageUnavailable = errors.New("simulation: storage is unavailable")
	ErrInputDeferred      = errors.New("simulation: input deferred while paused")
	ErrStaleSchedule      = errors.New("simulation: scheduled role command is stale")
	ErrStartMismatch      = errors.New("simulation: start registration does not match the accepted command")
	// ErrStartPending rejects a registration while another wave-start registration
	// is already accepted and the session is still WAITING: a WAITING session owns
	// at most one start registration, and production never queues a second one.
	ErrStartPending = errors.New("simulation: a wave-start registration is already pending")
)

// RoleCommand is one already-canonicalized, authorized role selection. Text
// parsing, Unicode normalization and provider identity mapping belong to M03; the
// reducer accepts only this validated shape and never raw comments.
//
// ScheduledTick is the effective logical tick of the target wave: 0..2999 for a
// current-wave change, or 3000 when the source commits at tick 2999 after the
// wave has ended. In the latter case TargetWave must be current+1 and the
// registration becomes effective at tick 0 of that wave.
type RoleCommand struct {
	InputSeq int64
	ActorKey domain.ActorKey
	Role     domain.Role
	// TargetWave is the wave that owns the effect; ScheduledTick is the effective
	// tick inside that wave. NEXT-wave registrations use ScheduledTick 0 and are
	// accepted only during RESULTS, never by a numeric tick alone.
	TargetWave    int64
	ScheduledTick int
	// AcceptedTick is the committed session tick when the command was accepted.
	// AcceptedTickUnspecified lets a caller mean "the current committed tick"; an
	// explicit mismatch is rejected so a command can never be applied retroactively.
	AcceptedTick int
}

// AcceptedTickUnspecified asks the engine to record the current committed tick.
const AcceptedTickUnspecified = -2

// RoleQueue holds accepted but not yet effective current-wave commands. A future
// command lives here until the logical clock reaches its scheduled tick, so it can
// never mutate an earlier authoritative snapshot.
type RoleQueue struct {
	Entries []RoleCommand
	// Ballots are accepted ballots that have not yet reached their effective tick.
	// They never appear in authoritative ballot state before ApplyDueVotes.
	Ballots []VoteCommand
}

// Len reports the number of accepted, not-yet-applied commands.
func (q RoleQueue) Len() int { return len(q.Entries) }

// Clone returns an independent copy of the queue.
func (q RoleQueue) Clone() RoleQueue {
	clone := RoleQueue{}
	if q.Entries != nil {
		clone.Entries = make([]RoleCommand, len(q.Entries))
		copy(clone.Entries, q.Entries)
	}
	if q.Ballots != nil {
		clone.Ballots = make([]VoteCommand, len(q.Ballots))
		copy(clone.Ballots, q.Ballots)
	}
	return clone
}

func (q RoleQueue) cloneRoles() []RoleCommand {
	if q.Entries == nil {
		return nil
	}
	out := make([]RoleCommand, len(q.Entries))
	copy(out, q.Entries)
	return out
}

func (q RoleQueue) cloneBallots() []VoteCommand {
	if q.Ballots == nil {
		return nil
	}
	out := make([]VoteCommand, len(q.Ballots))
	copy(out, q.Ballots)
	return out
}

// RoleOutcome is the internal processing result for one command.
type RoleOutcome string

const (
	OutcomeScheduled         RoleOutcome = "scheduled"
	OutcomeRegistered        RoleOutcome = "registered"
	OutcomeRoleChanged       RoleOutcome = "role_changed"
	OutcomeDuplicateSameRole RoleOutcome = "duplicate_same_role"
	OutcomeCooldownRejected  RoleOutcome = "cooldown_rejected"
	OutcomeNextWaveQueued    RoleOutcome = "next_wave_queued"
)

// RoleDecision is the internal outcome for one command. It may carry the private
// actor key for durable acknowledgement; it is never serialized into public events.
type RoleDecision struct {
	Outcome      RoleOutcome
	InputSeq     int64
	ActorKey     domain.ActorKey
	Role         domain.Role
	PreviousRole domain.Role
	TargetWave   int64
	Tick         int
}

// RoleEventType is the canonical public domain-event identifier.
type RoleEventType string

const (
	// Canonical names from 03_SIMULATION.md.
	EventRoleAccepted         RoleEventType = "role_accepted"
	EventRoleChanged          RoleEventType = "role_changed"
	EventRoleRejectedCooldown RoleEventType = "role_rejected_cooldown"
	EventNextWaveQueued       RoleEventType = "next_wave_queued"
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

// DeferredError reports that a valid batch was accepted as deferred work but was
// not applied because the simulation is operator-paused. The caller must retain
// the commands (they are returned unchanged) and resubmit them after resume. This
// is deliberately an error, never a success acknowledgement.
type DeferredError struct {
	Commands []RoleCommand
}

func (e *DeferredError) Error() string {
	return fmt.Sprintf("simulation: %d role command(s) deferred while paused", len(e.Commands))
}

// Is lets callers test with errors.Is(err, ErrInputDeferred).
func (e *DeferredError) Is(target error) bool { return target == ErrInputDeferred }

// AcceptRoleCommands validates a batch and schedules it without changing
// current-wave authoritative state. RESULTS commands (tick >= 2800, or a command
// scheduled at 3000 after tick 2999) update only the explicit NEXT-wave pending
// queue, never the current wave. Any invalid command rejects the whole batch.
func AcceptRoleCommands(state domain.SessionState, queue RoleQueue, commands []RoleCommand) (domain.SessionState, RoleQueue, []RoleDecision, error) {
	if err := validateActiveState(state); err != nil {
		return state, queue, nil, fmt.Errorf("%w: %v", ErrInvalidState, err)
	}
	switch state.Lifecycle {
	case domain.LifecycleStopped:
		return state, queue, nil, ErrStopped
	case domain.LifecycleStorageUnavailable:
		// A storage outage must never acknowledge input that cannot be durable.
		return state, queue, nil, ErrStorageUnavailable
	case domain.LifecycleSourceDisconnected:
		// No new source events may be fabricated while the source is down.
		return state, queue, nil, ErrSourceDisconnected
	}

	ordered, err := validateRoleCommands(state, queue, commands)
	if err != nil {
		return state, queue, nil, err
	}
	if state.Lifecycle == domain.LifecyclePaused {
		// Validation already passed; report a typed deferral and return the
		// commands so the caller cannot mistake this for an applied success.
		return state.Clone(), queue.Clone(), nil, &DeferredError{Commands: ordered}
	}

	work := state.Clone()
	decisions := make([]RoleDecision, 0, len(ordered))
	accepted := queue.Clone()
	for _, command := range ordered {
		decision := RoleDecision{
			InputSeq:   command.InputSeq,
			ActorKey:   command.ActorKey,
			Role:       command.Role,
			TargetWave: command.TargetWave,
			Tick:       command.ScheduledTick,
		}
		// Ownership comes from TargetWave, never from the numeric scheduled tick:
		// a command for the current wave effective at 2800 is final-boundary work,
		// while a NEXT-wave command is accepted only during RESULTS.
		if command.TargetWave == work.Wave.Number+1 {
			decision.Outcome = queueNextWave(&work, command)
			decisions = append(decisions, decision)
			continue
		}
		accepted.Entries = append(accepted.Entries, command)
		decision.Outcome = OutcomeScheduled
		decisions = append(decisions, decision)
	}
	sortQueue(accepted)
	if err := work.Validate(); err != nil {
		return state, queue, nil, fmt.Errorf("%w: accepted queue produced an invalid state: %v", ErrInvalidState, err)
	}
	return work, accepted, decisions, nil
}

// ApplyDueRoles applies every queued command whose scheduled tick is at or before
// the committed logical tick, in deterministic (tick, input_seq) order, and
// removes it from the queue. It is the only place where a future command can
// change current-wave membership, so a command scheduled at tick 800 cannot affect
// tick 600.
func ApplyDueRoles(state domain.SessionState, queue RoleQueue) (domain.SessionState, RoleQueue, []RoleEvent, []RoleDecision, error) {
	if err := validateActiveState(state); err != nil {
		return state, queue, nil, nil, fmt.Errorf("%w: %v", ErrInvalidState, err)
	}
	if len(queue.Entries) == 0 {
		return state.Clone(), queue.Clone(), nil, nil, nil
	}
	if state.Lifecycle == domain.LifecyclePaused || len(state.PauseReasons) > 0 {
		// The logical clock is frozen; nothing becomes due.
		return state.Clone(), queue.Clone(), nil, nil, nil
	}
	if state.Lifecycle == domain.LifecycleWaiting {
		return state, queue, nil, nil, fmt.Errorf("%w: cannot apply role commands before the wave starts", ErrInvalidState)
	}

	work := state.Clone()
	remaining := RoleQueue{Ballots: queue.cloneBallots()}
	events := make([]RoleEvent, 0, len(queue.Entries))
	decisions := make([]RoleDecision, 0, len(queue.Entries))
	index := make(map[domain.ActorKey]int, len(work.Members))
	for i, member := range work.Members {
		index[member.ActorKey] = i
	}

	for _, command := range queue.Entries {
		if command.TargetWave != work.Wave.Number {
			return state, queue, nil, nil, fmt.Errorf("%w: command targets wave %d while the session is on %d", ErrStaleSchedule, command.TargetWave, work.Wave.Number)
		}
		if command.ScheduledTick < 0 || command.ScheduledTick > domain.FinalCombatTick {
			return state, queue, nil, nil, fmt.Errorf("%w: queued command tick %d is invalid", ErrStaleSchedule, command.ScheduledTick)
		}
		if command.ScheduledTick > state.Tick() {
			remaining.Entries = append(remaining.Entries, command)
			continue
		}
		decision := RoleDecision{
			InputSeq:   command.InputSeq,
			ActorKey:   command.ActorKey,
			Role:       command.Role,
			TargetWave: command.TargetWave,
			Tick:       command.ScheduledTick,
		}

		position, exists := index[command.ActorKey]
		if !exists {
			work.Members = append(work.Members, domain.WaveMember{
				ActorKey:   command.ActorKey,
				Role:       command.Role,
				JoinedTick: command.ScheduledTick,
			})
			index[command.ActorKey] = len(work.Members) - 1
			events = append(events, RoleEvent{
				Type:       EventRoleAccepted,
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
			// A registration replayed at the exact join tick is the accepted
			// registration itself: it emits role_accepted but changes nothing.
			if member.JoinedTick == command.ScheduledTick && member.LastChangeTick == nil && !member.FirstSwitchUsed {
				events = append(events, RoleEvent{
					Type:       EventRoleAccepted,
					WaveNo:     work.Wave.Number,
					Tick:       command.ScheduledTick,
					Role:       command.Role,
					TargetWave: work.Wave.Number,
				})
			}
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
				return state, queue, nil, nil, fmt.Errorf("%w: cooldown arithmetic: %v", ErrInvalidCommand, overflowErr)
			}
			if elapsed < domain.RoleChangeCooldownTick {
				events = append(events, RoleEvent{
					Type:   EventRoleRejectedCooldown,
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
	sortQueue(remaining)
	if err := work.Validate(); err != nil {
		return state, queue, nil, nil, fmt.Errorf("%w: role application produced an invalid state: %v", ErrInvalidState, err)
	}
	return work, remaining, events, decisions, nil
}

// AdvanceWithRoles is the authoritative boundary orchestration: it advances the
// logical clock, applies every command that becomes effective at the new tick in
// deterministic order, and only then resolves gameplay for that boundary. On any
// failure the original state and queue are returned unchanged.
//
// KindStartWave is authorized only through the accepted queue: the queue must be
// non-empty and its first command must be the tick-0 registration whose actor and
// role exactly match the supplied registration. That command is consumed once.
func AdvanceWithRoles(state domain.SessionState, queue RoleQueue, input Input) (domain.SessionState, RoleQueue, []Event, []RoleEvent, error) {
	if input.Kind == KindStartWave {
		if len(queue.Entries) == 0 {
			return state, queue, nil, nil, fmt.Errorf("%w: start requires an accepted registration", ErrNoRegistration)
		}
		head := queue.Entries[0]
		if head.ScheduledTick != 0 || head.TargetWave != state.Wave.Number {
			return state, queue, nil, nil, fmt.Errorf("%w: queued registration is not a valid wave start", ErrStaleSchedule)
		}
		if input.Registration == nil || input.Registration.ActorKey != head.ActorKey || input.Registration.Role != head.Role {
			return state, queue, nil, nil, fmt.Errorf("%w: registration does not match the accepted command", ErrStartMismatch)
		}
		remaining := RoleQueue{Entries: append([]RoleCommand(nil), queue.Entries[1:]...), Ballots: queue.cloneBallots()}
		next, clockEvents, err := Next(state, Input{
			Kind:         KindStartWave,
			Registration: &Registration{ActorKey: head.ActorKey, Role: head.Role},
		})
		if err != nil {
			return state, queue, nil, nil, err
		}
		applied, leftover, dueEvents, _, applyErr := ApplyDueRoles(next, remaining)
		if applyErr != nil {
			return state, queue, nil, nil, applyErr
		}
		voted, voteQueue, voteErr := ApplyDueVotes(applied, leftover)
		if voteErr != nil {
			return state, queue, nil, nil, voteErr
		}
		resolved, gameplayEvents, resolveErr := resolveBoundary(voted)
		if resolveErr != nil {
			return state, queue, nil, nil, resolveErr
		}
		leftover = voteQueue
		roleEvents := []RoleEvent{{
			Type:       EventRoleAccepted,
			WaveNo:     resolved.Wave.Number,
			Tick:       0,
			Role:       head.Role,
			TargetWave: resolved.Wave.Number,
		}}
		roleEvents = append(roleEvents, dueEvents...)
		events := mergeBoundaryEvents(resolved, clockEvents, roleEvents, gameplayEvents)
		return resolved, leftover, events, roleEvents, nil
	}

	next, clockEvents, err := Next(state, input)
	if err != nil {
		return state, queue, nil, nil, err
	}
	applied, remaining, roleEvents, _, applyErr := ApplyDueRoles(next, queue)
	if applyErr != nil {
		return state, queue, nil, nil, applyErr
	}
	voted, voteQueue, voteErr := ApplyDueVotes(applied, remaining)
	if voteErr != nil {
		return state, queue, nil, nil, voteErr
	}
	resolved, gameplayEvents, resolveErr := resolveBoundary(voted)
	if resolveErr != nil {
		return state, queue, nil, nil, resolveErr
	}
	events := mergeBoundaryEvents(resolved, clockEvents, roleEvents, gameplayEvents)
	return resolved, voteQueue, events, roleEvents, nil
}

// validateRoleCommands validates the batch before any mutation and returns it in
// deterministic input-seq order. Acceptance position (AcceptedTick), target-wave
// ownership and effective tick are validated together, so a numeric tick alone can
// never reserve a future wave.
func validateRoleCommands(state domain.SessionState, queue RoleQueue, commands []RoleCommand) ([]RoleCommand, error) {
	ordered := make([]RoleCommand, len(commands))
	copy(ordered, commands)
	sort.SliceStable(ordered, func(i, j int) bool { return ordered[i].InputSeq < ordered[j].InputSeq })

	seen := make(map[int64]bool, len(ordered))
	waiting := state.Lifecycle == domain.LifecycleWaiting
	first := true
	for index := range ordered {
		command := &ordered[index]
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
		if command.AcceptedTick == AcceptedTickUnspecified {
			command.AcceptedTick = state.Tick()
		}
		if command.AcceptedTick != state.Tick() {
			return nil, fmt.Errorf("%w: command accepted at %d but the committed tick is %d", ErrRetroactiveInput, command.AcceptedTick, state.Tick())
		}
		if command.ScheduledTick < 0 || command.ScheduledTick > domain.FinalCombatTick {
			return nil, fmt.Errorf("%w: scheduled tick %d outside 0..2800", ErrInvalidCommand, command.ScheduledTick)
		}
		if command.TargetWave < 1 {
			return nil, fmt.Errorf("%w: invalid target wave", ErrInvalidCommand)
		}
		// One monotonic input_seq namespace covers queue entries and ballots, for
		// every lifecycle including WAITING. The check reads the accepted queue, not
		// just the current batch, so a second call cannot reuse an accepted sequence.
		for _, queued := range queue.Entries {
			if queued.InputSeq == command.InputSeq {
				return nil, fmt.Errorf("%w: %d", ErrDuplicateInputSeq, command.InputSeq)
			}
		}
		for _, queued := range queue.Ballots {
			if queued.InputSeq == command.InputSeq {
				return nil, fmt.Errorf("%w: %d (already queued as a ballot)", ErrDuplicateInputSeq, command.InputSeq)
			}
		}

		if waiting {
			// A WAITING session owns at most one accepted wave-start registration.
			// Only the first eligible registration starts the wave; a later arrival
			// must be rejected explicitly instead of becoming a competing tick-0
			// start that could overwrite the accepted role.
			if len(queue.Entries) > 0 {
				return nil, fmt.Errorf("%w: %d registration(s) already queued", ErrStartPending, len(queue.Entries))
			}
			if !first {
				return nil, fmt.Errorf("%w: only the first registration may start a wave", ErrInvalidCommand)
			}
			if command.ScheduledTick != 0 || command.TargetWave != state.Wave.Number {
				return nil, fmt.Errorf("%w: the first registration must target wave %d at tick 0", ErrInvalidCommand, state.Wave.Number)
			}
			first = false
			continue
		}
		first = false

		if command.TargetWave == state.Wave.Number+1 {
			// NEXT-wave registration: only during RESULTS, always effective at 0.
			if state.Tick() >= domain.RolloverTick {
				return nil, fmt.Errorf("%w: NEXT-wave registration closed at the tick-3000 control boundary", ErrInvalidCommand)
			}
			if state.Tick() < domain.FinalCombatTick {
				return nil, fmt.Errorf("%w: NEXT-wave registration is only accepted during RESULTS", ErrInvalidCommand)
			}
			if command.ScheduledTick != 0 {
				return nil, fmt.Errorf("%w: NEXT-wave registration is effective at tick 0", ErrInvalidCommand)
			}
			if state.Wave.Number == math.MaxInt64 {
				return nil, fmt.Errorf("%w: wave number overflow", ErrInvalidCommand)
			}
			continue
		}
		if command.TargetWave != state.Wave.Number {
			return nil, fmt.Errorf("%w: command targets wave %d while the session is on %d", ErrInvalidCommand, command.TargetWave, state.Wave.Number)
		}
		// Current-wave work: effective strictly after the committed tick, including
		// the final boundary at 2800 when accepted at 2799.
		if command.ScheduledTick <= state.Tick() {
			return nil, fmt.Errorf("%w: tick %d already committed at %d", ErrRetroactiveInput, command.ScheduledTick, state.Tick())
		}
	}
	return ordered, nil
}
func sortQueue(queue RoleQueue) {
	sort.SliceStable(queue.Entries, func(i, j int) bool {
		if queue.Entries[i].ScheduledTick != queue.Entries[j].ScheduledTick {
			return queue.Entries[i].ScheduledTick < queue.Entries[j].ScheduledTick
		}
		return queue.Entries[i].InputSeq < queue.Entries[j].InputSeq
	})
}

// queueNextWave applies a role choice to the pending NEXT-wave queue, preserving
// the same first-switch and cooldown contract while the current wave finishes.
func queueNextWave(state *domain.SessionState, command RoleCommand) RoleOutcome {
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
			return OutcomeNextWaveQueued
		}
		if command.ScheduledTick-*pending.LastChangeTick < domain.RoleChangeCooldownTick {
			return OutcomeCooldownRejected
		}
		pending.Role = command.Role
		tick := command.ScheduledTick
		pending.LastChangeTick = &tick
		return OutcomeNextWaveQueued
	}
	state.Pending = append(state.Pending, domain.PendingMember{
		ActorKey:   command.ActorKey,
		Role:       command.Role,
		TargetWave: state.Wave.Number + 1,
	})
	return OutcomeNextWaveQueued
}
