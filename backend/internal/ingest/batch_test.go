package ingest

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hung98dev/ttgqldl/backend/internal/source"
)

type fakeAppender struct {
	mu          sync.Mutex
	calls       int
	block       chan struct{}
	entered     chan struct{}
	enteredOnce sync.Once
	err         error
	sequence    atomic.Int64
}

func (f *fakeAppender) AppendSession(_ context.Context, _ string, candidates []DurableCandidate) ([]DurableResult, error) {
	f.mu.Lock()
	f.calls++
	f.mu.Unlock()
	if f.entered != nil {
		f.enteredOnce.Do(func() { close(f.entered) })
	}
	if f.block != nil {
		<-f.block
	}
	if f.err != nil {
		return nil, f.err
	}
	results := make([]DurableResult, len(candidates))
	for index := range results {
		results[index] = DurableResult{InputSeq: f.sequence.Add(1), WaveNo: 1, ScheduledTick: 1}
	}
	return results, nil
}

func (f *fakeAppender) LookupReceipt(context.Context, string, string, string) (DurableResult, bool, error) {
	return DurableResult{}, false, nil
}

func (f *fakeAppender) LookupCandidate(context.Context, DurableCandidate) (DurableResult, bool, error) {
	return DurableResult{}, false, nil
}

func testBatcher(t *testing.T, appender SessionAppender, mutate func(*BatcherConfig)) *Batcher {
	t.Helper()
	config := DefaultBatcherConfig()
	config.FlushInterval = 5 * time.Millisecond
	if mutate != nil {
		mutate(&config)
	}
	batcher, err := NewBatcher(appender, config)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = batcher.Close(ctx)
	})
	return batcher
}

func submitAsync(batcher *Batcher, candidate DurableCandidate, reservation *Reservation) <-chan submitResponse {
	result := make(chan submitResponse, 1)
	go func() {
		durable, err := batcher.Submit(context.Background(), candidate, reservation)
		result <- submitResponse{result: durable, err: err}
	}()
	return result
}

func TestBatcherFullBatch256(t *testing.T) {
	pool := testPool(t)
	session := createSession(t, pool, "mock", "running", 1, 120, 0)
	appender := newTestAppender(t, pool)
	batcher := testBatcher(t, appender, func(config *BatcherConfig) {
		config.MaxBatch = 256
		config.FlushInterval = time.Second
	})
	actor := actorFor(20)
	action := mustAction(t, "XÂY")
	results := make([]<-chan submitResponse, 256)
	for index := range results {
		results[index] = submitAsync(batcher, candidateFor(session.ID, fmt.Sprintf("full-%d", index), actor, action), nil)
	}
	for index, channel := range results {
		response := <-channel
		if response.err != nil || response.result.InputSeq == 0 {
			t.Fatalf("submit %d = %+v/%v", index, response.result, response.err)
		}
	}
	stats := batcher.Stats()
	if stats.Batches != 1 || stats.Transactions != 1 || stats.Inputs != 256 || stats.MaxBatch != 256 {
		t.Fatalf("stats = %+v", stats)
	}
	if got := scalarInt(t, pool, `SELECT count(*) FROM ttg.input_events WHERE session_id = $1`, session.ID); got != 256 {
		t.Fatalf("events = %d", got)
	}
}

func TestBatcherSplit257(t *testing.T) {
	pool := testPool(t)
	session := createSession(t, pool, "mock", "running", 1, 130, 0)
	appender := newTestAppender(t, pool)
	timer := make(chan time.Time, 1)
	batcher := testBatcher(t, appender, func(config *BatcherConfig) {
		config.MaxBatch = 256
		config.FlushInterval = 50 * time.Millisecond
		config.After = func(time.Duration) <-chan time.Time { return timer }
	})
	actor := actorFor(21)
	action := mustAction(t, "THỦ")
	results := make([]<-chan submitResponse, 257)
	for index := range results {
		results[index] = submitAsync(batcher, candidateFor(session.ID, fmt.Sprintf("split-%d", index), actor, action), nil)
	}
	deadline := time.Now().Add(5 * time.Second)
	for batcher.Stats().Inputs < 256 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	timer <- time.Now()
	for index, channel := range results {
		if response := <-channel; response.err != nil {
			t.Fatalf("submit %d: %v", index, response.err)
		}
	}
	stats := batcher.Stats()
	if stats.Batches != 2 || stats.Inputs != 257 || stats.MaxBatch != 256 {
		t.Fatalf("stats = %+v", stats)
	}
}

func TestBatcherTimedFlush255WithInjectedTimer(t *testing.T) {
	pool := testPool(t)
	session := createSession(t, pool, "mock", "running", 1, 140, 0)
	appender := newTestAppender(t, pool)
	timer := make(chan time.Time, 1)
	batcher := testBatcher(t, appender, func(config *BatcherConfig) {
		config.MaxBatch = 256
		config.FlushInterval = 50 * time.Millisecond
		config.After = func(time.Duration) <-chan time.Time { return timer }
	})
	actor := actorFor(22)
	action := mustAction(t, "ĐÁNH")
	results := make([]<-chan submitResponse, 255)
	for index := range results {
		results[index] = submitAsync(batcher, candidateFor(session.ID, fmt.Sprintf("timed-%d", index), actor, action), nil)
	}
	deadline := time.Now().Add(5 * time.Second)
	for batcher.Stats().Submitted < 255 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	timer <- time.Now()
	for index, channel := range results {
		if response := <-channel; response.err != nil {
			t.Fatalf("submit %d: %v", index, response.err)
		}
	}
	stats := batcher.Stats()
	if stats.Batches != 1 || stats.Inputs != 255 {
		t.Fatalf("stats = %+v", stats)
	}
}

func TestBatcherRealDeadline(t *testing.T) {
	pool := testPool(t)
	session := createSession(t, pool, "mock", "running", 1, 150, 0)
	appender := newTestAppender(t, pool)
	batcher := testBatcher(t, appender, func(config *BatcherConfig) {
		config.MaxBatch = 256
		config.FlushInterval = 50 * time.Millisecond
	})
	actor := actorFor(23)
	action := mustAction(t, "BUILD")
	started := time.Now()
	results := make([]<-chan submitResponse, 10)
	for index := range results {
		results[index] = submitAsync(batcher, candidateFor(session.ID, fmt.Sprintf("deadline-%d", index), actor, action), nil)
	}
	for index, channel := range results {
		if response := <-channel; response.err != nil {
			t.Fatalf("submit %d: %v", index, response.err)
		}
	}
	elapsed := time.Since(started)
	if elapsed < 40*time.Millisecond || elapsed > 2*time.Second {
		t.Fatalf("timed flush latency = %s", elapsed)
	}
	if stats := batcher.Stats(); stats.Batches != 1 || stats.Inputs != 10 {
		t.Fatalf("stats = %+v", stats)
	}
}

func TestBatcherCloseFlushesQueue(t *testing.T) {
	pool := testPool(t)
	session := createSession(t, pool, "mock", "running", 1, 160, 0)
	appender := newTestAppender(t, pool)
	batcher, err := NewBatcher(appender, BatcherConfig{
		QueueCapacity: 32, MaxBatch: 256, FlushInterval: time.Second, After: func(time.Duration) <-chan time.Time { return nil },
	})
	if err != nil {
		t.Fatal(err)
	}
	actor := actorFor(24)
	action := mustAction(t, "DEFEND")
	results := make([]<-chan submitResponse, 10)
	for index := range results {
		results[index] = submitAsync(batcher, candidateFor(session.ID, fmt.Sprintf("close-%d", index), actor, action), nil)
	}
	deadline := time.Now().Add(2 * time.Second)
	for batcher.Stats().Submitted < 10 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := batcher.Close(ctx); err != nil {
		t.Fatalf("close: %v", err)
	}
	for index, channel := range results {
		if response := <-channel; response.err != nil {
			t.Fatalf("submit %d after close: %v", index, response.err)
		}
	}
	if stats := batcher.Stats(); stats.Inputs != 10 {
		t.Fatalf("stats = %+v", stats)
	}
	if _, err := batcher.Submit(context.Background(), DurableCandidate{}, nil); !errors.Is(err, ErrBatcherClosed) {
		t.Fatalf("submit after close = %v", err)
	}
}

func TestBatcherQueueFullBackpressure(t *testing.T) {
	block := make(chan struct{})
	entered := make(chan struct{})
	appender := &fakeAppender{block: block, entered: entered}
	batcher := testBatcher(t, appender, func(config *BatcherConfig) {
		config.QueueCapacity = 2
		config.MaxBatch = 1
		config.FlushInterval = time.Second
	})
	first := submitAsync(batcher, DurableCandidate{SessionID: "s", Provider: "mock", ProviderEventID: "q1", ActorKey: actorFor(25), Action: mustAction(t, "XÂY")}, nil)
	<-entered
	second := submitAsync(batcher, DurableCandidate{SessionID: "s", Provider: "mock", ProviderEventID: "q2", ActorKey: actorFor(25), Action: mustAction(t, "XÂY")}, nil)
	third := submitAsync(batcher, DurableCandidate{SessionID: "s", Provider: "mock", ProviderEventID: "q3", ActorKey: actorFor(25), Action: mustAction(t, "XÂY")}, nil)
	deadline := time.Now().Add(2 * time.Second)
	for batcher.Stats().QueueDepth < 2 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if _, err := batcher.Submit(context.Background(), DurableCandidate{SessionID: "s", Provider: "mock", ProviderEventID: "q4", ActorKey: actorFor(25), Action: mustAction(t, "XÂY")}, nil); !errors.Is(err, ErrQueueFull) {
		t.Fatalf("queue full error = %v", err)
	}
	close(block)
	for _, channel := range []<-chan submitResponse{first, second, third} {
		if response := <-channel; response.err != nil {
			t.Fatalf("queued submit: %v", response.err)
		}
	}
	if stats := batcher.Stats(); stats.RejectedQueue != 1 {
		t.Fatalf("stats = %+v", stats)
	}
}

func TestBatcherCommitUnknownReconciles(t *testing.T) {
	pool := testPool(t)
	session := createSession(t, pool, "mock", "running", 1, 170, 0)
	appender := newTestAppender(t, pool)
	appender.afterCommit = func() error { return errors.New("ack lost") }
	batcher := testBatcher(t, appender, nil)
	result, err := batcher.Submit(context.Background(), candidateFor(session.ID, "batch-unknown", actorFor(26), mustAction(t, "XÂY")), nil)
	if err != nil {
		t.Fatalf("submit: %v", err)
	}
	if result.InputSeq != 1 {
		t.Fatalf("result = %+v", result)
	}
	stats := batcher.Stats()
	if stats.CommitUnknown != 1 || stats.Committed != 1 || stats.Inputs != 1 {
		t.Fatalf("stats = %+v", stats)
	}
	if got := scalarInt(t, pool, `SELECT count(*) FROM ttg.input_events WHERE session_id = $1`, session.ID); got != 1 {
		t.Fatalf("events = %d", got)
	}
}

func TestBatcherConflictAndReservationLifecycle(t *testing.T) {
	pool := testPool(t)
	session := createSession(t, pool, "mock", "running", 1, 180, 0)
	appender := newTestAppender(t, pool)
	batcher := testBatcher(t, appender, nil)
	actor := actorFor(27)
	builder := mustAction(t, "XÂY")
	fighter := mustAction(t, "ATTACK")
	if _, err := batcher.Submit(context.Background(), candidateFor(session.ID, "batch-conflict", actor, builder), nil); err != nil {
		t.Fatal(err)
	}
	conflict, err := batcher.Submit(context.Background(), candidateFor(session.ID, "batch-conflict", actor, fighter), nil)
	if err != nil || !conflict.Conflict {
		t.Fatalf("conflict = %+v/%v", conflict, err)
	}
	if stats := batcher.Stats(); stats.Conflicts != 1 || stats.Inputs != 1 {
		t.Fatalf("stats = %+v", stats)
	}

	// A definitive storage failure releases the W0304 reservation so retry remains
	// possible; the source cursor is not acknowledged.
	deduper, _ := newTestDeduper(t, nil)
	event := commentEvent("session-a", "mock", "reservation-batch")
	reservation := mustAccept(t, deduper, event, actorFor(28), builder)
	failing := &fakeAppender{err: ErrStorageUnavailable}
	failingBatcher := testBatcher(t, failing, nil)
	if _, err := failingBatcher.Submit(context.Background(), DurableCandidate{
		SessionID: "session-a", Provider: "mock", ProviderEventID: "reservation-batch",
		Fingerprint: payloadFingerprint(actorFor(28), source.KindComment, builder),
		ActorKey:    actorFor(28), Action: builder,
	}, reservation); !errors.Is(err, ErrStorageUnavailable) {
		t.Fatalf("failure = %v", err)
	}
	retry, _, err := deduper.Admit(event, actorFor(28), builder)
	if err != nil || retry.Class != ClassAccepted {
		t.Fatalf("reservation was not released: %+v/%v", retry, err)
	}
}
