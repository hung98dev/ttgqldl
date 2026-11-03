import type { Capability } from './api';

/** Village lifecycle as published by the backend (mirrors the Unity client). */
export type VillageLifecycle =
  | 'waiting'
  | 'running'
  | 'paused'
  | 'source_disconnected'
  | 'storage_unavailable'
  | 'stopped';

export type SourceStatus = 'connected' | 'disconnected' | 'mock' | 'replay';

/** Row model for the village list (M07 endpoint, not implemented yet). */
export interface VillageSummary {
  readonly id: string;
  readonly name: string;
  readonly lifecycle: VillageLifecycle;
  readonly source_status: SourceStatus;
  readonly wave_no: number;
  readonly created_at: string;
}

/** Body of a future POST /admin/v1/villages. */
export interface CreateVillageRequest {
  readonly name: string;
  readonly region: string;
  readonly max_wall_hp: number;
}

export interface CapabilityGate {
  readonly capability: Capability;
  readonly allowed: boolean;
}
