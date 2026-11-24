import { describe, expect, it } from 'vitest';
import { MODULES, isEnabled, moduleOfRoute, visibleNav } from './modules';
import { NAV } from './nav';

describe('moduleOfRoute', () => {
  it('names the module of each page of a module', () => {
    expect(moduleOfRoute('cashflow-month-view')).toBe('cashflow');
    expect(moduleOfRoute('goal')).toBe('goals');
    expect(moduleOfRoute('saving')).toBe('savings');
    expect(moduleOfRoute('contract')).toBe('credits');
    expect(moduleOfRoute('instrument')).toBe('investments');
  });

  it('leaves the pages that are always there without a module', () => {
    for (const name of ['home', 'net-worth', 'notifications', 'settings', 'login', undefined]) {
      expect(moduleOfRoute(name), String(name)).toBeNull();
    }
  });
});

describe('isEnabled', () => {
  it('hides nothing until the server has answered', () => {
    for (const module of MODULES) expect(isEnabled(null, module)).toBe(true);
  });

  it('shows a module only when the server runs it', () => {
    expect(isEnabled(['cashflow'], 'cashflow')).toBe(true);
    expect(isEnabled(['cashflow'], 'goals')).toBe(false);
    expect(isEnabled([], null)).toBe(true);
  });
});

describe('visibleNav', () => {
  const labels = (enabled: string[] | null) => visibleNav(NAV, enabled).map((i) => i.label);

  it('keeps the whole menu when every module runs', () => {
    expect(labels([...MODULES])).toEqual(NAV.map((i) => i.label));
    expect(labels(null)).toEqual(NAV.map((i) => i.label));
  });

  it('drops the items of the modules that do not run, in the same order', () => {
    expect(labels(['cashflow', 'savings'])).toEqual([
      'Overview',
      'Cashflow',
      'Net worth',
      'Savings',
    ]);
  });

  it('gives every module an item', () => {
    expect(new Set(NAV.flatMap((i) => (i.module ? [i.module] : [])))).toEqual(new Set(MODULES));
  });
});
