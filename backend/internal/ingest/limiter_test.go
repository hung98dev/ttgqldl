package ingest

import (
	"testing"
	"time"

	"github.com/hung98dev/ttgqldl/backend/internal/source"
)

func rateEvent(index int) source.Event {
	return commentEvent("session-a", "mock", "rate-"+string(rune('a'+index%26))+"-"+string(rune('a'+index/26)))
}

func TestRateBurstAndRefillBoundary(t *testing.T) {
	deduper, clock := newTestDeduper(t, nil)
	actor := actorFor(20)
	action := mustAction(t, "XÂY")
	for index := 0; index < 10; index++ {
		mustFinalize(t, mustAccept(t, deduper, rateEvent(index), actor, action))
	}
	decision, reservation, err := admit(t, deduper, rateEvent(10), actor, action)
	if err != nil || decision.Class != ClassRateLimited || reservation != nil {
		t.Fatalf("11th event = %+v/%v", decision, err)
	}

	clock.Advance(1900 * time.Millisecond)
	decision, _, err = admit(t, deduper, rateEvent(11), actor, action)
	if err != nil || decision.Class != ClassRateLimited {
		t.Fatalf("1.9s refill = %+v/%v", decision, err)
	}
	clock.Advance(100 * time.Millisecond) // exactly 2s: 0.5 * 2 = 1 token
	decision, reservation, err = admit(t, deduper, rateEvent(12), actor, action)
	if err != nil || decision.Class != ClassAccepted || reservation == nil {
		t.Fatalf("2s refill = %+v/%v", decision, err)
	}
	mustFinalize(t, reservation)
	clock.Advance(100 * time.Millisecond)
	decision, _, err = admit(t, deduper, rateEvent(13), actor, action)
	if err != nil || decision.Class != ClassRateLimited {
		t.Fatalf("0.1s after refill = %+v/%v", decision, err)
	}
}

func TestRateRefundOnRelease(t *testing.T) {
	deduper, _ := newTestDeduper(t, nil)
	actor := actorFor(21)
	action := mustAction(t, "THỦ")
	var last *Reservation
	for index := 0; index < 10; index++ {
		last = mustAccept(t, deduper, rateEvent(index), actor, action)
	}
	decision, _, _ := admit(t, deduper, rateEvent(10), actor, action)
	if decision.Class != ClassRateLimited {
		t.Fatalf("exhaustion = %+v", decision)
	}
	if err := last.Release(); err != nil {
		t.Fatal(err)
	}
	decision, reservation, err := admit(t, deduper, rateEvent(11), actor, action)
	if err != nil || decision.Class != ClassAccepted || reservation == nil {
		t.Fatalf("refunded event = %+v/%v", decision, err)
	}
	mustFinalize(t, reservation)
}

func TestRateIsolationAndClockRollback(t *testing.T) {
	deduper, clock := newTestDeduper(t, nil)
	first := actorFor(22)
	second := actorFor(23)
	action := mustAction(t, "ĐÁNH")
	for index := 0; index < 10; index++ {
		mustFinalize(t, mustAccept(t, deduper, rateEvent(index), first, action))
	}
	decision, _, _ := admit(t, deduper, rateEvent(10), first, action)
	if decision.Class != ClassRateLimited {
		t.Fatalf("first actor = %+v", decision)
	}
	decision, reservation, err := admit(t, deduper, rateEvent(11), second, action)
	if err != nil || decision.Class != ClassAccepted || reservation == nil {
		t.Fatalf("second actor = %+v/%v", decision, err)
	}
	mustFinalize(t, reservation)

	clock.now = clock.now.Add(-100 * time.Second)
	decision, _, _ = admit(t, deduper, rateEvent(12), first, action)
	if decision.Class != ClassRateLimited {
		t.Fatalf("clock rollback minted tokens: %+v", decision)
	}
	clock.Advance(2 * time.Second) // still before the already-credited instant
	decision, _, _ = admit(t, deduper, rateEvent(13), first, action)
	if decision.Class != ClassRateLimited {
		t.Fatalf("refill before high-water = %+v", decision)
	}
	clock.Advance(100 * time.Second) // t0+2s: one token beyond the high-water mark
	decision, reservation, err = admit(t, deduper, rateEvent(14), first, action)
	if err != nil || decision.Class != ClassAccepted || reservation == nil {
		t.Fatalf("post-rollback refill = %+v/%v", decision, err)
	}
	mustFinalize(t, reservation)
}

func TestBucketIdleEvictionIsBounded(t *testing.T) {
	deduper, clock := newTestDeduper(t, func(config *Config) {
		config.BucketIdleTTL = time.Second
	})
	actor := actorFor(24)
	action := mustAction(t, "BUILD")
	for index := 0; index < 10; index++ {
		mustFinalize(t, mustAccept(t, deduper, rateEvent(index), actor, action))
	}
	if stats := deduper.Stats(); stats.Buckets != 1 {
		t.Fatalf("buckets = %d", stats.Buckets)
	}
	clock.Advance(2 * time.Second) // idle beyond BucketIdleTTL and one refill token
	decision, reservation, err := admit(t, deduper, rateEvent(20), actor, action)
	if err != nil || decision.Class != ClassAccepted || reservation == nil {
		t.Fatalf("post-idle event = %+v/%v", decision, err)
	}
	mustFinalize(t, reservation)
	// A fresh bucket after idle eviction has full burst capacity again.
	for index := 21; index < 30; index++ {
		decision, reservation, err := admit(t, deduper, rateEvent(index), actor, action)
		if err != nil || decision.Class != ClassAccepted || reservation == nil {
			t.Fatalf("post-idle burst %d = %+v/%v", index, decision, err)
		}
		mustFinalize(t, reservation)
	}
}

func TestRateDoesNotChargeDuplicatesOrConflicts(t *testing.T) {
	deduper, _ := newTestDeduper(t, nil)
	actor := actorFor(25)
	action := mustAction(t, "XÂY")
	event := commentEvent("session-a", "mock", "dup-rate")
	mustFinalize(t, mustAccept(t, deduper, event, actor, action))
	for index := 0; index < 50; index++ {
		decision, _, err := admit(t, deduper, event, actor, action)
		if err != nil || decision.Class != ClassDuplicateProviderID {
			t.Fatalf("duplicate %d = %+v/%v", index, decision, err)
		}
	}
	// Nine burst tokens remain because the first accepted event and no duplicate charged.
	for index := 0; index < 9; index++ {
		event := commentEvent("session-a", "mock", "free-"+string(rune('a'+index)))
		decision, reservation, err := admit(t, deduper, event, actor, action)
		if err != nil || decision.Class != ClassAccepted || reservation == nil {
			t.Fatalf("burst %d = %+v/%v", index, decision, err)
		}
		mustFinalize(t, reservation)
	}
	decision, _, _ := admit(t, deduper, commentEvent("session-a", "mock", "free-exhausted"), actor, action)
	if decision.Class != ClassRateLimited {
		t.Fatalf("expected exhaustion after nine free tokens: %+v", decision)
	}
}

func TestBestEffortRateStillApplies(t *testing.T) {
	deduper, _ := newTestDeduper(t, func(config *Config) {
		config.RateCapacity = 2
	})
	actor := actorFor(26)
	event := commentEvent("session-a", "mock", "")
	for index, message := range []string{"XÂY", "THỦ"} {
		decision, reservation, err := admit(t, deduper, event, actor, mustAction(t, message))
		if err != nil || decision.Class != ClassAccepted || reservation == nil {
			t.Fatalf("message %d = %+v/%v", index, decision, err)
		}
		mustFinalize(t, reservation)
	}
	decision, _, err := admit(t, deduper, event, actor, mustAction(t, "ĐÁNH"))
	if err != nil || decision.Class != ClassRateLimited || !decision.BestEffort {
		t.Fatalf("best-effort rate limit = %+v/%v", decision, err)
	}
}

func TestMaxActorsCapacity(t *testing.T) {
	deduper, _ := newTestDeduper(t, func(config *Config) {
		config.MaxActors = 1
	})
	first := actorFor(27)
	second := actorFor(28)
	action := mustAction(t, "XÂY")
	mustFinalize(t, mustAccept(t, deduper, commentEvent("session-a", "mock", "a1"), first, action))
	decision, reservation, err := admit(t, deduper, commentEvent("session-a", "mock", "a2"), second, action)
	if err == nil || decision.Class != ClassCapacityExceeded || reservation != nil {
		t.Fatalf("actor capacity = %+v/%v", decision, err)
	}
}

func TestNoNegativeTokensAfterManyRejections(t *testing.T) {
	deduper, clock := newTestDeduper(t, nil)
	actor := actorFor(29)
	action := mustAction(t, "THỦ")
	for index := 0; index < 10; index++ {
		mustFinalize(t, mustAccept(t, deduper, rateEvent(index), actor, action))
	}
	for index := 0; index < 100; index++ {
		decision, _, _ := admit(t, deduper, rateEvent(10+index), actor, action)
		if decision.Class != ClassRateLimited {
			t.Fatalf("rejection %d = %+v", index, decision)
		}
		clock.Advance(10 * time.Millisecond)
	}
	clock.Advance(2 * time.Second)
	decision, reservation, err := admit(t, deduper, rateEvent(200), actor, action)
	if err != nil || decision.Class != ClassAccepted || reservation == nil {
		t.Fatalf("post-rejection refill = %+v/%v", decision, err)
	}
	mustFinalize(t, reservation)
}
