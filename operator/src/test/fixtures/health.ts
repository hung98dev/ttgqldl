import type { BackendHealth } from '@/contracts/health';
import type { HealthApi } from '@/api/health';
import type { DemoData } from '@/api/demo';
import type { AuthAdapter } from '@/features/authentication/auth';
import type { Role } from '@/contracts/api';

export function okHealth(checkedAt = 1_700_000_000_000, writerEpoch = 42): BackendHealth {
  return {
    health: { status: 'alive' },
    readiness: { status: 'ready', writer_epoch: writerEpoch },
    readinessDetail: null,
    checkedAt,
  };
}

export function degradedHealth(detail = 'readiness failed: postgres', checkedAt = 1_700_000_000_000): BackendHealth {
  return {
    health: { status: 'alive' },
    readiness: null,
    readinessDetail: detail,
    checkedAt,
  };
}

/** An API whose check() always resolves with the supplied health. */
export function staticHealthApi(health: BackendHealth): HealthApi {
  return { check: () => Promise.resolve(health) };
}

export function failingHealthApi(error: Error): HealthApi {
  return { check: () => Promise.reject(error) };
}

/** Auth adapter that reports a role without any network or storage. */
export function fixedAuth(role: Role): AuthAdapter {
  return {
    id: 'fixed',
    getSession: () => Promise.resolve({ kind: 'authenticated', role, displayName: `fixed-${role}` }),
    login: () => Promise.resolve({ kind: 'authenticated', role, displayName: `fixed-${role}` }),
    logout: () => Promise.resolve(),
  };
}

export function emptyDemoData(): DemoData {
  return { villages: [], sessions: [], moderation: [], eraseRequests: [] };
}
