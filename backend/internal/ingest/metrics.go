package ingest

import (
	"encoding/json"
	"errors"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

// AlertLevel is the bounded operator-facing state. It is deliberately coarse.
type AlertLevel string

const (
	AlertUnknown  AlertLevel = "unknown"
	AlertOK       AlertLevel = "ok"
	AlertWarning  AlertLevel = "warning"
	AlertDegraded AlertLevel = "degraded"
)

// Alert codes are fixed; no user-controlled label can enter the metric surface.
const (
	AlertCodeQueueHigh     = "QUEUE_UTILIZATION_HIGH"
	AlertCodeDropped       = "DROPPED_GAME_EVENT"
	AlertCodeStorageDown   = "STORAGE_UNAVAILABLE"
	AlertCodeCommitUnknown = "COMMIT_UNKNOWN"
	AlertCodeLatencyHigh   = "INTERNAL_LATENCY_HIGH"
)

// QueueAlertThreshold is the documented sustained queue-utilization threshold.
const QueueAlertThreshold = 0.75

// QueueAlertDuration is the documented sustained interval.
const QueueAlertDuration = 30 * time.Second

// LatencyAlertThreshold is the documented internal p95 threshold.
const LatencyAlertThreshold = time.Second

// LatencyAlertDuration is the documented sustained latency interval.
const LatencyAlertDuration = 5 * time.Minute

// LatencyAlertMinSamples avoids declaring a latency alert from too few samples.
const LatencyAlertMinSamples = 20

// DropReason is a fixed, bounded reason vocabulary for a confirmed lost valid action.
type DropReason uint8

const (
	DropUnknown DropReason = iota
	DropQueueFull
	DropStorageFailure
	DropCommitUnknown
	DropPolicy
)

func (r DropReason) String() string {
	switch r {
	case DropQueueFull:
		return "queue_full"
	case DropStorageFailure:
		return "storage_failure"
	case DropCommitUnknown:
		return "commit_unknown"
	case DropPolicy:
		return "policy"
	default:
		return "unknown"
	}
}

// AlertState is a bounded snapshot. Codes contains at most four fixed values.
type AlertState struct {
	Level                    AlertLevel `json:"level"`
	Codes                    []string   `json:"codes"`
	Observed                 bool       `json:"observed"`
	QueueUtilization         float64    `json:"queue_utilization"`
	QueueSustained           bool       `json:"queue_sustained"`
	LatencySustained         bool       `json:"latency_sustained"`
	LatencyP95Millis         float64    `json:"latency_p95_ms"`
	DroppedGameEvents        int64      `json:"dropped_game_events"`
	StorageUnavailable       bool       `json:"storage_unavailable"`
	CommitUnknownOutstanding int64      `json:"commit_unknown_outstanding"`
	AsOf                     time.Time  `json:"as_of"`
}

type alertMonitor struct {
	mu           sync.Mutex
	observed     bool
	highSince    time.Time
	sustained    bool
	latencySince time.Time
	latencyHigh  bool
	dropped      bool
	storageDown  bool
	asOf         time.Time
}

func (m *alertMonitor) observeQueue(utilization float64, now time.Time) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.observed = true
	m.asOf = now
	if utilization > QueueAlertThreshold {
		if m.highSince.IsZero() {
			m.highSince = now
		}
		if !now.Before(m.highSince.Add(QueueAlertDuration)) {
			m.sustained = true
		}
		return
	}
	m.highSince = time.Time{}
	m.sustained = false
}

func (m *alertMonitor) observeLatency(p95Millis float64, samples int64, now time.Time) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.observed = true
	m.asOf = now
	if samples < LatencyAlertMinSamples || p95Millis <= float64(LatencyAlertThreshold/time.Millisecond) {
		m.latencySince = time.Time{}
		m.latencyHigh = false
		return
	}
	if m.latencySince.IsZero() {
		m.latencySince = now
	}
	if !now.Before(m.latencySince.Add(LatencyAlertDuration)) {
		m.latencyHigh = true
	}
}

func (m *alertMonitor) recordDrop() {
	m.mu.Lock()
	m.dropped = true
	m.mu.Unlock()
}

func (m *alertMonitor) setStorage(down bool) {
	m.mu.Lock()
	m.storageDown = down
	m.mu.Unlock()
}

func (m *alertMonitor) snapshot(queueUtilization, p95Millis float64, droppedCount, commitUnknown int64) AlertState {
	m.mu.Lock()
	defer m.mu.Unlock()
	state := AlertState{
		Observed:                 m.observed,
		QueueUtilization:         queueUtilization,
		QueueSustained:           m.sustained,
		LatencySustained:         m.latencyHigh,
		LatencyP95Millis:         p95Millis,
		DroppedGameEvents:        droppedCount,
		StorageUnavailable:       m.storageDown,
		CommitUnknownOutstanding: commitUnknown,
		AsOf:                     m.asOf,
	}
	switch {
	case !m.observed:
		state.Level = AlertUnknown
	case m.dropped || m.storageDown:
		state.Level = AlertDegraded
	case m.sustained || m.latencyHigh || commitUnknown > 0:
		state.Level = AlertWarning
	default:
		state.Level = AlertOK
	}
	codes := make([]string, 0, 5)
	if m.dropped {
		codes = append(codes, AlertCodeDropped)
	}
	if m.storageDown {
		codes = append(codes, AlertCodeStorageDown)
	}
	if commitUnknown > 0 {
		codes = append(codes, AlertCodeCommitUnknown)
	}
	if m.sustained {
		codes = append(codes, AlertCodeQueueHigh)
	}
	if m.latencyHigh {
		codes = append(codes, AlertCodeLatencyHigh)
	}
	state.Codes = codes
	return state
}

// Metrics is a fixed-size, race-safe ingestion observability registry. It uses atomic
// counters and a bounded latency histogram; no per-event objects, label maps or raw
// identifiers are retained.
type Metrics struct {
	inputReceived       atomic.Int64
	inputValidated      atomic.Int64
	invalidGrammar      atomic.Int64
	oversized           atomic.Int64
	duplicateProvider   atomic.Int64
	conflictingProvider atomic.Int64
	repeatNoop          atomic.Int64
	rateLimited         atomic.Int64
	invalidIdentity     atomic.Int64
	ignoredNonCommand   atomic.Int64
	capacityExceeded    atomic.Int64
	queueOverflow       atomic.Int64
	unauthorizedSource  atomic.Int64
	storageUnavailable  atomic.Int64
	commitUnknownTotal  atomic.Int64
	commitUnknownOut    atomic.Int64
	retryExhausted      atomic.Int64
	acceptedCandidates  atomic.Int64
	enqueued            atomic.Int64
	durableCommitted    atomic.Int64
	durableDuplicates   atomic.Int64
	durableConflicts    atomic.Int64
	batchFailures       atomic.Int64
	callerCancelled     atomic.Int64
	reconciled          atomic.Int64
	lostGameEvents      atomic.Int64
	cooldown            atomic.Int64
	blockedUser         atomic.Int64

	queueDepth    atomic.Int64
	queueCapacity atomic.Int64
	queueHigh     atomic.Int64
	inFlight      atomic.Int64
	batches       atomic.Int64
	transactions  atomic.Int64
	inputs        atomic.Int64
	maxBatch      atomic.Int64
	latency       latencyHistogram
	alerts        alertMonitor
}

// NewMetrics returns an empty registry.
func NewMetrics() *Metrics { return &Metrics{} }

// ObserveReceived records a source event at the receive stage.
func (m *Metrics) ObserveReceived() { m.inputReceived.Add(1) }

// ObserveValidated records a source event that passed source validation.
func (m *Metrics) ObserveValidated() { m.inputValidated.Add(1) }

// ObserveInvalidGrammar records an input rejected by command grammar.
func (m *Metrics) ObserveInvalidGrammar() { m.invalidGrammar.Add(1) }

// ObserveOversized records an input rejected by the byte bound.
func (m *Metrics) ObserveOversized() { m.oversized.Add(1) }

// ObserveInvalidIdentity records a comment without a valid scoped actor identity.
func (m *Metrics) ObserveInvalidIdentity() { m.invalidIdentity.Add(1) }

// ObserveUnauthorizedSource records a rejected source/session binding.
func (m *Metrics) ObserveUnauthorizedSource() { m.unauthorizedSource.Add(1) }

// ObserveAdmission records the single final W0304 classification for one admission.
// Accepted here is an in-memory candidate, not durable or gameplay acceptance.
func (m *Metrics) ObserveAdmission(decision Decision, err error) {
	switch decision.Class {
	case ClassAccepted:
		m.acceptedCandidates.Add(1)
		return
	case ClassDuplicateProviderID:
		m.duplicateProvider.Add(1)
		return
	case ClassConflictingProviderID:
		m.conflictingProvider.Add(1)
		return
	case ClassRepeatNoop:
		m.repeatNoop.Add(1)
		return
	case ClassRateLimited:
		m.rateLimited.Add(1)
		return
	case ClassInvalidIdentity:
		m.invalidIdentity.Add(1)
		return
	case ClassIgnoredNonCommand:
		m.ignoredNonCommand.Add(1)
		return
	case ClassCapacityExceeded:
		m.capacityExceeded.Add(1)
		return
	}
	switch {
	case errors.Is(err, ErrInvalidIdentity):
		m.invalidIdentity.Add(1)
	case errors.Is(err, ErrInvalidAction), errors.Is(err, ErrInvalidInput):
		m.invalidGrammar.Add(1)
	case errors.Is(err, ErrCacheFull):
		m.capacityExceeded.Add(1)
	}
}

// ObserveQueue updates queue gauges and the sustained-utilization alert. It is called
// with an injected clock; queue depth excludes the batch already held by the worker.
func (m *Metrics) ObserveQueue(depth, capacity int, now time.Time) {
	if capacity < 0 {
		capacity = 0
	}
	m.queueDepth.Store(int64(depth))
	m.queueCapacity.Store(int64(capacity))
	for {
		high := m.queueHigh.Load()
		if int64(depth) <= high || m.queueHigh.CompareAndSwap(high, int64(depth)) {
			break
		}
	}
	utilization := 0.0
	if capacity > 0 {
		utilization = float64(depth) / float64(capacity)
	}
	m.alerts.observeQueue(utilization, now)
}

// ObserveInFlight records the batch size currently held by the worker.
func (m *Metrics) ObserveInFlight(size int) { m.inFlight.Store(int64(size)) }

// ObserveBatch records one flush attempt and its size.
func (m *Metrics) ObserveBatch(size int) {
	m.batches.Add(1)
	for {
		current := m.maxBatch.Load()
		if int64(size) <= current || m.maxBatch.CompareAndSwap(current, int64(size)) {
			break
		}
	}
}

// ObserveTransaction records one PostgreSQL append transaction attempt.
func (m *Metrics) ObserveTransaction() { m.transactions.Add(1) }

// ObserveEnqueued records one accepted submission entering the queue.
func (m *Metrics) ObserveEnqueued() { m.enqueued.Add(1) }

// ObserveQueueRejected records a valid submission rejected by a full queue. This is
// backpressure, not a confirmed permanent loss.
func (m *Metrics) ObserveQueueRejected() { m.queueOverflow.Add(1) }

// ObserveCallerCancelled records a caller cancellation after enqueue. The durable
// outcome is unknown to the caller and is not counted as a loss.
func (m *Metrics) ObserveCallerCancelled() { m.callerCancelled.Add(1) }

// ObserveDurableCommitted records a PostgreSQL commit (or verified reconciliation) and
// the receive-to-commit latency when it is meaningful.
func (m *Metrics) ObserveDurableCommitted(latency time.Duration) {
	m.durableCommitted.Add(1)
	m.inputs.Add(1)
	m.latency.observe(latency)
	m.alerts.setStorage(false)
}

// ObserveDurableDuplicate records a durable receipt duplicate (no new input event).
func (m *Metrics) ObserveDurableDuplicate() { m.durableDuplicates.Add(1) }

// ObserveDurableConflict records a durable conflicting receipt (no new input event).
func (m *Metrics) ObserveDurableConflict() { m.durableConflicts.Add(1) }

// ObserveBatchFailure records a definitive batch/transaction failure.
func (m *Metrics) ObserveBatchFailure() { m.batchFailures.Add(1) }

// ObserveStorageUnavailable records a storage failure and raises degradation.
func (m *Metrics) ObserveStorageUnavailable() {
	m.storageUnavailable.Add(1)
	m.alerts.setStorage(true)
}

// ObserveStorageRecovered clears the storage-degradation signal after a successful
// transaction or verified reconciliation.
func (m *Metrics) ObserveStorageRecovered() { m.alerts.setStorage(false) }

// ObserveCooldown is reserved for the M04 reducer. W0306 never invents a cooldown.
func (m *Metrics) ObserveCooldown() { m.cooldown.Add(1) }

// ObserveBlockedUser is reserved for M07 moderation. W0306 never invents a block.
func (m *Metrics) ObserveBlockedUser() { m.blockedUser.Add(1) }

// ObserveRetryExhausted records an exhausted bounded retry budget.
func (m *Metrics) ObserveRetryExhausted() { m.retryExhausted.Add(1) }

// ObserveCommitUnknown records an ambiguous commit. It remains outstanding until a
// verified reconciliation clears it.
func (m *Metrics) ObserveCommitUnknown() {
	m.commitUnknownTotal.Add(1)
	m.commitUnknownOut.Add(1)
}

// ObserveReconciled records a verified reconciliation of an ambiguous commit.
func (m *Metrics) ObserveReconciled() {
	m.reconciled.Add(1)
	for {
		current := m.commitUnknownOut.Load()
		if current <= 0 {
			return
		}
		if m.commitUnknownOut.CompareAndSwap(current, current-1) {
			return
		}
	}
}

// RecordDroppedGameEvent records a confirmed lost valid game action and raises the
// permanent degradation signal. Only a caller that has exhausted safe retry/replay
// paths may call this; queue overflow and backpressure must not call it.
func (m *Metrics) RecordDroppedGameEvent(_ DropReason) {
	m.lostGameEvents.Add(1)
	m.alerts.recordDrop()
}

// Snapshot returns a bounded immutable view. GameplayApplied is nil until M04 wires the
// authoritative reducer; it is never fabricated as zero.
type Snapshot struct {
	InputReceived            int64      `json:"input_received"`
	InputValidated           int64      `json:"input_validated"`
	InvalidGrammar           int64      `json:"invalid_grammar"`
	Oversized                int64      `json:"oversized"`
	DuplicateProviderID      int64      `json:"duplicate_provider_id"`
	ConflictingProviderID    int64      `json:"conflicting_provider_id"`
	RepeatNoop               int64      `json:"repeat_noop"`
	RateLimited              int64      `json:"rate_limited"`
	InvalidIdentity          int64      `json:"invalid_identity"`
	IgnoredNonCommand        int64      `json:"ignored_non_command"`
	CapacityExceeded         int64      `json:"capacity_exceeded"`
	QueueOverflow            int64      `json:"queue_overflow"`
	UnauthorizedSource       int64      `json:"unauthorized_source"`
	StorageUnavailable       int64      `json:"storage_unavailable"`
	CommitUnknown            int64      `json:"commit_unknown"`
	RetryExhausted           int64      `json:"retry_exhausted"`
	AcceptedCandidates       int64      `json:"accepted_candidates"`
	Enqueued                 int64      `json:"enqueued"`
	DurableCommitted         int64      `json:"durable_committed"`
	DurableDuplicates        int64      `json:"durable_duplicates"`
	DurableConflicts         int64      `json:"durable_conflicts"`
	BatchFailures            int64      `json:"batch_failures"`
	CallerCancelled          int64      `json:"caller_cancelled"`
	Reconciled               int64      `json:"reconciled"`
	LostGameEvents           int64      `json:"lost_game_events"`
	Cooldown                 int64      `json:"cooldown"`
	BlockedUser              int64      `json:"blocked_user"`
	QueueDepth               int64      `json:"queue_depth"`
	QueueCapacity            int64      `json:"queue_capacity"`
	QueueUtilization         float64    `json:"queue_utilization"`
	QueueHighWater           int64      `json:"queue_high_water"`
	InFlight                 int64      `json:"in_flight"`
	Batches                  int64      `json:"batches"`
	Transactions             int64      `json:"transactions"`
	Inputs                   int64      `json:"inputs"`
	MaxBatch                 int64      `json:"max_batch"`
	CommitUnknownOutstanding int64      `json:"commit_unknown_outstanding"`
	LatencySamples           int64      `json:"latency_samples"`
	LatencyP50Millis         float64    `json:"latency_p50_ms"`
	LatencyP95Millis         float64    `json:"latency_p95_ms"`
	LatencyP99Millis         float64    `json:"latency_p99_ms"`
	GameplayApplied          *int64     `json:"gameplay_applied"`
	Alert                    AlertState `json:"alert"`
}

// Snapshot builds a consistent bounded view without advancing alert time.
func (m *Metrics) Snapshot() Snapshot {
	return m.buildSnapshot()
}

// SnapshotAt evaluates the sustained-queue alert at an injected instant and returns a
// consistent view. Operators/scrapers should use this so a queue that stays high without
// new submissions still raises the documented 30 s alert.
func (m *Metrics) SnapshotAt(now time.Time) Snapshot {
	capacity := m.queueCapacity.Load()
	utilization := 0.0
	if capacity > 0 {
		utilization = float64(m.queueDepth.Load()) / float64(capacity)
	}
	m.alerts.observeQueue(utilization, now)
	_, p95, _, samples := m.latency.percentiles()
	m.alerts.observeLatency(p95, samples, now)
	return m.buildSnapshot()
}

func (m *Metrics) buildSnapshot() Snapshot {
	depth := m.queueDepth.Load()
	capacity := m.queueCapacity.Load()
	utilization := 0.0
	if capacity > 0 {
		utilization = float64(depth) / float64(capacity)
	}
	unknownOut := m.commitUnknownOut.Load()
	if unknownOut < 0 {
		unknownOut = 0
	}
	p50, p95, p99, samples := m.latency.percentiles()
	return Snapshot{
		InputReceived: m.inputReceived.Load(), InputValidated: m.inputValidated.Load(),
		InvalidGrammar: m.invalidGrammar.Load(), Oversized: m.oversized.Load(),
		DuplicateProviderID: m.duplicateProvider.Load(), ConflictingProviderID: m.conflictingProvider.Load(),
		RepeatNoop: m.repeatNoop.Load(), RateLimited: m.rateLimited.Load(),
		InvalidIdentity: m.invalidIdentity.Load(), IgnoredNonCommand: m.ignoredNonCommand.Load(),
		CapacityExceeded: m.capacityExceeded.Load(), QueueOverflow: m.queueOverflow.Load(),
		UnauthorizedSource: m.unauthorizedSource.Load(), StorageUnavailable: m.storageUnavailable.Load(),
		CommitUnknown: m.commitUnknownTotal.Load(), RetryExhausted: m.retryExhausted.Load(),
		AcceptedCandidates: m.acceptedCandidates.Load(), Enqueued: m.enqueued.Load(),
		DurableCommitted: m.durableCommitted.Load(), DurableDuplicates: m.durableDuplicates.Load(),
		DurableConflicts: m.durableConflicts.Load(), BatchFailures: m.batchFailures.Load(),
		CallerCancelled: m.callerCancelled.Load(), Reconciled: m.reconciled.Load(),
		LostGameEvents: m.lostGameEvents.Load(), Cooldown: m.cooldown.Load(), BlockedUser: m.blockedUser.Load(),
		QueueDepth: depth, QueueCapacity: capacity,
		QueueUtilization: utilization, QueueHighWater: m.queueHigh.Load(), InFlight: m.inFlight.Load(),
		Batches: m.batches.Load(), Transactions: m.transactions.Load(), Inputs: m.inputs.Load(),
		MaxBatch: m.maxBatch.Load(), CommitUnknownOutstanding: unknownOut,
		LatencySamples: samples, LatencyP50Millis: p50, LatencyP95Millis: p95, LatencyP99Millis: p99,
		Alert: m.alerts.snapshot(utilization, p95, m.lostGameEvents.Load(), unknownOut),
	}
}

// Handler returns a read-only JSON metrics endpoint. It is intentionally not wired into
// the M01 server skeleton: no ingestion pipeline exists there yet, and exporting fake
// zeros would be dishonest. M04/M07 must mount it only after wiring a real registry.
func (m *Metrics) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		encoder := json.NewEncoder(w)
		encoder.SetEscapeHTML(true)
		if err := encoder.Encode(m.SnapshotAt(time.Now())); err != nil {
			return
		}
	})
}

// latencyHistogram is a fixed-bucket approximation. Buckets are upper bounds in
// milliseconds; percentiles report the upper bound of the containing bucket.
type latencyHistogram struct {
	counts [15]atomic.Int64
	sum    atomic.Int64
	count  atomic.Int64
}

var latencyBounds = [...]int64{1, 2, 5, 10, 20, 50, 100, 200, 500, 1000, 2000, 5000, 10000, 30000, 60000}

func (h *latencyHistogram) observe(duration time.Duration) {
	if duration < 0 {
		duration = 0
	}
	millis := duration.Milliseconds()
	index := len(h.counts) - 1
	for i, bound := range latencyBounds {
		if millis <= bound {
			index = i
			break
		}
	}
	h.counts[index].Add(1)
	h.sum.Add(duration.Nanoseconds())
	h.count.Add(1)
}

func (h *latencyHistogram) percentiles() (float64, float64, float64, int64) {
	total := h.count.Load()
	if total == 0 {
		return 0, 0, 0, 0
	}
	quantile := func(q float64) float64 {
		target := int64(float64(total)*q + 0.999999)
		if target < 1 {
			target = 1
		}
		var cumulative int64
		for i := range h.counts {
			cumulative += h.counts[i].Load()
			if cumulative >= target {
				return float64(latencyBounds[i])
			}
		}
		return float64(latencyBounds[len(latencyBounds)-1])
	}
	return quantile(0.50), quantile(0.95), quantile(0.99), total
}
