// Package reducer implements the W0402 durable, fenced simulation transition.
//
// The pure simulation package computes gameplay; this package loads the committed
// private runtime, accepts eligible durable inputs using their original accepted tick,
// advances exactly one logical transition, and persists the state, materialized tables,
// applied input statuses, domain events and stream outbox inside the W0401 lease fence.
package reducer

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/hung98dev/ttgqldl/backend/internal/canonical"
	"github.com/hung98dev/ttgqldl/backend/internal/domain"
	"github.com/hung98dev/ttgqldl/backend/internal/simulation"
	"github.com/hung98dev/ttgqldl/backend/internal/storage"
	"github.com/hung98dev/ttgqldl/backend/internal/writer"
	"math"
	"sort"
)

var (
	ErrNoWork            = errors.New("reducer: no eligible work")
	ErrRuntimeMissing    = errors.New("reducer: durable runtime state is missing")
	ErrRuntimeIncomplete = errors.New("reducer: pending inputs are not represented in runtime state")
	ErrInputBacklog      = errors.New("reducer: eligible input backlog is too large")
	ErrInvalidInput      = errors.New("reducer: invalid durable input")
	ErrInconsistentState = errors.New("reducer: durable state is inconsistent")
	ErrCommitUnverified  = errors.New("reducer: commit outcome could not be verified")
	ErrRevisionExhausted = errors.New("reducer: reducer revision exhausted")
)

const (
	runtimeFormat     = 1
	maxEligibleInputs = 5000
)

// Result describes one committed (or verified-reconciled) transition.
type Result struct {
	Revision     int64
	WaveNo       int64
	Tick         int
	StateHash    string
	Applied      []int64
	QueuedNext   []int64
	DomainEvents int
	Outbox       int
	Reconciled   bool
}

// Fencer is the narrow W0401 boundary used by the reducer. writer.Manager implements
// it; tests can substitute a controlled fence for ambiguous-commit reconciliation.
type Fencer interface {
	WithFence(ctx context.Context, lease writer.Lease, sessionID string, fn func(context.Context, writer.FencedTx) error) error
}

// Reducer owns the durable transition orchestration.
type Reducer struct {
	fence Fencer
	pool  *storage.Pool
	fault func(stage string) error
}

// New validates the dependencies.
func New(fence Fencer, pool *storage.Pool) (*Reducer, error) {
	if fence == nil || pool == nil {
		return nil, errors.New("reducer: fence and pool are required")
	}
	return &Reducer{fence: fence, pool: pool}, nil
}

type runtimeEnvelope struct {
	Format     int                           `json:"format"`
	State      json.RawMessage               `json:"state"`
	Checkpoint simulation.CheckpointEnvelope `json:"checkpoint"`
	QueuedNext []int64                       `json:"queued_next,omitempty"`
	Digest     string                        `json:"digest"`
}

func encodeRuntime(state domain.SessionState, queue simulation.RoleQueue, queuedNext []int64) (string, error) {
	stateJSON, err := json.Marshal(state)
	if err != nil {
		return "", err
	}
	checkpoint, err := simulation.BuildCheckpointEnvelope(state, queue)
	if err != nil {
		return "", err
	}
	queuedNext = normalizeSequences(queuedNext)
	envelope := runtimeEnvelope{Format: runtimeFormat, State: stateJSON, Checkpoint: checkpoint, QueuedNext: queuedNext}
	digest, err := runtimeDigest(envelope)
	if err != nil {
		return "", err
	}
	envelope.Digest = digest
	encoded, err := json.Marshal(envelope)
	if err != nil {
		return "", err
	}
	return string(encoded), nil
}

func decodeRuntime(raw []byte) (domain.SessionState, simulation.RoleQueue, []int64, error) {
	if len(raw) == 0 {
		return domain.SessionState{}, simulation.RoleQueue{}, nil, ErrRuntimeMissing
	}
	var envelope runtimeEnvelope
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return domain.SessionState{}, simulation.RoleQueue{}, nil, fmt.Errorf("%w: runtime JSON: %v", ErrInconsistentState, err)
	}
	if envelope.Format != runtimeFormat || envelope.Digest == "" {
		return domain.SessionState{}, simulation.RoleQueue{}, nil, ErrInconsistentState
	}
	digest, err := runtimeDigest(envelope)
	if err != nil || digest != envelope.Digest {
		return domain.SessionState{}, simulation.RoleQueue{}, nil, fmt.Errorf("%w: runtime digest mismatch", ErrInconsistentState)
	}
	var state domain.SessionState
	if err := json.Unmarshal(envelope.State, &state); err != nil {
		return domain.SessionState{}, simulation.RoleQueue{}, nil, fmt.Errorf("%w: runtime state: %v", ErrInconsistentState, err)
	}
	queue := simulation.RoleQueue{Entries: append([]simulation.RoleCommand(nil), envelope.Checkpoint.RoleQueue...), Ballots: append([]simulation.VoteCommand(nil), envelope.Checkpoint.BallotQueue...)}
	if err := simulation.VerifyCheckpointEnvelope(envelope.Checkpoint, state, queue); err != nil {
		return domain.SessionState{}, simulation.RoleQueue{}, nil, fmt.Errorf("%w: checkpoint: %v", ErrInconsistentState, err)
	}
	if err := state.Validate(); err != nil {
		return domain.SessionState{}, simulation.RoleQueue{}, nil, fmt.Errorf("%w: state: %v", ErrInconsistentState, err)
	}
	return state, queue, normalizeSequences(envelope.QueuedNext), nil
}

func runtimeDigest(envelope runtimeEnvelope) (string, error) {
	payload := struct {
		Format     int                           `json:"format"`
		Checkpoint simulation.CheckpointEnvelope `json:"checkpoint"`
		QueuedNext []int64                       `json:"queued_next,omitempty"`
	}{Format: envelope.Format, Checkpoint: envelope.Checkpoint, QueuedNext: normalizeSequences(envelope.QueuedNext)}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(encoded)
	return hex.EncodeToString(sum[:]), nil
}

func normalizeSequences(values []int64) []int64 {
	if len(values) == 0 {
		return nil
	}
	out := append([]int64(nil), values...)
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	unique := out[:0]
	for _, value := range out {
		if len(unique) == 0 || unique[len(unique)-1] != value {
			unique = append(unique, value)
		}
	}
	return unique
}

func sequenceSet(values []int64) map[int64]struct{} {
	set := make(map[int64]struct{}, len(values))
	for _, value := range values {
		set[value] = struct{}{}
	}
	return set
}

func queueSequences(queue simulation.RoleQueue) []int64 {
	out := make([]int64, 0, len(queue.Entries)+len(queue.Ballots))
	for _, entry := range queue.Entries {
		out = append(out, entry.InputSeq)
	}
	for _, ballot := range queue.Ballots {
		out = append(out, ballot.InputSeq)
	}
	return normalizeSequences(out)
}

func (r *Reducer) check(stage string) error {
	if r.fault != nil {
		return r.fault(stage)
	}
	return nil
}

func canonicalStateHash(state domain.SessionState) (string, error) {
	return canonical.Hash(state)
}

func nextRevision(current int64) (int64, error) {
	if current < 0 {
		return 0, ErrInconsistentState
	}
	if current == math.MaxInt64 {
		return 0, ErrRevisionExhausted
	}
	return current + 1, nil
}
