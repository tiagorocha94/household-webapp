# Releasing

A release is a `vX.Y.Z` git tag on `main`. The binary learns its version from
the tag (see [the README](../../README.md#quick-start)), so nothing in the code
changes at release time.

## Before tagging

1. CI is green on `main`, and `make check` and `make lint` pass locally.
2. In [CHANGELOG.md](../../CHANGELOG.md), rename `## [Unreleased]` to
   `## [vX.Y.Z] - YYYY-MM-DD` (the day you tag) and add a new empty
   `## [Unreleased]` above it. Only user-facing changes are listed
   ([rules](changelog-guidelines.md)).
3. Set `info.version` in [openapi.yaml](../architecture/openapi.yaml) to `X.Y.Z`.
   `TestOpenAPIVersionMatchesTheLatestRelease` fails when it differs from the
   newest version in the changelog.
4. Commit these as `chore: prepare vX.Y.Z` and push.

## Tagging

```bash
git tag -a vX.Y.Z -m "vX.Y.Z"
git push origin vX.Y.Z
make build && ./household-app -version   # prints vX.Y.Z
```

Commits after the tag report `vX.Y.Z-<commits since>-g<hash>`, and `-dirty`
when the working tree has uncommitted changes.

## Versioning

While the API is read-only and changes together with its only client, bump the
minor version for new endpoints or fields and for breaking changes, and the
patch version for fixes. Settle on stricter rules when a separate frontend
depends on the API.
