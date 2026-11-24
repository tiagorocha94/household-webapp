-- Which cashflow category feeds the deposits of a goals account, from which
-- month on. The category is a name only, so goals and cashflow still import
-- independently (ADR 013).
CREATE TABLE IF NOT EXISTS goal_deposit_sources (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    account_id INTEGER NOT NULL REFERENCES goal_accounts(id) ON DELETE CASCADE,
    category TEXT NOT NULL UNIQUE CHECK (LENGTH(TRIM(category)) > 0),
    from_month TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
