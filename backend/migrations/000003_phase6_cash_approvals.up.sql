-- Phase 6: opening balances, cash reconciliation, transaction approvals

CREATE TABLE IF NOT EXISTS opening_balances (
    id CHAR(36) PRIMARY KEY,
    branch_id CHAR(36) NOT NULL,
    currency CHAR(3) NOT NULL,
    amount DECIMAL(19,4) NOT NULL,
    balance_date DATETIME(3) NOT NULL,
    notes VARCHAR(512) NULL,
    created_by_id CHAR(36) NOT NULL,
    status VARCHAR(16) NOT NULL DEFAULT 'approved',
    created_at DATETIME(3) NOT NULL,
    updated_at DATETIME(3) NOT NULL,
    deleted_at DATETIME(3) NULL,
    UNIQUE KEY uk_ob_branch_currency_date (branch_id, currency, balance_date),
    KEY idx_ob_deleted (deleted_at),
    CONSTRAINT fk_ob_branch FOREIGN KEY (branch_id) REFERENCES branches(id),
    CONSTRAINT fk_ob_user FOREIGN KEY (created_by_id) REFERENCES users(id),
    CONSTRAINT chk_ob_currency CHECK (currency IN ('USD', 'KHR'))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS cash_reconciliations (
    id CHAR(36) PRIMARY KEY,
    branch_id CHAR(36) NOT NULL,
    currency CHAR(3) NOT NULL,
    reconcile_date DATETIME(3) NOT NULL,
    expected_amount DECIMAL(19,4) NOT NULL,
    actual_amount DECIMAL(19,4) NOT NULL,
    difference DECIMAL(19,4) NOT NULL,
    notes VARCHAR(1024) NULL,
    created_by_id CHAR(36) NOT NULL,
    status VARCHAR(16) NOT NULL DEFAULT 'completed',
    created_at DATETIME(3) NOT NULL,
    updated_at DATETIME(3) NOT NULL,
    deleted_at DATETIME(3) NULL,
    KEY idx_cr_branch (branch_id),
    KEY idx_cr_date (reconcile_date),
    KEY idx_cr_deleted (deleted_at),
    CONSTRAINT fk_cr_branch FOREIGN KEY (branch_id) REFERENCES branches(id),
    CONSTRAINT fk_cr_user FOREIGN KEY (created_by_id) REFERENCES users(id),
    CONSTRAINT chk_cr_currency CHECK (currency IN ('USD', 'KHR'))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS transaction_approvals (
    id CHAR(36) PRIMARY KEY,
    transaction_id CHAR(36) NOT NULL,
    actor_id CHAR(36) NOT NULL,
    action VARCHAR(16) NOT NULL,
    notes VARCHAR(512) NULL,
    created_at DATETIME(3) NOT NULL,
    KEY idx_ta_txn (transaction_id),
    KEY idx_ta_actor (actor_id),
    CONSTRAINT fk_ta_txn FOREIGN KEY (transaction_id) REFERENCES transactions(id),
    CONSTRAINT fk_ta_actor FOREIGN KEY (actor_id) REFERENCES users(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
