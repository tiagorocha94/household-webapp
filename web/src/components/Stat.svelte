<script lang="ts">
  let {
    label,
    value,
    tone,
    note,
  }: { label: string; value: string; tone?: 'in' | 'out'; note?: string } = $props();
</script>

<div class="stat">
  <dt>{label}</dt>
  <dd class={tone}>{value}</dd>
  {#if note}<dd class="note">{note}</dd>{/if}
</div>

<style>
  .stat {
    min-width: 0;
  }
  dt {
    color: var(--muted);
    font-size: 0.9rem;
  }
  dd {
    margin: 0;
    font-family: var(--serif);
    font-size: 1.7rem;
    line-height: 1.25;
  }
  dd.note {
    font-family: inherit;
    font-size: 0.85rem;
    color: var(--muted);
  }
</style>
