package ingest

import (
	"context"
	"errors"
	"math"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/hung98dev/ttgqldl/backend/internal/storage"
)

func TestDurableConcurrentDuplicate(t *testing.T) {
	pool := testPool(t)
	session := createSession(t, pool, "mock", "running", 1, 10, 0)
	appender := newTestAppender(t, pool)
	candidate := candidateFor(session.ID, "concurrent-dup", actorFor(5), mustAction(t, "XÂY"))

	var wait sync.WaitGroup
	results := make(chan []DurableResult, 2)
	for index := 0; index < 2; index++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			result, err := appender.AppendSession(context.Background(), session.ID, []DurableCandidate{candidate})
			if err != nil {
				t.Errorf("append: %v", err)
				return
			}
			results <- result
		}()
	}
	wait.Wait()
	close(results)
	created, duplicates := 0, 0
	for result := range results {
		if result[0].Duplicate {
			duplicates++
		} else {
			created++
		}
	}
	if created != 1 || duplicates != 1 {
		t.Fatalf("created=%d duplicates=%d", created, duplicates)
	}
	if got := scalarInt(t, pool, `SELECT count(*) FROM ttg.input_events WHERE session_id = $1`, session.ID); got != 1 {
		t.Fatalf("events = %d", got)
	}
}

func TestDurableConcurrentBatchesMonotonic(t *testing.T) {
	pool := testPool(t)
	session := createSession(t, pool, "mock", "running", 1, 20, 0)
	appender := newTestAppender(t, pool)
	actor := actorFor(6)
	action := mustAction(t, "THỦ")
	var wait sync.WaitGroup
	sequences := make(chan int64, 100)
	for batch := 0; batch < 10; batch++ {
		wait.Add(1)
		go func(batch int) {
			defer wait.Done()
			candidates := make([]DurableCandidate, 10)
			for index := range candidates {
				candidates[index] = candidateFor(session.ID, "concurrent-"+string(rune('a'+batch))+"-"+string(rune('a'+index)), actor, action)
			}
			results, err := appender.AppendSession(context.Background(), session.ID, candidates)
			if err != nil {
				t.Errorf("append: %v", err)
				return
			}
			for _, result := range results {
				sequences <- result.InputSeq
			}
		}(batch)
	}
	wait.Wait()
	close(sequences)
	seen := make(map[int64]bool)
	for sequence := range sequences {
		if seen[sequence] {
			t.Fatalf("duplicate sequence %d", sequence)
		}
		seen[sequence] = true
	}
	if len(seen) != 100 {
		t.Fatalf("sequences = %d", len(seen))
	}
	ordered := make([]int64, 0, len(seen))
	for sequence := range seen {
		ordered = append(ordered, sequence)
	}
	sort.Slice(ordered, func(i, j int) bool { return ordered[i] < ordered[j] })
	for index, sequence := range ordered {
		if sequence != int64(index+1) {
			t.Fatalf("non-consecutive sequence at %d: %d", index, sequence)
		}
	}
	if got := scalarInt(t, pool, `SELECT input_sequence FROM ttg.sessions WHERE id = $1`, session.ID); got != 100 {
		t.Fatalf("session sequence = %d", got)
	}
}

func TestDurableWaitingConcurrentRegistrations(t *testing.T) {
	pool := testPool(t)
	session := createSession(t, pool, "mock", "waiting", 1, -1, 0)
	appender := newTestAppender(t, pool)
	actor := actorFor(7)
	action := mustAction(t, "XÂY")
	var wait sync.WaitGroup
	results := make(chan DurableResult, 2)
	for index := 0; index < 2; index++ {
		wait.Add(1)
		go func(index int) {
			defer wait.Done()
			result, err := appender.AppendSession(context.Background(), session.ID, []DurableCandidate{
				candidateFor(session.ID, "first-start-"+string(rune('a'+index)), actor, action),
			})
			if err != nil {
				t.Errorf("append: %v", err)
				return
			}
			results <- result[0]
		}(index)
	}
	wait.Wait()
	close(results)
	for result := range results {
		if result.ScheduledTick != 0 || result.WaveNo != 1 || result.InputSeq < 1 || result.InputSeq > 2 {
			t.Fatalf("first-start scheduling = %+v", result)
		}
	}
	// W0305 does not mutate lifecycle or claim reducer start ownership; it preserves the
	// deterministic tick-0 schedule and unique sequence for the reducer's one-start rule.
	if got := scalarInt(t, pool, `SELECT count(*) FROM ttg.input_events WHERE session_id = $1`, session.ID); got != 2 {
		t.Fatalf("events = %d", got)
	}
}

func TestDurableMissingEventID(t *testing.T) {
	pool := testPool(t)
	session := createSession(t, pool, "mock", "running", 1, 30, 0)
	appender := newTestAppender(t, pool)
	actor := actorFor(8)
	action := mustAction(t, "A")
	results, err := appender.AppendSession(context.Background(), session.ID, []DurableCandidate{
		candidateFor(session.ID, "", actor, action),
		candidateFor(session.ID, "", actor, action),
	})
	if err != nil {
		t.Fatal(err)
	}
	if results[0].InputSeq != 1 || results[1].InputSeq != 2 {
		t.Fatalf("missing-id sequences = %+v", results)
	}
	if got := scalarInt(t, pool, `SELECT count(*) FROM ttg.input_receipts WHERE session_id = $1`, session.ID); got != 0 {
		t.Fatalf("missing-id fabricated %d receipts", got)
	}
}

func TestDurableRestartDuplicate(t *testing.T) {
	pool := testPool(t)
	session := createSession(t, pool, "mock", "running", 1, 40, 0)
	actor := actorFor(9)
	action := mustAction(t, "ĐÁNH")
	candidate := candidateFor(session.ID, "restart-1", actor, action)
	first := newTestAppender(t, pool)
	if _, err := first.AppendSession(context.Background(), session.ID, []DurableCandidate{candidate}); err != nil {
		t.Fatal(err)
	}
	second := newTestAppender(t, pool)
	result, err := second.AppendSession(context.Background(), session.ID, []DurableCandidate{candidate})
	if err != nil || !result[0].Duplicate || result[0].InputSeq != 1 {
		t.Fatalf("restart duplicate = %+v/%v", result, err)
	}
}

func TestDurableInvalidCandidateDoesNotMutate(t *testing.T) {
	pool := testPool(t)
	session := createSession(t, pool, "mock", "running", 1, 50, 0)
	appender := newTestAppender(t, pool)
	invalid := candidateFor(session.ID, "invalid-1", "", mustAction(t, "XÂY"))
	if _, err := appender.AppendSession(context.Background(), session.ID, []DurableCandidate{invalid}); !errors.Is(err, ErrInvalidIdentity) {
		t.Fatalf("invalid candidate error = %v", err)
	}
	if got := scalarInt(t, pool, `SELECT count(*) FROM ttg.input_events WHERE session_id = $1`, session.ID); got != 0 {
		t.Fatalf("invalid candidate wrote %d events", got)
	}
	if got := scalarInt(t, pool, `SELECT input_sequence FROM ttg.sessions WHERE id = $1`, session.ID); got != 0 {
		t.Fatalf("invalid candidate advanced sequence to %d", got)
	}
}

func TestDurableRollbackLeavesNoGhosts(t *testing.T) {
	pool := testPool(t)
	session := createSession(t, pool, "mock", "running", 1, 60, 0)
	appender := newTestAppender(t, pool)
	appender.beforeCommit = func() error { return errors.New("simulated crash before commit") }
	_, err := appender.AppendSession(context.Background(), session.ID, []DurableCandidate{
		candidateFor(session.ID, "rollback-1", actorFor(10), mustAction(t, "XÂY")),
	})
	if err == nil {
		t.Fatal("simulated crash did not fail")
	}
	if got := scalarInt(t, pool, `SELECT count(*) FROM ttg.input_events WHERE session_id = $1`, session.ID); got != 0 {
		t.Fatalf("rollback left %d events", got)
	}
	if got := scalarInt(t, pool, `SELECT count(*) FROM ttg.input_receipts WHERE session_id = $1`, session.ID); got != 0 {
		t.Fatalf("rollback left %d receipts", got)
	}
	if got := scalarInt(t, pool, `SELECT input_sequence FROM ttg.sessions WHERE id = $1`, session.ID); got != 0 {
		t.Fatalf("rollback consumed sequence %d", got)
	}
	appender.beforeCommit = nil
	result, err := appender.AppendSession(context.Background(), session.ID, []DurableCandidate{
		candidateFor(session.ID, "rollback-1", actorFor(10), mustAction(t, "XÂY")),
	})
	if err != nil || result[0].InputSeq != 1 {
		t.Fatalf("retry after rollback = %+v/%v", result, err)
	}
}

func TestDurableAmbiguousCommitReconciles(t *testing.T) {
	pool := testPool(t)
	session := createSession(t, pool, "mock", "running", 1, 70, 0)
	appender := newTestAppender(t, pool)
	candidate := candidateFor(session.ID, "ambiguous-1", actorFor(11), mustAction(t, "XÂY"))
	appender.afterCommit = func() error { return errors.New("acknowledgment lost") }
	_, err := appender.AppendSession(context.Background(), session.ID, []DurableCandidate{candidate})
	if !errors.Is(err, ErrCommitUnknown) {
		t.Fatalf("ambiguous commit error = %v", err)
	}
	if got := scalarInt(t, pool, `SELECT count(*) FROM ttg.input_events WHERE session_id = $1`, session.ID); got != 1 {
		t.Fatalf("committed rows = %d", got)
	}
	looked, found, err := appender.LookupCandidate(context.Background(), candidate)
	if err != nil || !found || looked.InputSeq != 1 {
		t.Fatalf("lookup = %+v/%v/%v", looked, found, err)
	}
	appender.afterCommit = nil
	result, err := appender.AppendSession(context.Background(), session.ID, []DurableCandidate{candidate})
	if err != nil || !result[0].Duplicate || result[0].InputSeq != 1 {
		t.Fatalf("retry after ambiguous commit = %+v/%v", result, err)
	}
}

func TestDurableRetryOnSerializationError(t *testing.T) {
	pool := testPool(t)
	session := createSession(t, pool, "mock", "running", 1, 80, 0)
	appender := newTestAppender(t, pool, WithMaxRetries(3))
	attempts := 0
	appender.beforeCommit = func() error {
		attempts++
		if attempts == 1 {
			return &pgconn.PgError{Code: "40001", Message: "serialization failure"}
		}
		return nil
	}
	result, err := appender.AppendSession(context.Background(), session.ID, []DurableCandidate{
		candidateFor(session.ID, "retry-1", actorFor(12), mustAction(t, "XÂY")),
	})
	if err != nil || result[0].InputSeq != 1 || attempts != 2 {
		t.Fatalf("retry result=%+v err=%v attempts=%d", result, err, attempts)
	}
	if got := scalarInt(t, pool, `SELECT count(*) FROM ttg.input_events WHERE session_id = $1`, session.ID); got != 1 {
		t.Fatalf("retry left %d events", got)
	}
}

func TestDurableRetryExhaustion(t *testing.T) {
	pool := testPool(t)
	session := createSession(t, pool, "mock", "running", 1, 90, 0)
	appender := newTestAppender(t, pool, WithMaxRetries(2))
	appender.beforeCommit = func() error {
		return &pgconn.PgError{Code: "40P01", Message: "deadlock detected"}
	}
	_, err := appender.AppendSession(context.Background(), session.ID, []DurableCandidate{
		candidateFor(session.ID, "retry-exhaust", actorFor(13), mustAction(t, "XÂY")),
	})
	if !errors.Is(err, ErrRetryExhausted) {
		t.Fatalf("retry exhaustion error = %v", err)
	}
	if got := scalarInt(t, pool, `SELECT count(*) FROM ttg.input_events WHERE session_id = $1`, session.ID); got != 0 {
		t.Fatalf("retry exhaustion left %d events", got)
	}
}

func TestDurableSequenceOverflow(t *testing.T) {
	pool := testPool(t)
	session := createSession(t, pool, "mock", "running", 1, 100, math.MaxInt64-1)
	appender := newTestAppender(t, pool)
	_, err := appender.AppendSession(context.Background(), session.ID, []DurableCandidate{
		candidateFor(session.ID, "overflow-1", actorFor(14), mustAction(t, "XÂY")),
		candidateFor(session.ID, "overflow-2", actorFor(14), mustAction(t, "XÂY")),
	})
	if !errors.Is(err, ErrSequenceOverflow) {
		t.Fatalf("overflow error = %v", err)
	}
	if got := scalarInt(t, pool, `SELECT count(*) FROM ttg.input_events WHERE session_id = $1`, session.ID); got != 0 {
		t.Fatalf("overflow wrote %d events", got)
	}
}

func TestDurableStorageUnavailable(t *testing.T) {
	pool, err := storage.Open(context.Background(), storage.Options{
		DSN: "postgres://app:pw@127.0.0.1:1/ttgqldl?sslmode=disable", ConnectTimeout: 300 * time.Millisecond,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	appender, err := NewPostgresAppender(pool)
	if err != nil {
		t.Fatal(err)
	}
	unavailableSession := testUUID(t)
	_, err = appender.AppendSession(context.Background(), unavailableSession, []DurableCandidate{
		candidateFor(unavailableSession, "unavailable", actorFor(15), mustAction(t, "XÂY")),
	})
	if !errors.Is(err, ErrStorageUnavailable) {
		t.Fatalf("storage unavailable error = %v", err)
	}
}

func TestDurablePrivacyAndPayloadShape(t *testing.T) {
	pool := testPool(t)
	session := createSession(t, pool, "mock", "running", 1, 110, 0)
	appender := newTestAppender(t, pool)
	rawViewer := "provider-viewer-secret-036"
	rawComment := "XÂY with a raw comment that must not persist"
	action := mustAction(t, "XÂY")
	candidate := candidateFor(session.ID, "privacy-1", actorFor(16), action)
	// The raw provider viewer id/comment are not fields of DurableCandidate by design.
	_ = rawViewer
	_ = rawComment
	if _, err := appender.AppendSession(context.Background(), session.ID, []DurableCandidate{candidate}); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var payload, actor string
	if err := tx.QueryRow(ctx, `SELECT canonical_payload::text, actor_key FROM ttg.input_events WHERE session_id = $1`, session.ID).Scan(&payload, &actor); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(payload, "XÂY") || strings.Contains(payload, rawComment) || strings.Contains(payload, rawViewer) {
		t.Fatalf("payload leaked raw text: %s", payload)
	}
	if actor != string(candidate.ActorKey) {
		t.Fatalf("actor key = %q", actor)
	}
	if got := scalarInt(t, pool, `SELECT count(*) FROM information_schema.columns WHERE table_schema = 'ttg' AND table_name = 'input_events' AND column_name IN ('provider_viewer_id','message','alias')`); got != 0 {
		t.Fatalf("input_events has raw identity columns")
	}
}
