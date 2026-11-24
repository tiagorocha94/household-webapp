import { afterEach, beforeEach, describe, expect, it } from 'vitest';
import { ApiError, describeError } from '../api/client';
import { setLocale } from './i18n/index.svelte';
import { formatDay, formatMonth } from './month';
import { NAV, navLabel } from './nav';
import { badgeText, groupNotifications, reviewBreakdown, reviewHeadline } from './notifications';
import { describeAttention, expenseNote, savingsAccountNote, savingsNote } from './overview';
import { readingText, updatedText, whatLabel } from './updated';

beforeEach(() => setLocale('pt'));
afterEach(() => setLocale('en'));

const attention = (kind: string, name = '') =>
  ({ kind, name, month: '2026-11', amountCents: 10000, pct: 114 }) as never;

describe('the shell in Portuguese', () => {
  it('formats months and days in Portuguese', () => {
    expect(formatMonth('2026-02')).toBe('fev. 2026');
    expect(formatDay('2026-10-05')).toBe('5 out.');
  });

  it('translates the menu', () => {
    expect(NAV.map(navLabel)).toEqual([
      'Visão geral',
      'Cashflow',
      'Património líquido',
      'Objetivos',
      'Poupança',
      'Créditos',
      'Investimentos',
    ]);
  });

  it('translates the Updated line and what is being read', () => {
    expect(updatedText(new Date(2026, 10, 21, 19, 1))).toBe('Atualizado às 19:01');
    expect(readingText('goals')).toBe('A ler os seus objetivos…');
    expect(whatLabel('something else')).toBe('something else');
  });

  it('translates error messages', () => {
    expect(describeError(new ApiError(404, 'x'))).toBe('Isto não existe.');
    expect(describeError(new ApiError(403, 'x'))).toBe('Não tem acesso a isto.');
    expect(describeError(new Error('x'))).toBe('Algo correu mal. Tente de novo.');
  });

  it('translates the notifications', () => {
    const groups = groupNotifications(
      [attention('credit_unpaid', 'Car loan'), attention('budget_over', 'Fun')],
      '2026-11',
    );
    expect(groups.map((g) => g.title)).toEqual([
      'Introduzir ou corrigir dados',
      'Plano e orçamento',
    ]);
    expect(groups[0].lines[0].line.linkLabel).toBe('Abrir créditos');
    expect(reviewHeadline([attention('budget_over')])).toBe('1 coisa para rever');
    expect(reviewHeadline([attention('budget_over'), attention('credit_unpaid')])).toBe(
      '2 coisas para rever',
    );
    expect(reviewBreakdown([attention('credit_unpaid'), attention('budget_over')])).toBe(
      'Créditos 1 · Objetivos e orçamento 1',
    );
    expect(badgeText(12)).toBe('9+');
  });

  it('translates the overview sentences', () => {
    expect(describeAttention(attention('unassigned_goals'), '2026-11').topic).toBe('Objetivos');
    expect(expenseNote({ expenseCents: 5000, previousExpenseCents: 5000 })).toBe(
      'Igual ao mês passado até agora',
    );
    expect(savingsNote({ balanceCents: 5000, targetCents: 10000, interestCents: 100 })).toContain(
      'de meta',
    );
    expect(
      savingsAccountNote({ owner: null, targetCents: null, progressPct: 0, interestCents: 0 }),
    ).toBe('Casa · sem meta definida');
  });
});
