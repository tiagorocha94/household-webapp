# 021 Details of money moved are written from the phone

Status: accepted and built in the server and phone app

Context: a cashflow movement in a category with a role (savings, investments,
credits) says that money left the account but not where it went. Home warns
when the month's total in such a category is larger than what the module holds
([ADR 016](016-home-says-what-needs-a-look.md)), and the person types the
detail later on the web. The detail is known when the movement is added, but
the phone can only write cashflow movements
([ADR 011](011-devices-write-cashflow-movements.md)), and that rule assumes
one writer for one person's data.

Decision:
- **A movement can carry the detail of where it went.**
  - **Savings:** the savings account, as a deposit. A movement is money
    leaving cashflow, so withdrawals are not entered from the phone.
  - **Investments:** one investment account (an entity, such as Trading212) and
    an amount for each of its instruments. An account can have several
    instruments and one movement can be split across them.
  - **Credits:** the credit, the principal and interest it paid, and whether it
    is the regular payment of the month or an extraordinary one.
  - **Goals:** nothing; they follow the movements by themselves.
- **The detail is always optional.** The money may not have arrived or been
  invested yet. A movement without its detail, or with only part of it, is
  kept and marked **Details missing**, from the moment it is created. What is
  known is kept: the lines of a split can add up to less than the movement
  while it is incomplete, never to more, and it is complete when they are
  equal. The form can save an incomplete split. A credit's principal and
  interest must add up to the amount.
- **Months, not days.** Cashflow movements keep their date. The detail is
  stored by month: the month of the movement's date, or another month if the
  detail is added later.
- **A value seen now is optional and never filled in.** An investment
  instrument or savings account can get a snapshot of its value, typed by
  the person. Nothing is recorded when it is left empty, so the last value
  stays. A reminder a few days later is a later step.
- **Recurring movements can name the account** and the amount for each
  instrument that is already known. The movements they create keep what is
  known, and the rest stays to be assigned. A value is never part of a
  recurring setup.
- **Only the owner changes what has an owner.** Savings accounts and
  investment accounts (entities) have an owner or none, and an investment
  account's instruments take its owner. Money can be added to one only from
  its owner's device, or from anyone's if it has no owner; credits have no
  owner, so anyone can add a payment. The server decides this from the
  device's person, and the phone lists only what the person can use. When
  only one choice is left, the phone selects it and disables the input.
- **The movement and its detail are one record.** They are sent together with
  one `sync_id`, so they arrive together or not at all, and the rows of the
  detail (the savings movement, the contributions, the credit payment) carry
  that `sync_id` as `movement` in the CSV files. Editing the amount
  reopens the split. Deleting the movement removes its detail; snapshots the
  person typed stay.
- **Writes go to the database and are exported to the CSV files.** For these
  modules, a write from a phone updates the database first and the module's
  file is rewritten after it, with the same "CSV out of date" handling as the
  cashflow file in [ADR 001](001-csv-is-the-durable-store.md). The files stay
  the copy that can rebuild everything; a test must show that exporting,
  resetting and importing gives the same data.
- **The emergency mark on savings withdrawals is removed.** Nothing used it.
- Rows that exist today are not linked to any movement and are left alone.

Consequences: a stolen device token could now change financial records, not
only add movements, so an app lock comes before the first write ships. A
second device of the same person writes the same records; the later write wins
for a record. The details of the wire format, the new routes and the device
reads of savings, investments and credits are decided when each is built.
