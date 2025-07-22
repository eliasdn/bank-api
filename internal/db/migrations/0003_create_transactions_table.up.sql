CREATE TABLE transactions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    account_id INTEGER NOT NULL,
    amount DECIMAL(15,2) NOT NULL,
    transaction_type TEXT NOT NULL CHECK(transaction_type IN ('deposit', 'withdrawal', 'transfer')),
    reference TEXT,
    status TEXT NOT NULL CHECK(status IN ('pending', 'completed', 'failed')) DEFAULT 'pending',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (account_id) REFERENCES accounts(id) ON DELETE CASCADE
);