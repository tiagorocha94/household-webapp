<script lang="ts">
  import { api, type OverviewContract, type OverviewGoal } from '../api/client';
  import Async from '../components/Async.svelte';
  import Pill from '../components/Pill.svelte';
  import ProgressBar from '../components/ProgressBar.svelte';
  import SortHeader from '../components/SortHeader.svelte';
  import Stat from '../components/Stat.svelte';
  import Stats from '../components/Stats.svelte';
  import { formatSavingsRatePct, netTone } from '../lib/cashflow';
  import { groupLabel, t } from '../lib/i18n/index.svelte';
  import { formatCents, formatPercent } from '../lib/money';
  import { formatMonth } from '../lib/month';
  import { reviewBreakdown, reviewHeadline } from '../lib/notifications';
  import {
    expenseNote,
    isEmpty,
    nextInstalment,
    savedNote,
    savingsAccountNote,
    savingsNote,
  } from '../lib/overview';
  import { Resource } from '../lib/resource.svelte';
  import { href } from '../lib/router.svelte';
  import { sortRows, toggleSort, type Sort } from '../lib/sort';

  // One request: the server composes every figure on this page.
  const overview = new Resource(() => api.overview());

  type GoalColumn = 'name' | 'ends' | 'target' | 'saved' | 'progress';
  const goalValues: Record<GoalColumn, (g: OverviewGoal) => string | number> = {
    name: (g) => g.name,
    ends: (g) => g.endMonth,
    target: (g) => g.targetCents,
    saved: (g) => g.savedCents,
    progress: (g) => g.progressPct,
  };
  let goalSort = $state<Sort<GoalColumn> | null>(null);
  const onGoalSort = (column: GoalColumn) => (goalSort = toggleSort(goalSort, column));

  type ContractColumn = 'name' | 'outstanding' | 'instalment' | 'next';
  const contractValues: Record<ContractColumn, (c: OverviewContract) => string | number> = {
    name: (c) => c.name,
    outstanding: (c) => c.outstandingCents,
    instalment: (c) => c.instalmentCents,
    next: (c) => c.nextInstalmentMonth ?? '',
  };
  let contractSort = $state<Sort<ContractColumn> | null>(null);
  const onContractSort = (column: ContractColumn) =>
    (contractSort = toggleSort(contractSort, column));
</script>

<h1>{t('shell.nav.overview')}</h1>

<Async resource={overview} what="overview">
  {#snippet children(o)}
    {#if isEmpty(o)}
      <p class="muted">
        {t('shell.home.empty')}
      </p>
    {:else}
      {@const headline = reviewHeadline(o.attention)}
      {#if headline}
        <p class="review">
          <span>
            <strong class="out">{headline}</strong>
            <span class="muted small">{reviewBreakdown(o.attention)}</span>
          </span>
          <a href={href('/notifications')}>{t('shell.home.seeNotifications')}</a>
        </p>
      {/if}

      <Stats>
        <Stat
          label={t('shell.nav.netWorth')}
          value={formatCents(o.netWorth.netCents)}
          note={t('shell.home.netWorthNote', {
            assets: formatCents(o.netWorth.assetsCents),
            credit: formatCents(o.netWorth.creditCents),
          })}
        />
        <Stat
          label={t('shell.home.balanceInGoals')}
          value={formatCents(o.goals.savedCents)}
          note={t('shell.home.ofTargeted', { amount: formatCents(o.goals.targetCents) })}
        />
        <Stat
          label={t('shell.home.creditOutstanding')}
          value={formatCents(o.credit.outstandingCents)}
          note={t('shell.home.dueEachMonth', {
            amount: formatCents(o.credit.monthlyInstalmentCents),
          })}
        />
        <Stat
          label={t('shell.home.invested')}
          value={formatCents(o.investments.currentValueCents)}
        />
        {#if o.savings.accounts > 0}
          <Stat
            label={t('shell.nav.savings')}
            value={formatCents(o.savings.balanceCents)}
            note={savingsNote(o.savings) ?? undefined}
          />
        {/if}
      </Stats>

      <div class="columns">
        <section class="section" aria-labelledby="cashflow">
          <h2 id="cashflow">{t('shell.home.thisMonth')}</h2>
          {#if o.cashflow.incomeCents === 0 && o.cashflow.expenseCents === 0}
            <p class="muted">{t('shell.home.noMovements')}</p>
          {:else}
            <Stats>
              <Stat
                label={t('shell.home.income')}
                value={formatCents(o.cashflow.incomeCents)}
                tone="in"
              />
              <Stat
                label={t('shell.home.expenses')}
                value={formatCents(o.cashflow.expenseCents)}
                tone="out"
                note={[savedNote(o.cashflow), expenseNote(o.cashflow)]
                  .filter(Boolean)
                  .join(' · ') || undefined}
              />
              <Stat
                label={t('shell.home.saved')}
                value={formatCents(o.cashflow.netCents)}
                tone={netTone(o.cashflow.netCents)}
                note={t('shell.home.ofIncome', {
                  pct: formatSavingsRatePct(o.cashflow.savingsRatePct),
                })}
              />
            </Stats>
          {/if}

          {#if o.cashflow.groups.length > 0}
            <h3>{t('shell.home.budgetGroups')}</h3>
            <ul class="groups">
              {#each o.cashflow.groups as g (g.name)}
                <li>
                  <div class="line">
                    <span>{groupLabel(g.name)}</span>
                    <span class="muted small">
                      {t('shell.home.spentOfBudget', {
                        spent: formatCents(g.spentCents),
                        budget: formatCents(g.budgetCents),
                      })}
                    </span>
                  </div>
                  {#if g.budgetCents > 0}
                    <ProgressBar
                      pct={g.usedPct}
                      over={g.usedPct > 100}
                      label={t('shell.home.budgetSpentLabel', {
                        name: groupLabel(g.name),
                        pct: formatPercent(g.usedPct),
                      })}
                    />
                  {/if}
                </li>
              {/each}
            </ul>
          {/if}
          <p class="more"><a href={href('/cashflow')}>{t('shell.nav.cashflow')}</a></p>
        </section>

        <section class="section" aria-labelledby="goals">
          <h2 id="goals">{t('shell.nav.goals')}</h2>
          {#if o.activeGoals.length === 0}
            <p class="muted">{t('shell.home.noActiveGoals')}</p>
          {:else}
            <div class="scroll">
              <table class="ledger">
                <thead>
                  <tr>
                    <SortHeader
                      label={t('shell.home.colGoal')}
                      column="name"
                      sort={goalSort}
                      onsort={onGoalSort}
                    />
                    <SortHeader
                      label={t('shell.home.colEnds')}
                      column="ends"
                      sort={goalSort}
                      onsort={onGoalSort}
                    />
                    <SortHeader
                      label={t('shell.home.saved')}
                      column="saved"
                      sort={goalSort}
                      onsort={onGoalSort}
                      numeric
                    />
                    <SortHeader
                      label={t('shell.home.colProgress')}
                      column="progress"
                      sort={goalSort}
                      onsort={onGoalSort}
                    />
                  </tr>
                </thead>
                <tbody>
                  {#each sortRows(o.activeGoals, goalSort, goalValues) as goal (goal.id)}
                    <tr>
                      <td>
                        <a href={href(`/goals/${goal.id}`)}>{goal.name}</a>
                        {#if goal.behind}<Pill text={t('shell.home.behindPlan')} tone="out" />{/if}
                      </td>
                      <td class="muted nowrap">{formatMonth(goal.endMonth)}</td>
                      <td class="num">
                        {formatCents(goal.savedCents)}
                        <span class="muted small"
                          >{t('shell.home.ofAmount', {
                            amount: formatCents(goal.targetCents),
                          })}</span
                        >
                      </td>
                      <td
                        ><ProgressBar
                          pct={goal.progressPct}
                          label={t('shell.home.progressOf', { name: goal.name })}
                        /></td
                      >
                    </tr>
                  {/each}
                </tbody>
              </table>
            </div>
          {/if}
          <p class="more"><a href={href('/goals')}>{t('shell.home.allGoals')}</a></p>
        </section>
      </div>

      <div class="columns">
        <section class="section" aria-labelledby="credits">
          <h2 id="credits">{t('shell.nav.credits')}</h2>
          {#if o.contracts.length === 0}
            <p class="muted">{t('shell.home.noContracts')}</p>
          {:else}
            <div class="scroll">
              <table class="ledger">
                <thead>
                  <tr>
                    <SortHeader
                      label={t('shell.home.colContract')}
                      column="name"
                      sort={contractSort}
                      onsort={onContractSort}
                    />
                    <SortHeader
                      label={t('shell.home.colOutstanding')}
                      column="outstanding"
                      sort={contractSort}
                      onsort={onContractSort}
                      numeric
                    />
                    <SortHeader
                      label={t('shell.home.colInstalment')}
                      column="instalment"
                      sort={contractSort}
                      onsort={onContractSort}
                      numeric
                    />
                    <SortHeader
                      label={t('shell.home.colNext')}
                      column="next"
                      sort={contractSort}
                      onsort={onContractSort}
                    />
                  </tr>
                </thead>
                <tbody>
                  {#each sortRows(o.contracts, contractSort, contractValues) as c (c.id)}
                    <tr>
                      <td>
                        <a href={href(`/credits/${c.id}`)}>{c.name}</a>
                        {#if c.endMonth}
                          <span class="muted small"
                            >{t('shell.home.endsIn', { month: formatMonth(c.endMonth) })}</span
                          >
                        {/if}
                      </td>
                      <td class="num">{formatCents(c.outstandingCents)}</td>
                      <td class="num"
                        >{c.instalmentCents > 0 ? formatCents(c.instalmentCents) : '—'}</td
                      >
                      <td class="muted nowrap">{nextInstalment(c)}</td>
                    </tr>
                  {/each}
                </tbody>
              </table>
            </div>
            <p class="muted small">
              {t('shell.home.instalmentNote')}
            </p>
          {/if}
          <p class="more"><a href={href('/credits')}>{t('shell.home.allCredits')}</a></p>
        </section>

        <section class="section" aria-labelledby="investments">
          <h2 id="investments">{t('shell.nav.investments')}</h2>
          {#if o.holdings.length === 0}
            <p class="muted">{t('shell.home.noInvestments')}</p>
          {:else}
            <table class="ledger">
              <tbody>
                {#each o.holdings as h (h.id)}
                  <tr>
                    <td><a href={href(`/investments/entities/${h.id}`)}>{h.name}</a></td>
                    <td class="num">{formatCents(h.valueCents)}</td>
                  </tr>
                {/each}
                <tr>
                  <td><strong>{t('shell.common.total')}</strong></td>
                  <td class="num"
                    ><strong>{formatCents(o.investments.currentValueCents)}</strong></td
                  >
                </tr>
              </tbody>
            </table>
          {/if}
          <p class="more"><a href={href('/investments')}>{t('shell.home.allInvestments')}</a></p>
        </section>
      </div>

      {#if o.savings.accounts > 0}
        <div class="columns">
          <section class="section" aria-labelledby="savings">
            <h2 id="savings">{t('shell.nav.savings')}</h2>
            <ul class="groups">
              {#each o.savings.items as a (a.id)}
                <li>
                  <div class="line">
                    <a href={href(`/savings/${a.id}`)}>{a.name}</a>
                    <strong>{formatCents(a.balanceCents)}</strong>
                  </div>
                  {#if a.targetCents !== null}
                    <ProgressBar
                      pct={a.progressPct}
                      label={t('shell.home.progressOf', { name: a.name })}
                    />
                  {/if}
                  <span class="muted small">{savingsAccountNote(a)}</span>
                </li>
              {/each}
            </ul>
            <table class="ledger">
              <tbody>
                <tr>
                  <td><strong>{t('shell.common.total')}</strong></td>
                  <td class="num"><strong>{formatCents(o.savings.balanceCents)}</strong></td>
                </tr>
              </tbody>
            </table>
            <p class="more"><a href={href('/savings')}>{t('shell.home.allSavings')}</a></p>
          </section>
        </div>
      {/if}
    {/if}
  {/snippet}
</Async>

<style>
  .review {
    display: flex;
    flex-wrap: wrap;
    justify-content: space-between;
    gap: 0.3rem 1rem;
    margin: 0 0 1.5rem;
    padding: 0.6rem 0;
    border-bottom: 1px solid var(--rule);
  }
  .columns {
    display: flex;
    flex-wrap: wrap;
    gap: 0 3rem;
  }
  .columns > .section {
    flex: 1 1 28rem;
    min-width: 0;
  }
  h3 {
    margin: 1.4rem 0 0.4rem;
    font-size: 1rem;
  }
  .groups {
    list-style: none;
    margin: 0;
    padding: 0;
  }
  .groups li {
    display: flex;
    flex-direction: column;
    gap: 0.3rem;
    padding: 0.6rem 0;
    border-bottom: 1px solid var(--rule);
  }
  .line {
    display: flex;
    justify-content: space-between;
    gap: 1rem;
  }
  .small {
    font-size: 0.85rem;
  }
  .scroll {
    overflow-x: auto;
  }
  .nowrap {
    white-space: nowrap;
  }
  .more {
    margin: 0.8rem 0 0;
    font-size: 0.9rem;
  }
</style>
