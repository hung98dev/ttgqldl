package reducer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sort"

	"github.com/hung98dev/ttgqldl/backend/internal/canonical"
	"github.com/hung98dev/ttgqldl/backend/internal/domain"
	"github.com/hung98dev/ttgqldl/backend/internal/simulation"
	"github.com/hung98dev/ttgqldl/backend/internal/writer"
	"github.com/jackc/pgx/v5"
)

type sessionRow struct {
	ID             string
	VillageID      string
	Mode           string
	Lifecycle      string
	WaveNo         int64
	WaveTick       int32
	PrivacyEpoch   int64
	RulesVersion   string
	WriterEpoch    int64
	InputSequence  int64
	StreamSequence int64
	LastReduced    int32
	Revision       int64
	Runtime        []byte
}

type villageRow struct {
	Level             int16
	WallTier          int16
	WallHP            int64
	TownhallHP        int64
	Wood              int64
	Stone             int64
	Food              int64
	ConstructionWork  int64
	TotalWins         int64
	ConsecutiveWins   int16
	ConsecutiveLosses int16
}

type inputRow struct {
	InputSeq      int64
	AcceptedTick  int32
	ScheduledTick int32
	WaveNo        int64
	ActorKey      string
	Kind          string
	Payload       []byte
	RulesVersion  string
}

// Step performs exactly one fenced logical transition and commits every durable
// effect atomically.
func (r *Reducer) Step(ctx context.Context, lease writer.Lease, sessionID string) (Result, error) {
	var result Result
	var expectedRevision int64
	var expectedHash string
	err := r.fence.WithFence(ctx, lease, sessionID, func(ctx context.Context, tx writer.FencedTx) error {
		session, err := loadSession(ctx, tx, sessionID)
		if err != nil {
			return err
		}
		if session.WriterEpoch != lease.Epoch {
			return writer.ErrStaleEpoch
		}
		if session.RulesVersion != domain.RulesVersion {
			return fmt.Errorf("%w: unsupported rules version", ErrInconsistentState)
		}
		village, err := loadVillage(ctx, tx, session.VillageID)
		if err != nil {
			return err
		}
		state, queue, queuedNext, err := loadRuntimeState(session)
		if err != nil {
			return err
		}
		if err := verifySessionState(session, state); err != nil {
			return err
		}
		if err := r.check("after_load"); err != nil {
			return err
		}

		oldState := state.Clone()
		var next domain.SessionState
		var nextQueue simulation.RoleQueue
		var events []simulation.Event
		var applied []int64
		if state.Tick() == domain.RolloverTick {
			next, nextQueue, events, _, err = simulation.AdvanceWithRoles(state, queue, simulation.Input{Kind: simulation.KindRollover})
			if err != nil {
				return fmt.Errorf("%w: rollover: %v", ErrInconsistentState, err)
			}
			applied = append(applied, queuedNext...)
			queuedNext = nil
			// Any current-wave queue entries that became due at the control boundary are
			// consumed as well; the normal W0305 schedule does not produce them.
			applied = append(applied, queueSequences(queue)...)
			applied = normalizeSequences(applied)
			_ = oldState
		} else {
			if state.Lifecycle != domain.LifecycleWaiting && state.Lifecycle != domain.LifecycleRunning {
				return ErrNoWork
			}
			if err := r.check("before_select"); err != nil {
				return err
			}
			inputs, err := loadEligibleInputs(ctx, tx, session.ID, state.Tick())
			if err != nil {
				return err
			}
			if err := verifyPendingRepresented(ctx, tx, session.ID, state.Tick(), queue, queuedNext); err != nil {
				return err
			}
			if err := r.check("after_select"); err != nil {
				return err
			}
			roles, votes, nextWaveSeqs, err := decodeInputs(inputs, state)
			if err != nil {
				return err
			}
			queueBefore := queue.Clone()
			if state.Lifecycle == domain.LifecycleWaiting {
				if len(votes) > 0 {
					return fmt.Errorf("%w: votes cannot start a waiting session", ErrInvalidInput)
				}
				if len(roles) == 0 {
					return ErrNoWork
				}
				state, queue, _, err = simulation.AcceptRoleCommands(state, queue, roles)
				if err != nil {
					return fmt.Errorf("%w: accept roles: %v", ErrInvalidInput, err)
				}
				if len(queue.Entries) == 0 {
					return ErrNoWork
				}
				head := queue.Entries[0]
				next, nextQueue, events, _, err = simulation.AdvanceWithRoles(state, queue, simulation.Input{
					Kind:         simulation.KindStartWave,
					Registration: &simulation.Registration{ActorKey: head.ActorKey, Role: head.Role},
				})
				if err != nil {
					return fmt.Errorf("%w: start wave: %v", ErrInconsistentState, err)
				}
			} else {
				if len(roles) > 0 {
					state, queue, _, err = simulation.AcceptRoleCommands(state, queue, roles)
					if err != nil {
						return fmt.Errorf("%w: accept roles: %v", ErrInvalidInput, err)
					}
				}
				if len(votes) > 0 {
					state, queue, _, err = simulation.AcceptVoteCommands(state, queue, votes)
					if err != nil {
						return fmt.Errorf("%w: accept votes: %v", ErrInvalidInput, err)
					}
				}
				next, nextQueue, events, _, err = simulation.AdvanceWithRoles(state, queue, simulation.Input{Kind: simulation.KindAdvanceTick})
				if err != nil {
					return fmt.Errorf("%w: advance: %v", ErrInconsistentState, err)
				}
			}
			queuedNext = normalizeSequences(append(queuedNext, nextWaveSeqs...))
			consumed := queueSequences(queueBefore)
			after := sequenceSet(queueSequences(nextQueue))
			for _, seq := range queueSequences(queue) {
				if _, exists := after[seq]; !exists {
					consumed = append(consumed, seq)
				}
			}
			consumedSet := sequenceSet(consumed)
			for _, seq := range queueSequences(queue) {
				if _, exists := consumedSet[seq]; !exists {
					continue
				}
				if !contains(queueSequences(nextQueue), seq) {
					applied = append(applied, seq)
				}
			}
			// Every selected current-tick input must be consumed, queued, or explicitly
			// queued for the next wave. This is the anti-silent-drop guard.
			handled := sequenceSet(append(append(queueSequences(nextQueue), queuedNext...), applied...))
			for _, input := range inputs {
				if _, ok := handled[input.InputSeq]; !ok {
					return fmt.Errorf("%w: input %d was neither applied nor queued", ErrInconsistentState, input.InputSeq)
				}
			}
			applied = normalizeSequences(applied)
		}
		if err := r.check("after_apply"); err != nil {
			return err
		}
		if err := next.Validate(); err != nil {
			return fmt.Errorf("%w: transition produced invalid state: %v", ErrInconsistentState, err)
		}
		next.InputSequence = session.InputSequence
		next.PrivacyEpoch = session.PrivacyEpoch
		stateHash, err := canonical.Hash(next)
		if err != nil {
			return err
		}
		revision, err := nextRevision(session.Revision)
		if err != nil {
			return err
		}
		runtimeJSON, err := encodeRuntime(next, nextQueue, queuedNext)
		if err != nil {
			return err
		}
		streamSequence := session.StreamSequence
		if len(events) > 0 {
			if streamSequence > math.MaxInt64-int64(len(events)) {
				return fmt.Errorf("%w: stream sequence overflow", ErrInconsistentState)
			}
			streamSequence += int64(len(events))
		}
		appliedTick := next.Tick()
		if state.Tick() == domain.RolloverTick {
			appliedTick = domain.RolloverTick
		}
		if err := r.persist(ctx, tx, session, village, oldState, next, nextQueue, runtimeJSON, events, applied, revision, streamSequence, stateHash, appliedTick); err != nil {
			return err
		}
		if err := r.check("before_commit"); err != nil {
			return err
		}
		expectedRevision = revision
		expectedHash = stateHash
		result = Result{
			Revision: revision, WaveNo: next.Wave.Number, Tick: next.Tick(), StateHash: stateHash,
			Applied: append([]int64(nil), applied...), QueuedNext: append([]int64(nil), queuedNext...),
			DomainEvents: len(events), Outbox: len(events),
		}
		return nil
	})
	if err != nil {
		if errors.Is(err, writer.ErrCommitUnknown) {
			return r.reconcile(ctx, sessionID, expectedRevision, expectedHash, result)
		}
		return Result{}, err
	}
	return result, nil
}

func (r *Reducer) reconcile(ctx context.Context, sessionID string, revision int64, hash string, result Result) (Result, error) {
	if revision == 0 || hash == "" {
		return Result{}, writer.ErrCommitUnknown
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Result{}, ErrCommitUnverified
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var gotRevision int64
	var gotHash *string
	var waveNo int64
	var tick int32
	if err := tx.QueryRow(ctx, `SELECT reducer_revision, last_state_hash, wave_no, wave_tick FROM ttg.sessions WHERE id = $1`, sessionID).Scan(&gotRevision, &gotHash, &waveNo, &tick); err != nil {
		return Result{}, ErrCommitUnverified
	}
	if gotRevision != revision || gotHash == nil || *gotHash != hash {
		return Result{}, writer.ErrCommitUnknown
	}
	result.Reconciled = true
	result.WaveNo = waveNo
	result.Tick = int(tick)
	return result, nil
}

func loadSession(ctx context.Context, tx writer.FencedTx, sessionID string) (sessionRow, error) {
	var row sessionRow
	err := tx.QueryRow(ctx, `SELECT id::text, village_id::text, mode, lifecycle, wave_no, wave_tick,
		privacy_epoch, rules_version, writer_epoch, input_sequence, stream_sequence,
		last_reduced_tick, reducer_revision, runtime_state
		FROM ttg.sessions WHERE id = $1`, sessionID).Scan(
		&row.ID, &row.VillageID, &row.Mode, &row.Lifecycle, &row.WaveNo, &row.WaveTick,
		&row.PrivacyEpoch, &row.RulesVersion, &row.WriterEpoch, &row.InputSequence,
		&row.StreamSequence, &row.LastReduced, &row.Revision, &row.Runtime,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return sessionRow{}, writer.ErrSessionNotFound
	}
	if err != nil {
		return sessionRow{}, err
	}
	return row, nil
}

func loadVillage(ctx context.Context, tx writer.FencedTx, villageID string) (villageRow, error) {
	var row villageRow
	err := tx.QueryRow(ctx, `SELECT level, wall_tier, wall_hp, townhall_hp, wood, stone, food,
		construction_work, total_wins, consecutive_wins, consecutive_losses
		FROM ttg.villages WHERE id = $1`, villageID).Scan(
		&row.Level, &row.WallTier, &row.WallHP, &row.TownhallHP, &row.Wood, &row.Stone,
		&row.Food, &row.ConstructionWork, &row.TotalWins, &row.ConsecutiveWins, &row.ConsecutiveLosses,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return villageRow{}, writer.ErrVillageNotFound
	}
	if err != nil {
		return villageRow{}, err
	}
	return row, nil
}

func loadRuntimeState(session sessionRow) (domain.SessionState, simulation.RoleQueue, []int64, error) {
	mode, err := parseMode(session.Mode)
	if err != nil {
		return domain.SessionState{}, simulation.RoleQueue{}, nil, err
	}
	lifecycle, err := parseLifecycle(session.Lifecycle)
	if err != nil {
		return domain.SessionState{}, simulation.RoleQueue{}, nil, err
	}
	if len(session.Runtime) == 0 {
		if session.WaveNo != 1 || session.WaveTick != -1 || session.LastReduced != -1 {
			return domain.SessionState{}, simulation.RoleQueue{}, nil, ErrRuntimeMissing
		}
		state, err := domain.NewInitialState(domain.VillageID(session.VillageID), domain.SessionID(session.ID), mode)
		if err != nil {
			return domain.SessionState{}, simulation.RoleQueue{}, nil, err
		}
		state.Lifecycle = lifecycle
		state.InputSequence = session.InputSequence
		state.PrivacyEpoch = session.PrivacyEpoch
		return state, simulation.RoleQueue{}, nil, nil
	}
	state, queue, queuedNext, err := decodeRuntime(session.Runtime)
	if err != nil {
		return domain.SessionState{}, simulation.RoleQueue{}, nil, err
	}
	state.InputSequence = session.InputSequence
	state.PrivacyEpoch = session.PrivacyEpoch
	return state, queue, queuedNext, nil
}

func verifySessionState(session sessionRow, state domain.SessionState) error {
	if state.ID != domain.SessionID(session.ID) || state.VillageID != domain.VillageID(session.VillageID) {
		return fmt.Errorf("%w: runtime identity does not match session", ErrInconsistentState)
	}
	if state.Wave.Number != session.WaveNo || state.Tick() != int(session.WaveTick) || int(session.LastReduced) != state.Tick() {
		return fmt.Errorf("%w: runtime tick/wave does not match session", ErrInconsistentState)
	}
	if state.Lifecycle.String() != session.Lifecycle {
		return fmt.Errorf("%w: runtime lifecycle does not match session", ErrInconsistentState)
	}
	if !state.Revision.Active() || state.Revision.RulesVersion != session.RulesVersion {
		return fmt.Errorf("%w: runtime revision does not match session", ErrInconsistentState)
	}
	return nil
}

func loadEligibleInputs(ctx context.Context, tx writer.FencedTx, sessionID string, acceptedTick int) ([]inputRow, error) {
	rows, err := tx.Query(ctx, `SELECT input_seq, accepted_tick, scheduled_tick, wave_no, actor_key,
		kind, canonical_payload, rules_version
		FROM ttg.input_events
		WHERE session_id = $1 AND applied_at_tick IS NULL AND accepted_tick = $2
		ORDER BY input_seq
		LIMIT $3`, sessionID, acceptedTick, maxEligibleInputs+1)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	inputs := make([]inputRow, 0, 16)
	for rows.Next() {
		var row inputRow
		if err := rows.Scan(&row.InputSeq, &row.AcceptedTick, &row.ScheduledTick, &row.WaveNo, &row.ActorKey, &row.Kind, &row.Payload, &row.RulesVersion); err != nil {
			return nil, err
		}
		inputs = append(inputs, row)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(inputs) > maxEligibleInputs {
		return nil, ErrInputBacklog
	}
	return inputs, nil
}

func verifyPendingRepresented(ctx context.Context, tx writer.FencedTx, sessionID string, currentTick int, queue simulation.RoleQueue, queuedNext []int64) error {
	represented := sequenceSet(append(queueSequences(queue), queuedNext...))
	rows, err := tx.Query(ctx, `SELECT input_seq FROM ttg.input_events
		WHERE session_id = $1 AND applied_at_tick IS NULL AND accepted_tick < $2`, sessionID, currentTick)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var seq int64
		if err := rows.Scan(&seq); err != nil {
			return err
		}
		if _, ok := represented[seq]; !ok {
			return fmt.Errorf("%w: input %d", ErrRuntimeIncomplete, seq)
		}
	}
	return rows.Err()
}

func decodeInputs(inputs []inputRow, state domain.SessionState) ([]simulation.RoleCommand, []simulation.VoteCommand, []int64, error) {
	roles := make([]simulation.RoleCommand, 0, len(inputs))
	votes := make([]simulation.VoteCommand, 0, len(inputs))
	nextWave := make([]int64, 0)
	for _, input := range inputs {
		if input.RulesVersion != state.Revision.RulesVersion {
			return nil, nil, nil, fmt.Errorf("%w: input rules version", ErrInvalidInput)
		}
		actor := domain.ActorKey(input.ActorKey)
		if !actor.Valid() {
			return nil, nil, nil, fmt.Errorf("%w: actor", ErrInvalidInput)
		}
		var payload struct {
			Kind   string `json:"kind"`
			Role   string `json:"role"`
			Ballot string `json:"ballot"`
		}
		if err := json.Unmarshal(input.Payload, &payload); err != nil {
			return nil, nil, nil, fmt.Errorf("%w: payload", ErrInvalidInput)
		}
		switch input.Kind {
		case "role":
			role, err := domain.ParseRole(payload.Role)
			if err != nil {
				return nil, nil, nil, fmt.Errorf("%w: role", ErrInvalidInput)
			}
			command := simulation.RoleCommand{
				InputSeq: input.InputSeq, ActorKey: actor, Role: role,
				TargetWave: input.WaveNo, ScheduledTick: int(input.ScheduledTick), AcceptedTick: int(input.AcceptedTick),
			}
			roles = append(roles, command)
			if input.WaveNo == state.Wave.Number+1 {
				nextWave = append(nextWave, input.InputSeq)
			}
		case "vote":
			option, err := parseVote(payload.Ballot)
			if err != nil {
				return nil, nil, nil, err
			}
			votes = append(votes, simulation.VoteCommand{
				InputSeq: input.InputSeq, ActorKey: actor, TargetWave: input.WaveNo,
				AcceptedTick: int(input.AcceptedTick), ScheduledTick: int(input.ScheduledTick), Option: option,
			})
		default:
			return nil, nil, nil, fmt.Errorf("%w: kind", ErrInvalidInput)
		}
	}
	return roles, votes, nextWave, nil
}

func parseMode(value string) (domain.SourceMode, error) {
	switch value {
	case "mock":
		return domain.SourceModeMock, nil
	case "replay":
		return domain.SourceModeReplay, nil
	case "authorized_live":
		return domain.SourceModeAuthorizedLive, nil
	default:
		return 0, fmt.Errorf("%w: mode", ErrInconsistentState)
	}
}

func parseLifecycle(value string) (domain.Lifecycle, error) {
	for _, lifecycle := range []domain.Lifecycle{domain.LifecycleWaiting, domain.LifecycleRunning, domain.LifecyclePaused, domain.LifecycleSourceDisconnected, domain.LifecycleStorageUnavailable, domain.LifecycleStopped} {
		if lifecycle.String() == value {
			return lifecycle, nil
		}
	}
	return 0, fmt.Errorf("%w: lifecycle", ErrInconsistentState)
}

func parseVote(value string) (domain.VoteOption, error) {
	switch value {
	case "A":
		return domain.VoteOptionA, nil
	case "B":
		return domain.VoteOptionB, nil
	case "C":
		return domain.VoteOptionC, nil
	default:
		return 0, fmt.Errorf("%w: ballot", ErrInvalidInput)
	}
}

func contains(values []int64, target int64) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func sortedInt64(values []int64) []int64 {
	out := append([]int64(nil), values...)
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}
