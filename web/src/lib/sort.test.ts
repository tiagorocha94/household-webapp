import { describe, expect, it } from 'vitest';
import { ariaSort, sortRows, toggleSort, type Sort } from './sort';

type Column = 'name' | 'amount';
const values = {
  name: (r: { name: string; amount: number }) => r.name,
  amount: (r: { name: string; amount: number }) => r.amount,
};
const rows = [
  { name: 'b', amount: 5 },
  { name: 'A', amount: 20 },
  { name: 'c', amount: 5 },
];

describe('toggleSort', () => {
  it('goes ascending, descending, then back to no sort', () => {
    const first = toggleSort<Column>(null, 'name');
    expect(first).toEqual({ column: 'name', direction: 'ascending' });
    const second = toggleSort(first, 'name');
    expect(second).toEqual({ column: 'name', direction: 'descending' });
    expect(toggleSort(second, 'name')).toBeNull();
  });
  it('starts a new column ascending', () => {
    const sort: Sort<Column> = { column: 'name', direction: 'descending' };
    expect(toggleSort(sort, 'amount')).toEqual({ column: 'amount', direction: 'ascending' });
  });
});

describe('ariaSort', () => {
  it('reports the direction only for the sorted column', () => {
    const sort: Sort<Column> = { column: 'name', direction: 'descending' };
    expect(ariaSort(sort, 'name')).toBe('descending');
    expect(ariaSort(sort, 'amount')).toBe('none');
    expect(ariaSort(null, 'name')).toBe('none');
  });
});

describe('sortRows', () => {
  it('keeps the own order without a sort, and does not change the input', () => {
    const copy = sortRows(rows, null, values);
    expect(copy).toEqual(rows);
    expect(copy).not.toBe(rows);
  });
  it('sorts text ignoring case', () => {
    const sorted = sortRows(rows, { column: 'name', direction: 'ascending' }, values);
    expect(sorted.map((r) => r.name)).toEqual(['A', 'b', 'c']);
  });
  it('sorts numbers as numbers, not text', () => {
    const sorted = sortRows(rows, { column: 'amount', direction: 'descending' }, values);
    expect(sorted.map((r) => r.amount)).toEqual([20, 5, 5]);
  });
  it('keeps equal values in their own order, in either direction', () => {
    const up = sortRows(rows, { column: 'amount', direction: 'ascending' }, values);
    const down = sortRows(rows, { column: 'amount', direction: 'descending' }, values);
    expect(up.map((r) => r.name)).toEqual(['b', 'c', 'A']);
    expect(down.map((r) => r.name)).toEqual(['A', 'b', 'c']);
  });
  it('orders embedded numbers naturally', () => {
    const items = [{ name: 'Loan 10' }, { name: 'Loan 9' }];
    const sorted = sortRows(
      items,
      { column: 'name', direction: 'ascending' },
      { name: (r) => r.name },
    );
    expect(sorted.map((r) => r.name)).toEqual(['Loan 9', 'Loan 10']);
  });
});
