import type { BackendHealth } from '@/contracts/health';
import { ApiError, isApiError, toApiError } from './errors';
import { parseHealthResponse, parseReadyResponse } from './guards';
import type { HttpClient } from './http';

export interface HealthApi {
  /**
   * Reads /healthz then /readyz.
   *
   * A 503 from /readyz is a normal, expected answer (the process is alive but a
   * dependency is down) and returns `readiness: null` plus the backend reason.
   * Any other failure propagates as an ApiError so the UI shows ERROR, not a
   * fabricated healthy state.
   */
  check(signal?: AbortSignal): Promise<BackendHealth>;
}

export interface HealthApiOptions {
  readonly now?: () => number;
  /** Attempts for /healthz; health checks are idempotent reads. */
  readonly retries?: number;
}

export function createHealthApi(client: HttpClient, options: HealthApiOptions = {}): HealthApi {
  const now = options.now ?? (() => Date.now());
  const retries = options.retries ?? 2;

  return {
    async check(signal) {
      const health = await client.getJson('/healthz', parseHealthResponse, { signal, retries });

      try {
        const readiness = await client.getJson('/readyz', parseReadyResponse, {
          signal,
          retries: 1,
        });
        return {
          health,
          readiness,
          readinessDetail: null,
          checkedAt: now(),
        };
      } catch (cause) {
        const error = toApiError(cause);
        if (error.kind === 'http' && error.status === 503) {
          return {
            health,
            readiness: null,
            readinessDetail: error.message,
            checkedAt: now(),
          };
        }
        if (error.kind === 'aborted') {
          throw error;
        }
        // The process answered /healthz but /readyz could not be read at all.
        throw new ApiError({
          kind: error.kind,
          status: error.status,
          code: error.code || 'READINESS_UNREADABLE',
          message: error.message,
          requestId: error.requestId,
          retryable: true,
        });
      }
    },
  };
}

export function isHealthFailure(value: unknown): value is ApiError {
  return isApiError(value);
}
