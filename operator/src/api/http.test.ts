import { describe, expect, it, vi } from 'vitest';
import { createHttpClient } from '@/api/http';
import { ApiError } from '@/api/errors';
import { parseHealthResponse } from '@/api/guards';

interface FakeResponseInit {
  readonly status?: number;
  readonly body?: unknown;
  readonly invalidJson?: boolean;
}

function fakeResponse(init: FakeResponseInit = {}): Response {
  const status = init.status ?? 200;
  const response = {
    ok: status >= 200 && status < 300,
    status,
    json: () =>
      init.invalidJson === true
        ? Promise.reject(new SyntaxError('unexpected token'))
        : Promise.resolve(init.body),
  };
  return response as unknown as Response;
}

function envelope(code: string, message: string, retryable: boolean): unknown {
  return { error: { code, message, request_id: 'req-1', retryable } };
}

describe('http client', () => {
  it('returns the validated body on success', async () => {
    const fetchImpl = vi.fn<typeof fetch>().mockResolvedValue(fakeResponse({ body: { status: 'alive' } }));
    const client = createHttpClient({ fetchImpl });

    await expect(client.getJson('/healthz', parseHealthResponse)).resolves.toEqual({ status: 'alive' });
    expect(fetchImpl).toHaveBeenCalledTimes(1);
    const [url, init] = fetchImpl.mock.calls[0] ?? [];
    expect(url).toBe('/healthz');
    expect(init?.credentials).toBe('include');
  });

  it('converts a 503 error envelope into an ApiError with the backend reason', async () => {
    const fetchImpl = vi
      .fn<typeof fetch>()
      .mockResolvedValue(fakeResponse({ status: 503, body: envelope('UNAVAILABLE', 'readiness failed: postgres', true) }));
    const client = createHttpClient({ fetchImpl });

    const failure = await client.getJson('/readyz', parseHealthResponse).catch((error: unknown) => error);

    expect(failure).toBeInstanceOf(ApiError);
    const error = failure as ApiError;
    expect(error.kind).toBe('http');
    expect(error.status).toBe(503);
    expect(error.code).toBe('UNAVAILABLE');
    expect(error.message).toBe('readiness failed: postgres');
    expect(error.requestId).toBe('req-1');
  });

  it('reports a malformed JSON body as a protocol error', async () => {
    const fetchImpl = vi.fn<typeof fetch>().mockResolvedValue(fakeResponse({ invalidJson: true }));
    const client = createHttpClient({ fetchImpl });

    const failure = await client.getJson('/healthz', parseHealthResponse).catch((error: unknown) => error);
    expect((failure as ApiError).kind).toBe('protocol');
  });

  it('rejects a structurally wrong body instead of passing it to the UI', async () => {
    const fetchImpl = vi.fn<typeof fetch>().mockResolvedValue(fakeResponse({ body: { status: 'dead' } }));
    const client = createHttpClient({ fetchImpl });

    const failure = await client.getJson('/healthz', parseHealthResponse).catch((error: unknown) => error);
    expect((failure as ApiError).kind).toBe('protocol');
    expect((failure as ApiError).message).toContain('health.status');
  });

  it('retries a transient network failure and then succeeds', async () => {
    const fetchImpl = vi
      .fn<typeof fetch>()
      .mockRejectedValueOnce(new TypeError('connection refused'))
      .mockResolvedValueOnce(fakeResponse({ body: { status: 'alive' } }));
    const client = createHttpClient({ fetchImpl, retryDelayMs: 0 });

    await expect(client.getJson('/healthz', parseHealthResponse, { retries: 1 })).resolves.toEqual({
      status: 'alive',
    });
    expect(fetchImpl).toHaveBeenCalledTimes(2);
  });

  it('does not retry a 404', async () => {
    const fetchImpl = vi
      .fn<typeof fetch>()
      .mockResolvedValue(fakeResponse({ status: 404, body: envelope('NOT_FOUND', 'no route', false) }));
    const client = createHttpClient({ fetchImpl, retryDelayMs: 0 });

    const failure = await client.getJson('/missing', parseHealthResponse, { retries: 2 }).catch(
      (error: unknown) => error,
    );
    expect((failure as ApiError).status).toBe(404);
    expect(fetchImpl).toHaveBeenCalledTimes(1);
  });

  it('aborts on timeout and reports a timeout error', async () => {
    const hanging: typeof fetch = (_input, init) =>
      new Promise<Response>((_resolve, reject) => {
        init?.signal?.addEventListener('abort', () => {
          reject(new DOMException('Aborted', 'AbortError'));
        });
      });
    const client = createHttpClient({ fetchImpl: hanging, timeoutMs: 5 });

    const failure = await client.getJson('/healthz', parseHealthResponse).catch((error: unknown) => error);
    expect((failure as ApiError).kind).toBe('timeout');
  });

  it('propagates a caller abort without retrying', async () => {
    const hanging: typeof fetch = (_input, init) =>
      new Promise<Response>((_resolve, reject) => {
        init?.signal?.addEventListener('abort', () => {
          reject(new DOMException('Aborted', 'AbortError'));
        });
      });
    const client = createHttpClient({ fetchImpl: hanging, timeoutMs: 5_000 });
    const controller = new AbortController();
    const pending = client.getJson('/healthz', parseHealthResponse, {
      signal: controller.signal,
      retries: 3,
    });
    controller.abort();

    const failure = await pending.catch((error: unknown) => error);
    expect((failure as ApiError).kind).toBe('aborted');
  });
});

describe('http client hardening', () => {
  it('refuses a cross-origin API base that would leak cookies', () => {
    expect(() =>
      createHttpClient({ baseUrl: 'https://evil.example.com', origin: 'http://localhost:5173' }),
    ).toThrow(/refusing cross-origin API base/);
  });

  it('allows an explicit cross-origin base but never sends cookies there', async () => {
    const fetchImpl = vi.fn<typeof fetch>().mockResolvedValue(fakeResponse({ body: { status: 'alive' } }));
    const client = createHttpClient({
      baseUrl: 'https://api.example.com',
      origin: 'http://localhost:5173',
      allowCrossOrigin: true,
      fetchImpl,
    });

    await client.getJson('/healthz', parseHealthResponse);
    const [, init] = fetchImpl.mock.calls[0] ?? [];
    expect(init?.credentials).toBe('same-origin');
  });

  it('sends cookies for a same-origin absolute base', async () => {
    const fetchImpl = vi.fn<typeof fetch>().mockResolvedValue(fakeResponse({ body: { status: 'alive' } }));
    const client = createHttpClient({
      baseUrl: 'https://console.example.com',
      origin: 'https://console.example.com',
      fetchImpl,
    });

    await client.getJson('/healthz', parseHealthResponse);
    const [, init] = fetchImpl.mock.calls[0] ?? [];
    expect(init?.credentials).toBe('include');
  });

  it('times out when the response body never finishes parsing', async () => {
    const hangingBody = {
      ok: true,
      status: 200,
      json: () => new Promise<unknown>(() => undefined),
    } as unknown as Response;
    const fetchImpl = vi.fn<typeof fetch>().mockResolvedValue(hangingBody);
    const client = createHttpClient({ fetchImpl, timeoutMs: 10 });

    const failure = await client.getJson('/healthz', parseHealthResponse).catch((error: unknown) => error);
    expect((failure as ApiError).kind).toBe('timeout');
  });

  it('stops retrying when the caller cancels during backoff', async () => {
    const fetchImpl = vi.fn<typeof fetch>().mockRejectedValue(new TypeError('connection refused'));
    const client = createHttpClient({ fetchImpl, retryDelayMs: 5_000 });
    const controller = new AbortController();
    const pending = client.getJson('/healthz', parseHealthResponse, {
      signal: controller.signal,
      retries: 3,
    });

    setTimeout(() => {
      controller.abort();
    }, 20);

    const failure = await pending.catch((error: unknown) => error);
    expect((failure as ApiError).kind).toBe('aborted');
    expect(fetchImpl).toHaveBeenCalledTimes(1);
  });

  it('does not start a retry after the caller has already cancelled', async () => {
    const fetchImpl = vi.fn<typeof fetch>().mockResolvedValue(fakeResponse({ body: { status: 'alive' } }));
    const client = createHttpClient({ fetchImpl });
    const controller = new AbortController();
    controller.abort();

    const failure = await client
      .getJson('/healthz', parseHealthResponse, { signal: controller.signal, retries: 2 })
      .catch((error: unknown) => error);
    expect((failure as ApiError).kind).toBe('aborted');
    expect(fetchImpl).not.toHaveBeenCalled();
  });
});