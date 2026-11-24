<script lang="ts">
  import { iconComponent } from '../lib/icons';
  import { t } from '../lib/i18n/index.svelte';
  import { badgeText } from '../lib/notifications';
  import { href } from '../lib/router.svelte';

  interface Props {
    count: number;
    username: string | undefined;
    onmenu: () => void;
    onsignout: () => void;
  }
  let { count, username, onmenu, onsignout }: Props = $props();

  const MenuIcon = iconComponent('menu');
  const BellIcon = iconComponent('bell');
  const SignOut = iconComponent('log-out');
  const GearIcon = iconComponent('settings');
</script>

<header class="topbar">
  <button type="button" class="menu" aria-label={t('shell.nav.openMenu')} onclick={onmenu}>
    <MenuIcon size={22} aria-hidden="true" />
  </button>
  <a class="brand" href={href('/')}>Household</a>
  <div class="right">
    <a
      class="bell"
      href={href('/notifications')}
      aria-label={count > 0
        ? t('shell.topbar.notificationsCount', { count })
        : t('shell.notifications')}
      title={t('shell.notifications')}
    >
      <BellIcon size={20} aria-hidden="true" />
      {#if count > 0}<span class="badge" aria-hidden="true">{badgeText(count)}</span>{/if}
    </a>
    <a
      class="gear"
      href={href('/settings')}
      aria-label={t('shell.topbar.settings')}
      title={t('shell.topbar.settings')}
    >
      <GearIcon size={20} aria-hidden="true" />
    </a>
    <span class="who">{username}</span>
    <button
      type="button"
      class="out"
      aria-label={t('shell.topbar.signOut')}
      title={t('shell.topbar.signOut')}
      onclick={onsignout}
    >
      <SignOut size={20} aria-hidden="true" />
    </button>
  </div>
</header>

<style>
  .topbar {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    padding: 0.5rem 2rem;
    border-bottom: 1px solid var(--rule);
  }
  .menu,
  .brand {
    display: none;
  }
  .right {
    margin-left: auto;
    display: flex;
    align-items: center;
    gap: 1.75rem;
  }
  .bell,
  .gear,
  .out {
    position: relative;
    display: flex;
    align-items: center;
    justify-content: center;
    width: 2.75rem;
    height: 2.75rem;
    border: 0;
    background: transparent;
    color: var(--muted);
    cursor: pointer;
  }
  .bell:hover,
  .gear:hover,
  .out:hover {
    color: var(--ink);
  }
  .badge {
    position: absolute;
    top: 0.45rem;
    right: 0.4rem;
    min-width: 0.95rem;
    height: 0.95rem;
    border-radius: 50%;
    background: var(--out);
    color: #fff;
    font-size: 0.6rem;
    font-weight: 700;
    line-height: 0.95rem;
    text-align: center;
    box-shadow: 0 0 0 2px var(--bg);
  }
  .who {
    color: var(--muted);
    font-size: 0.9rem;
  }
  @media (max-width: 56rem) {
    .topbar {
      padding: 0.25rem 0.5rem;
    }
    .menu {
      display: flex;
      align-items: center;
      justify-content: center;
      width: 2.75rem;
      height: 2.75rem;
      border: 0;
      background: transparent;
      color: var(--ink);
      cursor: pointer;
    }
    .brand {
      display: inline;
      font-family: var(--serif);
      font-size: 1.2rem;
      font-weight: 600;
      color: var(--ink);
      text-decoration: none;
    }
    .right {
      gap: 0.5rem;
    }
    .who {
      display: none;
    }
  }
</style>
