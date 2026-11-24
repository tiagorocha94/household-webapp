import { render } from 'svelte/server';
import { describe, expect, it } from 'vitest';
import CategoryLabel from './CategoryLabel.svelte';
import Pill from './Pill.svelte';
import ProgressBar from './ProgressBar.svelte';
import Stat from './Stat.svelte';

// The components are rendered to a string, so the props are checked by what comes out.
const html = (component: unknown, props: Record<string, unknown>) =>
  render(component as never, { props } as never).body;

describe('Stat', () => {
  it('shows the label, the value and the tone of the value', () => {
    const out = html(Stat, { label: 'Free', value: '280,00 €', tone: 'out' });
    expect(out).toMatch(/<dt[^>]*>Free<\/dt>/);
    expect(out).toMatch(/<dd class="out[^"]*">280,00 €<\/dd>/);
  });

  it('adds a note only when there is one', () => {
    expect(html(Stat, { label: 'A', value: '1' })).not.toContain('note');
    expect(html(Stat, { label: 'A', value: '1', note: 'of 10' })).toContain('of 10');
  });
});

describe('ProgressBar', () => {
  it('is named and holds its percentage', () => {
    const out = html(ProgressBar, { pct: 42, label: 'Food: 42% of the budget spent' });
    expect(out).toContain('role="progressbar"');
    expect(out).toContain('aria-label="Food: 42% of the budget spent"');
    expect(out).toContain('aria-valuenow="42"');
  });

  it('keeps the bar between 0 and 100 while the text says the real figure', () => {
    const out = html(ProgressBar, { pct: 130, label: 'Over', over: true });
    expect(out).toContain('aria-valuenow="100"');
    expect(out).toContain('130');
  });
});

describe('Pill', () => {
  it('shows its text with the tone as a class', () => {
    expect(html(Pill, { text: 'Active', tone: 'in' })).toMatch(/class="pill in[ "]/);
  });
});

describe('CategoryLabel', () => {
  it('shows the name alone without a look', () => {
    const out = html(CategoryLabel, { name: 'Groceries' });
    expect(out).toContain('Groceries');
    expect(out).not.toContain('badge');
  });

  it('puts the icon on the colour of the category and hides it from screen readers', () => {
    const out = html(CategoryLabel, {
      name: 'Groceries',
      look: { icon: 'shopping-basket', color: '#16A34A' },
    });
    expect(out).toContain('Groceries');
    expect(out).toContain('aria-hidden="true"');
    expect(out.toLowerCase()).toContain('#16a34a');
  });
});
