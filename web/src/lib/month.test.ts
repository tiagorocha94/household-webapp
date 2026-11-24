import { describe, expect, it } from 'vitest';
import {
  currentMonth,
  formatDay,
  formatMonth,
  isMonth,
  monthsBetween,
  previousMonth,
} from './month';

describe('formatMonth', () => {
  it('formats a month with its short name and year', () => {
    expect(formatMonth('2026-10')).toBe('Oct 2026');
    expect(formatMonth('2026-09')).toBe('Sept 2026');
    expect(formatMonth('2026-01')).toBe('Jan 2026');
  });

  it('returns anything that is not a month unchanged', () => {
    expect(formatMonth('')).toBe('');
    expect(formatMonth('2026-13')).toBe('2026-13');
  });
});

describe('currentMonth', () => {
  it('pads the month number', () => {
    expect(currentMonth(new Date(2026, 0, 31))).toBe('2026-01');
    expect(currentMonth(new Date(2026, 11, 1))).toBe('2026-12');
  });
});

describe('monthsBetween', () => {
  it('lists every month inclusive, across a year boundary', () => {
    expect(monthsBetween('2025-11', '2026-02')).toEqual([
      '2025-11',
      '2025-12',
      '2026-01',
      '2026-02',
    ]);
    expect(monthsBetween('2026-05', '2026-05')).toEqual(['2026-05']);
  });
  it('is empty when the end is before the start or a month is invalid', () => {
    expect(monthsBetween('2026-05', '2026-01')).toEqual([]);
    expect(monthsBetween('nope', '2026-01')).toEqual([]);
  });
});

describe('isMonth', () => {
  it('accepts only canonical months', () => {
    expect(isMonth('2026-10')).toBe(true);
    expect(isMonth('2026-12')).toBe(true);
    for (const bad of [
      '',
      '2026',
      '2026-1',
      '2026-00',
      '2026-13',
      '2026-10-01',
      ' 2026-10',
      'abc',
    ]) {
      expect(isMonth(bad)).toBe(false);
    }
  });
});

describe('previousMonth', () => {
  it('steps back one month, across a year boundary', () => {
    expect(previousMonth('2026-10')).toBe('2026-09');
    expect(previousMonth('2026-02')).toBe('2026-01');
    expect(previousMonth('2026-01')).toBe('2025-12');
  });

  it('is empty for anything that is not a month', () => {
    expect(previousMonth('')).toBe('');
    expect(previousMonth('2026-13')).toBe('');
  });
});

describe('formatDay', () => {
  it('formats a day with its number and short month', () => {
    expect(formatDay('2026-10-05')).toBe('5 Oct');
    expect(formatDay('2026-12-31')).toBe('31 Dec');
  });

  it('returns anything that is not a day unchanged', () => {
    expect(formatDay('')).toBe('');
    expect(formatDay('2026-10')).toBe('2026-10');
    expect(formatDay('2026-02-32')).toBe('2026-02-32');
  });
});
