<script lang="ts">
  import type { Snippet } from 'svelte';

  let { children }: { children: Snippet } = $props();
</script>

<dl class="stats">{@render children()}</dl>

<style>
  .stats {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(9.5rem, 1fr));
    gap: 1.2rem 2rem;
    margin: 1.2rem 0 0;
    padding: 1rem 0;
    border-top: 1px solid var(--ink);
    border-bottom: 1px solid var(--rule);
  }
</style>
