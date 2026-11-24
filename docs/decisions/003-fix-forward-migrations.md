# 003 Migrations are fix-forward

Status: accepted

Context: the database holds state that is not only in the CSV files: enrolled
devices, sessions and the movements that phones sent since the last export.
Rebuilding the database for every schema change loses sessions and, before
devices were saved to a file, enrolments. Editing a migration that already ran
leaves existing databases different from new ones.

Decision: a migration that has shipped is never edited. A schema change is a
new, numbered `NNNN_description.sql` file. It moves the existing values to
the new shape in the same file (for example `UPDATE` before `DROP COLUMN`), so
a database keeps its data and a new one reaches the same schema. A table that
is no longer used is dropped by a later migration, not by deleting the file
that created it. There are no down migrations. Migrations never seed data;
data comes from the CSV files.

Consequences: the history is a list of small files, and the schema is the
result of all of them, not one readable file per module (each module's
`schema.md` names the migrations that define its tables). A database is never
reset for a schema change; `make reset` stays for starting over from the CSV
files. A comment may be corrected in a shipped file, nothing else. Rules are in
[database.md](../architecture/database.md#migrations).
