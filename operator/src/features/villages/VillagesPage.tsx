import { useState, type FormEvent } from 'react';
import { useT } from '@/app/i18n/LocaleProvider';
import { useApiConfig } from '@/app/providers/ApiProvider';
import { capability } from '@/api/capabilities';
import { Button } from '@/components/common/Button';
import { Card } from '@/components/common/Card';
import { DataTable, type Column } from '@/components/common/DataTable';
import { TextField } from '@/components/common/TextField';
import { EmptyState, UnavailableState } from '@/components/feedback/StateViews';
import type { CreateVillageSource, VillageSummary } from '@/contracts/village';

const MAX_DISPLAY_NAME = 64;

interface FormErrors {
  displayName?: string;
  source?: string;
}

/**
 * The form mirrors the documented POST /admin/v1/villages contract exactly:
 * `display_name` (1..64) and `source` (`mock` | `replay` | `authorized_live`).
 * LIVE creation is not offered because GATE-LIVE is blocked, and the object
 * built here is the contract type that will be sent once M07 exists, so no
 * invented field (region, wall HP, …) can leak into the production payload.
 */
export function VillagesPage() {
  const t = useT();
  const { demo, demoData } = useApiConfig();
  const createCapability = capability('villages.create');
  const listCapability = capability('villages.list');

  const [created, setCreated] = useState<readonly VillageSummary[]>([]);
  const [displayName, setDisplayName] = useState('');
  const [source, setSource] = useState<CreateVillageSource>('mock');
  const [errors, setErrors] = useState<FormErrors>({});
  const [simulated, setSimulated] = useState(false);

  const rows: readonly VillageSummary[] = demo ? [...(demoData?.villages ?? []), ...created] : [];

  function validate(): FormErrors {
    const next: FormErrors = {};
    const trimmed = displayName.trim();
    if (trimmed === '') {
      next.displayName = t('villages.form.nameRequired');
    } else if (trimmed.length > MAX_DISPLAY_NAME) {
      next.displayName = t('villages.form.nameTooLong');
    }
    if (source !== 'mock' && source !== 'replay') {
      next.source = t('villages.form.sourceRequired');
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

    // Demo-only local simulation, shaped exactly like the documented contract.
    const id = `demo-village-${String(created.length + 3).padStart(2, '0')}`;
    setCreated((current) => [
      ...current,
      {
        id,
        display_name: displayName.trim(),
        lifecycle: 'waiting',
        source_status: source === 'replay' ? 'replay' : 'mock',
        wave_no: 1,
        created_at: new Date(0).toISOString(),
      },
    ]);
    setSimulated(true);
    setDisplayName('');
  }

  const columns: Column<VillageSummary>[] = [
    { key: 'name', header: t('villages.list.name'), render: (row) => row.display_name },
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
            value={displayName}
            onChange={setDisplayName}
            error={errors.displayName}
            required
            disabled={!demo}
            placeholder="Làng Ven Sông"
          />

          <div className="field">
            <label className="field__label" htmlFor="village-source">
              {t('villages.form.source')}
            </label>
            <select
              id="village-source"
              className="field__input"
              value={source}
              disabled={!demo}
              onChange={(event) => {
                setSource(event.target.value === 'replay' ? 'replay' : 'mock');
              }}
              aria-invalid={errors.source ? true : undefined}
            >
              <option value="mock">mock</option>
              <option value="replay">replay</option>
              <option value="authorized_live" disabled>
                authorized_live ({t('state.blocked')})
              </option>
            </select>
            <p className="field__hint">{t('villages.form.sourceHint')}</p>
            {errors.source ? (
              <p className="field__error" role="alert">
                {errors.source}
              </p>
            ) : null}
          </div>

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
            <DataTable caption={t('villages.title')} columns={columns} rows={rows} rowKey={(row) => row.id} />
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
