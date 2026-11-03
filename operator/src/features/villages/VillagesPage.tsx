import { useMemo, useState, type FormEvent } from 'react';
import { useT } from '@/app/i18n/LocaleProvider';
import { useApiConfig } from '@/app/providers/ApiProvider';
import { capability } from '@/api/capabilities';
import { Button } from '@/components/common/Button';
import { Card } from '@/components/common/Card';
import { DataTable, type Column } from '@/components/common/DataTable';
import { TextField } from '@/components/common/TextField';
import { EmptyState, UnavailableState } from '@/components/feedback/StateViews';
import type { VillageSummary } from '@/contracts/village';

const MAX_NAME = 40;
const MIN_WALL = 100;
const MAX_WALL = 100_000;

interface FormErrors {
  name?: string;
  maxWallHp?: string;
}

export function VillagesPage() {
  const t = useT();
  const { demo, demoData } = useApiConfig();
  const createCapability = capability('villages.create');
  const listCapability = capability('villages.list');

  const [created, setCreated] = useState<readonly VillageSummary[]>([]);
  const [name, setName] = useState('');
  const [region, setRegion] = useState('vn-south');
  const [maxWallHp, setMaxWallHp] = useState('1000');
  const [errors, setErrors] = useState<FormErrors>({});
  const [simulated, setSimulated] = useState(false);

  const rows = useMemo<readonly VillageSummary[]>(
    () => (demo ? [...(demoData?.villages ?? []), ...created] : []),
    [demo, demoData, created],
  );

  function validate(): FormErrors {
    const next: FormErrors = {};
    const trimmed = name.trim();
    if (trimmed === '') {
      next.name = t('villages.form.nameRequired');
    } else if (trimmed.length > MAX_NAME) {
      next.name = t('villages.form.nameTooLong');
    }
    const wall = Number.parseInt(maxWallHp, 10);
    if (!Number.isFinite(wall) || wall < MIN_WALL || wall > MAX_WALL) {
      next.maxWallHp = t('villages.form.maxWallHpRange');
    }
    return next;
  }

  function onSubmit(event: FormEvent<HTMLFormElement>): void {
    event.preventDefault();
    const nextErrors = validate();
    setErrors(nextErrors);
    if (Object.keys(nextErrors).length > 0 || !demo) {
      return;
    }

    // Demo-only local simulation. Nothing is sent to the backend and the row is
    // labelled as simulated.
    const id = `demo-village-${String(created.length + 3).padStart(2, '0')}`;
    setCreated((current) => [
      ...current,
      {
        id,
        name: name.trim(),
        lifecycle: 'waiting',
        source_status: 'mock',
        wave_no: 0,
        created_at: new Date(0).toISOString(),
      },
    ]);
    setSimulated(true);
    setName('');
  }

  const columns: Column<VillageSummary>[] = [
    { key: 'name', header: t('villages.list.name'), render: (row) => row.name },
    { key: 'lifecycle', header: t('villages.list.lifecycle'), render: (row) => row.lifecycle },
    { key: 'source', header: t('villages.list.source'), render: (row) => row.source_status },
    { key: 'wave', header: t('villages.list.wave'), render: (row) => String(row.wave_no) },
  ];

  return (
    <div className="page">
      <header className="page__header">
        <h2 className="page__title">{t('villages.title')}</h2>
        <p className="page__subtitle">{t('villages.subtitle')}</p>
      </header>

      <Card title={t('villages.form.title')}>
        <form className="form" onSubmit={onSubmit} noValidate>
          <TextField
            label={t('villages.form.name')}
            value={name}
            onChange={setName}
            error={errors.name}
            required
            disabled={!demo}
            placeholder="Làng Ven Sông"
          />
          <TextField
            label={t('villages.form.region')}
            value={region}
            onChange={setRegion}
            disabled={!demo}
          />
          <TextField
            label={t('villages.form.maxWallHp')}
            value={maxWallHp}
            onChange={setMaxWallHp}
            inputMode="numeric"
            error={errors.maxWallHp}
            disabled={!demo}
          />
          <div className="form__actions">
            <Button type="submit" variant="primary" disabled={!demo}>
              {t('villages.form.submit')}
            </Button>
            {demo ? (
              <span className="form__note">{t('demo.action.simulated')}</span>
            ) : (
              <span className="form__note">{t('villages.form.disabled')}</span>
            )}
          </div>
        </form>
        {simulated ? (
          <p className="notice notice--demo" role="status">
            {t('demo.action.simulated')}
          </p>
        ) : null}
      </Card>

      <Card title={t('villages.list.name')}>
        {demo ? (
          rows.length === 0 ? (
            <EmptyState title={t('villages.empty.title')} body={t('villages.empty.body')} />
          ) : (
            <DataTable
              caption={t('villages.title')}
              columns={columns}
              rows={rows}
              rowKey={(row) => row.id}
            />
          )
        ) : (
          <UnavailableState
            title={t('villages.title')}
            reason={listCapability.reason}
            owner={listCapability.owner}
          />
        )}
      </Card>

      <Card title={t('dashboard.capabilities.title')}>
        <p className="muted">
          <code>villages.create</code> — {createCapability.state} ({createCapability.owner})
        </p>
      </Card>
    </div>
  );
}
