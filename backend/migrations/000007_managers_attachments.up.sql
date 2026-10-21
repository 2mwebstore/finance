-- Managers approve/reject their users' transactions; transactions get rich-text descriptions
-- and one optional attachment (stored in object storage, e.g. Cloudflare R2).
ALTER TABLE transactions
    MODIFY COLUMN description TEXT NULL,
    ADD COLUMN attachment_key VARCHAR(255) NULL,
    ADD COLUMN attachment_name VARCHAR(255) NULL,
    ADD COLUMN attachment_type VARCHAR(128) NULL,
    ADD COLUMN attachment_size BIGINT NOT NULL DEFAULT 0;

CREATE TABLE IF NOT EXISTS user_managers (
    user_id CHAR(36) NOT NULL,
    manager_id CHAR(36) NOT NULL,
    PRIMARY KEY (user_id, manager_id),
    KEY idx_user_managers_manager_id (manager_id),
    CONSTRAINT fk_user_managers_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT fk_user_managers_manager FOREIGN KEY (manager_id) REFERENCES users(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
