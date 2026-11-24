# 011 Enrolled devices write cashflow movements by sync ID

Status: accepted and built, on the server and in the phone app

Context: cashflow movements are typed on the phones of the household's people
(the mobile app, `household-cashflow`) and reached this server by appending
exported CSV files by hand. The goal is that a movement added on a phone,
online or offline, ends up on the server by itself, and that the server becomes
the place where the data is managed. The server runs on a personal computer
that is sometimes off; the phones must keep working then. The server is
reached over Tailscale, so only the household's devices can connect.

Decision:
- **One write path.** Enrolled devices may add, change and delete cashflow
  movements through a single endpoint, `POST /api/v1/sync/cashflow`. A device
  can only change the movements of the person it is enrolled for. Every other
  endpoint stays read-only, and the web interface still never changes data.
- **Every movement has a `sync_id`.** A UUID made by the phone, mandatory on
  every movement, unique, and the key for retries: sending the same operation
  twice changes nothing. The movements CSV gets a required sixth column
  `sync_id`. Nothing is imported without it, so old five-column files are no
  longer accepted.
- **One-way for now.** Data flows from phone to server only. Each person has one
  device, and that device is the only writer of that person's movements, so no
  conflicts can happen. Pulling data back is a later decision.
- **Devices have their own login.** The owner enrols a device with a one-time
  code from a command-line command; the device receives a long-lived random
  token, stored hashed on the server (and saved as that hash in `devices.csv`)
  and revocable per device. This is separate
  from the web login cookie and is used only by the sync endpoint. The token
  decides who the movements belong to; the phone never names its person.
- **The CSV stays the durable store for now.** After every write the server
  rewrites `cashflow_movements.csv` (atomically, behind one function so it can
  be removed). Because start-up replaces a person's rows with the file, the
  database keeps a "CSV out of date" flag: while it is set, start-up exports
  first and does not import the movements file. Moving the cashflow data to live
  only in the database (and later another engine) is a separate, later decision.

Consequences: for cashflow movements, editing the CSV file and restarting is no
longer the only way to change them, and the file is not edited by hand once
sync is in use ([001](001-csv-is-the-durable-store.md)). The API has one write
endpoint ([004](004-json-api-only-interface.md)). Everyone still sees every
movement, but only the owner's device changes a person's movements
([010](010-household-data-is-shared.md)). Enrolled devices are kept in
`devices.csv`, so a rebuilt database keeps them. The mobile app's side is in
`household-cashflow/docs/design.md`. The server must be kept running and its CSV
files backed up.
