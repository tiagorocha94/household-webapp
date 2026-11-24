# 012 Enrolled devices read the goals endpoints

Status: accepted and built, on the server and in the phone app

Context: the phone app (`household-cashflow`) wants to show the household's
goals: totals, progress and a projection, and later a home-screen widget. The
goals endpoints need the session cookie, and the phone only has a device token
([011](011-devices-write-cashflow-movements.md)), which the server accepts on
the sync endpoint and nowhere else. The phone cannot log in: it has no
password, and a password on a phone is a bigger step than reading.

Decision:
- **The six goals routes take either credential.** `GET /goals/accounts`,
  `/goals/accounts/{id}`, `/goals/accounts/{id}/goals`,
  `/goals/accounts/{id}/movements`, `/goals/accounts/{id}/projection` and
  `/goals/{id}` accept the session cookie or `Authorization: Bearer <device
  token>`. A request that carries a bearer token is checked as a device and
  counts toward the same failure limits, whatever cookies it has.
- **Further access is deliberate.** Later decisions opened the budget
  ([015](015-devices-read-budget.md)), the owner's balance
  ([019](019-devices-read-their-own-balance.md)) and the savings, investment
  and credit reads ([022](022-devices-read-savings-investments-credits.md)).
  Cashflow movements and all other routes remain closed to device reads.
- **Still read-only, still shared.** No new data and no new endpoint. Every
  logged-in person already reads every account ([010](010-household-data-is-shared.md)),
  so a device reads exactly what its person could read in the browser.

Consequences: a stolen device token can now read the goals as well as write
that person's cashflow movements. It can be revoked per device
(`device revoke`), as before. If the phone later needs credits, investments or
cashflow, each area is added to the same helper (`RequireSessionOrDevice`)
deliberately, one at a time.
