-- Roles decide which branches' data their users see: 'assigned' (each user's own branches),
-- 'custom' (the branches in role_branches) or 'all'. Owner, admin and Super Admin see all.
SET @add_scope = IF(
    (SELECT COUNT(*) FROM information_schema.columns
      WHERE table_schema = DATABASE() AND table_name = 'roles' AND column_name = 'data_scope') = 0,
    'ALTER TABLE roles ADD COLUMN data_scope VARCHAR(16) NOT NULL DEFAULT ''assigned''',
    'SELECT 1');
PREPARE add_scope FROM @add_scope;
EXECUTE add_scope;
DEALLOCATE PREPARE add_scope;
UPDATE roles SET data_scope = 'all' WHERE code IN ('super_admin', 'owner', 'admin');

CREATE TABLE IF NOT EXISTS role_branches (
    role_id CHAR(36) NOT NULL,
    branch_id CHAR(36) NOT NULL,
    PRIMARY KEY (role_id, branch_id),
    KEY idx_role_branches_branch_id (branch_id),
    CONSTRAINT fk_role_branches_role FOREIGN KEY (role_id) REFERENCES roles(id) ON DELETE CASCADE,
    CONSTRAINT fk_role_branches_branch FOREIGN KEY (branch_id) REFERENCES branches(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
