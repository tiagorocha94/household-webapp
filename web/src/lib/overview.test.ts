import { describe, expect, it } from 'vitest';
import type { Overview } from '../api/client';
import {
  describeAttention,
  expenseNote,
  isEmpty,
  nextInstalment,
  savedNote,
  savingsAccountNote,
  savingsNote,
} from './overview';

const nbsp = /\s/g;
const plain = (text: string) => text.replace(nbsp, ' ');

describe('describeAttention', () => {
  it('names the account, the month and the shortfall of a goals shortfall', () => {
    const line = describeAttention(
      { kind: 'goals_shortfall', name: 'Household', month: '2026-12', amountCents: 42000, pct: 0 },
      '2026-11',
    );
    expect(line.topic).toBe('Goals');
    expect(plain(line.text)).toBe(
      'The planned allocations of Household for Dec 2026 are 420,00 € more than the money available.',
    );
    expect(line.path).toBe('/goals');
  });

  it('links a budget group over its budget to the current month', () => {
    const line = describeAttention(
      { kind: 'budget_over', name: 'Fun', month: '', amountCents: 2500, pct: 114 },
      '2026-11',
    );
    expect(line.topic).toBe('Budget');
    expect(plain(line.text)).toBe('Fun is at 114% of its budget this month (25,00 € over).');
    expect(line.path).toBe('/cashflow/months/2026-11');
  });
});

describe('describeAttention for unassigned money', () => {
  it('points savings to the savings page and investments to the investments page', () => {
    const savings = describeAttention(
      { kind: 'unassigned_savings', name: '', month: '2026-11', amountCents: 10000, pct: 0 },
      '2026-11',
    );
    expect(savings.topic).toBe('Savings');
    expect(plain(savings.text)).toBe('100,00 € put aside in Nov 2026 is not in a savings deposit.');
    expect(savings.path).toBe('/savings');
    const investments = describeAttention(
      { kind: 'unassigned_investments', name: '', month: '2026-10', amountCents: 15000, pct: 0 },
      '2026-11',
    );
    expect(investments.topic).toBe('Investments');
    expect(investments.path).toBe('/investments');
  });
});

describe('describeAttention for credits expenses without a payment', () => {
  it('says money was paid on credits and links to credits', () => {
    const line = describeAttention(
      { kind: 'unassigned_credits', name: '', month: '2026-11', amountCents: 75000, pct: 0 },
      '2026-11',
    );
    expect(line.topic).toBe('Credits');
    expect(plain(line.text)).toBe(
      '750,00 € paid on credits in Nov 2026 is not in a credit payment.',
    );
    expect(line.path).toBe('/credits');
  });
});

describe('describeAttention for an unpaid credit', () => {
  it('names the credit and the month and links to credits', () => {
    const line = describeAttention(
      { kind: 'credit_unpaid', name: 'Car loan', month: '2026-11', amountCents: 0, pct: 0 },
      '2026-11',
    );
    expect(line.topic).toBe('Credits');
    expect(line.text).toBe('Car loan: no payment recorded for Nov 2026.');
    expect(line.path).toBe('/credits');
  });
});

describe('expenseNote', () => {
  it('says nothing without a previous month to compare with', () => {
    expect(expenseNote({ expenseCents: 5000, previousExpenseCents: 0 })).toBeNull();
  });

  it('says more, less or the same', () => {
    expect(plain(expenseNote({ expenseCents: 23100, previousExpenseCents: 21400 }) ?? '')).toBe(
      '17,00 € more than last month so far',
    );
    expect(plain(expenseNote({ expenseCents: 100, previousExpenseCents: 1100 }) ?? '')).toBe(
      '10,00 € less than last month so far',
    );
    expect(expenseNote({ expenseCents: 700, previousExpenseCents: 700 })).toBe(
      'The same as last month so far',
    );
  });
});

describe('savedNote', () => {
  it('shows the saved part of the expenses, or nothing', () => {
    expect(plain(savedNote({ savedCents: 25000 }) ?? '')).toBe('of which saved 250,00 €');
    expect(savedNote({ savedCents: 0 })).toBeNull();
  });
});

describe('nextInstalment', () => {
  it('shows the month, or a dash when there is no regular payment yet', () => {
    expect(nextInstalment({ nextInstalmentMonth: '2026-12' })).toBe('Dec 2026');
    expect(nextInstalment({ nextInstalmentMonth: null })).toBe('—');
  });
});

describe('isEmpty', () => {
  const empty: Overview = {
    month: '2026-11',
    attention: [],
    goals: { savedCents: 0, targetCents: 0 },
    credit: { outstandingCents: 0, monthlyInstalmentCents: 0, activeContracts: 0 },
    investments: { currentValueCents: 0 },
    savings: {
      balanceCents: 0,
      targetCents: 0,
      accounts: 0,
      interestCents: 0,
      items: [],
    },
    netWorth: { assetsCents: 0, cashCents: 0, cashByUser: [], creditCents: 0, netCents: 0 },
    cashflow: {
      month: '2026-11',
      incomeCents: 0,
      expenseCents: 0,
      savedCents: 0,
      netCents: 0,
      savingsRatePct: 0,
      previousExpenseCents: 0,
      groups: [],
    },
    activeGoals: [],
    contracts: [],
    holdings: [],
  };

  it('is true only when every section is empty', () => {
    expect(isEmpty(empty)).toBe(true);
    expect(isEmpty({ ...empty, holdings: [{ id: 1, name: 'Broker', valueCents: 100 }] })).toBe(
      false,
    );
    expect(isEmpty({ ...empty, cashflow: { ...empty.cashflow, expenseCents: 1 } })).toBe(false);
    expect(isEmpty({ ...empty, savings: { ...empty.savings, accounts: 1 } })).toBe(false);
  });
});

describe('savingsNote', () => {
  it('gives the progress towards the targets and the interest', () => {
    const note = savingsNote({ balanceCents: 1055000, targetCents: 1500000, interestCents: 55000 });
    expect(note).toContain('70%');
    expect(note).toContain('targeted');
    expect(note).toContain('interest');
  });

  it('is null without targets or interest', () => {
    expect(savingsNote({ balanceCents: 100, targetCents: 0, interestCents: 0 })).toBeNull();
  });
});

describe('savingsAccountNote', () => {
  it('names the owner, the progress and the interest', () => {
    const note = savingsAccountNote({
      owner: 'alex',
      targetCents: 1000000,
      progressPct: 74,
      interestCents: 40000,
    });
    expect(note).toMatch(/^alex · 74% of .*10.*000.* · .*400.* interest$/);
  });

  it('says household and no target when there are none', () => {
    expect(
      savingsAccountNote({ owner: null, targetCents: null, progressPct: 0, interestCents: 0 }),
    ).toBe('Household · no target set');
  });
});
