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
	// Every PIN stored before pin_upgrade_required existed is a 4-digit one (see migration 000008).
	backfillPINUpgrade := db.Migrator().HasTable(&models.User{}) &&
		!db.Migrator().HasColumn(&models.User{}, "pin_upgrade_required")
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
		&models.OpeningBalance{},
		&models.CashReconciliation{},
		&models.TransactionApproval{},
		&models.AppSetting{},
	)
	if err != nil || !backfillPINUpgrade {
		return err
	}
	return db.Exec("UPDATE users SET pin_upgrade_required = TRUE").Error
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
