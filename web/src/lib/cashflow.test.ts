import { describe, expect, it } from 'vitest';
import type { CashflowCategoryTotal, CashflowMonth } from '../api/client';
import {
  roleLabel,
  categoryShares,
  chartMonths,
  formatChange,
  formatRateChange,
  formatSavingsRatePct,
  groupUse,
  movementSortValues,
  monthChange,
  netTone,
  newestFirst,
  periodTotals,
  signedAmount,
  yearsOf,
  yearTotals,
} from './cashflow';

const compact = (text: string) => text.replace(/\s/g, '');

function month(m: string, income: number, expense: number, count = 1): CashflowMonth {
  const net = income - expense;
  return {
    month: m,
    incomeCents: income,
    expenseCents: expense,
    netCents: net,
    savingsRatePct: income > 0 ? Math.trunc((net * 10000) / income) : 0,
    movementCount: count,
  };
}

describe('formatSavingsRatePct', () => {
  it('divides the scaled percentage by 100 and shows only a minus sign', () => {
    expect(compact(formatSavingsRatePct(2500))).toBe('25,00%');
    expect(compact(formatSavingsRatePct(8143))).toBe('81,43%');
    expect(compact(formatSavingsRatePct(-8000))).toBe('-80,00%');
    expect(compact(formatSavingsRatePct(0))).toBe('0,00%');
    expect(compact(formatSavingsRatePct(-0))).toBe('0,00%');
  });
});

describe('formatRateChange', () => {
  it('shows the change in points with its sign', () => {
    expect(compact(formatRateChange(250))).toBe('+2,50p.p.');
    expect(compact(formatRateChange(-1000))).toBe('-10,00p.p.');
    expect(compact(formatRateChange(0))).toBe('0,00p.p.');
  });
});

describe('formatChange', () => {
  it('shows an amount with its sign', () => {
    expect(compact(formatChange(12000))).toBe('+120,00€');
    expect(compact(formatChange(-500))).toBe('−5,00€');
    expect(compact(formatChange(0))).toBe('0,00€');
  });
});

describe('netTone', () => {
  it('colours what is left over', () => {
    expect(netTone(1)).toBe('in');
    expect(netTone(-1)).toBe('out');
    expect(netTone(0)).toBeUndefined();
  });
});

describe('signedAmount', () => {
  it('is positive for income and negative for expenses', () => {
    expect(signedAmount({ direction: 'income', amountCents: 500 })).toBe(500);
    expect(signedAmount({ direction: 'expense', amountCents: 500 })).toBe(-500);
  });
});

describe('newestFirst and chartMonths', () => {
  const months = [month('2026-01', 1, 1), month('2026-03', 1, 1), month('2026-02', 1, 1)];

  it('orders months newest first without changing the input', () => {
    expect(newestFirst(months).map((m) => m.month)).toEqual(['2026-03', '2026-02', '2026-01']);
    expect(months.map((m) => m.month)).toEqual(['2026-01', '2026-03', '2026-02']);
  });

  it('gives the latest months oldest first for the chart', () => {
    expect(chartMonths(months).map((m) => m.month)).toEqual(['2026-01', '2026-02', '2026-03']);
    expect(chartMonths(months, 2).map((m) => m.month)).toEqual(['2026-02', '2026-03']);
    expect(chartMonths([])).toEqual([]);
  });
});

describe('monthChange', () => {
  const months = [
    month('2026-03', 300000, 100000),
    month('2026-02', 250000, 150000),
    month('2025-12', 100000, 50000),
    month('2026-01', 100000, 50000),
  ];

  it('measures a month against the calendar month before it', () => {
    const change = monthChange(months, '2026-03');
    expect(change).toEqual({
      previous: '2026-02',
      incomeCents: 50000,
      expenseCents: -50000,
      netCents: 100000,
      // 6666 (66.66%, truncated like the API) against 4000 (40.00%).
      savingsRatePct: 2666,
    });
  });

  it('crosses a year boundary', () => {
    expect(monthChange(months, '2026-01')?.previous).toBe('2025-12');
  });

  it('is null when the month before has no movements or the month is unknown', () => {
    expect(monthChange(months, '2025-12')).toBeNull();
    expect(monthChange(months, '2026-07')).toBeNull();
  });
});

describe('periodTotals and yearTotals', () => {
  const months = [
    month('2026-01', 300000, 100000, 4),
    month('2026-02', 100000, 150000, 3),
    month('2025-12', 200000, 100000, 2),
  ];

  it('adds up months and derives the rate from the sums', () => {
    // Income 600000, expenses 350000, net 250000: 41.66%, not the average of the months.
    expect(periodTotals(months)).toEqual({
      incomeCents: 600000,
      expenseCents: 350000,
      netCents: 250000,
      savingsRatePct: 4166,
      movementCount: 9,
    });
  });

  it('is zero for no months, and the rate is 0 without income', () => {
    expect(periodTotals([])).toEqual({
      incomeCents: 0,
      expenseCents: 0,
      netCents: 0,
      savingsRatePct: 0,
      movementCount: 0,
    });
    expect(periodTotals([month('2026-05', 0, 3000)]).savingsRatePct).toBe(0);
    // A loss too small to reach -0.01% is 0, never negative zero.
    expect(
      Object.is(periodTotals([month('2026-05', 2_000_000, 2_000_001)]).savingsRatePct, 0),
    ).toBe(true);
  });

  it('totals only the months of one year', () => {
    expect(yearTotals(months, '2026')).toMatchObject({
      incomeCents: 400000,
      expenseCents: 250000,
      netCents: 150000,
      savingsRatePct: 3750,
    });
    expect(yearTotals(months, '2025').incomeCents).toBe(200000);
    expect(yearTotals(months, '2024').incomeCents).toBe(0);
  });

  it('lists the years that have months, newest first', () => {
    expect(yearsOf(months)).toEqual(['2026', '2025']);
    expect(yearsOf([])).toEqual([]);
  });
});

describe('categoryShares', () => {
  const totals: CashflowCategoryTotal[] = [
    {
      categoryId: 1,
      name: 'rent',
      direction: 'expense',
      totalCents: 60000,
      movementCount: 1,
      role: '',
    },
    {
      categoryId: 2,
      name: 'food',
      direction: 'expense',
      totalCents: 30000,
      movementCount: 5,
      role: '',
    },
    {
      categoryId: 3,
      name: 'fun',
      direction: 'expense',
      totalCents: 10000,
      movementCount: 2,
      role: '',
    },
    {
      categoryId: 4,
      name: 'salary',
      direction: 'income',
      totalCents: 200000,
      movementCount: 1,
      role: '',
    },
  ];

  it('shares an expense total between its categories, largest first', () => {
    const shares = categoryShares(totals, 'expense');
    expect(shares.map((s) => s.category.name)).toEqual(['rent', 'food', 'fun']);
    expect(shares.map((s) => s.sharePct)).toEqual([60, 30, 10]);
  });

  it('keeps income apart from expenses', () => {
    const shares = categoryShares(totals, 'income');
    expect(shares).toHaveLength(1);
    expect(shares[0].sharePct).toBe(100);
  });

  it('breaks ties by name and is empty without categories', () => {
    const tied: CashflowCategoryTotal[] = [
      {
        categoryId: 1,
        name: 'b',
        direction: 'expense',
        totalCents: 100,
        movementCount: 1,
        role: '',
      },
      {
        categoryId: 2,
        name: 'a',
        direction: 'expense',
        totalCents: 100,
        movementCount: 1,
        role: '',
      },
    ];
    expect(categoryShares(tied, 'expense').map((s) => s.category.name)).toEqual(['a', 'b']);
    expect(categoryShares([], 'expense')).toEqual([]);
  });
});

describe('groupUse', () => {
  it('shows the share of the budget spent and what is left', () => {
    expect(groupUse({ budgetCents: 200000, spentCents: 50000, leftCents: 150000 })).toEqual({
      usedPct: 25,
      over: false,
      remainingCents: 150000,
    });
  });

  it('goes above 100 and reports the excess when over budget', () => {
    expect(groupUse({ budgetCents: 34500, spentCents: 41275, leftCents: -6775 })).toEqual({
      usedPct: (41275 * 100) / 34500,
      over: true,
      remainingCents: 6775,
    });
  });

  it('is exactly used up with nothing left, not over', () => {
    const use = groupUse({ budgetCents: 100, spentCents: 100, leftCents: 0 });
    expect(use).toEqual({ usedPct: 100, over: false, remainingCents: 0 });
  });

  it('has no percentage without a budget', () => {
    expect(groupUse({ budgetCents: 0, spentCents: 5000, leftCents: -5000 }).usedPct).toBe(0);
  });
});

describe('roleLabel', () => {
  it('names each role and nothing for none', () => {
    expect(roleLabel('savings')).toBe('Saved');
    expect(roleLabel('investments')).toBe('Invested');
    expect(roleLabel('goals')).toBe('Goals');
    expect(roleLabel('credits')).toBe('Credits');
    expect(roleLabel('')).toBe('');
  });
});

describe('movementSortValues', () => {
  const movement = {
    id: 1,
    date: '2026-10-03',
    username: 'ana',
    category: 'Groceries',
    description: 'Market',
    direction: 'expense' as const,
    amountCents: 1250,
  };

  it('sorts each column by what it shows', () => {
    expect(movementSortValues.date(movement as never)).toBe('2026-10-03');
    expect(movementSortValues.user(movement as never)).toBe('ana');
    expect(movementSortValues.category(movement as never)).toBe('Groceries');
    expect(movementSortValues.description(movement as never)).toBe('Market');
  });

  it('sorts the amount with the direction applied', () => {
    expect(movementSortValues.amount(movement as never)).toBe(-1250);
    expect(movementSortValues.amount({ ...movement, direction: 'income' } as never)).toBe(1250);
  });
});
