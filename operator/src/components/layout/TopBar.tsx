import { useT } from '@/app/i18n/LocaleProvider';
import { useApiConfig } from '@/app/providers/ApiProvider';
import { useTheme } from '@/app/providers/ThemeProvider';
import { HealthIndicator } from '@/components/feedback/HealthIndicator';
import { StatusBadge } from '@/components/common/StatusBadge';
import { useHealthView } from '@/app/providers/HealthProvider';

export function TopBar() {
  const t = useT();
  const health = useHealthView();
  const { demo } = useApiConfig();
  const { theme, toggle } = useTheme();

  return (
    <header className="topbar">
      <div className="topbar__left">
        <h1 className="topbar__title">{t('app.title')}</h1>
        <span className="topbar__subtitle">{t('app.subtitle')}</span>
      </div>

      <div className="topbar__right">
        <StatusBadge tone={demo ? 'demo' : 'ok'}>{demo ? t('topbar.mode.demo') : t('topbar.mode.live')}</StatusBadge>
        <HealthIndicator health={health} />
        <button
          type="button"
          className="btn btn--ghost"
          onClick={toggle}
          aria-label={theme === 'dark' ? t('topbar.theme.toLight') : t('topbar.theme.toDark')}
        >
          <span aria-hidden="true">{theme === 'dark' ? '☀' : '☾'}</span>
          <span>{theme === 'dark' ? t('topbar.theme.light') : t('topbar.theme.dark')}</span>
        </button>
      </div>
    </header>
  );
}
