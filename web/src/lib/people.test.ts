import { describe, expect, it } from 'vitest';
import type { CashflowUserMonth, CashflowUserTotal } from '../api/client';
import { personSplit, putAsideHistory } from './people';

const user = (over: Partial<CashflowUserTotal>): CashflowUserTotal => ({
  username: 'alice',
  incomeCents: 320000,
  expenseCents: 0,
  netCents: 0,
  spentCents: 0,
  putAsideCents: 0,
  cashCents: null,
  ...over,
});

describe('personSplit', () => {
  it('splits income into spent, put aside and left over', () => {
    const s = personSplit(user({ spentCents: 118000, putAsideCents: 115000 }));
    expect(s.leftOverCents).toBe(87000);
    expect(s.putAsideShare).toBe(36);
    expect(s.spentShare).toBe(37);
    expect(s.spentWidth + s.putAsideWidth + s.leftOverWidth).toBeCloseTo(100);
  });

  it('never lets the bar overflow when more went out than came in', () => {
    const s = personSplit(user({ incomeCents: 100000, spentCents: 90000, putAsideCents: 60000 }));
    expect(s.leftOverCents).toBe(-50000);
    expect(s.leftOverWidth).toBe(0);
    expect(s.spentWidth + s.putAsideWidth).toBeCloseTo(100);
  });

  it('has no shares without income', () => {
    const s = personSplit(user({ incomeCents: 0, spentCents: 5000 }));
    expect(s.putAsideShare).toBeNull();
    expect(s.spentShare).toBeNull();
    expect(s.spentWidth).toBe(100);
  });
});

describe('putAsideHistory', () => {
  const rows: CashflowUserMonth[] = [
    { month: '2026-09', username: 'alice', incomeCents: 300000, putAsideCents: 90000 },
    { month: '2026-11', username: 'alice', incomeCents: 300000, putAsideCents: 60000 },
    { month: '2026-11', username: 'bob', incomeCents: 200000, putAsideCents: 40000 },
  ];

  it('lists the six months ending at the month, oldest first', () => {
    expect(putAsideHistory(rows, '2026-11').months).toEqual([
      '2026-06',
      '2026-07',
      '2026-08',
      '2026-09',
      '2026-10',
      '2026-11',
    ]);
  });

  it('puts each person in a row, null where they had no movements', () => {
    const h = putAsideHistory(rows, '2026-11');
    expect(h.people.map((p) => p.username)).toEqual(['alice', 'bob']);
    expect(h.people[0].cells).toEqual([
      null,
      null,
      null,
      { putAsideCents: 90000, sharePct: 30 },
      null,
      { putAsideCents: 60000, sharePct: 20 },
    ]);
    expect(h.people[1].cells[3]).toBeNull();
    expect(h.people[1].cells[5]).toEqual({ putAsideCents: 40000, sharePct: 20 });
  });

  it('adds everyone up for the household, null in a month without rows', () => {
    const h = putAsideHistory(rows, '2026-11');
    expect(h.household[5]).toEqual({ putAsideCents: 100000, sharePct: 20 });
    expect(h.household[3]).toEqual({ putAsideCents: 90000, sharePct: 30 });
    expect(h.household[4]).toBeNull();
  });
});
