import { createContext, use, type ReactNode } from 'react';
import { useOperatorApi } from '@/app/providers/ApiProvider';
import { useHealth, type HealthView } from '@/hooks/useHealth';

const HealthContext = createContext<HealthView | null>(null);

function pollInterval(): number {
  const raw = import.meta.env['VITE_OPERATOR_HEALTH_POLL_MS'];
  const parsed = raw === undefined ? Number.NaN : Number.parseInt(raw, 10);
  return Number.isFinite(parsed) && parsed >= 0 ? parsed : 30_000;
}

/** One health poll for the whole console; pages read it, they never re-poll. */
export function HealthProvider({ children, pollMs }: { children: ReactNode; pollMs?: number }) {
  const api = useOperatorApi();
  const health = useHealth(api.health, { pollMs: pollMs ?? pollInterval() });
  return <HealthContext value={health}>{children}</HealthContext>;
}

export function useHealthView(): HealthView {
  const context = use(HealthContext);
  if (context === null) {
    throw new Error('useHealthView must be used inside HealthProvider');
  }
  return context;
}
