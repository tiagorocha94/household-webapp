<script lang="ts">
  import type { CashflowMonth } from '../../api/client';
  import { api } from '../../api/client';
  import Async from '../../components/Async.svelte';
  import CashflowChart from '../../components/CashflowChart.svelte';
  import SortHeader from '../../components/SortHeader.svelte';
  import Stat from '../../components/Stat.svelte';
  import Stats from '../../components/Stats.svelte';
  import {
    chartMonths,
    formatChange,
    formatRateChange,
    formatSavingsRatePct,
    monthChange,
    netTone,
    newestFirst,
    yearsOf,
    yearTotals,
  } from '../../lib/cashflow';
  import { t } from '../../lib/i18n/index.svelte';
  import { formatCents } from '../../lib/money';
  import { formatMonth } from '../../lib/month';
  import { Resource } from '../../lib/resource.svelte';
  import { href } from '../../lib/router.svelte';
  import { sortRows, toggleSort, type Sort } from '../../lib/sort';

  const months = new Resource(() => api.cashflowMonths());

  type Column = 'month' | 'income' | 'expense' | 'net' | 'rate' | 'count';
  const values: Record<Column, (m: CashflowMonth) => string | number> = {
    month: (m) => m.month,
    income: (m) => m.incomeCents,
    expense: (m) => m.expenseCents,
    net: (m) => m.netCents,
    rate: (m) => m.savingsRatePct,
    count: (m) => m.movementCount,
  };
  let sort = $state<Sort<Column> | null>(null);
  const onsort = (column: Column) => (sort = toggleSort(sort, column));

  const yearRows = (list: CashflowMonth[]) =>
    yearsOf(list).map((year) => ({ year, ...yearTotals(list, year) }));

  type YearColumn = 'year' | 'income' | 'expense' | 'net' | 'rate';
  type YearRow = ReturnType<typeof yearRows>[number];
  const yearValues: Record<YearColumn, (y: YearRow) => string | number> = {
    year: (y) => y.year,
    income: (y) => y.incomeCents,
    expense: (y) => y.expenseCents,
    net: (y) => y.netCents,
    rate: (y) => y.savingsRatePct,
  };
  let yearSort = $state<Sort<YearColumn> | null>(null);
  const onyearsort = (column: YearColumn) => (yearSort = toggleSort(yearSort, column));
</script>

<h1>{t('cashflow.title')}</h1>

<Async resource={months} what="months">
  {#snippet children(list)}
    {#if list.length === 0}
      <p class="muted">{t('cashflow.noCashflow')}</p>
    {:else}
      {@const ordered = newestFirst(list)}
      {@const latest = ordered[0]}
      {@const change = monthChange(list, latest.month)}
      {@const versus = (delta: string) =>
        t('cashflow.vsMonth', { change: delta, month: formatMonth(change?.previous ?? '') })}
      <p class="muted">
        {t('cashflow.latestMonth')}
        <a href={href(`/cashflow/months/${latest.month}`)}>{formatMonth(latest.month)}</a>
      </p>
      <Stats>
        <Stat
          label={t('cashflow.income')}
          value={formatCents(latest.incomeCents)}
          note={change ? versus(formatChange(change.incomeCents)) : undefined}
        />
        <Stat
          label={t('cashflow.expenses')}
          value={formatCents(latest.expenseCents)}
          note={change ? versus(formatChange(change.expenseCents)) : undefined}
        />
        <Stat
          label={t('cashflow.leftOver')}
          value={formatCents(latest.netCents)}
          tone={netTone(latest.netCents)}
          note={change ? versus(formatChange(change.netCents)) : undefined}
        />
        <Stat
          label={t('cashflow.savingsRate')}
          value={formatSavingsRatePct(latest.savingsRatePct)}
          tone={netTone(latest.netCents)}
          note={change ? versus(formatRateChange(change.savingsRatePct)) : undefined}
        />
      </Stats>

      <section class="section">
        <h2>{t('cashflow.incomeAndExpenses')}</h2>
        <CashflowChart months={chartMonths(list)} />
      </section>

      <section class="section">
        <h2>{t('cashflow.byYear')}</h2>
        <table class="ledger">
          <thead>
            <tr>
              <SortHeader
                label={t('cashflow.year')}
                column="year"
                sort={yearSort}
                onsort={onyearsort}
              />
              <SortHeader
                label={t('cashflow.income')}
                column="income"
                sort={yearSort}
                onsort={onyearsort}
                numeric
              />
              <SortHeader
                label={t('cashflow.expenses')}
                column="expense"
                sort={yearSort}
                onsort={onyearsort}
                numeric
              />
              <SortHeader
                label={t('cashflow.leftOver')}
                column="net"
                sort={yearSort}
                onsort={onyearsort}
                numeric
              />
              <SortHeader
                label={t('cashflow.savingsRate')}
                column="rate"
                sort={yearSort}
                onsort={onyearsort}
                numeric
              />
            </tr>
          </thead>
          <tbody>
            {#each sortRows(yearRows(list), yearSort, yearValues) as y (y.year)}
              <tr>
                <td>{y.year}</td>
                <td class="num">{formatCents(y.incomeCents)}</td>
                <td class="num">{formatCents(y.expenseCents)}</td>
                <td class="num {netTone(y.netCents) ?? ''}">{formatCents(y.netCents)}</td>
                <td class="num {netTone(y.netCents) ?? ''}"
                  >{formatSavingsRatePct(y.savingsRatePct)}</td
                >
              </tr>
            {/each}
          </tbody>
        </table>
      </section>

      <section class="section">
        <h2>{t('cashflow.allMonths')}</h2>
        <table class="ledger">
          <thead>
            <tr>
              <SortHeader label={t('cashflow.month')} column="month" {sort} {onsort} />
              <SortHeader label={t('cashflow.income')} column="income" {sort} {onsort} numeric />
              <SortHeader label={t('cashflow.expenses')} column="expense" {sort} {onsort} numeric />
              <SortHeader label={t('cashflow.leftOver')} column="net" {sort} {onsort} numeric />
              <SortHeader label={t('cashflow.savingsRate')} column="rate" {sort} {onsort} numeric />
              <SortHeader label={t('cashflow.movements')} column="count" {sort} {onsort} numeric />
            </tr>
          </thead>
          <tbody>
            {#each sortRows(ordered, sort, values) as m (m.month)}
              <tr>
                <td><a href={href(`/cashflow/months/${m.month}`)}>{formatMonth(m.month)}</a></td>
                <td class="num">{formatCents(m.incomeCents)}</td>
                <td class="num">{formatCents(m.expenseCents)}</td>
                <td class="num {netTone(m.netCents) ?? ''}">{formatCents(m.netCents)}</td>
                <td class="num {netTone(m.netCents) ?? ''}"
                  >{formatSavingsRatePct(m.savingsRatePct)}</td
                >
                <td class="num">{m.movementCount}</td>
              </tr>
            {/each}
          </tbody>
        </table>
      </section>
    {/if}
  {/snippet}
</Async>
