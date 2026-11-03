import type { ModerationTarget, PrivacyEraseStatus } from '@/contracts/moderation';
import type { SessionSummary } from '@/contracts/session';
import type { VillageSummary } from '@/contracts/village';

/**
 * Deterministic demo fixtures for the explicit demo build only.
 *
 * They are frozen, contain no real people, no provider identifiers and no
 * actor keys. Every consumer must show the DEMO banner while they are used.
 */
export interface DemoData {
  readonly villages: readonly VillageSummary[];
  readonly sessions: readonly SessionSummary[];
  readonly moderation: readonly ModerationTarget[];
  readonly eraseRequests: readonly PrivacyEraseStatus[];
}

const VILLAGES: readonly VillageSummary[] = [
  {
    id: 'demo-village-01',
    name: 'Làng Ven Sông',
    lifecycle: 'running',
    source_status: 'mock',
    wave_no: 3,
    created_at: '2026-01-01T00:00:00Z',
  },
  {
    id: 'demo-village-02',
    name: 'Làng Đồi Gió',
    lifecycle: 'paused',
    source_status: 'mock',
    wave_no: 5,
    created_at: '2026-01-02T00:00:00Z',
  },
] as const;

const SESSIONS: readonly SessionSummary[] = [
  {
    id: 'demo-session-01',
    village_id: 'demo-village-01',
    state: 'running',
    wave_no: 3,
    source_status: 'mock',
    last_event_seq: '1042',
    started_at: '2026-01-03T09:00:00Z',
  },
  {
    id: 'demo-session-02',
    village_id: 'demo-village-02',
    state: 'paused',
    wave_no: 5,
    source_status: 'mock',
    last_event_seq: '2048',
    started_at: '2026-01-03T10:00:00Z',
  },
] as const;

const MODERATION: readonly ModerationTarget[] = [
  {
    public_ref: 'demo-ref-a1',
    display_alias: 'Người xem 001',
    role: 'fighter',
    blocked: false,
    hidden: false,
  },
  {
    public_ref: 'demo-ref-b2',
    // Deliberately hostile-looking alias: the UI must render it as text.
    display_alias: '<img src=x onerror="alert(1)">',
    role: 'builder',
    blocked: false,
    hidden: false,
  },
  {
    public_ref: 'demo-ref-c3',
    display_alias: 'Người xem 003',
    role: 'guard',
    blocked: true,
    hidden: true,
  },
] as const;

const ERASE_REQUESTS: readonly PrivacyEraseStatus[] = [
  {
    request_id: 'demo-erase-01',
    state: 'applied',
    requested_at: '2026-01-04T08:00:00Z',
    applied_at: '2026-01-04T08:00:05Z',
    detail: null,
  },
  {
    request_id: 'demo-erase-02',
    state: 'blocked',
    requested_at: '2026-01-04T09:00:00Z',
    applied_at: null,
    detail: 'Cần quyền admin và xác nhận thủ công (M07).',
  },
] as const;

export function demoData(): DemoData {
  return {
    villages: VILLAGES,
    sessions: SESSIONS,
    moderation: MODERATION,
    eraseRequests: ERASE_REQUESTS,
  };
}
