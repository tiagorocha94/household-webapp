# 002 Money as integer cents

Status: accepted

Context: floating-point amounts drift: parts stop adding up to their total, and
every comparison needs a rounding tolerance.

Decision: money is an integer number of cents at every layer: database
columns, Go (`money.Cents`), CSV files and the JSON API. Conversion to euros is
the client's job, and totals are split with `money.Split`.

Consequences: sums are exact and need no tolerances. A query or field that
still treats an amount as euros fails loudly. Naming and column rules are in
[database.md](../architecture/database.md#money-values).
