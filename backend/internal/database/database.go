package database

import (
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/cambodia-finance/income-expense/internal/config"
	"github.com/cambodia-finance/income-expense/internal/models"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// newLogger logs only slow queries and real errors unless DB_LOG_SQL is on. "Record not found" is
// an ordinary answer in this code (seeds, logins, refresh tokens), and colour codes are noise in
// hosted log viewers.
func newLogger(cfg *config.Config, out io.Writer) logger.Interface {
	level := logger.Warn
	if cfg.LogSQL {
		level = logger.Info
	}
	return logger.New(log.New(out, "", log.LstdFlags), logger.Config{
		SlowThreshold:             200 * time.Millisecond,
		LogLevel:                  level,
		IgnoreRecordNotFoundError: true,
	})
}

func Connect(cfg *config.Config) (*gorm.DB, error) {
	db, err := gorm.Open(mysql.Open(cfg.DSN()), &gorm.Config{
		Logger: newLogger(cfg, os.Stdout),
		NowFunc: func() time.Time {
			loc, _ := time.LoadLocation(cfg.Timezone)
			return time.Now().In(loc)
		},
	})
	if err != nil {
		return nil, fmt.Errorf("connect database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("get sql db: %w", err)
	}
	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetConnMaxLifetime(time.Hour)

	return db, nil
}

// AutoMigrate is used only for development / first bootstrap.
// Production must use versioned SQL migrations.
func AutoMigrate(db *gorm.DB) error {
	err := db.AutoMigrate(
		&models.User{},
		&models.Role{},
		&models.Permission{},
		&models.Branch{},
		&models.UserRole{},
		&models.RolePermission{},
		&models.UserBranch{},
		&models.UserManager{},
		&models.Session{},
		&models.AuditLog{},
		&models.Setting{},
		&models.Category{},
		&models.PaymentMethod{},
		&models.TransactionNumberSeq{},
		&models.Transaction{},
		&models.TransactionAttachment{},
		&models.OpeningBalance{},
		&models.CashReconciliation{},
		&models.TransactionApproval{},
		&models.AppSetting{},
	)
	if err != nil {
		return err
	}
	if err := db.AutoMigrate(models.InventoryModels()...); err != nil {
		return err
	}
	// Platform tables come from SQL (with their foreign keys), not from the structs.
	if err := EnsurePlatformTables(db); err != nil {
		return err
	}
	// 4-digit PINs are valid again (PINs are 4 to 6 digits), so nobody has to replace theirs;
	// drop the flag that sent those users to the profile page (see migration 000009).
	if db.Migrator().HasColumn(&models.User{}, "pin_upgrade_required") {
		if err := db.Migrator().DropColumn(&models.User{}, "pin_upgrade_required"); err != nil {
			return err
		}
	}
	return migrateLegacyAttachments(db)
}

func Ping(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Ping()
}

func MustConnect(cfg *config.Config) *gorm.DB {
	db, err := Connect(cfg)
	if err != nil {
		log.Fatalf("database connection failed: %v", err)
	}
	return db
}

const createAttachmentsTable = `CREATE TABLE IF NOT EXISTS transaction_attachments (
	id CHAR(36) NOT NULL,
	transaction_id CHAR(36) NOT NULL,
	storage_key VARCHAR(255) NOT NULL,
	name VARCHAR(255) NOT NULL,
	content_type VARCHAR(128) NOT NULL,
	size BIGINT NOT NULL DEFAULT 0,
	stage VARCHAR(16) NOT NULL DEFAULT 'submitted',
	created_by_id CHAR(36) NOT NULL,
	created_at DATETIME(3) NULL,
	PRIMARY KEY (id),
	KEY idx_transaction_attachments_transaction_id (transaction_id)
)`

// EnsureRequiredTables creates tables added after the first release if they are missing, and moves
// data into them. It only ever adds (never alters or drops), so it is safe on every start, with or
// without AUTO_MIGRATE or the SQL migrations.
func EnsureRequiredTables(db *gorm.DB) error {
	// A brand-new database gets every table from AutoMigrate or the SQL migrations instead.
	if !db.Migrator().HasTable(&models.Transaction{}) {
		return nil
	}
	if !db.Migrator().HasTable(&models.TransactionAttachment{}) {
		log.Println("creating missing table transaction_attachments")
		// Plain SQL with no foreign key: a key that can't be created (e.g. a collation that differs
		// from the transactions table) must not stop the server from starting. AutoMigrate adds it
		// later when it can.
		if err := db.Exec(createAttachmentsTable).Error; err != nil {
			return err
		}
	}
	if !db.Migrator().HasColumn(&models.TransactionAttachment{}, "stage") {
		log.Println("adding column transaction_attachments.stage")
		if err := db.Exec("ALTER TABLE transaction_attachments ADD COLUMN stage VARCHAR(16) NOT NULL DEFAULT 'submitted'").Error; err != nil {
			return err
		}
	}
	if err := ensureRoleDataScope(db); err != nil {
		return err
	}
	if err := ownBranchesOnly(db); err != nil {
		return err
	}
	if err := ensureInventoryTables(db); err != nil {
		return err
	}
	if err := EnsurePlatformTables(db); err != nil {
		return err
	}
	return migrateLegacyAttachments(db)
}

// ensureRoleDataScope adds roles.data_scope (see migration 000012).
func ensureRoleDataScope(db *gorm.DB) error {
	if !db.Migrator().HasTable(&models.Role{}) {
		return nil
	}
	if !db.Migrator().HasColumn(&models.Role{}, "data_scope") {
		log.Println("adding column roles.data_scope")
		if err := db.Exec("ALTER TABLE roles ADD COLUMN data_scope VARCHAR(16) NOT NULL DEFAULT 'assigned'").Error; err != nil {
			return err
		}
	}
	// Roles for one branch (nil = shared).
	if !db.Migrator().HasColumn(&models.Role{}, "branch_id") {
		log.Println("adding column roles.branch_id")
		if err := db.Exec("ALTER TABLE roles ADD COLUMN branch_id CHAR(36) NULL, ADD INDEX idx_roles_branch_id (branch_id)").Error; err != nil {
			return err
		}
	}
	return nil
}

// ownBranchesOnly: data is never shared between branches, so every role sees only its users'
// own branches. Roles with the retired scopes ("custom" with picked branches, "all") become
// "assigned" and their picked branches are cleared. Safe on every start and after a restore.
func ownBranchesOnly(db *gorm.DB) error {
	if !db.Migrator().HasTable(&models.Role{}) || !db.Migrator().HasColumn(&models.Role{}, "data_scope") {
		return nil
	}
	res := db.Model(&models.Role{}).Where("data_scope <> ?", models.DataScopeAssigned).Update("data_scope", models.DataScopeAssigned)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected > 0 {
		log.Printf("%d role(s) now see only their users' own branches", res.RowsAffected)
	}
	if db.Migrator().HasTable("role_branches") {
		if err := db.Exec("DELETE FROM role_branches").Error; err != nil {
			return err
		}
	}
	return nil
}

// ensureInventoryTables creates the inventory tables, or brings them up to date, when a table or
// a newer column is missing (see models/inventory.go; they have no foreign keys to older tables,
// so this can't fail on an older table's collation). It only adds.
func ensureInventoryTables(db *gorm.DB) error {
	if !inventoryUpToDate(db) {
		log.Println("updating inventory tables")
		if err := db.AutoMigrate(models.InventoryModels()...); err != nil {
			return err
		}
	}
	// The current KHR-per-USD rate (see services.CurrencyService), for rows saved before rates.
	const rate = "(SELECT COALESCE(MAX(CAST(`value` AS DECIMAL(19,4))), 4100) FROM settings WHERE `key` = 'currency.rate_khr')"
	// Rows from before discounts, shipping and landed cost: their totals and costs stay as they were.
	// Then rows from before USD-based inventory: KHR-priced products move to USD, and documents get
	// their rate and USD amounts (a KHR purchase's landed cost is converted once, while its USD
	// total is still unset).
	for _, q := range []string{
		"UPDATE purchase_items SET landed_cost = unit_cost WHERE landed_cost = 0 AND unit_cost <> 0",
		"UPDATE purchases SET subtotal = total WHERE subtotal = 0 AND total <> 0 AND discount = 0 AND shipping = 0",
		"UPDATE sales SET subtotal = total WHERE subtotal = 0 AND total <> 0 AND discount = 0 AND shipping = 0",
		"UPDATE sale_items SET net_total = line_total WHERE net_total = 0 AND line_total <> 0",
		"UPDATE stock_levels sl JOIN products p ON p.id = sl.product_id SET sl.avg_cost = ROUND(sl.avg_cost / " + rate + ", 4) WHERE p.currency = 'KHR'",
		"UPDATE products SET sale_price = ROUND(sale_price / " + rate + ", 4), cost_price = ROUND(cost_price / " + rate + ", 4), currency = 'USD' WHERE currency = 'KHR'",
		"UPDATE purchases SET exchange_rate = " + rate + " WHERE exchange_rate = 0",
		"UPDATE sales SET exchange_rate = " + rate + " WHERE exchange_rate = 0",
		"UPDATE inventory_expenses SET exchange_rate = " + rate + " WHERE exchange_rate = 0",
		"UPDATE purchase_items pi JOIN purchases p ON p.id = pi.purchase_id SET pi.landed_cost = ROUND(pi.landed_cost / p.exchange_rate, 4) WHERE p.currency = 'KHR' AND p.total_base = 0 AND p.total <> 0",
		"UPDATE purchases SET total_base = CASE WHEN currency = 'KHR' THEN ROUND(total / exchange_rate, 4) ELSE total END WHERE total_base = 0 AND total <> 0",
		"UPDATE sales SET total_base = CASE WHEN currency = 'KHR' THEN ROUND(total / exchange_rate, 4) ELSE total END WHERE total_base = 0 AND total <> 0",
		"UPDATE sale_items si JOIN sales s ON s.id = si.sale_id SET si.net_total_base = CASE WHEN s.currency = 'KHR' THEN ROUND(si.net_total / s.exchange_rate, 4) ELSE si.net_total END WHERE si.net_total_base = 0 AND si.net_total <> 0",
		"UPDATE inventory_expenses SET amount_base = CASE WHEN currency = 'KHR' THEN ROUND(amount / exchange_rate, 4) ELSE amount END WHERE amount_base = 0 AND amount <> 0",
		// A document's rate is units of its currency per 1 USD, so USD documents have rate 1.
		"UPDATE purchases SET exchange_rate = 1 WHERE currency = 'USD' AND exchange_rate <> 1",
		"UPDATE sales SET exchange_rate = 1 WHERE currency = 'USD' AND exchange_rate <> 1",
		"UPDATE inventory_expenses SET exchange_rate = 1 WHERE currency = 'USD' AND exchange_rate <> 1",
	} {
		if err := db.Exec(q).Error; err != nil {
			return err
		}
	}
	return nil
}

func inventoryUpToDate(db *gorm.DB) bool {
	for _, m := range models.InventoryModels() {
		if !db.Migrator().HasTable(m) {
			return false
		}
	}
	for m, cols := range models.InventoryColumns {
		for _, col := range cols {
			if !db.Migrator().HasColumn(m, col) {
				return false
			}
		}
	}
	return true
}

// migrateLegacyAttachments moves each transaction's old single attachment into
// transaction_attachments (see migration 000010). Rows it has moved are cleared, so it is safe on every start.
func migrateLegacyAttachments(db *gorm.DB) error {
	var legacy []models.Transaction
	if err := db.Select("id", "attachment_key", "attachment_name", "attachment_type", "attachment_size", "created_by_id", "updated_at").
		Where("attachment_key <> ''").Find(&legacy).Error; err != nil {
		return err
	}
	for _, t := range legacy {
		err := db.Transaction(func(tx *gorm.DB) error {
			if err := tx.Create(&models.TransactionAttachment{
				TransactionID: t.ID, StorageKey: t.AttachmentKey, Name: t.AttachmentName,
				ContentType: t.AttachmentType, Size: t.AttachmentSize, CreatedByID: t.CreatedByID, CreatedAt: t.UpdatedAt,
			}).Error; err != nil {
				return err
			}
			return tx.Model(&models.Transaction{}).Where("id = ?", t.ID).Updates(map[string]interface{}{
				"attachment_key": "", "attachment_name": "", "attachment_type": "", "attachment_size": 0,
			}).Error
		})
		if err != nil {
			return err
		}
	}
	return nil
}
