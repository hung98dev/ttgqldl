import type { ReactNode } from 'react';

export function Card({
  title,
  subtitle,
  actions,
  children,
  tone = 'default',
}: {
  title?: string;
  subtitle?: string;
  actions?: ReactNode;
  children?: ReactNode;
  tone?: 'default' | 'warning' | 'danger';
}) {
  return (
    <section className={`card card--${tone}`} aria-label={title}>
      {title || actions ? (
        <header className="card__header">
          <div>
            {title ? <h2 className="card__title">{title}</h2> : null}
            {subtitle ? <p className="card__subtitle">{subtitle}</p> : null}
          </div>
          {actions ? <div className="card__actions">{actions}</div> : null}
        </header>
      ) : null}
      <div className="card__body">{children}</div>
    </section>
  );
}

export function StatCard({
  label,
  value,
  hint,
  tone = 'default',
}: {
  label: string;
  value: ReactNode;
  hint?: string;
  tone?: 'default' | 'ok' | 'warning' | 'danger';
}) {
  return (
    <div className={`stat stat--${tone}`}>
      <span className="stat__label">{label}</span>
      <span className="stat__value">{value}</span>
      {hint ? <span className="stat__hint">{hint}</span> : null}
    </div>
  );
}
