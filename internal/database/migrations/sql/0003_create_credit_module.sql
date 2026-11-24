-- Credit contracts: loans, visible to every logged-in user.
CREATE TABLE IF NOT EXISTS credit_contracts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE CHECK (LENGTH(TRIM(name)) > 0),
    category TEXT NOT NULL CHECK (category IN ('mortgage', 'car', 'personal', 'other')),
    contract_number TEXT NOT NULL CHECK (LENGTH(TRIM(contract_number)) > 0),
    lender TEXT NOT NULL CHECK (LENGTH(TRIM(lender)) > 0),
    original_principal_cents INTEGER NOT NULL CHECK (original_principal_cents > 0),
    start_month TEXT NOT NULL CHECK (
        start_month GLOB '2[0-1][0-9][0-9]-[0-1][0-9]'
        AND CAST(SUBSTR(start_month, 6, 2) AS INTEGER) BETWEEN 1 AND 12
    ),
    expected_end_month TEXT CHECK (
        expected_end_month IS NULL OR (
            expected_end_month GLOB '2[0-1][0-9][0-9]-[0-1][0-9]'
            AND CAST(SUBSTR(expected_end_month, 6, 2) AS INTEGER) BETWEEN 1 AND 12
        )
    ),
    notes TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(lender, contract_number),
    CHECK (expected_end_month IS NULL OR expected_end_month >= start_month)
);

-- Credit payments: actual monthly payments split into principal and interest.
-- The schema supports multiple extraordinary payments in a month, while a
-- partial unique index limits regular instalments to one per contract/month.
CREATE TABLE IF NOT EXISTS credit_payments (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    contract_id INTEGER NOT NULL,
    month TEXT NOT NULL CHECK (
        month GLOB '2[0-1][0-9][0-9]-[0-1][0-9]'
        AND CAST(SUBSTR(month, 6, 2) AS INTEGER) BETWEEN 1 AND 12
    ),
    type TEXT NOT NULL DEFAULT 'regular' CHECK (type IN ('regular', 'extraordinary')),
    principal_cents INTEGER NOT NULL CHECK (principal_cents >= 0),
    interest_cents INTEGER NOT NULL CHECK (interest_cents >= 0),
    description TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (contract_id) REFERENCES credit_contracts(id) ON DELETE CASCADE,
    CHECK (principal_cents + interest_cents > 0)
);

CREATE INDEX IF NOT EXISTS idx_credit_payments_contract_month
ON credit_payments (contract_id, month);

CREATE UNIQUE INDEX IF NOT EXISTS idx_credit_payments_regular_unique
ON credit_payments (contract_id, month)
WHERE type = 'regular';

-- Contract summary view: one row per contract with totals (in integer cents)
-- derived from the complete payment history, including the status: 'paid_off'
-- once no principal is outstanding, otherwise 'active'. Nothing stores the
-- status and sync does not maintain it.
CREATE VIEW IF NOT EXISTS credit_contract_summary_view AS
SELECT
    c.id AS contract_id,
    c.name,
    c.category,
    c.contract_number,
    c.lender,
    c.original_principal_cents,
    c.start_month,
    c.expected_end_month,
    c.notes,
    c.created_at,
    c.updated_at,
    CAST(COUNT(p.id) AS INTEGER) AS payment_count,
    CAST(COALESCE(SUM(p.principal_cents), 0) AS INTEGER) AS total_principal_paid_cents,
    CAST(COALESCE(SUM(p.interest_cents), 0) AS INTEGER) AS total_interest_paid_cents,
    CAST(COALESCE(SUM(p.principal_cents + p.interest_cents), 0) AS INTEGER) AS total_paid_cents,
    CAST(c.original_principal_cents - COALESCE(SUM(p.principal_cents), 0) AS INTEGER) AS outstanding_principal_cents,
    CAST(MIN(COALESCE(SUM(p.principal_cents), 0) * 100 / c.original_principal_cents, 100) AS INTEGER) AS progress_pct,
    MAX(p.month) AS last_payment_month,
    CASE WHEN c.original_principal_cents - COALESCE(SUM(p.principal_cents), 0) <= 0
        THEN 'paid_off' ELSE 'active'
    END AS status
FROM credit_contracts c
LEFT JOIN credit_payments p ON p.contract_id = c.id
GROUP BY c.id;
