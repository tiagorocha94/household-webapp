# 018 Cash is worked out from typed bank balances

Status: accepted and built

Context: net worth counted savings, investments, goals and credit, but not the
money sitting in the bank accounts, which is often most of what a person has.
Every month's income and expenses are already in cashflow, and money put into
savings, investments, goals and credits is already an expense there. So the
bank balance only lacks a starting point.

Decision:
- **One typed balance is the anchor.** `cashflow_balances.csv` holds, per person
  and month, what the bank account held at the end of the month. A person has
  one balance, not one per account.
- **Everything else is derived.** A person's cash in a month is the latest typed
  balance at or before it plus the net cashflow after that balance, up to the
  month. A month before every typed balance counts back from the earliest one.
  Nothing is stored: the figure is worked out on request, so correcting an old
  movement or balance changes every figure that depends on it.
- **A new balance corrects drift.** Movements that were never recorded make the
  derived cash drift from the bank. Typing the real balance for a later month
  resets it from there.
- **No balance: start from zero.** A person who never typed a balance has cash
  counted from zero at their first movement, so the whole history is tracked
  even without a bank figure. A person with neither a balance nor a movement has
  no cash. (The first version left such a person out; changed because the
  history is worth keeping even when nobody types a balance.)
- **Net worth adds it.** The overview's `netWorth` includes each person's cash
  and returns it by person.

Consequences: the cash figure is only as good as the recorded movements and the
last typed balance. Cash is per person only; the household figure is their sum.
A zero start is only right when the first movement is the person's true start;
typing a balance later corrects it. The rules and columns are in the [cashflow CSV](../modules/cashflow/csv.md) and
the [home overview](../modules/home/overview.md).
