import { useMemo, useState } from 'react';
import { useT } from '@/app/i18n/LocaleProvider';
import { useApiConfig } from '@/app/providers/ApiProvider';
import { capability } from '@/api/capabilities';
import { Button } from '@/components/common/Button';
import { Card } from '@/components/common/Card';
import { DataTable, type Column } from '@/components/common/DataTable';
import { StatusBadge } from '@/components/common/StatusBadge';
import { TextField } from '@/components/common/TextField';
import { EmptyState, UnavailableState } from '@/components/feedback/StateViews';
import type { ModerationTarget, PrivacyEraseStatus } from '@/contracts/moderation';
import type { MessageKey } from '@/app/i18n/messages';

const ERASE_LABEL: Record<PrivacyEraseStatus['state'], MessageKey> = {
  queued: 'moderation.erase.state.queued',
  applying: 'moderation.erase.state.applying',
  applied: 'moderation.erase.state.applied',
  blocked: 'moderation.erase.state.blocked',
  failed: 'moderation.erase.state.failed',
};

export function ModerationPage() {
  const t = useT();
  const { demo, demoData } = useApiConfig();
  const searchCapability = capability('moderation.search');
  const mutateCapability = capability('moderation.mutate');
  const eraseCapability = capability('privacy.erase');

  const [query, setQuery] = useState('');

  const targets = useMemo<readonly ModerationTarget[]>(() => {
    if (!demo) {
      return [];
    }
    const all = demoData?.moderation ?? [];
    const needle = query.trim().toLowerCase();
    if (needle === '') {
      return all;
    }
    return all.filter(
      (target) =>
        target.display_alias.toLowerCase().includes(needle) ||
        target.public_ref.toLowerCase().includes(needle),
    );
  }, [demo, demoData, query]);

  const targetColumns: Column<ModerationTarget>[] = [
    {
      key: 'alias',
      // Rendered as text through React: an alias containing markup must never
      // become an element.
      header: t('moderation.alias'),
      render: (row) => <span>{row.display_alias}</span>,
    },
    { key: 'role', header: t('moderation.role'), render: (row) => row.role },
    {
      key: 'state',
      header: t('moderation.state'),
      render: (row) => (
        <StatusBadge tone={row.blocked ? 'danger' : row.hidden ? 'warning' : 'ok'}>
          {row.blocked
            ? t('moderation.state.blocked')
            : row.hidden
              ? t('moderation.state.hidden')
              : t('moderation.state.active')}
        </StatusBadge>
      ),
    },
    {
      key: 'actions',
      header: t('action.hide'),
      render: () => (
        <div className="row-actions">
          <Button disabled title={mutateCapability.reason}>
            {t('action.hide')}
          </Button>
          <Button disabled title={mutateCapability.reason}>
            {t('action.block')}
          </Button>
        </div>
      ),
    },
  ];

  const eraseColumns: Column<PrivacyEraseStatus>[] = [
    { key: 'id', header: 'request_id', render: (row) => <code>{row.request_id}</code> },
    {
      key: 'state',
      header: t('moderation.state'),
      render: (row) => (
        <StatusBadge tone={row.state === 'applied' ? 'ok' : row.state === 'blocked' ? 'danger' : 'warning'}>
          {t(ERASE_LABEL[row.state])}
        </StatusBadge>
      ),
    },
    {
      key: 'requested',
      header: t('moderation.erase.title'),
      render: (row) => new Date(row.requested_at).toLocaleString('vi-VN'),
    },
    { key: 'detail', header: t('moderation.state'), render: (row) => row.detail ?? '—' },
  ];

  return (
    <div className="page">
      <header className="page__header">
        <h2 className="page__title">{t('moderation.title')}</h2>
        <p className="page__subtitle">{t('moderation.subtitle')}</p>
      </header>

      <Card title={t('moderation.search.label')}>
        <TextField
          label={t('moderation.search.label')}
          value={query}
          onChange={setQuery}
          placeholder={t('moderation.search.placeholder')}
          disabled={!demo}
          hint={demo ? undefined : searchCapability.reason}
        />
        {demo ? (
          targets.length === 0 ? (
            <EmptyState title={t('moderation.search.empty')} body={t('moderation.search.disabled')} />
          ) : (
            <DataTable
              caption={t('moderation.title')}
              columns={targetColumns}
              rows={targets}
              rowKey={(row) => row.public_ref}
            />
          )
        ) : (
          <UnavailableState
            title={t('moderation.title')}
            reason={searchCapability.reason}
            owner={searchCapability.owner}
          />
        )}
      </Card>

      <Card title={t('moderation.erase.title')}>
        {demo ? (
          <DataTable
            caption={t('moderation.erase.title')}
            columns={eraseColumns}
            rows={demoData?.eraseRequests ?? []}
            rowKey={(row) => row.request_id}
            empty={<EmptyState title={t('moderation.erase.empty')} />}
          />
        ) : (
          <UnavailableState
            title={t('moderation.erase.title')}
            reason={eraseCapability.reason}
            owner={eraseCapability.owner}
          />
        )}
      </Card>
    </div>
  );
}
