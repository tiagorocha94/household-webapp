// Sample data for the mock API, typed against the API contract so a change to
// docs/architecture/openapi.yaml shows up here as a type error. Months are relative to
// today, so "planned", "active" and "completed" goals stay that way.
import type { components } from '../src/api/schema';

type Schemas = components['schemas'];
type Goal = Schemas['Goal'];
type Movement = Schemas['Movement'];
type Contract = Schemas['Contract'];
type Payment = Schemas['Payment'];

export const users = [
  { id: 1, username: 'alex' },
  { id: 2, username: 'maria' },
];

/** A GET route: handle returns the body, or null for 404. */
export interface MockRoute {
  pattern: RegExp;
  handle: (ids: number[]) => unknown;
}

/** The month `offset` months from now, as "YYYY-MM". */
function month(offset: number): string {
  const d = new Date();
  d.setDate(1);
  d.setMonth(d.getMonth() + offset);
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}`;
}

function monthsBetween(start: string, end: string): number {
  const [sy, sm] = start.split('-').map(Number);
  const [ey, em] = end.split('-').map(Number);
  return (ey - sy) * 12 + (em - sm) + 1;
}

const accounts: Schemas['Account'][] = [
  { id: 1, name: 'Household', monthlyDepositCents: 36600 },
  { id: 2, name: 'Personal', monthlyDepositCents: 20000 },
  { id: 3, name: 'Holiday pot', monthlyDepositCents: 0 },
];

interface GoalSeed {
  id: number;
  accountId: number;
  name: string;
  targetCents: number;
  start: number;
  end: number;
  status: Goal['status'];
}

const goalSeeds: GoalSeed[] = [
  {
    id: 1,
    accountId: 1,
    name: 'Summer holiday',
    targetCents: 240000,
    start: -8,
    end: 3,
    status: 'active',
  },
  {
    id: 2,
    accountId: 1,
    name: 'Emergency fund',
    targetCents: 600000,
    start: -11,
    end: 12,
    status: 'active',
  },
  { id: 3, accountId: 1, name: 'New sofa', targetCents: 90000, start: 2, end: 7, status: 'active' },
  {
    id: 4,
    accountId: 1,
    name: 'Boiler repair',
    targetCents: 45000,
    start: -10,
    end: -4,
    status: 'completed',
  },
  { id: 5, accountId: 2, name: 'Laptop', targetCents: 120000, start: -3, end: 8, status: 'active' },
];

const movements: Movement[] = [];
let movementId = 1;

function add(
  accountId: number,
  goalId: number | null,
  goalName: string,
  type: Movement['type'],
  amountCents: number,
  at: string,
  description = '',
) {
  movements.push({
    id: movementId++,
    accountId,
    goalId,
    goalName,
    type,
    amountCents,
    month: at,
    description,
    createdAt: '2026-01-01T00:00:00Z',
    accountInflow: type === 'deposit' || type === 'release',
    goalInflow: type === 'allocation' || type === 'cover',
  });
}

for (const accountId of [1, 2]) {
  for (let offset = -12; offset <= 0; offset++) {
    add(
      accountId,
      null,
      '',
      'deposit',
      accountId === 1 ? 90000 : 40000,
      month(offset),
      'Monthly deposit',
    );
  }
}

const goals: Goal[] = goalSeeds.map((seed) => {
  const start = month(seed.start);
  const end = month(seed.end);
  const total = monthsBetween(start, end);
  const per = Math.ceil(seed.targetCents / total);
  let allocated = 0;
  for (let offset = seed.start; offset <= Math.min(0, seed.end); offset++) {
    const amount =
      seed.status === 'completed' && offset === seed.end ? seed.targetCents - allocated : per;
    add(seed.accountId, seed.id, seed.name, 'allocation', amount, month(offset));
    allocated += amount;
  }
  let expenses = 0;
  if (seed.id === 4) {
    expenses = 38000;
    add(seed.accountId, seed.id, seed.name, 'expense', expenses, month(-4), 'Plumber invoice');
    add(
      seed.accountId,
      seed.id,
      seed.name,
      'release',
      allocated - expenses,
      month(-4),
      'Goal completed: remaining balance',
    );
  }
  if (seed.id === 1) {
    expenses = 60000;
    add(seed.accountId, seed.id, seed.name, 'expense', expenses, month(-1), 'Flights');
  }
  const released = seed.id === 4 ? allocated - expenses : 0;
  const balance = allocated - expenses - released;
  return {
    id: seed.id,
    accountId: seed.accountId,
    name: seed.name,
    targetCents: seed.targetCents,
    startMonth: start,
    endMonth: end,
    status: seed.status,
    totalMonths: total,
    elapsedMonths: Math.max(0, Math.min(total, monthsBetween(start, month(0)))),
    monthlyAllocationCents: per,
    allocatedCents: allocated,
    expensesCents: expenses,
    balanceCents: balance,
    closingBalanceCents: seed.id === 4 ? released : null,
    progressPct: Math.round((allocated * 100) / seed.targetCents),
  };
});

/** Newest month first, and within a month the latest event first, like the API. */
function ledger(rows: Movement[]): Movement[] {
  const order = { release: 0, cover: 0, expense: 1, allocation: 2, deposit: 3 };
  return [...rows].sort(
    (a, b) => b.month.localeCompare(a.month) || order[a.type] - order[b.type] || b.id - a.id,
  );
}

function sum(rows: Movement[], type: Movement['type']): number {
  return rows.filter((m) => m.type === type).reduce((total, m) => total + m.amountCents, 0);
}

function balance(accountId: number): Schemas['Balance'] {
  const rows = movements.filter((m) => m.accountId === accountId);
  const deposits = sum(rows, 'deposit');
  const allocated = sum(rows, 'allocation');
  const expenses = sum(rows, 'expense');
  const released = sum(rows, 'release');
  const covered = sum(rows, 'cover');
  return {
    depositsCents: deposits,
    allocatedCents: allocated,
    expensesCents: expenses,
    releasedCents: released,
    coveredCents: covered,
    balanceCents: deposits - expenses,
    unallocatedCents: deposits - allocated + released - covered,
  };
}

function projection(accountId: number): Schemas['Projection'] {
  const monthlyDeposit = accounts.find((a) => a.id === accountId)?.monthlyDepositCents ?? 0;
  const own = goals.filter((g) => g.accountId === accountId && g.status === 'active');
  const lastEnd = own.reduce((max, g) => (g.endMonth > max ? g.endMonth : max), month(6));
  const months: Schemas['Projection']['months'] = [];
  let running = 0;
  let unallocated = 0;
  for (let offset = -12; month(offset) <= lastEnd; offset++) {
    const at = month(offset);
    const rows = movements.filter((m) => m.accountId === accountId && m.month === at);
    const future = offset > 0;
    const deposits = future ? monthlyDeposit : sum(rows, 'deposit');
    const allocations = future
      ? own
          .filter((g) => g.startMonth <= at && at <= g.endMonth)
          .reduce((t, g) => t + g.monthlyAllocationCents, 0)
      : sum(rows, 'allocation');
    const expenses = future ? 0 : sum(rows, 'expense');
    const released = future ? 0 : sum(rows, 'release');
    const covered = future ? 0 : sum(rows, 'cover');
    running += deposits - expenses;
    unallocated += deposits - allocations + released - covered;
    months.push({
      month: at,
      depositsCents: deposits,
      allocationsCents: allocations,
      expensesCents: expenses,
      releasedCents: released,
      coveredCents: covered,
      balanceCents: running,
      unallocatedCents: unallocated,
      isProjection: future,
    });
  }
  return { monthlyDepositCents: monthlyDeposit, months };
}

interface ContractSeed {
  id: number;
  name: string;
  category: Contract['category'];
  lender: string;
  principal: number;
  start: number;
  end: number | null;
  payments: number;
  installment: number;
  rate: number;
  notes: string;
}

const contractSeeds: ContractSeed[] = [
  {
    id: 1,
    name: 'Home mortgage',
    category: 'mortgage',
    lender: 'Banco Exemplo',
    principal: 18000000,
    start: -48,
    end: 312,
    payments: 49,
    installment: 71000,
    rate: 0.0028,
    notes: 'Variable rate, reviewed every six months.',
  },
  {
    id: 2,
    name: 'Family car',
    category: 'car',
    lender: 'Auto Finance',
    principal: 1800000,
    start: -30,
    end: 30,
    payments: 31,
    installment: 32000,
    rate: 0.004,
    notes: '',
  },
  {
    id: 3,
    name: 'Washing machine',
    category: 'personal',
    lender: 'Store credit',
    principal: 60000,
    start: -14,
    end: -3,
    payments: 12,
    installment: 5200,
    rate: 0.001,
    notes: 'Paid off early.',
  },
  {
    id: 4,
    name: "Maria's course loan",
    category: 'other',
    lender: 'Study Bank',
    principal: 400000,
    start: -10,
    end: 20,
    payments: 11,
    installment: 14000,
    rate: 0.002,
    notes: '',
  },
];

const payments = new Map<number, Payment[]>();
let paymentId = 1;

const contracts: Contract[] = contractSeeds.map((seed) => {
  const rows: Payment[] = [];
  let outstanding = seed.principal;
  for (let i = 0; i < seed.payments; i++) {
    const interest = Math.round(outstanding * seed.rate);
    const principal =
      seed.id === 3 && i === seed.payments - 1 ? outstanding : seed.installment - interest;
    outstanding -= principal;
    rows.push({
      id: paymentId++,
      month: month(seed.start + i),
      type: 'regular',
      principalCents: principal,
      interestCents: interest,
      totalCents: principal + interest,
      description: '',
      createdAt: '2026-01-01T00:00:00Z',
    });
  }
  if (seed.id === 1) {
    const extra = 500000;
    outstanding -= extra;
    rows.push({
      id: paymentId++,
      month: month(-6),
      type: 'extraordinary',
      principalCents: extra,
      interestCents: 0,
      totalCents: extra,
      description: 'Bonus paid in',
      createdAt: '2026-01-01T00:00:00Z',
    });
  }
  rows.sort((a, b) => b.month.localeCompare(a.month) || b.id - a.id);
  payments.set(seed.id, rows);
  const principalPaid = seed.principal - outstanding;
  const interestPaid = rows.reduce((t, p) => t + p.interestCents, 0);
  return {
    id: seed.id,
    name: seed.name,
    category: seed.category,
    contractNumber: `00${seed.id}-2022`,
    lender: seed.lender,
    originalPrincipalCents: seed.principal,
    startMonth: month(seed.start),
    expectedEndMonth: seed.end === null ? null : month(seed.end),
    status: outstanding <= 0 ? 'paid_off' : 'active',
    notes: seed.notes,
    paymentCount: rows.length,
    principalPaidCents: principalPaid,
    interestPaidCents: interestPaid,
    totalPaidCents: principalPaid + interestPaid,
    outstandingPrincipalCents: outstanding,
    progressPct: Math.floor((principalPaid * 100) / seed.principal),
    lastPaymentMonth: rows[0]?.month ?? null,
    expectedDay: null,
  };
});

// Investments: two entities, three instruments, built from the monthly rows so
// the totals stay consistent with the history.
type InvestmentEntity = Schemas['InvestmentEntity'];
type InvestmentInstrument = Schemas['InvestmentInstrument'];
type InvestmentContribution = Schemas['InvestmentContribution'];
type InvestmentSnapshot = Schemas['InvestmentSnapshot'];

const investmentMonths = ['2026-01', '2026-02', '2026-03', '2026-04', '2026-05', '2026-06'];

const instrumentSeeds = [
  {
    id: 1,
    entityId: 1,
    name: 'World ETF',
    type: 'etf' as const,
    monthly: 20000,
    user: 'alex' as string | null,
    values: [20500, 41500, 63000, 80000, 98000, 123500],
  },
  {
    id: 2,
    entityId: 2,
    name: 'Retirement plan',
    type: 'ppr' as const,
    monthly: 10000,
    user: 'sam' as string | null,
    values: [10000, 20300, 30100, 29000, 40400, 51000],
  },
  {
    id: 3,
    entityId: 3,
    name: 'Certificados de Aforro',
    type: 'certificados_aforro' as const,
    monthly: 0,
    user: null as string | null,
    values: [100000, 100300, 100600, 100900, 101200, 101500],
  },
];

const investmentContributions = new Map<number, InvestmentContribution[]>();
const investmentSnapshots = new Map<number, InvestmentSnapshot[]>();
let investmentRowId = 1;
for (const seed of instrumentSeeds) {
  const contributionRows: InvestmentContribution[] =
    seed.monthly > 0
      ? investmentMonths.map((month) => ({
          id: investmentRowId++,
          month,
          amountCents: seed.monthly,
          description: '',
          createdAt: '2026-07-01T00:00:00Z',
        }))
      : [
          {
            id: investmentRowId++,
            month: '2026-01',
            amountCents: 100000,
            description: 'Purchase',
            createdAt: '2026-07-01T00:00:00Z',
          },
        ];
  const snapshotRows: InvestmentSnapshot[] = investmentMonths.map((month, i) => ({
    id: investmentRowId++,
    month,
    valueCents: seed.values[i],
    description: '',
    createdAt: '2026-07-01T00:00:00Z',
  }));
  // The pension plan reports quarterly, so its chart shows estimated months.
  const reported = seed.id === 2 ? snapshotRows.filter((_, i) => i !== 2 && i !== 3) : snapshotRows;
  investmentContributions.set(seed.id, contributionRows.reverse());
  investmentSnapshots.set(seed.id, reported.reverse());
}

const instruments: InvestmentInstrument[] = instrumentSeeds.map((seed) => {
  const contributionRows = investmentContributions.get(seed.id) ?? [];
  const allocated = contributionRows.reduce((sum, c) => sum + c.amountCents, 0);
  const current = seed.values[seed.values.length - 1];
  return {
    id: seed.id,
    entityId: seed.entityId,
    entityName: ['Trading Broker', 'Pension Manager', 'Estado'][seed.entityId - 1],
    name: seed.name,
    type: seed.type,
    notes: '',
    user: seed.user,
    contributionCount: contributionRows.length,
    snapshotCount: seed.values.length,
    allocatedCents: allocated,
    currentValueCents: current,
    gainCents: current - allocated,
    gainPct: allocated > 0 ? Math.trunc(((current - allocated) * 10000) / allocated) : 0,
    lastSnapshotMonth: investmentMonths[investmentMonths.length - 1],
  };
});

const entities: InvestmentEntity[] = [
  { id: 1, name: 'Trading Broker', kind: 'broker', user: 'alex' },
  { id: 2, name: 'Pension Manager', kind: 'investment_firm', user: null },
  { id: 3, name: 'Estado', kind: 'state', user: null },
].map((e) => {
  const own = instruments.filter((i) => i.entityId === e.id);
  const allocated = own.reduce((sum, i) => sum + i.allocatedCents, 0);
  const current = own.reduce((sum, i) => sum + i.currentValueCents, 0);
  return {
    ...e,
    kind: e.kind as InvestmentEntity['kind'],
    instrumentCount: own.length,
    allocatedCents: allocated,
    currentValueCents: current,
    gainCents: current - allocated,
    gainPct: allocated > 0 ? Math.trunc(((current - allocated) * 10000) / allocated) : 0,
  };
});

// Cashflow: six months of movements for the two users, with the summaries built
// from the rows so the figures stay consistent. The month two back spends more
// than it earns, and a month has two identical movements on one day.
type CashflowCategory = Schemas['CashflowCategory'];
type CashflowMonth = Schemas['CashflowMonth'];
type CashflowMovement = Schemas['CashflowMovement'];
type CashflowCategoryTotal = Schemas['CashflowCategoryTotal'];
type CashflowUserTotal = Schemas['CashflowUserTotal'];

const cashflowCategories: CashflowCategory[] = [
  {
    id: 1,
    name: 'Bank Fees',
    direction: 'expense',
    icon: 'landmark',
    color: '#78716C',
    role: '',
  },
  { id: 2, name: 'Bonus', direction: 'income', icon: 'award', color: '#F59E0B', role: '' },
  {
    id: 3,
    name: 'Education',
    direction: 'expense',
    icon: 'graduation-cap',
    color: '#7C3AED',
    role: '',
  },
  {
    id: 4,
    name: 'Electricity',
    direction: 'expense',
    icon: 'zap',
    color: '#EAB308',
    role: '',
  },
  { id: 5, name: 'Gift', direction: 'income', icon: 'gift', color: '#EC4899', role: '' },
  { id: 6, name: 'Gifts', direction: 'expense', icon: 'gift', color: '#EC4899', role: '' },
  {
    id: 7,
    name: 'Groceries',
    direction: 'expense',
    icon: 'shopping-basket',
    color: '#65A30D',
    role: '',
  },
  {
    id: 8,
    name: 'Health',
    direction: 'expense',
    icon: 'heart-pulse',
    color: '#E11D48',
    role: '',
  },
  {
    id: 9,
    name: 'Insurance',
    direction: 'expense',
    icon: 'shield-check',
    color: '#64748B',
    role: '',
  },
  {
    id: 10,
    name: 'Interest',
    direction: 'income',
    icon: 'percent',
    color: '#0D9488',
    role: '',
  },
  {
    id: 11,
    name: 'Investments',
    direction: 'expense',
    icon: 'trending-up',
    color: '#0D9488',
    role: '',
  },
  {
    id: 12,
    name: 'Leisure',
    direction: 'expense',
    icon: 'gamepad-2',
    color: '#D946EF',
    role: '',
  },
  {
    id: 13,
    name: 'Meal Allowance',
    direction: 'income',
    icon: 'utensils',
    color: '#84CC16',
    role: '',
  },
  {
    id: 14,
    name: 'Miscellaneous',
    direction: 'expense',
    icon: 'shapes',
    color: '#9CA3AF',
    role: '',
  },
  {
    id: 15,
    name: 'Mortgage',
    direction: 'expense',
    icon: 'house',
    color: '#8D6E63',
    role: '',
  },
  {
    id: 16,
    name: 'Objectives',
    direction: 'expense',
    icon: 'target',
    color: '#A855F7',
    role: '',
  },
  {
    id: 17,
    name: 'Others',
    direction: 'income',
    icon: 'ellipsis',
    color: '#9CA3AF',
    role: '',
  },
  {
    id: 18,
    name: 'Restaurant',
    direction: 'expense',
    icon: 'utensils',
    color: '#E76F51',
    role: '',
  },
  {
    id: 19,
    name: 'Salary',
    direction: 'income',
    icon: 'briefcase',
    color: '#16A34A',
    role: '',
  },
  {
    id: 20,
    name: 'Security',
    direction: 'expense',
    icon: 'shield',
    color: '#475569',
    role: '',
  },
  {
    id: 21,
    name: 'Telecom',
    direction: 'expense',
    icon: 'smartphone',
    color: '#6366F1',
    role: '',
  },
  {
    id: 22,
    name: 'Transportation',
    direction: 'expense',
    icon: 'car-front',
    color: '#0284C7',
    role: '',
  },
  {
    id: 23,
    name: 'Water',
    direction: 'expense',
    icon: 'droplets',
    color: '#2196F3',
    role: '',
  },
];
const categoryByName = new Map(cashflowCategories.map((c) => [c.name, c]));

const cashflowMovements: CashflowMovement[] = [];
let cashflowRowId = 1;

function addMovement(
  offset: number,
  day: number,
  username: string,
  category: string,
  amountCents: number,
  description = '',
) {
  const c = categoryByName.get(category) as CashflowCategory;
  const date = `${month(offset)}-${String(day).padStart(2, '0')}`;
  cashflowMovements.push({
    id: cashflowRowId++,
    date,
    username,
    category: c.name,
    direction: c.direction,
    amountCents,
    description,
    createdAt: `${date}T09:00:00Z`,
  });
}

for (let offset = -5; offset <= 0; offset++) {
  const wobble = (offset + 5) * 1500;
  addMovement(offset, 1, 'alex', 'Salary', 210000, 'Salary');
  addMovement(offset, 1, 'maria', 'Salary', 180000, 'Salary');
  addMovement(offset, 2, 'alex', 'Mortgage', 70000, 'Mortgage');
  addMovement(offset, 5, 'alex', 'Groceries', 14000 + wobble);
  addMovement(offset, 19, 'alex', 'Groceries', 9000);
  addMovement(offset, 6, 'maria', 'Groceries', 4500);
  addMovement(offset, 8, 'alex', 'Electricity', 9500, 'Electricity and water');
  addMovement(offset, 10, 'maria', 'Transportation', 4000, 'Monthly pass');
  addMovement(offset, 12, 'alex', 'Transportation', 3500);
  addMovement(offset, 21, 'alex', 'Restaurant', 6000 + wobble / 2);
  addMovement(offset, 14, 'maria', 'Restaurant', 5500);
  addMovement(offset, 28, 'maria', 'Restaurant', 4200);
}
// Two identical movements on one day.
addMovement(0, 6, 'maria', 'Groceries', 4500);
// A one-off that pushes a month above its income.
addMovement(-2, 15, 'alex', 'Insurance', 260000, 'Annual insurance');
addMovement(-3, 20, 'maria', 'Others', 8900, 'Returned shoes');

const cashflowDates = (m: string) => cashflowMovements.filter((x) => x.date.startsWith(`${m}-`));

function cashflowMonthSummary(m: string, rows: CashflowMovement[]): CashflowMonth {
  const incomeCents = rows
    .filter((r) => r.direction === 'income')
    .reduce((sum, r) => sum + r.amountCents, 0);
  const expenseCents = rows
    .filter((r) => r.direction === 'expense')
    .reduce((sum, r) => sum + r.amountCents, 0);
  const netCents = incomeCents - expenseCents;
  return {
    month: m,
    incomeCents,
    expenseCents,
    netCents,
    savingsRatePct: incomeCents > 0 ? Math.trunc((netCents * 10000) / incomeCents) : 0,
    movementCount: rows.length,
  };
}

const cashflowMonths: CashflowMonth[] = [
  ...new Set(cashflowMovements.map((x) => x.date.slice(0, 7))),
]
  .sort()
  .reverse()
  .map((m) => cashflowMonthSummary(m, cashflowDates(m)));

function cashflowMonthDetail(m: string) {
  const rows = cashflowDates(m);
  if (rows.length === 0) return null;
  const byCategory = new Map<number, CashflowCategoryTotal>();
  const byUser = new Map<string, CashflowUserTotal>();
  for (const r of rows) {
    const c = categoryByName.get(r.category) as CashflowCategory;
    const total = byCategory.get(c.id) ?? {
      categoryId: c.id,
      name: c.name,
      direction: c.direction,
      totalCents: 0,
      movementCount: 0,
      role: c.role,
    };
    total.totalCents += r.amountCents;
    total.movementCount += 1;
    byCategory.set(c.id, total);

    const user = byUser.get(r.username) ?? {
      username: r.username,
      incomeCents: 0,
      expenseCents: 0,
      netCents: 0,
      spentCents: 0,
      putAsideCents: 0,
      cashCents: null,
    };
    if (r.direction === 'income') user.incomeCents += r.amountCents;
    else {
      user.expenseCents += r.amountCents;
      if (['savings', 'investments', 'goals'].includes(c.role)) user.putAsideCents += r.amountCents;
      else user.spentCents += r.amountCents;
    }
    user.netCents = user.incomeCents - user.expenseCents;
    byUser.set(r.username, user);
  }
  return {
    ...cashflowMonthSummary(m, rows),
    savedCents: [...byCategory.values()]
      .filter((c) => c.role === 'savings')
      .reduce((sum, c) => sum + c.totalCents, 0),
    byCategory: [...byCategory.values()].sort(
      (a, b) =>
        a.direction.localeCompare(b.direction) ||
        b.totalCents - a.totalCents ||
        a.name.localeCompare(b.name),
    ),
    byUser: [...byUser.values()].sort((a, b) => a.username.localeCompare(b.username)),
    peopleHistory: [],
  };
}

function cashflowMonthMovements(m: string) {
  return cashflowDates(m).sort((a, b) => b.date.localeCompare(a.date) || b.id - a.id);
}

/** The month a mock route captured as a year and a month number. */
const monthOf = ([year, number]: number[]) => `${year}-${String(number).padStart(2, '0')}`;

type SavingsAccount = Schemas['SavingsAccount'];
type SavingsMovement = Schemas['SavingsMovement'];
type SavingsSnapshot = Schemas['SavingsSnapshot'];

const savingsAccounts: SavingsAccount[] = [
  {
    id: 1,
    name: 'Emergency fund, Certificados de Aforro',
    owner: 'alex',
    targetCents: 1000000,
    notes: '',
    balanceCents: 740000,
    depositedCents: 700000,
    withdrawnCents: 0,
    interestCents: 40000,
    progressPct: 74,
    movementCount: 2,
    snapshotCount: 2,
    lastSnapshotMonth: month(0),
  },
  {
    id: 2,
    name: 'Emergency fund, savings account',
    owner: 'maria',
    targetCents: 500000,
    notes: '',
    balanceCents: 315000,
    depositedCents: 350000,
    withdrawnCents: 50000,
    interestCents: 15000,
    progressPct: 63,
    movementCount: 3,
    snapshotCount: 2,
    lastSnapshotMonth: month(0),
  },
  {
    id: 3,
    name: 'Holiday pot',
    owner: null,
    targetCents: null,
    notes: '',
    balanceCents: 0,
    depositedCents: 0,
    withdrawnCents: 0,
    interestCents: 0,
    progressPct: 0,
    movementCount: 0,
    snapshotCount: 0,
    lastSnapshotMonth: null,
  },
];

const savingsMovements = new Map<number, SavingsMovement[]>([
  [
    1,
    [
      {
        id: 2,
        month: month(-3),
        type: 'deposit',
        amountCents: 200000,
        description: 'Top-up',
        createdAt: '2026-01-01T00:00:00Z',
      },
      {
        id: 1,
        month: month(-8),
        type: 'deposit',
        amountCents: 500000,
        description: 'Opening',
        createdAt: '2026-01-01T00:00:00Z',
      },
    ],
  ],
  [
    2,
    [
      {
        id: 5,
        month: month(-1),
        type: 'deposit',
        amountCents: 25000,
        description: 'Monthly saving',
        createdAt: '2026-01-01T00:00:00Z',
      },
      {
        id: 4,
        month: month(-4),
        type: 'withdrawal',
        amountCents: 50000,
        description: 'Car repair',
        createdAt: '2026-01-01T00:00:00Z',
      },
      {
        id: 3,
        month: month(-9),
        type: 'deposit',
        amountCents: 325000,
        description: 'Opening',
        createdAt: '2026-01-01T00:00:00Z',
      },
    ],
  ],
]);

const savingsSnapshots = new Map<number, SavingsSnapshot[]>([
  [
    1,
    [
      {
        id: 2,
        month: month(0),
        valueCents: 740000,
        description: '',
        createdAt: '2026-01-01T00:00:00Z',
      },
      {
        id: 1,
        month: month(-6),
        valueCents: 715000,
        description: '',
        createdAt: '2026-01-01T00:00:00Z',
      },
    ],
  ],
  [
    2,
    [
      {
        id: 4,
        month: month(0),
        valueCents: 315000,
        description: '',
        createdAt: '2026-01-01T00:00:00Z',
      },
      {
        id: 3,
        month: month(-5),
        valueCents: 305000,
        description: '',
        createdAt: '2026-01-01T00:00:00Z',
      },
    ],
  ],
]);

/** Sample net worth history: eight months of figures that grow a little each month. */
function netWorthHistory(): Schemas['NetWorthMonth'][] {
  return Array.from({ length: 8 }, (_, i) => {
    const savingsCents = 7_200_000 + i * 130_000;
    const investmentsCents = 9_800_000 + i * 175_000;
    const goalsCents = 1_100_000 + i * 62_000;
    const cashCents = 980_000 + i * 79_000;
    const creditCents = 23_304_000 - i * 21_500;
    const assetsCents = savingsCents + investmentsCents + goalsCents + cashCents;
    return {
      month: `2026-${String(i + 4).padStart(2, '0')}`,
      savingsCents,
      investmentsCents,
      goalsCents,
      cashCents,
      assetsCents,
      creditCents,
      netCents: assetsCents - creditCents,
    };
  });
}

export const routes: MockRoute[] = [
  { pattern: /^\/overview\/net-worth$/, handle: () => netWorthHistory() },
  { pattern: /^\/savings\/accounts$/, handle: () => savingsAccounts },
  {
    pattern: /^\/savings\/accounts\/(\d+)$/,
    handle: ([id]) => savingsAccounts.find((a) => a.id === id) ?? null,
  },
  {
    pattern: /^\/savings\/accounts\/(\d+)\/movements$/,
    handle: ([id]) =>
      savingsAccounts.some((a) => a.id === id) ? (savingsMovements.get(id) ?? []) : null,
  },
  {
    pattern: /^\/savings\/accounts\/(\d+)\/snapshots$/,
    handle: ([id]) =>
      savingsAccounts.some((a) => a.id === id) ? (savingsSnapshots.get(id) ?? []) : null,
  },
  { pattern: /^\/goals\/accounts$/, handle: () => accounts },
  {
    pattern: /^\/goals\/accounts\/(\d+)$/,
    handle: ([id]) => {
      const account = accounts.find((a) => a.id === id);
      return account ? { account, balance: balance(id) } : null;
    },
  },
  {
    pattern: /^\/goals\/accounts\/(\d+)\/goals$/,
    handle: ([id]) =>
      accounts.some((a) => a.id === id) ? goals.filter((g) => g.accountId === id) : null,
  },
  {
    pattern: /^\/goals\/accounts\/(\d+)\/movements$/,
    handle: ([id]) =>
      accounts.some((a) => a.id === id)
        ? ledger(movements.filter((m) => m.accountId === id))
        : null,
  },
  {
    pattern: /^\/goals\/accounts\/(\d+)\/projection$/,
    handle: ([id]) => (accounts.some((a) => a.id === id) ? projection(id) : null),
  },
  {
    pattern: /^\/goals\/(\d+)$/,
    handle: ([id]) => {
      const goal = goals.find((g) => g.id === id);
      if (!goal) return null;
      return { goal, movements: ledger(movements.filter((m) => m.goalId === id)) };
    },
  },
  { pattern: /^\/credits$/, handle: () => contracts },
  {
    pattern: /^\/credits\/(\d+)$/,
    handle: ([id]) => contracts.find((c) => c.id === id) ?? null,
  },
  {
    pattern: /^\/credits\/(\d+)\/payments$/,
    handle: ([id]) => (contracts.some((c) => c.id === id) ? (payments.get(id) ?? []) : null),
  },
  { pattern: /^\/investments\/entities$/, handle: () => entities },
  {
    pattern: /^\/investments\/entities\/(\d+)$/,
    handle: ([id]) => entities.find((e) => e.id === id) ?? null,
  },
  {
    pattern: /^\/investments\/entities\/(\d+)\/instruments$/,
    handle: ([id]) =>
      entities.some((e) => e.id === id) ? instruments.filter((i) => i.entityId === id) : null,
  },
  {
    pattern: /^\/investments\/instruments\/(\d+)$/,
    handle: ([id]) => instruments.find((i) => i.id === id) ?? null,
  },
  {
    pattern: /^\/investments\/instruments\/(\d+)\/contributions$/,
    handle: ([id]) =>
      instruments.some((i) => i.id === id) ? (investmentContributions.get(id) ?? []) : null,
  },
  {
    pattern: /^\/investments\/instruments\/(\d+)\/snapshots$/,
    handle: ([id]) =>
      instruments.some((i) => i.id === id) ? (investmentSnapshots.get(id) ?? []) : null,
  },
  { pattern: /^\/cashflow\/categories$/, handle: () => cashflowCategories },
  { pattern: /^\/cashflow\/months$/, handle: () => cashflowMonths },
  {
    pattern: /^\/cashflow\/months\/(\d{4})-(\d{2})$/,
    handle: (parts) => cashflowMonthDetail(monthOf(parts)),
  },
  {
    pattern: /^\/cashflow\/months\/(\d{4})-(\d{2})\/movements$/,
    handle: (parts) => cashflowMonthMovements(monthOf(parts)),
  },
];
