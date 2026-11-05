package ingest

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/hung98dev/ttgqldl/backend/internal/commands"
	"github.com/hung98dev/ttgqldl/backend/internal/db"
	"github.com/hung98dev/ttgqldl/backend/internal/domain"
	"github.com/hung98dev/ttgqldl/backend/internal/storage"
)

// DurableCandidate is one accepted W0304 candidate ready for transactional append. It
// contains only typed canonical action data and a compact fingerprint; raw comments,
// aliases and provider viewer ids are never carried into PostgreSQL.
type DurableCandidate struct {
	SessionID       string
	Provider        string
	ProviderEventID string
	Fingerprint     [32]byte
	ActorKey        domain.ActorKey
	Action          commands.Action
}

// DurableResult is the authoritative outcome of one candidate. InputSeq is zero for a
// duplicate whose historical receipt has no association.
type DurableResult struct {
	InputSeq      int64
	WaveNo        int64
	ScheduledTick int32
	AcceptedTick  int32
	Duplicate     bool
	Conflict      bool
}

// PostgresAppender performs the narrow W0305 append transaction. It never mutates
// reducer state; it only locks the session, allocates sequences, appends immutable input
// events and writes stable-ID receipts.
type PostgresAppender struct {
	pool         *storage.Pool
	maxRetries   int
	beforeCommit func() error
	afterCommit  func() error
}

// AppenderOption customizes retry bounds or injects test-only failure points.
type AppenderOption func(*PostgresAppender)

// WithMaxRetries bounds serialization/deadlock retries (minimum 1).
func WithMaxRetries(retries int) AppenderOption {
	return func(appender *PostgresAppender) {
		if retries < 1 {
			retries = 1
		}
		appender.maxRetries = retries
	}
}

// NewPostgresAppender wraps the storage pool. A nil pool is rejected.
func NewPostgresAppender(pool *storage.Pool, options ...AppenderOption) (*PostgresAppender, error) {
	if pool == nil {
		return nil, fmt.Errorf("%w: storage pool is required", ErrInvalidInput)
	}
	appender := &PostgresAppender{pool: pool, maxRetries: 3}
	for _, option := range options {
		option(appender)
	}
	return appender, nil
}

// AppendSession appends up to 256 candidates for one session in one transaction. It
// retries only SQLSTATE 40001/40P01, bounded by maxRetries.
func (a *PostgresAppender) AppendSession(ctx context.Context, sessionID string, candidates []DurableCandidate) ([]DurableResult, error) {
	if len(candidates) == 0 || len(candidates) > 256 {
		return nil, fmt.Errorf("%w: batch size %d", ErrInvalidInput, len(candidates))
	}
	sessionUUID, err := parseUUID(sessionID)
	if err != nil {
		return nil, fmt.Errorf("%w: session id", ErrInvalidInput)
	}
	for index := range candidates {
		if candidates[index].SessionID != sessionID {
			return nil, fmt.Errorf("%w: mixed session batch", ErrInvalidInput)
		}
		if err := validateDurableCandidate(candidates[index]); err != nil {
			return nil, err
		}
	}
	var lastErr error
	for attempt := 0; attempt < a.maxRetries; attempt++ {
		results, err := a.appendOnce(ctx, sessionUUID, candidates)
		if err == nil {
			return results, nil
		}
		lastErr = err
		if !retryablePostgres(err) {
			return nil, err
		}
	}
	return nil, fmt.Errorf("%w: %v", ErrRetryExhausted, lastErr)
}

func (a *PostgresAppender) appendOnce(ctx context.Context, sessionUUID pgtype.UUID, candidates []DurableCandidate) ([]DurableResult, error) {
	tx, err := a.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrStorageUnavailable, err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	queries := db.New(tx)

	session, err := queries.LockSession(ctx, sessionUUID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrSessionNotFound
		}
		return nil, fmt.Errorf("%w: %v", ErrStorageUnavailable, err)
	}
	if err := validateSessionForIngest(session, candidates[0].Provider); err != nil {
		return nil, err
	}
	waveNo, scheduledTick := scheduleForSession(session)
	results := make([]DurableResult, len(candidates))
	newIndexes := make([]int, 0, len(candidates))

	for index, candidate := range candidates {
		if candidate.ProviderEventID == "" {
			newIndexes = append(newIndexes, index)
			continue
		}
		receipt, err := queries.GetInputReceipt(ctx, db.GetInputReceiptParams{
			SessionID:       sessionUUID,
			ProviderSource:  candidate.Provider,
			ProviderEventID: candidate.ProviderEventID,
		})
		if errors.Is(err, pgx.ErrNoRows) {
			newIndexes = append(newIndexes, index)
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrStorageUnavailable, err)
		}
		stored := ""
		if receipt.PayloadFingerprint.Valid {
			stored = receipt.PayloadFingerprint.String
		}
		expected := hex.EncodeToString(candidate.Fingerprint[:])
		if stored != "" && stored != expected {
			results[index] = DurableResult{Conflict: true}
			continue
		}
		result := DurableResult{Duplicate: true, WaveNo: waveNo, ScheduledTick: scheduledTick, AcceptedTick: session.WaveTick}
		if receipt.InputSeq.Valid {
			result.InputSeq = receipt.InputSeq.Int64
			if event, eventErr := queries.GetInputEvent(ctx, db.GetInputEventParams{SessionID: sessionUUID, InputSeq: receipt.InputSeq.Int64}); eventErr == nil {
				result.WaveNo = event.WaveNo
				result.ScheduledTick = event.ScheduledTick
				result.AcceptedTick = event.AcceptedTick
			}
		}
		results[index] = result
	}

	if len(newIndexes) > 0 {
		count := int64(len(newIndexes))
		if session.InputSequence > math.MaxInt64-count {
			return nil, ErrSequenceOverflow
		}
		firstSeq := session.InputSequence + 1
		if err := queries.SetSessionInputSequence(ctx, db.SetSessionInputSequenceParams{
			ID: sessionUUID, InputSequence: firstSeq + count - 1,
		}); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrStorageUnavailable, err)
		}
		for offset, index := range newIndexes {
			candidate := candidates[index]
			sequence := firstSeq + int64(offset)
			payload, kind, err := canonicalPayload(candidate.Action)
			if err != nil {
				return nil, err
			}
			if err := queries.InsertInputEvent(ctx, db.InsertInputEventParams{
				SessionID:        sessionUUID,
				InputSeq:         sequence,
				AcceptedTick:     session.WaveTick,
				ScheduledTick:    scheduledTick,
				WaveNo:           waveNo,
				ActorKey:         string(candidate.ActorKey),
				Kind:             kind,
				CanonicalPayload: payload,
				RulesVersion:     session.RulesVersion,
			}); err != nil {
				return nil, fmt.Errorf("%w: %v", ErrStorageUnavailable, err)
			}
			if candidate.ProviderEventID != "" {
				if err := queries.InsertInputReceipt(ctx, db.InsertInputReceiptParams{
					SessionID:          sessionUUID,
					ProviderSource:     candidate.Provider,
					ProviderEventID:    candidate.ProviderEventID,
					PayloadFingerprint: pgtype.Text{String: hex.EncodeToString(candidate.Fingerprint[:]), Valid: true},
					InputSeq:           pgtype.Int8{Int64: sequence, Valid: true},
				}); err != nil {
					return nil, fmt.Errorf("%w: %v", ErrStorageUnavailable, err)
				}
			}
			results[index] = DurableResult{
				InputSeq: sequence, WaveNo: waveNo, ScheduledTick: scheduledTick, AcceptedTick: session.WaveTick,
			}
		}
	}

	if a.beforeCommit != nil {
		if err := a.beforeCommit(); err != nil {
			return nil, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		if retryablePostgres(err) {
			return nil, err
		}
		if reconciled, ok := a.reconcile(ctx, sessionUUID, candidates, results); ok {
			return reconciled, nil
		}
		return nil, fmt.Errorf("%w: %v", ErrCommitUnknown, err)
	}
	if a.afterCommit != nil {
		if err := a.afterCommit(); err != nil {
			return results, fmt.Errorf("%w: %v", ErrCommitUnknown, err)
		}
	}
	return results, nil
}

// reconcile queries durable receipts after an ambiguous commit. It proves success only
// when every stable candidate has a matching receipt and associated input event.
func (a *PostgresAppender) reconcile(ctx context.Context, sessionUUID pgtype.UUID, candidates []DurableCandidate, results []DurableResult) ([]DurableResult, bool) {
	tx, err := a.pool.Begin(ctx)
	if err != nil {
		return nil, false
	}
	defer func() { _ = tx.Rollback(ctx) }()
	queries := db.New(tx)
	reconciled := make([]DurableResult, len(candidates))
	for index, candidate := range candidates {
		if candidate.ProviderEventID == "" {
			return nil, false
		}
		receipt, err := queries.GetInputReceipt(ctx, db.GetInputReceiptParams{
			SessionID: sessionUUID, ProviderSource: candidate.Provider, ProviderEventID: candidate.ProviderEventID,
		})
		if err != nil || !receipt.PayloadFingerprint.Valid || !receipt.InputSeq.Valid {
			return nil, false
		}
		if receipt.PayloadFingerprint.String != hex.EncodeToString(candidate.Fingerprint[:]) {
			return nil, false
		}
		event, err := queries.GetInputEvent(ctx, db.GetInputEventParams{SessionID: sessionUUID, InputSeq: receipt.InputSeq.Int64})
		if err != nil {
			return nil, false
		}
		reconciled[index] = DurableResult{
			InputSeq: event.InputSeq, WaveNo: event.WaveNo, ScheduledTick: event.ScheduledTick, AcceptedTick: event.AcceptedTick,
		}
	}
	return reconciled, true
}

// LookupReceipt returns the durable association for one stable provider event, if any.
func (a *PostgresAppender) LookupReceipt(ctx context.Context, sessionID, provider, eventID string) (DurableResult, bool, error) {
	sessionUUID, err := parseUUID(sessionID)
	if err != nil {
		return DurableResult{}, false, ErrInvalidInput
	}
	tx, err := a.pool.Begin(ctx)
	if err != nil {
		return DurableResult{}, false, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	queries := db.New(tx)
	receipt, err := queries.GetInputReceipt(ctx, db.GetInputReceiptParams{
		SessionID: sessionUUID, ProviderSource: provider, ProviderEventID: eventID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return DurableResult{}, false, nil
	}
	if err != nil {
		return DurableResult{}, false, err
	}
	result := DurableResult{Duplicate: true}
	if receipt.InputSeq.Valid {
		result.InputSeq = receipt.InputSeq.Int64
		if event, eventErr := queries.GetInputEvent(ctx, db.GetInputEventParams{SessionID: sessionUUID, InputSeq: receipt.InputSeq.Int64}); eventErr == nil {
			result.WaveNo = event.WaveNo
			result.ScheduledTick = event.ScheduledTick
			result.AcceptedTick = event.AcceptedTick
		}
	}
	return result, true, nil
}

func validateDurableCandidate(candidate DurableCandidate) error {
	if candidate.Provider == "" || len(candidate.Provider) > 64 {
		return fmt.Errorf("%w: provider", ErrInvalidInput)
	}
	if len(candidate.ProviderEventID) > 256 {
		return fmt.Errorf("%w: provider event id", ErrInvalidInput)
	}
	if !candidate.ActorKey.Valid() {
		return fmt.Errorf("%w: actor key", ErrInvalidIdentity)
	}
	if !candidate.Action.Valid() {
		return fmt.Errorf("%w: action", ErrInvalidAction)
	}
	return nil
}

func validateSessionForIngest(session db.TtgSession, provider string) error {
	switch session.Mode {
	case "mock", "replay":
	case "authorized_live":
		return ErrLiveBlocked
	default:
		return ErrSessionUnavailable
	}
	if provider != session.Mode {
		return ErrProviderMismatch
	}
	switch session.Lifecycle {
	case "waiting", "running", "paused", "source_disconnected":
	case "stopped", "storage_unavailable":
		return ErrSessionUnavailable
	default:
		return ErrSessionUnavailable
	}
	if session.RulesVersion == "" {
		return ErrSessionUnavailable
	}
	return nil
}

// scheduleForSession derives scheduling solely from committed session state. It never
// mutates the session and never advances a logical tick.
func scheduleForSession(session db.TtgSession) (int64, int32) {
	if session.Lifecycle == "waiting" || session.WaveTick < 0 {
		return session.WaveNo, 0
	}
	if session.WaveTick >= 2800 { // RESULTS: registrations target the NEXT wave.
		return session.WaveNo + 1, 0
	}
	return session.WaveNo, session.WaveTick + 1
}

func canonicalPayload(action commands.Action) ([]byte, string, error) {
	switch action.Kind {
	case commands.KindRole:
		payload, err := json.Marshal(struct {
			Kind string `json:"kind"`
			Role string `json:"role"`
		}{Kind: "role", Role: action.Role.String()})
		return payload, "role", err
	case commands.KindBallot:
		payload, err := json.Marshal(struct {
			Kind   string `json:"kind"`
			Ballot string `json:"ballot"`
		}{Kind: "vote", Ballot: action.Ballot.String()})
		return payload, "vote", err
	default:
		return nil, "", fmt.Errorf("%w: action kind", ErrInvalidAction)
	}
}

func parseUUID(value string) (pgtype.UUID, error) {
	var uuid pgtype.UUID
	if err := uuid.Scan(value); err != nil {
		return pgtype.UUID{}, err
	}
	return uuid, nil
}

func retryablePostgres(err error) bool {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return false
	}
	return pgErr.Code == "40001" || pgErr.Code == "40P01"
}

// LookupCandidate resolves one candidate only when the durable receipt fingerprint
// matches, so an ambiguous commit cannot mistake a conflicting reuse for success.
func (a *PostgresAppender) LookupCandidate(ctx context.Context, candidate DurableCandidate) (DurableResult, bool, error) {
	sessionUUID, err := parseUUID(candidate.SessionID)
	if err != nil {
		return DurableResult{}, false, ErrInvalidInput
	}
	tx, err := a.pool.Begin(ctx)
	if err != nil {
		return DurableResult{}, false, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	queries := db.New(tx)
	receipt, err := queries.GetInputReceipt(ctx, db.GetInputReceiptParams{
		SessionID: sessionUUID, ProviderSource: candidate.Provider, ProviderEventID: candidate.ProviderEventID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return DurableResult{}, false, nil
	}
	if err != nil {
		return DurableResult{}, false, err
	}
	if !receipt.PayloadFingerprint.Valid || receipt.PayloadFingerprint.String != hex.EncodeToString(candidate.Fingerprint[:]) {
		return DurableResult{}, false, nil
	}
	result := DurableResult{Duplicate: true}
	if receipt.InputSeq.Valid {
		result.InputSeq = receipt.InputSeq.Int64
		if event, eventErr := queries.GetInputEvent(ctx, db.GetInputEventParams{SessionID: sessionUUID, InputSeq: receipt.InputSeq.Int64}); eventErr == nil {
			result.WaveNo = event.WaveNo
			result.ScheduledTick = event.ScheduledTick
			result.AcceptedTick = event.AcceptedTick
		}
	}
	return result, true, nil
}
