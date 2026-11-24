# Goal Allocation

## Overview

The goal allocation module tracks financial goals funded through monthly
allocations from shared or personal accounts. All money events are tracked in
a unified ledger. The module is read-only through the [JSON API](../../architecture/openapi.yaml);
data is maintained in the CSV files ([csv.md](csv.md)). Any authenticated user
can read everything ([ADR 010](../../decisions/010-household-data-is-shared.md)),
and enrolled phones can read it too ([ADR 012](../../decisions/012-devices-read-goals.md)).

## Accounts

- A household can have several accounts (for example "Household", "Personal").
- Account names are unique, because the other CSV files refer to accounts by
  name.
- Each account has a `monthly_deposit_cents`, the deposit its projection
  assumes for every future month. Every logged-in user can read every account
  ([ADR 010](../../decisions/010-household-data-is-shared.md)).

## Movements (Ledger)

All money events flow through a single ledger (`goal_account_movements`). The
`Direction` below is the **account (bank-statement) perspective**: whether the
event increases or decreases the money available in the account.

| Type         | Direction | Description                         |
|--------------|-----------|-------------------------------------|
| `deposit`    | Inflow    | Money added to the account          |
| `allocation` | Outflow   | Money reserved for a goal           |
| `expense`    | Outflow   | Money spent from a goal             |
| `release`    | Inflow    | Leftover of a closed goal returned to the unallocated money |
| `cover`      | Outflow   | Overspend of a closed goal covered by the unallocated money |

From the **goal's own perspective** the same events look different, because
money reserved for a goal enters it, the leftover returned on close leaves it,
and the cover that makes up an overspend on close enters it:

| Type         | Goal Direction | Description                      |
|--------------|----------------|----------------------------------|
| `allocation` | Inflow         | Money added to the goal          |
| `expense`    | Outflow        | Money spent from the goal        |
| `release`    | Outflow        | Leftover moved out of the goal on close |
| `cover`      | Inflow         | Money put into an overspent goal on close, so it ends at zero |

(`deposit` is not goal-scoped and does not appear in a goal's movements.)

The API exposes both views on each movement as `accountInflow` and
`goalInflow`. Amounts are always positive, and clients derive the sign from
those flags.

Movement lists are ordered newest month first, and within a month the latest
logical event first (`release` or `cover`, `expense`, `allocation`, `deposit`).

### Balance

All balances are computed via SQL views for consistency:

- **Account balance** = `deposits - expenses` (real money, matches the bank statement)
- **Unallocated pool** = `deposits - allocated + released - covered` (free to assign to goals; the web and phone screens call it "Free")
- **Goal balance** = `allocated - expenses - released + covered` (money held by this goal; negative while it has spent more than it holds)
- **Verification**: `unallocated + sum(all goal balances) = account balance`

A negative unallocated pool means the account is over-allocated.

## Goals

- A goal has a name, a target amount, a start month and an end month (YYYY-MM).
- The monthly allocation is computed from the target and the number of months
  (inclusive); see Allocations.
- Goal names are unique within an account.

### States

| State       | Description                                                  |
|-------------|--------------------------------------------------------------|
| `active`    | Goal is in progress or planned (start month not yet reached) |
| `completed` | Goal was closed by hand: allocations and expenses are done    |
| `cancelled` | Goal was cancelled                                           |

- Clients can separate "active" from "planned" by comparing `startMonth` with
  the current month.
- A goal never completes on its own: its allocations may be done while its
  expenses still come. It is completed by setting its status in
  `goal_account_goals.csv` (closing from the app is not built yet; it should be
  allowed only once the goal is fully allocated). A goal is cancelled the same
  way, and its remaining balance returns to the account only if a `release`
  movement is recorded in the movements CSV.
- A goal is expected to end near zero: it exists to be spent, unlike a savings
  or an emergency account. Its balance reads green above zero and red below it.
  When the spent amount goes over the target, clients show a warning that other
  goals may need adapting. Nothing else changes.

### Closing

When sync finds a completed goal it has not settled, it records one closing
movement and the goal's **closing balance** (`closingBalanceCents`):

- Leftover (balance above zero): a `release` of that amount returns it to the
  unallocated money.
- Overspent (balance below zero): a `cover` of the difference takes it from the
  unallocated money, so the goal ends at zero.
- Balance zero: no movement.

The closing balance is the balance the goal had just before the closing
movement, so it is positive for a release, negative for a cover, and matches
the closing movement. It is `null` while the goal is not completed. It is
derived from the closing movement and not stored in the CSV, so a closing
movement already in the movements CSV is kept as it is. The movement type
`refund` of earlier versions is read as `release`.

### Allocations

- Only one allocation per goal per month (enforced by a partial unique index).
- Sync creates the missing allocations of active goals up to the current
  month. Other allocations can be recorded in the movements CSV.
- One rule, `goals.Schedule`, drives both sync and the projection. The target
  is split in exact cents with `money.Split` (leftover cents go one each to
  the first months), so allocations sum exactly to the target, and the final
  month absorbs any drift between the schedule and what was recorded, so
  earlier rounding or hand-edited rows cannot leave a goal short.

## Projection

The projection covers 12 months back to the last end month of any active goal
(or 6 months ahead when there is none). The API returns the whole window.
Past months use recorded movements. Future months use the account's recurring
deposit (`monthly_deposit_cents` in `goal_accounts.csv`, returned as
`monthlyDepositCents`) and each active goal's planned allocations, split in
exact cents like sync does. Running balance and unallocated totals start from
everything recorded before the window.

The recurring deposit is the amount configured for the account, on purpose: it
is not derived from past deposits, because months with extra money (a tax
refund, the extra summer and Christmas salaries) are exceptions and must not
raise the expected amount.

Each month has `isProjection`, which is true for months after the current one.
When `unallocatedCents` goes negative, planned allocations exceed the money
expected to be available: a funding shortfall.

## Automation

`sync` runs on every start after the import (see
[csv-seed.md](../../architecture/csv-seed.md#sync-behavior)). For goals it creates
missing allocations up to the current month and settles the goals completed by
hand (see Closing). It completes nothing itself. The code is in
`internal/goals/seed_sync.go`.
