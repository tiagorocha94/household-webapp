-- The day of the month by which a credit payment is expected to be recorded.
-- Empty means the payment is never chased.
ALTER TABLE credit_contracts ADD COLUMN expected_day INTEGER CHECK (expected_day IS NULL OR expected_day BETWEEN 1 AND 28);
