<script lang="ts">
  import { iconComponent } from '../lib/icons';
  import { t } from '../lib/i18n/index.svelte';
  import { NAV, isCurrent, navLabel } from '../lib/nav';
  import { href } from '../lib/router.svelte';

  interface Props {
    path: string;
    version: string;
    collapsed: boolean;
    /** The drawer is open; it only matters below the desktop width. */
    open: boolean;
    ontoggle: () => void;
    onclose: () => void;
  }
  let { path, version, collapsed, open, ontoggle, onclose }: Props = $props();

  const Toggle = $derived(iconComponent(collapsed ? 'panel-left-open' : 'panel-left-close'));
</script>

<svelte:window onkeydown={(e) => open && e.key === 'Escape' && onclose()} />

{#if open}
  <button type="button" class="scrim" aria-label={t('shell.nav.closeMenu')} onclick={onclose}
  ></button>
{/if}

<aside class="side" class:collapsed class:open>
  <div class="head">
    <a class="brand" href={href('/')}>Household</a>
    <button
      type="button"
      class="toggle"
      aria-label={collapsed ? t('shell.nav.expandMenu') : t('shell.nav.collapseMenu')}
      aria-expanded={!collapsed}
      onclick={ontoggle}
    >
      <Toggle size={18} aria-hidden="true" />
    </button>
  </div>
  <nav aria-label={t('shell.nav.main')}>
    <ul>
      {#each NAV as item (item.path)}
        {@const Icon = iconComponent(item.icon)}
        <li>
          <a
            href={href(item.path)}
            aria-current={isCurrent(item, path) ? 'page' : undefined}
            title={collapsed ? navLabel(item) : undefined}
          >
            <Icon size={20} aria-hidden="true" />
            <span class="label">{navLabel(item)}</span>
          </a>
        </li>
      {/each}
    </ul>
  </nav>
  <div class="foot">
    {#if version}<span class="label ver">{t('shell.nav.apiVersion', { version })}</span>{/if}
  </div>
</aside>

<style>
  .side {
    position: sticky;
    top: 0;
    align-self: start;
    height: 100vh;
    width: 14.5rem;
    flex: none;
    display: flex;
    flex-direction: column;
    border-right: 1px solid var(--rule);
    background: var(--bg);
    overflow-y: auto;
  }
  .side.collapsed {
    width: 4rem;
  }
  .head {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 1.25rem 0.5rem 0.75rem 1rem;
  }
  .collapsed .head {
    flex-direction: column;
    gap: 0.5rem;
    padding: 1rem 0 0.5rem;
  }
  .collapsed .brand {
    display: none;
  }
  .brand {
    font-family: var(--serif);
    font-size: 1.25rem;
    font-weight: 600;
    color: var(--ink);
    text-decoration: none;
  }
  .toggle {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    border: 0;
    background: transparent;
    color: var(--muted);
    cursor: pointer;
    font: inherit;
    padding: 0.4rem;
    min-height: 2.75rem;
    min-width: 2.75rem;
    justify-content: center;
  }
  .toggle:hover {
    color: var(--ink);
  }
  ul {
    list-style: none;
    margin: 0;
    padding: 0.25rem 0;
  }
  nav a {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    min-height: 2.75rem;
    padding: 0 1rem;
    border-left: 4px solid transparent;
    color: var(--ink);
    text-decoration: none;
  }
  nav a:hover {
    background: color-mix(in srgb, var(--accent) 8%, transparent);
  }
  nav a[aria-current='page'] {
    border-left-color: var(--accent);
    background: color-mix(in srgb, var(--accent) 12%, transparent);
    color: var(--accent);
    font-weight: 600;
  }
  .collapsed nav a {
    justify-content: center;
    padding: 0;
  }
  .collapsed .label {
    display: none;
  }
  .foot {
    margin-top: auto;
    display: flex;
    flex-direction: column;
    align-items: flex-start;
    gap: 0.15rem;
    padding: 0.75rem 1rem 1rem;
    border-top: 1px solid var(--rule);
    font-size: 0.9rem;
  }
  .collapsed .foot {
    align-items: center;
    padding: 0.75rem 0;
  }
  .ver {
    color: var(--muted);
    font-size: 0.8rem;
  }
  .scrim {
    display: none;
  }

  /* Below the desktop width the menu is a drawer opened from the top bar. */
  @media (max-width: 56rem) {
    .side,
    .side.collapsed {
      position: fixed;
      inset: 0 auto 0 0;
      width: 17.5rem;
      z-index: 20;
      transform: translateX(-100%);
      transition: transform 0.15s ease-out;
      box-shadow: 4px 0 24px rgba(0, 0, 0, 0.3);
    }
    .side.open {
      transform: none;
    }
    .side.collapsed .label {
      display: inline;
    }
    .side.collapsed .brand {
      display: inline;
    }
    .toggle {
      display: none;
    }
    .scrim {
      display: block;
      position: fixed;
      inset: 0;
      z-index: 10;
      border: 0;
      background: rgba(0, 0, 0, 0.4);
    }
    .collapsed nav a {
      justify-content: flex-start;
      padding: 0 1rem;
    }
    .collapsed .head {
      flex-direction: row;
      padding: 1.25rem 0.5rem 0.75rem 1rem;
    }
    .collapsed .foot {
      align-items: flex-start;
      padding: 0.75rem 1rem 1rem;
    }
  }
  @media (prefers-reduced-motion: reduce) {
    .side {
      transition: none;
    }
  }
</style>
