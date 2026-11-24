# 015 Enrolled devices read the budget by group

Status: accepted and built, on the server and in the phone app

Context: the phone app (`household-cashflow`) gets a Budget page: what the
household spent in a month by budget group, as a donut with the leftover, and
each group's spending against its plan. The figures are the household's, not
the phone's, so they must come from the server and not from the movements kept
on one phone. The existing month endpoint needs the session cookie, and
[ADR 012](012-devices-read-goals.md) left cashflow closed to devices, because
it also carries every person's movements.

Decision:
- **One new read-only route, for session or device:**
  `GET /api/v1/cashflow/budget/{month}`. It returns only what the page shows:
  the month's income, expenses and difference, the budget groups (share,
  budget, spent, left, and the categories with what each spent), and the list
  of months that have movements, so the phone can offer a month picker. It
  uses the same figures as the budget section of the web month page
  ([budget groups](../modules/cashflow/budgets.md)); nothing is computed
  twice.
- **A month with no movements is not an error.** It returns zero income and
  spending, with the groups listed, so the phone shows an empty month instead
  of handling a 404.
- **No income means no plan.** Budgets are shares of the income, so they are
  zero for a month without income; the phone then shows spending and its
  shares only.
- **Nothing else opens.** Movements, people and every other cashflow route
  stay for the web session. A device sees totals by group and category, never
  who spent what or any movement's note.
- **Without groups, the groups list is empty.** The phone says budget groups
  are not set up.

Consequences: a stolen device token can now read the goals and the month's
budget totals, as well as write that person's movements. It is revoked per
device, as before. The phone keeps the last answer with the time it was read
and clears it on disconnect, as it does for the goals.
