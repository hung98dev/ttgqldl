import { useCallback, useEffect, useRef, useState } from 'react';
import type { HealthApi } from '@/api/health';
import { toApiError, type ApiError } from '@/api/errors';
import type { BackendHealth } from '@/contracts/health';

export type HealthStatus = 'loading' | 'ok' | 'degraded' | 'unreachable';

export interface HealthView {
  readonly status: HealthStatus;
  readonly backend: BackendHealth | null;
  readonly error: ApiError | null;
  readonly refreshing: boolean;
  readonly refresh: () => void;
}

interface HealthState {
  readonly status: HealthStatus;
  readonly backend: BackendHealth | null;
  readonly error: ApiError | null;
}

const INITIAL: HealthState = { status: 'loading', backend: null, error: null };

/**
 * Polls /healthz and /readyz.
 *
 * Every request is abortable; unmounting cancels the in-flight probe so a slow
 * backend cannot update state after the page is gone. A 503 from /readyz is a
 * `degraded` answer, not a transport failure.
 */
export function useHealth(api: HealthApi, options: { pollMs?: number } = {}): HealthView {
  const pollMs = options.pollMs ?? 30_000;
  const [state, setState] = useState<HealthState>(INITIAL);
  const [refreshing, setRefreshing] = useState(false);
  const controllerRef = useRef<AbortController | null>(null);
  const mountedRef = useRef(true);
  const hasDataRef = useRef(false);

  const run = useCallback(async () => {
    controllerRef.current?.abort();
    const controller = new AbortController();
    controllerRef.current = controller;

    if (hasDataRef.current) {
      setRefreshing(true);
    } else {
      setState((current) => ({ ...current, status: 'loading' }));
    }

    try {
      const backend = await api.check(controller.signal);
      if (!mountedRef.current || controller.signal.aborted) {
        return;
      }
      hasDataRef.current = true;
      setState({
        status: backend.readiness === null ? 'degraded' : 'ok',
        backend,
        error: null,
      });
    } catch (cause) {
      if (!mountedRef.current || controller.signal.aborted) {
        return;
      }
      const error = toApiError(cause);
      if (error.kind === 'aborted') {
        return;
      }
      hasDataRef.current = false;
      setState({ status: 'unreachable', backend: null, error });
    } finally {
      if (mountedRef.current) {
        setRefreshing(false);
      }
    }
  }, [api]);

  useEffect(() => {
    mountedRef.current = true;
    void run();

    const onVisible = (): void => {
      if (document.visibilityState === 'visible') {
        void run();
      }
    };

    const timer = pollMs > 0 ? window.setInterval(() => void run(), pollMs) : null;
    document.addEventListener('visibilitychange', onVisible);

    return () => {
      mountedRef.current = false;
      controllerRef.current?.abort();
      if (timer !== null) {
        window.clearInterval(timer);
      }
      document.removeEventListener('visibilitychange', onVisible);
    };
  }, [run, pollMs]);

  const refresh = useCallback(() => {
    void run();
  }, [run]);

  return { ...state, refreshing, refresh };
}
