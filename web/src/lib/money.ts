// Money arrives from the API as integer cents. It is formatted only here,
// for Portuguese readers: 123456 becomes "1 234,56 €".
// pt-PT only groups digits from five digits up by default (1234 stays
// "1234", 12 345 gets a space). true (always) groups from four, so every amount is
// grouped the same way.
const euro = new Intl.NumberFormat('pt-PT', {
  style: 'currency',
  currency: 'EUR',
  useGrouping: true,
});
const percent = new Intl.NumberFormat('pt-PT', { style: 'percent', maximumFractionDigits: 0 });

const euroWhole = new Intl.NumberFormat('pt-PT', {
  style: 'currency',
  currency: 'EUR',
  maximumFractionDigits: 0,
  useGrouping: true,
});

/** Formats cents as whole euros, for axis labels: 123456 becomes "1 235 €". */
export function formatWholeEuros(cents: number): string {
  return euroWhole.format(cents / 100);
}

/** Formats an amount in cents as euros. */
export function formatCents(cents: number): string {
  return euro.format(cents / 100);
}

/** Formats a whole percentage such as 42 as "42%". */
export function formatPercent(pct: number): string {
  return percent.format(pct / 100);
}
