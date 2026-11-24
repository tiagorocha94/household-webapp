// Which feature modules the server runs, and which pages belong to which.
import type { NavItem } from './nav';

export const MODULES = ['cashflow', 'goals', 'savings', 'credits', 'investments'] as const;
export type ModuleName = (typeof MODULES)[number];

/**
 * The module a page belongs to, by the name of its route, or null for pages that are always
 * there (Overview, Net worth, Notifications, Settings, Login).
 */
export function moduleOfRoute(name: string | undefined): ModuleName | null {
  switch (name) {
    case 'cashflow':
    case 'cashflow-month':
    case 'cashflow-month-view':
      return 'cashflow';
    case 'accounts':
    case 'account':
    case 'goal':
      return 'goals';
    case 'savings':
    case 'saving':
      return 'savings';
    case 'credits':
    case 'contract':
      return 'credits';
    case 'investments':
    case 'entity':
    case 'instrument':
      return 'investments';
    default:
      return null;
  }
}

/**
 * Whether a module runs. `enabled` is what the server said, or null while that is not known yet,
 * and then nothing is hidden.
 */
export function isEnabled(enabled: readonly string[] | null, module: ModuleName | null): boolean {
  return module === null || enabled === null || enabled.includes(module);
}

/** The menu items of the modules that run. */
export function visibleNav(
  items: readonly NavItem[],
  enabled: readonly string[] | null,
): NavItem[] {
  return items.filter((item) => isEnabled(enabled, item.module ?? null));
}
