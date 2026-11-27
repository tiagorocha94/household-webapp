# Database backups

The CSV files are the durable store today ([ADR 001](../decisions/001-csv-is-the-durable-store.md)),
and the server keeps them in step with what phones write. A database backup is a
second copy that does not depend on them, and the way to keep your data once the
CSV files are gone. It is off unless you turn it on.

## Configuration

```yaml
backup:
  dir: backups   # where the files go
  keep: 14       # how many daily backups to keep
```

| Key | Meaning |
|-----|---------|
| `keep` | How many daily backups to keep. Zero or left out means no backups, whatever `dir` says. A negative number is an error. |
| `dir` | Required when `keep` is above zero. Resolved like `seed.dir` and `database.path`: relative to where the server is started, so it ends up next to your data when you start the server from your own data directory. It must not be the seed directory. The server creates it, readable only by its owner. |

Keep the directory outside the seed directory, and out of version control
(`/backups/` is ignored in this repository): a backup holds password hashes,
device token hashes and all your money data.

## What it does

- Copies the live SQLite database with `VACUUM INTO`, which gives a consistent
  snapshot while the server keeps serving. It is the real database, so it holds
  everything phones have written, not only what the CSV files say.
- Runs once when the server starts (after start-up has synced), and then once a
  day, on the first minute after the local date changes. It is the last of the
  [background jobs](overview.md#background-jobs), so it holds what the goal
  allocations job just created.
- Writes one file a day, `household-YYYY-MM-DD.db`. A later run on the same day
  replaces it, so restarting the server never piles up files.
- Writes to a temporary file and renames it, so a failed run never leaves a
  partial backup, and the file is readable only by its owner (`0600`).
- Removes the oldest `household-YYYY-MM-DD.db` files beyond `keep`. Files with
  any other name are never touched.
- A failure is logged (`job failed`, with the reason) and the next run tries
  again. It never stops the server.

A backup is an ordinary SQLite file: open it with any SQLite tool to look at it.

## Restoring

1. Stop the server.
2. Copy the backup over `database.path`, and delete the `-wal` and `-shm` files
   next to it if any.
3. Move the CSV files of the data you want from the backup out of the seed
   directory (or the whole directory). At start-up a CSV file that exists is
   authoritative, and the server would mirror it over the restored database; a
   file that is absent is skipped, and the export that follows rewrites it from
   the database.
4. Start the server.

If the CSV files are intact and current, you do not need a backup: `make reset`
rebuilds the database from them.
