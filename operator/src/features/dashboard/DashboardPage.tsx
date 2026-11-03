import { useT } from '@/app/i18n/LocaleProvider';
import { useApiConfig } from '@/app/providers/ApiProvider';
import { useHealthView } from '@/app/providers/HealthProvider';
import { Button } from '@/components/common/Button';
import { Card, StatCard } from '@/components/common/Card';
import { DataTable, type Column } from '@/components/common/DataTable';
import { StatusBadge, type BadgeTone } from '@/components/common/StatusBadge';
import { ErrorState, LoadingState } from '@/components/feedback/StateViews';
import { CAPABILITIES } from '@/api/capabilities';
import type { Capability } from '@/contracts/api';

const capabilityTone: Record<Capability['state'], BadgeTone> = {
  AVAILABLE: 'ok',
  UNAVAILABLE: 'neutral',
  BLOCKED: 'danger',
  ERROR: 'warning',
};

export function DashboardPage() {
  const t = useT();
  const health = useHealthView();
  const { demo, demoData } = useApiConfig();

  if (health.status === 'loading') {
    return <LoadingState />;
  }

  if (health.status === 'unreachable') {
    return (
      <ErrorState
        title={t('health.dead')}
        error={health.error}
        onRetry={health.refresh}
        retryLabel={t('health.retry')}
      />
    );
  }

  const readiness = health.backend?.readiness ?? null;
  const capabilityColumns: Column<Capability>[] = [
    { key: 'capability', header: t('nav.dashboard'), render: (row) => <code>{row.id}</code> },
    {
      key: 'state',
      header: t('state.available'),
      render: (row) => <StatusBadge tone={capabilityTone[row.state]}>{row.state}</StatusBadge>,
    },
    { key: 'owner', header: t('dashboard.capabilities.owner'), render: (row) => row.owner },
    { key: 'reason', header: t('dashboard.capabilities.reason'), render: (row) => row.reason },
  ];

  return (
    <div className="page">
      <header className="page__header">
        <div>
          <h2 className="page__title">{t('dashboard.title')}</h2>
          <p className="page__subtitle">{t('dashboard.subtitle')}</p>
        </div>
        <Button variant="secondary" busy={health.refreshing} onClick={health.refresh}>
          {health.refreshing ? t('health.retrying') : t('health.retry')}
        </Button>
      </header>

      <div className="stats">
        <StatCard
          label={t('dashboard.backend.title')}
          tone={health.status === 'ok' ? 'ok' : 'warning'}
          value={health.status === 'ok' ? t('health.alive') : t('health.notReady')}
          hint={health.backend ? `${t('health.checkedAt')} ${new Date(health.backend.checkedAt).toLocaleTimeString('vi-VN')}` : undefined}
        />
        <StatCard
          label={t('dashboard.database.title')}
          tone={readiness === null ? 'danger' : 'ok'}
          value={readiness === null ? t('health.notReady') : t('health.ready')}
          hint={readiness === null ? (health.backend?.readinessDetail ?? undefined) : `${t('health.writerEpoch')} ${String(readiness.writer_epoch)}`}
        />
        <StatCard
          label={t('dashboard.session.title')}
          tone="default"
          value={demo ? String(demoData?.sessions.length ?? 0) : '—'}
          hint={demo ? t('demo.banner') : t('dashboard.session.none')}
        />
        <StatCard
          label={t('dashboard.data.title')}
          tone={demo ? 'warning' : 'default'}
          value={demo ? t('dashboard.data.mock') : t('dashboard.data.live')}
        />
      </div>

      <Card title={t('dashboard.capabilities.title')} subtitle={t('dashboard.capabilities.hint')}>
        <DataTable
          caption={t('dashboard.capabilities.title')}
          columns={capabilityColumns}
          rows={CAPABILITIES}
          rowKey={(row) => row.id}
        />
      </Card>

      <Card title={t('dashboard.approvals.title')} tone="warning">
        <p>{t('dashboard.approvals.body')}</p>
      </Card>
    </div>
  );
}
