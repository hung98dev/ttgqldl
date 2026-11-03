import { useT } from '@/app/i18n/LocaleProvider';
import { EmptyState } from '@/components/feedback/StateViews';

export function AccessDeniedPage() {
  const t = useT();
  return (
    <div className="page page--narrow">
      <EmptyState
        title={t('auth.accessDenied.title')}
        body={`${t('auth.accessDenied.body')} ${t('auth.role.unknown')}.`}
      />
    </div>
  );
}
