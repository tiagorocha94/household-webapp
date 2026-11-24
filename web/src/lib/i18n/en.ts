import { en as cashflow } from './messages/cashflow';
import { en as goals } from './messages/goals';
import { en as investments } from './messages/investments';
import { en as shell } from './messages/shell';

/** Every text of the app in English; the keys are the keys of every other language. */
export const en = { ...shell, ...cashflow, ...goals, ...investments } as const;

export type MessageKey = keyof typeof en;
