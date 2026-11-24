-- What a person's bank account held at the end of a month, typed in. It is the
-- starting point for the cash balance of every other month, which is worked out
-- from the cashflow of the months in between. The balance can be negative.
CREATE TABLE IF NOT EXISTS cashflow_balances (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    month TEXT NOT NULL CHECK (
        month GLOB '2[0-1][0-9][0-9]-[0-1][0-9]'
        AND CAST(SUBSTR(month, 6, 2) AS INTEGER) BETWEEN 1 AND 12
    ),
    balance_cents INTEGER NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    UNIQUE (user_id, month)
);
