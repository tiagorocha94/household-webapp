import { describe, expect, it } from 'vitest';
import type { OverviewAttention } from '../api/client';
import { badgeText, groupNotifications, reviewBreakdown, reviewHeadline } from './notifications';

const item = (kind: string, name = '', month = '2026-11'): OverviewAttention =>
  ({ kind, name, month, amountCents: 10000, pct: 0 }) as OverviewAttention;

const items = [
  item('goals_shortfall', 'Emergency', '2027-01'),
  item('unassigned_savings'),
  item('credit_unpaid', 'Car loan'),
  item('budget_over', 'Fun'),
  item('unassigned_investments', '', '2026-10'),
  item('unassigned_goals'),
];

describe('groupNotifications', () => {
  it('puts data to enter first and plan and budget second', () => {
    const groups = groupNotifications(items, '2026-11');
    expect(groups.map((g) => g.title)).toEqual(['Enter or fix data', 'Plan and budget']);
    expect(groups[0].lines.map((l) => l.line.topic)).toEqual([
      'Savings',
      'Credits',
      'Investments',
      'Goals',
    ]);
    expect(groups[1].lines.map((l) => l.line.topic)).toEqual(['Goals', 'Budget']);
  });

  it('leaves out a group with nothing in it', () => {
    expect(groupNotifications([item('budget_over', 'Fun')], '2026-11').map((g) => g.title)).toEqual(
      ['Plan and budget'],
    );
    expect(groupNotifications([], '2026-11')).toEqual([]);
  });

  it('gives every line its own key', () => {
    const keys = groupNotifications(items, '2026-11').flatMap((g) => g.lines.map((l) => l.key));
    expect(new Set(keys).size).toBe(keys.length);
  });
});

describe('the review summary', () => {
  it('counts and breaks down the things to review', () => {
    expect(reviewHeadline(items)).toBe('6 things to review');
    expect(reviewBreakdown(items)).toBe('Unassigned money 3 · Credits 1 · Goals and budget 2');
  });

  it('is singular for one and absent for none', () => {
    expect(reviewHeadline([item('credit_unpaid', 'Car loan')])).toBe('1 thing to review');
    expect(reviewHeadline([])).toBeNull();
    expect(reviewBreakdown([])).toBe('');
  });
});

describe('badgeText', () => {
  it('shows up to 9 and then 9+', () => {
    expect(badgeText(4)).toBe('4');
    expect(badgeText(9)).toBe('9');
    expect(badgeText(10)).toBe('9+');
  });
});
