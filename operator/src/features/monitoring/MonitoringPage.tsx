import { useT } from '@/app/i18n/LocaleProvider';
import { useHealthView } from '@/app/providers/HealthProvider';
import { capability } from '@/api/capabilities';
import { Card, StatCard } from '@/components/common/Card';
import { StatusBadge } from '@/components/common/StatusBadge';
import { ErrorState, LoadingState, UnavailableState } from '@/components/feedback/StateViews';

export function MonitoringPage() {
  const t = useT();
  const health = useHealthView();
  const metricsCapability = capability('monitoring.read');
  const liveCapability = capability('live.source');

  return (
    <div className="page">
      <header className="page__header">
        <h2 className="page__title">{t('monitoring.title')}</h2>
        <p className="page__subtitle">{t('monitoring.subtitle')}</p>
      </header>

      {health.status === 'loading' ? <LoadingState /> : null}
      {health.status === 'unreachable' ? (
        <ErrorState error={health.error} onRetry={health.refresh} retryLabel={t('health.retry')} />
      ) : null}

      <div className="stats">
        <StatCard
          label={t('monitoring.backend')}
          tone={health.status === 'ok' ? 'ok' : 'warning'}
          value={<StatusBadge tone={health.status === 'ok' ? 'ok' : 'warning'}>{health.status}</StatusBadge>}
        />
        <StatCard
          label={t('monitoring.database')}
          tone={health.backend?.readiness ? 'ok' : 'danger'}
          value={
            <StatusBadge tone={health.backend?.readiness ? 'ok' : 'danger'}>
              {health.backend?.readiness ? t('state.available') : t('health.notReady')}
            </StatusBadge>
          }
          hint={health.backend?.readiness ? `${t('health.writerEpoch')} ${String(health.backend.readiness.writer_epoch)}` : undefined}
        />
        <StatCard
          label={t('monitoring.source')}
          tone="danger"
          value={<StatusBadge tone="danger">{t('state.blocked')}</StatusBadge>}
          hint={liveCapability.reason}
        />
      </div>

      <Card title={t('monitoring.title')}>
        <div className="metric-grid">
          {[t('monitoring.queue'), t('monitoring.dropped'), t('monitoring.tick'), t('monitoring.viewers')].map(
            (label) => (
              <div className="metric" key={label}>
                <span className="metric__label">{label}</span>
                <span className="metric__value">—</span>
                <span className="metric__hint">{t('monitoring.placeholder')}</span>
              </div>
            ),
          )}
        </div>
      </Card>

      <UnavailableState
        title={t('monitoring.title')}
        reason={metricsCapability.reason}
        owner={metricsCapability.owner}
      />
    </div>
  );
}
