import { describe, expect, it } from 'vitest';
import { categoryNamesPt, groupNamesPt } from './category-names';
import { en } from './en';
import {
  categoryLabel,
  chosenLocale,
  detectLocale,
  fill,
  groupLabel,
  getLocale,
  localeTag,
  setLocale,
  t,
} from './index.svelte';
import { pt } from './pt';

const placeholders = (text: string) => [...text.matchAll(/\{(\w+)\}/g)].map((m) => m[1]).sort();

describe('the texts', () => {
  it('have the same keys in both languages', () => {
    expect(Object.keys(pt).sort()).toEqual(Object.keys(en).sort());
  });

  it('have the same placeholders in both languages', () => {
    for (const key of Object.keys(en) as (keyof typeof en)[]) {
      expect(placeholders(pt[key]), key).toEqual(placeholders(en[key]));
    }
  });

  it('are never empty', () => {
    for (const [key, text] of Object.entries(pt as Record<string, string>))
      expect(text.trim(), key).not.toBe('');
  });
});

describe('detectLocale', () => {
  it('takes the first language that is Portuguese or English', () => {
    expect(detectLocale(['pt-PT', 'en-GB'])).toBe('pt');
    expect(detectLocale(['pt-BR'])).toBe('pt');
    expect(detectLocale(['fr-FR', 'en-US', 'pt'])).toBe('en');
  });

  it('falls back to English', () => {
    expect(detectLocale([])).toBe('en');
    expect(detectLocale(['de', 'fr'])).toBe('en');
  });
});

describe('chosenLocale', () => {
  it('accepts only the two languages', () => {
    expect(chosenLocale('pt')).toBe('pt');
    expect(chosenLocale('en')).toBe('en');
    expect(chosenLocale('fr')).toBeNull();
    expect(chosenLocale(null)).toBeNull();
  });
});

describe('fill', () => {
  it('fills the placeholders and leaves unknown ones alone', () => {
    expect(fill('{a} and {b} and {c}', { a: 1, b: 'two' })).toBe('1 and two and {c}');
  });
});

describe('the current language', () => {
  it('changes the names of categories and groups, and the Intl tag', () => {
    setLocale('en');
    expect(getLocale()).toBe('en');
    expect(categoryLabel('Groceries')).toBe('Groceries');
    expect(localeTag()).toBe('en-GB');
    setLocale('pt');
    expect(categoryLabel('Groceries')).toBe('Supermercado');
    expect(groupLabel('Goals & Savings')).toBe('Objetivos e poupança');
    expect(localeTag()).toBe('pt-PT');
    setLocale('en');
  });

  it('shows a name without a translation as it is', () => {
    setLocale('pt');
    expect(categoryLabel('Something new')).toBe('Something new');
    expect(groupLabel('constructor')).toBe('constructor');
    setLocale('en');
  });

  it('knows every category and group of the CSV files', () => {
    expect(Object.keys(categoryNamesPt).length).toBeGreaterThanOrEqual(24);
    expect(Object.keys(groupNamesPt)).toEqual([
      'Essential',
      'Investment',
      'Goals & Savings',
      'Fun',
    ]);
    expect(typeof t).toBe('function');
  });
});
