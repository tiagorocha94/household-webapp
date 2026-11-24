// How cashflow is labelled, totalled and compared. Pure, so it is tested
// without a browser.
import type {
  CashflowCategory,
  CashflowCategoryTotal,
  CashflowGroupTotal,
  CashflowMonth,
  CashflowMovement,
} from '../api/client';
import { t } from './i18n/index.svelte';
import { formatCents } from './money';
import { previousMonth } from './month';

export type Direction = CashflowCategory['direction'];

const rate = new Intl.NumberFormat('pt-PT', {
  style: 'percent',
  minimumFractionDigits: 2,
  maximumFractionDigits: 2,
});

const points = new Intl.NumberFormat('pt-PT', {
  minimumFractionDigits: 2,
  maximumFractionDigits: 2,
  signDisplay: 'exceptZero',
});

/** Formats the API's savingsRatePct, a percentage scaled by 100: 2500 becomes "25,00%". */
/** The short label for a category role, such as "Saved" for savings. */
export function roleLabel(role: string): string {
  switch (role) {
    case 'savings':
    case 'investments':
    case 'goals':
    case 'credits':
      return t(`cashflow.role.${role}`);
    default:
      return '';
  }
}

export function formatSavingsRatePct(savingsRatePct: number): string {
  // "|| 0" turns a negative zero into zero, so it never shows as "-0,00%".
  return rate.format(savingsRatePct / 10000 || 0);
}

/** Formats a change in the savings rate in scaled points: 250 becomes "+2,50 p.p.". */
export function formatRateChange(deltaPct: number): string {
  return `${points.format(deltaPct / 100)} p.p.`;
}

/** Formats a change in cents with its sign: 12000 becomes "+120,00 €", -500 "−5,00 €". */
export function formatChange(deltaCents: number): string {
  if (deltaCents === 0) return formatCents(0);
  return `${deltaCents > 0 ? '+' : '−'}${formatCents(Math.abs(deltaCents))}`;
}

/** Whether what is left over is positive, negative, or neither, for colouring. */
export function netTone(netCents: number): 'in' | 'out' | undefined {
  if (netCents > 0) return 'in';
  if (netCents < 0) return 'out';
  return undefined;
}

/** A movement's amount with the direction applied: income is positive, expenses negative. */
export function signedAmount(
  movement: Pick<CashflowMovement, 'direction' | 'amountCents'>,
): number {
  return movement.direction === 'income' ? movement.amountCents : -movement.amountCents;
}

/** The months newest first, whatever order they arrive in. */
export function newestFirst(months: readonly CashflowMonth[]): CashflowMonth[] {
  return [...months].sort((a, b) => b.month.localeCompare(a.month));
}

export interface MonthChange {
  /** The calendar month before, which the change is measured against. */
  previous: string;
  incomeCents: number;
  expenseCents: number;
  netCents: number;
  /** In scaled points like the API: 250 means the rate rose by 2.50 points. */
  savingsRatePct: number;
}

/**
 * How a month differs from the calendar month before it, or null when that
 * month has no movements and so nothing to compare with. Computed here, not by
 * the API.
 */
export function monthChange(months: readonly CashflowMonth[], month: string): MonthChange | null {
  const current = months.find((m) => m.month === month);
  const previous = months.find((m) => m.month === previousMonth(month));
  if (!current || !previous) return null;
  return {
    previous: previous.month,
    incomeCents: current.incomeCents - previous.incomeCents,
    expenseCents: current.expenseCents - previous.expenseCents,
    netCents: current.netCents - previous.netCents,
    savingsRatePct: current.savingsRatePct - previous.savingsRatePct,
  };
}

export interface PeriodTotals {
  incomeCents: number;
  expenseCents: number;
  netCents: number;
  /** Scaled by 100 like the API; derived from the sums, 0 without income. */
  savingsRatePct: number;
  movementCount: number;
}

/** Adds up months; the savings rate is derived from the sums, not averaged. */
export function periodTotals(months: readonly CashflowMonth[]): PeriodTotals {
  const incomeCents = months.reduce((sum, m) => sum + m.incomeCents, 0);
  const expenseCents = months.reduce((sum, m) => sum + m.expenseCents, 0);
  const netCents = incomeCents - expenseCents;
  return {
    incomeCents,
    expenseCents,
    netCents,
    savingsRatePct: incomeCents > 0 ? Math.trunc((netCents * 10000) / incomeCents) || 0 : 0,
    movementCount: months.reduce((sum, m) => sum + m.movementCount, 0),
  };
}

/** The totals of the months that fall in a calendar year such as "2026". */
export function yearTotals(months: readonly CashflowMonth[], year: string): PeriodTotals {
  return periodTotals(months.filter((m) => m.month.startsWith(`${year}-`)));
}

/** The calendar years that have months, newest first. */
export function yearsOf(months: readonly CashflowMonth[]): string[] {
  return [...new Set(months.map((m) => m.month.slice(0, 4)))].sort().reverse();
}

/** The latest months, oldest first, for the chart. */
export function chartMonths(months: readonly CashflowMonth[], limit = 12): CashflowMonth[] {
  return newestFirst(months).slice(0, limit).reverse();
}

/** Each category's share of its direction's total, 0 to 100, largest first. */
export function categoryShares(
  totals: readonly CashflowCategoryTotal[],
  direction: Direction,
): { category: CashflowCategoryTotal; sharePct: number }[] {
  const same = totals.filter((t) => t.direction === direction);
  const sum = same.reduce((acc, t) => acc + t.totalCents, 0);
  return same
    .map((category) => ({ category, sharePct: sum > 0 ? (category.totalCents * 100) / sum : 0 }))
    .sort(
      (a, b) =>
        b.category.totalCents - a.category.totalCents ||
        a.category.name.localeCompare(b.category.name),
    );
}

/** How a category is drawn: a Lucide icon name on a `#RRGGBB` colour. */
export interface CategoryLook {
  icon: string;
  color: string;
}

/** Each category's look by name, from the category list. */
export function categoryLooks(categories: readonly CashflowCategory[]): Map<string, CategoryLook> {
  return new Map(categories.map((c) => [c.name, { icon: c.icon, color: c.color }]));
}

export interface GroupUse {
  /** How much of the budget was spent, 0 or more (above 100 when over). */
  usedPct: number;
  /** Whether more was spent than the budget. */
  over: boolean;
  /** The money left, or the amount over the budget, never negative. */
  remainingCents: number;
}

/** How much of a group's budget a month used. Without a budget nothing is compared. */
export function groupUse(
  group: Pick<CashflowGroupTotal, 'budgetCents' | 'spentCents' | 'leftCents'>,
): GroupUse {
  return {
    usedPct: group.budgetCents > 0 ? (group.spentCents * 100) / group.budgetCents : 0,
    over: group.leftCents < 0,
    remainingCents: Math.abs(group.leftCents),
  };
}

/** The pages of one month: a summary and a page for each long list. */
export type MonthView = 'summary' | 'budget' | 'categories' | 'people' | 'movements';

const monthViews: readonly MonthView[] = ['summary', 'budget', 'categories', 'people', 'movements'];

/** True when `value` names a page of a month. */
export function isMonthView(value: string): value is MonthView {
  return (monthViews as readonly string[]).includes(value);
}

/** The columns a month's movements can be sorted by. */
export type MovementColumn = 'date' | 'user' | 'category' | 'description' | 'amount';

/** The value each column sorts by. */
export const movementSortValues: Record<
  MovementColumn,
  (movement: CashflowMovement) => string | number
> = {
  date: (m) => m.date,
  user: (m) => m.username,
  category: (m) => m.category,
  description: (m) => m.description,
  amount: (m) => signedAmount(m),
};
