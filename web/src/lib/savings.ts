// How savings accounts are labelled, totalled and charted. Pure, so it is
// tested without a browser.
import type { SavingsAccount, SavingsMovement, SavingsSnapshot } from '../api/client';
import { t, type MessageKey } from './i18n/index.svelte';
import { buildHistory, type HistoryPoint } from './investments';

/** The text for a savings movement type, in the current language. */
export function movementLabel(type: SavingsMovement['type']): string {
  return t(`goals.savings.movement.${type}` as MessageKey);
}

export interface SavingsTotals {
  savedCents: number;
  /** The targets of the accounts that have one. */
  targetCents: number;
  /** Share of those targets reached, as a whole percent; 0 without any target. */
  progressPct: number;
  interestCents: number;
}

/**
 * The totals over every account. Progress only compares the accounts that have
 * a target, so an account without one does not lower it.
 */
export function totals(accounts: readonly SavingsAccount[]): SavingsTotals {
  const sum: SavingsTotals = {
    savedCents: 0,
    targetCents: 0,
    progressPct: 0,
    interestCents: 0,
  };
  let towardsTarget = 0;
  for (const a of accounts) {
    sum.savedCents += a.balanceCents;
    sum.interestCents += a.interestCents;
    if (a.targetCents !== null) {
      sum.targetCents += a.targetCents;
      towardsTarget += Math.max(a.balanceCents, 0);
    }
  }
  if (sum.targetCents > 0) sum.progressPct = Math.floor((towardsTarget * 100) / sum.targetCents);
  return sum;
}

/** What is still missing to reach the target, or null without one. Never negative. */
export function remainingCents(account: Pick<SavingsAccount, 'targetCents' | 'balanceCents'>) {
  if (account.targetCents === null) return null;
  return Math.max(account.targetCents - account.balanceCents, 0);
}

/**
 * The balance by month for the history chart. Deposits less withdrawals take
 * the place of "money in", so the chart shows what interest added on top.
 */
export function buildBalanceHistory(
  movements: readonly Pick<SavingsMovement, 'month' | 'type' | 'amountCents'>[],
  snapshots: readonly Pick<SavingsSnapshot, 'id' | 'month' | 'valueCents'>[],
): HistoryPoint[] {
  return buildHistory(
    movements.map((m) => ({
      month: m.month,
      amountCents: m.type === 'withdrawal' ? -m.amountCents : m.amountCents,
    })),
    snapshots,
  );
}
