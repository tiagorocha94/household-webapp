import type { MessageKey } from './en';
import { pt as cashflow } from './messages/cashflow';
import { pt as goals } from './messages/goals';
import { pt as investments } from './messages/investments';
import { pt as shell } from './messages/shell';

/** Every text of the app in Portuguese (Portugal). */
export const pt: Record<MessageKey, string> = { ...shell, ...cashflow, ...goals, ...investments };
