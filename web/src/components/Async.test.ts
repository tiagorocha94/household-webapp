import { createRawSnippet } from 'svelte';
import { render } from 'svelte/server';
import { describe, expect, it } from 'vitest';
import { ApiError } from '../api/client';
import { Resource } from '../lib/resource.svelte';
import Async from './Async.svelte';

const children = createRawSnippet<[string]>((data) => ({
  render: () => `<p>${data()}</p>`,
}));

// Async is generic; the snippet is typed for the one string the tests use.
const page = (resource: Resource<string>, what?: string) =>
  render(Async as never, { props: { resource, what, children } } as never).body;

async function settled<T>(fetch: () => Promise<T>): Promise<Resource<T>> {
  const resource = new Resource(fetch);
  await new Promise((resolve) => setTimeout(resolve, 0));
  return resource;
}

describe('Async', () => {
  it('says what a page is reading while it loads', () => {
    const resource = new Resource(() => new Promise<string>(() => {}));
    expect(page(resource, 'goals')).toContain('Reading your goals…');
  });

  it('says only Loading for a read inside a page', () => {
    const resource = new Resource(() => new Promise<string>(() => {}));
    const html = page(resource);
    expect(html).toContain('Loading…');
    expect(html).not.toContain('Reading your');
  });

  it('shows the data, the time it was read and Refresh for the main read of a page', async () => {
    const resource = await settled(async () => 'hello');
    const html = page(resource, 'goals');
    expect(html).toContain('<p>hello</p>');
    expect(html).toMatch(/Updated \d\d:\d\d/);
    expect(html).toContain('Refresh');
  });

  it('shows the data without the Updated line for a read inside a page', async () => {
    const resource = await settled(async () => 'hello');
    const html = page(resource);
    expect(html).toContain('<p>hello</p>');
    expect(html).not.toContain('Updated');
  });

  it('shows the reason and Try again when the server cannot be reached', async () => {
    const resource = await settled(async () => {
      throw new ApiError(0, 'network');
    });
    const html = page(resource, 'goals');
    expect(html).toContain('Cannot reach the server');
    expect(html).toContain('Try again');
  });
});
