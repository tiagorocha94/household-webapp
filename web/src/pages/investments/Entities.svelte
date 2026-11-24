<script lang="ts">
  import type { InvestmentEntity } from '../../api/client';
  import { api } from '../../api/client';
  import Async from '../../components/Async.svelte';
  import Pill from '../../components/Pill.svelte';
  import SortHeader from '../../components/SortHeader.svelte';
  import Stat from '../../components/Stat.svelte';
  import Stats from '../../components/Stats.svelte';
  import {
    allocationShares,
    formatGainPct,
    gainTone,
    kindLabel,
    totals,
  } from '../../lib/investments';
  import { t } from '../../lib/i18n/index.svelte';
  import { formatCents, formatPercent } from '../../lib/money';
  import { Resource } from '../../lib/resource.svelte';
  import { href } from '../../lib/router.svelte';
  import { sortRows, toggleSort, type Sort } from '../../lib/sort';

  const entities = new Resource(() => api.investments());

  type Column = 'name' | 'kind' | 'instruments' | 'allocated' | 'value' | 'gain';
  const values: Record<Column, (e: InvestmentEntity) => string | number> = {
    name: (e) => e.name,
    kind: (e) => kindLabel[e.kind],
    instruments: (e) => e.instrumentCount,
    allocated: (e) => e.allocatedCents,
    value: (e) => e.currentValueCents,
    gain: (e) => e.gainPct,
  };
  let sort = $state<Sort<Column> | null>(null);
  const onsort = (column: Column) => (sort = toggleSort(sort, column));
</script>

<h1>{t('invest.title')}</h1>

<Async resource={entities} what="investments">
  {#snippet children(list)}
    {#if list.length === 0}
      <p class="muted">{t('invest.empty')}</p>
    {:else}
      {@const sum = totals(list)}
      <Stats>
        <Stat label={t('invest.moneyIn')} value={formatCents(sum.allocatedCents)} />
        <Stat label={t('invest.currentValue')} value={formatCents(sum.currentValueCents)} />
        <Stat
          label={t('invest.gain')}
          value={formatCents(sum.gainCents)}
          tone={gainTone(sum.gainCents)}
        />
        <Stat
          label={t('invest.return')}
          value={formatGainPct(sum.gainPct)}
          tone={gainTone(sum.gainCents)}
        />
      </Stats>

      <section class="section">
        <h2>{t('invest.whereMoney')}</h2>
        <ul class="shares">
          {#each allocationShares(list) as { entity, sharePct } (entity.id)}
            <li>
              <a href={href(`/investments/entities/${entity.id}`)}>{entity.name}</a>
              <span
                class="bar"
                role="img"
                aria-label={t('invest.shareAria', {
                  name: entity.name,
                  percent: formatPercent(sharePct),
                })}
              >
                <span class="fill" style:width="{sharePct}%"></span>
              </span>
              <span class="num">{formatPercent(sharePct)}</span>
            </li>
          {/each}
        </ul>
      </section>

      <section class="section">
        <h2>{t('invest.entities')}</h2>
        <table class="ledger">
          <thead>
            <tr>
              <SortHeader label={t('invest.entity')} column="name" {sort} {onsort} />
              <SortHeader label={t('invest.kind')} column="kind" {sort} {onsort} />
              <SortHeader
                label={t('invest.instruments')}
                column="instruments"
                {sort}
                {onsort}
                numeric
              />
              <SortHeader label={t('invest.moneyIn')} column="allocated" {sort} {onsort} numeric />
              <SortHeader label={t('invest.value')} column="value" {sort} {onsort} numeric />
              <SortHeader label={t('invest.return')} column="gain" {sort} {onsort} numeric />
            </tr>
          </thead>
          <tbody>
            {#each sortRows(list, sort, values) as e (e.id)}
              <tr>
                <td><a href={href(`/investments/entities/${e.id}`)}>{e.name}</a></td>
                <td><Pill text={kindLabel[e.kind]} tone="neutral" /></td>
                <td class="num">{e.instrumentCount}</td>
                <td class="num">{formatCents(e.allocatedCents)}</td>
                <td class="num">{formatCents(e.currentValueCents)}</td>
                <td class="num {gainTone(e.gainCents) ?? ''}">{formatGainPct(e.gainPct)}</td>
              </tr>
            {/each}
          </tbody>
        </table>
      </section>
    {/if}
  {/snippet}
</Async>

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
    grid-template-columns: minmax(8rem, 14rem) 1fr 4rem;
    align-items: center;
    gap: 1rem;
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
    background: var(--series-1);
  }
</style>
