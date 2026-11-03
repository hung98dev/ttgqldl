/**
 * Village contracts, aligned with docs/plan/05_API_WEBSOCKET.md:
 *   POST /admin/v1/villages {display_name:1..64, source}
 *     -> 201 {id, display_name, level:1, wall_tier:1}
 *   GET  /admin/v1/villages?... -> 200 {items:[VillageSummary], next_cursor}
 *
 * Nothing here adds fields the specification does not define.
 */

/** Village lifecycle as published by the backend. */
export type VillageLifecycle =
  | 'waiting'
  | 'running'
  | 'paused'
  | 'source_disconnected'
  | 'storage_unavailable'
  | 'stopped';

export type SourceStatus = 'connected' | 'disconnected' | 'mock' | 'replay';

/** Allowed `source` values for creation. `authorized_live` is gated. */
export type CreateVillageSource = 'mock' | 'replay' | 'authorized_live';

/** Row model for the village list (M07 endpoint, not implemented yet). */
export interface VillageSummary {
  readonly id: string;
  readonly display_name: string;
  readonly lifecycle: VillageLifecycle;
  readonly source_status: SourceStatus;
  readonly wave_no: number;
  readonly created_at: string;
}

/** Body of a future POST /admin/v1/villages. */
export interface CreateVillageRequest {
  readonly display_name: string;
  readonly source: CreateVillageSource;
}
