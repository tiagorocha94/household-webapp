-- Which revision of a module's CSV files was last written, for the modules that
-- devices write to (ADR 021). revision goes up with every change a device
-- makes; csv_revision is the revision the files hold. While revision is above
-- csv_revision the files are older than the database, and a start-up does not
-- import them. Cashflow keeps its own copy of this in cashflow_csv_state.
CREATE TABLE IF NOT EXISTS module_csv_state (
    module TEXT PRIMARY KEY CHECK (LENGTH(TRIM(module)) > 0),
    revision INTEGER NOT NULL DEFAULT 0,
    csv_revision INTEGER NOT NULL DEFAULT 0
);
