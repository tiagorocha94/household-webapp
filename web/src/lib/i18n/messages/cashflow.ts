// Texts of the "cashflow" area, in English and Portuguese (Portugal). Keys start with "cashflow.".
export const en = {
  // Shared words
  'cashflow.title': 'Cashflow',
  'cashflow.income': 'Income',
  'cashflow.expenses': 'Expenses',
  'cashflow.leftOver': 'Left over',
  'cashflow.savingsRate': 'Savings rate',
  'cashflow.movements': 'Movements',
  'cashflow.month': 'Month',
  'cashflow.year': 'Year',
  'cashflow.date': 'Date',
  'cashflow.person': 'Person',
  'cashflow.category': 'Category',
  'cashflow.description': 'Description',
  'cashflow.amount': 'Amount',
  'cashflow.spent': 'Spent',
  'cashflow.putAside': 'Put aside',
  'cashflow.showMore': 'Show more ({count} left)',
  'cashflow.vsMonth': '{change} vs {month}',

  // Category roles
  'cashflow.role.savings': 'Saved',
  'cashflow.role.investments': 'Invested',
  'cashflow.role.goals': 'Goals',
  'cashflow.role.credits': 'Credits',

  // Month tabs
  'cashflow.tabsLabel': 'This month',
  'cashflow.tab.summary': 'Summary',
  'cashflow.tab.budget': 'Budget by group',
  'cashflow.tab.categories': 'Categories',
  'cashflow.tab.people': 'People',
  'cashflow.tab.movements': 'Movements',

  // Month page
  'cashflow.allMonths': 'All months',
  'cashflow.budgetByGroup': 'Budget by group',
  'cashflow.details': 'Details',
  'cashflow.noIncomeNoBudget': 'No income this month, so there is no budget to compare with.',
  'cashflow.ofBudget': 'of {amount}',
  'cashflow.groupBudgetSpent': '{name}: {percent} of the budget spent',
  'cashflow.whereMoneyWent': 'Where the money went',
  'cashflow.allCategories': 'All categories',
  'cashflow.noExpenses': 'There are no expenses this month.',
  'cashflow.latestMovements': 'Latest movements',
  'cashflow.allMovements': 'All {count} movements',
  'cashflow.noMovements': 'No movements this month.',
  'cashflow.ofWhichSaved': 'of which saved {amount}',
  'cashflow.whereItCameFrom': 'Where it came from',

  // Group budgets
  'cashflow.groupsHelp':
    "Each group gets its share of this month's income ({amount}). Open a group to see its categories.",
  'cashflow.over': 'Over by',
  'cashflow.left': 'Left',
  'cashflow.groupEmpty': 'Nothing was spent in this group.',
  'cashflow.groupPercentNote': "Percent = share of the group's spending.",

  // Share list
  'cashflow.shareOfExpenses': '{name}: {percent} of the expenses',

  // People
  'cashflow.byPerson': 'By person',
  'cashflow.putAsideHelp':
    'Put aside is the money moved to savings, investments and goals. Everything else spent, credit payments included, counts as spent.',
  'cashflow.cash': 'Cash {amount}',
  'cashflow.splitLabel': 'Income split into spent, put aside and left over',
  'cashflow.shareBefore': 'Put aside',
  'cashflow.shareMiddle': 'of income, spent',
  'cashflow.putAsideLast6': 'Put aside, last 6 months',
  'cashflow.household': 'Household',
  'cashflow.rowPutAside': '{name}, put aside',
  'cashflow.rowOfIncome': '{name}, of income',

  // Months page
  'cashflow.noCashflow':
    'No cashflow yet. Movements appear here once they are added to the CSV files.',
  'cashflow.latestMonth': 'Latest month:',
  'cashflow.incomeAndExpenses': 'Income and expenses',
  'cashflow.byYear': 'By year',

  // Chart
  'cashflow.nothingToChart': 'There is nothing to chart yet.',
  'cashflow.chartSummary': 'Income and expenses for each month from {from} to {to}.',
  'cashflow.showTable': 'Show as a table',

  // Goal movements table
  'cashflow.noGoalMovements':
    'No movements yet. They appear here once they are added to the CSV files.',
  'cashflow.filterByType': 'Filter by type',
  'cashflow.all': 'All',
  'cashflow.type': 'Type',
  'cashflow.goal': 'Goal',
  'cashflow.account': 'Account',
  'cashflow.balanceAfter': 'Balance after',
  'cashflow.closing': 'Closing',
  'cashflow.movementType.deposit': 'Deposit',
  'cashflow.movementType.allocation': 'Allocation',
  'cashflow.movementType.expense': 'Expense',
  'cashflow.movementType.release': 'Release',
  'cashflow.movementType.cover': 'Cover',
  'cashflow.closingNote.release': 'leftover returned to account',
  'cashflow.closingNote.cover': 'overspend covered by account',
} as const;

export const pt: Record<keyof typeof en, string> = {
  'cashflow.title': 'Cashflow',
  'cashflow.income': 'Receitas',
  'cashflow.expenses': 'Despesas',
  'cashflow.leftOver': 'Sobra',
  'cashflow.savingsRate': 'Taxa de poupança',
  'cashflow.movements': 'Movimentos',
  'cashflow.month': 'Mês',
  'cashflow.year': 'Ano',
  'cashflow.date': 'Data',
  'cashflow.person': 'Pessoa',
  'cashflow.category': 'Categoria',
  'cashflow.description': 'Descrição',
  'cashflow.amount': 'Valor',
  'cashflow.spent': 'Gasto',
  'cashflow.putAside': 'Posto de parte',
  'cashflow.showMore': 'Mostrar mais (faltam {count})',
  'cashflow.vsMonth': '{change} face a {month}',

  'cashflow.role.savings': 'Poupado',
  'cashflow.role.investments': 'Investido',
  'cashflow.role.goals': 'Objetivos',
  'cashflow.role.credits': 'Créditos',

  'cashflow.tabsLabel': 'Este mês',
  'cashflow.tab.summary': 'Resumo',
  'cashflow.tab.budget': 'Orçamento por grupo',
  'cashflow.tab.categories': 'Categorias',
  'cashflow.tab.people': 'Pessoas',
  'cashflow.tab.movements': 'Movimentos',

  'cashflow.allMonths': 'Todos os meses',
  'cashflow.budgetByGroup': 'Orçamento por grupo',
  'cashflow.details': 'Detalhes',
  'cashflow.noIncomeNoBudget': 'Sem receitas neste mês, por isso não há orçamento para comparar.',
  'cashflow.ofBudget': 'de {amount}',
  'cashflow.groupBudgetSpent': '{name}: {percent} do orçamento gasto',
  'cashflow.whereMoneyWent': 'Para onde foi o dinheiro',
  'cashflow.allCategories': 'Todas as categorias',
  'cashflow.noExpenses': 'Não há despesas neste mês.',
  'cashflow.latestMovements': 'Últimos movimentos',
  'cashflow.allMovements': 'Todos os {count} movimentos',
  'cashflow.noMovements': 'Sem movimentos neste mês.',
  'cashflow.ofWhichSaved': 'das quais poupadas {amount}',
  'cashflow.whereItCameFrom': 'De onde veio',

  'cashflow.groupsHelp':
    'Cada grupo recebe a sua parte das receitas deste mês ({amount}). Abra um grupo para ver as suas categorias.',
  'cashflow.over': 'Excedido em',
  'cashflow.left': 'Resta',
  'cashflow.groupEmpty': 'Não se gastou nada neste grupo.',
  'cashflow.groupPercentNote': 'Percentagem = parte da despesa do grupo.',

  'cashflow.shareOfExpenses': '{name}: {percent} das despesas',

  'cashflow.byPerson': 'Por pessoa',
  'cashflow.putAsideHelp':
    'Posto de parte é o dinheiro movido para poupança, investimentos e objetivos. Tudo o resto que se gasta, incluindo os pagamentos de créditos, conta como gasto.',
  'cashflow.cash': 'Dinheiro {amount}',
  'cashflow.splitLabel': 'Receitas repartidas entre gasto, posto de parte e sobra',
  'cashflow.shareBefore': 'Posto de parte',
  'cashflow.shareMiddle': 'das receitas, gasto',
  'cashflow.putAsideLast6': 'Posto de parte, últimos 6 meses',
  'cashflow.household': 'Agregado familiar',
  'cashflow.rowPutAside': '{name}, posto de parte',
  'cashflow.rowOfIncome': '{name}, das receitas',

  'cashflow.noCashflow':
    'Ainda não há cashflow. Os movimentos aparecem aqui depois de adicionados aos ficheiros CSV.',
  'cashflow.latestMonth': 'Último mês:',
  'cashflow.incomeAndExpenses': 'Receitas e despesas',
  'cashflow.byYear': 'Por ano',

  'cashflow.nothingToChart': 'Ainda não há nada para mostrar no gráfico.',
  'cashflow.chartSummary': 'Receitas e despesas de cada mês, de {from} a {to}.',
  'cashflow.showTable': 'Mostrar como tabela',

  'cashflow.noGoalMovements':
    'Ainda não há movimentos. Aparecem aqui depois de adicionados aos ficheiros CSV.',
  'cashflow.filterByType': 'Filtrar por tipo',
  'cashflow.all': 'Todos',
  'cashflow.type': 'Tipo',
  'cashflow.goal': 'Objetivo',
  'cashflow.account': 'Conta',
  'cashflow.balanceAfter': 'Saldo depois',
  'cashflow.closing': 'Fecho',
  'cashflow.movementType.deposit': 'Depósito',
  'cashflow.movementType.allocation': 'Atribuição',
  'cashflow.movementType.expense': 'Despesa',
  'cashflow.movementType.release': 'Libertação',
  'cashflow.movementType.cover': 'Cobertura',
  'cashflow.closingNote.release': 'sobra devolvida à conta',
  'cashflow.closingNote.cover': 'excesso coberto pela conta',
};
