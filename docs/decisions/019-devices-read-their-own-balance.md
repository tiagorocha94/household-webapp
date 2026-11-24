# 019 Enrolled devices read their owner's cash balance

Status: accepted and built in the server and phone app

Context: the phone app shows the person's bank balance, which moves as they add
expenses and incomes ([ADR 018](018-cash-balance-from-typed-balances.md)). The
figure comes from typed balances and the whole cashflow, so it must be worked
out on the server. The month endpoints need the session cookie, and
[ADR 012](012-devices-read-goals.md) and [ADR 015](015-devices-read-budget.md)
left every person's figures closed to devices.

Decision:
- **One new read-only route, for session or device:**
  `GET /api/v1/cashflow/balance/{month}`. It returns the month and
  `balanceCents`: what the person had in the bank at the end of that month,
  worked out as in ADR 018. The phone asks for the current month.
- **Only the caller's own cash.** The person comes from the session or from the
  device token, never from the path or a parameter, so there is no way to ask
  for someone else's balance.
- **No cash is not an error.** `balanceCents` is null only when the person has
  neither a typed balance nor a movement, and the phone shows a dash.
- **Nothing else opens.** People totals, other people's cash and net worth stay
  for the web session.

Consequences: a stolen device token can now also read its owner's balance, as
well as the goals, the month's budget totals and the owner's movements it can
write. It is revoked per device, as before. The phone keeps the last answer
with the time it was read and clears it on disconnect, as it does for the goals.
