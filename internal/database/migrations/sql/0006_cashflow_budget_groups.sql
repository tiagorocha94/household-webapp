-- Budget groups: named shares of a month's income that categories belong to
-- (docs/modules/cashflow/budgets.md). The groups come from the seed file
-- cashflow_budget_groups.csv; position keeps the order of that file. A category
-- names at most one group; removing a group leaves its categories ungrouped.
CREATE TABLE IF NOT EXISTS cashflow_budget_groups (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE CHECK (LENGTH(TRIM(name)) > 0),
    share_pct INTEGER NOT NULL CHECK (share_pct BETWEEN 1 AND 100),
    position INTEGER NOT NULL CHECK (position >= 1),
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

ALTER TABLE cashflow_categories
ADD COLUMN budget_group_id INTEGER REFERENCES cashflow_budget_groups(id) ON DELETE SET NULL;
