/**
 * One error type for every failure the UI can show.
 *
 * `kind` drives presentation and retry behaviour; `code`/`requestId` come from
 * the backend envelope and are safe to display (they never contain secrets).
 */
export type ApiErrorKind = 'http' | 'network' | 'timeout' | 'protocol' | 'aborted';

export class ApiError extends Error {
  readonly kind: ApiErrorKind;
  readonly status: number | null;
  readonly code: string;
  readonly requestId: string;
  readonly retryable: boolean;

  constructor(init: {
    kind: ApiErrorKind;
    message: string;
    status?: number | null;
    code?: string;
    requestId?: string;
    retryable?: boolean;
  }) {
    super(init.message);
    this.name = 'ApiError';
    this.kind = init.kind;
    this.status = init.status ?? null;
    this.code = init.code ?? '';
    this.requestId = init.requestId ?? '';
    this.retryable = init.retryable ?? false;
  }

  static aborted(): ApiError {
    return new ApiError({ kind: 'aborted', message: 'request aborted' });
  }

  static timeout(timeoutMs: number): ApiError {
    return new ApiError({
      kind: 'timeout',
      message: `request timed out after ${String(timeoutMs)}ms`,
      retryable: true,
    });
  }

  static network(cause: unknown): ApiError {
    const detail = cause instanceof Error ? cause.message : 'unknown network error';
    return new ApiError({ kind: 'network', message: detail, retryable: true });
  }

  static protocol(message: string): ApiError {
    return new ApiError({ kind: 'protocol', message });
  }
}

export function isApiError(value: unknown): value is ApiError {
  return value instanceof ApiError;
}

/** Converts anything thrown into an ApiError without leaking internals. */
export function toApiError(value: unknown): ApiError {
  if (isApiError(value)) {
    return value;
  }
  if (value instanceof DOMException && value.name === 'AbortError') {
    return ApiError.aborted();
  }
  if (value instanceof Error) {
    return new ApiError({ kind: 'network', message: value.message, retryable: true });
  }
  return new ApiError({ kind: 'network', message: 'unexpected failure', retryable: true });
}
