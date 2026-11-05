package ingest

import (
	"fmt"
	"math"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/hung98dev/ttgqldl/backend/internal/commands"
	"github.com/hung98dev/ttgqldl/backend/internal/domain"
	"github.com/hung98dev/ttgqldl/backend/internal/source"
)

// Clock supplies server-observed acceptance time. It is mandatory; no package-level
// time.Now call is made by the deduplication or limiter paths.
type Clock func() time.Time

// Defaults for an explicit configuration. Callers may override any field before New.
const (
	DefaultMaxEntries          = 100_000
	DefaultMaxActors           = 100_000
	DefaultStableTTL           = 10 * time.Minute
	DefaultWindow              = 10 * time.Second
	DefaultWindowSamples       = 8
	DefaultPendingTTL          = 30 * time.Second
	DefaultRateCapacity        = 10.0
	DefaultRateRefillPerSecond = 0.5
	DefaultBucketIdleTTL       = 10 * time.Minute
)

// Config is the explicit bounded-ingest configuration.
type Config struct {
	// MaxEntries bounds stable receipts plus best-effort window entries. Unexpired
	// entries are never evicted; reaching capacity is reported as degraded.
	MaxEntries int
	// MaxActors bounds token buckets.
	MaxActors int
	// StableTTL is how long a finalized stable-id receipt suppresses duplicates.
	StableTTL time.Duration
	// Window is the best-effort suppression window for missing event ids.
	Window time.Duration
	// WindowSamples is the per-key sliding-window sample count.
	WindowSamples int
	// PendingTTL bounds an unresolved reservation; no reservation stays pending forever.
	PendingTTL time.Duration
	// RateCapacity is the token-bucket burst size.
	RateCapacity float64
	// RateRefillPerSecond is the sustained refill rate.
	RateRefillPerSecond float64
	// BucketIdleTTL evicts inactive token buckets.
	BucketIdleTTL time.Duration
	// Now is required.
	Now Clock
	// Metrics optionally records W0306 admission stages. It never affects decisions.
	Metrics *Metrics
}

// DefaultConfig returns the reviewed defaults with a real clock. Tests inject a fixed
// clock; production code may override this field explicitly.
func DefaultConfig() Config {
	return Config{
		MaxEntries:          DefaultMaxEntries,
		MaxActors:           DefaultMaxActors,
		StableTTL:           DefaultStableTTL,
		Window:              DefaultWindow,
		WindowSamples:       DefaultWindowSamples,
		PendingTTL:          DefaultPendingTTL,
		RateCapacity:        DefaultRateCapacity,
		RateRefillPerSecond: DefaultRateRefillPerSecond,
		BucketIdleTTL:       DefaultBucketIdleTTL,
		Now:                 time.Now,
	}
}

func (c Config) normalized() Config {
	if c.MaxEntries == 0 {
		c.MaxEntries = DefaultMaxEntries
	}
	if c.MaxActors == 0 {
		c.MaxActors = DefaultMaxActors
	}
	if c.StableTTL == 0 {
		c.StableTTL = DefaultStableTTL
	}
	if c.Window == 0 {
		c.Window = DefaultWindow
	}
	if c.WindowSamples == 0 {
		c.WindowSamples = DefaultWindowSamples
	}
	if c.PendingTTL == 0 {
		c.PendingTTL = DefaultPendingTTL
	}
	if c.RateCapacity == 0 {
		c.RateCapacity = DefaultRateCapacity
	}
	if c.RateRefillPerSecond == 0 {
		c.RateRefillPerSecond = DefaultRateRefillPerSecond
	}
	if c.BucketIdleTTL == 0 {
		c.BucketIdleTTL = DefaultBucketIdleTTL
	}
	return c
}

func (c Config) validate() error {
	switch {
	case c.MaxEntries < 1:
		return fmt.Errorf("%w: MaxEntries must be positive", ErrInvalidInput)
	case c.MaxActors < 1:
		return fmt.Errorf("%w: MaxActors must be positive", ErrInvalidInput)
	case c.StableTTL <= 0, c.Window <= 0, c.PendingTTL <= 0, c.BucketIdleTTL <= 0:
		return fmt.Errorf("%w: TTL values must be positive", ErrInvalidInput)
	case c.WindowSamples < 1:
		return fmt.Errorf("%w: WindowSamples must be positive", ErrInvalidInput)
	case math.IsNaN(c.RateCapacity) || math.IsInf(c.RateCapacity, 0):
		return fmt.Errorf("%w: rate capacity must be finite", ErrInvalidInput)
	case math.IsNaN(c.RateRefillPerSecond) || math.IsInf(c.RateRefillPerSecond, 0):
		return fmt.Errorf("%w: rate refill must be finite", ErrInvalidInput)
	case c.RateCapacity <= 0, c.RateRefillPerSecond <= 0:
		return fmt.Errorf("%w: rate parameters must be positive", ErrInvalidInput)
	case c.Now == nil:
		return fmt.Errorf("%w: clock is required", ErrInvalidInput)
	}
	return nil
}

// Classification is the typed ingress decision.
type Classification uint8

const (
	// ClassAccepted means the event may proceed as a candidate to the future W0305
	// durable transaction and reducer. It is not a durable or gameplay acceptance.
	ClassAccepted Classification = iota + 1
	// ClassDuplicateProviderID means a stable event id was already seen with the same
	// verified identity/action fingerprint.
	ClassDuplicateProviderID
	// ClassConflictingProviderID means a stable event id was reused with a different
	// verified identity/action fingerprint. It must never produce a gameplay action.
	ClassConflictingProviderID
	// ClassRepeatNoop means a missing-event-id command repeated inside the bounded
	// best-effort window.
	ClassRepeatNoop
	// ClassRateLimited means the per-actor token bucket rejected a new actionable event.
	ClassRateLimited
	// ClassInvalidIdentity means a comment arrived without a valid scoped actor key.
	ClassInvalidIdentity
	// ClassIgnoredNonCommand means the event kind cannot produce a gameplay action.
	ClassIgnoredNonCommand
	// ClassCapacityExceeded means the bounded in-memory cache cannot preserve its
	// contract and durable ingestion is required.
	ClassCapacityExceeded
)

// String returns a stable diagnostic label.
func (c Classification) String() string {
	switch c {
	case ClassAccepted:
		return "accept_candidate"
	case ClassDuplicateProviderID:
		return "duplicate_provider_id"
	case ClassConflictingProviderID:
		return "conflicting_provider_id"
	case ClassRepeatNoop:
		return "repeat_noop_candidate"
	case ClassRateLimited:
		return "rate_limited"
	case ClassInvalidIdentity:
		return "invalid_identity"
	case ClassIgnoredNonCommand:
		return "ignored_non_command"
	case ClassCapacityExceeded:
		return "capacity_exceeded"
	default:
		return "invalid"
	}
}

// Decision is the typed result of Admit. BestEffort marks the missing-event-id path.
type Decision struct {
	Class      Classification
	BestEffort bool
}

// Stats is a bounded diagnostic snapshot; it never includes identities or event ids.
type Stats struct {
	StableEntries int
	WindowEntries int
	Buckets       int
	Pending       int
	// ExpiryNodes is the live expiry-heap length. It is bounded by active state plus
	// a small compaction allowance, never by total admissions.
	ExpiryNodes int
}

// Deduper is the race-safe bounded in-memory ingress filter. A single mutex serializes
// decisions so simultaneous submissions of one event id cannot both be first-seen.
type Deduper struct {
	mu      sync.Mutex
	cfg     Config
	stable  map[[32]byte]*stableEntry
	windows map[[32]byte]*windowEntry
	buckets map[domain.ActorKey]*bucket
	heap    expiryHeap
	seq     uint64
	metrics *Metrics
}

// New validates the explicit configuration and constructs an empty deduper.
func New(config Config) (*Deduper, error) {
	config = config.normalized()
	if err := config.validate(); err != nil {
		return nil, err
	}
	return &Deduper{
		cfg:     config,
		stable:  make(map[[32]byte]*stableEntry, 64),
		windows: make(map[[32]byte]*windowEntry, 64),
		buckets: make(map[domain.ActorKey]*bucket, 64),
		metrics: config.Metrics,
	}, nil
}

// Stats returns bounded cache counts.
func (d *Deduper) Stats() Stats {
	d.mu.Lock()
	defer d.mu.Unlock()
	stats := Stats{StableEntries: len(d.stable), WindowEntries: len(d.windows), Buckets: len(d.buckets), ExpiryNodes: len(d.heap)}
	for _, entry := range d.stable {
		if entry.pending {
			stats.Pending++
		}
	}
	return stats
}

// Admit classifies one validated source event with its resolved private actor key and
// typed parsed action. On ClassAccepted the returned reservation must be finalized after
// the downstream operation succeeds, or released on failure/cancellation.
func (d *Deduper) Admit(event source.Event, actor domain.ActorKey, action commands.Action) (decision Decision, reservation *Reservation, err error) {
	defer func() {
		if d.metrics != nil {
			d.metrics.ObserveAdmission(decision, err)
		}
	}()
	if err := validateAdmitInput(event, actor, action); err != nil {
		return Decision{}, nil, err
	}
	now := d.cfg.Now()
	d.mu.Lock()
	defer d.mu.Unlock()
	defer d.maybeCompactLocked()
	d.purgeLocked(now)

	if event.Kind != source.KindComment {
		return d.admitNonCommandLocked(event, now)
	}
	if event.ProviderEventID != "" {
		return d.admitStableLocked(event, actor, action, now)
	}
	return d.admitWindowLocked(event, actor, action, now)
}

func (d *Deduper) admitStableLocked(event source.Event, actor domain.ActorKey, action commands.Action, now time.Time) (Decision, *Reservation, error) {
	key := stableScope(event.SourceSession, event.Provider, event.ProviderEventID)
	fingerprint := payloadFingerprint(actor, event.Kind, action)
	if existing, ok := d.stable[key]; ok {
		if existing.fingerprint == fingerprint {
			return Decision{Class: ClassDuplicateProviderID}, nil, nil
		}
		return Decision{Class: ClassConflictingProviderID}, nil, nil
	}
	if d.atCapacityLocked() {
		return Decision{Class: ClassCapacityExceeded}, nil, ErrCacheFull
	}
	allowed, capacity := d.takeTokenLocked(actor, now)
	if capacity {
		return Decision{Class: ClassCapacityExceeded}, nil, ErrCacheFull
	}
	if !allowed {
		return Decision{Class: ClassRateLimited}, nil, nil
	}
	d.seq++
	entry := &stableEntry{
		fingerprint: fingerprint,
		pending:     true,
		reservation: d.seq,
		expires:     now.Add(d.cfg.PendingTTL),
		generation:  1,
	}
	d.stable[key] = entry
	d.pushExpiry(expiryItem{at: entry.expires, seq: d.seq, kind: expiryStable, key: key, generation: entry.generation})
	reservation := &Reservation{
		deduper:    d,
		id:         d.seq,
		stableKey:  key,
		actor:      actor,
		state:      reservationPending,
		bestEffort: false,
	}
	return Decision{Class: ClassAccepted}, reservation, nil
}

func (d *Deduper) admitWindowLocked(event source.Event, actor domain.ActorKey, action commands.Action, now time.Time) (Decision, *Reservation, error) {
	key := windowScope(event.SourceSession, actor, event.Kind, action)
	if existing, ok := d.windows[key]; ok {
		// A pending best-effort sample whose PendingTTL elapsed is no longer a claim and
		// must not suppress a retry or keep memory alive.
		filtered := existing.samples[:0]
		for _, sample := range existing.samples {
			if sample.pending && !now.Before(sample.pendingUntil) {
				continue
			}
			filtered = append(filtered, sample)
		}
		if len(filtered) == 0 {
			delete(d.windows, key)
		} else if len(filtered) != len(existing.samples) {
			existing.samples = filtered
			existing.generation++
			existing.expires = windowExpiry(existing, d.cfg.Window)
			d.pushWindowExpiry(key, existing)
		} else {
			existing.samples = filtered
		}
		for _, sample := range existing.samples {
			if now.Before(sample.at.Add(d.cfg.Window)) {
				return Decision{Class: ClassRepeatNoop, BestEffort: true}, nil, nil
			}
		}
	}
	if d.atCapacityLocked() {
		return Decision{Class: ClassCapacityExceeded, BestEffort: true}, nil, ErrCacheFull
	}
	allowed, capacity := d.takeTokenLocked(actor, now)
	if capacity {
		return Decision{Class: ClassCapacityExceeded, BestEffort: true}, nil, ErrCacheFull
	}
	if !allowed {
		return Decision{Class: ClassRateLimited, BestEffort: true}, nil, nil
	}
	d.seq++
	sampleID := d.seq
	entry, exists := d.windows[key]
	if !exists {
		entry = &windowEntry{generation: 1}
		d.windows[key] = entry
	} else {
		entry.generation++
	}
	entry.samples = append(entry.samples, windowSample{
		at: now, id: sampleID, pending: true, pendingUntil: now.Add(d.cfg.PendingTTL),
	})
	if len(entry.samples) > d.cfg.WindowSamples {
		entry.samples = entry.samples[len(entry.samples)-d.cfg.WindowSamples:]
	}
	entry.expires = windowExpiry(entry, d.cfg.Window)
	d.pushWindowExpiry(key, entry)
	reservation := &Reservation{
		deduper:    d,
		id:         sampleID,
		windowKey:  key,
		sampleID:   sampleID,
		actor:      actor,
		state:      reservationPending,
		bestEffort: true,
	}
	return Decision{Class: ClassAccepted, BestEffort: true}, reservation, nil
}

// admitNonCommandLocked records a stable non-command event id for conflict detection but
// never returns a reservation. Capacity is best effort for non-game events: if the cache
// is full the event is ignored without blocking gameplay traffic.
func (d *Deduper) admitNonCommandLocked(event source.Event, now time.Time) (Decision, *Reservation, error) {
	if event.ProviderEventID == "" {
		return Decision{Class: ClassIgnoredNonCommand}, nil, nil
	}
	key := stableScope(event.SourceSession, event.Provider, event.ProviderEventID)
	fingerprint := payloadFingerprint("", event.Kind, commands.Action{})
	if existing, ok := d.stable[key]; ok {
		if existing.fingerprint == fingerprint {
			return Decision{Class: ClassDuplicateProviderID}, nil, nil
		}
		return Decision{Class: ClassConflictingProviderID}, nil, nil
	}
	if d.atCapacityLocked() {
		return Decision{Class: ClassIgnoredNonCommand}, nil, nil
	}
	d.seq++
	entry := &stableEntry{
		fingerprint: fingerprint,
		expires:     now.Add(d.cfg.StableTTL),
		generation:  1,
	}
	d.stable[key] = entry
	d.pushExpiry(expiryItem{at: entry.expires, seq: d.seq, kind: expiryStable, key: key, generation: entry.generation})
	return Decision{Class: ClassIgnoredNonCommand}, nil, nil
}

func (d *Deduper) atCapacityLocked() bool {
	return len(d.stable)+len(d.windows) >= d.cfg.MaxEntries
}

// purgeLocked removes expired entries using the deterministic expiry heap. Only expired
// entries are removed; unexpired receipts are never evicted.
func (d *Deduper) purgeLocked(now time.Time) {
	for len(d.heap) > 0 {
		top := d.heap[0]
		if top.at.After(now) {
			return
		}
		item := heapPop(&d.heap)
		switch item.kind {
		case expiryStable:
			entry, ok := d.stable[item.key]
			if ok && entry.generation == item.generation && !now.Before(entry.expires) {
				delete(d.stable, item.key)
			}
		case expiryWindow:
			entry, ok := d.windows[item.key]
			if ok && entry.generation == item.generation && !now.Before(entry.expires) {
				delete(d.windows, item.key)
			}
		case expiryBucket:
			bucket, ok := d.buckets[item.actor]
			if !ok {
				continue
			}
			if !now.Before(bucket.lastSeen.Add(d.cfg.BucketIdleTTL)) {
				delete(d.buckets, item.actor)
				continue
			}
			// The bucket was used after this heap item was queued: re-queue at its
			// current idle expiry instead of growing the heap per event.
			d.pushExpiry(expiryItem{
				at:         bucket.lastSeen.Add(d.cfg.BucketIdleTTL),
				seq:        item.seq + 1,
				kind:       expiryBucket,
				actor:      item.actor,
				generation: bucket.generation,
			})
		}
	}
}

func windowExpiry(entry *windowEntry, window time.Duration) time.Time {
	var latest time.Time
	for _, sample := range entry.samples {
		if sample.at.After(latest) {
			latest = sample.at
		}
	}
	return latest.Add(window)
}

func validateAdmitInput(event source.Event, actor domain.ActorKey, action commands.Action) error {
	if event.SourceSession == "" || len(event.SourceSession) > 128 || !utf8.ValidString(event.SourceSession) {
		return fmt.Errorf("%w: session", ErrInvalidInput)
	}
	if event.Provider == "" || len(event.Provider) > 64 || !utf8.ValidString(event.Provider) {
		return fmt.Errorf("%w: provider", ErrInvalidInput)
	}
	if len(event.ProviderEventID) > 1024 || utf8.RuneCountInString(event.ProviderEventID) > 256 || !utf8.ValidString(event.ProviderEventID) {
		return fmt.Errorf("%w: provider event id", ErrInvalidInput)
	}
	if !event.Kind.Valid() {
		return fmt.Errorf("%w: event kind", ErrInvalidInput)
	}
	if event.Kind != source.KindComment {
		return nil
	}
	if !actor.Valid() {
		return fmt.Errorf("%w: actor key", ErrInvalidIdentity)
	}
	if !action.Valid() {
		return fmt.Errorf("%w: action", ErrInvalidAction)
	}
	return nil
}

// Reservation is the atomic in-memory claim for one accepted candidate. Finalize commits
// the receipt after a successful downstream operation; Release removes it and refunds the
// rate token after failure or cancellation. Methods are idempotent for the same terminal
// action and never panic.
type Reservation struct {
	mu         sync.Mutex
	deduper    *Deduper
	id         uint64
	stableKey  [32]byte
	windowKey  [32]byte
	sampleID   uint64
	actor      domain.ActorKey
	state      reservationState
	bestEffort bool
}

type reservationState uint8

const (
	reservationPending reservationState = iota + 1
	reservationFinalized
	reservationReleased
)

// Finalize commits a pending reservation. For a stable id the receipt becomes durable
// only in memory until W0305; for the best-effort path it keeps the accepted sample.
func (r *Reservation) Finalize() error {
	if r == nil {
		return ErrReservationClosed
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	switch r.state {
	case reservationFinalized:
		return nil
	case reservationReleased:
		return ErrReservationClosed
	}
	now := r.deduper.cfg.Now()
	d := r.deduper
	d.mu.Lock()
	defer d.mu.Unlock()
	defer d.maybeCompactLocked()
	d.purgeLocked(now)
	if r.bestEffort {
		entry, ok := d.windows[r.windowKey]
		index := -1
		if ok {
			for i := range entry.samples {
				if entry.samples[i].id == r.sampleID {
					index = i
					break
				}
			}
		}
		if !ok || index < 0 || !now.Before(entry.samples[index].pendingUntil) {
			if ok && index >= 0 {
				d.removeWindowSampleLocked(r.windowKey, entry, index)
			}
			d.refundLocked(r.actor, now)
			r.state = reservationReleased
			return ErrReservationExpired
		}
		entry.samples[index].pending = false
		r.state = reservationFinalized
		return nil
	}
	entry, ok := d.stable[r.stableKey]
	if !ok || entry.reservation != r.id || !entry.pending || !now.Before(entry.expires) {
		d.refundLocked(r.actor, now)
		r.state = reservationReleased
		return ErrReservationExpired
	}
	entry.pending = false
	entry.expires = now.Add(d.cfg.StableTTL)
	entry.generation++
	d.pushExpiry(expiryItem{at: entry.expires, seq: r.id, kind: expiryStable, key: r.stableKey, generation: entry.generation})
	r.state = reservationFinalized
	return nil
}

// Release abandons a pending reservation, removes its receipt/sample and refunds the rate
// token so a transient downstream failure does not permanently consume the event.
func (r *Reservation) Release() error {
	if r == nil {
		return nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	switch r.state {
	case reservationReleased:
		return nil
	case reservationFinalized:
		return ErrReservationClosed
	}
	now := r.deduper.cfg.Now()
	d := r.deduper
	d.mu.Lock()
	defer d.mu.Unlock()
	defer d.maybeCompactLocked()
	expired := true
	if r.bestEffort {
		if entry, ok := d.windows[r.windowKey]; ok {
			for index := range entry.samples {
				if entry.samples[index].id != r.sampleID {
					continue
				}
				expired = !now.Before(entry.samples[index].pendingUntil)
				d.removeWindowSampleLocked(r.windowKey, entry, index)
				break
			}
		}
	} else if entry, ok := d.stable[r.stableKey]; ok && entry.reservation == r.id && entry.pending {
		delete(d.stable, r.stableKey)
		expired = false
	}
	d.refundLocked(r.actor, now)
	r.state = reservationReleased
	if expired {
		return ErrReservationExpired
	}
	return nil
}

// removeWindowSampleLocked removes one sample, deletes an empty window and re-queues the
// window expiry when samples remain.
func (d *Deduper) removeWindowSampleLocked(key [32]byte, entry *windowEntry, index int) {
	entry.samples = append(entry.samples[:index], entry.samples[index+1:]...)
	if len(entry.samples) == 0 {
		delete(d.windows, key)
		return
	}
	entry.generation++
	entry.expires = windowExpiry(entry, d.cfg.Window)
	d.pushWindowExpiry(key, entry)
}
