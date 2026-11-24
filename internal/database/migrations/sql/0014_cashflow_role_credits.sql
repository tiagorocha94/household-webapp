-- A category can also have the role credits: money paid on credits. SQLite
-- cannot change a CHECK, so the column is made again with the wider list and
-- the values are copied over.
ALTER TABLE cashflow_categories
ADD COLUMN role_new TEXT CHECK (role_new IS NULL OR role_new IN ('savings', 'investments', 'goals', 'credits'));

UPDATE cashflow_categories SET role_new = role;

ALTER TABLE cashflow_categories DROP COLUMN role;
ALTER TABLE cashflow_categories RENAME COLUMN role_new TO role;
