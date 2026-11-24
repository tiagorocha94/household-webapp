import { render } from 'svelte/server';
import { afterEach, describe, expect, it } from 'vitest';
import { clearLanguageChoice, setLocale } from '../lib/i18n/index.svelte';
import Settings from '../pages/Settings.svelte';

afterEach(() => {
  setLocale('en');
  clearLanguageChoice();
});

const html = () => render(Settings as never, {} as never).body as string;

describe('Settings', () => {
  it('offers the five choices and says they stay in the browser', () => {
    const out = html();
    for (const text of ['Theme', 'Language', 'Text size', 'Table rows', 'Accent colour']) {
      expect(out, text).toContain(text);
    }
    expect(out).toContain('Saved in this browser only');
  });

  it('marks System as chosen for the theme and the language at the start', () => {
    const out = html();
    expect(out.match(/aria-pressed="true"[^>]*>(?:<!--.*?-->)*System/g)?.length).toBe(2);
  });

  it('is in Portuguese when the language is', () => {
    setLocale('pt');
    const out = html();
    expect(out).toContain('Definições');
    expect(out).toContain('Tamanho do texto');
    expect(out).toContain('Cor de destaque');
  });
});
