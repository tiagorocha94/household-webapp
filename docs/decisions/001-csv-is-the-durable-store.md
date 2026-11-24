# 001 CSV is the durable store

Status: accepted

Context: the application is personal and its data is small, human-edited and
worth keeping under version control. Letting the database be authoritative
after a first seed made it unclear whether editing a file still had any effect.

Decision: CSV files are the durable store between runs. Every start migrates
the schema, mirrors the files into the database, runs sync and exports the
result back to the files. Files that exist are authoritative; any failure
aborts startup, and export only runs after a successful import.

Two things are written at run time and therefore have their own rules:
- **Cashflow movements** that enrolled phones send ([011](011-devices-write-cashflow-movements.md)).
  The server rewrites `cashflow_movements.csv` after every write, and while the
  file is behind the database, start-up exports first and does not import it.
- **Enrolled devices.** `devices.csv` is written on every start (the hash of
  each token, never the token) and only restored for a user who has no device
  row, so a rebuilt database keeps the enrolled phones.

[ADR 021](021-details-of-money-moved.md) adds the savings, investment and credit
records the phone writes: the database takes the write and the file is
rewritten after it.

Consequences: editing a file and restarting is how data changes, and removing
a row from a file removes it from the database. A malformed file cannot
overwrite good ones. Because the database can be rebuilt from the files
(`make reset`), back up the seed directory; it is under version control in a separate private
repository ([ADR 017](017-seed-data-in-a-private-repo.md)), with bcrypt hashes
in `users.csv` and token hashes in `devices.csv`. The exact rules are in [csv-seed.md](../architecture/csv-seed.md).
