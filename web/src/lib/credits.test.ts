import { afterEach, describe, expect, it } from 'vitest';
import { setLocale } from './i18n/index.svelte';
import type { Contract } from '../api/client';
import {
  contractCategoryLabel,
  groupContracts,
  interestSharePct,
  paymentLabel,
  totals,
} from './credits';

function contract(over: Partial<Contract>): Contract {
  return {
    id: 1,
    name: 'Home',
    category: 'mortgage',
    contractNumber: '1',
    lender: 'Bank',
    originalPrincipalCents: 100000,
    startMonth: '2020-01',
    expectedEndMonth: null,
    status: 'active',
    notes: '',
    paymentCount: 0,
    principalPaidCents: 0,
    interestPaidCents: 0,
    totalPaidCents: 0,
    outstandingPrincipalCents: 100000,
    progressPct: 0,
    lastPaymentMonth: null,
    expectedDay: null,
    ...over,
  };
}

describe('groupContracts', () => {
  it('splits by status and puts the biggest debt first', () => {
    const { active, paidOff } = groupContracts([
      contract({ id: 1, name: 'Car', outstandingPrincipalCents: 500 }),
      contract({ id: 2, name: 'Old', status: 'paid_off', outstandingPrincipalCents: 0 }),
      contract({ id: 3, name: 'Home', outstandingPrincipalCents: 9000 }),
    ]);
    expect(active.map((c) => c.id)).toEqual([3, 1]);
    expect(paidOff.map((c) => c.id)).toEqual([2]);
  });
});

describe('interestSharePct', () => {
  it('is the interest part of what was paid', () => {
    expect(interestSharePct({ interestPaidCents: 25, totalPaidCents: 100 })).toBe(25);
  });
  it('is 0 when nothing was paid', () => {
    expect(interestSharePct({ interestPaidCents: 0, totalPaidCents: 0 })).toBe(0);
  });
});

describe('totals', () => {
  it('counts the outstanding debt of active contracts only', () => {
    const sum = totals([
      contract({
        id: 1,
        originalPrincipalCents: 100000,
        totalPaidCents: 20000,
        interestPaidCents: 3000,
        outstandingPrincipalCents: 83000,
      }),
      contract({
        id: 2,
        status: 'paid_off',
        originalPrincipalCents: 50000,
        totalPaidCents: 52000,
        interestPaidCents: 2000,
        outstandingPrincipalCents: 0,
      }),
    ]);
    expect(sum).toEqual({
      outstandingCents: 83000,
      borrowedCents: 150000,
      paidCents: 72000,
      interestPaidCents: 5000,
      activeCount: 1,
    });
  });
});

describe('labels', () => {
  afterEach(() => setLocale('en'));

  it('are English by default', () => {
    expect(contractCategoryLabel('mortgage')).toBe('Mortgage');
    expect(paymentLabel('extraordinary')).toBe('Extraordinary');
  });

  it('follow the language', () => {
    setLocale('pt');
    expect(contractCategoryLabel('mortgage')).toBe('Habitação');
    expect(contractCategoryLabel('car')).toBe('Automóvel');
    expect(paymentLabel('extraordinary')).toBe('Extraordinário');
  });
});
