import { describe, expect, it } from 'vitest';
import {
  allocationShares,
  buildHistory,
  formatGainPct,
  gainTone,
  ownerLabel,
  sortInstruments,
  totals,
  typeLabel,
} from './investments';
import type { InvestmentInstrument } from '../api/client';

function instrument(over: Partial<InvestmentInstrument>): InvestmentInstrument {
  return {
    id: 1,
    entityId: 1,
    entityName: 'Broker',
    name: 'ETF',
    type: 'etf',
    notes: '',
    user: null,
    contributionCount: 0,
    snapshotCount: 0,
    allocatedCents: 0,
    currentValueCents: 0,
    gainCents: 0,
    gainPct: 0,
    lastSnapshotMonth: null,
    ...over,
  };
}

describe('formatGainPct', () => {
  it('divides the scaled percentage by 100 and shows the sign', () => {
    expect(formatGainPct(734).replace(/\s/g, '')).toBe('+7,34%');
    expect(formatGainPct(-400).replace(/\s/g, '')).toBe('-4,00%');
    expect(formatGainPct(0).replace(/\s/g, '')).toBe('0,00%');
  });
});

describe('ownerLabel', () => {
  it('names the person, or the household when there is none', () => {
    expect(ownerLabel('alice')).toBe('alice');
    expect(ownerLabel(null)).toBe('Household');
  });
});

describe('gainTone', () => {
  it('tells gains from losses', () => {
    expect(gainTone(1)).toBe('in');
    expect(gainTone(-1)).toBe('out');
    expect(gainTone(0)).toBeUndefined();
  });
});

describe('totals', () => {
  it('adds up and derives the scaled percentage from the sums', () => {
    expect(
      totals([
        { allocatedCents: 25000, currentValueCents: 24000 },
        { allocatedCents: 20000, currentValueCents: 22000 },
      ]),
    ).toEqual({ allocatedCents: 45000, currentValueCents: 46000, gainCents: 1000, gainPct: 222 });
  });

  it('is zero without allocations', () => {
    expect(totals([])).toEqual({
      allocatedCents: 0,
      currentValueCents: 0,
      gainCents: 0,
      gainPct: 0,
    });
  });
});

describe('allocationShares', () => {
  it('orders by allocation and shares sum to 100', () => {
    const shares = allocationShares([
      { name: 'B', allocatedCents: 25 },
      { name: 'A', allocatedCents: 75 },
    ]);
    expect(shares.map((s) => [s.entity.name, s.sharePct])).toEqual([
      ['A', 75],
      ['B', 25],
    ]);
  });

  it('gives no share when nothing is allocated', () => {
    expect(allocationShares([{ name: 'A', allocatedCents: 0 }])[0].sharePct).toBe(0);
  });
});

describe('sortInstruments', () => {
  it('puts the most valuable first, then by name, without changing the input', () => {
    const input = [
      instrument({ id: 1, name: 'B', currentValueCents: 5 }),
      instrument({ id: 2, name: 'A', currentValueCents: 5 }),
      instrument({ id: 3, name: 'C', currentValueCents: 9 }),
    ];
    expect(sortInstruments(input).map((i) => i.name)).toEqual(['C', 'A', 'B']);
    expect(input.map((i) => i.name)).toEqual(['B', 'A', 'C']);
  });
});

describe('typeLabel', () => {
  it('has friendly labels', () => {
    expect(typeLabel.certificados_aforro).toBe('Certificados de Aforro');
    expect(typeLabel.ppr).toBe('PPR');
  });
});

describe('buildHistory', () => {
  it('accumulates contributions and sums a month with several', () => {
    const history = buildHistory(
      [
        { month: '2026-01', amountCents: 10000 },
        { month: '2026-01', amountCents: 5000 },
        { month: '2026-03', amountCents: 10000 },
      ],
      [
        { id: 1, month: '2026-01', valueCents: 15500 },
        { id: 2, month: '2026-02', valueCents: 16000 },
        { id: 4, month: '2026-02', valueCents: 17000 },
        { id: 3, month: '2026-02', valueCents: 16500 },
        { id: 5, month: '2026-03', valueCents: 25500 },
      ],
    );
    expect(history).toEqual([
      { month: '2026-01', allocatedCents: 15000, valueCents: 15500, estimated: false },
      { month: '2026-02', allocatedCents: 15000, valueCents: 17000, estimated: false },
      { month: '2026-03', allocatedCents: 25000, valueCents: 25500, estimated: false },
    ]);
  });

  it('estimates missing months on the straight line between two snapshots', () => {
    // 200 to 300 over four months is 25 a month.
    const history = buildHistory(
      [{ month: '2026-01', amountCents: 20000 }],
      [
        { id: 1, month: '2026-01', valueCents: 20000 },
        { id: 2, month: '2026-05', valueCents: 30000 },
      ],
    );
    expect(history.map((p) => [p.month, p.valueCents, p.estimated])).toEqual([
      ['2026-01', 20000, false],
      ['2026-02', 22500, true],
      ['2026-03', 25000, true],
      ['2026-04', 27500, true],
      ['2026-05', 30000, false],
    ]);
  });

  it('keeps every calendar month across a year boundary and rounds to whole cents', () => {
    const history = buildHistory(
      [],
      [
        { id: 1, month: '2025-11', valueCents: 100 },
        { id: 2, month: '2026-02', valueCents: 110 },
      ],
    );
    expect(history.map((p) => [p.month, p.valueCents])).toEqual([
      ['2025-11', 100],
      ['2025-12', 103],
      ['2026-01', 107],
      ['2026-02', 110],
    ]);
  });

  it('has no value before the first snapshot or after the last', () => {
    const history = buildHistory(
      [
        { month: '2026-01', amountCents: 100 },
        { month: '2026-04', amountCents: 100 },
      ],
      [{ id: 1, month: '2026-02', valueCents: 150 }],
    );
    expect(history.map((p) => [p.month, p.valueCents, p.estimated])).toEqual([
      ['2026-01', null, false],
      ['2026-02', 150, false],
      ['2026-03', null, false],
      ['2026-04', null, false],
    ]);
  });

  it('is empty without data', () => {
    expect(buildHistory([], [])).toEqual([]);
  });
});
