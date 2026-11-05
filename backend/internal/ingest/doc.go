// Package ingest owns the W0304 in-memory ingress boundary: provider-event
// deduplication, conflicting-payload detection, best-effort suppression of repeated
// commands without stable event ids, per-actor token-bucket rate limiting and explicit
// reservation/finalization so a failed downstream operation does not permanently consume
// a game event.
//
// This is NOT durable exactly-once ingestion. ACCEPT_CANDIDATE only means the event may
// proceed to the future W0305 database transaction and the existing simulation reducer;
// only a committed input_receipts row plus reducer acceptance establishes durable
// processing. In-memory receipts are bounded and expire, and a cache-capacity failure is
// reported explicitly rather than silently weakening the guarantee.
//
// Stable provider event ids are scoped by (session, provider, event_id); they are never
// viewer identities and are never treated as globally unique. Missing-event-id events use
// a bounded sliding window over (session, scoped actor key, canonical command), never raw
// comment text, aliases, provider timestamps or per-message randomness.
//
// W0305 adds the PostgreSQL-authoritative append boundary: PostgresAppender locks the
// session row, validates mode/provider/lifecycle, classifies durable receipts, allocates
// consecutive input sequences, appends immutable input_events and fingerprint-bound
// receipts, and acknowledges only after commit. Batcher provides the bounded 256/50 ms
// FIFO queue (20,000 default) with explicit queue-full backpressure and graceful flush.
// It never mutates reducer state, marks applied_at_tick, or claims exactly-once gameplay;
// M04 owns reducer application, fencing and recovery.
//
// W0306 adds a fixed-size Metrics registry: stage counters (received/validated), the
// final W0304 classification, queue depth/utilization/high-water, batch/transaction and
// durable outcomes, bounded commit-latency buckets and explicit lost-game-event
// accounting. Sustained queue >75% for 30s, sustained internal p95>1s for 5min (with at
// least 20 samples), storage failure, unresolved commit-unknown and any confirmed loss
// produce bounded alert codes. Metrics use only atomic counters and a fixed histogram;
// no labels, identities, comments or event ids are retained. Metrics.Handler is a
// read-only JSON endpoint intentionally not mounted by the M01 server skeleton: there is
// no wired ingestion pipeline yet, and exporting fabricated zeros would be dishonest.
// M04/M07 must mount it only with a real registry.
//
// The package is deterministic under an injected clock, race-free, bounded and
// privacy-conscious: no raw comments, aliases or provider viewer ids are retained, and
// diagnostics expose counts only.
package ingest
