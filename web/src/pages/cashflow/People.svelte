<script lang="ts">
  import type { CashflowMonthDetail } from '../../api/client';
  import { netTone } from '../../lib/cashflow';
  import { t } from '../../lib/i18n/index.svelte';
  import { formatCents } from '../../lib/money';
  import { formatMonth } from '../../lib/month';
  import { personSplit, putAsideHistory, type PutAsideCell } from '../../lib/people';

  let { d, month }: { d: CashflowMonthDetail; month: string } = $props();
  const history = $derived(putAsideHistory(d.peopleHistory, month));
</script>

<section class="section">
  <h2>{t('cashflow.byPerson')}</h2>
  <p class="muted">
    {t('cashflow.putAsideHelp')}
  </p>
  <div class="people">
    {#each d.byUser as u (u.username)}
      {@const split = personSplit(u)}
      <article class="person">
        <h3>
          {u.username}
          {#if u.cashCents !== null}<small class="muted"
              >{t('cashflow.cash', { amount: formatCents(u.cashCents) })}</small
            >{/if}
        </h3>
        <p class="income">
          <span>{t('cashflow.income')}</span><strong>{formatCents(u.incomeCents)}</strong>
        </p>
        <div class="split" role="img" aria-label={t('cashflow.splitLabel')}>
          <span class="spent" style:width="{split.spentWidth}%"></span>
          <span class="aside" style:width="{split.putAsideWidth}%"></span>
          <span class="left" style:width="{split.leftOverWidth}%"></span>
        </div>
        <dl class="parts">
          <div>
            <dt><i class="spent"></i>{t('cashflow.spent')}</dt>
            <dd>{formatCents(split.spentCents)}</dd>
          </div>
          <div>
            <dt><i class="aside"></i>{t('cashflow.putAside')}</dt>
            <dd>{formatCents(split.putAsideCents)}</dd>
          </div>
          <div>
            <dt><i class="left"></i>{t('cashflow.leftOver')}</dt>
            <dd class={netTone(split.leftOverCents) ?? ''}>
              {formatCents(split.leftOverCents)}
            </dd>
          </div>
        </dl>
        {#if split.putAsideShare !== null && split.spentShare !== null}
          <p class="muted">
            {t('cashflow.shareBefore')} <strong>{split.putAsideShare}%</strong>
            {t('cashflow.shareMiddle')}
            <strong>{split.spentShare}%</strong>
          </p>
        {/if}
      </article>
    {/each}
  </div>
</section>
{#if history.people.length > 0}
  <section class="section">
    <h2>{t('cashflow.putAsideLast6')}</h2>
    <table class="ledger">
      <thead>
        <tr>
          <th></th>
          {#each history.months as m (m)}<th class="num">{formatMonth(m)}</th>{/each}
        </tr>
      </thead>
      <tbody>
        {#each history.people as p (p.username)}
          {@render putAsideRows(p.username, p.cells)}
        {/each}
        {@render putAsideRows(t('cashflow.household'), history.household, true)}
      </tbody>
    </table>
  </section>
{/if}

{#snippet putAsideRows(name: string, cells: (PutAsideCell | null)[], total = false)}
  <tr class:total>
    <td>{t('cashflow.rowPutAside', { name })}</td>
    {#each cells as c, i (i)}
      <td class="num">{c === null ? '—' : formatCents(c.putAsideCents)}</td>
    {/each}
  </tr>
  <tr class:total>
    <td>{t('cashflow.rowOfIncome', { name })}</td>
    {#each cells as c, i (i)}
      <td class="num">{c === null || c.sharePct === null ? '—' : `${c.sharePct}%`}</td>
    {/each}
  </tr>
{/snippet}

<style>
  .people {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(17rem, 1fr));
    gap: 1.2rem;
  }
  .person {
    padding: 1rem;
    border: 1px solid var(--rule);
    border-radius: 6px;
    min-width: 0;
  }
  .person h3 {
    display: flex;
    justify-content: space-between;
    align-items: baseline;
    gap: 0.6rem;
    margin: 0 0 0.6rem;
  }
  .person h3 small {
    font-weight: 400;
  }
  .income {
    display: flex;
    justify-content: space-between;
    margin: 0 0 0.5rem;
  }
  .split {
    display: flex;
    height: 1rem;
    border-radius: 0.3rem;
    overflow: hidden;
    background: var(--rule);
  }
  .split span {
    display: block;
    height: 100%;
  }
  .spent {
    background: var(--series-3, #8d6e63);
  }
  .aside {
    background: var(--series-1);
  }
  .left {
    background: var(--series-2);
  }
  .parts {
    margin: 0.6rem 0;
  }
  .parts div {
    display: flex;
    justify-content: space-between;
    padding: 0.15rem 0;
  }
  .parts dt,
  .parts dd {
    margin: 0;
  }
  .parts i {
    display: inline-block;
    width: 0.65rem;
    height: 0.65rem;
    margin-right: 0.5rem;
    border-radius: 2px;
  }
  tr.total td {
    font-weight: 700;
  }
</style>
