-- A transaction can have more than one attachment (up to 2: e.g. a receipt and an invoice).
-- The old single attachment columns on transactions are moved here and then cleared.
CREATE TABLE IF NOT EXISTS transaction_attachments (
    id CHAR(36) NOT NULL,
    transaction_id CHAR(36) NOT NULL,
    storage_key VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    content_type VARCHAR(128) NOT NULL,
    size BIGINT NOT NULL DEFAULT 0,
    created_by_id CHAR(36) NOT NULL,
    created_at DATETIME(3) NULL,
    PRIMARY KEY (id),
    KEY idx_transaction_attachments_transaction_id (transaction_id),
    CONSTRAINT fk_transaction_attachments_transaction FOREIGN KEY (transaction_id) REFERENCES transactions(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

INSERT INTO transaction_attachments (id, transaction_id, storage_key, name, content_type, size, created_by_id, created_at)
SELECT UUID(), id, attachment_key, COALESCE(attachment_name, ''), COALESCE(attachment_type, ''), attachment_size, created_by_id, updated_at
FROM transactions
WHERE attachment_key IS NOT NULL AND attachment_key <> '';

UPDATE transactions
SET attachment_key = '', attachment_name = '', attachment_type = '', attachment_size = 0
WHERE attachment_key IS NOT NULL AND attachment_key <> '';
