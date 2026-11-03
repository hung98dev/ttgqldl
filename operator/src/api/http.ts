import { ApiError, isApiError } from './errors';

export interface HttpClientOptions {
  /** Same-origin by default; the Vite dev proxy forwards /healthz and /readyz. */
  readonly baseUrl?: string;
  readonly timeoutMs?: number;
  readonly fetchImpl?: typeof fetch;
  readonly retryDelayMs?: number;
  /**
   * Cross-origin API bases are refused unless this is set explicitly, because
   * every request carries cookies (`credentials: include`). A misconfiguration
   * must never forward a session to an unintended origin.
   */
  readonly allowCrossOrigin?: boolean;
  /** Injectable origin for tests; defaults to window.location.origin. */
  readonly origin?: string;
}

export interface RequestOptions {
  readonly signal?: AbortSignal;
  readonly timeoutMs?: number;
  /** Extra attempts after the first; only used for idempotent reads. */
  readonly retries?: number;
}

export interface HttpClient {
  /**
   * Performs a GET and validates the JSON body with `parse`. The validator is
   * part of the call so a response can never reach the UI unvalidated.
   */
  getJson<T>(path: string, parse: (json: unknown) => T, options?: RequestOptions): Promise<T>;
}

const DEFAULT_TIMEOUT_MS = 4000;
const DEFAULT_RETRY_DELAY_MS = 250;

function joinUrl(baseUrl: string, path: string): string {
  if (baseUrl === '') {
    return path;
  }
  return `${baseUrl.replace(/\/+$/, '')}${path.startsWith('/') ? path : `/${path}`}`;
}

function shouldRetry(error: ApiError): boolean {
  if (error.kind === 'network' || error.kind === 'timeout') {
    return true;
  }
  if (error.kind === 'http' && error.status !== null) {
    return error.status >= 500 && error.status !== 501;
  }
  return false;
}

/** Sleeps, but rejects immediately when the caller cancels. */
function abortableSleep(ms: number, signal?: AbortSignal): Promise<void> {
  if (signal?.aborted) {
    return Promise.reject(ApiError.aborted());
  }
  if (ms <= 0) {
    return Promise.resolve();
  }
  return new Promise<void>((resolve, reject) => {
    const timer = setTimeout(() => {
      signal?.removeEventListener('abort', onAbort);
      resolve();
    }, ms);
    function onAbort(): void {
      clearTimeout(timer);
      reject(ApiError.aborted());
    }
    signal?.addEventListener('abort', onAbort, { once: true });
  });
}

/**
 * Races a promise against a hard deadline so body parsing cannot hang forever.
 * The losing promise is ignored; nothing mutates state after the deadline.
 */
function withDeadline<T>(promise: Promise<T>, ms: number, timeoutMs: number): Promise<T> {
  if (!Number.isFinite(ms) || ms <= 0) {
    return Promise.reject(ApiError.timeout(timeoutMs));
  }
  return new Promise<T>((resolve, reject) => {
    let settled = false;
    const timer = setTimeout(() => {
      if (!settled) {
        settled = true;
        reject(ApiError.timeout(timeoutMs));
      }
    }, ms);
    promise.then(
      (value) => {
        if (!settled) {
          settled = true;
          clearTimeout(timer);
          resolve(value);
        }
      },
      (error: unknown) => {
        if (!settled) {
          settled = true;
          clearTimeout(timer);
          reject(error);
        }
      },
    );
  });
}

function resolveCredentials(options: HttpClientOptions): RequestCredentials {
  const baseUrl = options.baseUrl ?? '';
  if (!/^https?:\/\//i.test(baseUrl)) {
    // Relative base: the browser sends cookies to this origin only.
    return 'include';
  }

  let target: URL;
  try {
    target = new URL(baseUrl);
  } catch {
    throw new Error(`invalid API base URL: ${baseUrl}`);
  }

  const origin = options.origin ?? (typeof window !== 'undefined' ? window.location.origin : '');
  const sameOrigin = origin !== '' && target.origin === origin;

  if (!sameOrigin) {
    if (options.allowCrossOrigin !== true) {
      throw new Error(
        `refusing cross-origin API base ${target.origin}: cookies would leave ${origin || 'this origin'}. ` +
          'Use the same-origin proxy or set allowCrossOrigin explicitly.',
      );
    }
    // Even with explicit opt-in, never attach cookies to a foreign origin.
    return 'same-origin';
  }

  return 'include';
}

/**
 * Minimal typed JSON client.
 *
 * - one AbortController per attempt, composed with the caller's signal
 * - a single deadline covers connect, headers **and** body parsing
 * - retries are explicit, idempotent-only, abortable and never run after cancel
 * - all failures become ApiError; nothing throws raw DOMException into the UI
 * - `credentials: 'include'` for same-origin only, so the future HttpOnly
 *   session cookie and CSRF header flow work unchanged once M05 lands
 */
export function createHttpClient(options: HttpClientOptions = {}): HttpClient {
  const baseUrl = options.baseUrl ?? '';
  const defaultTimeoutMs = options.timeoutMs ?? DEFAULT_TIMEOUT_MS;
  const retryDelayMs = options.retryDelayMs ?? DEFAULT_RETRY_DELAY_MS;
  const fetchImpl = options.fetchImpl ?? globalThis.fetch;
  const credentials = resolveCredentials(options);

  async function attempt<T>(
    path: string,
    parse: (json: unknown) => T,
    request: RequestOptions,
  ): Promise<T> {
    const timeoutMs = request.timeoutMs ?? defaultTimeoutMs;
    const started = Date.now();
    const remaining = (): number => timeoutMs - (Date.now() - started);

    const controller = new AbortController();
    let timedOut = false;
    let callerAborted = false;

    const onCallerAbort = (): void => {
      callerAborted = true;
      controller.abort();
    };

    if (request.signal) {
      if (request.signal.aborted) {
        throw ApiError.aborted();
      }
      request.signal.addEventListener('abort', onCallerAbort, { once: true });
    }

    const timer = setTimeout(() => {
      timedOut = true;
      controller.abort();
    }, timeoutMs);

    try {
      let response: Response;
      try {
        response = await fetchImpl(joinUrl(baseUrl, path), {
          method: 'GET',
          headers: { Accept: 'application/json' },
          credentials,
          signal: controller.signal,
          cache: 'no-store',
        });
      } catch (cause) {
        if (callerAborted) {
          throw ApiError.aborted();
        }
        if (timedOut) {
          throw ApiError.timeout(timeoutMs);
        }
        throw ApiError.network(cause);
      }

      if (!response.ok) {
        const envelope = await withDeadline(readErrorEnvelope(response), remaining(), timeoutMs);
        throw new ApiError({
          kind: 'http',
          status: response.status,
          message: envelope?.error.message ?? `HTTP ${String(response.status)}`,
          code: envelope?.error.code ?? `HTTP_${String(response.status)}`,
          requestId: envelope?.error.request_id ?? '',
          retryable: envelope?.error.retryable ?? response.status >= 500,
        });
      }

      let json: unknown;
      try {
        json = await withDeadline(response.json(), remaining(), timeoutMs);
      } catch (cause) {
        if (isApiError(cause)) {
          throw cause;
        }
        throw ApiError.protocol(`${path} returned a body that is not valid JSON`);
      }

      try {
        return parse(json);
      } catch (cause) {
        if (isApiError(cause)) {
          throw cause;
        }
        throw ApiError.protocol(`${path} returned an unexpected body`);
      }
    } finally {
      clearTimeout(timer);
      request.signal?.removeEventListener('abort', onCallerAbort);
    }
  }

  const client: HttpClient = {
    async getJson<T>(path: string, parse: (json: unknown) => T, request: RequestOptions = {}) {
      const retries = Math.max(0, request.retries ?? 0);
      let lastError: ApiError | null = null;

      for (let index = 0; index <= retries; index += 1) {
        if (request.signal?.aborted) {
          throw ApiError.aborted();
        }
        try {
          return await attempt(path, parse, request);
        } catch (cause) {
          const error = isApiError(cause) ? cause : ApiError.network(cause);
          if (error.kind === 'aborted' || index === retries || !shouldRetry(error)) {
            throw error;
          }
          lastError = error;
          await abortableSleep(retryDelayMs * (index + 1), request.signal);
        }
      }

      throw lastError ?? ApiError.network('no attempt was made');
    },
  };

  return client;
}

async function readErrorEnvelope(
  response: Response,
): Promise<{ error: { code: string; message: string; request_id: string; retryable: boolean } } | null> {
  try {
    const json: unknown = await response.json();
    if (typeof json !== 'object' || json === null) {
      return null;
    }
    const body = json as { error?: unknown };
    const error = body.error;
    if (typeof error !== 'object' || error === null) {
      return null;
    }
    const detail = error as Record<string, unknown>;
    if (typeof detail['code'] !== 'string' || typeof detail['message'] !== 'string') {
      return null;
    }
    return {
      error: {
        code: detail['code'],
        message: detail['message'],
        request_id: typeof detail['request_id'] === 'string' ? detail['request_id'] : '',
        retryable: detail['retryable'] === true,
      },
    };
  } catch {
    return null;
  }
}
