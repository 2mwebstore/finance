package database

import (
	"log"
	"regexp"

	"gorm.io/gorm"
)

// Platform tables (HQ = branch): platforms, their HQ assignments, weekly win/lose results and
// weekly settlements. Same SQL as migrations/000013_platforms.up.sql (a test keeps them in step).
// Only ever adds: tables that are missing, then foreign keys that are missing.
//
// The tables take the collation of branches.id: foreign keys need the same collation on both
// sides, and databases differ (the SQL migrations use utf8mb4_unicode_ci; one built by AutoMigrate
// on MySQL 8 has utf8mb4_0900_ai_ci).

var collationName = regexp.MustCompile(`^utf8mb4_[0-9a-z_]+$`)

// branchCollation is the collation of branches.id (utf8mb4_unicode_ci if it can't be read).
func branchCollation(db *gorm.DB) string {
	var name string
	db.Raw(`SELECT collation_name FROM information_schema.columns
		WHERE table_schema = DATABASE() AND table_name = 'branches' AND column_name = 'id'`).Scan(&name)
	if !collationName.MatchString(name) {
		return "utf8mb4_unicode_ci"
	}
	return name
}

func tableOptions(collation string) string {
	return " ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=" + collation
}

var platformTables = []string{
	`CREATE TABLE IF NOT EXISTS platforms (
	id CHAR(36) NOT NULL,
	code VARCHAR(64) NOT NULL,
	name VARCHAR(128) NOT NULL,
	default_rate DECIMAL(7,4) NOT NULL DEFAULT 0,
	default_currency VARCHAR(3) NOT NULL DEFAULT 'USD',
	is_active TINYINT(1) NOT NULL DEFAULT 1,
	created_at DATETIME(3) NULL,
	updated_at DATETIME(3) NULL,
	deleted_at DATETIME(3) NULL,
	PRIMARY KEY (id),
	UNIQUE KEY uq_platforms_code (code),
	KEY idx_platforms_deleted_at (deleted_at)
)`,
	`CREATE TABLE IF NOT EXISTS hq_platform_assignments (
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
	end_reason VARCHAR(255) NOT NULL DEFAULT '',
	created_at DATETIME(3) NULL,
	updated_at DATETIME(3) NULL,
	deleted_at DATETIME(3) NULL,
	PRIMARY KEY (id),
	UNIQUE KEY uq_hq_platform_active (branch_id, platform_id, active_key),
	KEY idx_hq_platform_assignments_platform (platform_id),
	KEY idx_hq_platform_assignments_deleted_at (deleted_at)
)`,
	`CREATE TABLE IF NOT EXISTS weekly_platform_results (
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
)`,
	`CREATE TABLE IF NOT EXISTS weekly_settlements (
	id CHAR(36) NOT NULL,
	branch_id CHAR(36) NOT NULL,
	week_start DATE NOT NULL,
	week_end DATE NOT NULL,
	status VARCHAR(16) NOT NULL,
	finalized_by_id CHAR(36) NOT NULL,
	finalized_at DATETIME(3) NOT NULL,
	reopened_by_id CHAR(36) NULL,
	reopened_at DATETIME(3) NULL,
	reopen_reason VARCHAR(512) NOT NULL DEFAULT '',
	created_at DATETIME(3) NULL,
	updated_at DATETIME(3) NULL,
	deleted_at DATETIME(3) NULL,
	PRIMARY KEY (id),
	UNIQUE KEY uq_weekly_settlement (branch_id, week_start),
	KEY idx_weekly_settlements_deleted_at (deleted_at)
)`,
	`CREATE TABLE IF NOT EXISTS weekly_settlement_totals (
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
)`,
}

type foreignKey struct{ table, name, columns, ref string }

var platformForeignKeys = []foreignKey{
	{"hq_platform_assignments", "fk_hq_platform_assignments_branch", "branch_id", "branches(id)"},
	{"hq_platform_assignments", "fk_hq_platform_assignments_platform", "platform_id", "platforms(id)"},
	{"hq_platform_assignments", "fk_hq_platform_assignments_created_by", "created_by_id", "users(id)"},
	{"weekly_platform_results", "fk_weekly_platform_results_branch", "branch_id", "branches(id)"},
	{"weekly_platform_results", "fk_weekly_platform_results_platform", "platform_id", "platforms(id)"},
	{"weekly_platform_results", "fk_weekly_platform_results_assignment", "assignment_id", "hq_platform_assignments(id)"},
	{"weekly_platform_results", "fk_weekly_platform_results_created_by", "created_by_id", "users(id)"},
	{"weekly_settlements", "fk_weekly_settlements_branch", "branch_id", "branches(id)"},
	{"weekly_settlements", "fk_weekly_settlements_finalized_by", "finalized_by_id", "users(id)"},
	{"weekly_settlement_totals", "fk_weekly_settlement_totals_settlement", "settlement_id", "weekly_settlements(id) ON DELETE CASCADE"},
}

func (fk foreignKey) sql() string {
	return "ALTER TABLE " + fk.table + " ADD CONSTRAINT " + fk.name + " FOREIGN KEY (" + fk.columns + ") REFERENCES " + fk.ref
}

// The platforms the business already uses; added once, when the platforms table is created.
// Their rates are set on the Platforms page.
const seedPlatforms = `INSERT IGNORE INTO platforms (id, code, name, default_rate, default_currency, is_active, created_at, updated_at) VALUES
	(UUID(), 'LCMMJB', 'LCMMJB', 0, 'USD', 1, NOW(3), NOW(3)),
	(UUID(), '12K_YMLLJB_V1', '12K_YMLLJB_V1', 0, 'USD', 1, NOW(3), NOW(3))`

// EnsurePlatformTables creates the platform tables when missing and adds missing foreign keys.
// A foreign key that can't be added (e.g. an older table with another collation) is logged and
// skipped, so the server still starts; the unique keys that stop duplicates are in the tables.
func EnsurePlatformTables(db *gorm.DB) error {
	if !db.Migrator().HasTable("branches") || !db.Migrator().HasTable("users") {
		return nil // a brand-new database: AutoMigrate or the SQL migrations come first
	}
	created := !db.Migrator().HasTable("platforms")
	options := tableOptions(branchCollation(db))
	for _, stmt := range platformTables {
		if err := db.Exec(stmt + options).Error; err != nil {
			return err
		}
	}
	for _, fk := range platformForeignKeys {
		var n int64
		if err := db.Raw(`SELECT COUNT(*) FROM information_schema.table_constraints
			WHERE table_schema = DATABASE() AND table_name = ? AND constraint_name = ? AND constraint_type = 'FOREIGN KEY'`, fk.table, fk.name).Scan(&n).Error; err != nil {
			return err
		}
		if n == 0 {
			if err := db.Exec(fk.sql()).Error; err != nil {
				log.Printf("platforms: could not add foreign key %s: %v", fk.name, err)
			}
		}
	}
	if created {
		log.Println("created platform tables")
		return db.Exec(seedPlatforms).Error
	}
	return nil
}
