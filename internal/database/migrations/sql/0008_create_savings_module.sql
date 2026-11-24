-- Savings accounts: money kept aside and within reach, such as an emergency
-- fund. user_id is the owner; NULL means the household. target_cents is an
-- optional amount to reach; there are no dates and no monthly plan.
CREATE TABLE IF NOT EXISTS savings_accounts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE CHECK (LENGTH(TRIM(name)) > 0),
    user_id INTEGER REFERENCES users(id) ON DELETE SET NULL,
    target_cents INTEGER CHECK (target_cents IS NULL OR target_cents > 0),
    notes TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Movements: money put in (deposit) or taken out (withdrawal) in a month.
-- emergency marks a withdrawal made for an emergency. Several per month are
-- allowed. Putting money back is a deposit.
CREATE TABLE IF NOT EXISTS savings_movements (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    account_id INTEGER NOT NULL REFERENCES savings_accounts(id) ON DELETE CASCADE,
    month TEXT NOT NULL CHECK (
        month GLOB '2[0-1][0-9][0-9]-[0-1][0-9]'
        AND CAST(SUBSTR(month, 6, 2) AS INTEGER) BETWEEN 1 AND 12
    ),
    type TEXT NOT NULL CHECK (type IN ('deposit', 'withdrawal')),
    amount_cents INTEGER NOT NULL CHECK (amount_cents > 0),
    emergency INTEGER NOT NULL DEFAULT 0 CHECK (emergency IN (0, 1) AND (emergency = 0 OR type = 'withdrawal')),
    description TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_savings_movements_account_month
ON savings_movements (account_id, month);

-- Snapshots: the balance seen at a month. The full history is kept; the
-- balance is the latest snapshot by month, then id.
CREATE TABLE IF NOT EXISTS savings_snapshots (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    account_id INTEGER NOT NULL REFERENCES savings_accounts(id) ON DELETE CASCADE,
    month TEXT NOT NULL CHECK (
        month GLOB '2[0-1][0-9][0-9]-[0-1][0-9]'
        AND CAST(SUBSTR(month, 6, 2) AS INTEGER) BETWEEN 1 AND 12
    ),
    value_cents INTEGER NOT NULL CHECK (value_cents >= 0),
    description TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_savings_snapshots_account_month
ON savings_snapshots (account_id, month);
