-- An expense category can also be marked as investments: money moved to an
-- investment rather than spent. Like the savings flag, its movements stay
-- expenses; Home compares them with the contributions typed in the investments
-- module and warns about the difference.
ALTER TABLE cashflow_categories
ADD COLUMN is_investment INTEGER NOT NULL DEFAULT 0 CHECK (is_investment IN (0, 1));
