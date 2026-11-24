<script lang="ts">
  import type { CashflowMonthDetail, CashflowMovement } from '../../api/client';
  import { api } from '../../api/client';
  import Async from '../../components/Async.svelte';
  import CategoryLabel from '../../components/CategoryLabel.svelte';
  import ProgressBar from '../../components/ProgressBar.svelte';
  import Stat from '../../components/Stat.svelte';
  import Stats from '../../components/Stats.svelte';
  import {
    categoryLooks,
    categoryShares,
    formatSavingsRatePct,
    groupUse,
    movementSortValues,
    netTone,
    type MovementColumn,
    type MonthView,
  } from '../../lib/cashflow';
  import { groupLabel, t, type MessageKey } from '../../lib/i18n/index.svelte';
  import { formatCents, formatPercent } from '../../lib/money';
  import { formatMonth } from '../../lib/month';
  import { Resource } from '../../lib/resource.svelte';
  import { href } from '../../lib/router.svelte';
  import { sortRows, toggleSort, type Sort } from '../../lib/sort';
  import GroupBudgets from './GroupBudgets.svelte';
  import MovementTable from './MovementTable.svelte';
  import People from './People.svelte';
  import ShareList from './ShareList.svelte';

  let { month, view = 'summary' }: { month: string; view?: MonthView } = $props();

  const detail = new Resource(() => api.cashflowMonth(month));
  const movements = new Resource(() => api.cashflowMovements(month));
  const categories = new Resource(() => api.cashflowCategories());
  // The badges appear once the categories have loaded; the figures never wait for them.
  const looks = $derived(categoryLooks(categories.data ?? []));

  const tabs: { view: MonthView; label: MessageKey }[] = [
    { view: 'summary', label: 'cashflow.tab.summary' },
    { view: 'budget', label: 'cashflow.tab.budget' },
    { view: 'categories', label: 'cashflow.tab.categories' },
    { view: 'people', label: 'cashflow.tab.people' },
    { view: 'movements', label: 'cashflow.tab.movements' },
  ];
  /** How many rows the summary shows of each list. */
  const top = 5;

  const pageSize = 25;
  let shown = $state(pageSize);
  let sort = $state<Sort<MovementColumn> | null>(null);
  const onsort = (column: MovementColumn) => (sort = toggleSort(sort, column));
</script>

{#snippet budgetShort(d: CashflowMonthDetail)}
  {#if d.byGroup.length > 0}
    <section class="section">
      <div class="head">
        <h2>{t('cashflow.budgetByGroup')}</h2>
        <a href={href(`/cashflow/months/${month}/budget`)}>{t('cashflow.details')}</a>
      </div>
      {#if d.incomeCents === 0}
        <p class="muted">{t('cashflow.noIncomeNoBudget')}</p>
      {/if}
      <ul class="rows">
        {#each d.byGroup as g (g.name)}
          {@const use = groupUse(g)}
          <li>
            <div class="line">
              <strong>{groupLabel(g.name)}</strong>
              <span>
                <strong>{formatCents(g.spentCents)}</strong>
                {#if d.incomeCents > 0}
                  <span class="muted"
                    >{t('cashflow.ofBudget', { amount: formatCents(g.budgetCents) })}</span
                  >
                {/if}
              </span>
            </div>
            {#if d.incomeCents > 0}
              <ProgressBar
                pct={use.usedPct}
                over={use.over}
                label={t('cashflow.groupBudgetSpent', {
                  name: groupLabel(g.name),
                  percent: formatPercent(use.usedPct),
                })}
              />
            {/if}
          </li>
        {/each}
      </ul>
    </section>
  {/if}
{/snippet}

{#snippet moneyShort(d: CashflowMonthDetail)}
  {@const shares = categoryShares(d.byCategory, 'expense')}
  <section class="section">
    <div class="head">
      <h2>{t('cashflow.whereMoneyWent')}</h2>
      {#if shares.length > top}<a href={href(`/cashflow/months/${month}/categories`)}
          >{t('cashflow.allCategories')}</a
        >{/if}
    </div>
    {#if shares.length === 0}
      <p class="muted">{t('cashflow.noExpenses')}</p>
    {:else}
      <ShareList shares={shares.slice(0, top)} {looks} />
    {/if}
  </section>
{/snippet}

{#snippet movementsShort(list: CashflowMovement[])}
  <section class="section">
    <div class="head">
      <h2>{t('cashflow.latestMovements')}</h2>
      {#if list.length > top}
        <a href={href(`/cashflow/months/${month}/movements`)}
          >{t('cashflow.allMovements', { count: list.length })}</a
        >
      {/if}
    </div>
    {#if list.length === 0}
      <p class="muted">{t('cashflow.noMovements')}</p>
    {:else}
      <MovementTable rows={list.slice(0, top)} {looks} {sort} {onsort} />
    {/if}
  </section>
{/snippet}

<p><a href={href('/cashflow')}>{t('cashflow.allMonths')}</a></p>
<h1>{formatMonth(month)}</h1>
<nav class="tabs" aria-label={t('cashflow.tabsLabel')}>
  {#each tabs as tab (tab.view)}
    <a
      href={href(
        tab.view === 'summary'
          ? `/cashflow/months/${month}`
          : `/cashflow/months/${month}/${tab.view}`,
      )}
      aria-current={view === tab.view ? 'page' : undefined}>{t(tab.label)}</a
    >
  {/each}
</nav>

<Async resource={detail} what="month">
  {#snippet children(d)}
    <Stats>
      <Stat label={t('cashflow.income')} value={formatCents(d.incomeCents)} />
      <Stat
        label={t('cashflow.expenses')}
        value={formatCents(d.expenseCents)}
        note={d.savedCents > 0
          ? t('cashflow.ofWhichSaved', { amount: formatCents(d.savedCents) })
          : undefined}
      />
      <Stat
        label={t('cashflow.leftOver')}
        value={formatCents(d.netCents)}
        tone={netTone(d.netCents)}
      />
      <Stat
        label={t('cashflow.savingsRate')}
        value={formatSavingsRatePct(d.savingsRatePct)}
        tone={netTone(d.netCents)}
      />
    </Stats>

    {#if view === 'summary'}
      {@render budgetShort(d)}
      {@render moneyShort(d)}
    {:else if view === 'budget'}
      {#if d.byGroup.length > 0}
        <GroupBudgets {d} {looks} />
      {/if}
    {:else if view === 'categories'}
      <section class="section">
        <h2>{t('cashflow.whereMoneyWent')}</h2>
        {#if categoryShares(d.byCategory, 'expense').length === 0}
          <p class="muted">{t('cashflow.noExpenses')}</p>
        {:else}
          <ShareList shares={categoryShares(d.byCategory, 'expense')} {looks} />
        {/if}
      </section>

      {@const income = categoryShares(d.byCategory, 'income')}
      {#if income.length > 0}
        <section class="section">
          <h2>{t('cashflow.whereItCameFrom')}</h2>
          <table class="ledger">
            <thead>
              <tr>
                <th>{t('cashflow.category')}</th>
                <th class="num">{t('cashflow.movements')}</th>
                <th class="num">{t('cashflow.amount')}</th>
              </tr>
            </thead>
            <tbody>
              {#each income as { category } (category.categoryId)}
                <tr>
                  <td><CategoryLabel name={category.name} look={looks.get(category.name)} /></td>
                  <td class="num">{category.movementCount}</td>
                  <td class="num">{formatCents(category.totalCents)}</td>
                </tr>
              {/each}
            </tbody>
          </table>
        </section>
      {/if}
    {:else if view === 'people'}
      <People {d} {month} />
    {/if}
  {/snippet}
</Async>

{#if view === 'summary'}
  <Async resource={movements}>
    {#snippet children(list)}
      {@render movementsShort(list)}
    {/snippet}
  </Async>
{:else if view === 'movements'}
  <section class="section">
    <h2>{t('cashflow.movements')}</h2>
    <Async resource={movements}>
      {#snippet children(list)}
        {#if list.length === 0}
          <p class="muted">{t('cashflow.noMovements')}</p>
        {:else}
          {@const ordered = sortRows(list, sort, movementSortValues)}
          <MovementTable rows={ordered.slice(0, shown)} {looks} {sort} {onsort} />
          {#if ordered.length > shown}
            <p>
              <button type="button" onclick={() => (shown += pageSize)}>
                {t('cashflow.showMore', { count: ordered.length - shown })}
              </button>
            </p>
          {/if}
        {/if}
      {/snippet}
    </Async>
  </section>
{/if}

<style>
  .tabs {
    display: flex;
    flex-wrap: wrap;
    gap: 0.25rem 1.5rem;
    border-bottom: 1px solid var(--rule);
    margin-bottom: 1.5rem;
  }
  .tabs a {
    padding: 0.6rem 0;
    min-height: 1.5rem;
    color: var(--ink);
    text-decoration: none;
    border-bottom: 2px solid transparent;
  }
  .tabs a[aria-current='page'] {
    color: var(--accent);
    font-weight: 600;
    border-bottom-color: var(--accent);
  }
  .head {
    display: flex;
    justify-content: space-between;
    align-items: baseline;
    gap: 1rem;
  }
  .rows {
    list-style: none;
    margin: 0;
    padding: 0;
  }
  .rows li {
    padding: 0.6rem 0;
    border-bottom: 1px solid var(--rule);
  }
  .rows .line {
    display: flex;
    justify-content: space-between;
    gap: 1rem;
  }
</style>
