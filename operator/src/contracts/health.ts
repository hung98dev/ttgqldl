/**
 * GET /healthz and GET /readyz, exactly as implemented by
 * backend/internal/app/handlers.go.
 */

/** 200 body of GET /healthz. */
export interface HealthResponse {
  readonly status: 'alive';
}

/** 200 body of GET /readyz. `writer_epoch` is a 64-bit value. */
export interface ReadyResponse {
  readonly status: 'ready';
  readonly writer_epoch: number;
}

/** Health of the console's own view of the backend. */
export interface BackendHealth {
  readonly health: HealthResponse;
  /** Null when readiness returned 503: the process is alive but cannot serve. */
  readonly readiness: ReadyResponse | null;
  /** Backend-provided reason when readiness failed. */
  readonly readinessDetail: string | null;
  readonly checkedAt: number;
}
