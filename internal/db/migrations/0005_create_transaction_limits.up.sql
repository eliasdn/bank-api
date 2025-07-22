-- Create transaction_limits table
CREATE TABLE IF NOT EXISTS transaction_limits (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    account_id INTEGER NOT NULL,
    daily_limit REAL NOT NULL DEFAULT 10000,
    monthly_limit REAL NOT NULL DEFAULT 100000,
    min_amount REAL NOT NULL DEFAULT 0.01,
    max_amount REAL NOT NULL DEFAULT 50000,
    daily_used REAL NOT NULL DEFAULT 0,
    monthly_used REAL NOT NULL DEFAULT 0,
    last_reset_date DATETIME NOT NULL,
    transfer_limit REAL NOT NULL DEFAULT 5000,
    withdrawal_limit REAL NOT NULL DEFAULT 5000,
    deposit_limit REAL NOT NULL DEFAULT 10000,
    transfer_used REAL NOT NULL DEFAULT 0,
    withdrawal_used REAL NOT NULL DEFAULT 0,
    deposit_used REAL NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (account_id) REFERENCES accounts(id) ON DELETE CASCADE,
    UNIQUE(account_id)
);

-- Create transaction_usage table
CREATE TABLE IF NOT EXISTS transaction_usage (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    account_id INTEGER NOT NULL,
    transaction_id INTEGER NOT NULL,
    amount REAL NOT NULL,
    type TEXT NOT NULL CHECK (type IN ('deposit', 'withdrawal', 'transfer')),
    date DATETIME NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (account_id) REFERENCES accounts(id) ON DELETE CASCADE,
    FOREIGN KEY (transaction_id) REFERENCES transactions(id) ON DELETE CASCADE,
    UNIQUE(transaction_id)
);

-- Create indexes for performance
CREATE INDEX IF NOT EXISTS idx_transaction_limits_account_id ON transaction_limits(account_id);
CREATE INDEX IF NOT EXISTS idx_transaction_usage_account_id ON transaction_usage(account_id);
CREATE INDEX IF NOT EXISTS idx_transaction_usage_date ON transaction_usage(date);
CREATE INDEX IF NOT EXISTS idx_transaction_usage_type ON transaction_usage(type);