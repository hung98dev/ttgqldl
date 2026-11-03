/** Moderation and privacy-erasure contracts (M07). */

export type ModerationAction = 'hide_participant' | 'block_participant';

export interface ModerationTarget {
  /** Server-assigned opaque reference; never an actor key or provider id. */
  readonly public_ref: string;
  readonly display_alias: string;
  readonly role: string;
  readonly blocked: boolean;
  readonly hidden: boolean;
}

export type PrivacyEraseState = 'queued' | 'applying' | 'applied' | 'blocked' | 'failed';

/** Status view of one erasure request. */
export interface PrivacyEraseStatus {
  readonly request_id: string;
  readonly state: PrivacyEraseState;
  readonly requested_at: string;
  readonly applied_at: string | null;
  /** Human-readable reason when the request is blocked. */
  readonly detail: string | null;
}

/** Operator metrics placeholders for the monitoring page (M08). */
export interface OperatorMetrics {
  readonly queue_depth: number;
  readonly dropped_events: number;
  readonly tick_rate_hz: number;
  readonly connected_viewers: number;
}
