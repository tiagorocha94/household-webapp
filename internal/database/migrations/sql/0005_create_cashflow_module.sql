-- Cashflow categories: the fixed set of categories a movement can have. Each
-- one is either income or expense; the direction decides which side of the
-- month's totals a movement counts on. icon is a Lucide icon name in kebab-case
-- (for example shopping-basket) and color a #RRGGBB colour; the interface uses
-- them to draw the category, and the mobile app draws it the same way.
CREATE TABLE IF NOT EXISTS cashflow_categories (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE CHECK (LENGTH(TRIM(name)) > 0),
    direction TEXT NOT NULL CHECK (direction IN ('income', 'expense')),
    icon TEXT NOT NULL CHECK (LENGTH(icon) > 0 AND icon NOT GLOB '*[^a-z0-9-]*'),
    color TEXT NOT NULL CHECK (
        color GLOB '#[0-9A-Fa-f][0-9A-Fa-f][0-9A-Fa-f][0-9A-Fa-f][0-9A-Fa-f][0-9A-Fa-f]'
    ),
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Cashflow movements: one row per movement of money, owned by a user. Amounts
-- are always positive; the category direction says whether it came in or went
-- out. Rows may repeat exactly (several identical movements in a day are
-- valid), so only sync_id is unique: a UUID made by the device that recorded the
-- movement. It names the movement for ever, so a retried or edited send finds
-- the same row. date is a full YYYY-MM-DD day; the month is derived with
-- SUBSTR(date, 1, 7) and never stored twice. The check is coarse (it does not
-- know month lengths); the CSV import validates real calendar dates.
CREATE TABLE IF NOT EXISTS cashflow_movements (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    sync_id TEXT NOT NULL UNIQUE CHECK (LENGTH(sync_id) = 36),
    user_id INTEGER NOT NULL,
    category_id INTEGER NOT NULL,
    date TEXT NOT NULL CHECK (
        date GLOB '2[0-1][0-9][0-9]-[0-1][0-9]-[0-3][0-9]'
        AND CAST(SUBSTR(date, 6, 2) AS INTEGER) BETWEEN 1 AND 12
        AND CAST(SUBSTR(date, 9, 2) AS INTEGER) BETWEEN 1 AND 31
    ),
    amount_cents INTEGER NOT NULL CHECK (amount_cents > 0),
    description TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (category_id) REFERENCES cashflow_categories(id) ON DELETE CASCADE
);

-- CSV state: one row, kept so a start never replaces the movements in the
-- database with an older movements file. revision counts the changes made
-- through the API; csv_revision is the revision the movements file last held.
-- While they differ, the file is behind the database. See
-- docs/modules/cashflow/csv.md. Rows are created on first use.
CREATE TABLE IF NOT EXISTS cashflow_csv_state (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    revision INTEGER NOT NULL DEFAULT 0,
    csv_revision INTEGER NOT NULL DEFAULT 0,
    CHECK (csv_revision <= revision)
);

CREATE INDEX IF NOT EXISTS idx_cashflow_movements_date
ON cashflow_movements (date);

CREATE INDEX IF NOT EXISTS idx_cashflow_movements_user_date
ON cashflow_movements (user_id, date);

-- Month summary view: one row per month that has movements, with income,
-- expenses and what is left, all derived and never stored. savings_rate_pct is
-- an integer scaled by 100 (2500 means 25.00%), may be negative, and is 0 when
-- income is 0.
CREATE VIEW IF NOT EXISTS cashflow_month_summary_view AS
SELECT
    SUBSTR(m.date, 1, 7) AS month,
    CAST(COALESCE(SUM(CASE WHEN c.direction = 'income' THEN m.amount_cents ELSE 0 END), 0) AS INTEGER) AS income_cents,
    CAST(COALESCE(SUM(CASE WHEN c.direction = 'expense' THEN m.amount_cents ELSE 0 END), 0) AS INTEGER) AS expense_cents,
    CAST(COALESCE(SUM(CASE WHEN c.direction = 'income' THEN m.amount_cents ELSE -m.amount_cents END), 0) AS INTEGER) AS net_cents,
    CAST(CASE WHEN SUM(CASE WHEN c.direction = 'income' THEN m.amount_cents ELSE 0 END) > 0 THEN
        SUM(CASE WHEN c.direction = 'income' THEN m.amount_cents ELSE -m.amount_cents END) * 10000
        / SUM(CASE WHEN c.direction = 'income' THEN m.amount_cents ELSE 0 END)
    ELSE 0 END AS INTEGER) AS savings_rate_pct,
    CAST(COUNT(m.id) AS INTEGER) AS movement_count
FROM cashflow_movements m
JOIN cashflow_categories c ON c.id = m.category_id
GROUP BY SUBSTR(m.date, 1, 7);

-- Month category view: the total per category in a month.
CREATE VIEW IF NOT EXISTS cashflow_month_category_view AS
SELECT
    SUBSTR(m.date, 1, 7) AS month,
    c.id AS category_id,
    c.name AS category_name,
    c.direction,
    CAST(SUM(m.amount_cents) AS INTEGER) AS total_cents,
    CAST(COUNT(m.id) AS INTEGER) AS movement_count
FROM cashflow_movements m
JOIN cashflow_categories c ON c.id = m.category_id
GROUP BY SUBSTR(m.date, 1, 7), c.id;

-- Month user view: income, expenses and what is left per person in a month.
CREATE VIEW IF NOT EXISTS cashflow_month_user_view AS
SELECT
    SUBSTR(m.date, 1, 7) AS month,
    u.id AS user_id,
    u.username,
    CAST(COALESCE(SUM(CASE WHEN c.direction = 'income' THEN m.amount_cents ELSE 0 END), 0) AS INTEGER) AS income_cents,
    CAST(COALESCE(SUM(CASE WHEN c.direction = 'expense' THEN m.amount_cents ELSE 0 END), 0) AS INTEGER) AS expense_cents,
    CAST(COALESCE(SUM(CASE WHEN c.direction = 'income' THEN m.amount_cents ELSE -m.amount_cents END), 0) AS INTEGER) AS net_cents
FROM cashflow_movements m
JOIN cashflow_categories c ON c.id = m.category_id
JOIN users u ON u.id = m.user_id
GROUP BY SUBSTR(m.date, 1, 7), u.id;
