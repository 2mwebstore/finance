-- Platforms and weekly win/lose (HQ = branch): platforms, partners, titled groups of
-- branch-platform assignments and their partners, weekly results with each partner's share, titled weekly entries, and weekly settlements. Same SQL
-- as internal/database/platforms.go. Only adds (but for two replaced unique keys): safe to run again. The tables take the collation of
-- branches.id, so their foreign keys can be created whichever collation the database uses.

SET @coll = COALESCE((SELECT collation_name FROM information_schema.columns
    WHERE table_schema = DATABASE() AND table_name = 'branches' AND column_name = 'id'), 'utf8mb4_unicode_ci');

-- partners
SET @ddl = CONCAT('CREATE TABLE IF NOT EXISTS partners (
    id CHAR(36) NOT NULL,
    name VARCHAR(128) NOT NULL,
    phone VARCHAR(32) NOT NULL DEFAULT '''',
    note VARCHAR(255) NOT NULL DEFAULT '''',
    default_rate DECIMAL(7,4) NOT NULL DEFAULT 0,
    is_active TINYINT(1) NOT NULL DEFAULT 1,
    created_at DATETIME(3) NULL,
    updated_at DATETIME(3) NULL,
    deleted_at DATETIME(3) NULL,
    PRIMARY KEY (id),
    UNIQUE KEY uq_partners_name (name),
    KEY idx_partners_deleted_at (deleted_at)
)', ' ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=', @coll);
PREPARE create_table FROM @ddl;
EXECUTE create_table;
DEALLOCATE PREPARE create_table;

-- platforms
SET @ddl = CONCAT('CREATE TABLE IF NOT EXISTS platforms (
    id CHAR(36) NOT NULL,
    code VARCHAR(64) NOT NULL,
    name VARCHAR(128) NOT NULL,
    default_rate DECIMAL(7,4) NOT NULL DEFAULT 0,
    default_currency VARCHAR(3) NOT NULL DEFAULT ''USD'',
    is_active TINYINT(1) NOT NULL DEFAULT 1,
    created_at DATETIME(3) NULL,
    updated_at DATETIME(3) NULL,
    deleted_at DATETIME(3) NULL,
    PRIMARY KEY (id),
    UNIQUE KEY uq_platforms_code (code),
    KEY idx_platforms_deleted_at (deleted_at)
)', ' ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=', @coll);
PREPARE create_table FROM @ddl;
EXECUTE create_table;
DEALLOCATE PREPARE create_table;

-- assignment_groups
SET @ddl = CONCAT('CREATE TABLE IF NOT EXISTS assignment_groups (
    id CHAR(36) NOT NULL,
    branch_id CHAR(36) NOT NULL,
    title VARCHAR(128) NOT NULL,
    currency VARCHAR(3) NOT NULL DEFAULT '''',
    starts_on DATE NOT NULL,
    is_active TINYINT(1) NOT NULL DEFAULT 1,
    created_by_id CHAR(36) NOT NULL,
    created_at DATETIME(3) NULL,
    updated_at DATETIME(3) NULL,
    PRIMARY KEY (id),
    KEY idx_assignment_groups_branch (branch_id)
)', ' ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=', @coll);
PREPARE create_table FROM @ddl;
EXECUTE create_table;
DEALLOCATE PREPARE create_table;

-- hq_platform_assignments
SET @ddl = CONCAT('CREATE TABLE IF NOT EXISTS hq_platform_assignments (
    id CHAR(36) NOT NULL,
    branch_id CHAR(36) NOT NULL,
    platform_id CHAR(36) NOT NULL,
    group_id CHAR(36) NULL,
    commission_rate DECIMAL(7,4) NOT NULL,
    currency VARCHAR(3) NOT NULL,
    is_active TINYINT(1) NOT NULL DEFAULT 1,
    active_key TINYINT NULL,
    starts_on DATE NOT NULL,
    ends_on DATE NULL,
    created_by_id CHAR(36) NOT NULL,
    ended_by_id CHAR(36) NULL,
    end_reason VARCHAR(255) NOT NULL DEFAULT '''',
    created_at DATETIME(3) NULL,
    updated_at DATETIME(3) NULL,
    deleted_at DATETIME(3) NULL,
    PRIMARY KEY (id),
    UNIQUE KEY uq_hq_platform_group_active (group_id, platform_id, active_key),
    KEY idx_hq_platform_assignments_branch (branch_id),
    KEY idx_hq_platform_assignments_platform (platform_id),
    KEY idx_hq_platform_assignments_group (group_id),
    KEY idx_hq_platform_assignments_deleted_at (deleted_at)
)', ' ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=', @coll);
PREPARE create_table FROM @ddl;
EXECUTE create_table;
DEALLOCATE PREPARE create_table;

-- assignment_partners
SET @ddl = CONCAT('CREATE TABLE IF NOT EXISTS assignment_partners (
    id CHAR(36) NOT NULL,
    assignment_id CHAR(36) NOT NULL,
    partner_id CHAR(36) NOT NULL,
    commission_rate DECIMAL(7,4) NOT NULL,
    created_at DATETIME(3) NULL,
    updated_at DATETIME(3) NULL,
    PRIMARY KEY (id),
    UNIQUE KEY uq_assignment_partner (assignment_id, partner_id),
    KEY idx_assignment_partners_partner (partner_id)
)', ' ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=', @coll);
PREPARE create_table FROM @ddl;
EXECUTE create_table;
DEALLOCATE PREPARE create_table;

-- weekly_platform_results
SET @ddl = CONCAT('CREATE TABLE IF NOT EXISTS weekly_platform_results (
    id CHAR(36) NOT NULL,
    entry_id CHAR(36) NULL,
    week_start DATE NOT NULL,
    week_end DATE NOT NULL,
    branch_id CHAR(36) NOT NULL,
    platform_id CHAR(36) NOT NULL,
    assignment_id CHAR(36) NULL,
    win_lose DECIMAL(19,4) NOT NULL,
    commission DECIMAL(19,4) NOT NULL,
    commission_rate DECIMAL(7,4) NOT NULL,
    currency VARCHAR(3) NOT NULL,
    adjusted_amount DECIMAL(19,4) NOT NULL,
    jame_ps DECIMAL(19,4) NOT NULL,
    formula_version VARCHAR(16) NOT NULL,
    created_by_id CHAR(36) NOT NULL,
    updated_by_id CHAR(36) NULL,
    created_at DATETIME(3) NULL,
    updated_at DATETIME(3) NULL,
    deleted_at DATETIME(3) NULL,
    PRIMARY KEY (id),
    UNIQUE KEY uq_weekly_result_entry (entry_id, platform_id),
    KEY idx_weekly_platform_results_branch_week (branch_id, week_start),
    KEY idx_weekly_platform_results_platform (platform_id),
    KEY idx_weekly_platform_results_assignment (assignment_id),
    KEY idx_weekly_platform_results_deleted_at (deleted_at)
)', ' ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=', @coll);
PREPARE create_table FROM @ddl;
EXECUTE create_table;
DEALLOCATE PREPARE create_table;

-- weekly_partner_results
SET @ddl = CONCAT('CREATE TABLE IF NOT EXISTS weekly_partner_results (
    id CHAR(36) NOT NULL,
    result_id CHAR(36) NOT NULL,
    partner_id CHAR(36) NOT NULL,
    commission_rate DECIMAL(7,4) NOT NULL,
    jame_ps DECIMAL(19,4) NOT NULL,
    created_at DATETIME(3) NULL,
    updated_at DATETIME(3) NULL,
    PRIMARY KEY (id),
    UNIQUE KEY uq_weekly_partner_result (result_id, partner_id),
    KEY idx_weekly_partner_results_partner (partner_id)
)', ' ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=', @coll);
PREPARE create_table FROM @ddl;
EXECUTE create_table;
DEALLOCATE PREPARE create_table;

-- weekly_group_results
SET @ddl = CONCAT('CREATE TABLE IF NOT EXISTS weekly_group_results (
    id CHAR(36) NOT NULL,
    result_id CHAR(36) NOT NULL,
    group_id CHAR(36) NULL,
    assignment_id CHAR(36) NOT NULL,
    commission_rate DECIMAL(7,4) NOT NULL,
    jame_ps DECIMAL(19,4) NOT NULL,
    created_at DATETIME(3) NULL,
    updated_at DATETIME(3) NULL,
    PRIMARY KEY (id),
    UNIQUE KEY uq_weekly_group_result (result_id, assignment_id),
    KEY idx_weekly_group_results_group (group_id),
    KEY idx_weekly_group_results_assignment (assignment_id)
)', ' ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=', @coll);
PREPARE create_table FROM @ddl;
EXECUTE create_table;
DEALLOCATE PREPARE create_table;

-- weekly_entries
SET @ddl = CONCAT('CREATE TABLE IF NOT EXISTS weekly_entries (
    id CHAR(36) NOT NULL,
    branch_id CHAR(36) NOT NULL,
    week_start DATE NOT NULL,
    title VARCHAR(128) NOT NULL,
    created_by_id CHAR(36) NOT NULL,
    updated_by_id CHAR(36) NULL,
    created_at DATETIME(3) NULL,
    updated_at DATETIME(3) NULL,
    PRIMARY KEY (id),
    KEY idx_weekly_entries_branch_week (branch_id, week_start),
    KEY idx_weekly_entries_created_at (created_at)
)', ' ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=', @coll);
PREPARE create_table FROM @ddl;
EXECUTE create_table;
DEALLOCATE PREPARE create_table;

-- weekly_settlements
SET @ddl = CONCAT('CREATE TABLE IF NOT EXISTS weekly_settlements (
    id CHAR(36) NOT NULL,
    branch_id CHAR(36) NOT NULL,
    week_start DATE NOT NULL,
    week_end DATE NOT NULL,
    status VARCHAR(16) NOT NULL,
    finalized_by_id CHAR(36) NOT NULL,
    finalized_at DATETIME(3) NOT NULL,
    reopened_by_id CHAR(36) NULL,
    reopened_at DATETIME(3) NULL,
    reopen_reason VARCHAR(512) NOT NULL DEFAULT '''',
    created_at DATETIME(3) NULL,
    updated_at DATETIME(3) NULL,
    deleted_at DATETIME(3) NULL,
    PRIMARY KEY (id),
    UNIQUE KEY uq_weekly_settlement (branch_id, week_start),
    KEY idx_weekly_settlements_deleted_at (deleted_at)
)', ' ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=', @coll);
PREPARE create_table FROM @ddl;
EXECUTE create_table;
DEALLOCATE PREPARE create_table;

-- weekly_settlement_totals
SET @ddl = CONCAT('CREATE TABLE IF NOT EXISTS weekly_settlement_totals (
    id CHAR(36) NOT NULL,
    settlement_id CHAR(36) NOT NULL,
    currency VARCHAR(3) NOT NULL,
    win_lose DECIMAL(19,4) NOT NULL,
    commission DECIMAL(19,4) NOT NULL,
    adjusted_amount DECIMAL(19,4) NOT NULL,
    jame_ps DECIMAL(19,4) NOT NULL,
    entries INT NOT NULL DEFAULT 0,
    PRIMARY KEY (id),
    UNIQUE KEY uq_weekly_settlement_total (settlement_id, currency)
)', ' ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=', @coll);
PREPARE create_table FROM @ddl;
EXECUTE create_table;
DEALLOCATE PREPARE create_table;

-- Columns added after the first builds, each added only when missing.

SET @col = IF(
    (SELECT COUNT(*) FROM information_schema.columns
      WHERE table_schema = DATABASE() AND table_name = 'hq_platform_assignments' AND column_name = 'group_id') = 0,
    'ALTER TABLE hq_platform_assignments ADD COLUMN group_id CHAR(36) NULL AFTER platform_id, ADD KEY idx_hq_platform_assignments_group (group_id)',
    'SELECT 1');
PREPARE add_column FROM @col;
EXECUTE add_column;
DEALLOCATE PREPARE add_column;

SET @col = IF(
    (SELECT COUNT(*) FROM information_schema.columns
      WHERE table_schema = DATABASE() AND table_name = 'weekly_platform_results' AND column_name = 'entry_id') = 0,
    'ALTER TABLE weekly_platform_results ADD COLUMN entry_id CHAR(36) NULL AFTER id',
    'SELECT 1');
PREPARE add_column FROM @col;
EXECUTE add_column;
DEALLOCATE PREPARE add_column;

-- Columns that may be empty since the first builds: a result saved on an entry has no assignment
-- of its own (its groups are in weekly_group_results).

SET @col = IF(
    (SELECT is_nullable FROM information_schema.columns
      WHERE table_schema = DATABASE() AND table_name = 'weekly_platform_results' AND column_name = 'assignment_id') = 'NO',
    'ALTER TABLE weekly_platform_results MODIFY assignment_id CHAR(36) NULL',
    'SELECT 1');
PREPARE null_column FROM @col;
EXECUTE null_column;
DEALLOCATE PREPARE null_column;

-- Weeks saved before entries could hold any agent: each result goes on its week's entry (one per
-- group and week then), and its Total in its assignment's group. Safe to run again.

UPDATE weekly_platform_results r JOIN weekly_entries e ON e.branch_id = r.branch_id AND e.week_start = r.week_start
	SET r.entry_id = e.id WHERE r.entry_id IS NULL;

INSERT IGNORE INTO weekly_group_results (id, result_id, group_id, assignment_id, commission_rate, jame_ps, created_at, updated_at)
	SELECT UUID(), r.id, a.group_id, a.id, r.commission_rate, r.jame_ps, NOW(3), NOW(3)
	FROM weekly_platform_results r JOIN hq_platform_assignments a ON a.id = r.assignment_id;

-- Keys added after the first builds, each added only when missing. An agent can be in several
-- groups of a branch: an active assignment is one per group. A week can have several entries: a
-- result is one per entry and agent. The keys they replace go once the new ones are in place.

SET @key = IF(
    (SELECT COUNT(*) FROM information_schema.statistics
      WHERE table_schema = DATABASE() AND table_name = 'hq_platform_assignments' AND index_name = 'idx_hq_platform_assignments_branch') = 0,
    'ALTER TABLE hq_platform_assignments ADD KEY idx_hq_platform_assignments_branch (branch_id)',
    'SELECT 1');
PREPARE add_key FROM @key;
EXECUTE add_key;
DEALLOCATE PREPARE add_key;

SET @key = IF(
    (SELECT COUNT(*) FROM information_schema.statistics
      WHERE table_schema = DATABASE() AND table_name = 'hq_platform_assignments' AND index_name = 'uq_hq_platform_group_active') = 0,
    'ALTER TABLE hq_platform_assignments ADD UNIQUE KEY uq_hq_platform_group_active (group_id, platform_id, active_key)',
    'SELECT 1');
PREPARE add_key FROM @key;
EXECUTE add_key;
DEALLOCATE PREPARE add_key;

SET @key = IF(
    (SELECT COUNT(*) FROM information_schema.statistics
      WHERE table_schema = DATABASE() AND table_name = 'weekly_entries' AND index_name = 'idx_weekly_entries_branch_week') = 0,
    'ALTER TABLE weekly_entries ADD KEY idx_weekly_entries_branch_week (branch_id, week_start)',
    'SELECT 1');
PREPARE add_key FROM @key;
EXECUTE add_key;
DEALLOCATE PREPARE add_key;

SET @key = IF(
    (SELECT COUNT(*) FROM information_schema.statistics
      WHERE table_schema = DATABASE() AND table_name = 'weekly_platform_results' AND index_name = 'uq_weekly_result_entry') = 0,
    'ALTER TABLE weekly_platform_results ADD UNIQUE KEY uq_weekly_result_entry (entry_id, platform_id)',
    'SELECT 1');
PREPARE add_key FROM @key;
EXECUTE add_key;
DEALLOCATE PREPARE add_key;

SET @key = IF(
    (SELECT COUNT(*) FROM information_schema.statistics
      WHERE table_schema = DATABASE() AND table_name = 'hq_platform_assignments' AND index_name = 'uq_hq_platform_active') > 0,
    'ALTER TABLE hq_platform_assignments DROP INDEX uq_hq_platform_active',
    'SELECT 1');
PREPARE drop_key FROM @key;
EXECUTE drop_key;
DEALLOCATE PREPARE drop_key;

SET @key = IF(
    (SELECT COUNT(*) FROM information_schema.statistics
      WHERE table_schema = DATABASE() AND table_name = 'weekly_platform_results' AND index_name = 'uq_weekly_platform_result') > 0,
    'ALTER TABLE weekly_platform_results DROP INDEX uq_weekly_platform_result',
    'SELECT 1');
PREPARE drop_key FROM @key;
EXECUTE drop_key;
DEALLOCATE PREPARE drop_key;

SET @key = IF(
    (SELECT COUNT(*) FROM information_schema.statistics
      WHERE table_schema = DATABASE() AND table_name = 'weekly_platform_results' AND index_name = 'uq_weekly_result_assignment') > 0,
    'ALTER TABLE weekly_platform_results DROP INDEX uq_weekly_result_assignment',
    'SELECT 1');
PREPARE drop_key FROM @key;
EXECUTE drop_key;
DEALLOCATE PREPARE drop_key;

SET @key = IF(
    (SELECT COUNT(*) FROM information_schema.statistics
      WHERE table_schema = DATABASE() AND table_name = 'weekly_entries' AND index_name = 'uq_weekly_entry') > 0,
    'ALTER TABLE weekly_entries DROP INDEX uq_weekly_entry',
    'SELECT 1');
PREPARE drop_key FROM @key;
EXECUTE drop_key;
DEALLOCATE PREPARE drop_key;

-- Foreign keys, each added only when missing.

SET @fk = IF(
    (SELECT COUNT(*) FROM information_schema.table_constraints
      WHERE table_schema = DATABASE() AND table_name = 'assignment_groups' AND constraint_name = 'fk_assignment_groups_branch' AND constraint_type = 'FOREIGN KEY') = 0,
    'ALTER TABLE assignment_groups ADD CONSTRAINT fk_assignment_groups_branch FOREIGN KEY (branch_id) REFERENCES branches(id)',
    'SELECT 1');
PREPARE add_fk FROM @fk;
EXECUTE add_fk;
DEALLOCATE PREPARE add_fk;

SET @fk = IF(
    (SELECT COUNT(*) FROM information_schema.table_constraints
      WHERE table_schema = DATABASE() AND table_name = 'assignment_groups' AND constraint_name = 'fk_assignment_groups_created_by' AND constraint_type = 'FOREIGN KEY') = 0,
    'ALTER TABLE assignment_groups ADD CONSTRAINT fk_assignment_groups_created_by FOREIGN KEY (created_by_id) REFERENCES users(id)',
    'SELECT 1');
PREPARE add_fk FROM @fk;
EXECUTE add_fk;
DEALLOCATE PREPARE add_fk;

SET @fk = IF(
    (SELECT COUNT(*) FROM information_schema.table_constraints
      WHERE table_schema = DATABASE() AND table_name = 'hq_platform_assignments' AND constraint_name = 'fk_hq_platform_assignments_group' AND constraint_type = 'FOREIGN KEY') = 0,
    'ALTER TABLE hq_platform_assignments ADD CONSTRAINT fk_hq_platform_assignments_group FOREIGN KEY (group_id) REFERENCES assignment_groups(id)',
    'SELECT 1');
PREPARE add_fk FROM @fk;
EXECUTE add_fk;
DEALLOCATE PREPARE add_fk;

SET @fk = IF(
    (SELECT COUNT(*) FROM information_schema.table_constraints
      WHERE table_schema = DATABASE() AND table_name = 'hq_platform_assignments' AND constraint_name = 'fk_hq_platform_assignments_branch' AND constraint_type = 'FOREIGN KEY') = 0,
    'ALTER TABLE hq_platform_assignments ADD CONSTRAINT fk_hq_platform_assignments_branch FOREIGN KEY (branch_id) REFERENCES branches(id)',
    'SELECT 1');
PREPARE add_fk FROM @fk;
EXECUTE add_fk;
DEALLOCATE PREPARE add_fk;

SET @fk = IF(
    (SELECT COUNT(*) FROM information_schema.table_constraints
      WHERE table_schema = DATABASE() AND table_name = 'hq_platform_assignments' AND constraint_name = 'fk_hq_platform_assignments_platform' AND constraint_type = 'FOREIGN KEY') = 0,
    'ALTER TABLE hq_platform_assignments ADD CONSTRAINT fk_hq_platform_assignments_platform FOREIGN KEY (platform_id) REFERENCES platforms(id)',
    'SELECT 1');
PREPARE add_fk FROM @fk;
EXECUTE add_fk;
DEALLOCATE PREPARE add_fk;

SET @fk = IF(
    (SELECT COUNT(*) FROM information_schema.table_constraints
      WHERE table_schema = DATABASE() AND table_name = 'hq_platform_assignments' AND constraint_name = 'fk_hq_platform_assignments_created_by' AND constraint_type = 'FOREIGN KEY') = 0,
    'ALTER TABLE hq_platform_assignments ADD CONSTRAINT fk_hq_platform_assignments_created_by FOREIGN KEY (created_by_id) REFERENCES users(id)',
    'SELECT 1');
PREPARE add_fk FROM @fk;
EXECUTE add_fk;
DEALLOCATE PREPARE add_fk;

SET @fk = IF(
    (SELECT COUNT(*) FROM information_schema.table_constraints
      WHERE table_schema = DATABASE() AND table_name = 'assignment_partners' AND constraint_name = 'fk_assignment_partners_assignment' AND constraint_type = 'FOREIGN KEY') = 0,
    'ALTER TABLE assignment_partners ADD CONSTRAINT fk_assignment_partners_assignment FOREIGN KEY (assignment_id) REFERENCES hq_platform_assignments(id) ON DELETE CASCADE',
    'SELECT 1');
PREPARE add_fk FROM @fk;
EXECUTE add_fk;
DEALLOCATE PREPARE add_fk;

SET @fk = IF(
    (SELECT COUNT(*) FROM information_schema.table_constraints
      WHERE table_schema = DATABASE() AND table_name = 'assignment_partners' AND constraint_name = 'fk_assignment_partners_partner' AND constraint_type = 'FOREIGN KEY') = 0,
    'ALTER TABLE assignment_partners ADD CONSTRAINT fk_assignment_partners_partner FOREIGN KEY (partner_id) REFERENCES partners(id)',
    'SELECT 1');
PREPARE add_fk FROM @fk;
EXECUTE add_fk;
DEALLOCATE PREPARE add_fk;

SET @fk = IF(
    (SELECT COUNT(*) FROM information_schema.table_constraints
      WHERE table_schema = DATABASE() AND table_name = 'weekly_platform_results' AND constraint_name = 'fk_weekly_platform_results_branch' AND constraint_type = 'FOREIGN KEY') = 0,
    'ALTER TABLE weekly_platform_results ADD CONSTRAINT fk_weekly_platform_results_branch FOREIGN KEY (branch_id) REFERENCES branches(id)',
    'SELECT 1');
PREPARE add_fk FROM @fk;
EXECUTE add_fk;
DEALLOCATE PREPARE add_fk;

SET @fk = IF(
    (SELECT COUNT(*) FROM information_schema.table_constraints
      WHERE table_schema = DATABASE() AND table_name = 'weekly_platform_results' AND constraint_name = 'fk_weekly_platform_results_platform' AND constraint_type = 'FOREIGN KEY') = 0,
    'ALTER TABLE weekly_platform_results ADD CONSTRAINT fk_weekly_platform_results_platform FOREIGN KEY (platform_id) REFERENCES platforms(id)',
    'SELECT 1');
PREPARE add_fk FROM @fk;
EXECUTE add_fk;
DEALLOCATE PREPARE add_fk;

SET @fk = IF(
    (SELECT COUNT(*) FROM information_schema.table_constraints
      WHERE table_schema = DATABASE() AND table_name = 'weekly_platform_results' AND constraint_name = 'fk_weekly_platform_results_assignment' AND constraint_type = 'FOREIGN KEY') = 0,
    'ALTER TABLE weekly_platform_results ADD CONSTRAINT fk_weekly_platform_results_assignment FOREIGN KEY (assignment_id) REFERENCES hq_platform_assignments(id)',
    'SELECT 1');
PREPARE add_fk FROM @fk;
EXECUTE add_fk;
DEALLOCATE PREPARE add_fk;

SET @fk = IF(
    (SELECT COUNT(*) FROM information_schema.table_constraints
      WHERE table_schema = DATABASE() AND table_name = 'weekly_platform_results' AND constraint_name = 'fk_weekly_platform_results_created_by' AND constraint_type = 'FOREIGN KEY') = 0,
    'ALTER TABLE weekly_platform_results ADD CONSTRAINT fk_weekly_platform_results_created_by FOREIGN KEY (created_by_id) REFERENCES users(id)',
    'SELECT 1');
PREPARE add_fk FROM @fk;
EXECUTE add_fk;
DEALLOCATE PREPARE add_fk;

SET @fk = IF(
    (SELECT COUNT(*) FROM information_schema.table_constraints
      WHERE table_schema = DATABASE() AND table_name = 'weekly_partner_results' AND constraint_name = 'fk_weekly_partner_results_result' AND constraint_type = 'FOREIGN KEY') = 0,
    'ALTER TABLE weekly_partner_results ADD CONSTRAINT fk_weekly_partner_results_result FOREIGN KEY (result_id) REFERENCES weekly_platform_results(id) ON DELETE CASCADE',
    'SELECT 1');
PREPARE add_fk FROM @fk;
EXECUTE add_fk;
DEALLOCATE PREPARE add_fk;

SET @fk = IF(
    (SELECT COUNT(*) FROM information_schema.table_constraints
      WHERE table_schema = DATABASE() AND table_name = 'weekly_partner_results' AND constraint_name = 'fk_weekly_partner_results_partner' AND constraint_type = 'FOREIGN KEY') = 0,
    'ALTER TABLE weekly_partner_results ADD CONSTRAINT fk_weekly_partner_results_partner FOREIGN KEY (partner_id) REFERENCES partners(id)',
    'SELECT 1');
PREPARE add_fk FROM @fk;
EXECUTE add_fk;
DEALLOCATE PREPARE add_fk;

SET @fk = IF(
    (SELECT COUNT(*) FROM information_schema.table_constraints
      WHERE table_schema = DATABASE() AND table_name = 'weekly_platform_results' AND constraint_name = 'fk_weekly_platform_results_entry' AND constraint_type = 'FOREIGN KEY') = 0,
    'ALTER TABLE weekly_platform_results ADD CONSTRAINT fk_weekly_platform_results_entry FOREIGN KEY (entry_id) REFERENCES weekly_entries(id)',
    'SELECT 1');
PREPARE add_fk FROM @fk;
EXECUTE add_fk;
DEALLOCATE PREPARE add_fk;

SET @fk = IF(
    (SELECT COUNT(*) FROM information_schema.table_constraints
      WHERE table_schema = DATABASE() AND table_name = 'weekly_group_results' AND constraint_name = 'fk_weekly_group_results_result' AND constraint_type = 'FOREIGN KEY') = 0,
    'ALTER TABLE weekly_group_results ADD CONSTRAINT fk_weekly_group_results_result FOREIGN KEY (result_id) REFERENCES weekly_platform_results(id) ON DELETE CASCADE',
    'SELECT 1');
PREPARE add_fk FROM @fk;
EXECUTE add_fk;
DEALLOCATE PREPARE add_fk;

SET @fk = IF(
    (SELECT COUNT(*) FROM information_schema.table_constraints
      WHERE table_schema = DATABASE() AND table_name = 'weekly_group_results' AND constraint_name = 'fk_weekly_group_results_group' AND constraint_type = 'FOREIGN KEY') = 0,
    'ALTER TABLE weekly_group_results ADD CONSTRAINT fk_weekly_group_results_group FOREIGN KEY (group_id) REFERENCES assignment_groups(id)',
    'SELECT 1');
PREPARE add_fk FROM @fk;
EXECUTE add_fk;
DEALLOCATE PREPARE add_fk;

SET @fk = IF(
    (SELECT COUNT(*) FROM information_schema.table_constraints
      WHERE table_schema = DATABASE() AND table_name = 'weekly_group_results' AND constraint_name = 'fk_weekly_group_results_assignment' AND constraint_type = 'FOREIGN KEY') = 0,
    'ALTER TABLE weekly_group_results ADD CONSTRAINT fk_weekly_group_results_assignment FOREIGN KEY (assignment_id) REFERENCES hq_platform_assignments(id)',
    'SELECT 1');
PREPARE add_fk FROM @fk;
EXECUTE add_fk;
DEALLOCATE PREPARE add_fk;

SET @fk = IF(
    (SELECT COUNT(*) FROM information_schema.table_constraints
      WHERE table_schema = DATABASE() AND table_name = 'weekly_entries' AND constraint_name = 'fk_weekly_entries_branch' AND constraint_type = 'FOREIGN KEY') = 0,
    'ALTER TABLE weekly_entries ADD CONSTRAINT fk_weekly_entries_branch FOREIGN KEY (branch_id) REFERENCES branches(id)',
    'SELECT 1');
PREPARE add_fk FROM @fk;
EXECUTE add_fk;
DEALLOCATE PREPARE add_fk;

SET @fk = IF(
    (SELECT COUNT(*) FROM information_schema.table_constraints
      WHERE table_schema = DATABASE() AND table_name = 'weekly_entries' AND constraint_name = 'fk_weekly_entries_created_by' AND constraint_type = 'FOREIGN KEY') = 0,
    'ALTER TABLE weekly_entries ADD CONSTRAINT fk_weekly_entries_created_by FOREIGN KEY (created_by_id) REFERENCES users(id)',
    'SELECT 1');
PREPARE add_fk FROM @fk;
EXECUTE add_fk;
DEALLOCATE PREPARE add_fk;

SET @fk = IF(
    (SELECT COUNT(*) FROM information_schema.table_constraints
      WHERE table_schema = DATABASE() AND table_name = 'weekly_settlements' AND constraint_name = 'fk_weekly_settlements_branch' AND constraint_type = 'FOREIGN KEY') = 0,
    'ALTER TABLE weekly_settlements ADD CONSTRAINT fk_weekly_settlements_branch FOREIGN KEY (branch_id) REFERENCES branches(id)',
    'SELECT 1');
PREPARE add_fk FROM @fk;
EXECUTE add_fk;
DEALLOCATE PREPARE add_fk;

SET @fk = IF(
    (SELECT COUNT(*) FROM information_schema.table_constraints
      WHERE table_schema = DATABASE() AND table_name = 'weekly_settlements' AND constraint_name = 'fk_weekly_settlements_finalized_by' AND constraint_type = 'FOREIGN KEY') = 0,
    'ALTER TABLE weekly_settlements ADD CONSTRAINT fk_weekly_settlements_finalized_by FOREIGN KEY (finalized_by_id) REFERENCES users(id)',
    'SELECT 1');
PREPARE add_fk FROM @fk;
EXECUTE add_fk;
DEALLOCATE PREPARE add_fk;

SET @fk = IF(
    (SELECT COUNT(*) FROM information_schema.table_constraints
      WHERE table_schema = DATABASE() AND table_name = 'weekly_settlement_totals' AND constraint_name = 'fk_weekly_settlement_totals_settlement' AND constraint_type = 'FOREIGN KEY') = 0,
    'ALTER TABLE weekly_settlement_totals ADD CONSTRAINT fk_weekly_settlement_totals_settlement FOREIGN KEY (settlement_id) REFERENCES weekly_settlements(id) ON DELETE CASCADE',
    'SELECT 1');
PREPARE add_fk FROM @fk;
EXECUTE add_fk;
DEALLOCATE PREPARE add_fk;

-- The platforms already in use (rates are set on the Platforms page). Kept if they exist.

INSERT IGNORE INTO platforms (id, code, name, default_rate, default_currency, is_active, created_at, updated_at) VALUES
    (UUID(), 'LCMMJB', 'LCMMJB', 0, 'USD', 1, NOW(3), NOW(3)),
    (UUID(), '12K_YMLLJB_V1', '12K_YMLLJB_V1', 0, 'USD', 1, NOW(3), NOW(3));
