# Changelog

User-facing changes only. Describes what users can see and do in the application. Internal implementation details, tooling and refactors do not belong here. See [changelog guidelines](docs/development/changelog-guidelines.md).

## [Unreleased]

### Database backups
- Keep a daily copy of the database by setting `backup` (`dir` and `keep`) in `config.yaml`. The copy is made while the server runs, at start-up and once a day, and the oldest ones beyond `keep` are removed. Without `keep` nothing is backed up.

### Background jobs
- A running server now creates the goal allocations of a new month by itself, once a day, so it no longer has to be restarted at the start of each month. It also forgets expired login failures every 15 minutes. Each job has its own schedule (see [Background jobs](docs/architecture/overview.md#background-jobs)).

## [v0.2.0] - 2026-11-26

The server can now run only some of its features, for a household that wants, say, just the cashflow.

### Modules
- The server can run only the features you choose with `modules` in `config.yaml` (cashflow, goals, savings, credits, investments); by default all of them run, and `GET /api/v1/info` lists the ones that do. A feature that is left out has no pages in the web app and its CSV files are not used.
- A phone's movement that says where its money went (a savings deposit, investment contributions or a credit payment) is refused when that feature is not running; movements without such a detail are still saved.
- The Home page and Net worth show only the features that are running: figures, lists, warnings and columns of a feature that is left out are not there.
- Each module can run alone: the demo data is tested with every module on its own and in groups, and the linter rejects a module that imports another. See [docs/modules](docs/modules/README.md).

## [v0.1.0] - 2026-11-24

First public release. The household's data is kept in CSV files that the server imports on start and writes back; the web app and the JSON API read it, and enrolled phones add cashflow movements.

### Cashflow
- A ledger of money in and out, per person and per month: what came in, went out and was left over, where it went by category, and the movements behind each total.
- A monthly budget by group (essential, goals and savings, investments and so on), set from the month's income, with how much of each group was used.
- A phone can send movements, with the savings account, investment accounts or credit payment each one went to; the server saves them and keeps the CSV files in step.

### Goals
- Goal accounts with monthly deposits, goals that are funded by monthly allocations, a balance per goal, a projection of when each will be reached and the money not yet given to any goal.

### Credits
- Credit contracts with every payment split into principal and interest, the outstanding principal and the total cost, and the payments that are due.

### Investments
- Accounts (a person's or the household's) with their instruments, the money put in, what each holding is worth in monthly snapshots and the gain against money in.

### Savings
- Savings accounts with deposits, withdrawals and observed balances, and progress towards an optional target.

### Overview
- A home page with the month at a glance, what needs attention, and a net worth history.

### Web app
- Light and dark themes, a Portuguese version that follows the browser's language, and a Settings page for the theme, language, text size, table rows and accent colour, kept in the browser only.
- Pages say what they are reading and when they were last refreshed.

### Phones
- Phones are enrolled with a one-time code and then read goals, credits, investments, savings and the budget with a device token, and send cashflow movements.
