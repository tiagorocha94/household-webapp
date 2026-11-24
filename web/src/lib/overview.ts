// What the Home page says about the figures the server composes. Pure, so it is
// tested without a browser.
import type {
  Overview,
  OverviewAttention,
  OverviewCashflow,
  OverviewContract,
  OverviewSavings,
  OverviewSavingsAccount,
} from '../api/client';
import { groupLabel, t, type MessageKey } from './i18n/index.svelte';
import { formatCents, formatPercent } from './money';
import { formatMonth } from './month';

export interface AttentionLine {
  /** What the line is about: "Goals" or "Budget". */
  topic: string;
  text: string;
  /** Where the problem is fixed, as a router path. */
  path: string;
  linkLabel: string;
}

/** What each kind of unassigned money is about, and where to add it. */
const UNASSIGNED: Record<
  string,
  { topic: MessageKey; text: MessageKey; link: MessageKey; path: string }
> = {
  unassigned_savings: {
    topic: 'shell.nav.savings',
    text: 'shell.attention.unassignedSavings',
    link: 'shell.attention.openSavings',
    path: '/savings',
  },
  unassigned_investments: {
    topic: 'shell.nav.investments',
    text: 'shell.attention.unassignedInvestments',
    link: 'shell.attention.openInvestments',
    path: '/investments',
  },
  unassigned_goals: {
    topic: 'shell.nav.goals',
    text: 'shell.attention.unassignedGoals',
    link: 'shell.attention.openGoals',
    path: '/goals',
  },
  unassigned_credits: {
    topic: 'shell.nav.credits',
    text: 'shell.attention.unassignedCredits',
    link: 'shell.attention.openCredits',
    path: '/credits',
  },
};

/** The sentence and the link for one thing that needs attention. */
export function describeAttention(item: OverviewAttention, current: string): AttentionLine {
  if (item.kind === 'goals_shortfall') {
    return {
      topic: t('shell.nav.goals'),
      text: t('shell.attention.goalsShortfall', {
        name: item.name,
        month: formatMonth(item.month),
        amount: formatCents(item.amountCents),
      }),
      path: '/goals',
      linkLabel: t('shell.attention.openGoals'),
    };
  }
  if (item.kind === 'credit_unpaid') {
    return {
      topic: t('shell.nav.credits'),
      text: t('shell.attention.creditUnpaid', { name: item.name, month: formatMonth(item.month) }),
      path: '/credits',
      linkLabel: t('shell.attention.openCredits'),
    };
  }
  const unassigned = UNASSIGNED[item.kind];
  if (unassigned) {
    return {
      topic: t(unassigned.topic),
      text: t(unassigned.text, {
        amount: formatCents(item.amountCents),
        month: formatMonth(item.month),
      }),
      path: unassigned.path,
      linkLabel: t(unassigned.link),
    };
  }
  return {
    topic: t('shell.common.budget'),
    text: t('shell.attention.budgetOver', {
      name: groupLabel(item.name),
      pct: formatPercent(item.pct),
      amount: formatCents(item.amountCents),
    }),
    path: `/cashflow/months/${current}`,
    linkLabel: t('shell.attention.openCashflow'),
  };
}

/** How this month's expenses compare with the previous month's up to the same day, or null with nothing to compare. */
export function expenseNote(
  cashflow: Pick<OverviewCashflow, 'expenseCents' | 'previousExpenseCents'>,
): string | null {
  if (cashflow.previousExpenseCents === 0) return null;
  const difference = cashflow.expenseCents - cashflow.previousExpenseCents;
  if (difference === 0) return t('shell.overview.expenseSame');
  return t(difference > 0 ? 'shell.overview.expenseMore' : 'shell.overview.expenseLess', {
    amount: formatCents(Math.abs(difference)),
  });
}

/** The part of this month's expenses that was put aside, or null when none was. */
export function savedNote(cashflow: Pick<OverviewCashflow, 'savedCents'>): string | null {
  return cashflow.savedCents > 0
    ? t('shell.overview.savedNote', { amount: formatCents(cashflow.savedCents) })
    : null;
}

/** The note under the Savings figure: the progress towards the targets, then the interest. */
export function savingsNote(
  savings: Pick<OverviewSavings, 'balanceCents' | 'targetCents' | 'interestCents'>,
): string | null {
  const parts: string[] = [];
  if (savings.targetCents > 0) {
    const pct = Math.round((savings.balanceCents * 100) / savings.targetCents);
    parts.push(
      t('shell.overview.targetProgress', {
        pct: formatPercent(pct),
        amount: formatCents(savings.targetCents),
      }),
    );
  }
  if (savings.interestCents > 0)
    parts.push(t('shell.overview.interest', { amount: formatCents(savings.interestCents) }));
  return parts.length > 0 ? parts.join(' · ') : null;
}

/** The line under a savings account: who, progress towards its target, interest. */
export function savingsAccountNote(
  a: Pick<OverviewSavingsAccount, 'owner' | 'targetCents' | 'progressPct' | 'interestCents'>,
): string {
  const parts = [a.owner ?? t('shell.common.householdOwner')];
  parts.push(
    a.targetCents === null
      ? t('shell.overview.noTarget')
      : t('shell.overview.accountProgress', {
          pct: a.progressPct,
          amount: formatCents(a.targetCents),
        }),
  );
  if (a.interestCents > 0) {
    parts.push(t('shell.overview.interest', { amount: formatCents(a.interestCents) }));
  }
  return parts.join(' · ');
}

/** The month of a contract's next instalment, or a dash when it cannot be estimated. */
export function nextInstalment(contract: Pick<OverviewContract, 'nextInstalmentMonth'>): string {
  return contract.nextInstalmentMonth ? formatMonth(contract.nextInstalmentMonth) : '—';
}

/** Whether there is nothing at all to show yet (a module that does not run has nothing), so the page explains where data comes from. */
export function isEmpty(o: Overview): boolean {
  return (
    o.activeGoals.length === 0 &&
    o.contracts.length === 0 &&
    o.holdings.length === 0 &&
    (o.savings?.accounts ?? 0) === 0 &&
    (o.cashflow?.incomeCents ?? 0) === 0 &&
    (o.cashflow?.expenseCents ?? 0) === 0
  );
}
