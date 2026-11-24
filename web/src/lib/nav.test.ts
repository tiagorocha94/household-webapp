import { describe, expect, it } from 'vitest';
import { NAV, isCurrent, loadCollapsed, saveCollapsed } from './nav';
import { ICON_NAMES } from './icons';

const item = (label: string) => NAV.find((i) => i.label === label)!;

describe('isCurrent', () => {
  it('marks Overview only on the home address', () => {
    expect(isCurrent(item('Overview'), '/')).toBe(true);
    expect(isCurrent(item('Overview'), '/goals')).toBe(false);
  });

  it('keeps an item current on its sub-pages', () => {
    expect(isCurrent(item('Goals'), '/accounts/3')).toBe(true);
    expect(isCurrent(item('Goals'), '/goals/7')).toBe(true);
    expect(isCurrent(item('Cashflow'), '/cashflow/months/2026-11')).toBe(true);
    expect(isCurrent(item('Savings'), '/credits')).toBe(false);
  });
});

describe('menu icons', () => {
  it('are all registered', () => {
    for (const i of NAV) expect(ICON_NAMES, i.label).toContain(i.icon);
  });
});

describe('menu order and icons shared with the phone app', () => {
  it('puts Cashflow right after Overview', () => {
    expect(NAV.slice(0, 2).map((i) => i.label)).toEqual(['Overview', 'Cashflow']);
  });

  it('uses the same icon as the phone for the pages both have', () => {
    expect(item('Savings').icon).toBe('vault');
    expect(item('Credits').icon).toBe('landmark');
    expect(item('Goals').icon).toBe('target');
    expect(item('Investments').icon).toBe('trending-up');
    expect(item('Overview').icon).toBe('house');
  });
});

describe('collapsed choice', () => {
  const store = () => {
    const data = new Map<string, string>();
    return {
      getItem: (k: string) => data.get(k) ?? null,
      setItem: (k: string, v: string) => void data.set(k, v),
    };
  };

  it('is remembered', () => {
    const s = store();
    expect(loadCollapsed(s)).toBe(false);
    saveCollapsed(true, s);
    expect(loadCollapsed(s)).toBe(true);
    saveCollapsed(false, s);
    expect(loadCollapsed(s)).toBe(false);
  });

  it('falls back when storage throws', () => {
    const broken = {
      getItem: () => {
        throw new Error('blocked');
      },
      setItem: () => {
        throw new Error('blocked');
      },
    };
    expect(loadCollapsed(broken)).toBe(false);
    expect(() => saveCollapsed(true, broken)).not.toThrow();
  });
});
