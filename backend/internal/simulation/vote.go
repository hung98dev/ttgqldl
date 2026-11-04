package simulation

import (
	"errors"
	"fmt"

	"github.com/hung98dev/ttgqldl/backend/internal/domain"
)

// VoteCommand is one already-canonicalized ballot. Chat parsing and identity
// mapping belong to M03; this layer accepts only validated domain values.
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

// VoteDeferredError reports validated ballots that were not stored because the
// session is operator-paused. The caller must retain and resubmit them.
type VoteDeferredError struct {
	Commands []VoteCommand
}

func (e *VoteDeferredError) Error() string {
	return fmt.Sprintf("simulation: %d ballot(s) deferred while paused", len(e.Commands))
}

func (e *VoteDeferredError) Is(target error) bool { return target == ErrInputDeferred }

// AcceptVoteCommands validates ballots and stores the latest accepted ballot per
// actor. Nothing is stored if any command is invalid. A ballot never changes the
// tally before the close boundary.
func AcceptVoteCommands(state domain.SessionState, commands []VoteCommand) (domain.SessionState, []VoteDecision, error) {
	if err := state.Validate(); err != nil {
		return state, nil, fmt.Errorf("%w: %v", ErrInvalidState, err)
	}
	switch state.Lifecycle {
	case domain.LifecycleStopped:
		return state, nil, ErrStopped
	case domain.LifecycleStorageUnavailable:
		return state, nil, ErrStorageUnavailable
	case domain.LifecycleSourceDisconnected:
		return state, nil, ErrSourceDisconnected
	}
	ordered := make([]VoteCommand, len(commands))
	copy(ordered, commands)
	sortVoteCommands(ordered)

	seen := make(map[int64]bool, len(ordered))
	for index := range ordered {
		command := &ordered[index]
		if command.InputSeq <= 0 {
			return state, nil, fmt.Errorf("%w: input_seq must be positive", ErrInvalidCommand)
		}
		if seen[command.InputSeq] {
			return state, nil, fmt.Errorf("%w: %d", ErrDuplicateInputSeq, command.InputSeq)
		}
		seen[command.InputSeq] = true
		if !command.ActorKey.Valid() {
			return state, nil, fmt.Errorf("%w: invalid actor key", ErrInvalidCommand)
		}
		if !command.Option.Valid() || command.Option == domain.VoteOptionNone {
			return state, nil, fmt.Errorf("%w: invalid ballot option", ErrInvalidCommand)
		}
		if !domain.VoteAvailable(state.Wave.Number) {
			return state, nil, fmt.Errorf("%w: voting is only available on even waves", ErrInvalidCommand)
		}
		if command.AcceptedTick == AcceptedTickUnspecified {
			command.AcceptedTick = state.Tick()
		}
		if command.AcceptedTick != state.Tick() {
			return state, nil, fmt.Errorf("%w: ballot accepted at %d but committed tick is %d", ErrRetroactiveInput, command.AcceptedTick, state.Tick())
		}
		if command.TargetWave != state.Wave.Number {
			return state, nil, fmt.Errorf("%w: ballot targets wave %d while the session is on %d", ErrInvalidCommand, command.TargetWave, state.Wave.Number)
		}
		if state.Tick() >= domain.VoteCloseTick {
			return state, nil, fmt.Errorf("%w: voting closed at tick %d", ErrInvalidCommand, domain.VoteCloseTick)
		}
		if !domain.BallotTickEligible(command.ScheduledTick) {
			return state, nil, fmt.Errorf("%w: ballot tick %d outside [800,1200)", ErrInvalidCommand, command.ScheduledTick)
		}
		position, isMember := -1, false
		for i, member := range state.Members {
			if member.ActorKey == command.ActorKey {
				position, isMember = i, true
				break
			}
		}
		if !isMember {
			return state, nil, fmt.Errorf("%w: only current-wave participants may vote", ErrInvalidCommand)
		}
		if state.Members[position].JoinedTick > command.ScheduledTick {
			return state, nil, fmt.Errorf("%w: participant is not effective at the ballot tick", ErrInvalidCommand)
		}
	}
	if state.Lifecycle == domain.LifecyclePaused {
		return state.Clone(), nil, &VoteDeferredError{Commands: ordered}
	}

	work := state.Clone()
	decisions := make([]VoteDecision, 0, len(ordered))
	for _, command := range ordered {
		work.Wave.Ballots = upsertBallot(work.Wave.Ballots, domain.Ballot{
			ActorKey: command.ActorKey,
			Option:   command.Option,
			Tick:     command.ScheduledTick,
			InputSeq: command.InputSeq,
		})
		decisions = append(decisions, VoteDecision{
			Outcome:  "ballot_accepted",
			InputSeq: command.InputSeq,
			ActorKey: command.ActorKey,
			Option:   command.Option,
			Tick:     command.ScheduledTick,
		})
	}
	if err := work.Validate(); err != nil {
		return state, nil, fmt.Errorf("%w: ballots produced an invalid state: %v", ErrInvalidState, err)
	}
	return work, decisions, nil
}

// upsertBallot keeps one ballot per actor: a later effective tick wins, and for
// the same tick the higher input sequence wins. The collection stays canonical.
func upsertBallot(ballots []domain.Ballot, ballot domain.Ballot) []domain.Ballot {
	for i := range ballots {
		if ballots[i].ActorKey != ballot.ActorKey {
			continue
		}
		replace := ballot.Tick > ballots[i].Tick ||
			(ballot.Tick == ballots[i].Tick && ballot.InputSeq >= ballots[i].InputSeq)
		if replace {
			ballots[i] = ballot
		}
		domain.SortBallots(ballots)
		return ballots
	}
	ballots = append(ballots, ballot)
	domain.SortBallots(ballots)
	return ballots
}

func sortVoteCommands(commands []VoteCommand) {
	for i := 0; i < len(commands); i++ {
		for j := i + 1; j < len(commands); j++ {
			if commands[j].InputSeq < commands[i].InputSeq {
				commands[i], commands[j] = commands[j], commands[i]
			}
		}
	}
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

// closeVote tallies the latest ballots, selects the option, deducts its cost
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
// Odd waves and unselected votes return 100/100.
func VoteModifiers(state domain.SessionState) (attackPercent, guardPercent int64) {
	if !domain.VoteAvailable(state.Wave.Number) || !state.Wave.Vote.Closed {
		return 100, 100
	}
	return domain.VoteAttackPercent(state.Wave.Vote.Selected), domain.VoteGuardPercent(state.Wave.Vote.Selected)
}

var _ = errors.Is
