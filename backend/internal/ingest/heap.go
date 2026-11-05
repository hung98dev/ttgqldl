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
	at time.Time
	id uint64
}

// windowEntry holds a small bounded sliding window of accepted best-effort commands for
// one (session, actor, action) scope.
type windowEntry struct {
	samples    []windowSample
	expires    time.Time
	generation uint64
}
