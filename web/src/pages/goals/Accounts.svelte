<script lang="ts">
  import { api } from '../../api/client';
  import Async from '../../components/Async.svelte';
  import Stat from '../../components/Stat.svelte';
  import Stats from '../../components/Stats.svelte';
  import { goalTotals } from '../../lib/goals';
  import { t } from '../../lib/i18n/index.svelte';
  import { formatCents } from '../../lib/money';
  import { Resource } from '../../lib/resource.svelte';
  import { href } from '../../lib/router.svelte';

  // The list endpoint has no balances or goals, so each account is fetched for its own.
  const accounts = new Resource(async () => {
    const list = await api.accounts();
    const rows = await Promise.all(
      list.map(async (account) => ({
        account,
        balance: (await api.account(account.id)).balance,
        goals: await api.accountGoals(account.id),
      })),
    );
    return rows;
  });
</script>

<h1>{t('goals.accounts.title')}</h1>

<Async resource={accounts} what="goal accounts">
  {#snippet children(rows)}
    {#if rows.length === 0}
      <p class="muted">{t('goals.accounts.empty')}</p>
    {:else}
      {@const sum = goalTotals(rows.flatMap((r) => r.goals))}
      <Stats>
        <Stat
          label={t('goals.accounts.balanceInGoals')}
          value={formatCents(sum.savedCents)}
          note={sum.targetCents > 0
            ? t('goals.accounts.ofTargeted', { target: formatCents(sum.targetCents) })
            : undefined}
        />
        <Stat label={t('goals.accounts.activeGoals')} value={String(sum.activeCount)} />
        <Stat label={t('goals.accounts.goalAccounts')} value={String(rows.length)} />
      </Stats>

      <section class="section">
        <h2>{t('goals.accounts.goalAccounts')}</h2>
        <table class="ledger">
          <thead>
            <tr>
              <th>{t('goals.accounts.account')}</th>
              <th class="num">{t('goals.accounts.monthlyDeposit')}</th>
              <th class="num">{t('goals.balance')}</th>
              <th class="num">{t('goals.free')}</th>
            </tr>
          </thead>
          <tbody>
            {#each rows as { account, balance } (account.id)}
              <tr>
                <td><a href={href(`/accounts/${account.id}`)}>{account.name}</a></td>
                <td class="num">{formatCents(account.monthlyDepositCents)}</td>
                <td class="num">{formatCents(balance.balanceCents)}</td>
                <td class="num {balance.unallocatedCents < 0 ? 'out' : ''}">
                  {formatCents(balance.unallocatedCents)}
                </td>
              </tr>
            {/each}
          </tbody>
        </table>
      </section>
    {/if}
  {/snippet}
</Async>
