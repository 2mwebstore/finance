-- Phase 3: Currency foundation, categories, payment methods, transactions

CREATE TABLE IF NOT EXISTS categories (
    id CHAR(36) PRIMARY KEY,
    code VARCHAR(64) NOT NULL,
    name VARCHAR(128) NOT NULL,
    type VARCHAR(16) NOT NULL COMMENT 'income or expense',
    description VARCHAR(512) NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    is_system BOOLEAN NOT NULL DEFAULT FALSE,
    created_at DATETIME(3) NOT NULL,
    updated_at DATETIME(3) NOT NULL,
    deleted_at DATETIME(3) NULL,
    UNIQUE KEY uk_category_type_code (type, code),
    KEY idx_categories_type (type),
    KEY idx_categories_deleted (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS payment_methods (
    id CHAR(36) PRIMARY KEY,
    code VARCHAR(64) NOT NULL,
    name VARCHAR(128) NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    is_system BOOLEAN NOT NULL DEFAULT FALSE,
    created_at DATETIME(3) NOT NULL,
    updated_at DATETIME(3) NOT NULL,
    deleted_at DATETIME(3) NULL,
    UNIQUE KEY uk_payment_methods_code (code),
    KEY idx_payment_methods_deleted (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS transaction_number_seqs (
    id CHAR(36) PRIMARY KEY,
    prefix VARCHAR(32) NOT NULL,
    last_value BIGINT NOT NULL DEFAULT 0,
    updated_at DATETIME(3) NOT NULL,
    UNIQUE KEY uk_txn_seq_prefix (prefix)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS transactions (
    id CHAR(36) PRIMARY KEY,
    transaction_number VARCHAR(32) NOT NULL,
    type VARCHAR(16) NOT NULL COMMENT 'income or expense',
    status VARCHAR(16) NOT NULL DEFAULT 'approved',
    branch_id CHAR(36) NOT NULL,
    category_id CHAR(36) NOT NULL,
    payment_method_id CHAR(36) NOT NULL,
    currency CHAR(3) NOT NULL COMMENT 'USD or KHR',
    amount DECIMAL(19,4) NOT NULL,
    transaction_date DATETIME(3) NOT NULL,
    description VARCHAR(1024) NULL,
    counterparty VARCHAR(255) NULL,
    created_by_id CHAR(36) NOT NULL,
    approved_by_id CHAR(36) NULL,
    approved_at DATETIME(3) NULL,
    voided_by_id CHAR(36) NULL,
    voided_at DATETIME(3) NULL,
    void_reason VARCHAR(512) NULL,
    reversed_by_txn_id CHAR(36) NULL,
    reverses_txn_id CHAR(36) NULL,
    created_at DATETIME(3) NOT NULL,
    updated_at DATETIME(3) NOT NULL,
    deleted_at DATETIME(3) NULL,
    UNIQUE KEY uk_transactions_number (transaction_number),
    KEY idx_txn_branch (branch_id),
    KEY idx_txn_type (type),
    KEY idx_txn_status (status),
    KEY idx_txn_currency (currency),
    KEY idx_txn_date (transaction_date),
    KEY idx_txn_category (category_id),
    KEY idx_txn_branch_currency_status_date (branch_id, currency, status, transaction_date),
    KEY idx_txn_deleted (deleted_at),
    CONSTRAINT fk_txn_branch FOREIGN KEY (branch_id) REFERENCES branches(id),
    CONSTRAINT fk_txn_category FOREIGN KEY (category_id) REFERENCES categories(id),
    CONSTRAINT fk_txn_payment FOREIGN KEY (payment_method_id) REFERENCES payment_methods(id),
    CONSTRAINT fk_txn_created_by FOREIGN KEY (created_by_id) REFERENCES users(id),
    CONSTRAINT chk_txn_currency CHECK (currency IN ('USD', 'KHR')),
    CONSTRAINT chk_txn_type CHECK (type IN ('income', 'expense')),
    CONSTRAINT chk_txn_amount_positive CHECK (amount > 0)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
