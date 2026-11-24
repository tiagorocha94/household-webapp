# 014 Savings are their own module

Status: accepted and built

Context: money kept as a cushion or emergency fund (a savings account,
Certificados de Aforro) is today either left out or recorded as an investment.
It is not invested, it is meant to be used in an emergency and put back, and it
earns low interest. Counting it as an investment would raise "Invested" and its
gain figures. The Home overview and the goals module do not cover it either:
goals have dates and a monthly plan, a cushion has neither.

Decision:
- **A new module, `savings`, with its own tables and CSV files**, read only
  through the API like the others. It shares no tables with investments, so it
  never adds to "Invested" or its gains. Moving a Certificados de Aforro from
  investments is a manual edit of the CSV files.
- **An account** has a name, an optional owner (a user, empty for the
  household) and an optional target. A target is per account; two accounts for
  one purpose are two accounts, one per owner. There are no end dates and no
  monthly plan.
- **Movements** are `deposit` and `withdrawal` in a month. Putting money back
  is a deposit. (An `emergency` mark on withdrawals was part of the first
  version and was removed: nothing used it.)
- **Snapshots** record the balance seen in a month, as investments do. The
  balance is the latest snapshot, or the deposits less the withdrawals while
  there is none. **Interest earned** is the balance less deposits plus
  withdrawals: derived, never typed.
- **Cashflow gets a role on expense categories** (`savings` here; later also
  `investments` and `goals`). A movement in a `savings` category stays an expense, so budgets, totals and the savings rate do not
  change. Views add "of which saved" beside the expenses.
- **Deposits stay typed, as in investments.** There is one savings category
  in cashflow, not one per account, so the money taken from the main account
  does not say which savings account receives it, and a deposit is entered by
  hand in `savings_movements.csv`. An automatic link was built and removed for this
  reason.

Consequences:
- The savings balance is part of the net worth figure on Home, with the
  investments' value and the money saved in goals, minus the credit
  outstanding.
- A transfer to a savings account is typed twice: as an expense in the savings
  category in cashflow and as a deposit in the savings CSV. Home compares the
  two for the last three months and warns about a difference.
- A new module adds a migration, three CSV files, a read API, a page and a
  navigation entry.
