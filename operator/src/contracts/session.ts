import type { SourceStatus } from './village';

/** Runtime state of a game session (matches the Unity lifecycle vocabulary). */
export type SessionState = 'idle' | 'running' | 'paused' | 'stopped';

/** Row model for the session list (M05/M07 endpoint, not implemented yet). */
export interface SessionSummary {
  readonly id: string;
  readonly village_id: string;
  readonly state: SessionState;
  readonly wave_no: number;
  readonly source_status: SourceStatus;
  /** Monotonic event sequence; a 64-bit value carried as a string. */
  readonly last_event_seq: string;
  readonly started_at: string | null;
}

/** Operator controls for a session. */
export type SessionAction = 'start' | 'pause' | 'resume' | 'stop';

/** Actions that change authoritative state and therefore need a confirmation. */
export const DESTRUCTIVE_SESSION_ACTIONS: readonly SessionAction[] = ['stop'];
