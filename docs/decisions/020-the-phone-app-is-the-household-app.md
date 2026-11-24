# 020 The phone app is the household app

Status: accepted and built; future areas require separate decisions

Context: the phone app was called Cashflow and did one job, adding movements.
It already shows goals, the budget and the balance, and the next steps add the
details of savings, investments and credits, and later maybe a calendar or a
shopping list. Several apps sharing data on one phone would need separate
sign-ins and syncs, and a second copy of the data would be the thing to keep
right.

Decision:
- **One app, called Household.** It keeps the Android application id
  (`io.github.tiagorocha94.cashflow`), because changing it would install a
  second app and lose the data on the phone. Only the name changes.
- **The server stays the one place the data lives.** The phone caches what it
  reads and queues what it writes, as before.
- **The menu is grouped by how often something is done.**
  - **Cashflow** comes first and is the main thing: Overview, Movements,
    Recurring, Graphs and Categories. Things added many times a month.
  - **Accounts**: Savings, Investments and Credits, read-only pages. They change
    from the movement that moved the money
    ([ADR 021](021-details-of-money-moved.md)), never from the page itself.
  - **Plans**: Goals and Budget, read-only.
  - **Settings** last. Calendar or Shopping would be groups of their own.
- Pages that read the server appear only while the phone is connected, as the
  goals and budget pages do today.

Consequences: adding an area means adding a menu group and its pages, not a new
app. The Overview page keeps the heading Cashflow, because it is the cashflow
page. The repository name and the server's own name are not changed.
