# Changelog guidelines

`CHANGELOG.md` is a user-facing document. It describes what a user of the
application can see and do. It is not a developer log. Update it only when a
commit changes that; commits that only change internal code, tooling,
documentation or tests do not touch it. The release steps are in
[releasing.md](releasing.md).

## Include

- New features visible to the user (API endpoints, fields, behavior, screens)
- Changes to existing user-facing behavior, including changes to the CSV files
  and the configuration a user edits
- Bug fixes that affect user experience
- Removed features that users previously had access to

## Leave out

- Internal tooling (CLI commands, Makefile, seed/export)
- Technical implementation (SQL views, indexes, constraints, schema changes)
- Infrastructure and dependencies (cross-platform support, libraries)
- Developer-only features and code refactors
- Security improvements not visible to the user (unless they change login flow)

## Style

- Write from the user's perspective: "Create goals" not "Added goal creation handler".
- Describe behavior, not implementation: "View account balance" not "SQL view computes balance".
- Group related changes under feature headings, one concise line per change.

## Examples

Good:
- Read goal accounts, balances and projections through the API
- Read credits and their payment history
- Goal allocations add up exactly to the goal target

Bad:
- Added SQL views for balance computation
- Cross-platform Makefile with run/sync/reset commands
- Default users: bob/pw and alice/pw
- CSV seed and export CLI commands
