import { createRawSnippet } from 'svelte';
import { render } from 'svelte/server';
import { afterEach, beforeEach, describe, expect, it } from 'vitest';
import { ApiError } from '../api/client';
import { setLocale } from '../lib/i18n/index.svelte';
import { Resource } from '../lib/resource.svelte';
import Async from './Async.svelte';

const children = createRawSnippet<[string]>((data) => ({
  render: () => `<p>${data()}</p>`,
}));

const page = (resource: Resource<string>, what?: string) =>
  render(Async as never, { props: { resource, what, children } } as never).body;

async function settled<T>(fetch: () => Promise<T>): Promise<Resource<T>> {
  const resource = new Resource(fetch);
  await new Promise((resolve) => setTimeout(resolve, 0));
  return resource;
}

beforeEach(() => setLocale('pt'));
afterEach(() => setLocale('en'));

describe('Async in Portuguese', () => {
  it('says what a page is reading, and only A carregar for a read inside a page', () => {
    const resource = new Resource(() => new Promise<string>(() => {}));
    expect(page(resource, 'goals')).toContain('A ler os seus objetivos…');
    expect(page(resource)).toContain('A carregar…');
  });

  it('shows the Updated line and Refresh', async () => {
    const html = page(await settled(async () => 'hello'), 'goals');
    expect(html).toMatch(/Atualizado às \d\d:\d\d/);
    expect(html).toContain('Atualizar');
  });

  it('shows the reason and Try again', async () => {
    const resource = await settled(async () => {
      throw new ApiError(0, 'network');
    });
    const html = page(resource, 'goals');
    expect(html).toContain('Não foi possível contactar o servidor');
    expect(html).toContain('Tentar de novo');
  });
});
