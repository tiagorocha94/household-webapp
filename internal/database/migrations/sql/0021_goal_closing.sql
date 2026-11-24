-- A goal is closed by hand now, and closing settles what is left in it: the
-- leftover goes back to the account's unallocated money (release), or, when the
-- goal spent more than it had, the account covers the difference (cover). The
-- old refund movement is the release. closing_balance_cents keeps the balance
-- the goal had when it was closed; it stays empty while the goal is open.
--
-- The movement type is part of a CHECK constraint, which SQLite cannot change in
-- place, so the movements table is rebuilt. The views read from it and are
-- dropped first and created again after.
DROP VIEW goal_account_balance_view;
DROP VIEW goal_account_goal_summary_view;

ALTER TABLE goal_account_goals ADD COLUMN closing_balance_cents INTEGER;

CREATE TABLE goal_account_movements_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    account_id INTEGER NOT NULL,
    goal_id INTEGER,
    type TEXT NOT NULL CHECK (type IN ('deposit', 'allocation', 'expense', 'release', 'cover')),
    amount_cents INTEGER NOT NULL CHECK (amount_cents > 0),
    month TEXT NOT NULL CHECK (
        month GLOB '2[0-1][0-9][0-9]-[0-1][0-9]'
        AND CAST(SUBSTR(month, 6, 2) AS INTEGER) BETWEEN 1 AND 12
    ),
    description TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (account_id) REFERENCES goal_accounts(id) ON DELETE CASCADE,
    FOREIGN KEY (goal_id) REFERENCES goal_account_goals(id) ON DELETE CASCADE,
    -- Deposits must not reference a goal; every other type must.
    CHECK (
        (type = 'deposit' AND goal_id IS NULL) OR
        (type != 'deposit' AND goal_id IS NOT NULL)
    )
);

INSERT INTO goal_account_movements_new (id, account_id, goal_id, type, amount_cents, month, description, created_at)
SELECT id, account_id, goal_id,
       CASE type WHEN 'refund' THEN 'release' ELSE type END,
       amount_cents, month, description, created_at
FROM goal_account_movements;

DROP TABLE goal_account_movements;
ALTER TABLE goal_account_movements_new RENAME TO goal_account_movements;

CREATE INDEX idx_goal_account_movements_account_type
ON goal_account_movements (account_id, type);

CREATE INDEX idx_goal_account_movements_goal_type
ON goal_account_movements (goal_id, type);

CREATE UNIQUE INDEX idx_goal_account_movements_allocation_unique
ON goal_account_movements (account_id, goal_id, type, month)
WHERE type = 'allocation';

-- unallocated = deposits - allocated + released - covered (free to assign to goals).
CREATE VIEW goal_account_balance_view AS
SELECT
    a.id AS account_id,
    a.name AS account_name,
    COALESCE(SUM(CASE WHEN m.type = 'deposit' THEN m.amount_cents ELSE 0 END), 0) AS total_deposits_cents,
    COALESCE(SUM(CASE WHEN m.type = 'allocation' THEN m.amount_cents ELSE 0 END), 0) AS total_allocated_cents,
    COALESCE(SUM(CASE WHEN m.type = 'expense' THEN m.amount_cents ELSE 0 END), 0) AS total_expenses_cents,
    COALESCE(SUM(CASE WHEN m.type = 'release' THEN m.amount_cents ELSE 0 END), 0) AS total_released_cents,
    COALESCE(SUM(CASE WHEN m.type = 'cover' THEN m.amount_cents ELSE 0 END), 0) AS total_covered_cents,
    COALESCE(SUM(CASE WHEN m.type = 'deposit' THEN m.amount_cents ELSE 0 END), 0)
        - COALESCE(SUM(CASE WHEN m.type = 'expense' THEN m.amount_cents ELSE 0 END), 0) AS balance_cents,
    COALESCE(SUM(CASE WHEN m.type = 'deposit' THEN m.amount_cents ELSE 0 END), 0)
        - COALESCE(SUM(CASE WHEN m.type = 'allocation' THEN m.amount_cents ELSE 0 END), 0)
        + COALESCE(SUM(CASE WHEN m.type = 'release' THEN m.amount_cents ELSE 0 END), 0)
        - COALESCE(SUM(CASE WHEN m.type = 'cover' THEN m.amount_cents ELSE 0 END), 0) AS unallocated_cents
FROM goal_accounts a
LEFT JOIN goal_account_movements m ON a.id = m.account_id
GROUP BY a.id;

-- goal_balance = allocated - expenses - released + covered (money in the goal;
-- zero once a closing movement has settled it, and negative while it is overspent).
CREATE VIEW goal_account_goal_summary_view AS
SELECT
    g.id AS goal_id,
    g.account_id,
    g.name,
    g.target_cents,
    g.start_month,
    g.end_month,
    g.status,
    g.closing_balance_cents,
    g.created_at,
    g.updated_at,
    MonthCount.months AS total_months,
    CASE WHEN MonthCount.months > 0
        THEN (g.target_cents + MonthCount.months - 1) / MonthCount.months
        ELSE 0
    END AS monthly_allocation_cents,
    COALESCE(SUM(CASE WHEN m.type = 'allocation' THEN m.amount_cents ELSE 0 END), 0) AS total_allocated_cents,
    COALESCE(SUM(CASE WHEN m.type = 'expense' THEN m.amount_cents ELSE 0 END), 0) AS total_expenses_cents,
    COALESCE(SUM(CASE WHEN m.type = 'release' THEN m.amount_cents ELSE 0 END), 0) AS total_released_cents,
    COALESCE(SUM(CASE WHEN m.type = 'cover' THEN m.amount_cents ELSE 0 END), 0) AS total_covered_cents,
    COALESCE(SUM(CASE WHEN m.type = 'allocation' THEN m.amount_cents ELSE 0 END), 0)
        - COALESCE(SUM(CASE WHEN m.type = 'expense' THEN m.amount_cents ELSE 0 END), 0)
        - COALESCE(SUM(CASE WHEN m.type = 'release' THEN m.amount_cents ELSE 0 END), 0)
        + COALESCE(SUM(CASE WHEN m.type = 'cover' THEN m.amount_cents ELSE 0 END), 0) AS goal_balance_cents,
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
