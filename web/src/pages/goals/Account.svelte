<script lang="ts">
  import { api, type Goal } from '../../api/client';
  import Async from '../../components/Async.svelte';
  import MovementsTable from '../../components/MovementsTable.svelte';
  import ProjectionChart from '../../components/ProjectionChart.svelte';
  import Pill from '../../components/Pill.svelte';
  import SortHeader from '../../components/SortHeader.svelte';
  import ProgressBar from '../../components/ProgressBar.svelte';
  import Stat from '../../components/Stat.svelte';
  import Stats from '../../components/Stats.svelte';
  import {
    countByPhase,
    filterGoals,
    balanceTone,
    goalPhase,
    phaseLabel,
    shownBalance,
    sortGoals,
    type PhaseFilter,
  } from '../../lib/goals';
  import { localeTag, t } from '../../lib/i18n/index.svelte';
  import { formatCents } from '../../lib/money';
  import { currentMonth, formatMonth } from '../../lib/month';
  import { Resource } from '../../lib/resource.svelte';
  import { href } from '../../lib/router.svelte';
  import { sortRows, toggleSort, type Sort } from '../../lib/sort';

  let { id }: { id: number } = $props();

  const detail = new Resource(() => api.account(id));
  const goals = new Resource(() => api.accountGoals(id));
  const movements = new Resource(() => api.accountMovements(id));
  const projection = new Resource(() => api.projection(id));

  const now = currentMonth();
  let filter = $state<PhaseFilter>('active');

  type Column = 'name' | 'state' | 'period' | 'target' | 'monthly' | 'balance' | 'progress';
  const values: Record<Column, (g: Goal) => string | number> = {
    name: (g) => g.name,
    state: (g) => phaseLabel(goalPhase(g, now)),
    period: (g) => g.startMonth,
    target: (g) => g.targetCents,
    monthly: (g) => g.monthlyAllocationCents,
    balance: (g) => shownBalance(g),
    progress: (g) => g.progressPct,
  };
  let sort = $state<Sort<Column> | null>(null);
  const onsort = (column: Column) => (sort = toggleSort(sort, column));
</script>

<p><a href={href('/goals')}>{t('goals.account.all')}</a></p>

<Async resource={detail} what="goals">
  {#snippet children({ account, balance })}
    <h1>{account.name}</h1>
    <p class="muted">
      {t('goals.account.assumes', { amount: formatCents(account.monthlyDepositCents) })}
    </p>
    <Stats>
      <Stat label={t('goals.balance')} value={formatCents(balance.balanceCents)} />
      <Stat
        label={t('goals.free')}
        value={formatCents(balance.unallocatedCents)}
        tone={balance.unallocatedCents < 0 ? 'out' : undefined}
      />
      <Stat label={t('goals.deposited')} value={formatCents(balance.depositsCents)} />
      <Stat label={t('goals.spent')} value={formatCents(balance.expensesCents)} />
    </Stats>
  {/snippet}
</Async>

<section class="section">
  <h2>{t('goals.account.goals')}</h2>
  <Async resource={goals}>
    {#snippet children(list)}
      {#if list.length === 0}
        <p class="muted">{t('goals.account.noGoals')}</p>
      {:else}
        {@const counts = countByPhase(list, now)}
        {@const rows = sortGoals(filterGoals(list, filter, now), now)}
        <div class="filters" role="group" aria-label={t('goals.account.filterByState')}>
          {#each ['active', 'planned', 'completed', 'cancelled', 'all'] as const as option}
            {#if option === 'all' || counts[option] > 0}
              <button
                type="button"
                aria-pressed={filter === option}
                onclick={() => (filter = option)}
              >
                {phaseLabel(option)} ({counts[option]})
              </button>
            {/if}
          {/each}
        </div>
        {#if rows.length === 0}
          <p class="muted">
            {t('goals.account.noneInState', {
              state: phaseLabel(filter).toLocaleLowerCase(localeTag()),
            })}
          </p>
        {:else}
          <table class="ledger">
            <thead>
              <tr>
                <SortHeader label={t('goals.account.goal')} column="name" {sort} {onsort} />
                <SortHeader label={t('goals.account.state')} column="state" {sort} {onsort} />
                <SortHeader label={t('goals.account.period')} column="period" {sort} {onsort} />
                <SortHeader label={t('goals.target')} column="target" {sort} {onsort} numeric />
                <SortHeader label={t('goals.perMonth')} column="monthly" {sort} {onsort} numeric />
                <SortHeader label={t('goals.balance')} column="balance" {sort} {onsort} numeric />
                <SortHeader label={t('goals.account.progress')} column="progress" {sort} {onsort} />
              </tr>
            </thead>
            <tbody>
              {#each sortRows(rows, sort, values) as goal (goal.id)}
                {@const phase = goalPhase(goal, now)}
                <tr>
                  <td><a href={href(`/goals/${goal.id}`)}>{goal.name}</a></td>
                  <td>
                    <Pill
                      text={phaseLabel(phase)}
                      tone={phase === 'active'
                        ? 'accent'
                        : phase === 'completed'
                          ? 'in'
                          : 'neutral'}
                    />
                  </td>
                  <td class="muted"
                    >{t('goals.period', {
                      from: formatMonth(goal.startMonth),
                      to: formatMonth(goal.endMonth),
                    })}</td
                  >
                  <td class="num">{formatCents(goal.targetCents)}</td>
                  <td class="num">{formatCents(goal.monthlyAllocationCents)}</td>
                  <td class="num {balanceTone(shownBalance(goal)) ?? ''}">
                    {formatCents(shownBalance(goal))}
                  </td>
                  <td
                    ><ProgressBar
                      pct={goal.progressPct}
                      label={t('goals.progressOf', { name: goal.name })}
                    /></td
                  >
                </tr>
              {/each}
            </tbody>
          </table>
        {/if}
      {/if}
    {/snippet}
  </Async>
</section>

<section class="section">
  <h2>{t('goals.account.projection')}</h2>
  <Async resource={projection}>
    {#snippet children(data)}
      <ProjectionChart projection={data} />
    {/snippet}
  </Async>
</section>

<section class="section">
  <h2>{t('goals.movements')}</h2>
  <Async resource={movements}>
    {#snippet children(list)}
      <MovementsTable movements={list} view="account" />
    {/snippet}
  </Async>
</section>
