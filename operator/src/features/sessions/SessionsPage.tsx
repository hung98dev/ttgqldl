import { useMemo, useState } from 'react';
import { useT } from '@/app/i18n/LocaleProvider';
import { useApiConfig } from '@/app/providers/ApiProvider';
import { capability } from '@/api/capabilities';
import { Button } from '@/components/common/Button';
import { Card } from '@/components/common/Card';
import { ConfirmDialog } from '@/components/common/Dialog';
import { DataTable, type Column } from '@/components/common/DataTable';
import { EmptyState, UnavailableState } from '@/components/feedback/StateViews';
import { hasPermission, useAuth } from '@/features/authentication/useAuth';
import type { SessionAction, SessionState, SessionSummary } from '@/contracts/session';
import type { MessageKey } from '@/app/i18n/messages';

const STATE_LABEL: Record<SessionState, MessageKey> = {
  idle: 'sessions.state.idle',
  running: 'sessions.state.running',
  paused: 'sessions.state.paused',
  stopped: 'sessions.state.stopped',
};

export function SessionsPage() {
  const t = useT();
  const { demo, demoData } = useApiConfig();
  const { session: authSession } = useAuth();
  const controlCapability = capability('sessions.control');
  const listCapability = capability('sessions.list');

  const canControl = hasPermission(authSession, 'sessions.control');
  const [overrides, setOverrides] = useState<Readonly<Record<string, SessionState>>>({});
  const [pendingStop, setPendingStop] = useState<SessionSummary | null>(null);
  const [notice, setNotice] = useState<string | null>(null);

  const rows = useMemo<readonly SessionSummary[]>(
    () => (demo ? (demoData?.sessions ?? []) : []),
    [demo, demoData],
  );

  const stateOf = (row: SessionSummary): SessionState => overrides[row.id] ?? row.state;
  const controlsEnabled = demo && canControl;

  function applyAction(row: SessionSummary, action: SessionAction): void {
    if (!controlsEnabled) {
      return;
    }
    if (action === 'stop') {
      setPendingStop(row);
      return;
    }
    const next: SessionState =
      action === 'start' ? 'running' : action === 'pause' ? 'paused' : 'running';
    setOverrides((current) => ({ ...current, [row.id]: next }));
    setNotice(t('demo.action.simulated'));
  }

  const columns: Column<SessionSummary>[] = [
    { key: 'id', header: t('sessions.list.id'), render: (row) => <code>{row.id}</code> },
    {
      key: 'state',
      header: t('sessions.list.state'),
      render: (row) => t(STATE_LABEL[stateOf(row)]),
    },
    { key: 'wave', header: t('sessions.list.wave'), render: (row) => String(row.wave_no) },
    { key: 'source', header: t('sessions.list.source'), render: (row) => row.source_status },
    { key: 'seq', header: t('sessions.list.lastSeq'), render: (row) => row.last_event_seq },
    {
      key: 'actions',
      header: t('action.start'),
      render: (row) => (
        <div className="row-actions">
          <Button
            variant="secondary"
            disabled={!controlsEnabled || stateOf(row) === 'running'}
            onClick={() => {
              applyAction(row, 'start');
            }}
            aria-label={`${t('action.start')} ${row.id}`}
          >
            {t('action.start')}
          </Button>
          <Button
            variant="secondary"
            disabled={!controlsEnabled || stateOf(row) !== 'running'}
            onClick={() => {
              applyAction(row, 'pause');
            }}
            aria-label={`${t('action.pause')} ${row.id}`}
          >
            {t('action.pause')}
          </Button>
          <Button
            variant="secondary"
            disabled={!controlsEnabled || stateOf(row) !== 'paused'}
            onClick={() => {
              applyAction(row, 'resume');
            }}
            aria-label={`${t('action.resume')} ${row.id}`}
          >
            {t('action.resume')}
          </Button>
          <Button
            variant="danger"
            disabled={!controlsEnabled || stateOf(row) === 'stopped'}
            onClick={() => {
              applyAction(row, 'stop');
            }}
            aria-label={`${t('action.stop')} ${row.id}`}
          >
            {t('action.stop')}
          </Button>
        </div>
      ),
    },
  ];

  return (
    <div className="page">
      <header className="page__header">
        <h2 className="page__title">{t('sessions.title')}</h2>
        <p className="page__subtitle">{t('sessions.subtitle')}</p>
      </header>

      {!controlsEnabled ? (
        <p className="notice notice--warning" role="note">
          {t('sessions.controls.disabled')} ({controlCapability.owner})
        </p>
      ) : null}

      {notice ? (
        <p className="notice notice--demo" role="status">
          {notice}
        </p>
      ) : null}

      <Card title={t('sessions.title')}>
        {demo ? (
          rows.length === 0 ? (
            <EmptyState title={t('sessions.empty.title')} body={t('sessions.empty.body')} />
          ) : (
            <DataTable
              caption={t('sessions.title')}
              columns={columns}
              rows={rows}
              rowKey={(row) => row.id}
            />
          )
        ) : (
          <UnavailableState
            title={t('sessions.title')}
            reason={listCapability.reason}
            owner={listCapability.owner}
          />
        )}
      </Card>

      <ConfirmDialog
        open={pendingStop !== null}
        title={t('sessions.confirm.stop.title')}
        description={t('sessions.confirm.stop.body')}
        confirmLabel={t('action.stop')}
        onCancel={() => {
          setPendingStop(null);
        }}
        onConfirm={() => {
          if (pendingStop !== null) {
            setOverrides((current) => ({ ...current, [pendingStop.id]: 'stopped' }));
            setNotice(t('demo.action.simulated'));
          }
          setPendingStop(null);
        }}
      />
    </div>
  );
}
