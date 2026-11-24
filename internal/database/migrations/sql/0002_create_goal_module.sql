-- Goal accounts: the top-level entity for the goals module. Every logged-in
-- user sees every account. monthly_deposit_cents is the deposit the projection
-- assumes for each future month.
CREATE TABLE IF NOT EXISTS goal_accounts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE,
    monthly_deposit_cents INTEGER NOT NULL CHECK (monthly_deposit_cents >= 0),
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Goals within an account.
CREATE TABLE IF NOT EXISTS goal_account_goals (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    account_id INTEGER NOT NULL,
    name TEXT NOT NULL,
    target_cents INTEGER NOT NULL CHECK (target_cents > 0),
    start_month TEXT NOT NULL CHECK (
        start_month GLOB '2[0-1][0-9][0-9]-[0-1][0-9]'
        AND CAST(SUBSTR(start_month, 6, 2) AS INTEGER) BETWEEN 1 AND 12
    ),
    end_month TEXT NOT NULL CHECK (
        end_month GLOB '2[0-1][0-9][0-9]-[0-1][0-9]'
        AND CAST(SUBSTR(end_month, 6, 2) AS INTEGER) BETWEEN 1 AND 12
    ),
    status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'completed', 'cancelled')),
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (account_id) REFERENCES goal_accounts(id) ON DELETE CASCADE,
    UNIQUE(account_id, name),
    CHECK (end_month >= start_month)
);

CREATE INDEX IF NOT EXISTS idx_goal_account_goals_status
ON goal_account_goals (account_id, status);

-- Unified ledger for all money movements.
CREATE TABLE IF NOT EXISTS goal_account_movements (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    account_id INTEGER NOT NULL,
    goal_id INTEGER,
    type TEXT NOT NULL CHECK (type IN ('deposit', 'allocation', 'expense', 'refund')),
    amount_cents INTEGER NOT NULL CHECK (amount_cents > 0),
    month TEXT NOT NULL CHECK (
        month GLOB '2[0-1][0-9][0-9]-[0-1][0-9]'
        AND CAST(SUBSTR(month, 6, 2) AS INTEGER) BETWEEN 1 AND 12
    ),
    description TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (account_id) REFERENCES goal_accounts(id) ON DELETE CASCADE,
    FOREIGN KEY (goal_id) REFERENCES goal_account_goals(id) ON DELETE CASCADE,
    -- Deposits must not reference a goal; allocation/expense/refund must.
    CHECK (
        (type = 'deposit' AND goal_id IS NULL) OR
        (type != 'deposit' AND goal_id IS NOT NULL)
    )
);

CREATE INDEX IF NOT EXISTS idx_goal_account_movements_account_type
ON goal_account_movements (account_id, type);

CREATE INDEX IF NOT EXISTS idx_goal_account_movements_goal_type
ON goal_account_movements (goal_id, type);

-- Only one allocation per goal per month.
CREATE UNIQUE INDEX IF NOT EXISTS idx_goal_account_movements_allocation_unique
ON goal_account_movements (account_id, goal_id, type, month)
WHERE type = 'allocation';


-- Account balance view: one row per account, all money in integer cents.
-- balance = deposits - expenses (real money in the account).
-- unallocated = deposits - allocated + refunded (free to assign to goals).
CREATE VIEW IF NOT EXISTS goal_account_balance_view AS
SELECT
    a.id AS account_id,
    a.name AS account_name,
    COALESCE(SUM(CASE WHEN m.type = 'deposit' THEN m.amount_cents ELSE 0 END), 0) AS total_deposits_cents,
    COALESCE(SUM(CASE WHEN m.type = 'allocation' THEN m.amount_cents ELSE 0 END), 0) AS total_allocated_cents,
    COALESCE(SUM(CASE WHEN m.type = 'expense' THEN m.amount_cents ELSE 0 END), 0) AS total_expenses_cents,
    COALESCE(SUM(CASE WHEN m.type = 'refund' THEN m.amount_cents ELSE 0 END), 0) AS total_refunded_cents,
    COALESCE(SUM(CASE WHEN m.type = 'deposit' THEN m.amount_cents ELSE 0 END), 0)
        - COALESCE(SUM(CASE WHEN m.type = 'expense' THEN m.amount_cents ELSE 0 END), 0) AS balance_cents,
    COALESCE(SUM(CASE WHEN m.type = 'deposit' THEN m.amount_cents ELSE 0 END), 0)
        - COALESCE(SUM(CASE WHEN m.type = 'allocation' THEN m.amount_cents ELSE 0 END), 0)
        + COALESCE(SUM(CASE WHEN m.type = 'refund' THEN m.amount_cents ELSE 0 END), 0) AS unallocated_cents
FROM goal_accounts a
LEFT JOIN goal_account_movements m ON a.id = m.account_id
GROUP BY a.id;

-- Goal summary view: one row per goal, all money in integer cents.
-- monthly_allocation_cents = ceil(target / months), the largest monthly
-- amount of the cent-exact split (see internal/money.Split).
-- goal_balance = allocated - expenses - refunded (money saved in this goal).
-- progress_pct = allocated / target * 100 (integer division, capped at 100).
-- Elapsed months are not here: they depend on the current month, which the
-- goals service computes in Go (internal/month) in the local time zone.
CREATE VIEW IF NOT EXISTS goal_account_goal_summary_view AS
SELECT
    g.id AS goal_id,
    g.account_id,
    g.name,
    g.target_cents,
    g.start_month,
    g.end_month,
    g.status,
    g.created_at,
    g.updated_at,
    MonthCount.months AS total_months,
    CASE WHEN MonthCount.months > 0
        THEN (g.target_cents + MonthCount.months - 1) / MonthCount.months
        ELSE 0
    END AS monthly_allocation_cents,
    COALESCE(SUM(CASE WHEN m.type = 'allocation' THEN m.amount_cents ELSE 0 END), 0) AS total_allocated_cents,
    COALESCE(SUM(CASE WHEN m.type = 'expense' THEN m.amount_cents ELSE 0 END), 0) AS total_expenses_cents,
    COALESCE(SUM(CASE WHEN m.type = 'refund' THEN m.amount_cents ELSE 0 END), 0) AS total_refunded_cents,
    COALESCE(SUM(CASE WHEN m.type = 'allocation' THEN m.amount_cents ELSE 0 END), 0)
        - COALESCE(SUM(CASE WHEN m.type = 'expense' THEN m.amount_cents ELSE 0 END), 0)
        - COALESCE(SUM(CASE WHEN m.type = 'refund' THEN m.amount_cents ELSE 0 END), 0) AS goal_balance_cents,
    CASE WHEN g.target_cents > 0
        THEN MIN(COALESCE(SUM(CASE WHEN m.type = 'allocation' THEN m.amount_cents ELSE 0 END), 0) * 100 / g.target_cents, 100)
        ELSE 0
    END AS progress_pct
FROM goal_account_goals g
LEFT JOIN goal_account_movements m ON g.id = m.goal_id
LEFT JOIN (
    SELECT id,
        (CAST(SUBSTR(end_month, 1, 4) AS INTEGER) - CAST(SUBSTR(start_month, 1, 4) AS INTEGER)) * 12
        + (CAST(SUBSTR(end_month, 6, 2) AS INTEGER) - CAST(SUBSTR(start_month, 6, 2) AS INTEGER))
        + 1 AS months
    FROM goal_account_goals
) MonthCount ON g.id = MonthCount.id
GROUP BY g.id;
