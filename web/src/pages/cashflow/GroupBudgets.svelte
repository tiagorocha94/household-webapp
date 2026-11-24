<script lang="ts">
  import type { CashflowMonthDetail } from '../../api/client';
  import CategoryLabel from '../../components/CategoryLabel.svelte';
  import ProgressBar from '../../components/ProgressBar.svelte';
  import { categoryShares, groupUse, type CategoryLook } from '../../lib/cashflow';
  import { groupLabel, t } from '../../lib/i18n/index.svelte';
  import { formatCents, formatPercent } from '../../lib/money';

  let { d, looks }: { d: CashflowMonthDetail; looks: Map<string, CategoryLook> } = $props();
</script>

<section class="section">
  <h2>{t('cashflow.budgetByGroup')}</h2>
  {#if d.incomeCents > 0}
    <p class="muted">
      {t('cashflow.groupsHelp', { amount: formatCents(d.incomeCents) })}
    </p>
  {:else}
    <p class="muted">{t('cashflow.noIncomeNoBudget')}</p>
  {/if}
  <div class="groups">
    {#each d.byGroup as g (g.name)}
      {@const use = groupUse(g)}
      <details>
        <summary class:nobudget={d.incomeCents === 0}>
          <span>
            <strong>{groupLabel(g.name)}</strong>
            <span class="muted small">
              {formatPercent(g.percent)} · {formatCents(g.budgetCents)}
            </span>
          </span>
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
          <span class="num"
            ><span class="muted small">{t('cashflow.spent')}</span>{formatCents(g.spentCents)}</span
          >
          {#if d.incomeCents > 0}
            <span class="num {use.over ? 'out' : use.remainingCents > 0 ? 'in' : ''}">
              <span class="muted small">{use.over ? t('cashflow.over') : t('cashflow.left')}</span>
              {formatCents(use.remainingCents)}
            </span>
          {/if}
        </summary>
        {#if g.categories.length === 0}
          <p class="muted small inner">{t('cashflow.groupEmpty')}</p>
        {:else}
          <table class="ledger inner">
            <tbody>
              {#each categoryShares(g.categories, 'expense') as { category, sharePct } (category.categoryId)}
                <tr>
                  <td>
                    <CategoryLabel name={category.name} look={looks.get(category.name)} />
                  </td>
                  <td class="num muted">{formatPercent(sharePct)}</td>
                  <td class="num">{formatCents(category.totalCents)}</td>
                </tr>
              {/each}
            </tbody>
          </table>
          <p class="muted small inner">{t('cashflow.groupPercentNote')}</p>
        {/if}
      </details>
    {/each}
  </div>
</section>

<style>
  .groups {
    border-top: 1px solid var(--rule);
  }
  details {
    border-bottom: 1px solid var(--rule);
  }
  summary {
    display: grid;
    grid-template-columns: minmax(9rem, 1.2fr) 3fr minmax(6rem, 1fr) minmax(6rem, 1fr);
    align-items: center;
    gap: 1rem;
    padding: 0.85rem 0;
    cursor: pointer;
  }
  summary.nobudget {
    grid-template-columns: minmax(9rem, 1.2fr) minmax(6rem, 1fr);
  }
  summary .num {
    display: grid;
    text-align: right;
  }
  .small {
    font-size: 0.85rem;
  }
  summary > span:first-child {
    display: grid;
  }
  .inner {
    margin: 0 0 0.85rem 1.25rem;
  }
</style>
