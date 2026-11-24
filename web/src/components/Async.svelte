<script lang="ts" generics="T">
  import type { Snippet } from 'svelte';
  import { describeError } from '../api/client';
  import type { Resource } from '../lib/resource.svelte';
  import { t } from '../lib/i18n/index.svelte';
  import { readingText, updatedText } from '../lib/updated';

  /**
   * `what` marks the main read of a page ("goals"): it names the loading text and adds the
   * "Updated" line with a Refresh button. Reads inside a page leave it out.
   */
  let {
    resource,
    what,
    children,
  }: { resource: Resource<T>; what?: string; children: Snippet<[T]> } = $props();
</script>

{#if resource.status === 'loading'}
  <p class="state" aria-live="polite">{what ? readingText(what) : t('shell.common.loading')}</p>
{:else if resource.status === 'error'}
  <div class="state error" role="alert">
    <p>{describeError(resource.error)}</p>
    <button type="button" onclick={() => resource.reload()}>{t('shell.common.tryAgain')}</button>
  </div>
{:else if resource.data !== undefined}
  {@render children(resource.data)}
  {#if what && resource.loadedAt}
    <p class="muted updated">
      {updatedText(resource.loadedAt)} ·
      <button class="refresh" type="button" onclick={() => resource.reload()}
        >{t('shell.common.refresh')}</button
      >
    </p>
  {/if}
{/if}

<style>
  .updated {
    margin: 1.5rem 0 0;
    font-size: 0.85rem;
  }
  .refresh {
    border: 0;
    background: none;
    padding: 0;
    color: var(--accent);
    text-decoration: underline;
    cursor: pointer;
  }
</style>
