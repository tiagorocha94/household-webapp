<script lang="ts" generics="V extends string">
  /** One choice out of a few, as a row of buttons. The chosen one is marked with aria-pressed. */
  interface Option {
    value: V;
    label: string;
    /** The language of the label, when it differs from the page's (for example "PT"). */
    lang?: string;
    /** A colour shown before the label, for a choice of colours. */
    swatch?: string;
  }
  let {
    label,
    options,
    value,
    onchange,
  }: { label: string; options: Option[]; value: V; onchange: (value: V) => void } = $props();
</script>

<div class="segmented" role="group" aria-label={label}>
  {#each options as option (option.value)}
    <button
      type="button"
      lang={option.lang}
      aria-pressed={value === option.value}
      onclick={() => onchange(option.value)}
    >
      {#if option.swatch}<span class="swatch" style:background={option.swatch} aria-hidden="true"
        ></span>{/if}{option.label}
    </button>
  {/each}
</div>

<style>
  .segmented {
    display: inline-flex;
    flex-wrap: wrap;
    border: 1px solid var(--rule);
    border-radius: 8px;
    overflow: hidden;
  }
  button {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    gap: 0.4rem;
    min-width: 2.75rem;
    min-height: 2.75rem;
    padding: 0 0.8rem;
    border: 0;
    background: transparent;
    color: var(--muted);
    font: inherit;
    font-size: 0.85rem;
    cursor: pointer;
  }
  button[aria-pressed='true'] {
    background: color-mix(in srgb, var(--accent) 14%, transparent);
    color: var(--accent);
    font-weight: 700;
  }
  .swatch {
    width: 0.9rem;
    height: 0.9rem;
    border-radius: 50%;
    border: 1px solid color-mix(in srgb, var(--ink) 30%, transparent);
  }
</style>
