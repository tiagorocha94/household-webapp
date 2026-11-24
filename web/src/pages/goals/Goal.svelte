<script lang="ts">
  import { api } from '../../api/client';
  import Async from '../../components/Async.svelte';
  import GoalChart from '../../components/GoalChart.svelte';
  import MovementsTable from '../../components/MovementsTable.svelte';
  import Pill from '../../components/Pill.svelte';
  import ProgressBar from '../../components/ProgressBar.svelte';
  import Stat from '../../components/Stat.svelte';
  import Stats from '../../components/Stats.svelte';
  import {
    balanceTone,
    goalPhase,
    goalSeries,
    overspentCents,
    phaseLabel,
    shownBalance,
  } from '../../lib/goals';
  import { t } from '../../lib/i18n/index.svelte';
  import { formatCents } from '../../lib/money';
  import { currentMonth, formatMonth } from '../../lib/month';
  import { Resource } from '../../lib/resource.svelte';
  import { href } from '../../lib/router.svelte';

  let { id }: { id: number } = $props();

  const detail = new Resource(() => api.goal(id));
  const now = currentMonth();
</script>

<Async resource={detail} what="goal">
  {#snippet children({ goal, movements })}
    {@const phase = goalPhase(goal, now)}
    {@const over = overspentCents(goal)}
    {@const closed = goal.status === 'completed' && goal.closingBalanceCents !== null}
    {@const balance = shownBalance(goal)}
    <p><a href={href(`/accounts/${goal.accountId}`)}>{t('goals.goal.back')}</a></p>
    <h1>{goal.name}</h1>
    <div class="head">
      <p class="muted">
        <Pill text={phaseLabel(phase)} tone={phase === 'completed' ? 'in' : 'accent'} />
        {t('goals.period', {
          from: formatMonth(goal.startMonth),
          to: formatMonth(goal.endMonth),
        })}{#if goal.status === 'active'}, {t('goals.goal.monthOf', {
            n: Math.min(goal.elapsedMonths, goal.totalMonths),
            total: goal.totalMonths,
          })}{/if}
      </p>
      <p class="figure">
        <span class="muted">{t('goals.target')}</span>
        {formatCents(goal.targetCents)}
      </p>
      <p class="figure">
        <span class="muted">{t('goals.perMonth')}</span>
        {formatCents(goal.monthlyAllocationCents)}
      </p>
    </div>
    <p><ProgressBar pct={goal.progressPct} label={t('goals.progressOf', { name: goal.name })} /></p>
    {#if over > 0}
      <p class="warning" role="note">
        <strong>{t('goals.goal.overTitle')}</strong>
        {t('goals.goal.overText', {
          spent: formatCents(goal.expensesCents),
          target: formatCents(goal.targetCents),
          over: formatCents(over),
        })}
      </p>
    {/if}
    <Stats>
      <Stat
        label={closed ? t('goals.goal.closingBalance') : t('goals.balance')}
        value={formatCents(balance)}
        tone={balanceTone(balance)}
        note={closed ? t('goals.goal.recordedWhenCompleted') : undefined}
      />
      <Stat label={t('goals.allocated')} value={formatCents(goal.allocatedCents)} />
      <Stat label={t('goals.spent')} value={formatCents(goal.expensesCents)} />
    </Stats>
    <GoalChart series={goalSeries(goal, movements, now)} />
    <section class="section">
      <h2>{t('goals.movements')}</h2>
      <MovementsTable {movements} view="goal" showGoal={false} />
    </section>
  {/snippet}
</Async>

<style>
  .head {
    display: flex;
    flex-wrap: wrap;
    align-items: baseline;
    gap: 0 1.6rem;
  }
  .head p {
    margin: 0 0 0.4rem;
  }
  .figure {
    font-family: var(--serif);
    font-size: 1.1rem;
  }
  .figure span {
    font-family: var(--sans);
    font-size: 0.9rem;
    margin-right: 0.3rem;
  }
  .warning {
    margin: 0 0 1rem;
    padding: 0.7rem 1rem;
    border-left: 3px solid var(--out);
    background: color-mix(in srgb, var(--out) 10%, var(--bg));
  }
  .warning strong {
    display: block;
    color: var(--out);
  }
</style>
