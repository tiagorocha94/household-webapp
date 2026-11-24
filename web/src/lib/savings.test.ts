import { afterEach, describe, expect, it } from 'vitest';
import { setLocale } from './i18n/index.svelte';
import type { SavingsAccount } from '../api/client';
import { buildBalanceHistory, movementLabel, remainingCents, totals } from './savings';

function account(over: Partial<SavingsAccount>): SavingsAccount {
  return {
    id: 1,
    name: 'Emergency fund',
    owner: null,
    targetCents: null,
    notes: '',
    balanceCents: 0,
    depositedCents: 0,
    withdrawnCents: 0,
    interestCents: 0,
    progressPct: 0,
    movementCount: 0,
    snapshotCount: 0,
    lastSnapshotMonth: null,
    ...over,
  };
}

describe('totals', () => {
  it('adds the accounts and compares only those with a target', () => {
    const sum = totals([
      account({ balanceCents: 740000, targetCents: 1000000, interestCents: 40000 }),
      account({ id: 2, balanceCents: 315000, targetCents: 500000 }),
      account({ id: 3, balanceCents: 100000 }),
    ]);
    expect(sum.savedCents).toBe(1155000);
    expect(sum.targetCents).toBe(1500000);
    expect(sum.progressPct).toBe(70);
    expect(sum.interestCents).toBe(40000);
  });

  it('is zero without accounts or targets', () => {
    expect(totals([])).toEqual({
      savedCents: 0,
      targetCents: 0,
      progressPct: 0,
      interestCents: 0,
    });
    expect(totals([account({ balanceCents: 5000 })]).progressPct).toBe(0);
  });
});

describe('remainingCents', () => {
  it('is what is missing, never negative, and null without a target', () => {
    expect(remainingCents(account({ targetCents: 500000, balanceCents: 315000 }))).toBe(185000);
    expect(remainingCents(account({ targetCents: 1000, balanceCents: 5000 }))).toBe(0);
    expect(remainingCents(account({ balanceCents: 5000 }))).toBeNull();
  });
});

describe('buildBalanceHistory', () => {
  it('takes withdrawals off the money in', () => {
    const history = buildBalanceHistory(
      [
        { month: '2026-01', type: 'deposit', amountCents: 300000 },
        { month: '2026-02', type: 'withdrawal', amountCents: 50000 },
      ],
      [{ id: 1, month: '2026-03', valueCents: 255000 }],
    );
    expect(history.map((p) => p.month)).toEqual(['2026-01', '2026-02', '2026-03']);
    expect(history.map((p) => p.allocatedCents)).toEqual([300000, 250000, 250000]);
    expect(history[2].valueCents).toBe(255000);
  });

  it('is empty without data', () => {
    expect(buildBalanceHistory([], [])).toEqual([]);
  });
});

describe('movementLabel', () => {
  afterEach(() => setLocale('en'));

  it('follows the language', () => {
    expect(movementLabel('withdrawal')).toBe('Withdrawal');
    setLocale('pt');
    expect(movementLabel('withdrawal')).toBe('Levantamento');
    expect(movementLabel('deposit')).toBe('Depósito');
  });
});
