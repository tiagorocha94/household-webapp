// The time a page's data was read, for the "Updated" line.
import { localeTag, t, type MessageKey } from './i18n/index.svelte';

/** "Updated 19:01" for a moment in the viewer's own time zone. */
export function updatedText(at: Date): string {
  const clock = new Intl.DateTimeFormat(localeTag(), { hour: '2-digit', minute: '2-digit' });
  return t('shell.common.updated', { time: clock.format(at) });
}

/** "Reading your goals…" for the thing a page is loading. */
export function readingText(what: string): string {
  return t('shell.reading', { what: whatLabel(what) });
}

/** The nouns a page passes to Async, with the text of each. */
const WHAT: Record<string, MessageKey> = {
  goals: 'shell.what.goals',
  goal: 'shell.what.goal',
  'goal accounts': 'shell.what.goalAccounts',
  notifications: 'shell.what.notifications',
  months: 'shell.what.months',
  month: 'shell.what.month',
  overview: 'shell.what.overview',
  credit: 'shell.what.credit',
  credits: 'shell.what.credits',
  'savings account': 'shell.what.savingsAccount',
  savings: 'shell.what.savings',
  investments: 'shell.what.investments',
  'investment account': 'shell.what.investmentAccount',
  instrument: 'shell.what.instrument',
  'net worth': 'shell.what.netWorth',
};

/** The noun in the current language; a noun that is not known is shown as it is. */
export function whatLabel(what: string): string {
  return Object.prototype.hasOwnProperty.call(WHAT, what) ? t(WHAT[what]) : what;
}
