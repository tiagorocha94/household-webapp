<script lang="ts">
  import { api, type Payment } from '../../api/client';
  import Async from '../../components/Async.svelte';
  import Pill from '../../components/Pill.svelte';
  import ProgressBar from '../../components/ProgressBar.svelte';
  import SortHeader from '../../components/SortHeader.svelte';
  import Stat from '../../components/Stat.svelte';
  import Stats from '../../components/Stats.svelte';
  import { contractCategoryLabel, interestSharePct, paymentLabel } from '../../lib/credits';
  import { t } from '../../lib/i18n/index.svelte';
  import { formatCents, formatPercent } from '../../lib/money';
  import { formatMonth } from '../../lib/month';
  import { Resource } from '../../lib/resource.svelte';
  import { href } from '../../lib/router.svelte';
  import { sortRows, toggleSort, type Sort } from '../../lib/sort';

  let { id }: { id: number } = $props();

  const detail = new Resource(() => api.contract(id));
  const payments = new Resource(() => api.contractPayments(id));

  const pageSize = 24;
  let shown = $state(pageSize);

  type Column = 'month' | 'type' | 'description' | 'principal' | 'interest' | 'total';
  const values: Record<Column, (p: Payment) => string | number> = {
    month: (p) => p.month,
    type: (p) => paymentLabel(p.type),
    description: (p) => p.description,
    principal: (p) => p.principalCents,
    interest: (p) => p.interestCents,
    total: (p) => p.totalCents,
  };
  let sort = $state<Sort<Column> | null>(null);
  const onsort = (column: Column) => (sort = toggleSort(sort, column));
</script>

<p><a href={href('/credits')}>{t('goals.credits.all')}</a></p>

<Async resource={detail} what="credit">
  {#snippet children(c)}
    <h1>{c.name}</h1>
    <p class="muted">
      <Pill text={contractCategoryLabel(c.category)} tone="neutral" />
      <Pill
        text={c.status === 'active' ? t('goals.credits.active') : t('goals.credits.paidOff')}
        tone={c.status === 'active' ? 'accent' : 'in'}
      />
      {c.lender}{c.contractNumber
        ? ` · ${t('goals.credits.contractNumber', { number: c.contractNumber })}`
        : ''}
    </p>
    <Stats>
      <Stat
        label={t('goals.credits.outstanding')}
        value={formatCents(c.outstandingPrincipalCents)}
      />
      <Stat label={t('goals.credits.borrowed')} value={formatCents(c.originalPrincipalCents)} />
      <Stat label={t('goals.credits.paidSoFar')} value={formatCents(c.totalPaidCents)} />
      <Stat label={t('goals.credits.interestPaid')} value={formatCents(c.interestPaidCents)} />
    </Stats>
    <p class="repaid">
      <ProgressBar pct={c.progressPct} label={t('goals.credits.repaidOn', { name: c.name })} />
      {t('goals.credits.repaidSuffix')}
    </p>
    <p class="muted">
      {c.expectedEndMonth
        ? t('goals.credits.startedEnds', {
            start: formatMonth(c.startMonth),
            end: formatMonth(c.expectedEndMonth),
          })
        : t('goals.credits.started', { start: formatMonth(c.startMonth) })}
      {t('goals.credits.interestShare', { pct: formatPercent(interestSharePct(c)) })}
    </p>
    {#if c.notes}<p>{c.notes}</p>{/if}
  {/snippet}
</Async>

<section class="section">
  <h2>{t('goals.credits.payments')}</h2>
  <Async resource={payments}>
    {#snippet children(list)}
      {#if list.length === 0}
        <p class="muted">{t('goals.credits.noPayments')}</p>
      {:else}
        <table class="ledger">
          <thead>
            <tr>
              <SortHeader label={t('goals.month')} column="month" {sort} {onsort} />
              <SortHeader label={t('goals.type')} column="type" {sort} {onsort} />
              <SortHeader
                label={t('goals.credits.description')}
                column="description"
                {sort}
                {onsort}
              />
              <SortHeader
                label={t('goals.credits.principal')}
                column="principal"
                {sort}
                {onsort}
                numeric
              />
              <SortHeader label={t('goals.interest')} column="interest" {sort} {onsort} numeric />
              <SortHeader label={t('goals.credits.total')} column="total" {sort} {onsort} numeric />
            </tr>
          </thead>
          <tbody>
            {#each sortRows(list, sort, values).slice(0, shown) as p (p.id)}
              <tr>
                <td>{formatMonth(p.month)}</td>
                <td>{paymentLabel(p.type)}</td>
                <td class="muted">{p.description}</td>
                <td class="num">{formatCents(p.principalCents)}</td>
                <td class="num">{formatCents(p.interestCents)}</td>
                <td class="num">{formatCents(p.totalCents)}</td>
              </tr>
            {/each}
          </tbody>
        </table>
        {#if list.length > shown}
          <p>
            <button type="button" onclick={() => (shown += pageSize)}>
              {t('goals.credits.showMore', { count: list.length - shown })}
            </button>
          </p>
        {/if}
      {/if}
    {/snippet}
  </Async>
</section>
