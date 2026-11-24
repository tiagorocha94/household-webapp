import { readFileSync } from 'node:fs';
import { describe, expect, it } from 'vitest';
import { ICON_NAMES, UNKNOWN_ICON, iconComponent } from './icons';

describe('icon registry', () => {
  it('uses kebab-case Lucide names', () => {
    for (const name of ICON_NAMES) expect(name).toMatch(/^[a-z][a-z0-9]*(-[a-z0-9]+)*$/);
  });

  it('draws every icon of the categories in demo/', () => {
    const csv = readFileSync(
      new URL('../../../demo/cashflow_categories.csv', import.meta.url),
      'utf8',
    );
    const rows = csv.trim().split('\n').slice(1);
    expect(rows.length).toBeGreaterThan(0);
    for (const row of rows) {
      const [name, , icon] = row.split(',');
      expect(ICON_NAMES, `${name} uses ${icon}`).toContain(icon);
    }
  });

  it('draws the question mark for a name it does not have', () => {
    expect(iconComponent('not-an-icon')).toBe(iconComponent(UNKNOWN_ICON));
    expect(iconComponent('toString')).toBe(iconComponent(UNKNOWN_ICON));
  });
});
