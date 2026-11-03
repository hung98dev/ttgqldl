import { NavLink } from 'react-router-dom';
import { useT } from '@/app/i18n/LocaleProvider';
import { NAV_ITEMS } from '@/app/routing/navigation';

export function Sidebar({
  collapsed,
  onToggle,
}: {
  collapsed: boolean;
  onToggle: () => void;
}) {
  const t = useT();

  return (
    <aside className="sidebar" data-collapsed={collapsed ? 'true' : 'false'}>
      <div className="sidebar__brand">
        <span className="sidebar__mark" aria-hidden="true">
          ⛨
        </span>
        {collapsed ? null : <span className="sidebar__name">TTGQLDL</span>}
      </div>

      <nav className="sidebar__nav" aria-label={t('nav.label')}>
        <ul className="sidebar__list">
          {NAV_ITEMS.map((item) => (
            <li key={item.to}>
              <NavLink
                to={item.to}
                end={item.end ?? false}
                className={({ isActive }) => `nav-link${isActive ? ' nav-link--active' : ''}`}
                title={collapsed ? t(item.labelKey) : undefined}
              >
                <span className="nav-link__icon" aria-hidden="true">
                  {item.icon}
                </span>
                <span className={collapsed ? 'sr-only' : 'nav-link__label'}>{t(item.labelKey)}</span>
              </NavLink>
            </li>
          ))}
        </ul>
      </nav>

      <button
        type="button"
        className="sidebar__toggle"
        onClick={onToggle}
        aria-expanded={!collapsed}
        aria-label={collapsed ? t('nav.expand') : t('nav.collapse')}
      >
        <span aria-hidden="true">{collapsed ? '»' : '«'}</span>
      </button>
    </aside>
  );
}
