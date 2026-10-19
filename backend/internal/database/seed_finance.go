package database

import (
	"fmt"
	"log"

	"github.com/cambodia-finance/income-expense/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func SeedCategoriesAndPaymentMethods(db *gorm.DB) error {
	incomeCats := []struct{ Code, Name string }{
		{"sales", "Sales"},
		{"services", "Services"},
		{"rental", "Rental Income"},
		{"other_income", "Other Income"},
	}
	expenseCats := []struct{ Code, Name string }{
		{"rent", "Rent"},
		{"salaries", "Salaries"},
		{"utilities", "Utilities"},
		{"inventory", "Inventory"},
		{"transportation", "Transportation"},
		{"marketing", "Marketing"},
		{"maintenance", "Maintenance"},
		{"miscellaneous", "Miscellaneous"},
	}
	methods := []struct{ Code, Name string }{
		{"cash", "Cash"},
		{"bank_transfer", "Bank Transfer"},
		{"khqr", "KHQR"},
		{"other", "Other"},
	}

	for _, c := range incomeCats {
		if err := upsertCategory(db, c.Code, c.Name, models.TxnTypeIncome); err != nil {
			return err
		}
	}
	for _, c := range expenseCats {
		if err := upsertCategory(db, c.Code, c.Name, models.TxnTypeExpense); err != nil {
			return err
		}
	}
	for _, m := range methods {
		if err := upsertPaymentMethod(db, m.Code, m.Name); err != nil {
			return err
		}
	}
	log.Println("seeded categories and payment methods")
	return nil
}

func upsertCategory(db *gorm.DB, code, name, typ string) error {
	var existing models.Category
	err := db.Where("code = ? AND type = ?", code, typ).First(&existing).Error
	if err == gorm.ErrRecordNotFound {
		cat := models.Category{
			Base:     models.Base{ID: uuid.New()},
			Code:     code,
			Name:     name,
			Type:     typ,
			IsActive: true,
			IsSystem: true,
		}
		if err := db.Create(&cat).Error; err != nil {
			return fmt.Errorf("create category %s/%s: %w", typ, code, err)
		}
		return nil
	}
	return err
}

func upsertPaymentMethod(db *gorm.DB, code, name string) error {
	var existing models.PaymentMethod
	err := db.Where("code = ?", code).First(&existing).Error
	if err == gorm.ErrRecordNotFound {
		pm := models.PaymentMethod{
			Base:     models.Base{ID: uuid.New()},
			Code:     code,
			Name:     name,
			IsActive: true,
			IsSystem: true,
		}
		if err := db.Create(&pm).Error; err != nil {
			return fmt.Errorf("create payment method %s: %w", code, err)
		}
		return nil
	}
	return err
}
