# Cashflow sync

Enrolled phones add, change and delete cashflow movements through one endpoint,
`POST /api/v1/sync/cashflow` ([openapi.yaml](../../architecture/openapi.yaml),
[ADR 011](../../decisions/011-devices-write-cashflow-movements.md)). It is the
only endpoint that changes data. Data flows from the phone to the server only.

## Who may call it

Only a device token sent as `Authorization: Bearer <token>`
([authentication](../../architecture/authentication.md#devices)); a web session
does not work. The token determines the movement owner, so the body never names
a person. Device tokens also work on the explicitly opened read routes listed
in the authentication guide.

## Request and answer

```json
{ "operations": [
  { "op": "upsert", "syncId": "3f2b8c1e-5d4a-4e7b-9a10-6c2d8e4f1a07",
    "category": "groceries", "date": "2026-03-05", "amountCents": 4000, "description": "weekly shop" },
  { "op": "delete", "syncId": "0b8a7d5e-2c1f-4a39-8e60-1d4f7a9c3b52" }
] }
```

```json
{ "results": [
  { "syncId": "3f2b8c1e-5d4a-4e7b-9a10-6c2d8e4f1a07", "status": "ok" },
  { "syncId": "0b8a7d5e-2c1f-4a39-8e60-1d4f7a9c3b52", "status": "rejected", "reason": "..." }
] }
```

- Up to 200 operations per request, applied in order. The answer has one result
  per operation, in the same order. More than 200, a body that is not this JSON,
  or a missing `operations` list answers 400 and nothing is applied. An empty
  list is fine.
- **`upsert`** adds the movement, or changes the one with the same `syncId`. The
  values follow the same rules as the CSV import: a real calendar day, an
  amount above zero (the category decides the direction), a category that
  exists, and a lowercase UUID. The description is trimmed.
- **`detail`** (upsert, optional) says where the money went, in the same
  request and the same transaction as the movement, so they arrive together or
  not at all. A detail holds one kind (savings, investments or credit):
  `"detail": {"savings": {"accountId": 3, "valueCents": 120000}}` adds a
  deposit of the movement's amount, in the movement's month, to that account.
  `valueCents` is the value seen now and is optional; leaving it out records
  nothing. Leaving `detail` out keeps the detail the movement has, and it
  follows a change of the amount or the date; `"detail": {}` removes it. The
  category must have the `savings` role, and the account must be the person's
  own or have no owner, otherwise the whole operation is `rejected` and the
  movement is not written. A deposit is written to the savings tables and
  `savings_*.csv` is rewritten after the batch, with the same "behind" handling
  as the movements file ([ADR 021](../../decisions/021-details-of-money-moved.md)).
  Investments:
  `"detail": {"investments": {"entityId": 2, "lines": [{"instrumentId": 5, "amountCents": 30000, "valueCents": 410000}, {"instrumentId": 6, "amountCents": 20000}]}}`
  puts each line into that instrument as a contribution in the movement's
  month. The lines may add up to less than the movement while the split is
  open, never to more, and sending the detail replaces the lines the movement
  had. The category must have the `investments` role, the account must be the
  person's own or have no owner, and every instrument must belong to it. A
  later change of the movement without a `detail` moves the contributions to
  its month, and is refused if it makes the movement smaller than the lines.
  `investment_contributions.csv` and `investment_snapshots.csv` are rewritten
  after the batch.
  Credit: `"detail": {"credit": {"creditId": 1, "principalCents": 25000, "interestCents": 5000, "extraordinary": false}}`
  adds a payment on that credit in the movement's month. Credits have no
  owner, so anyone can add one. The principal and the interest must add up to
  the movement's amount, the month cannot be before the credit starts, and a
  regular payment is one per credit and month (an extraordinary one can share
  the month). Sending the detail replaces the payment; a later change of the
  movement without a `detail` moves it to the new month, and is refused if the
  amount no longer matches, or the month already has a regular payment.
  `credit_payments.csv` is rewritten after the batch.
- **`delete`** removes the movement and its detail; snapshots stay. Deleting one that does not exist is fine.
- Both are **idempotent**: sending a batch again changes nothing, not even the
  CSV revision, so a phone that lost the answer can simply send it again. If the
  same `syncId` appears twice in a batch, the last operation wins.
- An operation that is refused is `rejected` with a reason and does not stop the
  others. Refusals: a bad value, an unknown category, an unknown operation, and a
  `syncId` that belongs to another person (a device can neither change nor
  delete another person's movement). `ok` also covers an operation that had
  nothing to change.
- The batch is one database transaction. If the database itself fails, nothing
  is kept, the answer is 500, and the phone sends the batch again.

## The CSV file

After a batch that changed something the server rewrites
`cashflow_movements.csv` from the database before it answers
(`cashflow.CSVMirror`, see [csv](csv.md)). If the file cannot be written, the
changes stay in the database, the problem is logged, and the file stays marked
as behind: the next batch or the next start writes it, and a start never
imports a file that is behind. The answer is still 200, because the movements
are saved.

## Versions

`GET /api/v1/info` reports `syncApi`, the version of this protocol (now 1). It
goes up when a change would break an app built for the old one, so the phone can
tell an old app from a new server.
