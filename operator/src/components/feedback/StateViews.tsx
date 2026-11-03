import type { ReactNode } from 'react';
import { ApiError, isApiError } from '@/api/errors';
import { useT } from '@/app/i18n/LocaleProvider';
import type { MessageKey } from '@/app/i18n/messages';

export function LoadingState({ label }: { label?: string }) {
  const t = useT();
  return (
    <div className="state state--loading" role="status" aria-live="polite">
      <span className="spinner" aria-hidden="true" />
      <span>{label ?? t('state.loading')}</span>
    </div>
  );
}

export function EmptyState({
  title,
  body,
  action,
}: {
  title: string;
  body?: string;
  action?: ReactNode;
}) {
  return (
    <div className="state state--empty" role="status">
      <h3 className="state__title">{title}</h3>
      {body ? <p className="state__body">{body}</p> : null}
      {action ? <div className="state__action">{action}</div> : null}
    </div>
  );
}

export function errorMessageKey(error: ApiError | null): MessageKey {
  if (error === null) {
    return 'error.unknown';
  }
  switch (error.kind) {
    case 'network':
      return 'health.error.network';
    case 'timeout':
      return 'health.error.timeout';
    case 'protocol':
      return 'health.error.protocol';
    case 'http':
      return 'health.error.http';
    default:
      return 'error.unknown';
  }
}

export function ErrorState({
  title,
  error,
  onRetry,
  retryLabel,
}: {
  title?: string;
  error: unknown;
  onRetry?: () => void;
  retryLabel?: string;
}) {
  const t = useT();
  const apiError = isApiError(error) ? error : null;
  const message = apiError === null ? t('error.unknown') : t(errorMessageKey(apiError));

  return (
    <div className="state state--error" role="alert">
      <h3 className="state__title">{title ?? t('state.error')}</h3>
      <p className="state__body">{message}</p>
      {apiError?.requestId ? (
        <p className="state__meta">
          {t('health.requestId')}: <code>{apiError.requestId}</code>
        </p>
      ) : null}
      {onRetry ? (
        <div className="state__action">
          <button type="button" className="btn btn--secondary" onClick={onRetry}>
            {retryLabel ?? t('action.retry')}
          </button>
        </div>
      ) : null}
    </div>
  );
}

export function UnavailableState({
  title,
  reason,
  owner,
  action,
}: {
  title: string;
  reason: string;
  owner: string;
  action?: ReactNode;
}) {
  const t = useT();
  return (
    <div className="state state--unavailable" role="status">
      <h3 className="state__title">
        {title} <span className="badge badge--neutral">{t('state.unavailable')}</span>
      </h3>
      <p className="state__body">{reason}</p>
      <p className="state__meta">
        {t('dashboard.capabilities.owner')}: <strong>{owner}</strong>
      </p>
      {action ? <div className="state__action">{action}</div> : null}
    </div>
  );
}

export { ApiError };
