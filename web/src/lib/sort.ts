// Click-to-sort for table columns. Pure, so it is tested without a browser.

export type Direction = 'ascending' | 'descending';

/** The column a table is sorted by and which way, or null for its own order. */
export interface Sort<K extends string> {
  column: K;
  direction: Direction;
}

/** What a click on a column header does: ascending, then descending, then back to the table's own order. */
export function toggleSort<K extends string>(current: Sort<K> | null, column: K): Sort<K> | null {
  if (current?.column !== column) return { column, direction: 'ascending' };
  return current.direction === 'ascending' ? { column, direction: 'descending' } : null;
}

/** The value of the header's aria-sort attribute. */
export function ariaSort<K extends string>(sort: Sort<K> | null, column: K): Direction | 'none' {
  return sort?.column === column ? sort.direction : 'none';
}

const collator = new Intl.Collator('pt-PT', { numeric: true, sensitivity: 'base' });

/**
 * The rows ordered by the sort, or in their own order for null. Equal values
 * keep their own order, so sorting by one column never scrambles another.
 */
export function sortRows<T, K extends string>(
  rows: readonly T[],
  sort: Sort<K> | null,
  values: Record<K, (row: T) => string | number>,
): T[] {
  if (sort === null) return [...rows];
  const valueOf = values[sort.column];
  const sign = sort.direction === 'ascending' ? 1 : -1;
  return rows
    .map((row, index) => ({ row, index, value: valueOf(row) }))
    .sort((a, b) => {
      const order =
        typeof a.value === 'number' && typeof b.value === 'number'
          ? a.value - b.value
          : collator.compare(String(a.value), String(b.value));
      return order * sign || a.index - b.index;
    })
    .map(({ row }) => row);
}
