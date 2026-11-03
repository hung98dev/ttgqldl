import { ApiError, isApiError } from './errors';

export interface HttpClientOptions {
  /** Same-origin by default; the Vite dev proxy forwards /healthz and /readyz. */
  readonly baseUrl?: string;
  readonly timeoutMs?: number;
  readonly fetchImpl?: typeof fetch;
  readonly retryDelayMs?: number;
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

function sleep(ms: number): Promise<void> {
  return new Promise((resolve) => {
    setTimeout(resolve, ms);
  });
}

/**
 * Minimal typed JSON client.
 *
 * - one AbortController per attempt, composed with the caller's signal
 * - bounded timeout per attempt
 * - retry only for idempotent reads and only for transient failures
 * - all failures become ApiError; nothing throws raw DOMException into the UI
 * - `credentials: 'include'` is set so the future HttpOnly session cookie and
 *   CSRF header flow work unchanged once M05 lands
 */
export function createHttpClient(options: HttpClientOptions = {}): HttpClient {
  const baseUrl = options.baseUrl ?? '';
  const defaultTimeoutMs = options.timeoutMs ?? DEFAULT_TIMEOUT_MS;
  const retryDelayMs = options.retryDelayMs ?? DEFAULT_RETRY_DELAY_MS;
  const fetchImpl = options.fetchImpl ?? globalThis.fetch;

  async function attempt<T>(
    path: string,
    parse: (json: unknown) => T,
    request: RequestOptions,
  ): Promise<T> {
    const timeoutMs = request.timeoutMs ?? defaultTimeoutMs;
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

    let response: Response;
    try {
      response = await fetchImpl(joinUrl(baseUrl, path), {
        method: 'GET',
        headers: { Accept: 'application/json' },
        credentials: 'include',
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
    } finally {
      clearTimeout(timer);
      request.signal?.removeEventListener('abort', onCallerAbort);
    }

    if (!response.ok) {
      const envelope = await readErrorEnvelope(response);
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
      json = await response.json();
    } catch {
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
  }

  const client: HttpClient = {
    async getJson<T>(path: string, parse: (json: unknown) => T, request: RequestOptions = {}) {
      const retries = Math.max(0, request.retries ?? 0);
      let lastError: ApiError | null = null;

      for (let index = 0; index <= retries; index += 1) {
        try {
          return await attempt(path, parse, request);
        } catch (cause) {
          const error = isApiError(cause) ? cause : ApiError.network(cause);
          if (error.kind === 'aborted' || index === retries || !shouldRetry(error)) {
            throw error;
          }
          lastError = error;
          await sleep(retryDelayMs * (index + 1));
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
