// Months arrive from the API as "YYYY-MM". They are formatted and compared only
// here. Canonical months sort chronologically as text.
import { getLocale, localeTag } from './i18n/index.svelte';

// The formats are made per call, so they follow the language chosen at that moment.
// Portuguese would give "02/2026" and "5/02" for these formats, so the short month name, which it
// writes as "fev.", is put next to the year or the day by hand.
const shortMonth = () => new Intl.DateTimeFormat(localeTag(), { month: 'short', timeZone: 'UTC' });
const label = {
  format: (date: Date): string =>
    getLocale() === 'pt'
      ? `${shortMonth().format(date)} ${date.getUTCFullYear()}`
      : new Intl.DateTimeFormat(localeTag(), {
          month: 'short',
          year: 'numeric',
          timeZone: 'UTC',
        }).format(date),
};

const pattern = /^(\d{4})-(0[1-9]|1[0-2])$/;

/** Formats "2026-10" as "Oct 2026". Anything else is returned unchanged. */
export function formatMonth(month: string): string {
  const match = pattern.exec(month);
  if (!match) return month;
  return label.format(new Date(Date.UTC(Number(match[1]), Number(match[2]) - 1, 1)));
}

/** Every month from first to last inclusive, as "YYYY-MM"; empty when last is before first. */
export function monthsBetween(first: string, last: string): string[] {
  const a = pattern.exec(first);
  const b = pattern.exec(last);
  if (!a || !b) return [];
  const months: string[] = [];
  let year = Number(a[1]);
  let month = Number(a[2]);
  const endYear = Number(b[1]);
  const endMonth = Number(b[2]);
  while (year < endYear || (year === endYear && month <= endMonth)) {
    months.push(`${year}-${String(month).padStart(2, '0')}`);
    month += 1;
    if (month > 12) {
      month = 1;
      year += 1;
    }
  }
  return months;
}

/** Returns the current local month as "YYYY-MM", like the server does. */
export function currentMonth(now: Date = new Date()): string {
  return `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}`;
}

/** Whether value is a canonical "YYYY-MM" month, for validating addresses. */
export function isMonth(value: string): boolean {
  return pattern.test(value);
}

/** The month before month, as "YYYY-MM"; empty when month is not a month. */
export function previousMonth(month: string): string {
  const match = pattern.exec(month);
  if (!match) return '';
  const year = Number(match[1]);
  const number = Number(match[2]);
  return number === 1 ? `${year - 1}-12` : `${year}-${String(number - 1).padStart(2, '0')}`;
}

const dayLabel = {
  format: (date: Date): string =>
    getLocale() === 'pt'
      ? `${date.getUTCDate()} ${shortMonth().format(date)}`
      : new Intl.DateTimeFormat(localeTag(), {
          day: 'numeric',
          month: 'short',
          timeZone: 'UTC',
        }).format(date),
};

const dayPattern = /^(\d{4})-(0[1-9]|1[0-2])-(0[1-9]|[12]\d|3[01])$/;

/** Formats "2026-10-05" as "5 Oct". Anything else is returned unchanged. */
export function formatDay(day: string): string {
  const match = dayPattern.exec(day);
  if (!match) return day;
  return dayLabel.format(
    new Date(Date.UTC(Number(match[1]), Number(match[2]) - 1, Number(match[3]))),
  );
}
