<script lang="ts">
  import type { InvestmentInstrument } from '../../api/client';
  import { api } from '../../api/client';
  import Async from '../../components/Async.svelte';
  import Pill from '../../components/Pill.svelte';
  import SortHeader from '../../components/SortHeader.svelte';
  import Stat from '../../components/Stat.svelte';
  import Stats from '../../components/Stats.svelte';
  import {
    formatGainPct,
    gainTone,
    kindLabel,
    sortInstruments,
    typeLabel,
    ownerLabel,
  } from '../../lib/investments';
  import { t } from '../../lib/i18n/index.svelte';
  import { formatCents } from '../../lib/money';
  import { formatMonth } from '../../lib/month';
  import { Resource } from '../../lib/resource.svelte';
  import { href } from '../../lib/router.svelte';
  import { sortRows, toggleSort, type Sort } from '../../lib/sort';

  let { id }: { id: number } = $props();

  const detail = new Resource(() => api.entity(id));
  const instruments = new Resource(() => api.entityInstruments(id));

  type Column = 'name' | 'type' | 'owner' | 'allocated' | 'value' | 'gain' | 'last';
  const values: Record<Column, (i: InvestmentInstrument) => string | number> = {
    name: (i) => i.name,
    type: (i) => typeLabel[i.type],
    owner: (i) => ownerLabel(i.user),
    allocated: (i) => i.allocatedCents,
    value: (i) => i.currentValueCents,
    gain: (i) => i.gainPct,
    last: (i) => i.lastSnapshotMonth ?? '',
  };
  let sort = $state<Sort<Column> | null>(null);
  const onsort = (column: Column) => (sort = toggleSort(sort, column));
</script>

<p><a href={href('/investments')}>{t('invest.allInvestments')}</a></p>

<Async resource={detail} what="investment account">
  {#snippet children(e)}
    <h1>{e.name}</h1>
    <p class="muted"><Pill text={kindLabel[e.kind]} tone="neutral" /></p>
    <Stats>
      <Stat label={t('invest.moneyIn')} value={formatCents(e.allocatedCents)} />
      <Stat label={t('invest.currentValue')} value={formatCents(e.currentValueCents)} />
      <Stat
        label={t('invest.gain')}
        value={formatCents(e.gainCents)}
        tone={gainTone(e.gainCents)}
      />
      <Stat
        label={t('invest.return')}
        value={formatGainPct(e.gainPct)}
        tone={gainTone(e.gainCents)}
      />
    </Stats>
  {/snippet}
</Async>

<section class="section">
  <h2>{t('invest.instruments')}</h2>
  <Async resource={instruments}>
    {#snippet children(list)}
      {#if list.length === 0}
        <p class="muted">{t('invest.noInstruments')}</p>
      {:else}
        <table class="ledger">
          <thead>
            <tr>
              <SortHeader label={t('invest.instrument')} column="name" {sort} {onsort} />
              <SortHeader label={t('invest.type')} column="type" {sort} {onsort} />
              <SortHeader label={t('invest.owner')} column="owner" {sort} {onsort} />
              <SortHeader label={t('invest.moneyIn')} column="allocated" {sort} {onsort} numeric />
              <SortHeader label={t('invest.value')} column="value" {sort} {onsort} numeric />
              <SortHeader label={t('invest.return')} column="gain" {sort} {onsort} numeric />
              <SortHeader label={t('invest.lastValue')} column="last" {sort} {onsort} />
            </tr>
          </thead>
          <tbody>
            {#each sortRows(sortInstruments(list), sort, values) as i (i.id)}
              <tr>
                <td><a href={href(`/investments/instruments/${i.id}`)}>{i.name}</a></td>
                <td><Pill text={typeLabel[i.type]} tone="neutral" /></td>
                <td class="muted">{ownerLabel(i.user)}</td>
                <td class="num">{formatCents(i.allocatedCents)}</td>
                <td class="num">{formatCents(i.currentValueCents)}</td>
                <td class="num {gainTone(i.gainCents) ?? ''}">{formatGainPct(i.gainPct)}</td>
                <td class="muted">{i.lastSnapshotMonth ? formatMonth(i.lastSnapshotMonth) : '—'}</td
                >
              </tr>
            {/each}
          </tbody>
        </table>
      {/if}
    {/snippet}
  </Async>
</section>
