// The pages in the side menu, and the small rules around it.
import { t, type MessageKey } from './i18n/index.svelte';

export interface NavItem {
  /** The English name, which identifies the item; what is shown comes from `key` (see navLabel). */
  label: string;
  /** The text shown for the item. */
  key: MessageKey;
  path: string;
  /** A Lucide icon name registered in icons.ts. */
  icon: string;
  /** The first segments of the addresses that belong to this item; empty means only "/". */
  sections: string[];
}

export const NAV: NavItem[] = [
  { label: 'Overview', key: 'shell.nav.overview', path: '/', icon: 'house', sections: [] },
  {
    label: 'Cashflow',
    key: 'shell.nav.cashflow',
    path: '/cashflow',
    icon: 'arrow-left-right',
    sections: ['cashflow'],
  },
  {
    label: 'Net worth',
    key: 'shell.nav.netWorth',
    path: '/net-worth',
    icon: 'chart-line',
    sections: ['net-worth'],
  },
  {
    label: 'Goals',
    key: 'shell.nav.goals',
    path: '/goals',
    icon: 'target',
    sections: ['goals', 'accounts'],
  },
  {
    label: 'Savings',
    key: 'shell.nav.savings',
    path: '/savings',
    icon: 'vault',
    sections: ['savings'],
  },
  {
    label: 'Credits',
    key: 'shell.nav.credits',
    path: '/credits',
    icon: 'landmark',
    sections: ['credits'],
  },
  {
    label: 'Investments',
    key: 'shell.nav.investments',
    path: '/investments',
    icon: 'trending-up',
    sections: ['investments'],
  },
];

/** The text of a menu item in the current language. */
export function navLabel(item: NavItem): string {
  return t(item.key);
}

/** True when the address at `path` belongs to `item`, so the item is marked as current. */
export function isCurrent(item: NavItem, path: string): boolean {
  if (item.sections.length === 0) return path === '/';
  return item.sections.includes(path.split('/')[1]);
}

const COLLAPSED_KEY = 'household.menu.collapsed';

/** Whether the menu was left collapsed in this browser; false when it cannot be read. */
export function loadCollapsed(
  storage: Pick<Storage, 'getItem'> | undefined = safeStorage(),
): boolean {
  try {
    return storage?.getItem(COLLAPSED_KEY) === '1';
  } catch {
    return false;
  }
}

/** Remembers whether the menu is collapsed; does nothing when storage is unavailable. */
export function saveCollapsed(
  collapsed: boolean,
  storage: Pick<Storage, 'setItem'> | undefined = safeStorage(),
): void {
  try {
    storage?.setItem(COLLAPSED_KEY, collapsed ? '1' : '0');
  } catch {
    // The choice is only a convenience.
  }
}

function safeStorage(): Storage | undefined {
  try {
    return window.localStorage;
  } catch {
    return undefined;
  }
}
