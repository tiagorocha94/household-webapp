# Frontend

The frontend is a Svelte 5, Vite and TypeScript single-page app in `web/`
([ADR 009](../decisions/009-frontend-svelte-spa.md)). It is read-only and talks
only to the [JSON API](openapi.yaml).

## Setup

Install Node (the version in `web/.nvmrc`), then from the repository root:

```bash
make run   # builds the interface, then the server, and serves both on :8080
```

There is one way to run it: the Go binary embeds `web/dist` and serves it at `/`
next to the API, on one port and one origin, which is also how it runs for real
(so the session cookie and the API need no CORS or proxy). After a change to
the interface, run `make run` again. `make web-build` only builds the interface.

## Screens

The pages follow one pattern: the overview is a summary, each module page opens
with its totals and a list, an item page holds the detail, and a long detail gets
sub-pages (a month of cashflow has a summary and pages for the budget by group,
categories, people and movements). Navigation is a menu on the left with an icon
for each module; it collapses to icons (remembered in the browser) and is a
drawer below 56rem.

The web and the phone app share their look where they show the same thing: the
green accent, the menu icons of the pages both have (Savings is the vault,
Credits the landmark), the names of figures (such as Free and Balance in goals),
euro and date formats, and the "Reading your …" and "Updated … · Refresh"
lines. The web keeps its own serif headings and wide tables. The main read of a
page is marked with `what` on `Async`, which gives it the loading text and the
Updated line; reads inside a page leave it out.

| Address | Screen |
|---------|--------|
| `#/notifications` | Notifications: everything that needs a look, grouped as data to enter or fix and plan and budget, from the overview's `attention` list |
| `#/` | Overview: how many things need review (linking to Notifications), key figures, this month's cashflow and budget groups, active goals, credits and investments, from one request ([Home overview](../modules/home/overview.md)) |
| `#/net-worth` | Net worth: the current figure, assets, credit owed and cash, a chart of assets, credit and net worth over 6 months, 12 months or all, and the last six months as a table, from `GET /api/v1/overview/net-worth` ([ADR 018](../decisions/018-cash-balance-from-typed-balances.md)) |
| `#/goals`, `#/accounts/:id`, `#/goals/:id` | Goals (totals, goal accounts), one account (goals, projection, movements), one goal |
| `#/savings`, `#/savings/:id` | Savings (totals, accounts), one account (balance by month, movements, snapshots) |
| `#/credits`, `#/credits/:id` | Credits (totals, active and paid off contracts), one contract with its payments |
| `#/investments`, `#/investments/entities/:id`, `#/investments/instruments/:id` | Investments (totals, where the money is, entities), one entity with its instruments, one instrument (history, contributions, value snapshots) |
| `#/cashflow` | Cashflow: income and expenses by month and year, the months |
| `#/cashflow/months/:month` | A month's summary: totals, budget by group, top categories, latest movements |
| `#/cashflow/months/:month/budget`, `/categories`, `/people`, `/movements` | The full budget by group (each group opens to its categories), categories (spending and income), by person (spent, put aside and cash per person, and the last six months of what each put aside), all movements |

Every screen shows loading, empty and error states. Addresses use the hash, so
the Go server needs no routing for them. Table headers sort on click
(ascending, descending, then the table's own order), through `SortHeader` and
`src/lib/sort.ts`.

## Language

The UI is English or Portuguese (Portugal). It starts in the browser's language
(Portuguese when that is the first of the two the browser prefers, otherwise
English) and an EN/PT switch (`LanguageSwitch`, on the sign-in page and in Settings)
remembers a choice in this browser's local storage under `household.language`. The texts are in `src/lib/i18n/messages/<area>.ts`, one
file per area (shell, cashflow, goals, investments), each with an English and a
Portuguese object; `t(key, params)` returns the current language's text and is
reactive. A test checks that both languages have the same keys and placeholders.
Names that come from the CSV files (categories, budget groups) keep their English
name as the identifier and are shown with `categoryLabel()` and `groupLabel()`;
names the people typed (goals, accounts) are never translated. Months and days
follow the language; euros are Portuguese-formatted in both. The words match the
phone app's where both show the same thing.

## Settings

The gear in the top bar opens `#/settings`, where the person picks how the app
looks in this browser: theme (System, Light, Dark), language (System, EN, PT),
text size (Normal, Large, Larger), table rows (Comfortable, Compact) and accent
colour (green, blue, purple, pink). Everything is the System or default choice at
first. Choices live only in this browser's local storage (`household.theme`,
`household.textSize`, `household.rows`, `household.accent`, `household.language`),
never on the server, and a default is stored as nothing. "Reset" clears them all.

They work through attributes on the root element (`data-theme`, `data-text-size`,
`data-rows`, `data-accent`) that `app.css` reads; no attribute means the system
decides. `src/lib/appearance.ts` holds the choices and the mapping to attributes
(tested), `appearance.svelte.ts` the reactive state, and `public/early.js` sets
the same attributes before the first paint (the CSP forbids inline scripts); a
test keeps the script and `appearance.ts` in step. Text size scales the root font
size, so sizes in CSS are `rem`, not `px`. Accent colours are shared with the
phone app, and each has a light and a dark value that reads at 4.5:1 on the page
(tested).

## Without Node

A binary built without the interface (`make build` before any `make web-build`,
or `go run ./cmd/server/`) holds only a placeholder and serves just the API; it
logs a warning saying so.

## Mock API

`npm --prefix web run dev:mock` starts a stand-in API on :8080 with sample data
(`web/dev/fixtures.ts`, typed against the API contract). Log in as `alex` /
`alex`. Use it to work on the UI without the Go server or any data.

## Commands

| Command | What it does |
|---------|--------------|
| `make run` | Build the interface and the server, then serve both |
| `make web-build` | Build static files into `web/dist` (installs the packages when missing) |
| `make web-check` | Format check, `svelte-check`, tests and build, as CI runs them |
| `make web-types` | Regenerate `web/src/api/schema.d.ts` from `docs/architecture/openapi.yaml` |

`make check` and `make lint` cover the Go code only and need no Node.

## API types

`web/src/api/schema.d.ts` is generated from the OpenAPI file and committed.
After an API change, update `docs/architecture/openapi.yaml`, run `make web-types` and
commit both together.
