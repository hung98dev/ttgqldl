import { useT } from '@/app/i18n/LocaleProvider';

/** Persistent, unmissable marker for demonstration data. */
export function DemoBanner() {
  const t = useT();
  return (
    <div className="demo-banner" role="note" aria-label={t('demo.banner')}>
      <strong className="demo-banner__label">{t('demo.banner')}</strong>
      <span className="demo-banner__explain">{t('demo.explain')}</span>
    </div>
  );
}
