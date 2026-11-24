import { describe, expect, it } from 'vitest';
import { isMonthView } from './cashflow';

describe('isMonthView', () => {
  it('accepts the pages of a month and nothing else', () => {
    expect(isMonthView('budget')).toBe(true);
    expect(isMonthView('movements')).toBe(true);
    expect(isMonthView('nope')).toBe(false);
  });
});
