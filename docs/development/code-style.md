# Code style

Follow the [Uber Go Style Guide](https://github.com/uber-go/guide/blob/master/style.md)
as the primary reference, falling back to the
[Google Go Style Guide](https://google.github.io/styleguide/go/). The rules
below are the project's own additions and choices.

## Doc comments

Every exported type, function, method, constant and variable has a doc comment
that starts with its name and uses complete sentences.

## Standard library first

Use the standard library unless it makes code unreasonably complex. Do not
introduce dependencies without explicit discussion.

## Errors

- Always return errors, never ignore them. Handle each error once: return it
  wrapped, or log it at a boundary (`api.ErrorStatus`, `main`), never both.
- Wrap as `<verb> <object>[ <id>]: %w`: lowercase, no trailing punctuation, no
  "failed to" or "error" (Uber guide).
  `fmt.Errorf("list goals for account %d: %w", id, err)`
- Wrap sentinels with the operation and don't repeat their text:
  `fmt.Errorf("get goal %d: %w", id, apperr.ErrNotFound)`.
- Use `errors.New` for a message with nothing to format.
- CSV row diagnostics are `row %d: <problem>`.
- Use `errors.Is` and `errors.As` for checks.
- Never discard errors from `strconv.ParseInt`, `time.Parse` or similar;
  validate and return early.
- In API handlers, wrap malformed input with `apperr.ErrBadRequest`.
- When writing files, check `Close()` explicitly; do not defer it.

## Logging

- Use `log/slog`, never the `log` package. Only `cmd/server` creates the
  logger; everything else receives a `*slog.Logger` as a parameter (named
  `log`, or `logger` on a struct) and never calls the package-level `slog`
  functions ([ADR 007](../decisions/007-inject-dependencies-and-logger.md)).
  Tests pass `testutil.Logger()`.
- Constant lowercase message, details as key-value pairs, `snake_case` keys,
  the error under `"error"`: `log.Info("completed goal", "goal_id", id)`.
- Levels: Debug for per-file detail, Info for lifecycle and data changes, Warn
  for recoverable oddities, Error for unexpected failures.
- Never log passwords or session tokens.
- The level comes from `log.level` in `config.yaml` (default `info`).

## Declaration order

Within a file, top to bottom:

1. constants, then variables;
2. types;
3. for each type, in the order the types are declared: its constructor
   (`NewX`), its exported methods, then its unexported methods;
4. functions that belong to no type, exported first when it reads naturally.

`make lint` enforces this with the `decorder` and `funcorder` linters in
`.golangci.yml`.

## Naming

- Clear, descriptive names; avoid abbreviations except `db`, `ctx`, `cfg`.
- Package names are short, lowercase, singular.
- Database naming, money and month rules are in
  [database.md](../architecture/database.md).

## Linting

`make lint` runs both golangci-lint and govulncheck locally. In CI,
golangci-lint runs for relevant pushes and version tags, while govulncheck runs
weekly, when started manually, and for version tags. `make check` runs the local
format, build, vet and test checks. Run `make check` and `make lint` before
pushing.
