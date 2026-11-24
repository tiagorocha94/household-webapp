import { readFileSync } from 'node:fs';
import { describe, expect, it } from 'vitest';
import { contrast, iconColorOn, luminance } from './color';

describe('luminance and contrast', () => {
  it('has the known extremes', () => {
    expect(luminance('#000000')).toBe(0);
    expect(luminance('#ffffff')).toBeCloseTo(1, 10);
    expect(contrast('#000000', '#ffffff')).toBeCloseTo(21, 5);
    expect(contrast('#ffffff', '#ffffff')).toBeCloseTo(1, 10);
  });

  it('is symmetric', () => {
    expect(contrast('#16A34A', '#ffffff')).toBeCloseTo(contrast('#ffffff', '#16A34A'), 10);
  });

  it('refuses a colour that is not #RRGGBB', () => {
    expect(() => luminance('red')).toThrow();
    expect(() => luminance('#fff')).toThrow();
  });
});

describe('iconColorOn', () => {
  it('uses white on dark and medium colours', () => {
    expect(iconColorOn('#0D9488')).toBe('#ffffff');
    expect(iconColorOn('#16A34A')).toBe('#ffffff');
    expect(iconColorOn('#475569')).toBe('#ffffff');
    expect(iconColorOn('#7C3AED')).toBe('#ffffff');
  });

  it('uses dark ink on light colours', () => {
    expect(iconColorOn('#EAB308')).toBe('#15202b');
    expect(iconColorOn('#84CC16')).toBe('#15202b');
    expect(iconColorOn('#F59E0B')).toBe('#15202b');
    expect(iconColorOn('#9CA3AF')).toBe('#15202b');
  });

  it('gives every category icon at least 3:1 against its badge', () => {
    const csv = readFileSync(
      new URL('../../../demo/cashflow_categories.csv', import.meta.url),
      'utf8',
    );
    for (const row of csv.trim().split('\n').slice(1)) {
      const [name, , , color] = row.split(',');
      expect(contrast(iconColorOn(color), color), name).toBeGreaterThanOrEqual(3);
    }
  });
});
