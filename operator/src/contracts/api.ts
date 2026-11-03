/**
 * Cross-cutting API conventions from docs/plan/05_API_WEBSOCKET.md.
 *
 * These types are the single source of truth for the wire contract; feature
 * modules import them and never re-declare them.
 */

/** The three operator roles from docs/plan/06_LIVE_SECURITY.md. */
export type Role = 'viewer' | 'operator' | 'admin';

/** Every non-2xx backend response uses this envelope. */
export interface ApiErrorEnvelope {
  readonly error: {
    readonly code: string;
    readonly message: string;
    readonly request_id: string;
    readonly retryable: boolean;
  };
}

/** Pagination envelope used by list endpoints (M07). */
export interface Page<T> {
  readonly items: readonly T[];
  readonly next_cursor: string | null;
  readonly total: number | null;
}

/**
 * Idempotency metadata for mutating operator calls. Large sequence identifiers
 * travel as strings so no precision is lost in JavaScript.
 */
export interface IdempotencyMetadata {
  readonly idempotency_key: string;
  readonly request_hash: string;
  readonly replayed: boolean;
}

/**
 * Capability model for the operator UI.
 *
 * - AVAILABLE: the endpoint exists and was verified against the backend.
 * - UNAVAILABLE: the endpoint is specified but not implemented yet.
 * - BLOCKED: an external gate (permission, GATE-LIVE, licence) prevents use.
 * - ERROR: an implemented endpoint failed for this session.
 */
export type CapabilityState = 'AVAILABLE' | 'UNAVAILABLE' | 'BLOCKED' | 'ERROR';

export type CapabilityId =
  | 'health.read'
  | 'readiness.read'
  | 'villages.list'
  | 'villages.create'
  | 'sessions.list'
  | 'sessions.control'
  | 'monitoring.read'
  | 'moderation.search'
  | 'moderation.mutate'
  | 'privacy.erase'
  | 'auth.session'
  | 'live.source';

export interface Capability {
  readonly id: CapabilityId;
  readonly state: CapabilityState;
  /** Why the capability is not AVAILABLE; shown verbatim to the operator. */
  readonly reason: string;
  /** Owning milestone or gate, for traceability. */
  readonly owner: string;
}

/** Discriminated union for any async data. */
export type AsyncState<T, E = string> =
  | { readonly status: 'idle' }
  | { readonly status: 'loading' }
  | { readonly status: 'success'; readonly data: T }
  | { readonly status: 'error'; readonly error: E };
