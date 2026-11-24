# 006 Feature modules are layered packages

Status: accepted

Context: a flat package per feature mixes HTTP, business and database code, and
a single folder of many files is hard to navigate.

Decision: each feature module keeps its domain types and pure logic in the
package root, with `store/`, `service/` and `api/` sub-packages for database
access, business rules and authorization, and HTTP handlers with their wire
types. Sub-packages import the root, never the reverse. The root also holds the
module's `seed.go` ([005](005-modules-own-their-seed-logic.md)).

Consequences: dependencies flow handler, service, store, and wiring sites alias
the sub-packages by module. One struct still serves store and service; separate
models per layer are in the [backlog](../development/backlog.md) for when computed fields
make the shapes differ. The layout is described in
[architecture.md](../architecture/overview.md#modules).
