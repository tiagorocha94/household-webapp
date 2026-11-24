# Savings schema

Migration `0008_create_savings_module.sql`.

```
users
  └─ savings_accounts (user_id, ON DELETE SET NULL)
       ├─ savings_movements (account_id, ON DELETE CASCADE)
       └─ savings_snapshots (account_id, ON DELETE CASCADE)
```

- `savings_accounts`: `name` unique, `target_cents` null or positive.
- `savings_movements`: `type` in (`deposit`, `withdrawal`), `amount_cents`
  positive.
- `savings_snapshots`: `value_cents` zero or more.

There are no views: the figures are derived in Go (`savings.Summarize`) from
the movements and snapshots of each account.
