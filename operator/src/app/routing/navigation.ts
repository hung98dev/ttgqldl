import type { MessageKey } from '@/app/i18n/messages';

export interface NavItem {
  readonly to: string;
  readonly labelKey: MessageKey;
  readonly icon: string;
  readonly end?: boolean;
}

/** The six operator sections. Order is the workflow order, not alphabetical. */
export const NAV_ITEMS: readonly NavItem[] = [
  { to: '/', labelKey: 'nav.dashboard', icon: '◎', end: true },
  { to: '/villages', labelKey: 'nav.villages', icon: '⌂' },
  { to: '/sessions', labelKey: 'nav.sessions', icon: '▤' },
  { to: '/monitoring', labelKey: 'nav.monitoring', icon: '◈' },
  { to: '/moderation', labelKey: 'nav.moderation', icon: '⚖' },
  { to: '/settings', labelKey: 'nav.settings', icon: '⚙' },
] as const;
