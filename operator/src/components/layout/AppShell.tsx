import { useState } from 'react';
import { Outlet } from 'react-router-dom';
import { useT } from '@/app/i18n/LocaleProvider';
import { useApiConfig } from '@/app/providers/ApiProvider';
import { Sidebar } from '@/components/layout/Sidebar';
import { TopBar } from '@/components/layout/TopBar';
import { DemoBanner } from '@/components/feedback/DemoBanner';
import { useMediaQuery } from '@/hooks/useMediaQuery';

export function AppShell() {
  const t = useT();
  const { demo } = useApiConfig();
  const narrow = useMediaQuery('(max-width: 1023px)');
  const [manualCollapsed, setManualCollapsed] = useState<boolean | null>(null);
  const collapsed = manualCollapsed ?? narrow;

  return (
    <div className={`shell${collapsed ? ' shell--collapsed' : ''}`}>
      <a className="skip-link" href="#main-content">
        {t('app.skipToContent')}
      </a>

      <Sidebar
        collapsed={collapsed}
        onToggle={() => {
          setManualCollapsed(!collapsed);
        }}
      />

      <div className="shell__content">
        <TopBar />
        {demo ? <DemoBanner /> : null}
        <main className="main" id="main-content" tabIndex={-1}>
          <Outlet />
        </main>
      </div>
    </div>
  );
}
