package database

import (
	"log"
	"regexp"

	"gorm.io/gorm"
)

// Platform tables (HQ = branch): platforms, partners, branch assignments and their partners,
// weekly win/lose results with each partner's share, and weekly settlements. Same SQL as migrations/000013_platforms.up.sql (a test keeps them in step).
// Adds what is missing: tables, columns, keys, then foreign keys. The only drops are the unique
// keys replaced when an agent could be in several groups and a week have several entries
// (oldPlatformKeys).
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
	`CREATE TABLE IF NOT EXISTS partners (
	id CHAR(36) NOT NULL,
	name VARCHAR(128) NOT NULL,
	phone VARCHAR(32) NOT NULL DEFAULT '',
	note VARCHAR(255) NOT NULL DEFAULT '',
	default_rate DECIMAL(7,4) NOT NULL DEFAULT 0,
	is_active TINYINT(1) NOT NULL DEFAULT 1,
	created_at DATETIME(3) NULL,
	updated_at DATETIME(3) NULL,
	deleted_at DATETIME(3) NULL,
	PRIMARY KEY (id),
	UNIQUE KEY uq_partners_name (name),
	KEY idx_partners_deleted_at (deleted_at)
)`,
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
	`CREATE TABLE IF NOT EXISTS assignment_groups (
	id CHAR(36) NOT NULL,
	branch_id CHAR(36) NOT NULL,
	title VARCHAR(128) NOT NULL,
	currency VARCHAR(3) NOT NULL DEFAULT '',
	starts_on DATE NOT NULL,
	is_active TINYINT(1) NOT NULL DEFAULT 1,
	created_by_id CHAR(36) NOT NULL,
	created_at DATETIME(3) NULL,
	updated_at DATETIME(3) NULL,
	PRIMARY KEY (id),
	KEY idx_assignment_groups_branch (branch_id)
)`,
	`CREATE TABLE IF NOT EXISTS hq_platform_assignments (
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
	end_reason VARCHAR(255) NOT NULL DEFAULT '',
	created_at DATETIME(3) NULL,
	updated_at DATETIME(3) NULL,
	deleted_at DATETIME(3) NULL,
	PRIMARY KEY (id),
	UNIQUE KEY uq_hq_platform_group_active (group_id, platform_id, active_key),
	KEY idx_hq_platform_assignments_branch (branch_id),
	KEY idx_hq_platform_assignments_platform (platform_id),
	KEY idx_hq_platform_assignments_group (group_id),
	KEY idx_hq_platform_assignments_deleted_at (deleted_at)
)`,
	`CREATE TABLE IF NOT EXISTS assignment_partners (
	id CHAR(36) NOT NULL,
	assignment_id CHAR(36) NOT NULL,
	partner_id CHAR(36) NOT NULL,
	commission_rate DECIMAL(7,4) NOT NULL,
	created_at DATETIME(3) NULL,
	updated_at DATETIME(3) NULL,
	PRIMARY KEY (id),
	UNIQUE KEY uq_assignment_partner (assignment_id, partner_id),
	KEY idx_assignment_partners_partner (partner_id)
)`,
	`CREATE TABLE IF NOT EXISTS weekly_platform_results (
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
)`,
	`CREATE TABLE IF NOT EXISTS weekly_partner_results (
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
)`,
	`CREATE TABLE IF NOT EXISTS weekly_group_results (
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
)`,
	`CREATE TABLE IF NOT EXISTS weekly_entries (
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

// platformColumns were added to a table after it first shipped: added when missing.
var platformColumns = []struct{ table, column, ddl string }{
	{"hq_platform_assignments", "group_id", "ALTER TABLE hq_platform_assignments ADD COLUMN group_id CHAR(36) NULL AFTER platform_id, ADD KEY idx_hq_platform_assignments_group (group_id)"},
	{"weekly_platform_results", "entry_id", "ALTER TABLE weekly_platform_results ADD COLUMN entry_id CHAR(36) NULL AFTER id"},
}

// platformNullable are columns that may be empty since they first shipped: changed when NOT NULL.
// A result saved on an entry has no assignment of its own (its groups are in weekly_group_results).
var platformNullable = []struct{ table, column, ddl string }{
	{"weekly_platform_results", "assignment_id", "ALTER TABLE weekly_platform_results MODIFY assignment_id CHAR(36) NULL"},
}

// The weeks saved before entries could hold any agent: each result goes on its week's entry (one per
// group and week then), and its Total in its assignment's group (none: an assignment made before
// groups). Run once, when the column or the table is added.
const fillResultEntries = `UPDATE weekly_platform_results r JOIN weekly_entries e ON e.branch_id = r.branch_id AND e.week_start = r.week_start
	SET r.entry_id = e.id WHERE r.entry_id IS NULL`

const fillGroupResults = `INSERT IGNORE INTO weekly_group_results (id, result_id, group_id, assignment_id, commission_rate, jame_ps, created_at, updated_at)
	SELECT UUID(), r.id, a.group_id, a.id, r.commission_rate, r.jame_ps, NOW(3), NOW(3)
	FROM weekly_platform_results r JOIN hq_platform_assignments a ON a.id = r.assignment_id`

// platformKeys were added after the tables first shipped: added when missing. An agent can be in
// several groups of a branch, so an active assignment is one per group (not per branch); a week
// can have several entries, so a result is one per entry and agent (not per branch, week and agent).
var platformKeys = []struct{ table, name, ddl string }{
	// branch_id's foreign keys used the old unique keys, so they get keys of their own first.
	{"hq_platform_assignments", "idx_hq_platform_assignments_branch", "ALTER TABLE hq_platform_assignments ADD KEY idx_hq_platform_assignments_branch (branch_id)"},
	{"hq_platform_assignments", "uq_hq_platform_group_active", "ALTER TABLE hq_platform_assignments ADD UNIQUE KEY uq_hq_platform_group_active (group_id, platform_id, active_key)"},
	{"weekly_entries", "idx_weekly_entries_branch_week", "ALTER TABLE weekly_entries ADD KEY idx_weekly_entries_branch_week (branch_id, week_start)"},
	{"weekly_platform_results", "uq_weekly_result_entry", "ALTER TABLE weekly_platform_results ADD UNIQUE KEY uq_weekly_result_entry (entry_id, platform_id)"},
}

// oldPlatformKey is a unique key replaced by one in platformKeys: dropped when there.
type oldPlatformKey struct{ table, name string }

var oldPlatformKeys = []oldPlatformKey{
	{"hq_platform_assignments", "uq_hq_platform_active"},
	{"weekly_platform_results", "uq_weekly_platform_result"},
	{"weekly_platform_results", "uq_weekly_result_assignment"},
	{"weekly_entries", "uq_weekly_entry"},
}

func (k oldPlatformKey) sql() string {
	return "ALTER TABLE " + k.table + " DROP INDEX " + k.name
}

func hasIndex(db *gorm.DB, table, name string) (bool, error) {
	var n int64
	err := db.Raw(`SELECT COUNT(*) FROM information_schema.statistics
		WHERE table_schema = DATABASE() AND table_name = ? AND index_name = ?`, table, name).Scan(&n).Error
	return n > 0, err
}

type foreignKey struct{ table, name, columns, ref string }

var platformForeignKeys = []foreignKey{
	{"assignment_groups", "fk_assignment_groups_branch", "branch_id", "branches(id)"},
	{"assignment_groups", "fk_assignment_groups_created_by", "created_by_id", "users(id)"},
	{"hq_platform_assignments", "fk_hq_platform_assignments_group", "group_id", "assignment_groups(id)"},
	{"hq_platform_assignments", "fk_hq_platform_assignments_branch", "branch_id", "branches(id)"},
	{"hq_platform_assignments", "fk_hq_platform_assignments_platform", "platform_id", "platforms(id)"},
	{"hq_platform_assignments", "fk_hq_platform_assignments_created_by", "created_by_id", "users(id)"},
	{"assignment_partners", "fk_assignment_partners_assignment", "assignment_id", "hq_platform_assignments(id) ON DELETE CASCADE"},
	{"assignment_partners", "fk_assignment_partners_partner", "partner_id", "partners(id)"},
	{"weekly_platform_results", "fk_weekly_platform_results_branch", "branch_id", "branches(id)"},
	{"weekly_platform_results", "fk_weekly_platform_results_platform", "platform_id", "platforms(id)"},
	{"weekly_platform_results", "fk_weekly_platform_results_assignment", "assignment_id", "hq_platform_assignments(id)"},
	{"weekly_platform_results", "fk_weekly_platform_results_created_by", "created_by_id", "users(id)"},
	{"weekly_partner_results", "fk_weekly_partner_results_result", "result_id", "weekly_platform_results(id) ON DELETE CASCADE"},
	{"weekly_partner_results", "fk_weekly_partner_results_partner", "partner_id", "partners(id)"},
	{"weekly_platform_results", "fk_weekly_platform_results_entry", "entry_id", "weekly_entries(id)"},
	{"weekly_group_results", "fk_weekly_group_results_result", "result_id", "weekly_platform_results(id) ON DELETE CASCADE"},
	{"weekly_group_results", "fk_weekly_group_results_group", "group_id", "assignment_groups(id)"},
	{"weekly_group_results", "fk_weekly_group_results_assignment", "assignment_id", "hq_platform_assignments(id)"},
	{"weekly_entries", "fk_weekly_entries_branch", "branch_id", "branches(id)"},
	{"weekly_entries", "fk_weekly_entries_created_by", "created_by_id", "users(id)"},
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
	linksCreated := !db.Migrator().HasTable("assignment_partners")
	sharesCreated := !db.Migrator().HasTable("weekly_partner_results")
	entriesCreated := !db.Migrator().HasTable("weekly_entries")
	groupsCreated := !db.Migrator().HasTable("assignment_groups")
	resultsBefore := db.Migrator().HasTable("weekly_platform_results")
	groupResultsCreated := !db.Migrator().HasTable("weekly_group_results")
	options := tableOptions(branchCollation(db))
	for _, stmt := range platformTables {
		if err := db.Exec(stmt + options).Error; err != nil {
			return err
		}
	}
	entryColumnAdded := false
	for _, c := range platformColumns {
		if !db.Migrator().HasColumn(c.table, c.column) {
			if err := db.Exec(c.ddl).Error; err != nil {
				return err
			}
			entryColumnAdded = entryColumnAdded || c.column == "entry_id"
		}
	}
	for _, c := range platformNullable {
		var nullable string
		if err := db.Raw(`SELECT is_nullable FROM information_schema.columns
			WHERE table_schema = DATABASE() AND table_name = ? AND column_name = ?`, c.table, c.column).Scan(&nullable).Error; err != nil {
			return err
		}
		if nullable == "NO" {
			if err := db.Exec(c.ddl).Error; err != nil {
				return err
			}
		}
	}
	// Assignments made before groups each become a group of their own, titled with the agent's
	// code, once, when the groups table is created. Ended ones stay as they are (history).
	if groupsCreated && !created {
		if err := db.Exec(`INSERT IGNORE INTO assignment_groups (id, branch_id, title, currency, starts_on, is_active, created_by_id, created_at, updated_at)
			SELECT a.id, a.branch_id, p.code, a.currency, a.starts_on, 1, a.created_by_id, a.created_at, NOW(3)
			FROM hq_platform_assignments a JOIN platforms p ON p.id = a.platform_id
			WHERE a.active_key = 1 AND a.group_id IS NULL AND a.deleted_at IS NULL`).Error; err != nil {
			return err
		}
		if err := db.Exec(`UPDATE hq_platform_assignments SET group_id = id WHERE active_key = 1 AND group_id IS NULL AND deleted_at IS NULL`).Error; err != nil {
			return err
		}
	}
	// Weeks saved before entries had titles get one entry each, once, when the table is created.
	if entriesCreated && !created {
		if err := db.Exec(`INSERT IGNORE INTO weekly_entries (id, branch_id, week_start, title, created_by_id, created_at, updated_at)
			SELECT UUID(), branch_id, week_start, CONCAT('Week of ', DATE_FORMAT(week_start, '%d %b %Y')), MIN(created_by_id), MIN(created_at), NOW(3)
			FROM weekly_platform_results WHERE deleted_at IS NULL GROUP BY branch_id, week_start`).Error; err != nil {
			return err
		}
	}
	if entryColumnAdded {
		if err := db.Exec(fillResultEntries).Error; err != nil {
			return err
		}
	}
	if groupResultsCreated && resultsBefore {
		if err := db.Exec(fillGroupResults).Error; err != nil {
			return err
		}
	}
	// The new keys first, then the old ones go. Rows that met the old keys meet the new ones (a
	// group is in one branch; an entry is one branch and week). A key that can't be changed is
	// logged and skipped, so the server still starts.
	for _, k := range platformKeys {
		has, err := hasIndex(db, k.table, k.name)
		if err != nil {
			return err
		}
		if !has {
			log.Printf("platforms: adding key %s", k.name)
			if err := db.Exec(k.ddl).Error; err != nil {
				log.Printf("platforms: could not add key %s: %v", k.name, err)
			}
		}
	}
	for _, k := range oldPlatformKeys {
		has, err := hasIndex(db, k.table, k.name)
		if err != nil {
			return err
		}
		if has {
			log.Printf("platforms: dropping key %s", k.name)
			if err := db.Exec(k.sql()).Error; err != nil {
				log.Printf("platforms: could not drop key %s: %v", k.name, err)
			}
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
	// An earlier test build kept one partner on the assignment and the result (partner_id): move
	// those into the partner tables once, when the tables are created.
	if linksCreated && db.Migrator().HasColumn("hq_platform_assignments", "partner_id") {
		if err := db.Exec(`INSERT IGNORE INTO assignment_partners (id, assignment_id, partner_id, commission_rate, created_at, updated_at)
			SELECT UUID(), id, partner_id, commission_rate, NOW(3), NOW(3) FROM hq_platform_assignments WHERE partner_id IS NOT NULL`).Error; err != nil {
			return err
		}
	}
	if sharesCreated && db.Migrator().HasColumn("weekly_platform_results", "partner_id") {
		if err := db.Exec(`INSERT IGNORE INTO weekly_partner_results (id, result_id, partner_id, commission_rate, jame_ps, created_at, updated_at)
			SELECT UUID(), id, partner_id, commission_rate, jame_ps, NOW(3), NOW(3) FROM weekly_platform_results WHERE partner_id IS NOT NULL`).Error; err != nil {
			return err
		}
	}
	if created {
		log.Println("created platform tables")
		return db.Exec(seedPlatforms).Error
	}
	return nil
}
