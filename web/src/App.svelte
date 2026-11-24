<script lang="ts">
  import { onMount } from 'svelte';
  import { t } from './lib/i18n/index.svelte';
  import { api } from './api/client';
  import { matchRoute, type Route } from './lib/router';
  import { isMonth } from './lib/month';
  import { isMonthView } from './lib/cashflow';
  import { isEnabled, moduleOfRoute } from './lib/modules';
  import { serverModules, setServerModules } from './lib/modules.svelte';
  import { loadCollapsed, saveCollapsed } from './lib/nav';
  import SideNav from './components/SideNav.svelte';
  import TopBar from './components/TopBar.svelte';
  import { attention, refreshAttention } from './lib/attention.svelte';
  import { href, navigate, router } from './lib/router.svelte';
  import { logOut, restoreSession, session } from './lib/session.svelte';
  import Account from './pages/goals/Account.svelte';
  import Accounts from './pages/goals/Accounts.svelte';
  import Contract from './pages/credits/Contract.svelte';
  import Contracts from './pages/credits/Contracts.svelte';
  import CashflowMonth from './pages/cashflow/Month.svelte';
  import CashflowMonths from './pages/cashflow/Months.svelte';
  import Goal from './pages/goals/Goal.svelte';
  import Home from './pages/Home.svelte';
  import Entities from './pages/investments/Entities.svelte';
  import Entity from './pages/investments/Entity.svelte';
  import Instrument from './pages/investments/Instrument.svelte';
  import Login from './pages/Login.svelte';
  import NetWorth from './pages/NetWorth.svelte';
  import Notifications from './pages/Notifications.svelte';
  import NotFound from './pages/NotFound.svelte';
  import Settings from './pages/Settings.svelte';
  import SavingsAccount from './pages/savings/Account.svelte';
  import SavingsAccounts from './pages/savings/Accounts.svelte';

  const routes: Route[] = [
    { name: 'home', pattern: '/' },
    { name: 'login', pattern: '/login' },
    { name: 'notifications', pattern: '/notifications' },
    { name: 'settings', pattern: '/settings' },
    { name: 'net-worth', pattern: '/net-worth' },
    { name: 'accounts', pattern: '/goals' },
    { name: 'account', pattern: '/accounts/:id' },
    { name: 'goal', pattern: '/goals/:id' },
    { name: 'savings', pattern: '/savings' },
    { name: 'saving', pattern: '/savings/:id' },
    { name: 'credits', pattern: '/credits' },
    { name: 'contract', pattern: '/credits/:id' },
    { name: 'investments', pattern: '/investments' },
    { name: 'entity', pattern: '/investments/entities/:id' },
    { name: 'instrument', pattern: '/investments/instruments/:id' },
    { name: 'cashflow', pattern: '/cashflow' },
    { name: 'cashflow-month', pattern: '/cashflow/months/:month' },
    { name: 'cashflow-month-view', pattern: '/cashflow/months/:month/:view' },
  ];

  let version = $state('');
  let collapsed = $state(loadCollapsed());
  let drawerOpen = $state(false);

  function toggleCollapsed() {
    collapsed = !collapsed;
    saveCollapsed(collapsed);
  }

  // Going to another page closes the drawer.
  $effect(() => {
    void router.path;
    drawerOpen = false;
  });
  const match = $derived(matchRoute(router.path, routes));
  // A page of a module this server does not run is not found.
  const moduleOff = $derived(!isEnabled(serverModules.enabled, moduleOfRoute(match?.name)));

  // The count on the bell is read again whenever the person goes to another page.
  $effect(() => {
    void router.path;
    if (session.status === 'authenticated') void refreshAttention();
  });

  onMount(async () => {
    // Asked at the same time as the session, so the menu is right when it first shows.
    const info = api
      .info()
      .then((info) => {
        version = info.version;
        setServerModules(info.modules);
      })
      .catch(() => {});
    await restoreSession();
    await info;
  });

  // Signed-in people never see the login page, and everyone else sees only that.
  $effect(() => {
    if (session.status === 'authenticated' && match?.name === 'login') navigate('/');
  });

  // A numeric id in the address, or NaN, which the pages report as not found.
  const id = $derived(Number(match?.params.id));
  // A month in the address, or empty, which the page reports as not found.
  const monthParam = $derived(match?.params.month ?? '');
  // The page of a month in the address; anything else is not found.
  const viewParam = $derived(match?.params.view ?? '');
</script>

{#if session.status === 'loading'}
  <p class="state" aria-live="polite">{t('shell.common.loading')}</p>
{:else if session.status === 'anonymous'}
  <Login />
{:else}
  <div class="shell">
    <SideNav
      path={router.path}
      {version}
      {collapsed}
      open={drawerOpen}
      ontoggle={toggleCollapsed}
      onclose={() => (drawerOpen = false)}
    />
    <div class="content">
      <TopBar
        count={attention.count}
        username={session.user?.username}
        onmenu={() => (drawerOpen = true)}
        onsignout={() => logOut()}
      />
      <main class="page">
        {#key router.path}
          {#if moduleOff}
            <NotFound />
          {:else if match?.name === 'home'}
            <Home />
          {:else if match?.name === 'net-worth'}
            <NetWorth />
          {:else if match?.name === 'notifications'}
            <Notifications />
          {:else if match?.name === 'settings'}
            <Settings />
          {:else if match?.name === 'accounts'}
            <Accounts />
          {:else if match?.name === 'account' && Number.isInteger(id)}
            <Account {id} />
          {:else if match?.name === 'goal' && Number.isInteger(id)}
            <Goal {id} />
          {:else if match?.name === 'savings'}
            <SavingsAccounts />
          {:else if match?.name === 'saving' && Number.isInteger(id)}
            <SavingsAccount {id} />
          {:else if match?.name === 'credits'}
            <Contracts />
          {:else if match?.name === 'contract' && Number.isInteger(id)}
            <Contract {id} />
          {:else if match?.name === 'investments'}
            <Entities />
          {:else if match?.name === 'entity' && Number.isInteger(id)}
            <Entity {id} />
          {:else if match?.name === 'instrument' && Number.isInteger(id)}
            <Instrument {id} />
          {:else if match?.name === 'cashflow'}
            <CashflowMonths />
          {:else if match?.name === 'cashflow-month' && isMonth(monthParam)}
            <CashflowMonth month={monthParam} />
          {:else if match?.name === 'cashflow-month-view' && isMonth(monthParam) && isMonthView(viewParam) && viewParam !== 'summary'}
            <CashflowMonth month={monthParam} view={viewParam} />
          {:else}
            <NotFound />
          {/if}
        {/key}
      </main>
    </div>
  </div>
{/if}

<style>
  .shell {
    display: flex;
    min-height: 100vh;
  }
  .content {
    flex: 1;
    min-width: 0;
  }
  .page {
    max-width: 72rem;
    margin: 0 auto;
    padding: 2rem 2rem 4rem;
  }
  @media (max-width: 56rem) {
    .page {
      padding: 1.25rem 1rem 3rem;
    }
  }
</style>
