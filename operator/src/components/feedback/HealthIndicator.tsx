import { useT } from '@/app/i18n/LocaleProvider';
import { StatusBadge } from '@/components/common/StatusBadge';
import type { BadgeTone } from '@/components/common/StatusBadge';
import type { HealthView } from '@/hooks/useHealth';

export function healthBadge(health: HealthView): { tone: BadgeTone; label: string } {
  switch (health.status) {
    case 'ok':
      return { tone: 'ok', label: 'READY' };
    case 'degraded':
      return { tone: 'warning', label: 'DEGRADED' };
    case 'unreachable':
      return { tone: 'danger', label: 'OFFLINE' };
    default:
      return { tone: 'neutral', label: '…' };
  }
}

export function HealthIndicator({ health }: { health: HealthView }) {
  const t = useT();
  const badge = healthBadge(health);
  const backend = health.backend;

  return (
    <div className="health-indicator" aria-live="polite">
      <StatusBadge tone={badge.tone}>{badge.label}</StatusBadge>
      <span className="health-indicator__text">
        {health.status === 'loading'
          ? t('health.checking')
          : backend === null
            ? t('health.dead')
            : backend.readiness === null
              ? t('health.notReady')
              : t('health.ready')}
      </span>
      {backend ? (
        <span className="health-indicator__meta">
          {t('health.checkedAt')} {new Date(backend.checkedAt).toLocaleTimeString('vi-VN')}
        </span>
      ) : null}
    </div>
  );
}
