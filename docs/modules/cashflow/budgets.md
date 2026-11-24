# Budget groups

Budget groups divide expense categories into shares of monthly income. The
feature is built in the database, CSV import, web interface and phone app.

## Rules

- `cashflow_budget_groups.csv` defines each group’s name, whole-number
  percentage and display order.
- `cashflow_categories.csv` assigns each expense category to at most one group.
  Income categories have no group.
- When any groups exist, every expense category needs one and the shares must
  total exactly 100%.
- The month’s income is the complete budget. A group receives its percentage in
  exact cents; leftover cents go to the largest group so all budgets add back
  to the income.
- Spending is the sum of the group’s expense categories. Remaining budget is
  budget minus spending and may be negative.
- With no groups, the feature is off everywhere.

The exact columns and validation rules are in [csv.md](csv.md).

## Interfaces

The web month page summarizes each group and links to its category breakdown.
A month without income shows spending but no plan.

The phone reads the household figures from the dedicated device route
([ADR 015](../../decisions/015-devices-read-budget.md)). Its **Spent** tab shows
each group’s share of income, or of total spending when there is no income.
**Plan vs actual** compares spending with the group budget.