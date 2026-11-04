package simulation

import (
	"fmt"
	"sort"

	"github.com/hung98dev/ttgqldl/backend/internal/domain"
)

// VoteCommand is one already-canonicalized ballot. Chat parsing and identity
// mapping belong to M03; this layer accepts only validated domain values.
//
// ScheduledTick is the effective tick inside the current wave and must match the
// ingest contract committed_tick+1, so a caller can never rewrite a committed
// boundary. AcceptedTick is the committed tick at acceptance time.
type VoteCommand struct {
	InputSeq      int64
	ActorKey      domain.ActorKey
	TargetWave    int64
	AcceptedTick  int
	ScheduledTick int
	Option        domain.VoteOption
}

// VoteDecision is the internal outcome for one ballot command.
type VoteDecision struct {
	Outcome  string
	InputSeq int64
	ActorKey domain.ActorKey
	Option   domain.VoteOption
	Tick     int
}

// VoteDeferredError reports validated ballots that were not queued because the
// session is operator-paused. The caller must retain and resubmit them.
type VoteDeferredError struct {
	Commands []VoteCommand
}

func (e *VoteDeferredError) Error() string {
	return fmt.Sprintf("simulation: %d ballot(s) deferred while paused", len(e.Commands))
}

func (e *VoteDeferredError) Is(target error) bool { return target == ErrInputDeferred }

// AcceptVoteCommands validates a batch and queues it without touching
// authoritative ballot state. A ballot becomes effective only through
// ApplyDueVotes at its scheduled tick.
func AcceptVoteCommands(state domain.SessionState, queue RoleQueue, commands []VoteCommand) (domain.SessionState, RoleQueue, []VoteDecision, error) {
	if err := validateActiveState(state); err != nil {
		return state, queue, nil, fmt.Errorf("%w: %v", ErrInvalidState, err)
	}
	switch state.Lifecycle {
	case domain.LifecycleStopped:
		return state, queue, nil, ErrStopped
	case domain.LifecycleStorageUnavailable:
		return state, queue, nil, ErrStorageUnavailable
	case domain.LifecycleSourceDisconnected:
		return state, queue, nil, ErrSourceDisconnected
	}
	ordered := make([]VoteCommand, len(commands))
	copy(ordered, commands)
	sort.SliceStable(ordered, func(i, j int) bool { return ordered[i].InputSeq < ordered[j].InputSeq })

	members := make(map[domain.ActorKey]int, len(state.Members))
	for _, member := range state.Members {
		members[member.ActorKey] = member.JoinedTick
	}
	seen := make(map[int64]bool, len(ordered))
	if !domain.VoteAvailable(state.Wave.Number) {
		return state, queue, nil, fmt.Errorf("%w: voting is only available on even waves", ErrInvalidCommand)
	}
	if state.Tick() >= domain.VoteCloseTick {
		return state, queue, nil, fmt.Errorf("%w: voting closed at tick %d", ErrInvalidCommand, domain.VoteCloseTick)
	}
	for index := range ordered {
		command := &ordered[index]
		if command.InputSeq <= 0 {
			return state, queue, nil, fmt.Errorf("%w: input_seq must be positive", ErrInvalidCommand)
		}
		if seen[command.InputSeq] {
			return state, queue, nil, fmt.Errorf("%w: %d", ErrDuplicateInputSeq, command.InputSeq)
		}
		seen[command.InputSeq] = true
		for _, queued := range queue.Ballots {
			if queued.InputSeq == command.InputSeq {
				return state, queue, nil, fmt.Errorf("%w: %d", ErrDuplicateInputSeq, command.InputSeq)
			}
		}
		for _, queued := range queue.Entries {
			if queued.InputSeq == command.InputSeq {
				return state, queue, nil, fmt.Errorf("%w: %d (already queued as a role command)", ErrDuplicateInputSeq, command.InputSeq)
			}
		}
		if !command.ActorKey.Valid() {
			return state, queue, nil, fmt.Errorf("%w: invalid actor key", ErrInvalidCommand)
		}
		if !command.Option.Valid() || command.Option == domain.VoteOptionNone {
			return state, queue, nil, fmt.Errorf("%w: invalid ballot option", ErrInvalidCommand)
		}
		if command.AcceptedTick == AcceptedTickUnspecified {
			command.AcceptedTick = state.Tick()
		}
		if command.AcceptedTick != state.Tick() {
			return state, queue, nil, fmt.Errorf("%w: ballot accepted at %d but committed tick is %d", ErrRetroactiveInput, command.AcceptedTick, state.Tick())
		}
		if command.TargetWave != state.Wave.Number {
			return state, queue, nil, fmt.Errorf("%w: ballot targets wave %d while the session is on %d", ErrInvalidCommand, command.TargetWave, state.Wave.Number)
		}
		if !domain.BallotTickEligible(command.ScheduledTick) {
			return state, queue, nil, fmt.Errorf("%w: ballot tick %d outside [800,1200)", ErrInvalidCommand, command.ScheduledTick)
		}
		// Ingest schedules at last committed tick + 1; a caller cannot choose an
		// arbitrary future tick and cannot rewrite a committed one.
		if command.ScheduledTick != state.Tick()+1 {
			return state, queue, nil, fmt.Errorf("%w: ballot tick %d must equal committed+1 (%d)", ErrRetroactiveInput, command.ScheduledTick, state.Tick()+1)
		}
		joined, isMember := members[command.ActorKey]
		if !isMember {
			return state, queue, nil, fmt.Errorf("%w: only current-wave participants may vote", ErrInvalidCommand)
		}
		if joined > command.ScheduledTick {
			return state, queue, nil, fmt.Errorf("%w: participant is not effective at the ballot tick", ErrInvalidCommand)
		}
	}
	if state.Lifecycle == domain.LifecyclePaused {
		return state.Clone(), queue.Clone(), nil, &VoteDeferredError{Commands: ordered}
	}

	decisions := make([]VoteDecision, 0, len(ordered))
	next := queue.Clone()
	for _, command := range ordered {
		next.Ballots = append(next.Ballots, command)
		decisions = append(decisions, VoteDecision{
			Outcome:  "ballot_queued",
			InputSeq: command.InputSeq,
			ActorKey: command.ActorKey,
			Option:   command.Option,
			Tick:     command.ScheduledTick,
		})
	}
	sortBallotQueue(next.Ballots)
	return state.Clone(), next, decisions, nil
}

// ApplyDueVotes applies every queued ballot whose effective tick has committed,
// in deterministic (tick, input_seq) order, and stores the latest ballot per
// actor. It never mutates the authoritative ballot collection before that tick.
func ApplyDueVotes(state domain.SessionState, queue RoleQueue) (domain.SessionState, RoleQueue, error) {
	if len(queue.Ballots) == 0 {
		return state, queue, nil
	}
	// Ballots only become due for an active canonical revision. The check runs only
	// when ballots are actually queued, so the quiet-tick path pays nothing.
	if err := validateActiveState(state); err != nil {
		return state, queue, fmt.Errorf("%w: %v", ErrInvalidState, err)
	}
	if state.Lifecycle != domain.LifecycleRunning || len(state.PauseReasons) > 0 {
		return state, queue, nil
	}
	work := state.Clone()
	remaining := queue.Clone()
	remaining.Ballots = nil
	for _, command := range queue.Ballots {
		if command.ScheduledTick > state.Tick() {
			remaining.Ballots = append(remaining.Ballots, command)
			continue
		}
		if command.TargetWave != work.Wave.Number {
			return state, queue, fmt.Errorf("%w: ballot targets wave %d while the session is on %d", ErrStaleSchedule, command.TargetWave, work.Wave.Number)
		}
		if command.ScheduledTick > work.Tick() {
			// Cannot happen because the queue is ordered, but keep the invariant.
			remaining.Ballots = append(remaining.Ballots, command)
			continue
		}
		work.Wave.Ballots = upsertBallot(work.Wave.Ballots, domain.Ballot{
			ActorKey: command.ActorKey,
			Option:   command.Option,
			Tick:     command.ScheduledTick,
			InputSeq: command.InputSeq,
		})
	}
	sortBallotQueue(remaining.Ballots)
	domain.SortBallots(work.Wave.Ballots)
	if err := work.Validate(); err != nil {
		return state, queue, fmt.Errorf("%w: due ballots produced an invalid state: %v", ErrInvalidState, err)
	}
	return work, remaining, nil
}

// upsertBallot keeps one ballot per actor with deterministic latest-wins rules.
func upsertBallot(ballots []domain.Ballot, ballot domain.Ballot) []domain.Ballot {
	for i := range ballots {
		if ballots[i].ActorKey != ballot.ActorKey {
			continue
		}
		if ballot.Tick > ballots[i].Tick || (ballot.Tick == ballots[i].Tick && ballot.InputSeq > ballots[i].InputSeq) {
			ballots[i] = ballot
		}
		return ballots
	}
	return append(ballots, ballot)
}

func sortBallotQueue(commands []VoteCommand) {
	sort.SliceStable(commands, func(i, j int) bool {
		if commands[i].ScheduledTick != commands[j].ScheduledTick {
			return commands[i].ScheduledTick < commands[j].ScheduledTick
		}
		return commands[i].InputSeq < commands[j].InputSeq
	})
}

// openVote opens the ballot window exactly once on an eligible wave.
func openVote(state domain.SessionState) (domain.SessionState, []Event, error) {
	if !domain.VoteAvailable(state.Wave.Number) || state.Wave.Vote.Open || state.Wave.Vote.Closed {
		return state, nil, nil
	}
	next := state.Clone()
	next.Wave.Vote = domain.VoteState{Open: true, Selected: domain.VoteOptionNone}
	return next, []Event{{
		Type:   EventVoteOpened,
		WaveNo: next.Wave.Number,
		Tick:   next.Wave.Tick,
		Phase:  next.Wave.Phase,
	}}, nil
}

// closeVote tallies the applied ballots, selects the option, deducts its cost
// exactly once and closes the window. It must run before tick-1200 economy.
func closeVote(state domain.SessionState) (domain.SessionState, []Event, error) {
	if !domain.VoteAvailable(state.Wave.Number) || state.Wave.Vote.Closed {
		return state, nil, nil
	}
	next := state.Clone()
	counts := domain.VoteCounts{}
	for _, ballot := range next.Wave.Ballots {
		switch ballot.Option {
		case domain.VoteOptionA:
			counts.A++
		case domain.VoteOptionB:
			counts.B++
		case domain.VoteOptionC:
			counts.C++
		}
	}
	selected, err := domain.SelectVote(counts, next.Village.Resources.Wood, next.Village.Resources.Food)
	if err != nil {
		return state, nil, fmt.Errorf("%w: vote selection: %v", ErrInvalidState, err)
	}
	woodCost, foodCost := domain.VoteCost(selected)
	if woodCost > next.Village.Resources.Wood || foodCost > next.Village.Resources.Food {
		return state, nil, fmt.Errorf("%w: selected vote is not affordable", ErrInvalidState)
	}
	next.Village.Resources.Wood -= woodCost
	next.Village.Resources.Food -= foodCost
	next.Wave.Vote = domain.VoteState{Open: false, Closed: true, Selected: selected, Counts: counts}
	return next, []Event{{
		Type:   EventVoteClosed,
		WaveNo: next.Wave.Number,
		Tick:   next.Wave.Tick,
		Phase:  next.Wave.Phase,
		Amount: int64(selected),
	}}, nil
}

// VoteModifiers returns the authoritative combat modifiers for the current vote.
func VoteModifiers(state domain.SessionState) (attackPercent, guardPercent int64) {
	if !domain.VoteAvailable(state.Wave.Number) || !state.Wave.Vote.Closed {
		return 100, 100
	}
	return domain.VoteAttackPercent(state.Wave.Vote.Selected), domain.VoteGuardPercent(state.Wave.Vote.Selected)
}
