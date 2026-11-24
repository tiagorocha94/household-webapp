# 016 Home says what needs a look

Status: accepted and built

Context: money put aside is typed twice: as an expense in cashflow and as a
deposit or contribution in the savings, investments or goals module. The two
can drift apart without anyone noticing, and a credit payment can be forgotten.
Linking the modules automatically was tried for goals and for savings accounts
and removed: it added accounts, sources and options for little gain, and the
person is happy to type both.

Decision:
- **A role on expense categories.** `cashflow_categories.csv` has a `role`
  column: empty, `savings`, `investments`, `goals` or `credits`. A role names where a
  category's money goes. It does not change how the movements are counted: they
  stay expenses.
- **Compare, do not link.** For the current month and the two before, Home
  compares what the categories of each role spent with what was typed in the
  matching module (savings deposits, investment contributions, goal deposits).
  When cashflow has more, there is an item with the month and the amount. More
  typed than spent is not an item. A role with no category is not checked.
- **Credits are checked by presence.** For the `credits` role Home only checks
  that a credit payment exists in a month that has such expenses, never the
  amounts: a cashflow instalment often includes insurance and fees that a
  credit payment does not hold. With several credits, one payment silences the
  warning for all of them.
- **Credits can be chased.** A credit may have an `expected_day` (1 to 28).
  When that day has passed and the month has no regular payment, there is an
  item naming the credit. A credit without the day is never checked. Incomes are
  not chased: people are paid on different days.
- **One place to look.** The overview's `attention` list carries every item,
  goals running short and budget groups over their plan included. The web app
  shows a bell with the count in a top bar and a Notifications page that lists
  them, and Home shows only a line with the count. The items are computed on
  every request and stored nowhere.

Consequences: three months is the window; an older gap is not shown, and there
is no way to dismiss an item (it is in the [backlog](../development/backlog.md)).
Both modules are still typed by hand, so the check is a reminder, not a
guarantee. The rules are in the
[home overview](../modules/home/overview.md) and the
[cashflow CSV](../modules/cashflow/csv.md).
