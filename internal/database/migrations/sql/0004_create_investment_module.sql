-- Investment entities: where money is held (a bank, a broker, the state...).
CREATE TABLE IF NOT EXISTS investment_entities (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE CHECK (LENGTH(TRIM(name)) > 0),
    kind TEXT NOT NULL CHECK (kind IN ('bank', 'broker', 'investment_firm', 'state', 'other')),
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Investment instruments: a single holding at an entity. user_id is the
-- person it belongs to; NULL means it belongs to the household. Everyone can
-- see every holding either way.
CREATE TABLE IF NOT EXISTS investment_instruments (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    entity_id INTEGER NOT NULL,
    name TEXT NOT NULL CHECK (LENGTH(TRIM(name)) > 0),
    type TEXT NOT NULL CHECK (type IN ('etf', 'fund', 'cash', 'ppr', 'certificados_aforro', 'other')),
    notes TEXT,
    user_id INTEGER,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (entity_id) REFERENCES investment_entities(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL,
    UNIQUE(entity_id, name)
);

-- Contributions: money added to an instrument. Several per month are allowed.
CREATE TABLE IF NOT EXISTS investment_contributions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    instrument_id INTEGER NOT NULL,
    month TEXT NOT NULL CHECK (
        month GLOB '2[0-1][0-9][0-9]-[0-1][0-9]'
        AND CAST(SUBSTR(month, 6, 2) AS INTEGER) BETWEEN 1 AND 12
    ),
    amount_cents INTEGER NOT NULL CHECK (amount_cents > 0),
    description TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (instrument_id) REFERENCES investment_instruments(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_investment_contributions_instrument_month
ON investment_contributions (instrument_id, month);

-- Snapshots: the observed value of an instrument at a month. The full history
-- is kept; the current value is the latest snapshot by month, then id.
CREATE TABLE IF NOT EXISTS investment_snapshots (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    instrument_id INTEGER NOT NULL,
    month TEXT NOT NULL CHECK (
        month GLOB '2[0-1][0-9][0-9]-[0-1][0-9]'
        AND CAST(SUBSTR(month, 6, 2) AS INTEGER) BETWEEN 1 AND 12
    ),
    value_cents INTEGER NOT NULL CHECK (value_cents >= 0),
    description TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (instrument_id) REFERENCES investment_instruments(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_investment_snapshots_instrument_month
ON investment_snapshots (instrument_id, month);

-- Instrument summary view: one row per instrument with the money put in, the
-- latest value and the gain, all derived and never stored. gain_pct is an
-- integer scaled by 100 (734 means +7.34%) and may be negative. Only the
-- contributions are joined; every snapshot figure is a correlated subquery,
-- so the two child tables cannot multiply each other's rows.
CREATE VIEW IF NOT EXISTS investment_instrument_summary_view AS
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
LEFT JOIN users u ON u.id = i.user_id
LEFT JOIN investment_contributions c ON c.instrument_id = i.id
GROUP BY i.id;

-- Entity summary view: one row per entity, aggregating its instruments.
CREATE VIEW IF NOT EXISTS investment_entity_summary_view AS
SELECT
    e.id AS entity_id,
    e.name,
    e.kind,
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
LEFT JOIN investment_instrument_summary_view v ON v.entity_id = e.id
GROUP BY e.id;
