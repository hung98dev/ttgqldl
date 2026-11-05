// Package source defines the input source adapter boundary for TTGQLDL described
// by docs/plan/06_LIVE_SECURITY.md. M03-W0302 implements the deterministic mock and
// replay adapters only. The authorized LIVE adapter belongs to M10 and remains
// blocked on GATE-LIVE; no adapter here calls a provider, reads cookies, scrapes,
// or claims provider authorization.
//
// Contract:
//
//   - Source is one adapter per session: Start(ctx, session, sink), Stop(ctx),
//     Health(ctx), Capabilities(ctx), Reconnect(ctx, cursor, sink).
//   - Start and Reconnect deliver synchronously on the caller's goroutine until EOF,
//     cancellation, Stop or a delivery failure. No goroutine is spawned internally.
//   - Sink(ctx, event) returns an error to reject/backpressure an event. The
//     delivery cursor advances only after a successful sink call, so a failed
//     callback is redelivered on reconnect. Sources never claim durable
//     exactly-once ingestion; durable acknowledgement is W0305.
//   - Cursor is a versioned, session- and source-bound position ("next event"
//     index). Wall-clock timestamps are never cursors, and a cursor from another
//     session or fixture is rejected.
//   - Events carry untrusted provider metadata. ValidateEvent applies the source
//     trust boundary (UTF-8, 500-byte comment, identifier/alias bounds, control and
//     bidi rejection) before anything reaches the command parser. Command grammar is
//     a separate parser concern.
//   - Only event_kind=comment may become a game command. Gift/like/share/follow/join
//     events are valid source events and must never grant a role, vote, XP or combat
//     advantage.
//   - Mock is deterministic from (session, seed, scenario, count): fixed synthetic
//     timestamps, stable synthetic event/viewer ids, no wall clock, no global
//     randomness, no network. Its synthetic identifiers are not authorized provider
//     identities.
//   - Replay streams committed NDJSON fixtures in file order. A manifest pins each
//     fixture's exact SHA256 and record count; the digest is the source identity used
//     by cursors. Records are decoded strictly (unknown fields, schema versions,
//     duplicate/out-of-order sequences, session/provider mismatches are rejected) and
//     malformed records are never silently skipped. The file is streamed line by line
//     with a 64 KiB record bound and bounded hashing memory; it is never loaded
//     whole.
//   - Script provenance is immutable. A mock owns a deep copy of its event slice and
//     delivers timestamp clones; caller or sink mutation cannot change delivered data
//     under an accepted source id. The mock source-script id is version 2, a
//     domain-separated length-prefixed encoding over every delivery-relevant field
//     (including timestamp presence); version-1 cursors fail closed with a mismatch.
//   - Replay verifies fixture bytes into a private snapshot and streams only that
//     snapshot, so replacing or mutating the original file after construction cannot
//     alter emitted records under the old digest. A reconnect re-verifies and fails
//     closed if the bytes changed. Symlink-resolved paths must remain inside the
//     configured fixture directory. ReplayConfig.MaxFixtureBytes (default 64 MiB) bounds
//     the fixture and snapshot; oversized/growing files, snapshot I/O failures and
//     cancellation fail closed without delivering a partial snapshot, and snapshot
//     temporary files are removed on every path.
//   - Identity is NOT derived here. Source emits validated raw provider metadata for
//     the identity layer (backend/internal/identity), which is the only approved
//     ActorKey path.
//
// Health is separate from the game lifecycle, and the simulation's pause-reason
// contract remains authoritative: a disconnected source does not automatically
// resume gameplay.
package source
