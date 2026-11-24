-- A savings movement, an investment contribution or a credit payment can be the
-- detail of a cashflow movement: where the money went (ADR 021). movement_sync_id
-- is that movement's sync_id. It is not a foreign key on purpose: the movements
-- file is mirrored into the database by replacing rows, and that must never
-- remove a detail. Rows added by hand have none.
ALTER TABLE savings_movements
ADD COLUMN movement_sync_id TEXT CHECK (movement_sync_id IS NULL OR LENGTH(movement_sync_id) = 36);
ALTER TABLE investment_contributions
ADD COLUMN movement_sync_id TEXT CHECK (movement_sync_id IS NULL OR LENGTH(movement_sync_id) = 36);
ALTER TABLE credit_payments
ADD COLUMN movement_sync_id TEXT CHECK (movement_sync_id IS NULL OR LENGTH(movement_sync_id) = 36);

CREATE INDEX IF NOT EXISTS idx_savings_movements_movement
ON savings_movements (movement_sync_id) WHERE movement_sync_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_investment_contributions_movement
ON investment_contributions (movement_sync_id) WHERE movement_sync_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_credit_payments_movement
ON credit_payments (movement_sync_id) WHERE movement_sync_id IS NOT NULL;
