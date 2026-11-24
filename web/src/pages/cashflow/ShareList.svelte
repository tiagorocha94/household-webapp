<script lang="ts">
  import CategoryLabel from '../../components/CategoryLabel.svelte';
  import Pill from '../../components/Pill.svelte';
  import { roleLabel, type CategoryLook, type categoryShares } from '../../lib/cashflow';
  import { categoryLabel, t } from '../../lib/i18n/index.svelte';
  import { formatCents, formatPercent } from '../../lib/money';

  let {
    shares,
    looks,
  }: { shares: ReturnType<typeof categoryShares>; looks: Map<string, CategoryLook> } = $props();
</script>

<ul class="shares">
  {#each shares as { category, sharePct } (category.categoryId)}
    <li>
      <span class="label">
        <CategoryLabel name={category.name} look={looks.get(category.name)} />
        {#if category.role}<Pill text={roleLabel(category.role)} tone="accent" />{/if}
      </span>
      <span
        class="bar"
        role="img"
        aria-label={t('cashflow.shareOfExpenses', {
          name: categoryLabel(category.name),
          percent: formatPercent(sharePct),
        })}
      >
        <span
          class="fill"
          style:width="{sharePct}%"
          style:background={looks.get(category.name)?.color}
        ></span>
      </span>
      <span class="num">{formatPercent(sharePct)}</span>
      <span class="num">{formatCents(category.totalCents)}</span>
    </li>
  {/each}
</ul>

<style>
  .shares {
    list-style: none;
    margin: 0;
    padding: 0;
    display: grid;
    gap: 0.6rem;
  }
  .shares li {
    display: grid;
    grid-template-columns: minmax(8rem, 14rem) 1fr 4rem 7rem;
    align-items: center;
    gap: 1rem;
  }
  .shares .label {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 0.4rem;
  }
  .shares .num {
    text-align: right;
  }
  .bar {
    display: block;
    height: 0.6rem;
    background: var(--rule);
    border-radius: 0.3rem;
    overflow: hidden;
  }
  .fill {
    display: block;
    height: 100%;
    background: var(--series-2);
  }
</style>
