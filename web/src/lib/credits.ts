// How contracts are grouped and labelled. Pure, so it is tested without a browser.
import type { Contract, Payment } from '../api/client';
import { t, type MessageKey } from './i18n/index.svelte';

/** The text for a contract category, in the current language. */
export function contractCategoryLabel(category: Contract['category']): string {
  return t(`goals.credits.category.${category}` as MessageKey);
}

/** The text for a payment type, in the current language. */
export function paymentLabel(type: Payment['type']): string {
  return t(`goals.credits.payment.${type}` as MessageKey);
}

/** Splits contracts into active and paid off; the biggest debt first, then by name. */
export function groupContracts(contracts: readonly Contract[]): {
  active: Contract[];
  paidOff: Contract[];
} {
  const byDebt = (a: Contract, b: Contract) =>
    b.outstandingPrincipalCents - a.outstandingPrincipalCents || a.name.localeCompare(b.name);
  return {
    active: contracts.filter((c) => c.status === 'active').sort(byDebt),
    paidOff: contracts.filter((c) => c.status === 'paid_off').sort(byDebt),
  };
}

/** Share of what was paid that went to interest, 0 to 100, or 0 when nothing was paid. */
export function interestSharePct(contract: Pick<Contract, 'interestPaidCents' | 'totalPaidCents'>) {
  if (contract.totalPaidCents <= 0) return 0;
  return (contract.interestPaidCents * 100) / contract.totalPaidCents;
}

export interface CreditTotals {
  /** What is still owed on the active contracts. */
  outstandingCents: number;
  borrowedCents: number;
  paidCents: number;
  interestPaidCents: number;
  activeCount: number;
}

/** The totals over every contract; outstanding only counts the active ones. */
export function totals(contracts: readonly Contract[]): CreditTotals {
  const sum: CreditTotals = {
    outstandingCents: 0,
    borrowedCents: 0,
    paidCents: 0,
    interestPaidCents: 0,
    activeCount: 0,
  };
  for (const c of contracts) {
    sum.borrowedCents += c.originalPrincipalCents;
    sum.paidCents += c.totalPaidCents;
    sum.interestPaidCents += c.interestPaidCents;
    if (c.status === 'active') {
      sum.activeCount++;
      sum.outstandingCents += c.outstandingPrincipalCents;
    }
  }
  return sum;
}
