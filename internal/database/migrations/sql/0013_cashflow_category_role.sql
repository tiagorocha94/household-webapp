-- One role replaces the savings and investments flags: an expense category says
-- where its money goes (savings, investments or goals), or has no role.
ALTER TABLE cashflow_categories
ADD COLUMN role TEXT CHECK (role IS NULL OR role IN ('savings', 'investments', 'goals'));

UPDATE cashflow_categories SET role = 'savings' WHERE is_savings = 1;
UPDATE cashflow_categories SET role = 'investments' WHERE is_investment = 1;

ALTER TABLE cashflow_categories DROP COLUMN is_savings;
ALTER TABLE cashflow_categories DROP COLUMN is_investment;
