<script lang="ts" generics="K extends string">
  import { ariaSort, type Sort } from '../lib/sort';

  let {
    label,
    column,
    sort,
    onsort,
    numeric = false,
  }: {
    label: string;
    column: K;
    sort: Sort<K> | null;
    onsort: (column: K) => void;
    numeric?: boolean;
  } = $props();

  const direction = $derived(ariaSort(sort, column));
</script>

<th class:num={numeric} aria-sort={direction}>
  <button type="button" class="sort" class:numeric onclick={() => onsort(column)}>
    {label}<span class="arrow" aria-hidden="true"
      >{direction === 'ascending' ? '▲' : direction === 'descending' ? '▼' : ''}</span
    >
  </button>
</th>

<style>
  .sort {
    position: relative;
    padding: 0;
    border: 0;
    background: none;
    color: inherit;
    font-weight: inherit;
    white-space: nowrap;
  }
  .sort:hover {
    color: var(--ink);
  }
  th[aria-sort='ascending'] .sort,
  th[aria-sort='descending'] .sort {
    color: var(--ink);
  }
  /* Beside the label, outside the text flow, so headers stay aligned with their figures. */
  .arrow {
    position: absolute;
    top: 50%;
    transform: translateY(-50%);
    left: 100%;
    margin-left: 0.3em;
    font-size: 0.7em;
  }
  .numeric .arrow {
    left: auto;
    right: 100%;
    margin: 0 0.3em 0 0;
  }
</style>
