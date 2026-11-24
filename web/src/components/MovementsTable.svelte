<script lang="ts">
  import type { Movement } from '../api/client';
  import { balancesAfter, signedCents } from '../lib/goals';
  import { t, type MessageKey } from '../lib/i18n/index.svelte';
  import { formatCents } from '../lib/money';
  import { formatMonth } from '../lib/month';
  import { href } from '../lib/router.svelte';
  import { sortRows, toggleSort, type Sort } from '../lib/sort';
  import Pill from './Pill.svelte';
  import SortHeader from './SortHeader.svelte';

  let {
    movements,
    view,
    showGoal = true,
  }: { movements: readonly Movement[]; view: 'account' | 'goal'; showGoal?: boolean } = $props();

  const pageSize = 25;
  let type = $state<Movement['type'] | 'all'>('all');
  let shown = $state(pageSize);

  // The movement types in the order the filter lists them, and the text each one shows.
  const typeOrder: Movement['type'][] = ['deposit', 'allocation', 'expense', 'release', 'cover'];
  const typeKey = (type: Movement['type']) => `cashflow.movementType.${type}` as const;
  // What closing a goal says about where the money went.
  const closingKey = (type: Movement['type']): MessageKey | undefined =>
    type === 'release' || type === 'cover' ? `cashflow.closingNote.${type}` : undefined;

  const types = $derived(typeOrder.filter((type) => movements.some((m) => m.type === type)));
  const rows = $derived(type === 'all' ? movements : movements.filter((m) => m.type === type));

  // A goal's balance after each movement, from the whole list so a filter keeps it right.
  const after = $derived(view === 'goal' ? balancesAfter(movements) : new Map<number, number>());

  type Column = 'month' | 'type' | 'goal' | 'description' | 'amount' | 'balance';
  let sort = $state<Sort<Column> | null>(null);
  const onsort = (column: Column) => (sort = toggleSort(sort, column));
  const values: Record<Column, (m: Movement) => string | number> = {
    month: (m) => m.month,
    type: (m) => t(typeKey(m.type)),
    goal: (m) => m.goalName,
    description: (m) => m.description,
    amount: (m) => signedCents(m, view),
    balance: (m) => after.get(m.id) ?? 0,
  };
  const ordered = $derived(sortRows(rows, sort, values));

  function choose(next: Movement['type'] | 'all') {
    type = next;
    shown = pageSize;
  }
</script>

{#if movements.length === 0}
  <p class="muted">{t('cashflow.noGoalMovements')}</p>
{:else}
  {#if types.length > 1}
    <div class="filters" role="group" aria-label={t('cashflow.filterByType')}>
      <button type="button" aria-pressed={type === 'all'} onclick={() => choose('all')}>
        {t('cashflow.all')}
      </button>
      {#each types as kind (kind)}
        <button type="button" aria-pressed={type === kind} onclick={() => choose(kind)}>
          {t(typeKey(kind))}
        </button>
      {/each}
    </div>
  {/if}
  <table class="ledger">
    <thead>
      <tr>
        <SortHeader label={t('cashflow.month')} column="month" {sort} {onsort} />
        <SortHeader label={t('cashflow.type')} column="type" {sort} {onsort} />
        {#if showGoal}<SortHeader label={t('cashflow.goal')} column="goal" {sort} {onsort} />{/if}
        <SortHeader label={t('cashflow.description')} column="description" {sort} {onsort} />
        <SortHeader label={t('cashflow.amount')} column="amount" {sort} {onsort} numeric />
        {#if view === 'goal'}
          <SortHeader label={t('cashflow.balanceAfter')} column="balance" {sort} {onsort} numeric />
        {/if}
      </tr>
    </thead>
    <tbody>
      {#each ordered.slice(0, shown) as m (m.id)}
        {@const signed = signedCents(m, view)}
        <tr>
          <td>{formatMonth(m.month)}</td>
          <td>
            {#if closingKey(m.type)}
              {t('cashflow.closing')}
              <Pill
                text={t(closingKey(m.type) ?? 'cashflow.closing')}
                tone={m.type === 'cover' ? 'out' : 'neutral'}
              />
            {:else}
              {t(typeKey(m.type))}
            {/if}
          </td>
          {#if showGoal}
            <td>
              {#if m.goalId !== null}
                <a href={href(`/goals/${m.goalId}`)}>{m.goalName}</a>
              {:else}
                <span class="muted">{t('cashflow.account')}</span>
              {/if}
            </td>
          {/if}
          <td class="muted">{m.description}</td>
          {#if view === 'goal' && closingKey(m.type)}
            <!-- A closing movement only settles the goal to zero, so it carries no sign or colour. -->
            <td class="num muted">{formatCents(m.amountCents)}</td>
          {:else}
            <td class="num {signed >= 0 ? 'in' : 'out'}">
              {signed >= 0 ? '+' : '−'}{formatCents(Math.abs(signed))}
            </td>
          {/if}
          {#if view === 'goal'}
            {@const running = after.get(m.id) ?? 0}
            <td class="num {running > 0 ? 'in' : running < 0 ? 'out' : ''}">
              {formatCents(running)}
            </td>
          {/if}
        </tr>
      {/each}
    </tbody>
  </table>
  {#if ordered.length > shown}
    <p>
      <button type="button" onclick={() => (shown += pageSize)}>
        {t('cashflow.showMore', { count: ordered.length - shown })}
      </button>
    </p>
  {/if}
{/if}
