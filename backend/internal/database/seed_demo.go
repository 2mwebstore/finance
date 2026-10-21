package database

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/cambodia-finance/income-expense/internal/auth"
	"github.com/cambodia-finance/income-expense/internal/models"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// SeedDemoData loads a full Cambodia demo dataset when SEED_DEMO=true
// (or when APP_ENV=development and no branches exist yet).
// Idempotent: skips existing demo records unless SEED_DEMO=true.
//
// Demo logins (PIN for all non-admin demo users: 246810)
//
//	manager1 / 246810     — Owner @ Phnom Penh
//	cashier1 / 246810     — Cashier @ Phnom Penh
//	accountant1 / 246810  — Accountant @ all branches
//	viewer1 / 246810      — Viewer @ Phnom Penh
//	manager_sr / 246810   — Manager @ Siem Reap
//
// Super admin is created via /auth/setup (not by this seed).
func SeedDemoData(db *gorm.DB) error {
	force := os.Getenv("SEED_DEMO") == "true" || os.Getenv("SEED_DEMO") == "1"

	var existing int64
	if err := db.Model(&models.Branch{}).Where("code = ?", "PP-01").Count(&existing).Error; err != nil {
		return err
	}
	if existing > 0 && !force {
		log.Println("demo seed: PP-01 exists, syncing owner role")
		return syncDemoOwnerRole(db)
	}
	if existing > 0 {
		log.Println("demo seed: data already present, ensuring optional records only")
	}

	return db.Transaction(func(tx *gorm.DB) error {
		branches, err := seedDemoBranches(tx)
		if err != nil {
			return err
		}
		users, err := seedDemoUsers(tx, branches, force)
		if err != nil {
			return err
		}
		if err := seedDemoOpeningBalances(tx, branches, users["manager1"]); err != nil {
			return err
		}
		if err := seedDemoTransactions(tx, branches, users); err != nil {
			return err
		}
		if err := seedDemoReconciliation(tx, branches["PP-01"], users["manager1"]); err != nil {
			return err
		}
		log.Println("demo seed: complete (branches, users, opening balances, transactions, reconciliation)")
		log.Println("demo seed: logins → manager1, cashier1, accountant1, viewer1, manager_sr / 246810")
		return nil
	})
}

func syncDemoOwnerRole(db *gorm.DB) error {
	return db.Transaction(func(tx *gorm.DB) error {
		var user models.User
		if err := tx.Where("username = ?", "manager1").First(&user).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				log.Println("demo seed: manager1 not found; owner role will be assigned on full seed")
				return nil
			}
			return err
		}

		var role models.Role
		if err := tx.Where("code = ?", "owner").First(&role).Error; err != nil {
			return fmt.Errorf("owner role: %w", err)
		}
		if err := tx.Model(&user).Association("Roles").Replace([]models.Role{role}); err != nil {
			return fmt.Errorf("assign owner role to manager1: %w", err)
		}
		return nil
	})
}

func seedDemoBranches(tx *gorm.DB) (map[string]models.Branch, error) {
	defs := []struct {
		Code, Name, Address, Phone string
	}{
		{"PP-01", "Phnom Penh HQ", "Street 271, Toul Kork, Phnom Penh", "+855 23 000 001"},
		{"SR-01", "Siem Reap", "Sivatha Blvd, Siem Reap", "+855 63 000 002"},
		{"BT-01", "Battambang", "National Road 5, Battambang", "+855 53 000 003"},
	}
	out := map[string]models.Branch{}
	for _, d := range defs {
		var b models.Branch
		err := tx.Where("code = ?", d.Code).First(&b).Error
		if err == gorm.ErrRecordNotFound {
			b = models.Branch{
				Base:     models.Base{ID: uuid.New()},
				Code:     d.Code,
				Name:     d.Name,
				Address:  d.Address,
				Phone:    d.Phone,
				IsActive: true,
			}
			if err := tx.Create(&b).Error; err != nil {
				return nil, fmt.Errorf("branch %s: %w", d.Code, err)
			}
		} else if err != nil {
			return nil, err
		}
		out[d.Code] = b
	}
	return out, nil
}

func seedDemoUsers(tx *gorm.DB, branches map[string]models.Branch, resetPIN bool) (map[string]models.User, error) {
	type def struct {
		Username, Email, FullName, RoleCode string
		BranchCodes                         []string
	}
	defs := []def{
		{"manager1", "manager.pp@demo.local", "Sokha Owner", "owner", []string{"PP-01"}},
		{"cashier1", "cashier.pp@demo.local", "Dara Cashier", "cashier", []string{"PP-01"}},
		{"accountant1", "accountant@demo.local", "Sreymom Accountant", "accountant", []string{"PP-01", "SR-01", "BT-01"}},
		{"viewer1", "viewer@demo.local", "Vannak Viewer", "viewer", []string{"PP-01"}},
		{"manager_sr", "manager.sr@demo.local", "Rithy SR Manager", "manager", []string{"SR-01"}},
	}

	hash, err := auth.HashPassword("246810")
	if err != nil {
		return nil, err
	}

	out := map[string]models.User{}
	for _, d := range defs {
		var u models.User
		err := tx.Where("username = ?", d.Username).First(&u).Error
		if err == gorm.ErrRecordNotFound {
			u = models.User{
				Base:         models.Base{ID: uuid.New()},
				Username:     d.Username,
				Email:        d.Email,
				PasswordHash: hash,
				FullName:     d.FullName,
				IsActive:     true,
			}
			if err := tx.Create(&u).Error; err != nil {
				return nil, fmt.Errorf("user %s: %w", d.Username, err)
			}
		} else if err != nil {
			return nil, err
		} else if resetPIN {
			// A forced demo seed restores the documented demo PIN for existing
			// accounts too, including databases seeded before PINs became 6 digits.
			if err := tx.Model(&u).Updates(map[string]interface{}{
				"password_hash":        hash,
				"pin_upgrade_required": false,
			}).Error; err != nil {
				return nil, fmt.Errorf("reset demo PIN for %s: %w", d.Username, err)
			}
		}

		var role models.Role
		if err := tx.Where("code = ?", d.RoleCode).First(&role).Error; err != nil {
			return nil, fmt.Errorf("role %s: %w", d.RoleCode, err)
		}
		_ = tx.Model(&u).Association("Roles").Replace([]models.Role{role})

		var blist []models.Branch
		for _, bc := range d.BranchCodes {
			if b, ok := branches[bc]; ok {
				blist = append(blist, b)
			}
		}
		_ = tx.Model(&u).Association("Branches").Replace(blist)
		if len(blist) > 0 {
			tx.Model(&u).Update("default_branch_id", blist[0].ID)
		}
		out[d.Username] = u
	}
	return out, nil
}

func seedDemoOpeningBalances(tx *gorm.DB, branches map[string]models.Branch, actor models.User) error {
	loc := time.FixedZone("Asia/Phnom_Penh", 7*3600)
	date := time.Date(2026, 10, 1, 0, 0, 0, 0, loc)
	rows := []struct {
		BranchCode, Currency, Amount string
	}{
		{"PP-01", "USD", "2500.0000"},
		{"PP-01", "KHR", "10000000.0000"},
		{"SR-01", "USD", "800.0000"},
		{"SR-01", "KHR", "3000000.0000"},
		{"BT-01", "USD", "500.0000"},
		{"BT-01", "KHR", "2000000.0000"},
	}
	for _, r := range rows {
		b := branches[r.BranchCode]
		var n int64
		tx.Model(&models.OpeningBalance{}).
			Where("branch_id = ? AND currency = ? AND balance_date = ?", b.ID, r.Currency, date).
			Count(&n)
		if n > 0 {
			continue
		}
		amt, _ := decimal.NewFromString(r.Amount)
		ob := models.OpeningBalance{
			Base:        models.Base{ID: uuid.New()},
			BranchID:    b.ID,
			Currency:    r.Currency,
			Amount:      amt,
			BalanceDate: date,
			Notes:       "Demo opening balance",
			CreatedByID: actor.ID,
			Status:      models.TxnStatusApproved,
		}
		if err := tx.Create(&ob).Error; err != nil {
			return err
		}
	}
	return nil
}

func seedDemoTransactions(tx *gorm.DB, branches map[string]models.Branch, users map[string]models.User) error {
	// Skip if we already have demo-tagged transactions
	var cnt int64
	tx.Model(&models.Transaction{}).Where("description LIKE ?", "DEMO:%").Count(&cnt)
	if cnt > 0 {
		return nil
	}

	var cats []models.Category
	if err := tx.Find(&cats).Error; err != nil {
		return err
	}
	catID := map[string]uuid.UUID{}
	for _, c := range cats {
		catID[c.Code] = c.ID
	}
	var pms []models.PaymentMethod
	if err := tx.Find(&pms).Error; err != nil {
		return err
	}
	pmID := map[string]uuid.UUID{}
	for _, p := range pms {
		pmID[p.Code] = p.ID
	}

	loc := time.FixedZone("Asia/Phnom_Penh", 7*3600)
	now := time.Now().In(loc)
	manager := users["manager1"]
	cashier := users["cashier1"]
	mgrSR := users["manager_sr"]

	type row struct {
		Type, Branch, Cat, PM, Currency, Amount, Desc string
		DaysAgo                                       int
		Pending                                       bool
		Actor                                         models.User
	}
	rows := []row{
		{"income", "PP-01", "sales", "cash", "USD", "350.00", "DEMO: Retail sales morning", 0, false, cashier},
		{"income", "PP-01", "sales", "khqr", "KHR", "1250000", "DEMO: KHQR walk-in sales", 0, false, cashier},
		{"income", "PP-01", "services", "bank_transfer", "USD", "120.00", "DEMO: Service invoice #D-101", 1, false, manager},
		{"expense", "PP-01", "rent", "bank_transfer", "USD", "800.00", "DEMO: October office rent", 2, false, manager},
		{"expense", "PP-01", "utilities", "cash", "KHR", "185000", "DEMO: Electricity bill", 1, false, manager},
		{"expense", "PP-01", "miscellaneous", "cash", "USD", "45.50", "DEMO: Office supplies", 0, false, cashier},
		{"income", "PP-01", "sales", "cash", "USD", "75.00", "DEMO: Pending large order", 0, true, cashier},
		{"income", "SR-01", "sales", "cash", "USD", "210.00", "DEMO: SR boutique sales", 0, false, mgrSR},
		{"income", "SR-01", "sales", "khqr", "KHR", "890000", "DEMO: SR tourist payments", 1, false, mgrSR},
		{"expense", "SR-01", "salaries", "bank_transfer", "USD", "400.00", "DEMO: SR staff advance", 3, false, mgrSR},
		{"income", "BT-01", "sales", "cash", "KHR", "450000", "DEMO: BT market day", 0, false, manager},
		{"expense", "BT-01", "transportation", "cash", "USD", "35.00", "DEMO: Delivery fuel", 1, false, manager},
	}

	incSeq, expSeq := 1, 1
	ym := now.Format("200601")

	for _, r := range rows {
		b := branches[r.Branch]
		amt, err := decimal.NewFromString(r.Amount)
		if err != nil {
			return err
		}
		status := models.TxnStatusApproved
		var approvedBy *uuid.UUID
		var approvedAt *time.Time
		if r.Pending {
			status = models.TxnStatusPending
		} else {
			approvedBy = &manager.ID
			t := now.AddDate(0, 0, -r.DaysAgo)
			approvedAt = &t
		}
		prefix := "INC"
		seq := &incSeq
		if r.Type == models.TxnTypeExpense {
			prefix = "EXP"
			seq = &expSeq
		}
		num := fmt.Sprintf("%s-%s-%06d", prefix, ym, *seq)
		*seq++

		day := now.AddDate(0, 0, -r.DaysAgo)
		categoryID := catID[r.Cat]
		txn := models.Transaction{
			Base:              models.Base{ID: uuid.New()},
			TransactionNumber: num,
			Type:              r.Type,
			Status:            status,
			BranchID:          b.ID,
			CategoryID:        &categoryID,
			PaymentMethodID:   pmID[r.PM],
			Currency:          r.Currency,
			Amount:            amt,
			TransactionDate:   day,
			Description:       r.Desc,
			CreatedByID:       r.Actor.ID,
			ApprovedByID:      approvedBy,
			ApprovedAt:        approvedAt,
		}
		if catID[r.Cat] == uuid.Nil {
			return fmt.Errorf("demo category missing: %s", r.Cat)
		}
		if pmID[r.PM] == uuid.Nil {
			// fallback first payment method
			for _, id := range pmID {
				txn.PaymentMethodID = id
				break
			}
		}
		if err := tx.Create(&txn).Error; err != nil {
			return fmt.Errorf("txn %s: %w", num, err)
		}
	}

	// Ensure number sequences are ahead of demo numbers
	_ = ensureSeq(tx, "income", ym, incSeq)
	_ = ensureSeq(tx, "expense", ym, expSeq)
	return nil
}

func ensureSeq(tx *gorm.DB, kind, period string, next int) error {
	prefixCode := "INC"
	if kind == "expense" {
		prefixCode = "EXP"
	}
	prefix := prefixCode + "-" + period
	var seq models.TransactionNumberSeq
	err := tx.Where("prefix = ?", prefix).First(&seq).Error
	last := int64(next - 1)
	if last < 0 {
		last = 0
	}
	if err == gorm.ErrRecordNotFound {
		return tx.Create(&models.TransactionNumberSeq{
			ID: uuid.New(), Prefix: prefix, LastValue: last, UpdatedAt: time.Now(),
		}).Error
	}
	if err != nil {
		return err
	}
	if seq.LastValue < last {
		return tx.Model(&seq).Updates(map[string]interface{}{"last_value": last, "updated_at": time.Now()}).Error
	}
	return nil
}

func seedDemoReconciliation(tx *gorm.DB, branch models.Branch, actor models.User) error {
	loc := time.FixedZone("Asia/Phnom_Penh", 7*3600)
	day := time.Now().In(loc).Truncate(24 * time.Hour)
	var n int64
	tx.Model(&models.CashReconciliation{}).
		Where("branch_id = ? AND currency = ? AND reconcile_date = ?", branch.ID, "USD", day).
		Count(&n)
	if n > 0 {
		return nil
	}
	exp, _ := decimal.NewFromString("2150.0000")
	act, _ := decimal.NewFromString("2145.5000")
	cr := models.CashReconciliation{
		Base:           models.Base{ID: uuid.New()},
		BranchID:       branch.ID,
		Currency:       "USD",
		ReconcileDate:  day,
		ExpectedAmount: exp,
		ActualAmount:   act,
		Difference:     act.Sub(exp),
		Notes:          "DEMO: end-of-day cash count short $4.50",
		CreatedByID:    actor.ID,
		Status:         "completed",
	}
	return tx.Create(&cr).Error
}
