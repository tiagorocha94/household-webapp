<script lang="ts">
  import { api } from '../api/client';
  import Async from '../components/Async.svelte';
  import NetWorthChart from '../components/NetWorthChart.svelte';
  import Stat from '../components/Stat.svelte';
  import Stats from '../components/Stats.svelte';
  import { formatChange } from '../lib/cashflow';
  import { t } from '../lib/i18n/index.svelte';
  import { isEnabled } from '../lib/modules';
  import { serverModules } from '../lib/modules.svelte';
  import { formatCents } from '../lib/money';
  import { formatMonth } from '../lib/month';
  import { changeOver, inRange, rangeLabel, ranges, type Range } from '../lib/networth';
  import { Resource } from '../lib/resource.svelte';

  const history = new Resource(() => api.netWorth());
  let range = $state<Range>(12);
  // The figures of a module that does not run are always zero, so they are left out.
  const runs = (module: Parameters<typeof isEnabled>[1]) =>
    isEnabled(serverModules.enabled, module);
</script>

<h1>{t('invest.networth.title')}</h1>

<Async resource={history} what="net worth">
  {#snippet children(months)}
    {#if months.length === 0}
      <p class="muted">{t('invest.networth.empty')}</p>
    {:else}
      {@const now = months[months.length - 1]}
      {@const change = changeOver(months)}
      <p class="muted">{t('invest.networth.intro')}</p>

      <Stats>
        <Stat
          label={t('invest.networth.netWorthMonth', { month: formatMonth(now.month) })}
          value={formatCents(now.netCents)}
          note={change
            ? t('invest.networth.since', {
                change: formatChange(change.netCents),
                month: formatMonth(change.since),
              })
            : undefined}
        />
        <Stat label={t('invest.networth.assets')} value={formatCents(now.assetsCents)} />
        {#if runs('credits')}
          <Stat
            label={t('invest.networth.owedOnCredits')}
            value={formatCents(now.creditCents)}
            note={change
              ? t('invest.networth.since', {
                  change: formatChange(change.creditCents),
                  month: formatMonth(change.since),
                })
              : undefined}
          />
        {/if}
        <Stat label={t('invest.networth.cash')} value={formatCents(now.cashCents)} />
      </Stats>

      <section class="section">
        <div class="filters" role="group" aria-label={t('invest.networth.range')}>
          {#each ranges as r (r)}
            <button type="button" aria-pressed={range === r} onclick={() => (range = r)}>
              {rangeLabel(r)}
            </button>
          {/each}
        </div>
        <NetWorthChart months={inRange(months, range)} />
      </section>

      <section class="section">
        <h2>{t('invest.networth.byMonth')}</h2>
        <div style="overflow-x: auto">
          <table class="ledger">
            <thead>
              <tr>
                <th>{t('invest.networth.month')}</th>
                {#if runs('savings')}
                  <th class="num">{t('invest.networth.savings')}</th>
                {/if}
                {#if runs('investments')}
                  <th class="num">{t('invest.networth.investments')}</th>
                {/if}
                {#if runs('goals')}
                  <th class="num">{t('invest.networth.goals')}</th>
                {/if}
                <th class="num">{t('invest.networth.cash')}</th>
                <th class="num">{t('invest.networth.assets')}</th>
                {#if runs('credits')}
                  <th class="num">{t('invest.networth.owed')}</th>
                {/if}
                <th class="num">{t('invest.networth.title')}</th>
              </tr>
            </thead>
            <tbody>
              {#each months.slice(-6).reverse() as m (m.month)}
                <tr>
                  <td>{formatMonth(m.month)}</td>
                  {#if runs('savings')}
                    <td class="num">{formatCents(m.savingsCents)}</td>
                  {/if}
                  {#if runs('investments')}
                    <td class="num">{formatCents(m.investmentsCents)}</td>
                  {/if}
                  {#if runs('goals')}
                    <td class="num">{formatCents(m.goalsCents)}</td>
                  {/if}
                  <td class="num">{formatCents(m.cashCents)}</td>
                  <td class="num">{formatCents(m.assetsCents)}</td>
                  {#if runs('credits')}
                    <td class="num">{formatCents(m.creditCents)}</td>
                  {/if}
                  <td class="num"><strong>{formatCents(m.netCents)}</strong></td>
                </tr>
              {/each}
            </tbody>
          </table>
        </div>
        <p class="muted">{t('invest.networth.note')}</p>
      </section>
    {/if}
  {/snippet}
</Async>

<style>
  .ledger th,
  .ledger td {
    white-space: nowrap;
  }
  .filters {
    display: flex;
    gap: 0.4rem;
    margin-bottom: 0.8rem;
  }
  .filters button {
    font: inherit;
    padding: 0.2rem 0.8rem;
    color: var(--ink);
    background: var(--bg);
    border: 1px solid var(--rule);
    border-radius: 4px;
    cursor: pointer;
  }
  .filters button[aria-pressed='true'] {
    color: var(--bg);
    background: var(--ink);
    border-color: var(--ink);
  }
</style>
