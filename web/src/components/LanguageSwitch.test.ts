import { render } from 'svelte/server';
import { afterEach, describe, expect, it } from 'vitest';
import { setLocale } from '../lib/i18n/index.svelte';
import LanguageSwitch from './LanguageSwitch.svelte';

afterEach(() => setLocale('en'));

const html = () => render(LanguageSwitch as never, {} as never).body as string;

describe('LanguageSwitch', () => {
  it('offers both languages and marks the current one', () => {
    setLocale('en');
    const out = html();
    expect(out).toMatch(/lang="en"[^>]*aria-pressed="true"/);
    expect(out).toMatch(/lang="pt"[^>]*aria-pressed="false"/);
  });

  it('follows the language and names the group in it', () => {
    setLocale('pt');
    const out = html();
    expect(out).toMatch(/lang="pt"[^>]*aria-pressed="true"/);
    expect(out).toContain('aria-label="Idioma"');
  });
});
