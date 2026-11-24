-- The emergency flag on savings withdrawals fed one figure nothing else used.
-- The withdrawals stay; only the flag goes.
ALTER TABLE savings_movements DROP COLUMN emergency;
