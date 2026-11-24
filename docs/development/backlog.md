# Backlog

Open limitations and ideas. Completed designs live in their module pages and
ADRs rather than here.

## Net worth

- Add assets no module records, such as property. Credit owed is currently
  subtracted without counting the property it financed.

## Savings

- Optionally warn when an account or instrument expected to receive monthly
  snapshots has none for the current month.
- Decide which main account saved money comes from.
- Decide whether saved amounts should contribute to the savings rate.

## Operations

- Add a smoke test that migrates a database or seed from the previous release.
  Current tests apply every migration only to an empty database.

## API and security

- Add per-route body limits and timeouts if more write endpoints appear. Today
  every route uses the shared 1 MiB and 10-second limits.
- Rate limits see only the proxy address behind `tailscale serve`. Trust a
  forwarding header for device limits if that becomes a problem.
- Web writes remain undecided. Phone writes are intentionally limited to
  enrolled-device cashflow sync
  ([ADR 011](../decisions/011-devices-write-cashflow-movements.md)).

## Product

- Recalculate remaining goal allocations when extra money is added.
- Create goal allocations on the first request of a new month, or on a daily
  timer, instead of only at startup. Until then the server must be restarted at
  the start of each month.
- Complete a goal from household-app (only from its final month onward, and only
  once fully allocated), which today is done by hand in the data
  ([ADR 023](../decisions/023-goals-are-closed-by-hand.md)).
- Let a user dismiss one old notification that is not worth fixing. Today an
  item disappears only when its data changes or it leaves the three-month
  window ([ADR 016](../decisions/016-home-says-what-needs-a-look.md)).
- Add credit projections such as amortization, EURIBOR periods, insurance, fees
  and taxes.
- Chores, shared lists, a household calendar and household documents: see
  [Shared household modules](#shared-household-modules).
- Track taxes, assets and insurance policies.

## Shared household modules

Shopping lists, chores, a household calendar (maintenance, inspections,
insurance and tax renewals) and household documents. Nothing here is decided
beyond what is written; open points are settled when one of them is picked.

**Shape**
- New modules are expected to live only in the database, with no CSV file, so
  [ADR 001](../decisions/001-csv-is-the-durable-store.md) would not apply to
  them. This is the default, not a rule: a module nobody has thought of yet may
  fit the existing CSV or sync designs better. Existing modules and the cashflow
  sync stay as they are.
- Changes go phone to server over the HTTP API, or are made on the server
  directly. Any enrolled device may write this shared data
  ([ADR 010](../decisions/010-household-data-is-shared.md)); the one-person rule
  of [ADR 011](../decisions/011-devices-write-cashflow-movements.md) is for
  cashflow only. The web app keeps reading the database on each navigation and
  needs no polling.
- Offline first is the default for everything on the phone, not only these
  modules: with the server unreachable the phone shows the data it last read
  (goal data included) and, for data it writes, keeps an outbox of unsent
  changes and a cursor, like cashflow.

**Receiving changes**
- Pull only, and only when the app opens or returns to the foreground (and after
  the phone's own sends). No SSE, no timer, no updates while the app is closed.
- Every record has a UUID made by the device, so a retried send changes nothing,
  and a revision number from the server that only goes up. A deleted record is
  kept as a marker so phones learn it is gone.
- One endpoint per module, `GET /api/v1/<module>/changes?since=N`, returning that
  module's changed records, its deletions and a new cursor. Each module has its
  own revision counter and its own cursor on the phone, so a module can be added,
  changed or left unsynced without touching the others, and a phone only asks for
  the modules it shows. A phone whose cursor is older than the markers kept
  fetches that module again in full.
- Optionally one small endpoint returning the latest revision of every module,
  so the phone can ask once on opening and fetch only the modules that moved.

**Conflicts**
- Two people mostly add different records, so nothing clashes: each appears on
  the other's phone.
- The same record changed twice: the last edit wins, per record, by the edit
  time on the phone (never later than the server's clock), the server's revision
  breaking ties.
- If real inconsistencies ever show up, keep both versions pending and let
  someone approve one in the web app. That would be the first web write, so it
  is decided only if the need appears.

**Calendar**
- A calendar the app owns, shared between the devices by the same mechanism.
  Linking an external calendar (Google, CalDAV) is a separate and larger
  project.

**Documents**
- Copies of ID cards, insurance, house papers and the like are stored on the
  server and downloaded to the phones. Nothing is sent from a phone, and editing
  a document is not synced.
- Open: where the files live and how they get there, whether some are visible
  to one person only (they are sensitive, and ADR 010 shares everything), whether
  a phone keeps downloaded copies for offline use, and how the files are
  backed up.

**Backups**
- Out of scope for now. SQLite stays. Since the CSV files were the off-machine
  copy, new modules need another (for example a scheduled dump committed to the
  private data repository), and a different database would be a separate decision.

## Architecture

Separate store, service and API models when their shapes meaningfully diverge.
Today one struct usually serves store and service; adding mappings before more
computed fields appear would create boilerplate without benefit.

The completed phone-app work is documented in
[ADRs 020–022](../decisions/020-the-phone-app-is-the-household-app.md) and in
the household-cashflow design guide.
