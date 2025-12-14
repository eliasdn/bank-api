-- Add composite indexes to optimize common queries
-- These indexes target the "get transactions for account" and "get accounts for user" queries
-- which filter by foreign key and sort by created_at DESC.

CREATE INDEX idx_transactions_account_created ON transactions(account_id, created_at DESC);
CREATE INDEX idx_accounts_user_created ON accounts(user_id, created_at DESC);
