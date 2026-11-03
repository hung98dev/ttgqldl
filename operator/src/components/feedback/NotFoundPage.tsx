import { useT } from '@/app/i18n/LocaleProvider';
import { EmptyState } from '@/components/feedback/StateViews';

export function NotFoundPage() {
  const t = useT();
  return (
    <EmptyState
      title="404"
      body={t('error.unknown')}
      action={
        <a className="btn btn--secondary" href="/">
          {t('nav.dashboard')}
        </a>
      }
    />
  );
}
