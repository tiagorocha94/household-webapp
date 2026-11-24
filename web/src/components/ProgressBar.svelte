<script lang="ts">
  import { formatPercent } from '../lib/money';

  let { pct, label, over = false }: { pct: number; label: string; over?: boolean } = $props();
  const clamped = $derived(Math.min(100, Math.max(0, pct)));
</script>

<span class="progress">
  <span
    class="track"
    role="progressbar"
    aria-label={label}
    aria-valuemin="0"
    aria-valuemax="100"
    aria-valuenow={clamped}
  >
    <span class="fill" class:over style:width="{clamped}%"></span>
  </span>
  <span class="pct" class:over>{formatPercent(pct)}</span>
</span>

<style>
  .progress {
    display: inline-flex;
    align-items: center;
    gap: 0.6rem;
    min-width: 9rem;
  }
  .track {
    flex: 1;
    height: 0.45rem;
    background: var(--rule);
    border-radius: 0.25rem;
    overflow: hidden;
  }
  .fill {
    display: block;
    height: 100%;
    background: var(--accent);
  }
  .fill.over {
    background: var(--out);
  }
  .pct.over {
    color: var(--out);
  }
  .pct {
    min-width: 2.6rem;
    text-align: right;
  }
</style>
