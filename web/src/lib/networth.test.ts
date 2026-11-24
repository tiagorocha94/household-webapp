import { describe, expect, it } from 'vitest';
import type { NetWorthMonth } from '../api/client';
import { changeOver, inRange } from './networth';

const month = (m: string, net: number, credit: number): NetWorthMonth => ({
  month: m,
  savingsCents: 0,
  investmentsCents: 0,
  goalsCents: 0,
  cashCents: 0,
  assetsCents: net + credit,
  creditCents: credit,
  netCents: net,
});

const months = ['2026-01', '2026-02', '2026-03', '2026-04', '2026-05'].map((m, i) =>
  month(m, 1000 * (i + 1), 5000 - 100 * i),
);

describe('inRange', () => {
  it('keeps the last n months and the one before them', () => {
    expect(inRange(months, 3).map((m) => m.month)).toEqual([
      '2026-02',
      '2026-03',
      '2026-04',
      '2026-05',
    ]);
  });

  it('keeps everything for "all" or a range longer than the history', () => {
    expect(inRange(months, 'all')).toHaveLength(5);
    expect(inRange(months, 12)).toHaveLength(5);
  });
});

describe('changeOver', () => {
  it('measures net worth and credit against the month n months back', () => {
    expect(changeOver(months, 3)).toEqual({ since: '2026-02', netCents: 3000, creditCents: -300 });
  });

  it('is null when the history is too short', () => {
    expect(changeOver(months.slice(0, 3), 3)).toBeNull();
  });
});
