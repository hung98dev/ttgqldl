package ingest

import (
	"math"
	"time"

	"github.com/hung98dev/ttgqldl/backend/internal/domain"
)

// bucket is one actor's token bucket. Capacity/refill come from the explicit Config; the
// bucket is scoped by the scoped private ActorKey, never by alias or provider viewer id.
type bucket struct {
	tokens     float64
	last       time.Time
	lastSeen   time.Time
	generation uint64
}

// takeTokenLocked refills and consumes one token. It reports whether a token was
// available and whether the actor cache is at capacity (degraded, durable ingestion
// required). It never allows a negative balance and clamps clock reversal to zero elapsed
// time so a rollback cannot mint tokens.
func (d *Deduper) takeTokenLocked(actor domain.ActorKey, now time.Time) (bool, bool) {
	if actor == "" {
		return false, false
	}
	b, exists := d.buckets[actor]
	if !exists {
		if len(d.buckets) >= d.cfg.MaxActors {
			return false, true
		}
		b = &bucket{
			tokens:     d.cfg.RateCapacity,
			last:       now,
			lastSeen:   now,
			generation: 1,
		}
		d.buckets[actor] = b
		// One heap item per bucket; stale items are re-queued lazily on purge, so the
		// heap cannot grow per event.
		d.pushExpiry(expiryItem{
			at:         b.lastSeen.Add(d.cfg.BucketIdleTTL),
			seq:        b.generation,
			kind:       expiryBucket,
			actor:      actor,
			generation: b.generation,
		})
	} else {
		// Monotonic effective time: a clock rollback never rewinds the refill baseline,
		// so returning to an already-credited instant cannot grant a second refill.
		effective := now
		if effective.Before(b.last) {
			effective = b.last
		}
		elapsed := effective.Sub(b.last)
		b.tokens = math.Min(d.cfg.RateCapacity, b.tokens+elapsed.Seconds()*d.cfg.RateRefillPerSecond)
		b.last = effective
		if now.After(b.lastSeen) {
			b.lastSeen = now
		}
		b.generation++
	}
	if b.tokens+1e-9 < 1 {
		return false, false
	}
	b.tokens -= 1
	if b.tokens < 0 {
		b.tokens = 0
	}
	return true, false
}

// refundLocked returns one token after a released reservation. Refunds are capped at
// capacity so they cannot be farmed into a larger burst.
func (d *Deduper) refundLocked(actor domain.ActorKey, now time.Time) {
	if actor == "" {
		return
	}
	b, exists := d.buckets[actor]
	if !exists {
		if len(d.buckets) >= d.cfg.MaxActors {
			return
		}
		b = &bucket{tokens: 0, last: now, lastSeen: now, generation: 1}
		d.buckets[actor] = b
		d.pushExpiry(expiryItem{
			at:         b.lastSeen.Add(d.cfg.BucketIdleTTL),
			seq:        b.generation,
			kind:       expiryBucket,
			actor:      actor,
			generation: b.generation,
		})
		return
	}
	effective := now
	if effective.Before(b.last) {
		effective = b.last
	}
	b.tokens = math.Min(d.cfg.RateCapacity, b.tokens+1)
	b.last = effective
	if now.After(b.lastSeen) {
		b.lastSeen = now
	}
	b.generation++
}
