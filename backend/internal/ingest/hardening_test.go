package ingest

import (
	"errors"
	"math"
	"testing"
	"time"
)

func TestExpiryHeapBoundedUnderFrozenClock(t *testing.T) {
	clock := &testClock{now: time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)}
	config := DefaultConfig()
	config.Now = clock.Now
	config.MaxEntries = 1
	config.MaxActors = 1
	config.PendingTTL = 30 * time.Second
	deduper, err := New(config)
	if err != nil {
		t.Fatal(err)
	}
	event := commentEvent("session-a", "mock", "same")
	action := mustAction(t, "XÂY")
	for index := 0; index < 100_000; index++ {
		reservation := mustAccept(t, deduper, event, actorFor(1), action)
		if err := reservation.Release(); err != nil {
			t.Fatal(err)
		}
	}
	stats := deduper.Stats()
	if stats.StableEntries != 0 || stats.Pending != 0 {
		t.Fatalf("stable state leaked: %+v", stats)
	}
	bound := 2*(stats.StableEntries+stats.WindowEntries+stats.Buckets) + 64
	if stats.ExpiryNodes > bound {
		t.Fatalf("expiry heap unbounded: nodes=%d bound=%d", stats.ExpiryNodes, bound)
	}
	if len(deduper.heap) != stats.ExpiryNodes {
		t.Fatalf("stats hide heap: len=%d stats=%d", len(deduper.heap), stats.ExpiryNodes)
	}
	t.Logf("stable loop: 100000 admissions, heap=%d, stats=%+v", len(deduper.heap), stats)

	// Missing-event-id admission/release must obey the same bound.
	missing := commentEvent("session-a", "mock", "")
	for index := 0; index < 100_000; index++ {
		reservation := mustAccept(t, deduper, missing, actorFor(1), action)
		if err := reservation.Release(); err != nil {
			t.Fatal(err)
		}
	}
	stats = deduper.Stats()
	if stats.WindowEntries != 0 {
		t.Fatalf("window state leaked: %+v", stats)
	}
	bound = 2*(stats.StableEntries+stats.WindowEntries+stats.Buckets) + 64
	if stats.ExpiryNodes > bound {
		t.Fatalf("window expiry heap unbounded: nodes=%d bound=%d", stats.ExpiryNodes, bound)
	}
	t.Logf("window loop: 100000 admissions, heap=%d, stats=%+v", len(deduper.heap), stats)
}

func TestBestEffortStaleReservationCannotFinalize(t *testing.T) {
	clock := &testClock{now: time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)}
	config := DefaultConfig()
	config.Now = clock.Now
	config.Window = 10 * time.Second
	config.PendingTTL = 30 * time.Second
	deduper, err := New(config)
	if err != nil {
		t.Fatal(err)
	}
	event := commentEvent("session-a", "mock", "")
	action := mustAction(t, "XÂY")
	stale := mustAccept(t, deduper, event, actorFor(3), action)
	clock.Advance(11 * time.Second)
	replacement := mustAccept(t, deduper, event, actorFor(3), action)
	clock.Advance(time.Second)
	if err := stale.Finalize(); !errors.Is(err, ErrReservationExpired) {
		t.Fatalf("stale window finalize = %v", err)
	}
	if err := replacement.Finalize(); err != nil {
		t.Fatalf("replacement finalize = %v", err)
	}
	// The stale reservation cannot mutate the replacement claim.
	decision := firstBestEffort(t, deduper, event, actorFor(3), action)
	if decision.Class != ClassRepeatNoop {
		t.Fatalf("replacement claim lost: %+v", decision)
	}

	// PendingTTL shorter than the window: finalize must still expire.
	clock2 := &testClock{now: time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)}
	config2 := DefaultConfig()
	config2.Now = clock2.Now
	config2.Window = 10 * time.Second
	config2.PendingTTL = 5 * time.Second
	deduper2, err := New(config2)
	if err != nil {
		t.Fatal(err)
	}
	stale2 := mustAccept(t, deduper2, event, actorFor(4), action)
	clock2.Advance(6 * time.Second)
	if err := stale2.Finalize(); !errors.Is(err, ErrReservationExpired) {
		t.Fatalf("stale pending finalize = %v", err)
	}
	// The expired pending sample no longer suppresses a retry.
	retry := mustAccept(t, deduper2, event, actorFor(4), action)
	if err := retry.Finalize(); err != nil {
		t.Fatal(err)
	}
}

func TestBestEffortStaleReleaseDoesNotRemoveReplacement(t *testing.T) {
	clock := &testClock{now: time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)}
	config := DefaultConfig()
	config.Now = clock.Now
	config.Window = 10 * time.Second
	config.PendingTTL = 30 * time.Second
	deduper, err := New(config)
	if err != nil {
		t.Fatal(err)
	}
	event := commentEvent("session-a", "mock", "")
	action := mustAction(t, "THỦ")
	stale := mustAccept(t, deduper, event, actorFor(5), action)
	clock.Advance(11 * time.Second)
	replacement := mustAccept(t, deduper, event, actorFor(5), action)
	clock.Advance(time.Second)
	if err := stale.Release(); !errors.Is(err, ErrReservationExpired) {
		t.Fatalf("stale release = %v", err)
	}
	if decision := firstBestEffort(t, deduper, event, actorFor(5), action); decision.Class != ClassRepeatNoop {
		t.Fatalf("replacement removed by stale release: %+v", decision)
	}
	if err := replacement.Finalize(); err != nil {
		t.Fatal(err)
	}
}

func TestClockRollbackNeverDoubleRefills(t *testing.T) {
	clock := &testClock{now: time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)}
	config := DefaultConfig()
	config.Now = clock.Now
	deduper, err := New(config)
	if err != nil {
		t.Fatal(err)
	}
	actor := actorFor(6)
	action := mustAction(t, "XÂY")
	event := commentEvent("session-a", "mock", "")
	for index := 0; index < 10; index++ {
		event.ProviderEventID = "exhaust-" + string(rune('a'+index))
		mustFinalize(t, mustAccept(t, deduper, event, actor, action))
	}
	clock.Advance(10 * time.Second)
	for index := 0; index < 5; index++ {
		event.ProviderEventID = "refill-" + string(rune('a'+index))
		mustFinalize(t, mustAccept(t, deduper, event, actor, action))
	}
	// Roll back and probe; the bucket must not rewind its refill baseline.
	clock.now = clock.now.Add(-10 * time.Second)
	event.ProviderEventID = "rollback-probe"
	if decision, _, _ := admit(t, deduper, event, actor, action); decision.Class != ClassRateLimited {
		t.Fatalf("rollback probe = %+v", decision)
	}
	for oscillation := 0; oscillation < 5; oscillation++ {
		clock.Advance(10 * time.Second)
		event.ProviderEventID = "forward-probe-" + string(rune('a'+oscillation))
		if decision, _, _ := admit(t, deduper, event, actor, action); decision.Class != ClassRateLimited {
			t.Fatalf("forward oscillation %d granted refill: %+v", oscillation, decision)
		}
		clock.now = clock.now.Add(-10 * time.Second)
	}
	clock.Advance(12 * time.Second) // two seconds beyond the credited high-water mark
	event.ProviderEventID = "after-high-water"
	decision, reservation, err := admit(t, deduper, event, actor, action)
	if err != nil || decision.Class != ClassAccepted || reservation == nil {
		t.Fatalf("high-water refill = %+v/%v", decision, err)
	}
	mustFinalize(t, reservation)
}

func TestNonFiniteRateConfigurationRejected(t *testing.T) {
	values := []float64{math.NaN(), math.Inf(1), math.Inf(-1)}
	for _, value := range values {
		config := DefaultConfig()
		config.Now = func() time.Time { return time.Unix(0, 0) }
		config.RateCapacity = value
		if _, err := New(config); !errors.Is(err, ErrInvalidInput) {
			t.Fatalf("capacity %v accepted: %v", value, err)
		}
		config = DefaultConfig()
		config.Now = func() time.Time { return time.Unix(0, 0) }
		config.RateRefillPerSecond = value
		if _, err := New(config); !errors.Is(err, ErrInvalidInput) {
			t.Fatalf("refill %v accepted: %v", value, err)
		}
	}
}
