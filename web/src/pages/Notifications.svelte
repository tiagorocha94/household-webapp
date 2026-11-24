<script lang="ts">
  import { api } from '../api/client';
  import Async from '../components/Async.svelte';
  import { t } from '../lib/i18n/index.svelte';
  import { groupNotifications } from '../lib/notifications';
  import { Resource } from '../lib/resource.svelte';
  import { href } from '../lib/router.svelte';

  const overview = new Resource(() => api.overview());
</script>

<h1>{t('shell.notifications')}</h1>

<Async resource={overview} what="notifications">
  {#snippet children(o)}
    {@const groups = groupNotifications(o.attention, o.month)}
    {#if groups.length === 0}
      <p class="muted">{t('shell.notifications.none')}</p>
    {:else}
      {#each groups as group (group.title)}
        <section class="section">
          <h2>{group.title}</h2>
          <ul class="list">
            {#each group.lines as { key, line } (key)}
              <li>
                <span><strong class="out">{line.topic}</strong> · {line.text}</span>
                <a href={href(line.path)}>{line.linkLabel}</a>
              </li>
            {/each}
          </ul>
        </section>
      {/each}
    {/if}
  {/snippet}
</Async>

<style>
  .list {
    list-style: none;
    margin: 0;
    padding: 0;
  }
  .list li {
    display: flex;
    flex-wrap: wrap;
    justify-content: space-between;
    gap: 0.3rem 1rem;
    padding: 0.6rem 0;
    border-bottom: 1px solid var(--rule);
  }
</style>
