-- Categories can be shared (branch_id NULL, managed by owner/admin) or belong to one branch.
ALTER TABLE categories
    ADD COLUMN branch_id CHAR(36) NULL AFTER is_system,
    ADD KEY idx_categories_branch (branch_id);
