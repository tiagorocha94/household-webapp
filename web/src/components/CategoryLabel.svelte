<script lang="ts">
  import { categoryLabel } from '../lib/i18n/index.svelte';
  import { iconColorOn } from '../lib/color';
  import { iconComponent } from '../lib/icons';
  import type { CategoryLook } from '../lib/cashflow';

  /** A category's name with its icon on its colour; without a look, just the name. */
  let { name, look }: { name: string; look?: CategoryLook } = $props();

  const Icon = $derived(look ? iconComponent(look.icon) : undefined);
</script>

<span class="label">
  {#if look && Icon}
    <span class="badge" style:background={look.color} aria-hidden="true">
      <Icon size={14} color={iconColorOn(look.color)} strokeWidth={2} />
    </span>
  {/if}
  <span>{categoryLabel(name)}</span>
</span>

<style>
  .label {
    display: inline-flex;
    align-items: center;
    gap: 0.5rem;
    min-width: 0;
  }
  .badge {
    display: inline-flex;
    flex: none;
    align-items: center;
    justify-content: center;
    width: 1.5rem;
    height: 1.5rem;
    border-radius: 50%;
    line-height: 0;
  }
</style>
