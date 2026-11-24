// How investments are labelled, totalled and charted. Pure, so it is tested
// without a browser.
import type {
  InvestmentContribution,
  InvestmentEntity,
  InvestmentInstrument,
  InvestmentSnapshot,
} from '../api/client';
import { t } from './i18n/index.svelte';
import { monthsBetween } from './month';

// Getters, so the text follows the language when it is read, not when the module loads.
export const kindLabel: Record<InvestmentEntity['kind'], string> = {
  get bank() {
    return t('invest.kind.bank');
  },
  get broker() {
    return t('invest.kind.broker');
  },
  get investment_firm() {
    return t('invest.kind.investment_firm');
  },
  get state() {
    return t('invest.kind.state');
  },
  get other() {
    return t('invest.kind.other');
  },
};

export const typeLabel: Record<InvestmentInstrument['type'], string> = {
  get etf() {
    return t('invest.type.etf');
  },
  get fund() {
    return t('invest.type.fund');
  },
  get ppr() {
    return t('invest.type.ppr');
  },
  get certificados_aforro() {
    return t('invest.type.certificados_aforro');
  },
  get other() {
    return t('invest.type.other');
  },
};

/** Who a holding belongs to: the person's username, or the household when unowned. */
export function ownerLabel(user: string | null): string {
  return user ?? t('invest.owner.household');
}

const gain = new Intl.NumberFormat('pt-PT', {
  style: 'percent',
  minimumFractionDigits: 2,
  maximumFractionDigits: 2,
  signDisplay: 'exceptZero',
});

/** Formats the API's gainPct, a percentage scaled by 100: 734 becomes "+7,34%". */
export function formatGainPct(gainPct: number): string {
  return gain.format(gainPct / 10000);
}

/** Whether a gain or loss is a gain, a loss, or neither, for colouring. */
export function gainTone(gainCents: number): 'in' | 'out' | undefined {
  if (gainCents > 0) return 'in';
  if (gainCents < 0) return 'out';
  return undefined;
}

export interface Totals {
  allocatedCents: number;
  currentValueCents: number;
  gainCents: number;
  /** Scaled by 100 like the API: 734 means +7.34%. */
  gainPct: number;
}

/** Adds up entities or instruments; the percentage is derived from the sums. */
export function totals(
  rows: readonly Pick<InvestmentEntity, 'allocatedCents' | 'currentValueCents'>[],
): Totals {
  const allocatedCents = rows.reduce((sum, r) => sum + r.allocatedCents, 0);
  const currentValueCents = rows.reduce((sum, r) => sum + r.currentValueCents, 0);
  const gainCents = currentValueCents - allocatedCents;
  const gainPct = allocatedCents > 0 ? Math.trunc((gainCents * 10000) / allocatedCents) : 0;
  return { allocatedCents, currentValueCents, gainCents, gainPct };
}

/** Each entity's share of the money put in, 0 to 100, largest first. */
export function allocationShares<T extends Pick<InvestmentEntity, 'name' | 'allocatedCents'>>(
  entities: readonly T[],
): { entity: T; sharePct: number }[] {
  const total = entities.reduce((sum, e) => sum + e.allocatedCents, 0);
  return entities
    .map((entity) => ({ entity, sharePct: total > 0 ? (entity.allocatedCents * 100) / total : 0 }))
    .sort(
      (a, b) =>
        b.entity.allocatedCents - a.entity.allocatedCents ||
        a.entity.name.localeCompare(b.entity.name),
    );
}

/** Instruments by current value, largest first, then by name. */
export function sortInstruments(
  instruments: readonly InvestmentInstrument[],
): InvestmentInstrument[] {
  return [...instruments].sort(
    (a, b) => b.currentValueCents - a.currentValueCents || a.name.localeCompare(b.name),
  );
}

export interface HistoryPoint {
  month: string;
  /** Money put in up to and including this month. */
  allocatedCents: number;
  /**
   * The observed value, or an estimate in a month without a snapshot that lies
   * between two snapshots. Null before the first snapshot and after the last.
   */
  valueCents: number | null;
  /** True when valueCents is interpolated rather than observed. */
  estimated: boolean;
}

/**
 * One point per calendar month from the first contribution or snapshot to the
 * last, oldest first, so gaps keep their real width. Allocated is cumulative.
 * The value is the month's snapshot (the highest id wins within a month, as in
 * the API). A month without one between two snapshots is estimated on the
 * straight line between them, rounded to whole cents; before the first
 * snapshot and after the last there is no value, because nothing is known.
 */
export function buildHistory(
  contributions: readonly Pick<InvestmentContribution, 'month' | 'amountCents'>[],
  snapshots: readonly Pick<InvestmentSnapshot, 'id' | 'month' | 'valueCents'>[],
): HistoryPoint[] {
  const used = [...contributions.map((c) => c.month), ...snapshots.map((s) => s.month)].sort();
  if (used.length === 0) return [];
  const months = monthsBetween(used[0], used[used.length - 1]);

  const observed = new Map<string, number>();
  for (const s of [...snapshots].sort((a, b) => a.month.localeCompare(b.month) || a.id - b.id)) {
    observed.set(s.month, s.valueCents);
  }

  const known = months.flatMap((month, i) => (observed.has(month) ? [i] : []));
  const values: (number | null)[] = months.map((month) => observed.get(month) ?? null);
  for (let k = 0; k + 1 < known.length; k++) {
    const from = known[k];
    const to = known[k + 1];
    const start = values[from] as number;
    const end = values[to] as number;
    for (let i = from + 1; i < to; i++) {
      values[i] = start + Math.round(((end - start) * (i - from)) / (to - from));
    }
  }

  const sortedContributions = [...contributions].sort((a, b) => a.month.localeCompare(b.month));
  let allocated = 0;
  let c = 0;
  return months.map((month, i) => {
    while (c < sortedContributions.length && sortedContributions[c].month <= month) {
      allocated += sortedContributions[c++].amountCents;
    }
    return {
      month,
      allocatedCents: allocated,
      valueCents: values[i],
      estimated: values[i] !== null && !observed.has(month),
    };
  });
}
