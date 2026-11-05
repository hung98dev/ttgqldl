package ingest

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/hung98dev/ttgqldl/backend/internal/commands"
	"github.com/hung98dev/ttgqldl/backend/internal/domain"
	"github.com/hung98dev/ttgqldl/backend/internal/identity"
	"github.com/hung98dev/ttgqldl/backend/internal/source"
)

type populationSubmission struct {
	candidate   DurableCandidate
	reservation *Reservation
	ack         chan error
}

// TestPopulationTenThousandDurable validates the 10,000-participant source generation,
// source validation, scoped HMAC identity, parser, W0304 dedupe and the real W0305
// batcher/PostgreSQL boundary. It deliberately uses the direct batcher boundary rather
// than Mock.Start: the mock sink is synchronous and acknowledges one event at a time, so
// a serial source callback would flush one-row batches and take minutes. Source cursor
// and ACK ordering are covered separately by TestPopulationSourceCursorContract.
func TestPopulationTenThousandDurable(t *testing.T) {
	pool := testPool(t)
	session := createSession(t, pool, "mock", "running", 1, 310, 0)
	mockSession := source.Session{ID: session.ID, Mode: source.ModeMock}
	const count = 10_000
	const seed = uint64(20261105)
	events, err := source.ScenarioEvents(mockSession, seed, source.ScenarioMultipleParticipants, count)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != count {
		t.Fatalf("events = %d", len(events))
	}
	key, err := identity.NewKey("v1", []byte("0123456789abcdef0123456789abcdef"))
	if err != nil {
		t.Fatal(err)
	}
	keys, err := identity.NewKeySet(key, nil)
	if err != nil {
		t.Fatal(err)
	}
	metrics := NewMetrics()
	deduper, err := New(Config{
		Now:        func() time.Time { return time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC) },
		Metrics:    metrics,
		MaxEntries: 20_000,
		MaxActors:  20_000,
	})
	if err != nil {
		t.Fatal(err)
	}
	appender := newTestAppender(t, pool)
	batcher, err := NewBatcher(appender, BatcherConfig{
		QueueCapacity: 20_000, MaxBatch: 256, FlushInterval: 50 * time.Millisecond,
		Metrics: metrics, Now: time.Now,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		closeCtx, closeCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer closeCancel()
		_ = batcher.Close(closeCtx)
	}()

	const workers = 256
	jobs := make(chan populationSubmission, workers)
	var workerGroup sync.WaitGroup
	workerGroup.Add(workers)
	for worker := 0; worker < workers; worker++ {
		go func() {
			defer workerGroup.Done()
			for job := range jobs {
				_, err := batcher.Submit(context.Background(), job.candidate, job.reservation)
				job.ack <- err
			}
		}()
	}

	actors := make(map[domain.ActorKey]struct{}, count)
	prepared := make([]populationSubmission, 0, count)
	prepareStart := time.Now()
	for index, event := range events {
		metrics.ObserveReceived()
		if err := source.ValidateEvent(event, mockSession, source.ModeMock); err != nil {
			t.Fatalf("validate %d: %v", index, err)
		}
		metrics.ObserveValidated()
		resolved, err := keys.Derive(identity.Input{
			Provider: event.Provider, ProviderVersion: "v1",
			BroadcasterKey: "village-w0307", ViewerID: event.ProviderViewerID,
		})
		if err != nil {
			t.Fatalf("identity %d: %v", index, err)
		}
		if _, exists := actors[resolved.ActorKey]; exists {
			t.Fatalf("duplicate actor key at %d", index)
		}
		actors[resolved.ActorKey] = struct{}{}
		action, err := commands.ParseComment(event.Message)
		if err != nil {
			t.Fatalf("parse %d: %v", index, err)
		}
		decision, reservation, err := deduper.Admit(event, resolved.ActorKey, action)
		if err != nil || decision.Class != ClassAccepted {
			t.Fatalf("admit %d = %+v/%v", index, decision, err)
		}
		prepared = append(prepared, populationSubmission{
			candidate:   candidateFor(event.SourceSession, event.ProviderEventID, resolved.ActorKey, action),
			reservation: reservation,
			ack:         make(chan error, 1),
		})
	}
	prepareElapsed := time.Since(prepareStart)
	durableStart := time.Now()
	for _, submission := range prepared {
		jobs <- submission
	}
	close(jobs)
	workerGroup.Wait()
	for index, submission := range prepared {
		if err := <-submission.ack; err != nil {
			t.Fatalf("durable submit %d: %v", index, err)
		}
	}
	elapsed := time.Since(durableStart)
	if len(actors) != count {
		t.Fatalf("unique actor keys = %d", len(actors))
	}
	if stats := deduper.Stats(); stats.StableEntries != count || stats.Pending != 0 {
		t.Fatalf("dedupe stats = %+v", stats)
	}
	snapshot := metrics.SnapshotAt(time.Now())
	if snapshot.InputReceived != count || snapshot.InputValidated != count ||
		snapshot.AcceptedCandidates != count || snapshot.Enqueued != count ||
		snapshot.DurableCommitted != count || snapshot.Inputs != count {
		t.Fatalf("metrics stages = %+v", snapshot)
	}
	if snapshot.DurableDuplicates != 0 || snapshot.DurableConflicts != 0 || snapshot.LostGameEvents != 0 ||
		snapshot.QueueOverflow != 0 || snapshot.BatchFailures != 0 || snapshot.CallerCancelled != 0 {
		t.Fatalf("unexpected rejection/loss metrics = %+v", snapshot)
	}
	if snapshot.QueueCapacity != 20_000 || snapshot.MaxBatch > 256 || snapshot.Alert.Level == AlertDegraded {
		t.Fatalf("queue/alert metrics = %+v", snapshot)
	}

	if got := scalarInt(t, pool, `SELECT count(*) FROM ttg.input_events WHERE session_id = $1`, session.ID); got != count {
		t.Fatalf("input_events = %d", got)
	}
	if got := scalarInt(t, pool, `SELECT count(*) FROM ttg.input_receipts WHERE session_id = $1`, session.ID); got != count {
		t.Fatalf("input_receipts = %d", got)
	}
	if got := scalarInt(t, pool, `SELECT count(DISTINCT input_seq) FROM ttg.input_events WHERE session_id = $1`, session.ID); got != count {
		t.Fatalf("distinct input_seq = %d", got)
	}
	if got := scalarInt(t, pool, `SELECT min(input_seq) FROM ttg.input_events WHERE session_id = $1`, session.ID); got != 1 {
		t.Fatalf("min input_seq = %d", got)
	}
	if got := scalarInt(t, pool, `SELECT max(input_seq) FROM ttg.input_events WHERE session_id = $1`, session.ID); got != count {
		t.Fatalf("max input_seq = %d", got)
	}
	if got := scalarInt(t, pool, `SELECT count(DISTINCT actor_key) FROM ttg.input_events WHERE session_id = $1`, session.ID); got != count {
		t.Fatalf("distinct actor_key = %d", got)
	}
	if got := scalarInt(t, pool, `SELECT count(*) FROM ttg.input_events WHERE session_id = $1 AND applied_at_tick IS NULL`, session.ID); got != count {
		t.Fatalf("unapplied inputs = %d", got)
	}
	if got := scalarInt(t, pool, `SELECT input_sequence FROM ttg.sessions WHERE id = $1`, session.ID); got != count {
		t.Fatalf("session input_sequence = %d", got)
	}
	if got := scalarInt(t, pool, `SELECT count(*) FROM ttg.input_events WHERE session_id = $1 AND (canonical_payload::text LIKE '%mock-viewer-%' OR canonical_payload::text LIKE '%XÂY%' OR canonical_payload::text LIKE '%THỦ%')`, session.ID); got != 0 {
		t.Fatalf("raw source data leaked into input_events: %d", got)
	}
	t.Logf("10k durable: prepare=%s durable=%s batches=%d transactions=%d max_batch=%d queue_high=%d latency_p95_ms=%.0f",
		prepareElapsed.Round(time.Millisecond), elapsed.Round(time.Millisecond), snapshot.Batches, snapshot.Transactions,
		snapshot.MaxBatch, snapshot.QueueHighWater, snapshot.LatencyP95Millis)
}

// TestPopulationSourceCursorContract proves the synchronous source ACK/cursor contract
// with the real batcher and a fake durable appender at 1,000 events. It is intentionally
// separate from the direct 10k batcher boundary: the mock sink advances its cursor only
// after the batcher returns a durable result.
func TestPopulationSourceCursorContract(t *testing.T) {
	session := source.Session{ID: "synthetic-source-cursor-w0307", Mode: source.ModeMock}
	const count = 1_000
	events, err := source.ScenarioEvents(session, 20261105, source.ScenarioMultipleParticipants, count)
	if err != nil {
		t.Fatal(err)
	}
	mock, err := source.NewMockWithEvents(session, events)
	if err != nil {
		t.Fatal(err)
	}
	key, err := identity.NewKey("v1", []byte("0123456789abcdef0123456789abcdef"))
	if err != nil {
		t.Fatal(err)
	}
	keys, err := identity.NewKeySet(key, nil)
	if err != nil {
		t.Fatal(err)
	}
	metrics := NewMetrics()
	deduper, err := New(Config{Now: func() time.Time { return time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC) }, Metrics: metrics, MaxEntries: 2_000, MaxActors: 2_000})
	if err != nil {
		t.Fatal(err)
	}
	batcher, err := NewBatcher(&fakeAppender{}, BatcherConfig{
		QueueCapacity: 2_000, MaxBatch: 1, FlushInterval: time.Millisecond, Metrics: metrics, Now: time.Now,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = batcher.Close(ctx)
	}()
	sink := func(ctx context.Context, event source.Event) error {
		metrics.ObserveReceived()
		if err := source.ValidateEvent(event, session, source.ModeMock); err != nil {
			return err
		}
		metrics.ObserveValidated()
		resolved, err := keys.Derive(identity.Input{Provider: event.Provider, ProviderVersion: "v1", BroadcasterKey: "village-w0307", ViewerID: event.ProviderViewerID})
		if err != nil {
			return err
		}
		action, err := commands.ParseComment(event.Message)
		if err != nil {
			return err
		}
		decision, reservation, err := deduper.Admit(event, resolved.ActorKey, action)
		if err != nil || decision.Class != ClassAccepted {
			return fmt.Errorf("admit: %+v/%v", decision, err)
		}
		_, err = batcher.Submit(ctx, candidateFor(event.SourceSession, event.ProviderEventID, resolved.ActorKey, action), reservation)
		return err
	}
	if err := mock.Start(context.Background(), session, sink); err != nil {
		t.Fatal(err)
	}
	health := mock.Health(context.Background())
	if health.Cursor.NextEvent != count || health.Delivered != count {
		t.Fatalf("cursor = %+v", health)
	}
	snapshot := metrics.Snapshot()
	if snapshot.InputReceived != count || snapshot.InputValidated != count ||
		snapshot.AcceptedCandidates != count || snapshot.DurableCommitted != count ||
		snapshot.LostGameEvents != 0 || snapshot.QueueOverflow != 0 {
		t.Fatalf("source cursor metrics = %+v", snapshot)
	}
}

func TestPopulationRedeliveryAfterRestart(t *testing.T) {
	pool := testPool(t)
	session := createSession(t, pool, "mock", "running", 1, 320, 0)
	mockSession := source.Session{ID: session.ID, Mode: source.ModeMock}
	const count = 100
	events, err := source.ScenarioEvents(mockSession, 20261105, source.ScenarioMultipleParticipants, count)
	if err != nil {
		t.Fatal(err)
	}
	key, err := identity.NewKey("v1", []byte("0123456789abcdef0123456789abcdef"))
	if err != nil {
		t.Fatal(err)
	}
	keys, err := identity.NewKeySet(key, nil)
	if err != nil {
		t.Fatal(err)
	}
	appender := newTestAppender(t, pool)
	firstMetrics := NewMetrics()
	firstDeduper, err := New(Config{Now: func() time.Time { return time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC) }, Metrics: firstMetrics, MaxEntries: 1000, MaxActors: 1000})
	if err != nil {
		t.Fatal(err)
	}
	firstBatcher, err := NewBatcher(appender, BatcherConfig{QueueCapacity: 100, MaxBatch: 1, FlushInterval: time.Millisecond, Metrics: firstMetrics, Now: time.Now})
	if err != nil {
		t.Fatal(err)
	}
	for _, event := range events {
		resolved, err := keys.Derive(identity.Input{Provider: event.Provider, ProviderVersion: "v1", BroadcasterKey: "village-w0307", ViewerID: event.ProviderViewerID})
		if err != nil {
			t.Fatal(err)
		}
		action, err := commands.ParseComment(event.Message)
		if err != nil {
			t.Fatal(err)
		}
		decision, reservation, err := firstDeduper.Admit(event, resolved.ActorKey, action)
		if err != nil || decision.Class != ClassAccepted {
			t.Fatalf("first admit = %+v/%v", decision, err)
		}
		if _, err := firstBatcher.Submit(context.Background(), candidateFor(event.SourceSession, event.ProviderEventID, resolved.ActorKey, action), reservation); err != nil {
			t.Fatal(err)
		}
	}
	if err := firstBatcher.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
	if got := scalarInt(t, pool, `SELECT count(*) FROM ttg.input_events WHERE session_id = $1`, session.ID); got != count {
		t.Fatalf("first input_events = %d", got)
	}

	// Fresh process/deduper: PostgreSQL receipts, not memory, must reject redelivery.
	secondMetrics := NewMetrics()
	secondDeduper, err := New(Config{Now: func() time.Time { return time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC) }, Metrics: secondMetrics, MaxEntries: 1000, MaxActors: 1000})
	if err != nil {
		t.Fatal(err)
	}
	secondBatcher, err := NewBatcher(appender, BatcherConfig{QueueCapacity: 100, MaxBatch: 1, FlushInterval: time.Millisecond, Metrics: secondMetrics, Now: time.Now})
	if err != nil {
		t.Fatal(err)
	}
	duplicates := 0
	for _, event := range events {
		resolved, err := keys.Derive(identity.Input{Provider: event.Provider, ProviderVersion: "v1", BroadcasterKey: "village-w0307", ViewerID: event.ProviderViewerID})
		if err != nil {
			t.Fatal(err)
		}
		action, err := commands.ParseComment(event.Message)
		if err != nil {
			t.Fatal(err)
		}
		decision, reservation, err := secondDeduper.Admit(event, resolved.ActorKey, action)
		if err != nil {
			t.Fatalf("second admit = %+v/%v", decision, err)
		}
		if decision.Class == ClassDuplicateProviderID {
			reservation = nil
		} else if decision.Class != ClassAccepted {
			t.Fatalf("second admit = %+v", decision)
		}
		result, err := secondBatcher.Submit(context.Background(), candidateFor(event.SourceSession, event.ProviderEventID, resolved.ActorKey, action), reservation)
		if err != nil || !result.Duplicate {
			t.Fatalf("redelivery result = %+v/%v", result, err)
		}
		duplicates++
	}
	if err := secondBatcher.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
	if duplicates != count {
		t.Fatalf("duplicates = %d", duplicates)
	}
	if got := scalarInt(t, pool, `SELECT count(*) FROM ttg.input_events WHERE session_id = $1`, session.ID); got != count {
		t.Fatalf("redelivery inserted rows: %d", got)
	}
	if got := scalarInt(t, pool, `SELECT max(input_seq) FROM ttg.input_events WHERE session_id = $1`, session.ID); got != count {
		t.Fatalf("redelivery allocated sequence: %d", got)
	}
	if got := scalarInt(t, pool, `SELECT count(*) FROM ttg.input_receipts WHERE session_id = $1`, session.ID); got != count {
		t.Fatalf("receipts changed: %d", got)
	}
	snapshot := secondMetrics.Snapshot()
	if snapshot.DurableDuplicates != count || snapshot.DurableCommitted != 0 || snapshot.LostGameEvents != 0 {
		t.Fatalf("redelivery metrics = %+v", snapshot)
	}
}
