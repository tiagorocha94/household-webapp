# 017 Seed data lives outside this repository

Status: accepted and built; the code later became public and the data stayed in a
private repository of its own (see the note at the end)

Context: `seed/` holds the household's real data, and it was gitignored, so it
had no history and no backup. The repository was private at the time. The sensitive parts
are the amounts, which a private repository is meant to hold, and the
credentials: `users.csv` had plain passwords, `devices.csv` has token
hashes.

Decision:
- **`seed/` is committed.** The data gets history and a backup, and every
  export that changes a CSV shows as a diff. The repository must stay private;
  if it is ever shared or made public, the data has to move out first, for
  example to a separate private repository used through `seed.dir`.
- **No password in the repository.** The `password` column of `users.csv`
  holds a bcrypt hash and nothing else. The server keeps the hash as it is and
  stores a changed one on the next start. A value that is not a valid bcrypt
  hash, a plain password included, stops the import and names the row, so a
  password can never be committed by mistake.
- **`devices.csv` is committed as it is.** It holds SHA-256 hashes of long
  random tokens, which cannot be reversed to a token.

Consequences: a password that was ever plain in the history stays readable
there; change those passwords when moving to hashes. A new user needs a hash made
first, with the command in the csv-seed doc. Whoever can read the
repository can read every amount. The format is in
[csv-seed.md](../architecture/csv-seed.md#userscsv).

Note, later: this repository is public, so it holds no real data. Whoever runs the server
keeps their data in a directory of their own, ideally a private repository, and starts the
server there or points `seed.dir` at it; `/seed/` is ignored here. The invented household in
`demo/` serves the tests and `make demo`, and is the starting point for a real set of files. The points above about
passwords and `devices.csv` still hold for your private data repository.
