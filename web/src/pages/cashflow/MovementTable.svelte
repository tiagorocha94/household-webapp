<script lang="ts">
  import type { CashflowMovement } from '../../api/client';
  import CategoryLabel from '../../components/CategoryLabel.svelte';
  import SortHeader from '../../components/SortHeader.svelte';
  import { signedAmount, type CategoryLook, type MovementColumn } from '../../lib/cashflow';
  import { t } from '../../lib/i18n/index.svelte';
  import { formatCents } from '../../lib/money';
  import { formatDay } from '../../lib/month';
  import type { Sort } from '../../lib/sort';

  let {
    rows,
    looks,
    sort,
    onsort,
  }: {
    rows: CashflowMovement[];
    looks: Map<string, CategoryLook>;
    sort: Sort<MovementColumn> | null;
    onsort: (column: MovementColumn) => void;
  } = $props();
</script>

<table class="ledger">
  <thead>
    <tr>
      <SortHeader label={t('cashflow.date')} column="date" {sort} {onsort} />
      <SortHeader label={t('cashflow.person')} column="user" {sort} {onsort} />
      <SortHeader label={t('cashflow.category')} column="category" {sort} {onsort} />
      <SortHeader label={t('cashflow.description')} column="description" {sort} {onsort} />
      <SortHeader label={t('cashflow.amount')} column="amount" {sort} {onsort} numeric />
    </tr>
  </thead>
  <tbody>
    {#each rows as m (m.id)}
      {@const signed = signedAmount(m)}
      <tr>
        <td>{formatDay(m.date)}</td>
        <td>{m.username}</td>
        <td><CategoryLabel name={m.category} look={looks.get(m.category)} /></td>
        <td class="muted">{m.description}</td>
        <td class="num {signed >= 0 ? 'in' : 'out'}">
          {signed >= 0 ? '+' : '−'}{formatCents(Math.abs(signed))}
        </td>
      </tr>
    {/each}
  </tbody>
</table>
