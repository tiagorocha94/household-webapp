# 004 JSON API as the only interface

Status: accepted

Context: data is maintained in CSV files ([001](001-csv-is-the-durable-store.md)),
the services do not depend on HTTP, and the frontend
([009](009-frontend-svelte-spa.md)) is a separate program that only talks to
the API.

Decision: the server exposes only a JSON API under `/api/v1`. It renders no
HTML itself. It also serves the prebuilt static files of the frontend (embedded
in the binary) at `/`, so the UI and the API share one origin. The API is
read-only, with one exception: enrolled phones send cashflow movements to
`POST /api/v1/sync/cashflow` ([011](011-devices-write-cashflow-movements.md)).
Login and logout use a session cookie, so the web API suits same-origin
clients; phones use a device token. The wire format is documented in
[openapi.yaml](../architecture/openapi.yaml).

Consequences: no business logic lives in the HTTP layer and authorization stays
in the services. Writes from the web interface are not planned: data changes go
through the CSV files. How a request is handled is in
[architecture.md](../architecture/overview.md#request-flow).
