package ingest

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/hung98dev/ttgqldl/backend/internal/commands"
	"github.com/hung98dev/ttgqldl/backend/internal/source"
)

func TestMetricsAdmissionClassification(t *testing.T) {
	t.Run("valid role and vote", func(t *testing.T) {
		metrics := NewMetrics()
		deduper, _ := newTestDeduper(t, func(config *Config) { config.Metrics = metrics })
		mustFinalize(t, mustAccept(t, deduper, commentEvent("s", "mock", "m1"), actorFor(40), mustAction(t, "XÂY")))
		mustFinalize(t, mustAccept(t, deduper, commentEvent("s", "mock", "m2"), actorFor(40), mustAction(t, "A")))
		snapshot := metrics.Snapshot()
		if snapshot.AcceptedCandidates != 2 || snapshot.DurableCommitted != 0 {
			t.Fatalf("accepted/durable = %+v", snapshot)
		}
	})
	t.Run("duplicate and conflict", func(t *testing.T) {
		metrics := NewMetrics()
		deduper, _ := newTestDeduper(t, func(config *Config) { config.Metrics = metrics })
		event := commentEvent("s", "mock", "same")
		actor := actorFor(41)
		mustFinalize(t, mustAccept(t, deduper, event, actor, mustAction(t, "XÂY")))
		if decision, _, _ := deduper.Admit(event, actor, mustAction(t, "XÂY")); decision.Class != ClassDuplicateProviderID {
			t.Fatalf("duplicate = %+v", decision)
		}
		if decision, _, _ := deduper.Admit(event, actor, mustAction(t, "THỦ")); decision.Class != ClassConflictingProviderID {
			t.Fatalf("conflict = %+v", decision)
		}
		snapshot := metrics.Snapshot()
		if snapshot.DuplicateProviderID != 1 || snapshot.ConflictingProviderID != 1 {
			t.Fatalf("duplicate/conflict = %+v", snapshot)
		}
	})
	t.Run("repeat and rate limit", func(t *testing.T) {
		metrics := NewMetrics()
		deduper, _ := newTestDeduper(t, func(config *Config) { config.Metrics = metrics })
		actor := actorFor(42)
		action := mustAction(t, "BUILD")
		mustFinalize(t, mustAccept(t, deduper, commentEvent("s", "mock", ""), actor, action))
		if decision, _, _ := deduper.Admit(commentEvent("s", "mock", ""), actor, action); decision.Class != ClassRepeatNoop {
			t.Fatalf("repeat = %+v", decision)
		}
		for index := 0; index < 9; index++ {
			mustFinalize(t, mustAccept(t, deduper, commentEvent("s", "mock", "rate-"+string(rune('a'+index))), actor, action))
		}
		if decision, _, _ := deduper.Admit(commentEvent("s", "mock", "rate-final"), actor, action); decision.Class != ClassRateLimited {
			t.Fatalf("rate = %+v", decision)
		}
		snapshot := metrics.Snapshot()
		if snapshot.RepeatNoop != 1 || snapshot.RateLimited != 1 {
			t.Fatalf("repeat/rate = %+v", snapshot)
		}
	})
	t.Run("invalid identity and ignored non-command", func(t *testing.T) {
		metrics := NewMetrics()
		deduper, _ := newTestDeduper(t, func(config *Config) { config.Metrics = metrics })
		if _, _, err := deduper.Admit(commentEvent("s", "mock", "identity"), "", mustAction(t, "XÂY")); !errors.Is(err, ErrInvalidIdentity) {
			t.Fatalf("identity error = %v", err)
		}
		gift := source.Event{Provider: "mock", SourceSession: "s", ProviderEventID: "gift", Kind: source.KindGift}
		if decision, _, _ := deduper.Admit(gift, "", commands.Action{}); decision.Class != ClassIgnoredNonCommand {
			t.Fatalf("gift = %+v", decision)
		}
		snapshot := metrics.Snapshot()
		if snapshot.InvalidIdentity != 1 || snapshot.IgnoredNonCommand != 1 {
			t.Fatalf("invalid/ignored = %+v", snapshot)
		}
	})
	t.Run("capacity", func(t *testing.T) {
		metrics := NewMetrics()
		deduper, _ := newTestDeduper(t, func(config *Config) {
			config.Metrics = metrics
			config.MaxEntries = 1
		})
		mustFinalize(t, mustAccept(t, deduper, commentEvent("s", "mock", "cap-1"), actorFor(43), mustAction(t, "XÂY")))
		if decision, _, err := deduper.Admit(commentEvent("s", "mock", "cap-2"), actorFor(43), mustAction(t, "XÂY")); decision.Class != ClassCapacityExceeded || !errors.Is(err, ErrCacheFull) {
			t.Fatalf("capacity = %+v/%v", decision, err)
		}
		if snapshot := metrics.Snapshot(); snapshot.CapacityExceeded != 1 {
			t.Fatalf("capacity metric = %+v", snapshot)
		}
	})
	t.Run("source-stage counters", func(t *testing.T) {
		metrics := NewMetrics()
		metrics.ObserveReceived()
		metrics.ObserveValidated()
		metrics.ObserveInvalidGrammar()
		metrics.ObserveOversized()
		metrics.ObserveUnauthorizedSource()
		snapshot := metrics.Snapshot()
		if snapshot.InputReceived != 1 || snapshot.InputValidated != 1 || snapshot.InvalidGrammar != 1 ||
			snapshot.Oversized != 1 || snapshot.UnauthorizedSource != 1 {
			t.Fatalf("source counters = %+v", snapshot)
		}
	})
}

func TestMetricsQueueAndDurableStages(t *testing.T) {
	metrics := NewMetrics()
	appender := &fakeAppender{}
	batcher := testBatcher(t, appender, func(config *BatcherConfig) {
		config.Metrics = metrics
		config.QueueCapacity = 100
		config.MaxBatch = 10
		config.FlushInterval = time.Second
	})
	results := make([]<-chan submitResponse, 5)
	for index := range results {
		results[index] = submitAsync(batcher, DurableCandidate{
			SessionID: "s", Provider: "mock", ProviderEventID: "q-" + string(rune('a'+index)),
			ActorKey: actorFor(44), Action: mustAction(t, "XÂY"),
		}, nil)
	}
	for index, channel := range results {
		if response := <-channel; response.err != nil {
			t.Fatalf("submit %d: %v", index, response.err)
		}
	}
	snapshot := metrics.SnapshotAt(time.Now())
	if snapshot.Enqueued != 5 || snapshot.DurableCommitted != 5 || snapshot.Inputs != 5 ||
		snapshot.Batches != 1 || snapshot.Transactions != 1 || snapshot.MaxBatch != 5 {
		t.Fatalf("queue/durable stages = %+v", snapshot)
	}
	if snapshot.QueueCapacity != 100 || snapshot.QueueDepth != 0 || snapshot.QueueHighWater > 5 {
		t.Fatalf("queue gauges = %+v", snapshot)
	}
	if snapshot.LostGameEvents != 0 || snapshot.Alert.Level == AlertDegraded {
		t.Fatalf("false degradation = %+v", snapshot.Alert)
	}
}

func TestMetricsQueueOverflowIsBackpressureNotLoss(t *testing.T) {
	metrics := NewMetrics()
	block := make(chan struct{})
	entered := make(chan struct{})
	appender := &fakeAppender{block: block, entered: entered}
	batcher := testBatcher(t, appender, func(config *BatcherConfig) {
		config.Metrics = metrics
		config.QueueCapacity = 2
		config.MaxBatch = 1
		config.FlushInterval = time.Second
	})
	first := submitAsync(batcher, DurableCandidate{SessionID: "s", Provider: "mock", ProviderEventID: "o1", ActorKey: actorFor(45), Action: mustAction(t, "XÂY")}, nil)
	<-entered
	second := submitAsync(batcher, DurableCandidate{SessionID: "s", Provider: "mock", ProviderEventID: "o2", ActorKey: actorFor(45), Action: mustAction(t, "XÂY")}, nil)
	third := submitAsync(batcher, DurableCandidate{SessionID: "s", Provider: "mock", ProviderEventID: "o3", ActorKey: actorFor(45), Action: mustAction(t, "XÂY")}, nil)
	deadline := time.Now().Add(2 * time.Second)
	for metrics.Snapshot().QueueDepth < 2 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if _, err := batcher.Submit(context.Background(), DurableCandidate{SessionID: "s", Provider: "mock", ProviderEventID: "o4", ActorKey: actorFor(45), Action: mustAction(t, "XÂY")}, nil); !errors.Is(err, ErrQueueFull) {
		t.Fatalf("queue full = %v", err)
	}
	close(block)
	for _, channel := range []<-chan submitResponse{first, second, third} {
		if response := <-channel; response.err != nil {
			t.Fatalf("queued submit: %v", response.err)
		}
	}
	snapshot := metrics.Snapshot()
	if snapshot.QueueOverflow != 1 || snapshot.LostGameEvents != 0 || snapshot.Alert.Level == AlertDegraded {
		t.Fatalf("overflow accounting = %+v", snapshot)
	}
}

func TestMetricsCallerCancellationDoesNotClaimLoss(t *testing.T) {
	metrics := NewMetrics()
	block := make(chan struct{})
	entered := make(chan struct{})
	appender := &fakeAppender{block: block, entered: entered}
	batcher := testBatcher(t, appender, func(config *BatcherConfig) {
		config.Metrics = metrics
		config.MaxBatch = 1
		config.FlushInterval = time.Second
	})
	ctx, cancel := context.WithCancel(context.Background())
	result := make(chan error, 1)
	go func() {
		_, err := batcher.Submit(ctx, DurableCandidate{SessionID: "s", Provider: "mock", ProviderEventID: "cancel-1", ActorKey: actorFor(46), Action: mustAction(t, "XÂY")}, nil)
		result <- err
	}()
	<-entered
	cancel()
	if err := <-result; !errors.Is(err, context.Canceled) {
		t.Fatalf("cancel error = %v", err)
	}
	close(block)
	deadline := time.Now().Add(2 * time.Second)
	for metrics.Snapshot().DurableCommitted < 1 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	snapshot := metrics.Snapshot()
	if snapshot.CallerCancelled != 1 || snapshot.DurableCommitted != 1 || snapshot.LostGameEvents != 0 {
		t.Fatalf("cancel accounting = %+v", snapshot)
	}
}

func TestMetricsAlertSustainedQueueAndRecovery(t *testing.T) {
	metrics := NewMetrics()
	start := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	metrics.ObserveQueue(80, 100, start)
	if state := metrics.SnapshotAt(start).Alert; state.Level != AlertOK || state.QueueSustained {
		t.Fatalf("premature alert = %+v", state)
	}
	if state := metrics.SnapshotAt(start.Add(29 * time.Second)).Alert; state.Level != AlertOK || state.QueueSustained {
		t.Fatalf("premature sustained alert = %+v", state)
	}
	state := metrics.SnapshotAt(start.Add(30 * time.Second)).Alert
	if state.Level != AlertWarning || !state.QueueSustained || len(state.Codes) != 1 || state.Codes[0] != AlertCodeQueueHigh {
		t.Fatalf("sustained alert = %+v", state)
	}
	metrics.ObserveQueue(50, 100, start.Add(31*time.Second))
	state = metrics.SnapshotAt(start.Add(31 * time.Second)).Alert
	if state.Level != AlertOK || state.QueueSustained || len(state.Codes) != 0 {
		t.Fatalf("recovery = %+v", state)
	}
}

func TestMetricsAlertExactThresholdAndLossAndStorage(t *testing.T) {
	metrics := NewMetrics()
	start := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	metrics.ObserveQueue(75, 100, start)
	if state := metrics.SnapshotAt(start.Add(31 * time.Second)).Alert; state.Level != AlertOK {
		t.Fatalf("exact 75%% alerted: %+v", state)
	}
	metrics.RecordDroppedGameEvent(DropStorageFailure)
	state := metrics.SnapshotAt(start.Add(32 * time.Second)).Alert
	if state.Level != AlertDegraded || state.DroppedGameEvents != 1 {
		t.Fatalf("loss alert = %+v", state)
	}
	metrics.ObserveStorageUnavailable()
	state = metrics.SnapshotAt(start.Add(33 * time.Second)).Alert
	if state.Level != AlertDegraded || !state.StorageUnavailable {
		t.Fatalf("storage alert = %+v", state)
	}
	metrics.ObserveStorageRecovered()
	state = metrics.SnapshotAt(start.Add(34 * time.Second)).Alert
	if state.Level != AlertDegraded || state.StorageUnavailable {
		t.Fatalf("loss must persist after storage recovery: %+v", state)
	}

	clean := NewMetrics()
	clean.ObserveCommitUnknown()
	if state := clean.SnapshotAt(start).Alert; state.Level != AlertWarning || state.CommitUnknownOutstanding != 1 {
		t.Fatalf("commit unknown alert = %+v", state)
	}
	clean.ObserveReconciled()
	if state := clean.SnapshotAt(start).Alert; state.Level != AlertOK || state.CommitUnknownOutstanding != 0 {
		t.Fatalf("reconciled alert = %+v", state)
	}
	unknown := NewMetrics().Snapshot()
	if unknown.Alert.Level != AlertUnknown || unknown.Alert.Observed {
		t.Fatalf("unknown state = %+v", unknown.Alert)
	}
}

func TestMetricsHandlerAndPrivacy(t *testing.T) {
	metrics := NewMetrics()
	metrics.ObserveInvalidGrammar()
	server := httptest.NewServer(metrics.Handler())
	defer server.Close()
	response, err := http.Get(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK || response.Header.Get("Cache-Control") != "no-store" {
		t.Fatalf("response = %d %q", response.StatusCode, response.Header.Get("Cache-Control"))
	}
	var snapshot Snapshot
	if err := json.NewDecoder(response.Body).Decode(&snapshot); err != nil {
		t.Fatal(err)
	}
	if snapshot.InvalidGrammar != 1 || snapshot.GameplayApplied != nil {
		t.Fatalf("snapshot = %+v", snapshot)
	}
	request, _ := http.NewRequest(http.MethodPost, server.URL, nil)
	response, err = http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusMethodNotAllowed || response.Header.Get("Allow") != http.MethodGet {
		t.Fatalf("POST = %d allow=%q", response.StatusCode, response.Header.Get("Allow"))
	}
	payload, _ := json.Marshal(metrics.Snapshot())
	text := string(payload)
	for _, forbidden := range []string{"provider_viewer", "actor_key", "comment", "secret", "session_id"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("metrics leaked %q: %s", forbidden, text)
		}
	}
}

func TestMetricsConcurrentUpdates(t *testing.T) {
	metrics := NewMetrics()
	var wait sync.WaitGroup
	for worker := 0; worker < 32; worker++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			for index := 0; index < 1000; index++ {
				metrics.ObserveReceived()
				metrics.ObserveValidated()
				metrics.ObserveAdmission(Decision{Class: ClassAccepted}, nil)
				_ = metrics.Snapshot()
			}
		}()
	}
	wait.Wait()
	snapshot := metrics.Snapshot()
	if snapshot.InputReceived != 32_000 || snapshot.InputValidated != 32_000 || snapshot.AcceptedCandidates != 32_000 {
		t.Fatalf("concurrent counts = %+v", snapshot)
	}
}

func TestMetricsUnknownCommitDoesNotReleaseReservation(t *testing.T) {
	metrics := NewMetrics()
	deduper, _ := newTestDeduper(t, nil)
	event := commentEvent("s", "mock", "unknown-reservation")
	reservation := mustAccept(t, deduper, event, actorFor(47), mustAction(t, "XÂY"))
	appender := &fakeAppender{err: ErrCommitUnknown}
	batcher := testBatcher(t, appender, func(config *BatcherConfig) {
		config.Metrics = metrics
		config.MaxBatch = 1
		config.FlushInterval = time.Millisecond
	})
	_, err := batcher.Submit(context.Background(), candidateFor("s", "unknown-reservation", actorFor(47), mustAction(t, "XÂY")), reservation)
	if !errors.Is(err, ErrCommitUnknown) {
		t.Fatalf("unknown commit = %v", err)
	}
	if stats := deduper.Stats(); stats.Pending != 1 || stats.StableEntries != 1 {
		t.Fatalf("reservation was released on unknown outcome: %+v", stats)
	}
	snapshot := metrics.Snapshot()
	if snapshot.CommitUnknownOutstanding != 1 || snapshot.LostGameEvents != 0 {
		t.Fatalf("unknown metrics = %+v", snapshot)
	}
	appender.err = nil
	if _, err := batcher.Submit(context.Background(), candidateFor("s", "unknown-reservation", actorFor(47), mustAction(t, "XÂY")), reservation); err != nil {
		t.Fatalf("retry: %v", err)
	}
	if stats := deduper.Stats(); stats.Pending != 0 {
		t.Fatalf("reservation not finalized: %+v", stats)
	}
}

func TestMetricsPostgresDurableOutcomes(t *testing.T) {
	pool := testPool(t)
	session := createSession(t, pool, "mock", "running", 1, 250, 0)
	appender := newTestAppender(t, pool)
	metrics := NewMetrics()
	batcher := testBatcher(t, appender, func(config *BatcherConfig) {
		config.Metrics = metrics
		config.MaxBatch = 1
		config.FlushInterval = time.Millisecond
	})
	actor := actorFor(48)
	action := mustAction(t, "XÂY")
	candidate := candidateFor(session.ID, "metrics-durable", actor, action)

	// Lost acknowledgment after commit is reconciled and counted as committed, never
	// as a durable success before the row exists.
	appender.afterCommit = func() error { return errors.New("ack lost") }
	if _, err := batcher.Submit(context.Background(), candidate, nil); err != nil {
		t.Fatalf("ambiguous submit: %v", err)
	}
	appender.afterCommit = nil
	snapshot := metrics.Snapshot()
	if snapshot.CommitUnknown != 1 || snapshot.Reconciled != 1 || snapshot.DurableCommitted != 1 ||
		snapshot.Inputs != 1 || snapshot.CommitUnknownOutstanding != 0 || snapshot.LostGameEvents != 0 {
		t.Fatalf("reconciliation metrics = %+v", snapshot)
	}

	if _, err := batcher.Submit(context.Background(), candidate, nil); err != nil {
		t.Fatalf("duplicate submit: %v", err)
	}
	conflict := candidate
	conflict.Fingerprint = payloadFingerprint(actor, source.KindComment, mustAction(t, "THỦ"))
	conflict.Action = mustAction(t, "THỦ")
	if result, err := batcher.Submit(context.Background(), conflict, nil); err != nil || !result.Conflict {
		t.Fatalf("conflict submit = %+v/%v", result, err)
	}
	snapshot = metrics.Snapshot()
	if snapshot.DurableDuplicates != 1 || snapshot.DurableConflicts != 1 || snapshot.Inputs != 1 {
		t.Fatalf("duplicate/conflict metrics = %+v", snapshot)
	}

	// A definitive pre-commit failure must not increment durable committed.
	failing := newTestAppender(t, pool)
	failing.beforeCommit = func() error { return errors.New("rollback") }
	failingMetrics := NewMetrics()
	failingBatcher := testBatcher(t, failing, func(config *BatcherConfig) {
		config.Metrics = failingMetrics
		config.MaxBatch = 1
		config.FlushInterval = time.Millisecond
	})
	if _, err := failingBatcher.Submit(context.Background(), candidateFor(session.ID, "metrics-rollback", actor, action), nil); err == nil {
		t.Fatal("rollback submit unexpectedly succeeded")
	}
	snapshot = failingMetrics.Snapshot()
	if snapshot.DurableCommitted != 0 || snapshot.Inputs != 0 || snapshot.BatchFailures != 1 {
		t.Fatalf("rollback metrics = %+v", snapshot)
	}
}

func TestMetricsLatencySustainedAlert(t *testing.T) {
	metrics := NewMetrics()
	start := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	for index := 0; index < 20; index++ {
		metrics.ObserveDurableCommitted(1500 * time.Millisecond)
	}
	if state := metrics.SnapshotAt(start).Alert; state.Level != AlertOK || state.LatencySustained {
		t.Fatalf("premature latency alert = %+v", state)
	}
	if state := metrics.SnapshotAt(start.Add(4 * time.Minute)).Alert; state.LatencySustained {
		t.Fatalf("latency alert before five minutes = %+v", state)
	}
	state := metrics.SnapshotAt(start.Add(5 * time.Minute)).Alert
	if state.Level != AlertWarning || !state.LatencySustained || !containsCode(state.Codes, AlertCodeLatencyHigh) {
		t.Fatalf("sustained latency alert = %+v", state)
	}
	for index := 0; index < 500; index++ {
		metrics.ObserveDurableCommitted(time.Millisecond)
	}
	state = metrics.SnapshotAt(start.Add(5*time.Minute + time.Second)).Alert
	if state.LatencySustained || containsCode(state.Codes, AlertCodeLatencyHigh) {
		t.Fatalf("latency recovery = %+v", state)
	}
}

func containsCode(codes []string, want string) bool {
	for _, code := range codes {
		if code == want {
			return true
		}
	}
	return false
}

func TestMetricsDefaultQueueCapacityInvariant(t *testing.T) {
	metrics := NewMetrics()
	batcher, err := NewBatcher(&fakeAppender{}, BatcherConfig{Metrics: metrics})
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = batcher.Close(ctx)
	}()
	snapshot := metrics.Snapshot()
	if snapshot.QueueCapacity != 20_000 || snapshot.QueueDepth != 0 {
		t.Fatalf("default queue = %+v", snapshot)
	}
}

func TestMetricsHighCardinalityIsBounded(t *testing.T) {
	metrics := NewMetrics()
	deduper, _ := newTestDeduper(t, func(config *Config) {
		config.Metrics = metrics
		config.MaxEntries = 2000
		config.MaxActors = 2000
	})
	action := mustAction(t, "XÂY")
	for index := 0; index < 1000; index++ {
		event := commentEvent("s", "mock", "cardinality-"+string(rune('a'+index%26))+"-"+string(rune('a'+(index/26)%26))+"-"+string(rune('a'+index/676)))
		decision, reservation, err := deduper.Admit(event, actorFor(1000+index), action)
		if err != nil || decision.Class != ClassAccepted {
			t.Fatalf("admit %d = %+v/%v", index, decision, err)
		}
		_ = reservation.Finalize()
	}
	payload, err := json.Marshal(metrics.Snapshot())
	if err != nil {
		t.Fatal(err)
	}
	if len(payload) > 4096 {
		t.Fatalf("metrics payload grew with cardinality: %d bytes", len(payload))
	}
	if strings.Contains(string(payload), "cardinality-") {
		t.Fatal("event id leaked into metrics")
	}
	if snapshot := metrics.Snapshot(); snapshot.AcceptedCandidates != 1000 {
		t.Fatalf("accepted = %d", snapshot.AcceptedCandidates)
	}
}
