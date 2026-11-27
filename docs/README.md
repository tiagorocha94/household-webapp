# Documentation

| Folder | What is in it |
|--------|---------------|
| [architecture](architecture/overview.md) | How the system is built: [overview](architecture/overview.md) · [CSV seed and export](architecture/csv-seed.md) · [database backups](architecture/backups.md) · [authentication](architecture/authentication.md) · [database conventions](architecture/database.md) · [frontend](architecture/frontend.md) · [API reference](architecture/openapi.yaml) |
| [development](development/workflow.md) | Working on the code: [workflow](development/workflow.md) · [code style](development/code-style.md) · [changelog guidelines](development/changelog-guidelines.md) · [releasing](development/releasing.md) · [backlog](development/backlog.md) |
| modules | [Home overview](modules/home/overview.md) · [Goals](modules/goals/overview.md) ([schema](modules/goals/schema.md), [csv](modules/goals/csv.md)) · [Credits](modules/credits/overview.md) ([schema](modules/credits/schema.md), [csv](modules/credits/csv.md)) · [Investments](modules/investments/overview.md) ([schema](modules/investments/schema.md), [csv](modules/investments/csv.md)) · [Savings](modules/savings/overview.md) ([schema](modules/savings/schema.md), [csv](modules/savings/csv.md)) · [Cashflow](modules/cashflow/overview.md) ([schema](modules/cashflow/schema.md), [csv](modules/cashflow/csv.md), [sync](modules/cashflow/sync.md), [budget groups](modules/cashflow/budgets.md)) |
| [decisions](#decisions) | One record per lasting choice, below |

## Conventions for these docs

- Each topic has one home; other pages link to it instead of repeating it.
- Modules usually have `overview.md`, `schema.md` and `csv.md`; cross-module
  behavior gets a focused page such as cashflow sync or budgets.
- The exhaustive route list and wire format live only in
  [openapi.yaml](architecture/openapi.yaml); behavior pages may name a route
  when it clarifies a rule.
- Decisions say why, not how: one record per choice, and rules and procedures
  belong in the pages above.
- Project structure listings use alphabetical order with folders first.

## Decisions

- [001 CSV is the durable store](decisions/001-csv-is-the-durable-store.md)
- [002 Money as integer cents](decisions/002-money-as-integer-cents.md)
- [003 Migrations are fix-forward](decisions/003-fix-forward-migrations.md)
- [004 JSON API as the only interface](decisions/004-json-api-only-interface.md)
- [005 Modules own their seed logic](decisions/005-modules-own-their-seed-logic.md)
- [006 Feature modules are layered packages](decisions/006-layered-feature-modules.md)
- [007 Dependencies, including the logger, are injected](decisions/007-inject-dependencies-and-logger.md)
- [008 Standard library router and plain middleware](decisions/008-standard-library-router.md)
- [009 The frontend is a Svelte single-page app in `web/`](decisions/009-frontend-svelte-spa.md)
- [010 Household data is shared by every user](decisions/010-household-data-is-shared.md)
- [011 Enrolled devices write cashflow movements by sync ID](decisions/011-devices-write-cashflow-movements.md)
- [012 Enrolled devices read the goals endpoints](decisions/012-devices-read-goals.md)
- 013 was withdrawn before it was accepted; the number is not reused.
- [014 Savings are their own module](decisions/014-savings-module.md)
- [015 Enrolled devices read the budget by group](decisions/015-devices-read-budget.md)
- [016 Home says what needs a look](decisions/016-home-says-what-needs-a-look.md)
- [017 Seed data lives in the private repository](decisions/017-seed-data-in-a-private-repo.md)
- [018 Cash is worked out from typed bank balances](decisions/018-cash-balance-from-typed-balances.md)
- [019 Enrolled devices read their owner's cash balance](decisions/019-devices-read-their-own-balance.md)
- [020 The phone app is the household app](decisions/020-the-phone-app-is-the-household-app.md)
- [021 Details of money moved are written from the phone](decisions/021-details-of-money-moved.md)
- [022 Enrolled devices read savings, investments and credits](decisions/022-devices-read-savings-investments-credits.md)
- [023 Goals are closed by hand and settle to zero](decisions/023-goals-are-closed-by-hand.md)
