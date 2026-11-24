-- Cash is not an investment: money held as cash is a savings account (the savings
-- module came after the instrument type), and net worth counts savings once. The
-- type is removed from the CHECK constraint, which SQLite cannot change in place,
-- so the table is rebuilt the way migration 0020 did it.
--
-- Migrations run before the CSV import, so an instrument of the removed type that
-- nothing refers to (no contribution, no snapshot) is deleted here; the CSV
-- no longer lists it. One that still has contributions or snapshots is kept and
-- makes the rebuild fail on the CHECK, so no history is ever lost silently: move
-- that money to a savings account first.
DROP VIEW investment_entity_summary_view;
DROP VIEW investment_instrument_summary_view;

DELETE FROM investment_instruments
WHERE type = 'cash'
  AND id NOT IN (SELECT instrument_id FROM investment_contributions)
  AND id NOT IN (SELECT instrument_id FROM investment_snapshots);

CREATE TEMP TABLE kept_contributions AS SELECT * FROM investment_contributions;
CREATE TEMP TABLE kept_snapshots AS SELECT * FROM investment_snapshots;
CREATE TEMP TABLE kept_instruments AS SELECT * FROM investment_instruments;

DROP TABLE investment_instruments;

CREATE TABLE investment_instruments (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    entity_id INTEGER NOT NULL,
    name TEXT NOT NULL CHECK (LENGTH(TRIM(name)) > 0),
    type TEXT NOT NULL CHECK (type IN ('etf', 'fund', 'ppr', 'certificados_aforro', 'other')),
    notes TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (entity_id) REFERENCES investment_entities(id) ON DELETE CASCADE,
    UNIQUE(entity_id, name)
);

INSERT INTO investment_instruments (id, entity_id, name, type, notes, created_at, updated_at)
SELECT id, entity_id, name, type, notes, created_at, updated_at FROM kept_instruments;
INSERT INTO investment_contributions SELECT * FROM kept_contributions;
INSERT INTO investment_snapshots SELECT * FROM kept_snapshots;

DROP TABLE kept_contributions;
DROP TABLE kept_snapshots;
DROP TABLE kept_instruments;

-- The views are the ones of migration 0020.

CREATE VIEW investment_instrument_summary_view AS
SELECT
    i.id AS instrument_id,
    i.entity_id,
    e.name AS entity_name,
    i.name,
    i.type,
    i.notes,
    u.username,
    i.created_at,
    i.updated_at,
    CAST(COUNT(c.id) AS INTEGER) AS contribution_count,
    CAST((SELECT COUNT(*) FROM investment_snapshots s WHERE s.instrument_id = i.id) AS INTEGER) AS snapshot_count,
    CAST(COALESCE(SUM(c.amount_cents), 0) AS INTEGER) AS allocated_cents,
    CAST(COALESCE((
        SELECT s.value_cents FROM investment_snapshots s
        WHERE s.instrument_id = i.id
        ORDER BY s.month DESC, s.id DESC LIMIT 1
    ), 0) AS INTEGER) AS current_value_cents,
    (SELECT MAX(s.month) FROM investment_snapshots s WHERE s.instrument_id = i.id) AS last_snapshot_month,
    CAST(COALESCE((
        SELECT s.value_cents FROM investment_snapshots s
        WHERE s.instrument_id = i.id
        ORDER BY s.month DESC, s.id DESC LIMIT 1
    ), 0) - COALESCE(SUM(c.amount_cents), 0) AS INTEGER) AS gain_cents,
    CAST(CASE WHEN COALESCE(SUM(c.amount_cents), 0) > 0 THEN
        (COALESCE((
            SELECT s.value_cents FROM investment_snapshots s
            WHERE s.instrument_id = i.id
            ORDER BY s.month DESC, s.id DESC LIMIT 1
        ), 0) - SUM(c.amount_cents)) * 10000 / SUM(c.amount_cents)
    ELSE 0 END AS INTEGER) AS gain_pct
FROM investment_instruments i
JOIN investment_entities e ON e.id = i.entity_id
LEFT JOIN users u ON u.id = e.user_id
LEFT JOIN investment_contributions c ON c.instrument_id = i.id
GROUP BY i.id;

CREATE VIEW investment_entity_summary_view AS
SELECT
    e.id AS entity_id,
    e.name,
    e.kind,
    u.username,
    e.created_at,
    e.updated_at,
    CAST(COUNT(v.instrument_id) AS INTEGER) AS instrument_count,
    CAST(COALESCE(SUM(v.allocated_cents), 0) AS INTEGER) AS allocated_cents,
    CAST(COALESCE(SUM(v.current_value_cents), 0) AS INTEGER) AS current_value_cents,
    CAST(COALESCE(SUM(v.current_value_cents), 0) - COALESCE(SUM(v.allocated_cents), 0) AS INTEGER) AS gain_cents,
    CAST(CASE WHEN COALESCE(SUM(v.allocated_cents), 0) > 0 THEN
        (SUM(v.current_value_cents) - SUM(v.allocated_cents)) * 10000 / SUM(v.allocated_cents)
    ELSE 0 END AS INTEGER) AS gain_pct
FROM investment_entities e
LEFT JOIN users u ON u.id = e.user_id
LEFT JOIN investment_instrument_summary_view v ON v.entity_id = e.id
GROUP BY e.id;
