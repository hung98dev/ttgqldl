import type { Capability, CapabilityId } from '@/contracts/api';

/**
 * The single capability matrix.
 *
 * It is intentionally static: the operator must never infer capability from a
 * successful /healthz. Only the two implemented endpoints are AVAILABLE; every
 * other entry names the owning milestone and stays disabled in the UI.
 */
export const CAPABILITIES: readonly Capability[] = [
  {
    id: 'health.read',
    state: 'AVAILABLE',
    reason: 'GET /healthz is implemented by the Go backend and verified.',
    owner: 'M01',
  },
  {
    id: 'readiness.read',
    state: 'AVAILABLE',
    reason: 'GET /readyz is implemented and returns 503 with an error envelope when a dependency fails.',
    owner: 'M01',
  },
  {
    id: 'auth.session',
    state: 'UNAVAILABLE',
    reason: 'Operator login, HttpOnly session cookie and CSRF flow are specified in M05 and not implemented.',
    owner: 'M05',
  },
  {
    id: 'villages.list',
    state: 'UNAVAILABLE',
    reason: 'GET /admin/v1/villages is not implemented yet.',
    owner: 'M07',
  },
  {
    id: 'villages.create',
    state: 'UNAVAILABLE',
    reason: 'POST /admin/v1/villages is not implemented yet and requires admin authorization.',
    owner: 'M07',
  },
  {
    id: 'sessions.list',
    state: 'UNAVAILABLE',
    reason: 'Session listing depends on M05 session control and M07 read models.',
    owner: 'M05',
  },
  {
    id: 'sessions.control',
    state: 'UNAVAILABLE',
    reason: 'start/pause/resume/stop require the M05 session API with idempotency keys.',
    owner: 'M05',
  },
  {
    id: 'monitoring.read',
    state: 'UNAVAILABLE',
    reason: 'Queue, drop and tick metrics endpoints are M08 work; /metrics is not exposed to the browser.',
    owner: 'M08',
  },
  {
    id: 'moderation.search',
    state: 'UNAVAILABLE',
    reason: 'Participant search is M07 work and requires operator authorization.',
    owner: 'M07',
  },
  {
    id: 'moderation.mutate',
    state: 'UNAVAILABLE',
    reason: 'Hide/block actions are M07 work and must be audited server-side.',
    owner: 'M07',
  },
  {
    id: 'privacy.erase',
    state: 'UNAVAILABLE',
    reason: 'Privacy erase requests and ledger export are M04/M07 work.',
    owner: 'M04',
  },
  {
    id: 'live.source',
    state: 'BLOCKED',
    reason: 'GATE-LIVE is blocked: no TikTok LIVE permission, approved source or production credential exists.',
    owner: 'M10',
  },
] as const;

const byId = new Map<CapabilityId, Capability>(CAPABILITIES.map((item) => [item.id, item]));

export function capability(id: CapabilityId): Capability {
  const found = byId.get(id);
  if (!found) {
    throw new Error(`unknown capability ${id}`);
  }
  return found;
}

export function isCapabilityAvailable(id: CapabilityId): boolean {
  return capability(id).state === 'AVAILABLE';
}

export function unavailableCapabilities(): readonly Capability[] {
  return CAPABILITIES.filter((item) => item.state !== 'AVAILABLE');
}
