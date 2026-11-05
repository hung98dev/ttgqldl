package ingest

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// SessionAppender is the narrow durable boundary consumed by the batcher. The
// PostgreSQL implementation is PostgresAppender; tests may use a fake for queue and
// timer behavior, but durable guarantees require real PostgreSQL.
type SessionAppender interface {
	AppendSession(ctx context.Context, sessionID string, candidates []DurableCandidate) ([]DurableResult, error)
	LookupReceipt(ctx context.Context, sessionID, provider, eventID string) (DurableResult, bool, error)
	LookupCandidate(ctx context.Context, candidate DurableCandidate) (DurableResult, bool, error)
}

// BatcherConfig is explicit and bounded.
type BatcherConfig struct {
	QueueCapacity int
	MaxBatch      int
	FlushInterval time.Duration
	// After supplies the flush timer; nil uses time.After. Tests inject a controlled
	// channel to prove deadline behavior without sleeping.
	After func(time.Duration) <-chan time.Time
	// Now supplies the receive-to-commit latency clock; nil uses time.Now.
	Now func() time.Time
	// Metrics optionally records W0306 queue and durable stages.
	Metrics *Metrics
}

// DefaultBatcherConfig returns the reviewed W0305 limits.
func DefaultBatcherConfig() BatcherConfig {
	return BatcherConfig{
		QueueCapacity: 20_000,
		MaxBatch:      256,
		FlushInterval: 50 * time.Millisecond,
		After:         time.After,
		Now:           time.Now,
	}
}

func (c BatcherConfig) normalized() BatcherConfig {
	if c.QueueCapacity == 0 {
		c.QueueCapacity = 20_000
	}
	if c.MaxBatch == 0 {
		c.MaxBatch = 256
	}
	if c.FlushInterval == 0 {
		c.FlushInterval = 50 * time.Millisecond
	}
	if c.After == nil {
		c.After = time.After
	}
	if c.Now == nil {
		c.Now = time.Now
	}
	return c
}

func (c BatcherConfig) validate() error {
	if c.QueueCapacity < 1 || c.QueueCapacity > 1_000_000 {
		return fmt.Errorf("%w: queue capacity", ErrInvalidInput)
	}
	if c.MaxBatch < 1 || c.MaxBatch > 256 {
		return fmt.Errorf("%w: max batch must be 1..256", ErrInvalidInput)
	}
	if c.FlushInterval <= 0 || c.FlushInterval > 5*time.Second {
		return fmt.Errorf("%w: flush interval", ErrInvalidInput)
	}
	return nil
}

type submitRequest struct {
	candidate   DurableCandidate
	reservation *Reservation
	enqueuedAt  time.Time
	result      chan submitResponse
}

type submitResponse struct {
	result DurableResult
	err    error
}

// BatcherStats is a diagnostic snapshot. It never contains identities or event ids.
type BatcherStats struct {
	Submitted     int64
	Committed     int64
	Duplicates    int64
	Conflicts     int64
	Inputs        int64
	Batches       int64
	Transactions  int64
	RejectedQueue int64
	Errors        int64
	CommitUnknown int64
	QueueDepth    int64
	QueueHighMark int64
	MaxBatch      int64
}

// Batcher is a bounded FIFO queue with one worker. A batch flushes at MaxBatch or at the
// FlushInterval deadline, whichever comes first. Submissions wait for the durable result;
// no acknowledgment is sent before commit.
type Batcher struct {
	appender  SessionAppender
	cfg       BatcherConfig
	metrics   *Metrics
	queue     chan submitRequest
	done      chan struct{}
	closed    atomic.Bool
	closeOnce sync.Once
	// mu serializes the closed-check/enqueue in Submit against Close closing the queue.
	mu sync.RWMutex

	submitted     atomic.Int64
	committed     atomic.Int64
	duplicates    atomic.Int64
	conflicts     atomic.Int64
	inputs        atomic.Int64
	batches       atomic.Int64
	transactions  atomic.Int64
	rejectedQueue atomic.Int64
	errors        atomic.Int64
	commitUnknown atomic.Int64
	queueHighMark atomic.Int64
	maxBatch      atomic.Int64
}

// NewBatcher starts one bounded worker. Close must be called to stop it.
func NewBatcher(appender SessionAppender, config BatcherConfig) (*Batcher, error) {
	if appender == nil {
		return nil, fmt.Errorf("%w: appender is required", ErrInvalidInput)
	}
	config = config.normalized()
	if err := config.validate(); err != nil {
		return nil, err
	}
	batcher := &Batcher{
		appender: appender,
		cfg:      config,
		metrics:  config.Metrics,
		queue:    make(chan submitRequest, config.QueueCapacity),
		done:     make(chan struct{}),
	}
	if batcher.metrics != nil {
		batcher.metrics.ObserveQueue(0, config.QueueCapacity, config.Now())
	}
	go batcher.run()
	return batcher, nil
}

// Submit enqueues one accepted candidate and waits for the durable result. A full queue
// is an explicit fail-closed backpressure result; nothing is silently dropped.
func (b *Batcher) Submit(ctx context.Context, candidate DurableCandidate, reservation *Reservation) (DurableResult, error) {
	if b == nil {
		return DurableResult{}, ErrBatcherClosed
	}
	request := submitRequest{
		candidate: candidate, reservation: reservation,
		enqueuedAt: b.cfg.Now(), result: make(chan submitResponse, 1),
	}
	b.mu.RLock()
	if b.closed.Load() {
		b.mu.RUnlock()
		return DurableResult{}, ErrBatcherClosed
	}
	select {
	case b.queue <- request:
		b.mu.RUnlock()
		b.submitted.Add(1)
		depth := int64(len(b.queue))
		for {
			current := b.queueHighMark.Load()
			if depth <= current || b.queueHighMark.CompareAndSwap(current, depth) {
				break
			}
		}
		if b.metrics != nil {
			b.metrics.ObserveEnqueued()
			b.metrics.ObserveQueue(len(b.queue), cap(b.queue), b.cfg.Now())
		}
	default:
		b.mu.RUnlock()
		b.rejectedQueue.Add(1)
		if b.metrics != nil {
			b.metrics.ObserveQueueRejected()
		}
		return DurableResult{}, ErrQueueFull
	}
	select {
	case response := <-request.result:
		return response.result, response.err
	case <-ctx.Done():
		// The worker still owns the request; a durable commit may occur. The source
		// cursor will not advance and a retry is deduplicated by the receipt.
		if b.metrics != nil {
			b.metrics.ObserveCallerCancelled()
		}
		return DurableResult{}, ctx.Err()
	}
}

// Close stops admission, flushes queued work and waits for the worker. It never drops
// an enqueued candidate.
func (b *Batcher) Close(ctx context.Context) error {
	if b == nil {
		return nil
	}
	b.closeOnce.Do(func() {
		b.mu.Lock()
		b.closed.Store(true)
		close(b.queue)
		b.mu.Unlock()
	})
	select {
	case <-b.done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Stats returns queue and outcome counters.
func (b *Batcher) Stats() BatcherStats {
	if b == nil {
		return BatcherStats{}
	}
	return BatcherStats{
		Submitted:     b.submitted.Load(),
		Committed:     b.committed.Load(),
		Duplicates:    b.duplicates.Load(),
		Conflicts:     b.conflicts.Load(),
		Inputs:        b.inputs.Load(),
		Batches:       b.batches.Load(),
		Transactions:  b.transactions.Load(),
		RejectedQueue: b.rejectedQueue.Load(),
		Errors:        b.errors.Load(),
		CommitUnknown: b.commitUnknown.Load(),
		QueueDepth:    int64(len(b.queue)),
		QueueHighMark: b.queueHighMark.Load(),
		MaxBatch:      b.maxBatch.Load(),
	}
}

func (b *Batcher) run() {
	defer close(b.done)
	for {
		first, ok := <-b.queue
		if !ok {
			return
		}
		batch := []submitRequest{first}
		timer := b.cfg.After(b.cfg.FlushInterval)
		closed := false
	collect:
		for len(batch) < b.cfg.MaxBatch {
			select {
			case request, ok := <-b.queue:
				if !ok {
					closed = true
					break collect
				}
				batch = append(batch, request)
			case <-timer:
				break collect
			}
		}
		if b.metrics != nil {
			b.metrics.ObserveBatch(len(batch))
			b.metrics.ObserveInFlight(len(batch))
			b.metrics.ObserveQueue(len(b.queue), cap(b.queue), b.cfg.Now())
		}
		b.flush(batch)
		if b.metrics != nil {
			b.metrics.ObserveInFlight(0)
			b.metrics.ObserveQueue(len(b.queue), cap(b.queue), b.cfg.Now())
		}
		if closed {
			return
		}
	}
}

func (b *Batcher) flush(batch []submitRequest) {
	b.batches.Add(1)
	if size := int64(len(batch)); size > b.maxBatch.Load() {
		b.maxBatch.Store(size)
	}
	// Group by session while preserving the first-seen session order.
	order := make([]string, 0, 2)
	groups := make(map[string][]int, 2)
	for index := range batch {
		session := batch[index].candidate.SessionID
		if _, exists := groups[session]; !exists {
			order = append(order, session)
		}
		groups[session] = append(groups[session], index)
	}
	for _, session := range order {
		indexes := groups[session]
		candidates := make([]DurableCandidate, len(indexes))
		for position, index := range indexes {
			candidates[position] = batch[index].candidate
		}
		b.transactions.Add(1)
		if b.metrics != nil {
			b.metrics.ObserveTransaction()
		}
		results, err := b.appender.AppendSession(context.Background(), session, candidates)
		if err != nil {
			unknown := errors.Is(err, ErrCommitUnknown)
			if unknown {
				b.commitUnknown.Add(1)
				if b.metrics != nil {
					b.metrics.ObserveCommitUnknown()
				}
				if reconciled, ok := b.reconcileBatch(session, candidates); ok {
					results = reconciled
					err = nil
					if b.metrics != nil {
						b.metrics.ObserveReconciled()
					}
				}
			}
			if err != nil {
				b.errors.Add(1)
				if b.metrics != nil {
					if unknown {
						// Ambiguous outcome: do not count as a definitive failure and do
						// not release the reservation; the durable state is unknown.
					} else {
						b.metrics.ObserveBatchFailure()
					}
					if errors.Is(err, ErrStorageUnavailable) {
						b.metrics.ObserveStorageUnavailable()
					}
					if errors.Is(err, ErrRetryExhausted) {
						b.metrics.ObserveRetryExhausted()
					}
				}
				for _, index := range indexes {
					if !unknown {
						b.releaseReservation(batch[index].reservation)
					}
					b.respond(batch[index], DurableResult{}, err)
				}
				continue
			}
		}
		if b.metrics != nil {
			b.metrics.ObserveStorageRecovered()
		}
		for position, index := range indexes {
			request := batch[index]
			result := results[position]
			switch {
			case result.Conflict:
				b.conflicts.Add(1)
				b.releaseReservation(request.reservation)
				if b.metrics != nil {
					b.metrics.ObserveDurableConflict()
				}
			case result.Duplicate:
				b.duplicates.Add(1)
				b.finalizeReservation(request.reservation)
				if b.metrics != nil {
					b.metrics.ObserveDurableDuplicate()
				}
			default:
				b.committed.Add(1)
				b.inputs.Add(1)
				b.finalizeReservation(request.reservation)
				if b.metrics != nil {
					latency := b.cfg.Now().Sub(request.enqueuedAt)
					if latency < 0 {
						latency = 0
					}
					b.metrics.ObserveDurableCommitted(latency)
				}
			}
			b.respond(request, result, nil)
		}
	}
}

func (b *Batcher) reconcileBatch(session string, candidates []DurableCandidate) ([]DurableResult, bool) {
	results := make([]DurableResult, len(candidates))
	for index, candidate := range candidates {
		if candidate.ProviderEventID == "" {
			return nil, false
		}
		result, found, err := b.appender.LookupCandidate(context.Background(), candidate)
		if err != nil || !found {
			return nil, false
		}
		// Reconciliation proves this submission is the committed writer for the
		// candidate; it is not a duplicate delivery.
		result.Duplicate = false
		results[index] = result
	}
	return results, true
}

func (b *Batcher) finalizeReservation(reservation *Reservation) {
	if reservation == nil {
		return
	}
	// A durable commit exists; an expired in-memory reservation must not be released.
	_ = reservation.Finalize()
}

func (b *Batcher) releaseReservation(reservation *Reservation) {
	if reservation == nil {
		return
	}
	_ = reservation.Release()
}

func (b *Batcher) respond(request submitRequest, result DurableResult, err error) {
	select {
	case request.result <- submitResponse{result: result, err: err}:
	default:
		// The caller canceled; the worker already resolved the durable state.
	}
}
