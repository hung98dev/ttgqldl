package ingest

import (
	"container/heap"
	"time"

	"github.com/hung98dev/ttgqldl/backend/internal/domain"
)

type expiryKind uint8

const (
	expiryStable expiryKind = iota + 1
	expiryWindow
	expiryBucket
)

// expiryItem is one deterministic expiry record. Stale generations are ignored, which
// lets entry updates push a new item without an O(n) heap rebuild.
type expiryItem struct {
	at         time.Time
	seq        uint64
	kind       expiryKind
	key        [32]byte
	actor      domain.ActorKey
	generation uint64
}

type expiryHeap []expiryItem

func (h expiryHeap) Len() int { return len(h) }
func (h expiryHeap) Less(i, j int) bool {
	if h[i].at.Equal(h[j].at) {
		return h[i].seq < h[j].seq
	}
	return h[i].at.Before(h[j].at)
}
func (h expiryHeap) Swap(i, j int) { h[i], h[j] = h[j], h[i] }
func (h *expiryHeap) Push(value any) {
	*h = append(*h, value.(expiryItem))
}
func (h *expiryHeap) Pop() any {
	old := *h
	last := old[len(old)-1]
	*h = old[:len(old)-1]
	return last
}

func (d *Deduper) pushExpiry(item expiryItem) {
	heap.Push(&d.heap, item)
}

func heapPop(h *expiryHeap) expiryItem {
	return heap.Pop(h).(expiryItem)
}

// stableEntry is a stable-id receipt. Pending entries reserve a candidate until
// Finalize/Release; committed entries suppress duplicates until StableTTL.
type stableEntry struct {
	fingerprint [32]byte
	pending     bool
	reservation uint64
	expires     time.Time
	generation  uint64
}

type windowSample struct {
	at           time.Time
	id           uint64
	pending      bool
	pendingUntil time.Time
}

// windowEntry holds a small bounded sliding window of accepted best-effort commands for
// one (session, actor, action) scope.
type windowEntry struct {
	samples    []windowSample
	expires    time.Time
	generation uint64
}

// maybeCompactLocked rebuilds the expiry heap when stale nodes outnumber live state.
// Released/finalized receipts can leave stale nodes until their original deadline; this
// amortized compaction enforces the documented bound (active*2 + 64) under frozen clocks.
func (d *Deduper) maybeCompactLocked() {
	active := len(d.stable) + len(d.windows) + len(d.buckets)
	if len(d.heap) <= active*2+64 {
		return
	}
	rebuilt := make(expiryHeap, 0, active)
	for key, entry := range d.stable {
		rebuilt = append(rebuilt, expiryItem{
			at: entry.expires, seq: entry.reservation, kind: expiryStable, key: key, generation: entry.generation,
		})
	}
	for key, entry := range d.windows {
		var seq uint64
		if len(entry.samples) > 0 {
			seq = entry.samples[len(entry.samples)-1].id
		}
		rebuilt = append(rebuilt, expiryItem{
			at: entry.expires, seq: seq, kind: expiryWindow, key: key, generation: entry.generation,
		})
	}
	for actor, bucket := range d.buckets {
		rebuilt = append(rebuilt, expiryItem{
			at: bucket.lastSeen.Add(d.cfg.BucketIdleTTL), seq: bucket.generation, kind: expiryBucket, actor: actor, generation: bucket.generation,
		})
	}
	heap.Init(&rebuilt)
	d.heap = rebuilt
}
func (d *Deduper) pushWindowExpiry(key [32]byte, entry *windowEntry) {
	if len(entry.samples) == 0 {
		return
	}
	d.pushExpiry(expiryItem{
		at: entry.expires, seq: entry.samples[len(entry.samples)-1].id,
		kind: expiryWindow, key: key, generation: entry.generation,
	})
}
