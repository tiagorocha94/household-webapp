# AGENTS.md (web)

Guidance for AI agents working on the frontend in `web/`. Read this file and
[docs/architecture/frontend.md](../docs/architecture/frontend.md); you do not need the Go code. The API
contract is [docs/architecture/openapi.yaml](../docs/architecture/openapi.yaml).

## Stack

Svelte 5 (runes: `$state`, `$derived`, `$props`), Vite, TypeScript, plain CSS.
A single-page app with hash routing. Read-only: it never changes data.
Dependencies are few on purpose; do not add one without discussion. Approved:
`openapi-fetch`, and `@lucide/svelte` (pinned to an exact version, the same as the
mobile cashflow app) for category icons.

## Commands (from the repository root)

- `make run`: build the interface, embed it in the server and serve both on :8080
  (the one way to run it; it installs the packages when missing)
- `make web-build`: only build `web/dist`
- `make web-check`: format check, `svelte-check`, tests and build, as CI runs them
- `make web-types`: regenerate `src/api/schema.d.ts` from the OpenAPI file

## Rules

- One `.svelte` file per screen or component; scoped `<style>`; shared design
  tokens (CSS variables) live in `src/app.css`.
- Never edit `src/api/schema.d.ts`; change the OpenAPI file and regenerate it.
- Money arrives as integer cents (`...Cents`); format it only through
  `src/lib/money.ts`. Months arrive as `YYYY-MM`; format them only through
  `src/lib/month.ts`.
- Icons are Lucide icons, imported only in `src/lib/icons.ts` (add a name there
  before a category uses it) and drawn through `CategoryLabel.svelte`. No emojis
  as icons.
- Every screen handles loading, empty and error states.
- Tables with data rows get sortable headers: `SortHeader` with `src/lib/sort.ts`.
- The UI is English and Portuguese (Portugal); euros are always formatted the
  Portuguese way; desktop first. Every user-facing text lives in
  `src/lib/i18n/messages/<area>.ts` (English and Portuguese side by side, same
  keys and placeholders; a test checks it) and is shown with `t()`. Never write
  a text into a component, never call `t()` at module level (the language can
  change), and show category and budget group names with `categoryLabel()` and
  `groupLabel()`. Words are the phone app's where both show the same thing.
- Colours come from the CSS tokens in `app.css` (never a raw colour in a
  component) and font sizes are `rem` (never `px`), so the theme, accent and
  text size settings reach everything. How the app looks (theme, language, text
  size, table rows, accent) is a per-browser choice kept in local storage, not
  on the server; see Settings in `docs/architecture/frontend.md`.
- Keep logic that can be tested out of components, in `src/lib`, with a
  `*.test.ts` next to it.
- Components are tested by rendering them to a string with `render` from
  `svelte/server`, with no extra dependency (`src/components/*.test.ts`). This
  checks what a component shows for given props; clicks are not tested. A
  component that reads `window` at import time (for example through the router)
  cannot be rendered this way.
- Formatting is Prettier (`npm --prefix web run format`).
