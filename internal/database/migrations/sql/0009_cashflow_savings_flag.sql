-- An expense category can be marked as savings: money moved to savings rather
-- than spent. Its movements stay expenses; views add them up as "of which
-- saved" (ADR 014).
ALTER TABLE cashflow_categories
ADD COLUMN is_savings INTEGER NOT NULL DEFAULT 0 CHECK (is_savings IN (0, 1));
