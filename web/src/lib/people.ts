// How a person's income splits into what they spent and what they put aside,
// and how that went over the last six months. Pure, so it is tested without a
// browser. Put aside is the money in the savings, investments and goals
// categories; credit payments count as spent (the server does the splitting).
import type { CashflowUserMonth, CashflowUserTotal } from '../api/client';
import { monthsBetween, previousMonth } from './month';

export interface PersonSplit {
  spentCents: number;
  putAsideCents: number;
  /** Income less everything spent and put aside; negative when more went out than came in. */
  leftOverCents: number;
  /** The widths of the three bar segments as percentages of the bar, never negative. */
  spentWidth: number;
  putAsideWidth: number;
  leftOverWidth: number;
  /** The share of income put aside, as a whole percent; null without income. */
  putAsideShare: number | null;
  /** The share of income spent, as a whole percent; null without income. */
  spentShare: number | null;
}

/** Splits one person's month into spent, put aside and left over. */
export function personSplit(u: CashflowUserTotal): PersonSplit {
  const leftOverCents = u.incomeCents - u.spentCents - u.putAsideCents;
  // The bar is the income, or what went out when that was more, so it never overflows.
  const whole = Math.max(u.incomeCents, u.spentCents + u.putAsideCents, 1);
  const width = (cents: number) => (Math.max(cents, 0) / whole) * 100;
  const share = (cents: number) =>
    u.incomeCents > 0 ? Math.round((cents * 100) / u.incomeCents) : null;
  return {
    spentCents: u.spentCents,
    putAsideCents: u.putAsideCents,
    leftOverCents,
    spentWidth: width(u.spentCents),
    putAsideWidth: width(u.putAsideCents),
    leftOverWidth: width(leftOverCents),
    putAsideShare: share(u.putAsideCents),
    spentShare: share(u.spentCents),
  };
}

/** What one person (or the household) put aside in a month, and its share of income. */
export interface PutAsideCell {
  putAsideCents: number;
  /** A whole percent of income; null without income. */
  sharePct: number | null;
}

export interface PutAsideHistory {
  /** The six months ending at the month, oldest first. */
  months: string[];
  /** One row per person, ordered by name; a month without movements is null. */
  people: { username: string; cells: (PutAsideCell | null)[] }[];
  /** The household: everyone's income and money put aside added up. */
  household: (PutAsideCell | null)[];
}

function cell(incomeCents: number, putAsideCents: number): PutAsideCell {
  return {
    putAsideCents,
    sharePct: incomeCents > 0 ? Math.round((putAsideCents * 100) / incomeCents) : null,
  };
}

/** Lays the server's people history out as a table of the six months ending at `month`. */
export function putAsideHistory(
  history: readonly CashflowUserMonth[],
  month: string,
): PutAsideHistory {
  let first = month;
  for (let i = 0; i < 5; i++) first = previousMonth(first);
  const months = monthsBetween(first, month);

  const usernames = [...new Set(history.map((h) => h.username))].sort((a, b) => a.localeCompare(b));
  const people = usernames.map((username) => ({
    username,
    cells: months.map((m) => {
      const row = history.find((h) => h.username === username && h.month === m);
      return row ? cell(row.incomeCents, row.putAsideCents) : null;
    }),
  }));
  const household = months.map((m) => {
    const rows = history.filter((h) => h.month === m);
    if (rows.length === 0) return null;
    return cell(
      rows.reduce((sum, r) => sum + r.incomeCents, 0),
      rows.reduce((sum, r) => sum + r.putAsideCents, 0),
    );
  });
  return { months, people, household };
}
