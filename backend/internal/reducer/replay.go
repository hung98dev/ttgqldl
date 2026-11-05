package reducer

import (
	"context"
	"errors"
	"fmt"
	"reflect"

	"github.com/hung98dev/ttgqldl/backend/internal/canonical"
	"github.com/hung98dev/ttgqldl/backend/internal/writer"
	"github.com/jackc/pgx/v5"
)

var (
	ErrReplayNoCheckpoint = errors.New("reducer: no usable checkpoint for replay")
	ErrReplayGap          = errors.New("reducer: replay history gap")
	ErrReplayMismatch     = errors.New("reducer: replay state mismatch")
	ErrReplayHistory      = errors.New("reducer: replay input history is inconsistent")
	ErrReplayTarget       = errors.New("reducer: replay target is invalid")
)

type ReplayResult struct {
	SessionID          string
	CheckpointRevision int64
	FromRevision       int64
	ToRevision         int64
	Transitions        int
	Inputs             int
	StateHash          string
}

type journalRow struct {
	Revision     int64
	Operation    string
	WaveNo       int64
	WaveTick     int
	StateHash    string
	InputSeqHigh int64
}

type checkpointRow struct {
	Revision int64
	State    []byte
	Hash     string
	WaveNo   int64
	WaveTick int
	InputSeq int64
	Rules    string
}

func (r *Reducer) VerifyReplay(ctx context.Context, sessionID string, targetRevision int64) (ReplayResult, error) {
	return r.verifyReplay(ctx, sessionID, 0, targetRevision)
}

// VerifyReplayFrom verifies from an explicitly selected checkpoint revision, allowing
// replay across later checkpoints (for example checkpoint 2400 through settlement 2800).
func (r *Reducer) VerifyReplayFrom(ctx context.Context, sessionID string, checkpointRevision, targetRevision int64) (ReplayResult, error) {
	return r.verifyReplay(ctx, sessionID, checkpointRevision, targetRevision)
}

func (r *Reducer) verifyReplay(ctx context.Context, sessionID string, checkpointRevision, targetRevision int64) (ReplayResult, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return ReplayResult{}, ErrReplayHistory
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var villageID, rules string
	var currentRevision int64
	var currentHash *string
	var currentWave int64
	var currentTick int32
	if err := tx.QueryRow(ctx, `SELECT village_id::text, rules_version, reducer_revision, last_state_hash, wave_no, wave_tick
		FROM ttg.sessions WHERE id = $1`, sessionID).Scan(&villageID, &rules, &currentRevision, &currentHash, &currentWave, &currentTick); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ReplayResult{}, writer.ErrSessionNotFound
		}
		return ReplayResult{}, ErrReplayHistory
	}
	if targetRevision == 0 {
		targetRevision = currentRevision
	}
	if targetRevision < 0 || targetRevision > currentRevision {
		return ReplayResult{}, ErrReplayTarget
	}

	checkpoint, err := loadCheckpoint(ctx, tx, sessionID, checkpointRevision, targetRevision)
	if err != nil {
		return ReplayResult{}, err
	}
	state, queue, queuedNext, err := verifyCheckpointEnvelope(checkpoint.State, sessionID, villageID, rules, checkpoint.Revision, checkpoint.Hash, checkpoint.WaveNo, checkpoint.WaveTick, checkpoint.InputSeq)
	if err != nil {
		return ReplayResult{}, err
	}
	if checkpoint.Revision > targetRevision {
		return ReplayResult{}, ErrReplayTarget
	}
	if checkpoint.Rules != rules {
		return ReplayResult{}, fmt.Errorf("%w: checkpoint rules version", ErrReplayMismatch)
	}

	journals, err := loadJournal(ctx, tx, sessionID, checkpoint.Revision, targetRevision)
	if err != nil {
		return ReplayResult{}, err
	}
	if targetRevision > checkpoint.Revision && len(journals) == 0 {
		return ReplayResult{}, ErrReplayGap
	}
	inputs, err := loadReplayInputs(ctx, tx, sessionID)
	if err != nil {
		return ReplayResult{}, err
	}
	type inputKey struct {
		Wave int64
		Tick int
	}
	byTick := make(map[inputKey][]inputRow)
	for _, input := range inputs {
		if input.AcceptedWave == nil {
			return ReplayResult{}, fmt.Errorf("%w: input %d has no accepted wave", ErrReplayHistory, input.InputSeq)
		}
		key := inputKey{Wave: *input.AcceptedWave, Tick: int(input.AcceptedTick)}
		byTick[key] = append(byTick[key], input)
	}
	replayed := 0
	revision := checkpoint.Revision
	for _, journal := range journals {
		if journal.Revision != revision+1 {
			return ReplayResult{}, fmt.Errorf("%w: expected revision %d, got %d", ErrReplayGap, revision+1, journal.Revision)
		}
		var selected []inputRow
		if state.Tick() != 3000 {
			for _, input := range byTick[inputKey{Wave: state.Wave.Number, Tick: state.Tick()}] {
				if input.InputSeq <= journal.InputSeqHigh {
					selected = append(selected, input)
				}
			}
		}
		transition, err := computeTransition(state, queue, queuedNext, selected)
		if err != nil {
			return ReplayResult{}, fmt.Errorf("%w: revision %d: %v", ErrReplayMismatch, journal.Revision, err)
		}
		if transition.Operation != journal.Operation || transition.State.Wave.Number != journal.WaveNo || transition.State.Tick() != journal.WaveTick {
			return ReplayResult{}, fmt.Errorf("%w: revision %d operation/metadata", ErrReplayMismatch, journal.Revision)
		}
		transition.State.InputSequence = journal.InputSeqHigh
		hash, err := canonical.Hash(transition.State)
		if err != nil || hash != journal.StateHash {
			return ReplayResult{}, fmt.Errorf("%w: revision %d hash", ErrReplayMismatch, journal.Revision)
		}
		state, queue, queuedNext = transition.State, transition.Queue, transition.QueuedNext
		revision = journal.Revision
		replayed++
	}
	if revision != targetRevision {
		return ReplayResult{}, fmt.Errorf("%w: ended at revision %d, target %d", ErrReplayGap, revision, targetRevision)
	}
	finalHash, err := canonical.Hash(state)
	if err != nil {
		return ReplayResult{}, err
	}
	if targetRevision == currentRevision {
		if currentHash == nil || *currentHash != finalHash || currentWave != state.Wave.Number || int(currentTick) != state.Tick() {
			return ReplayResult{}, fmt.Errorf("%w: target session metadata", ErrReplayMismatch)
		}
		var runtimeJSON []byte
		if err := tx.QueryRow(ctx, `SELECT runtime_state FROM ttg.sessions WHERE id = $1`, sessionID).Scan(&runtimeJSON); err != nil {
			return ReplayResult{}, ErrReplayHistory
		}
		authoritativeState, authoritativeQueue, authoritativeQueued, err := decodeRuntime(runtimeJSON)
		if err != nil {
			return ReplayResult{}, err
		}
		if !reflect.DeepEqual(state, authoritativeState) || !reflect.DeepEqual(queue, authoritativeQueue) || !reflect.DeepEqual(queuedNext, authoritativeQueued) {
			return ReplayResult{}, fmt.Errorf("%w: private continuation state", ErrReplayMismatch)
		}
	}
	return ReplayResult{
		SessionID: sessionID, CheckpointRevision: checkpoint.Revision, FromRevision: checkpoint.Revision,
		ToRevision: targetRevision, Transitions: replayed, Inputs: len(inputs), StateHash: finalHash,
	}, nil
}

func loadCheckpoint(ctx context.Context, tx pgx.Tx, sessionID string, checkpointRevision, target int64) (checkpointRow, error) {
	var row checkpointRow
	var err error
	if checkpointRevision > 0 {
		err = tx.QueryRow(ctx, `SELECT revision, state_json, state_hash, wave_no, wave_tick, input_seq, rules_version
			FROM ttg.checkpoints WHERE session_id = $1 AND revision = $2`, sessionID, checkpointRevision).Scan(
			&row.Revision, &row.State, &row.Hash, &row.WaveNo, &row.WaveTick, &row.InputSeq, &row.Rules,
		)
	} else {
		err = tx.QueryRow(ctx, `SELECT revision, state_json, state_hash, wave_no, wave_tick, input_seq, rules_version
			FROM ttg.checkpoints WHERE session_id = $1 AND revision <= $2
			ORDER BY revision DESC LIMIT 1`, sessionID, target).Scan(
			&row.Revision, &row.State, &row.Hash, &row.WaveNo, &row.WaveTick, &row.InputSeq, &row.Rules,
		)
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return checkpointRow{}, ErrReplayNoCheckpoint
	}
	if err != nil {
		return checkpointRow{}, ErrReplayHistory
	}
	return row, nil
}

func loadJournal(ctx context.Context, tx pgx.Tx, sessionID string, after, target int64) ([]journalRow, error) {
	rows, err := tx.Query(ctx, `SELECT revision, operation, wave_no, wave_tick, state_hash, input_seq_high
		FROM ttg.reducer_transitions
		WHERE session_id = $1 AND revision > $2 AND revision <= $3
		ORDER BY revision`, sessionID, after, target)
	if err != nil {
		return nil, ErrReplayHistory
	}
	defer rows.Close()
	out := make([]journalRow, 0, 16)
	for rows.Next() {
		var row journalRow
		if err := rows.Scan(&row.Revision, &row.Operation, &row.WaveNo, &row.WaveTick, &row.StateHash, &row.InputSeqHigh); err != nil {
			return nil, ErrReplayHistory
		}
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		return nil, ErrReplayHistory
	}
	return out, nil
}

func loadReplayInputs(ctx context.Context, tx pgx.Tx, sessionID string) ([]inputRow, error) {
	rows, err := tx.Query(ctx, `SELECT input_seq, accepted_wave, accepted_tick, scheduled_tick, wave_no, actor_key,
		kind, canonical_payload, rules_version
		FROM ttg.input_events WHERE session_id = $1 ORDER BY input_seq`, sessionID)
	if err != nil {
		return nil, ErrReplayHistory
	}
	defer rows.Close()
	inputs := make([]inputRow, 0, 16)
	seen := make(map[int64]struct{}, 16)
	for rows.Next() {
		var row inputRow
		if err := rows.Scan(&row.InputSeq, &row.AcceptedWave, &row.AcceptedTick, &row.ScheduledTick, &row.WaveNo, &row.ActorKey, &row.Kind, &row.Payload, &row.RulesVersion); err != nil {
			return nil, ErrReplayHistory
		}
		if _, exists := seen[row.InputSeq]; exists {
			return nil, fmt.Errorf("%w: duplicate input_seq %d", ErrReplayHistory, row.InputSeq)
		}
		seen[row.InputSeq] = struct{}{}
		inputs = append(inputs, row)
	}
	if err := rows.Err(); err != nil {
		return nil, ErrReplayHistory
	}
	return inputs, nil
}
