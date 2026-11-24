import createClient from 'openapi-fetch';
import { t } from '../lib/i18n/index.svelte';
import type { components, paths } from './schema';

export type User = components['schemas']['User'];
export type SavingsAccount = components['schemas']['SavingsAccount'];
export type SavingsMovement = components['schemas']['SavingsMovement'];
export type SavingsSnapshot = components['schemas']['SavingsSnapshot'];
export type Goal = components['schemas']['Goal'];
export type Movement = components['schemas']['Movement'];
export type Projection = components['schemas']['Projection'];
export type Contract = components['schemas']['Contract'];
export type Payment = components['schemas']['Payment'];
export type InvestmentEntity = components['schemas']['InvestmentEntity'];
export type InvestmentInstrument = components['schemas']['InvestmentInstrument'];
export type InvestmentContribution = components['schemas']['InvestmentContribution'];
export type InvestmentSnapshot = components['schemas']['InvestmentSnapshot'];
export type CashflowCategory = components['schemas']['CashflowCategory'];
export type CashflowMonth = components['schemas']['CashflowMonth'];
export type CashflowMonthDetail = components['schemas']['CashflowMonthDetail'];
export type CashflowCategoryTotal = components['schemas']['CashflowCategoryTotal'];
export type CashflowGroupTotal = components['schemas']['CashflowGroupTotal'];
export type CashflowUserTotal = components['schemas']['CashflowUserTotal'];
export type NetWorthMonth = components['schemas']['NetWorthMonth'];
export type CashflowUserMonth = components['schemas']['CashflowUserMonth'];
export type CashflowMovement = components['schemas']['CashflowMovement'];
export type Overview = components['schemas']['Overview'];
export type OverviewAttention = components['schemas']['OverviewAttention'];
export type OverviewCashflow = components['schemas']['OverviewCashflow'];
export type OverviewGoal = components['schemas']['OverviewGoal'];
export type OverviewContract = components['schemas']['OverviewContract'];
export type OverviewSavings = components['schemas']['OverviewSavings'];
export type OverviewSavingsAccount = components['schemas']['OverviewSavingsAccount'];

/** An error answer from the API, or a failure to reach it (status 0). */
export class ApiError extends Error {
  readonly status: number;
  readonly retryAfterSeconds: number | null;

  constructor(status: number, message: string, retryAfterSeconds: number | null = null) {
    super(message);
    this.name = 'ApiError';
    this.status = status;
    this.retryAfterSeconds = retryAfterSeconds;
  }
}

/** A message for the person using the app, without technical detail. */
export function describeError(error: unknown): string {
  if (!(error instanceof ApiError)) return t('shell.error.generic');
  switch (error.status) {
    case 0:
      return t('shell.error.unreachable');
    case 403:
      return t('shell.error.forbidden');
    case 404:
      return t('shell.error.notFound');
    default:
      return t('shell.error.generic');
  }
}

const client = createClient<paths>({ baseUrl: '/api/v1' });

let unauthorized: (() => void) | undefined;

/** Sets what happens when a request that needs a session is answered with 401. */
export function onUnauthorized(handler: () => void): void {
  unauthorized = handler;
}

client.use({
  onResponse({ request, response }) {
    const path = new URL(request.url).pathname;
    if (response.status === 401 && !path.endsWith('/auth/login') && !path.endsWith('/auth/me')) {
      unauthorized?.();
    }
    return undefined;
  },
});

interface Result<T> {
  data?: T;
  error?: unknown;
  response: Response;
}

/** Returns the data of a successful call, or throws an ApiError. */
async function unwrap<T>(call: Promise<Result<T>>): Promise<T> {
  let result: Result<T>;
  try {
    result = await call;
  } catch {
    throw new ApiError(0, 'network error');
  }
  if (result.data === undefined) throw failure(result);
  return result.data;
}

function failure(result: Result<unknown>): ApiError {
  const body = result.error as { error?: string } | undefined;
  const retry = Number(result.response.headers.get('Retry-After'));
  return new ApiError(
    result.response.status,
    body?.error ?? result.response.statusText,
    Number.isFinite(retry) && retry > 0 ? retry : null,
  );
}

export const api = {
  info: () => unwrap(client.GET('/info')),
  me: () => unwrap(client.GET('/auth/me')),
  login: (username: string, password: string) =>
    unwrap(client.POST('/auth/login', { body: { username, password } })),
  overview: () => unwrap(client.GET('/overview')),
  netWorth: () => unwrap(client.GET('/overview/net-worth')),
  accounts: () => unwrap(client.GET('/goals/accounts')),
  account: (id: number) => unwrap(client.GET('/goals/accounts/{id}', { params: { path: { id } } })),
  accountGoals: (id: number) =>
    unwrap(client.GET('/goals/accounts/{id}/goals', { params: { path: { id } } })),
  accountMovements: (id: number) =>
    unwrap(client.GET('/goals/accounts/{id}/movements', { params: { path: { id } } })),
  projection: (id: number) =>
    unwrap(client.GET('/goals/accounts/{id}/projection', { params: { path: { id } } })),
  goal: (id: number) => unwrap(client.GET('/goals/{id}', { params: { path: { id } } })),
  savings: () => unwrap(client.GET('/savings/accounts')),
  savingsAccount: (id: number) =>
    unwrap(client.GET('/savings/accounts/{id}', { params: { path: { id } } })),
  savingsMovements: (id: number) =>
    unwrap(client.GET('/savings/accounts/{id}/movements', { params: { path: { id } } })),
  savingsSnapshots: (id: number) =>
    unwrap(client.GET('/savings/accounts/{id}/snapshots', { params: { path: { id } } })),
  credits: () => unwrap(client.GET('/credits')),
  contract: (id: number) => unwrap(client.GET('/credits/{id}', { params: { path: { id } } })),
  contractPayments: (id: number) =>
    unwrap(client.GET('/credits/{id}/payments', { params: { path: { id } } })),
  investments: () => unwrap(client.GET('/investments/entities')),
  entity: (id: number) =>
    unwrap(client.GET('/investments/entities/{id}', { params: { path: { id } } })),
  entityInstruments: (id: number) =>
    unwrap(client.GET('/investments/entities/{id}/instruments', { params: { path: { id } } })),
  instrument: (id: number) =>
    unwrap(client.GET('/investments/instruments/{id}', { params: { path: { id } } })),
  instrumentContributions: (id: number) =>
    unwrap(client.GET('/investments/instruments/{id}/contributions', { params: { path: { id } } })),
  instrumentSnapshots: (id: number) =>
    unwrap(client.GET('/investments/instruments/{id}/snapshots', { params: { path: { id } } })),
  cashflowCategories: () => unwrap(client.GET('/cashflow/categories')),
  cashflowMonths: () => unwrap(client.GET('/cashflow/months')),
  cashflowMonth: (month: string) =>
    unwrap(client.GET('/cashflow/months/{month}', { params: { path: { month } } })),
  cashflowMovements: (month: string) =>
    unwrap(client.GET('/cashflow/months/{month}/movements', { params: { path: { month } } })),
  async logout(): Promise<void> {
    let result;
    try {
      result = await client.POST('/auth/logout');
    } catch {
      throw new ApiError(0, 'network error');
    }
    if (!result.response.ok) throw failure(result);
  },
};
