# Authentication

Every route under `/api/v1` needs a valid session cookie except login,
logout, enrolment and the sync endpoint (which takes a device token instead); the goals read endpoints take a session cookie or a device token (endpoints in [openapi.yaml](openapi.yaml)). There is no HTML
login page. It is built for personal use on a trusted network: no registration or
password management, and no TLS of its own. Known gaps are in the
[backlog](../development/backlog.md).

## Login

- The username is trimmed and lowercased (stored with `COLLATE NOCASE`); the
  password is checked against the bcrypt hash.
- An unknown user and a wrong password both answer 401 `Invalid
  credentials`. A malformed, empty or over 4 KiB body answers 400.
  An unknown user costs the same time as a wrong password, so the
  response time does not reveal which usernames exist.
- After 10 failed logins within 15 minutes from one client address, or for one
  username, further logins answer 429 with a `Retry-After` header until the
  window ends, even with the right password. A successful login clears the
  username's count but not the address's. The counts are in memory and use the
  connection's address, not forwarding headers.
- Success sets the `session_token` cookie and returns `{"id", "username"}`.

## Session

- A login creates a session with a random 32-byte hex token, sent in an
  `HttpOnly`, `SameSite=Lax` cookie with path `/`. Sessions expire 7 days after
  login (fixed, not sliding). They are kept in the `user_sessions` table as a
  SHA-256 hash of the token, so they survive a restart and a copy of the
  database cannot be used to log in. Expired sessions are removed at the next
  login, and `make reset` ends all sessions. Set `server.secure_cookies: true`
  when the server is reached over HTTPS to add the `Secure` flag.
- Logout answers 403 when the browser marks the request as cross-site
  (`Sec-Fetch-Site`, or `Origin` as a fallback), so another site cannot log
  the user out. Requests with neither header, such as curl, are allowed.
- Logout removes the token, expires the cookie and answers 204. It is
  idempotent and works without a session.
- `RequireAuth` injects the user into the context (`auth.UserFromContext`)
  or answers 401 JSON; it never redirects. Login and logout write their
  responses directly because they manage the cookie.

## Devices

A phone uses a device token instead of the web session cookie
([ADR 011](../decisions/011-devices-write-cashflow-movements.md)). The token can
write through [cashflow sync](../modules/cashflow/sync.md) and read the selected
goals, budget, owner-balance, savings, investment and credit routes opened by
[ADRs 012–022](../decisions/012-devices-read-goals.md). The exact routes are in
[OpenAPI](openapi.yaml).

`RequireSessionOrDevice` treats a bearer token as a device credential and any
other request as a web session. Routes that use `RequireAuth` remain
session-only.

- **Making a device.** `household-app device add [-config file] <user> <name>`
  prints a one-time enrolment code such as `K7QD2-M9XWP` (10 characters
  without 0, O, 1 or I) and, when `server.public_url` is set, the address to
  type in the app. The code is valid 15 minutes and works once. Only its
  SHA-256 hash is stored. The command works on the database directly, so it
  needs no running server, but the server must have started once so the users
  exist. `device list` shows every device with its status (`waiting`, `expired`,
  `enrolled`, `revoked`) and when it was last seen; `device revoke <name>` stops
  a device at once.
- **Enrolling.** `POST /api/v1/devices/enrol` with `{"code"}` exchanges a code
  for a random 32-byte hex token and answers `{"token", "username",
  "deviceName"}`. The token is shown only in that answer, has no expiry, and is
  stored as a SHA-256 hash. A wrong, expired, used or revoked code answers 401;
  a malformed or empty body answers 400. Codes ignore case, spaces and dashes.
  The exchange is a single statement, so a code cannot be used twice, and a
  failed attempt does not use it up.
- **Using the token.** `RequireDevice` reads `Authorization: Bearer <token>`,
  puts the device and its person in the context (`auth.DeviceFromContext`,
  `auth.UserFromContext`) and answers 401 with `WWW-Authenticate: Bearer`
  otherwise, including after a revoke. It never looks at the session cookie,
  and `RequireAuth` never looks at the token, so neither login opens the other's
  routes. The person always comes from the token, never from the request. The
  last-seen time is updated at most once a minute per device.
- **Limits.** One device per person that is not revoked, and active names are
  unique (not case-sensitive); to replace a phone, revoke the old device first.
  A code nobody used is removed when the next device is made. After 10 failed
  enrolments or failed tokens within 15 minutes from one client address, further
  attempts from it answer 429 with `Retry-After`, as for logins. Behind
  `tailscale serve` every request comes from the proxy's address, so the count
  is shared by all phones.
- **Storage.** Enrolled devices are written to `seed/devices.csv` on every
  start (username, name, the token's hash and the enrolment time; never the
  token) so that `make reset` keeps them. On import a device is restored only
  for a user who has no device row at all, so a phone enrolled since the last
  start is kept and a revoked one does not return. Pending enrolments are not
  saved: a code that was not used yet is made again.

## Users

Users come from `seed/users.csv` ([csv-seed.md](csv-seed.md)). The password
column holds a bcrypt hash and nothing else, so the committed file has no
password in it; a plain password is refused. A changed hash there is applied on
the next start. Every user can log in and sees all
data ([ADR 010](../decisions/010-household-data-is-shared.md)).
