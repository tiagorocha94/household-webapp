import { describe, expect, it } from 'vitest';
import { formatCents, formatPercent, formatWholeEuros } from './money';

// Intl separates thousands and the currency sign with non-breaking spaces.
const plain = (s: string) => s.replace(/[  ]/g, ' ');

describe('formatCents', () => {
  it('formats cents as euros with a decimal comma', () => {
    expect(plain(formatCents(12345))).toBe('123,45 €');
    expect(plain(formatCents(5))).toBe('0,05 €');
    expect(plain(formatCents(0))).toBe('0,00 €');
  });

  it('keeps the sign of negative amounts', () => {
    expect(plain(formatCents(-2550))).toBe('-25,50 €');
  });

  it('groups thousands too, not only tens of thousands', () => {
    expect(plain(formatCents(123456))).toBe('1 234,56 €');
    expect(plain(formatCents(999900))).toBe('9 999,00 €');
    expect(plain(formatWholeEuros(123456))).toBe('1 235 €');
  });

  it('groups large amounts', () => {
    expect(plain(formatCents(123456789))).toBe('1 234 567,89 €');
  });
});

describe('formatPercent', () => {
  it('formats whole percentages', () => {
    expect(plain(formatPercent(42))).toBe('42%');
    expect(plain(formatPercent(100))).toBe('100%');
  });
});
