-- Platforms and weekly win/lose (HQ = branch): platforms, HQ-platform assignments, weekly results
-- and weekly settlements. Same SQL as internal/database/platforms.go. Only adds: safe to run again.
-- The tables take the collation of branches.id, so their foreign keys can be created whichever
-- collation the database uses.

SET @coll = COALESCE((SELECT collation_name FROM information_schema.columns
    WHERE table_schema = DATABASE() AND table_name = 'branches' AND column_name = 'id'), 'utf8mb4_unicode_ci');

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

-- hq_platform_assignments
SET @ddl = CONCAT('CREATE TABLE IF NOT EXISTS hq_platform_assignments (
    id CHAR(36) NOT NULL,
    branch_id CHAR(36) NOT NULL,
    platform_id CHAR(36) NOT NULL,
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
    UNIQUE KEY uq_hq_platform_active (branch_id, platform_id, active_key),
    KEY idx_hq_platform_assignments_platform (platform_id),
    KEY idx_hq_platform_assignments_deleted_at (deleted_at)
)', ' ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=', @coll);
PREPARE create_table FROM @ddl;
EXECUTE create_table;
DEALLOCATE PREPARE create_table;

-- weekly_platform_results
SET @ddl = CONCAT('CREATE TABLE IF NOT EXISTS weekly_platform_results (
    id CHAR(36) NOT NULL,
    week_start DATE NOT NULL,
    week_end DATE NOT NULL,
    branch_id CHAR(36) NOT NULL,
    platform_id CHAR(36) NOT NULL,
    assignment_id CHAR(36) NOT NULL,
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
    UNIQUE KEY uq_weekly_platform_result (week_start, branch_id, platform_id),
    KEY idx_weekly_platform_results_branch_week (branch_id, week_start),
    KEY idx_weekly_platform_results_platform (platform_id),
    KEY idx_weekly_platform_results_assignment (assignment_id),
    KEY idx_weekly_platform_results_deleted_at (deleted_at)
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

-- Foreign keys, each added only when missing.

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
