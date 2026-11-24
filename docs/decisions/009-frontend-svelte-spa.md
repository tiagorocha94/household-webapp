# 009 The frontend is a Svelte single-page app in `web/`

Status: accepted

Context: the API ([004](004-json-api-only-interface.md)) had no client. The
frontend should be light on boilerplate (one file per screen, no module
ceremony), stay out of the way of the Go code, and not require Node to build
or run the server.

Decision: the frontend lives in `web/` of this repository as a self-contained
Svelte 5, Vite and TypeScript project, built to static files. It is a plain
single-page app with hash routing, so any static file server can host it.
Its API types are generated from `docs/architecture/openapi.yaml` and committed, and
CI fails when they are out of date. Styling is plain CSS and charts are small
hand-written SVG components, to keep dependencies few. The UI was English at
first (it also has a Portuguese version since, see
[frontend.md](../architecture/frontend.md)), shows money in euros with Portuguese
number formatting, and targets desktop first.
The frontend is embedded in the Go binary (`web/embed.go`) and `make run` is
the one way to run it, so there is no `server.static_dir`, no separate dev
server against the real API, and no CORS.

Consequences: a Go-only checkout builds and tests without Node. One commit can
change the API, the OpenAPI file and the UI together. UI work stays inside
`web/`, which has its own `AGENTS.md`, and CI runs the web checks only when
`web/` or the OpenAPI file change. If the project outgrows one repository,
`web/` depends only on HTTP and the OpenAPI file, so it can move out.
Details are in [frontend.md](../architecture/frontend.md).
