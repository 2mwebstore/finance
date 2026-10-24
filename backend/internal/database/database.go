package database

import (
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/cambodia-finance/income-expense/internal/config"
	"github.com/cambodia-finance/income-expense/internal/models"
	"github.com/cambodia-finance/income-expense/internal/permissions"
	"github.com/google/uuid"
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
	if err := retireAllDataScope(db); err != nil {
		return err
	}
	return migrateLegacyAttachments(db)
}

// ensureRoleDataScope adds roles.data_scope and the role_branches table (see migration 000012).
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
	if !db.Migrator().HasTable("role_branches") {
		log.Println("creating missing table role_branches")
		return db.Exec(`CREATE TABLE IF NOT EXISTS role_branches (
	role_id CHAR(36) NOT NULL,
	branch_id CHAR(36) NOT NULL,
	PRIMARY KEY (role_id, branch_id),
	KEY idx_role_branches_branch_id (branch_id)
)`).Error
	}
	return nil
}

// retireAllDataScope: "All branches" is no longer a data scope; seeing every branch comes from full
// access (Manage Branches) only. Roles that still have it keep what their users see: with Manage
// Branches they become "assigned" (they see every branch anyway), the others "custom" with every
// branch there is now. Changed roles no longer match, so it is safe on every start and after a
// restore of an older backup.
func retireAllDataScope(db *gorm.DB) error {
	if !db.Migrator().HasTable(&models.Role{}) || !db.Migrator().HasColumn(&models.Role{}, "data_scope") {
		return nil
	}
	var roles []models.Role
	if err := db.Where("data_scope = ?", models.DataScopeAll).Find(&roles).Error; err != nil {
		return err
	}
	if len(roles) == 0 {
		return nil
	}
	var fullRoleIDs []uuid.UUID
	if err := db.Table("role_permissions").
		Joins("JOIN permissions ON permissions.id = role_permissions.permission_id").
		Where("permissions.code = ?", permissions.BranchesManage).
		Pluck("role_permissions.role_id", &fullRoleIDs).Error; err != nil {
		return err
	}
	full := map[uuid.UUID]bool{}
	for _, id := range fullRoleIDs {
		full[id] = true
	}
	var branches []models.Branch
	if err := db.Find(&branches).Error; err != nil {
		return err
	}
	for i := range roles {
		role := &roles[i]
		scope := models.DataScopeCustom
		if full[role.ID] || len(branches) == 0 {
			scope = models.DataScopeAssigned
		}
		err := db.Transaction(func(tx *gorm.DB) error {
			if scope == models.DataScopeCustom {
				if err := tx.Model(role).Association("Branches").Replace(branches); err != nil {
					return err
				}
			}
			return tx.Model(&models.Role{}).Where("id = ?", role.ID).Update("data_scope", scope).Error
		})
		if err != nil {
			return fmt.Errorf("data scope of role %s: %w", role.Code, err)
		}
		log.Printf("role %s: data scope \"all\" is now %q", role.Code, scope)
	}
	return nil
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
