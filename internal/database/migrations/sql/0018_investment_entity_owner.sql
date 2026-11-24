-- An investment account (entity) belongs to a person, or to the household when
-- user_id is NULL (ADR 021). It replaces the owner of each instrument: the
-- instruments of an account now take its owner. An account takes the owner its
-- instruments had, when they all had the same one; otherwise it starts with
-- none. investment_instruments.user_id stays in the table (SQLite cannot drop a
-- column with a foreign key) but is emptied and no longer read or written.
ALTER TABLE investment_entities
ADD COLUMN user_id INTEGER REFERENCES users(id) ON DELETE SET NULL;

UPDATE investment_entities SET user_id = (
    SELECT MIN(i.user_id) FROM investment_instruments i WHERE i.entity_id = investment_entities.id
)
WHERE NOT EXISTS (
    SELECT 1 FROM investment_instruments i
    WHERE i.entity_id = investment_entities.id AND i.user_id IS NULL
)
AND (
    SELECT COUNT(DISTINCT i.user_id) FROM investment_instruments i
    WHERE i.entity_id = investment_entities.id
) = 1;

UPDATE investment_instruments SET user_id = NULL;

-- The views take the owner from the entity.
DROP VIEW investment_entity_summary_view;
DROP VIEW investment_instrument_summary_view;

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
