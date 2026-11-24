<script lang="ts">
  let { text, tone = 'neutral' }: { text: string; tone?: 'neutral' | 'accent' | 'in' | 'out' } =
    $props();
</script>

<span class="pill {tone}">{text}</span>

<style>
  .pill {
    display: inline-block;
    padding: 0 0.55rem;
    border: 1px solid currentColor;
    border-radius: 1rem;
    font-size: 0.85rem;
    line-height: 1.5;
    white-space: nowrap;
    margin-right: 0.4rem;
  }
  .neutral {
    color: var(--muted);
  }
  .accent {
    color: var(--accent);
  }
  .in {
    color: var(--in);
  }
  .out {
    color: var(--out);
  }
</style>
