// The net worth history: which months to show and how the figure changed. Pure,
// so it is tested without a browser.
import type { NetWorthMonth } from '../api/client';
import { t } from './i18n/index.svelte';

/** How much history the chart shows: a number of months, or everything. */
export type Range = 6 | 12 | 'all';

export const ranges: readonly Range[] = [6, 12, 'all'];

export function rangeLabel(range: Range): string {
  return range === 'all'
    ? t('invest.networth.rangeAll')
    : t('invest.networth.rangeMonths', { count: range });
}

/** The last `range` months and the one before them, so the line has a starting point. */
export function inRange(months: readonly NetWorthMonth[], range: number | 'all'): NetWorthMonth[] {
  return range === 'all' ? [...months] : months.slice(-(range + 1));
}

export interface NetWorthChange {
  /** The month the change is measured from. */
  since: string;
  netCents: number;
  creditCents: number;
}

/**
 * The change in net worth and in credit owed over the last `back` months with
 * data; null when the history is too short to look that far back.
 */
export function changeOver(months: readonly NetWorthMonth[], back = 3): NetWorthChange | null {
  if (months.length <= back) return null;
  const last = months[months.length - 1];
  const from = months[months.length - 1 - back];
  return {
    since: from.month,
    netCents: last.netCents - from.netCents,
    creditCents: last.creditCents - from.creditCents,
  };
}
