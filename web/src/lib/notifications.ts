// How the things that need attention are grouped on the Notifications page and
// counted on Home. Pure, so it is tested without a browser.
import type { OverviewAttention } from '../api/client';
import { t, type MessageKey } from './i18n/index.svelte';
import { describeAttention, type AttentionLine } from './overview';

export interface NotificationGroup {
  title: string;
  lines: { key: string; line: AttentionLine }[];
}

const DATA_KINDS = [
  'credit_unpaid',
  'unassigned_savings',
  'unassigned_investments',
  'unassigned_goals',
  'unassigned_credits',
];

/** The things to look at, grouped by what to do about them. Empty groups are left out. */
export function groupNotifications(
  items: OverviewAttention[],
  current: string,
): NotificationGroup[] {
  const groups: NotificationGroup[] = [
    { title: t('shell.notifications.groupData'), lines: [] },
    { title: t('shell.notifications.groupPlan'), lines: [] },
  ];
  for (const item of items) {
    const group = DATA_KINDS.includes(item.kind) ? groups[0] : groups[1];
    group.lines.push({
      key: item.kind + item.name + item.month,
      line: describeAttention(item, current),
    });
  }
  return groups.filter((g) => g.lines.length > 0);
}

const TOPICS: { label: MessageKey; kinds: string[] }[] = [
  {
    label: 'shell.review.topicUnassigned',
    kinds: ['unassigned_savings', 'unassigned_investments', 'unassigned_goals'],
  },
  { label: 'shell.nav.credits', kinds: ['credit_unpaid', 'unassigned_credits'] },
  { label: 'shell.review.topicGoalsBudget', kinds: ['goals_shortfall', 'budget_over'] },
];

/** "5 things to review", or null when there is nothing. */
export function reviewHeadline(items: OverviewAttention[]): string | null {
  if (items.length === 0) return null;
  return items.length === 1
    ? t('shell.review.headlineOne')
    : t('shell.review.headlineOther', { count: items.length });
}

/** How many of the things to review are about each topic, such as "Credits 1 · Goals and budget 2". */
export function reviewBreakdown(items: OverviewAttention[]): string {
  return TOPICS.map((topic) => ({
    label: t(topic.label),
    n: items.filter((i) => topic.kinds.includes(i.kind)).length,
  }))
    .filter((topic) => topic.n > 0)
    .map((topic) => `${topic.label} ${topic.n}`)
    .join(' · ');
}

/** The number drawn on the bell: a single digit up to 9, then "9+". */
export function badgeText(count: number): string {
  return count > 9 ? '9+' : String(count);
}
