<script lang="ts">
  import type { SavingsAccount } from '../../api/client';
  import { api } from '../../api/client';
  import Async from '../../components/Async.svelte';
  import ProgressBar from '../../components/ProgressBar.svelte';
  import SortHeader from '../../components/SortHeader.svelte';
  import Stat from '../../components/Stat.svelte';
  import Stats from '../../components/Stats.svelte';
  import { t } from '../../lib/i18n/index.svelte';
  import { ownerLabel } from '../../lib/investments';
  import { formatCents, formatPercent } from '../../lib/money';
  import { Resource } from '../../lib/resource.svelte';
  import { href } from '../../lib/router.svelte';
  import { totals } from '../../lib/savings';
  import { sortRows, toggleSort, type Sort } from '../../lib/sort';

  const accounts = new Resource(() => api.savings());

  type Column = 'name' | 'owner' | 'balance' | 'progress' | 'interest';
  const values: Record<Column, (a: SavingsAccount) => string | number> = {
    name: (a) => a.name,
    owner: (a) => ownerLabel(a.owner),
    balance: (a) => a.balanceCents,
    progress: (a) => (a.targetCents === null ? -1 : a.progressPct),
    interest: (a) => a.interestCents,
  };
  let sort = $state<Sort<Column> | null>(null);
  const onsort = (column: Column) => (sort = toggleSort(sort, column));
</script>

<h1>{t('goals.savings.title')}</h1>
<p class="muted">{t('goals.savings.intro')}</p>

<Async resource={accounts} what="savings">
  {#snippet children(list)}
    {#if list.length === 0}
      <p class="muted">{t('goals.savings.empty')}</p>
    {:else}
      {@const sum = totals(list)}
      <Stats>
        <Stat label={t('goals.savings.saved')} value={formatCents(sum.savedCents)} />
        <Stat
          label={t('goals.savings.ofTargets')}
          value={sum.targetCents > 0 ? formatCents(sum.targetCents) : '—'}
          note={sum.targetCents > 0
            ? t('goals.savings.reached', { pct: formatPercent(sum.progressPct) })
            : undefined}
        />
        <Stat label={t('goals.savings.interestEarned')} value={formatCents(sum.interestCents)} />
      </Stats>

      <section class="section">
        <h2>{t('goals.savings.accounts')}</h2>
        <div class="scroll">
          <table class="ledger">
            <thead>
              <tr>
                <SortHeader label={t('goals.savings.account')} column="name" {sort} {onsort} />
                <SortHeader label={t('goals.savings.owner')} column="owner" {sort} {onsort} />
                <SortHeader label={t('goals.balance')} column="balance" {sort} {onsort} numeric />
                <SortHeader
                  label={t('goals.savings.towardsTarget')}
                  column="progress"
                  {sort}
                  {onsort}
                />
                <SortHeader label={t('goals.interest')} column="interest" {sort} {onsort} numeric />
              </tr>
            </thead>
            <tbody>
              {#each sortRows(list, sort, values) as a (a.id)}
                <tr>
                  <td><a href={href(`/savings/${a.id}`)}>{a.name}</a></td>
                  <td>{ownerLabel(a.owner)}</td>
                  <td class="num">{formatCents(a.balanceCents)}</td>
                  <td>
                    {#if a.targetCents === null}
                      <span class="muted">{t('goals.savings.noTarget')}</span>
                    {:else}
                      <ProgressBar
                        pct={a.progressPct}
                        label={t('goals.savings.progressLabel', {
                          name: a.name,
                          pct: formatPercent(a.progressPct),
                          target: formatCents(a.targetCents),
                        })}
                      />
                      <span class="muted small"
                        >{t('goals.savings.ofAmount', { target: formatCents(a.targetCents) })}</span
                      >
                    {/if}
                  </td>
                  <td class="num">{formatCents(a.interestCents)}</td>
                </tr>
              {/each}
            </tbody>
          </table>
        </div>
        <p class="muted small">{t('goals.savings.accountsNote')}</p>
      </section>
    {/if}
  {/snippet}
</Async>

<style>
  .scroll {
    overflow-x: auto;
  }
  .small {
    font-size: 0.85rem;
  }
</style>
