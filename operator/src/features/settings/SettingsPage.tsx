import { useT } from '@/app/i18n/LocaleProvider';
import { useApiConfig } from '@/app/providers/ApiProvider';
import { useAuth } from '@/features/authentication/useAuth';
import { ROLE_PERMISSIONS, type Permission } from '@/features/authentication/auth';
import { CAPABILITIES } from '@/api/capabilities';
import { Card } from '@/components/common/Card';
import { DataTable, type Column } from '@/components/common/DataTable';
import { StatusBadge } from '@/components/common/StatusBadge';
import type { Capability, Role } from '@/contracts/api';

const ROLES: readonly Role[] = ['viewer', 'operator', 'admin'];

export function SettingsPage() {
  const t = useT();
  const { demo, apiBaseUrl } = useApiConfig();
  const { session } = useAuth();

  const capabilityColumns: Column<Capability>[] = [
    { key: 'id', header: 'capability', render: (row) => <code>{row.id}</code> },
    { key: 'state', header: 'state', render: (row) => <StatusBadge>{row.state}</StatusBadge> },
    { key: 'owner', header: t('dashboard.capabilities.owner'), render: (row) => row.owner },
  ];

  return (
    <div className="page">
      <header className="page__header">
        <h2 className="page__title">{t('settings.title')}</h2>
        <p className="page__subtitle">{t('settings.subtitle')}</p>
      </header>

      <Card title={t('settings.mode')}>
        <dl className="definition">
          <dt>{t('settings.mode')}</dt>
          <dd>{demo ? t('settings.mode.demo') : t('settings.mode.production')}</dd>
          <dt>{t('settings.apiBase')}</dt>
          <dd>
            <code>{apiBaseUrl === '' ? '/ (same-origin proxy)' : apiBaseUrl}</code>
          </dd>
          <dt>{t('settings.language')}</dt>
          <dd>Tiếng Việt (vi)</dd>
          <dt>{t('settings.version')}</dt>
          <dd>0.1.0</dd>
          <dt>{t('topbar.mode.live')}</dt>
          <dd>{session.kind === 'unavailable' ? session.reason : session.kind}</dd>
        </dl>
      </Card>

      <Card title={t('settings.roles')}>
        <DataTable
          caption={t('settings.roles')}
          columns={[
            { key: 'role', header: t('settings.roles'), render: (row: Role) => t(`settings.role.${row}`) },
            {
              key: 'permissions',
              header: t('dashboard.capabilities.title'),
              render: (row: Role) => (
                <span className="permissions">{ROLE_PERMISSIONS[row].join(', ')}</span>
              ),
            },
          ]}
          rows={ROLES}
          rowKey={(row) => row}
        />
      </Card>

      <Card title={t('settings.capabilities')}>
        <DataTable
          caption={t('settings.capabilities')}
          columns={capabilityColumns}
          rows={CAPABILITIES}
          rowKey={(row) => row.id}
        />
      </Card>
    </div>
  );
}

export type { Permission };
