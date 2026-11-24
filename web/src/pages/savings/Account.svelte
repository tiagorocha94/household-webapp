<script lang="ts">
  import { api, type SavingsMovement, type SavingsSnapshot } from '../../api/client';
  import Async from '../../components/Async.svelte';
  import HistoryChart from '../../components/HistoryChart.svelte';
  import ProgressBar from '../../components/ProgressBar.svelte';
  import SortHeader from '../../components/SortHeader.svelte';
  import Stat from '../../components/Stat.svelte';
  import Stats from '../../components/Stats.svelte';
  import { t } from '../../lib/i18n/index.svelte';
  import { ownerLabel } from '../../lib/investments';
  import { formatCents, formatPercent } from '../../lib/money';
  import { formatMonth } from '../../lib/month';
  import { Resource } from '../../lib/resource.svelte';
  import { href } from '../../lib/router.svelte';
  import { buildBalanceHistory, movementLabel, remainingCents } from '../../lib/savings';
  import { sortRows, toggleSort, type Sort } from '../../lib/sort';

  let { id }: { id: number } = $props();

  const detail = new Resource(() => api.savingsAccount(id));
  const movements = new Resource(() => api.savingsMovements(id));
  const snapshots = new Resource(() => api.savingsSnapshots(id));

  type MovementColumn = 'month' | 'type' | 'description' | 'amount';
  const movementValues: Record<MovementColumn, (m: SavingsMovement) => string | number> = {
    month: (m) => m.month,
    type: (m) => movementLabel(m.type),
    description: (m) => m.description,
    amount: (m) => (m.type === 'withdrawal' ? -m.amountCents : m.amountCents),
  };
  type SnapshotColumn = 'month' | 'description' | 'value';
  const snapshotValues: Record<SnapshotColumn, (s: SavingsSnapshot) => string | number> = {
    month: (s) => s.month,
    description: (s) => s.description,
    value: (s) => s.valueCents,
  };
  let movementSort = $state<Sort<MovementColumn> | null>(null);
  let snapshotSort = $state<Sort<SnapshotColumn> | null>(null);
  const onmovementsort = (column: MovementColumn) =>
    (movementSort = toggleSort(movementSort, column));
  const onsnapshotsort = (column: SnapshotColumn) =>
    (snapshotSort = toggleSort(snapshotSort, column));
</script>

<Async resource={detail} what="savings account">
  {#snippet children(a)}
    <p><a href={href('/savings')}>{t('goals.savings.all')}</a></p>
    <h1>{a.name}</h1>
    <p class="muted">{t('goals.savings.ownerLine', { owner: ownerLabel(a.owner) })}</p>
    <Stats>
      <Stat
        label={t('goals.balance')}
        value={formatCents(a.balanceCents)}
        note={a.lastSnapshotMonth
          ? t('goals.savings.snapshotOf', { month: formatMonth(a.lastSnapshotMonth) })
          : t('goals.savings.fromMovements')}
      />
      {#if a.targetCents !== null}
        {@const left = remainingCents(a)}
        <Stat
          label={t('goals.target')}
          value={formatCents(a.targetCents)}
          note={left
            ? t('goals.savings.toGo', { amount: formatCents(left) })
            : t('goals.savings.targetReached')}
        />
      {/if}
      <Stat label={t('goals.deposited')} value={formatCents(a.depositedCents)} />
      <Stat label={t('goals.savings.withdrawn')} value={formatCents(a.withdrawnCents)} />
      <Stat label={t('goals.savings.interestEarned')} value={formatCents(a.interestCents)} />
    </Stats>
    {#if a.targetCents !== null}
      <p>
        <strong>{t('goals.savings.pctOfTarget', { pct: formatPercent(a.progressPct) })}</strong>
      </p>
      <ProgressBar
        pct={a.progressPct}
        label={t('goals.savings.progressLabel', {
          name: a.name,
          pct: formatPercent(a.progressPct),
          target: formatCents(a.targetCents),
        })}
      />
      <p class="muted small">{t('goals.savings.noPlan')}</p>
    {/if}
    {#if a.notes}<p>{a.notes}</p>{/if}
  {/snippet}
</Async>

<section class="section">
  <h2>{t('goals.savings.balanceByMonth')}</h2>
  <Async resource={movements}>
    {#snippet children(movementList)}
      <Async resource={snapshots}>
        {#snippet children(snapshotList)}
          {@const history = buildBalanceHistory(movementList, snapshotList)}
          {#if history.length === 0}
            <p class="muted">{t('goals.savings.nothingYet')}</p>
          {:else}
            <HistoryChart
              {history}
              valueLabel={t('goals.balance')}
              inLabel={t('goals.savings.depositedLessWithdrawn')}
            />
          {/if}
        {/snippet}
      </Async>
    {/snippet}
  </Async>
</section>

<section class="section">
  <h2>{t('goals.movements')}</h2>
  <Async resource={movements}>
    {#snippet children(list)}
      {#if list.length === 0}
        <p class="muted">{t('goals.savings.noMovements')}</p>
      {:else}
        <div class="scroll">
          <table class="ledger">
            <thead>
              <tr>
                <SortHeader
                  label={t('goals.month')}
                  column="month"
                  sort={movementSort}
                  onsort={onmovementsort}
                />
                <SortHeader
                  label={t('goals.type')}
                  column="type"
                  sort={movementSort}
                  onsort={onmovementsort}
                />
                <SortHeader
                  label={t('goals.savings.note')}
                  column="description"
                  sort={movementSort}
                  onsort={onmovementsort}
                />
                <SortHeader
                  label={t('goals.amount')}
                  column="amount"
                  sort={movementSort}
                  onsort={onmovementsort}
                  numeric
                />
              </tr>
            </thead>
            <tbody>
              {#each sortRows(list, movementSort, movementValues) as m (m.id)}
                <tr>
                  <td>{formatMonth(m.month)}</td>
                  <td>{movementLabel(m.type)}</td>
                  <td class="muted">
                    {m.description}
                  </td>
                  <td class="num {m.type === 'deposit' ? 'in' : 'out'}">
                    {m.type === 'deposit' ? '+' : '−'}{formatCents(m.amountCents)}
                  </td>
                </tr>
              {/each}
            </tbody>
          </table>
        </div>
        <p class="muted small">{t('goals.savings.putBack')}</p>
      {/if}
    {/snippet}
  </Async>
</section>

<section class="section">
  <h2>{t('goals.savings.snapshots')}</h2>
  <Async resource={snapshots}>
    {#snippet children(list)}
      {#if list.length === 0}
        <p class="muted">{t('goals.savings.noSnapshots')}</p>
      {:else}
        <table class="ledger">
          <thead>
            <tr>
              <SortHeader
                label={t('goals.month')}
                column="month"
                sort={snapshotSort}
                onsort={onsnapshotsort}
              />
              <SortHeader
                label={t('goals.savings.note')}
                column="description"
                sort={snapshotSort}
                onsort={onsnapshotsort}
              />
              <SortHeader
                label={t('goals.balance')}
                column="value"
                sort={snapshotSort}
                onsort={onsnapshotsort}
                numeric
              />
            </tr>
          </thead>
          <tbody>
            {#each sortRows(list, snapshotSort, snapshotValues) as s (s.id)}
              <tr>
                <td>{formatMonth(s.month)}</td>
                <td class="muted">{s.description}</td>
                <td class="num">{formatCents(s.valueCents)}</td>
              </tr>
            {/each}
          </tbody>
        </table>
      {/if}
    {/snippet}
  </Async>
</section>

<style>
  .scroll {
    overflow-x: auto;
  }
  .small {
    font-size: 0.85rem;
  }
</style>
