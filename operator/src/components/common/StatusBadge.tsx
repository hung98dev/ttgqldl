import type { ReactNode } from 'react';

export type BadgeTone = 'neutral' | 'ok' | 'warning' | 'danger' | 'info' | 'demo';

export function StatusBadge({ tone = 'neutral', children }: { tone?: BadgeTone; children: ReactNode }) {
  return <span className={`badge badge--${tone}`}>{children}</span>;
}
