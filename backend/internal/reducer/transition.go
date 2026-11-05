package reducer

import (
	"fmt"
	"sort"

	"github.com/hung98dev/ttgqldl/backend/internal/domain"
	"github.com/hung98dev/ttgqldl/backend/internal/simulation"
)

const (
	OperationStart    = "start"
	OperationAdvance  = "advance"
	OperationRollover = "rollover"
)

type transitionResult struct {
	State      domain.SessionState
	Queue      simulation.RoleQueue
	QueuedNext []int64
	Events     []simulation.Event
	Applied    []int64
	Operation  string
}

func computeTransition(state domain.SessionState, queue simulation.RoleQueue, queuedNext []int64, inputs []inputRow) (transitionResult, error) {
	if state.Tick() == domain.RolloverTick {
		next, nextQueue, events, _, err := simulation.AdvanceWithRoles(state, queue, simulation.Input{Kind: simulation.KindRollover})
		if err != nil {
			return transitionResult{}, fmt.Errorf("%w: rollover: %v", ErrInconsistentState, err)
		}
		applied := append([]int64(nil), queuedNext...)
		applied = append(applied, queueSequences(queue)...)
		return transitionResult{
			State: next, Queue: nextQueue, QueuedNext: nil, Events: events,
			Applied: normalizeSequences(applied), Operation: OperationRollover,
		}, nil
	}
	if state.Lifecycle != domain.LifecycleWaiting && state.Lifecycle != domain.LifecycleRunning {
		return transitionResult{}, ErrNoWork
	}
	roles, votes, nextWaveSeqs, err := decodeInputs(inputs, state)
	if err != nil {
		return transitionResult{}, err
	}
	operation := OperationAdvance
	if state.Lifecycle == domain.LifecycleWaiting {
		operation = OperationStart
		if len(votes) > 0 {
			return transitionResult{}, fmt.Errorf("%w: votes cannot start a waiting session", ErrInvalidInput)
		}
		if len(roles) == 0 {
			return transitionResult{}, ErrNoWork
		}
		state, queue, _, err = simulation.AcceptRoleCommands(state, queue, roles)
		if err != nil {
			return transitionResult{}, fmt.Errorf("%w: accept roles: %v", ErrInvalidInput, err)
		}
		if len(queue.Entries) == 0 {
			return transitionResult{}, ErrNoWork
		}
		head := queue.Entries[0]
		next, nextQueue, events, _, err := simulation.AdvanceWithRoles(state, queue, simulation.Input{
			Kind:         simulation.KindStartWave,
			Registration: &simulation.Registration{ActorKey: head.ActorKey, Role: head.Role},
		})
		if err != nil {
			return transitionResult{}, fmt.Errorf("%w: start wave: %v", ErrInconsistentState, err)
		}
		applied := []int64{head.InputSeq}
		return transitionResult{State: next, Queue: nextQueue, QueuedNext: normalizeSequences(nextWaveSeqs), Events: events, Applied: applied, Operation: operation}, nil
	}
	queueBefore := queue.Clone()
	if len(roles) > 0 {
		state, queue, _, err = simulation.AcceptRoleCommands(state, queue, roles)
		if err != nil {
			return transitionResult{}, fmt.Errorf("%w: accept roles: %v", ErrInvalidInput, err)
		}
	}
	if len(votes) > 0 {
		state, queue, _, err = simulation.AcceptVoteCommands(state, queue, votes)
		if err != nil {
			return transitionResult{}, fmt.Errorf("%w: accept votes: %v", ErrInvalidInput, err)
		}
	}
	next, nextQueue, events, _, err := simulation.AdvanceWithRoles(state, queue, simulation.Input{Kind: simulation.KindAdvanceTick})
	if err != nil {
		return transitionResult{}, fmt.Errorf("%w: advance: %v", ErrInconsistentState, err)
	}
	queuedNext = normalizeSequences(append(queuedNext, nextWaveSeqs...))
	after := sequenceSet(queueSequences(nextQueue))
	applied := make([]int64, 0)
	for _, seq := range queueSequences(queueBefore) {
		if _, exists := after[seq]; !exists {
			applied = append(applied, seq)
		}
	}
	for _, seq := range queueSequences(queue) {
		if _, exists := after[seq]; !exists {
			applied = append(applied, seq)
		}
	}
	applied = normalizeSequences(applied)
	handled := sequenceSet(append(append(queueSequences(nextQueue), queuedNext...), applied...))
	for _, input := range inputs {
		if _, ok := handled[input.InputSeq]; !ok {
			return transitionResult{}, fmt.Errorf("%w: input %d was neither applied nor queued", ErrInconsistentState, input.InputSeq)
		}
	}
	return transitionResult{State: next, Queue: nextQueue, QueuedNext: queuedNext, Events: events, Applied: applied, Operation: operation}, nil
}

func checkpointReason(state domain.SessionState, operation string) (string, bool) {
	tick := state.Tick()
	if operation == "privacy" {
		return "privacy", true
	}
	switch {
	case tick == 0 && operation == OperationRollover:
		return "wave_start", true
	case tick == 0 && operation == OperationStart:
		return "initial", true
	case tick == domain.FinalCombatTick:
		return "settlement", true
	case tick == domain.RolloverTick:
		return "control", true
	case tick > 0 && tick%600 == 0:
		return "periodic", true
	default:
		return "", false
	}
}

func sortedInputs(inputs []inputRow) []inputRow {
	out := append([]inputRow(nil), inputs...)
	sort.Slice(out, func(i, j int) bool { return out[i].InputSeq < out[j].InputSeq })
	return out
}
