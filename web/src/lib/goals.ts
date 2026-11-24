// What a goal is today, and how a movement reads from the account's or the
// goal's point of view. Pure, so it is tested without a browser.
import type { Goal, Movement } from '../api/client';
import { t, type MessageKey } from './i18n/index.svelte';
import { monthsBetween } from './month';

export type Phase = 'active' | 'planned' | 'completed' | 'cancelled';
export type PhaseFilter = Phase | 'all';

const phases: readonly Phase[] = ['active', 'planned', 'completed', 'cancelled'];

/** The text for a goal state or the "all" filter, in the current language. */
export function phaseLabel(phase: PhaseFilter): string {
  return t(`goals.phase.${phase}` as MessageKey);
}

/** An active goal whose start month has not come yet is planned. */
export function goalPhase(goal: Pick<Goal, 'status' | 'startMonth'>, current: string): Phase {
  if (goal.status === 'active' && goal.startMonth > current) return 'planned';
  return goal.status as Phase;
}

/** Goals in reading order: active first, then planned, completed, cancelled; soonest end first. */
export function sortGoals(goals: readonly Goal[], current: string): Goal[] {
  const rank = (g: Goal) => phases.indexOf(goalPhase(g, current));
  return [...goals].sort(
    (a, b) =>
      rank(a) - rank(b) || a.endMonth.localeCompare(b.endMonth) || a.name.localeCompare(b.name),
  );
}

/** How many goals each filter would show. */
export function countByPhase(goals: readonly Goal[], current: string): Record<PhaseFilter, number> {
  const counts: Record<PhaseFilter, number> = {
    all: goals.length,
    active: 0,
    planned: 0,
    completed: 0,
    cancelled: 0,
  };
  for (const goal of goals) counts[goalPhase(goal, current)]++;
  return counts;
}

export function filterGoals(goals: readonly Goal[], filter: PhaseFilter, current: string): Goal[] {
  return filter === 'all' ? [...goals] : goals.filter((g) => goalPhase(g, current) === filter);
}

/**
 * The movement's amount with a sign: positive when money comes in. The account
 * and a goal see allocations, releases and covers in opposite directions.
 */
export function signedCents(movement: Movement, view: 'account' | 'goal'): number {
  const inflow = view === 'account' ? movement.accountInflow : movement.goalInflow;
  return inflow ? movement.amountCents : -movement.amountCents;
}

export interface GoalTotals {
  /** Saved in the active goals. */
  savedCents: number;
  /** The targets of the active goals. */
  targetCents: number;
  activeCount: number;
}

/** The totals over the active goals, as the overview counts them. */
export function goalTotals(
  goals: readonly Pick<Goal, 'status' | 'balanceCents' | 'targetCents'>[],
): GoalTotals {
  const sum: GoalTotals = { savedCents: 0, targetCents: 0, activeCount: 0 };
  for (const g of goals) {
    if (g.status !== 'active') continue;
    sum.activeCount++;
    sum.savedCents += g.balanceCents;
    sum.targetCents += g.targetCents;
  }
  return sum;
}

/** The balance a goal shows: the one it closed with once completed, else what it holds now. */
export function shownBalance(
  goal: Pick<Goal, 'status' | 'balanceCents' | 'closingBalanceCents'>,
): number {
  return goal.status === 'completed' && goal.closingBalanceCents !== null
    ? goal.closingBalanceCents
    : goal.balanceCents;
}

/** Green above zero, red below it, plain at zero. */
export function balanceTone(cents: number): 'in' | 'out' | undefined {
  if (cents > 0) return 'in';
  if (cents < 0) return 'out';
  return undefined;
}

/**
 * How much an open goal has spent beyond its target, or 0. Goals are for
 * spending, so only going over the target is worth a warning.
 */
export function overspentCents(
  goal: Pick<Goal, 'status' | 'expensesCents' | 'targetCents'>,
): number {
  if (goal.status !== 'active') return 0;
  return Math.max(0, goal.expensesCents - goal.targetCents);
}

/**
 * The goal's balance after each movement. The list is newest first, like the
 * API returns it, so the balance is built from the end.
 */
export function balancesAfter(movements: readonly Movement[]): Map<number, number> {
  const result = new Map<number, number>();
  let running = 0;
  for (let i = movements.length - 1; i >= 0; i--) {
    running += signedCents(movements[i], 'goal');
    result.set(movements[i].id, running);
  }
  return result;
}

export interface GoalSeries {
  months: string[];
  /** The balance at the end of each month, up to the last month that has happened. */
  balance: number[];
  /** What the balance would be if nothing was ever spent: the planned allocations added up. */
  planned: number[];
}

/**
 * The goal's balance over time against its plan. The target is split in whole
 * cents over the months, the leftover cents going one each to the first months,
 * like the server does.
 */
export function goalSeries(
  goal: Pick<Goal, 'startMonth' | 'endMonth' | 'targetCents'>,
  movements: readonly Movement[],
  current: string,
): GoalSeries {
  const lastMovement = movements.reduce<string>((last, m) => (m.month > last ? m.month : last), '');
  const last = lastMovement > goal.endMonth ? lastMovement : goal.endMonth;
  const months = monthsBetween(goal.startMonth, last);

  const planMonths = monthsBetween(goal.startMonth, goal.endMonth).length;
  const base = Math.floor(goal.targetCents / planMonths);
  const extra = goal.targetCents % planMonths;
  let plan = 0;
  const planned = months.map((_, i) => {
    if (i < planMonths) plan += base + (i < extra ? 1 : 0);
    return plan;
  });

  const cutoff = current < last ? current : last;
  const balance: number[] = [];
  let running = 0;
  for (const m of months) {
    if (m > cutoff) break;
    for (const x of movements)
      if (x.month === m && x.goalId !== null) running += signedCents(x, 'goal');
    balance.push(running);
  }
  return { months, balance, planned };
}
