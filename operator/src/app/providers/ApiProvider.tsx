import { createContext, use, useMemo, type ReactNode } from 'react';
import type { HealthApi } from '@/api/health';
import type { DemoData } from '@/api/demo';

export interface OperatorApi {
  readonly health: HealthApi;
}

interface ApiContextValue {
  readonly api: OperatorApi;
  readonly demo: boolean;
  readonly demoData: DemoData | null;
  readonly apiBaseUrl: string;
}

const ApiContext = createContext<ApiContextValue | null>(null);

export function ApiProvider({
  children,
  api,
  demo = false,
  demoData = null,
  apiBaseUrl = '',
}: {
  children: ReactNode;
  api: OperatorApi;
  demo?: boolean;
  demoData?: DemoData | null;
  apiBaseUrl?: string;
}) {
  const value = useMemo<ApiContextValue>(
    () => ({ api, demo, demoData: demo ? demoData : null, apiBaseUrl }),
    [api, demo, demoData, apiBaseUrl],
  );
  return <ApiContext value={value}>{children}</ApiContext>;
}

export function useOperatorApi(): OperatorApi {
  const context = use(ApiContext);
  if (context === null) {
    throw new Error('useOperatorApi must be used inside ApiProvider');
  }
  return context.api;
}

export function useApiConfig(): ApiContextValue {
  const context = use(ApiContext);
  if (context === null) {
    throw new Error('useApiConfig must be used inside ApiProvider');
  }
  return context;
}
