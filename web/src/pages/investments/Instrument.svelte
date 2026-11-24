<script lang="ts">
  import { api, type InvestmentContribution, type InvestmentSnapshot } from '../../api/client';
  import Async from '../../components/Async.svelte';
  import HistoryChart from '../../components/HistoryChart.svelte';
  import Pill from '../../components/Pill.svelte';
  import SortHeader from '../../components/SortHeader.svelte';
  import Stat from '../../components/Stat.svelte';
  import Stats from '../../components/Stats.svelte';
  import {
    buildHistory,
    formatGainPct,
    gainTone,
    ownerLabel,
    typeLabel,
  } from '../../lib/investments';
  import { t } from '../../lib/i18n/index.svelte';
  import { formatCents } from '../../lib/money';
  import { formatMonth } from '../../lib/month';
  import { Resource } from '../../lib/resource.svelte';
  import { href } from '../../lib/router.svelte';
  import { sortRows, toggleSort, type Sort } from '../../lib/sort';

  let { id }: { id: number } = $props();

  const detail = new Resource(() => api.instrument(id));
  const contributions = new Resource(() => api.instrumentContributions(id));
  const snapshots = new Resource(() => api.instrumentSnapshots(id));

  type Column = 'month' | 'description' | 'amount';
  const contributionValues: Record<Column, (c: InvestmentContribution) => string | number> = {
    month: (c) => c.month,
    description: (c) => c.description,
    amount: (c) => c.amountCents,
  };
  const snapshotValues: Record<Column, (s: InvestmentSnapshot) => string | number> = {
    month: (s) => s.month,
    description: (s) => s.description,
    amount: (s) => s.valueCents,
  };
  let contributionSort = $state<Sort<Column> | null>(null);
  let snapshotSort = $state<Sort<Column> | null>(null);
</script>

<Async resource={detail} what="instrument">
  {#snippet children(i)}
    <p><a href={href(`/investments/entities/${i.entityId}`)}>{i.entityName}</a></p>
    <h1>{i.name}</h1>
    <p class="muted">
      <Pill text={typeLabel[i.type]} tone="neutral" />
      {ownerLabel(i.user)}
    </p>
    <Stats>
      <Stat label={t('invest.moneyIn')} value={formatCents(i.allocatedCents)} />
      <Stat label={t('invest.currentValue')} value={formatCents(i.currentValueCents)} />
      <Stat
        label={t('invest.gain')}
        value={formatCents(i.gainCents)}
        tone={gainTone(i.gainCents)}
      />
      <Stat
        label={t('invest.return')}
        value={formatGainPct(i.gainPct)}
        tone={gainTone(i.gainCents)}
      />
    </Stats>
    {#if i.notes}<p>{i.notes}</p>{/if}
  {/snippet}
</Async>

<section class="section">
  <h2>{t('invest.history')}</h2>
  <Async resource={contributions}>
    {#snippet children(contributionList)}
      <Async resource={snapshots}>
        {#snippet children(snapshotList)}
          <HistoryChart history={buildHistory(contributionList, snapshotList)} />
        {/snippet}
      </Async>
    {/snippet}
  </Async>
</section>

<section class="section">
  <h2>{t('invest.contributions')}</h2>
  <Async resource={contributions}>
    {#snippet children(list)}
      {#if list.length === 0}
        <p class="muted">{t('invest.noContributions')}</p>
      {:else}
        <table class="ledger">
          <thead>
            <tr>
              <SortHeader
                label={t('invest.month')}
                column="month"
                sort={contributionSort}
                onsort={(c) => (contributionSort = toggleSort(contributionSort, c))}
              />
              <SortHeader
                label={t('invest.description')}
                column="description"
                sort={contributionSort}
                onsort={(c) => (contributionSort = toggleSort(contributionSort, c))}
              />
              <SortHeader
                label={t('invest.amount')}
                column="amount"
                sort={contributionSort}
                onsort={(c) => (contributionSort = toggleSort(contributionSort, c))}
                numeric
              />
            </tr>
          </thead>
          <tbody>
            {#each sortRows(list, contributionSort, contributionValues) as c (c.id)}
              <tr>
                <td>{formatMonth(c.month)}</td>
                <td class="muted">{c.description}</td>
                <td class="num">{formatCents(c.amountCents)}</td>
              </tr>
            {/each}
          </tbody>
        </table>
      {/if}
    {/snippet}
  </Async>
</section>

<section class="section">
  <h2>{t('invest.snapshots')}</h2>
  <Async resource={snapshots}>
    {#snippet children(list)}
      {#if list.length === 0}
        <p class="muted">{t('invest.noSnapshots')}</p>
      {:else}
        <table class="ledger">
          <thead>
            <tr>
              <SortHeader
                label={t('invest.month')}
                column="month"
                sort={snapshotSort}
                onsort={(c) => (snapshotSort = toggleSort(snapshotSort, c))}
              />
              <SortHeader
                label={t('invest.description')}
                column="description"
                sort={snapshotSort}
                onsort={(c) => (snapshotSort = toggleSort(snapshotSort, c))}
              />
              <SortHeader
                label={t('invest.value')}
                column="amount"
                sort={snapshotSort}
                onsort={(c) => (snapshotSort = toggleSort(snapshotSort, c))}
                numeric
              />
            </tr>
          </thead>
          <tbody>
            {#each sortRows(list, snapshotSort, snapshotValues) as s (s.id)}
              <tr>
                <td>{formatMonth(s.month)}</td>
                <td class="muted">{s.description}</td>
                <td class="num">{formatCents(s.valueCents)}</td>
              </tr>
            {/each}
          </tbody>
        </table>
      {/if}
    {/snippet}
  </Async>
</section>
