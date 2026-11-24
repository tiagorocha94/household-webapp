# 005 Modules own their seed logic

Status: accepted

Context: when a central package knows every module's tables and CSV rules,
adding a module means editing it, and rules that belong to one domain end up
duplicated or split across packages.

Decision: the `seed` package only orchestrates the startup data flow. It reads
and header-checks each module's files, hands the rows to the module inside a
transaction, runs the module's sync and writes its export. A module provides
the files it owns, an import, a sync that receives the current time and an
export, by satisfying the small `seed.Module` interface from its own
`seed.go`. Modules do not import `seed`, nor each other's data.
`cmd/server/main.go` lists the modules. Shared CSV and mirror helpers live in `csvdata`; month
arithmetic lives in `month`.

Consequences: a new module does not touch `seed`, and `seed` is tested with a
fake module. Tests that need several modules together live in `cmd/server`.
The flow itself is described in [csv-seed.md](../architecture/csv-seed.md).
