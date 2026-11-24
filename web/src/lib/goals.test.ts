import { afterEach, describe, expect, it } from 'vitest';
import { setLocale } from './i18n/index.svelte';
import type { Goal, Movement } from '../api/client';
import {
  balancesAfter,
  balanceTone,
  countByPhase,
  filterGoals,
  goalPhase,
  goalSeries,
  goalTotals,
  overspentCents,
  phaseLabel,
  shownBalance,
  signedCents,
  sortGoals,
} from './goals';

function goal(over: Partial<Goal>): Goal {
  return {
    id: 1,
    accountId: 1,
    name: 'Trip',
    targetCents: 120000,
    startMonth: '2026-01',
    endMonth: '2026-12',
    status: 'active',
    totalMonths: 12,
    elapsedMonths: 0,
    monthlyAllocationCents: 10000,
    allocatedCents: 0,
    expensesCents: 0,
    balanceCents: 0,
    closingBalanceCents: null,
    progressPct: 0,
    ...over,
  };
}

describe('goalPhase', () => {
  it('calls an active goal that has not started planned', () => {
    expect(goalPhase(goal({ startMonth: '2026-11' }), '2026-10')).toBe('planned');
  });
  it('keeps a started active goal active, including in its start month', () => {
    expect(goalPhase(goal({ startMonth: '2026-10' }), '2026-10')).toBe('active');
  });
  it('keeps completed and cancelled as they are', () => {
    expect(goalPhase(goal({ status: 'completed', startMonth: '2027-01' }), '2026-10')).toBe(
      'completed',
    );
    expect(goalPhase(goal({ status: 'cancelled' }), '2026-10')).toBe('cancelled');
  });
});

describe('sortGoals, countByPhase and filterGoals', () => {
  const goals = [
    goal({ id: 1, name: 'B', status: 'completed', endMonth: '2026-03' }),
    goal({ id: 2, name: 'A', endMonth: '2026-12' }),
    goal({ id: 3, name: 'C', startMonth: '2027-01', endMonth: '2027-06' }),
    goal({ id: 4, name: 'D', endMonth: '2026-11' }),
  ];
  it('orders active, planned, then completed, soonest end first', () => {
    expect(sortGoals(goals, '2026-10').map((g) => g.id)).toEqual([4, 2, 3, 1]);
  });
  it('counts and filters by phase', () => {
    expect(countByPhase(goals, '2026-10')).toEqual({
      all: 4,
      active: 2,
      planned: 1,
      completed: 1,
      cancelled: 0,
    });
    expect(filterGoals(goals, 'planned', '2026-10').map((g) => g.id)).toEqual([3]);
    expect(filterGoals(goals, 'all', '2026-10')).toHaveLength(4);
  });
});

describe('signedCents', () => {
  const allocation: Movement = {
    id: 1,
    accountId: 1,
    goalId: 1,
    goalName: 'Trip',
    type: 'allocation',
    amountCents: 5000,
    month: '2026-10',
    description: '',
    createdAt: '',
    accountInflow: false,
    goalInflow: true,
  };
  it('reads an allocation as out for the account and in for the goal', () => {
    expect(signedCents(allocation, 'account')).toBe(-5000);
    expect(signedCents(allocation, 'goal')).toBe(5000);
  });
});

describe('goalTotals', () => {
  it('adds up the active goals only', () => {
    expect(
      goalTotals([
        { status: 'active', balanceCents: 1000, targetCents: 5000 },
        { status: 'active', balanceCents: 500, targetCents: 2000 },
        { status: 'completed', balanceCents: 9000, targetCents: 9000 },
      ]),
    ).toEqual({ savedCents: 1500, targetCents: 7000, activeCount: 2 });
  });
});

function movement(over: Partial<Movement>): Movement {
  return {
    id: 1,
    accountId: 1,
    goalId: 1,
    goalName: 'Boiler',
    type: 'allocation',
    amountCents: 0,
    month: '2026-01',
    description: '',
    createdAt: '',
    accountInflow: false,
    goalInflow: true,
    ...over,
  };
}

describe('shownBalance and balanceTone', () => {
  it('shows the closing balance of a completed goal and the balance otherwise', () => {
    const base = { balanceCents: 0, closingBalanceCents: -350 };
    expect(shownBalance({ ...base, status: 'completed' })).toBe(-350);
    expect(shownBalance({ ...base, status: 'active', balanceCents: 900 })).toBe(900);
    expect(shownBalance({ status: 'completed', balanceCents: 0, closingBalanceCents: null })).toBe(
      0,
    );
  });
  it('colours by sign', () => {
    expect(balanceTone(1)).toBe('in');
    expect(balanceTone(-1)).toBe('out');
    expect(balanceTone(0)).toBeUndefined();
  });
});

describe('overspentCents', () => {
  it('is how far an open goal has spent past its target', () => {
    expect(overspentCents({ status: 'active', expensesCents: 215000, targetCents: 180000 })).toBe(
      35000,
    );
    expect(overspentCents({ status: 'active', expensesCents: 180000, targetCents: 180000 })).toBe(
      0,
    );
  });
  it('is zero once the goal is not open', () => {
    expect(
      overspentCents({ status: 'completed', expensesCents: 215000, targetCents: 180000 }),
    ).toBe(0);
  });
});

describe('balancesAfter', () => {
  it('builds the balance from the oldest movement, given newest first', () => {
    const list = [
      movement({ id: 3, type: 'cover', amountCents: 350, month: '2026-07', goalInflow: true }),
      movement({ id: 2, type: 'expense', amountCents: 2150, month: '2026-06', goalInflow: false }),
      movement({ id: 1, type: 'allocation', amountCents: 1800, month: '2026-01' }),
    ];
    const after = balancesAfter(list);
    expect([after.get(1), after.get(2), after.get(3)]).toEqual([1800, -350, 0]);
  });
});

describe('goalSeries', () => {
  const plan = { startMonth: '2026-01', endMonth: '2026-03', targetCents: 10000 };
  it('plans the target in whole cents, leftover cents first', () => {
    const s = goalSeries(plan, [], '2026-01');
    expect(s.months).toEqual(['2026-01', '2026-02', '2026-03']);
    expect(s.planned).toEqual([3334, 6667, 10000]);
  });
  it('follows the balance only up to the current month', () => {
    const s = goalSeries(
      plan,
      [
        movement({ id: 1, amountCents: 3334, month: '2026-01' }),
        movement({ id: 2, amountCents: 3333, month: '2026-02' }),
      ],
      '2026-02',
    );
    expect(s.balance).toEqual([3334, 6667]);
  });
  it('reaches past the end month for a movement that comes later', () => {
    const s = goalSeries(
      plan,
      [movement({ id: 1, type: 'expense', amountCents: 500, month: '2026-05', goalInflow: false })],
      '2026-06',
    );
    expect(s.months[s.months.length - 1]).toBe('2026-05');
    expect(s.planned[4]).toBe(10000);
    expect(s.balance[4]).toBe(-500);
  });
});

describe('phaseLabel', () => {
  afterEach(() => setLocale('en'));

  it('names the states in English', () => {
    expect(phaseLabel('all')).toBe('All');
    expect(phaseLabel('cancelled')).toBe('Cancelled');
  });

  it('names the states in Portuguese', () => {
    setLocale('pt');
    expect(phaseLabel('all')).toBe('Todos');
    expect(phaseLabel('active')).toBe('Ativos');
    expect(phaseLabel('completed')).toBe('Concluídos');
  });
});
