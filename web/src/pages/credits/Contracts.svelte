<script lang="ts">
  import type { Contract } from '../../api/client';
  import { api } from '../../api/client';
  import Async from '../../components/Async.svelte';
  import Pill from '../../components/Pill.svelte';
  import ProgressBar from '../../components/ProgressBar.svelte';
  import SortHeader from '../../components/SortHeader.svelte';
  import Stat from '../../components/Stat.svelte';
  import Stats from '../../components/Stats.svelte';
  import { contractCategoryLabel, groupContracts, totals } from '../../lib/credits';
  import { t } from '../../lib/i18n/index.svelte';
  import { formatCents } from '../../lib/money';
  import { formatMonth } from '../../lib/month';
  import { Resource } from '../../lib/resource.svelte';
  import { href } from '../../lib/router.svelte';
  import { sortRows, toggleSort, type Sort } from '../../lib/sort';

  const contracts = new Resource(() => api.credits());

  type Column = 'name' | 'category' | 'lender' | 'borrowed' | 'outstanding' | 'repaid' | 'last';
  const values: Record<Column, (c: Contract) => string | number> = {
    name: (c) => c.name,
    category: (c) => contractCategoryLabel(c.category),
    lender: (c) => c.lender,
    borrowed: (c) => c.originalPrincipalCents,
    outstanding: (c) => c.outstandingPrincipalCents,
    repaid: (c) => c.progressPct,
    last: (c) => c.lastPaymentMonth ?? '',
  };
  let sort = $state<Sort<Column> | null>(null);
  const onsort = (column: Column) => (sort = toggleSort(sort, column));
</script>

{#snippet table(rows: Contract[])}
  <table class="ledger">
    <thead>
      <tr>
        <SortHeader label={t('goals.credits.contract')} column="name" {sort} {onsort} />
        <SortHeader label={t('goals.credits.categoryColumn')} column="category" {sort} {onsort} />
        <SortHeader label={t('goals.credits.lender')} column="lender" {sort} {onsort} />
        <SortHeader label={t('goals.credits.borrowed')} column="borrowed" {sort} {onsort} numeric />
        <SortHeader
          label={t('goals.credits.outstanding')}
          column="outstanding"
          {sort}
          {onsort}
          numeric
        />
        <SortHeader label={t('goals.credits.repaid')} column="repaid" {sort} {onsort} />
        <SortHeader label={t('goals.credits.lastPayment')} column="last" {sort} {onsort} />
      </tr>
    </thead>
    <tbody>
      {#each sortRows(rows, sort, values) as c (c.id)}
        <tr>
          <td><a href={href(`/credits/${c.id}`)}>{c.name}</a></td>
          <td><Pill text={contractCategoryLabel(c.category)} tone="neutral" /></td>
          <td class="muted">{c.lender}</td>
          <td class="num">{formatCents(c.originalPrincipalCents)}</td>
          <td class="num">{formatCents(c.outstandingPrincipalCents)}</td>
          <td
            ><ProgressBar
              pct={c.progressPct}
              label={t('goals.credits.repaidOn', { name: c.name })}
            /></td
          >
          <td class="muted">{c.lastPaymentMonth ? formatMonth(c.lastPaymentMonth) : '—'}</td>
        </tr>
      {/each}
    </tbody>
  </table>
{/snippet}

<h1>{t('goals.credits.title')}</h1>

<Async resource={contracts} what="credits">
  {#snippet children(list)}
    {#if list.length === 0}
      <p class="muted">{t('goals.credits.empty')}</p>
    {:else}
      {@const { active, paidOff } = groupContracts(list)}
      {@const sum = totals(list)}
      <Stats>
        <Stat
          label={t('goals.credits.outstanding')}
          value={formatCents(sum.outstandingCents)}
          note={t(
            sum.activeCount === 1
              ? 'goals.credits.activeContractsOne'
              : 'goals.credits.activeContractsOther',
            { count: sum.activeCount },
          )}
        />
        <Stat label={t('goals.credits.borrowed')} value={formatCents(sum.borrowedCents)} />
        <Stat label={t('goals.credits.paidSoFar')} value={formatCents(sum.paidCents)} />
        <Stat label={t('goals.credits.interestPaid')} value={formatCents(sum.interestPaidCents)} />
      </Stats>

      <section class="section">
        <h2>{t('goals.credits.active')}</h2>
        {#if active.length === 0}
          <p class="muted">{t('goals.credits.noActive')}</p>
        {:else}
          {@render table(active)}
        {/if}
      </section>
      {#if paidOff.length > 0}
        <section class="section">
          <h2>{t('goals.credits.paidOff')}</h2>
          {@render table(paidOff)}
        </section>
      {/if}
    {/if}
  {/snippet}
</Async>
