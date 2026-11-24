# 023. Goals are closed by hand and settle to zero

Date: 2026-11-20

## Context

A goal is money set aside to be spent: its cost is estimated, then it is
allocated every month so that the amount is there at the final date. It is not
a savings or emergency account, so a goal is expected to end near zero once its
expenses are done. Those expenses often come one or two months after the last
allocation, and sometimes they go over the target.

Until now sync completed a goal on its own once its end month had passed and
refunded what was left. That closed goals before they were spent, and a goal
that overspent was never settled.

## Decision

- Nothing completes on its own. A goal is completed by setting its status to
  `completed` in the CSV. Closing from the app is left for later and is meant
  to be allowed only from the final month on, once the goal is fully allocated.
- Sync settles a completed goal once. A leftover is returned to the unallocated
  money as a `release` movement. An overspend is taken from the unallocated
  money as a `cover` movement. Either way the goal ends at zero and the ledger
  still records every movement. Amounts stay positive, so two types carry the
  two directions. `release` is the former `refund`.
- The goal keeps `closing_balance_cents`: the balance it had just before the
  closing movement (positive, or negative when overspent). It is null while the
  goal is not completed. It follows from the closing movement, so it is not a
  CSV column and the CSV files keep their headers.
- Clients show the balance green above zero and red below it, and warn when the
  spent amount is over the target. The warning changes nothing else.

## Consequences

- The `goal_account_movements` constraint changed, so migration 0021 rebuilds
  the table and turns `refund` rows into `release`. The CSV still accepts
  `refund` as `release`.
- The API drops `refundedCents` and `refundsCents` for `releasedCents` and
  `coveredCents`, and a goal gains `closingBalanceCents`.
- Goals that the old sync completed already hold a refund. Sync reads it as
  the closing movement and sets the closing balance to its amount.
