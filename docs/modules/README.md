# Modules

The server is made of five feature modules, and each one can run on its own:
[cashflow](cashflow/), [goals](goals/), [savings](savings/), [credits](credits/)
and [investments](investments/). The authentication, the Home overview and the
shell of the web app are always there.

## Choosing the modules

List the ones to run in `config.yaml`; leave `modules` out to run all of them:

```yaml
modules: [cashflow, goals]
```

For a module that does not run:

- its API routes answer 404, even without a session, and `POST /api/v1/sync/cashflow`
  refuses a detail meant for it;
- its CSV files are neither read nor written, so they can stay in the folder;
- `GET /api/v1/info` leaves it out of `modules`, and the web app hides its pages;
- the Home overview and the net worth leave out its figures (see
  [Home overview](home/overview.md#when-a-module-does-not-run)).

The database always has every table, so turning a module on later needs no
migration. An unknown name stops the server at start.

## How they stay independent

- A module never imports another module. `depguard` in `.golangci.yml` checks
  it, and tests are the only exception, because they wire modules together.
- Cashflow tells the modules where the money of a movement went through package
  `details`, which holds only the contract and a registry. Savings, investments
  and credits each add an applier (`internal/<module>/detail`) that writes into
  their own tables, and `cmd/server` registers the ones that run.
- Three places bring modules together on purpose: `cmd/server` (it starts
  them), `internal/overview` (the Home page) and the phone app. All of them
  treat a module that does not run as absent.
- Some CSV columns are plain links by ID between modules, for example the
  `movement` column of the savings, investments and credits files, which holds
  the sync ID of a cashflow movement. They are only text; without cashflow they
  point at nothing and are kept as they are.

## Adding a module

1. Keep it in `internal/<module>` with its own tables, services and CSV files,
   and give it a name in `internal/modules`.
2. Add its routes in `internal/server` inside the module's check, and its seed
   module in `cmd/server` when it runs.
3. Add its pages to `NAV` in `web/src/lib/nav.ts` with its `module`, and to
   `moduleOfRoute` in `web/src/lib/modules.ts`.
4. Add it to the `depguard` rules and to the module sets in
   `TestDemoDataStartsWithAnySetOfModules`.
