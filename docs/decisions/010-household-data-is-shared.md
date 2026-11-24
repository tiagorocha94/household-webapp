# 010 Household data is shared by every user

Status: accepted

Context: the app is for one household. Ownership only matters to decide who may
change something, and the web interface changes nothing.

Decision: every user who can log in sees all the household's data. Accounts,
goals, contracts and the like have no owner and no members, and the services
take no acting user. Authentication still protects every route; it only
decides whether you are in. `users.csv` holds the people who can log in and
nothing else. The one thing with an owner is a cashflow movement: it belongs to
the person whose phone sent it ([011](011-devices-write-cashflow-movements.md)),
and only that phone changes it.

Consequences: less schema, less code and simpler CSV files. If writes from the
web interface are ever added, decide then who may change what. Authentication
is in [authentication.md](../architecture/authentication.md).
