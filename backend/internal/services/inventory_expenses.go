package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/cambodia-finance/income-expense/internal/accounting"
	"github.com/cambodia-finance/income-expense/internal/models"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Inventory expenses: costs of the inventory side (shipping, packaging, storage…) in their own
// categories (type inv_expense). Saved at once; voiding cancels one. Reports take them off the
// sales profit.

type InventoryExpenseInput struct {
	BranchID        string `json:"branch_id" binding:"required,uuid"`
	CategoryID      string `json:"category_id" binding:"required,uuid"`
	ExpenseDate     string `json:"expense_date"`
	Currency        string `json:"currency" binding:"required,len=3"`
	Amount          string `json:"amount" binding:"required"`
	PaymentMethodID string `json:"payment_method_id"` // empty: Cash
	Payee           string `json:"payee" binding:"max=191"`
	Note            string `json:"note"` // rich text
}

type InventoryExpenseView struct {
	models.InventoryExpense
	BranchCode    string `json:"branch_code"`
	BranchName    string `json:"branch_name"`
	CategoryName  string `json:"category_name"`
	PaymentMethod string `json:"payment_method"`
	CreatedBy     string `json:"created_by"`
	VoidedBy      string `json:"voided_by"`
}

func (s *InventoryService) CreateExpense(ctx context.Context, in InventoryExpenseInput, scope BranchScope, actor uuid.UUID) (*InventoryExpenseView, error) {
	branchID := uuid.MustParse(in.BranchID)
	if !scope.Has(branchID) {
		return nil, ErrBranchAccessDenied
	}
	var cat models.Category
	if err := s.db.WithContext(ctx).First(&cat, "id = ? AND type = ? AND is_active = ?", in.CategoryID, models.CategoryTypeInventoryExpense, true).Error; err != nil {
		return nil, fieldError("category_id", "choose an active inventory expense category")
	}
	if cat.BranchID != nil && *cat.BranchID != branchID {
		return nil, fieldError("category_id", "this category belongs to another branch")
	}
	amount, err := accounting.ParseAmount(in.Amount)
	if err != nil {
		return nil, fieldError("amount", err.Error())
	}
	date, err := s.parseDocDate("expense_date", in.ExpenseDate)
	if err != nil {
		return nil, err
	}
	paymentID, err := s.paymentMethod(ctx, in.PaymentMethodID)
	if err != nil {
		return nil, err
	}
	note, err := cleanNote(in.Note)
	if err != nil {
		return nil, err
	}
	number, err := s.numbers.NextFor(ctx, "IEX")
	if err != nil {
		return nil, err
	}
	rate, err := currencyRate(ctx, s.db, in.Currency)
	if err != nil {
		return nil, err
	}
	exp := models.InventoryExpense{
		Base: models.Base{ID: uuid.New()}, Number: number, BranchID: branchID, CategoryID: cat.ID,
		ExpenseDate: date, Currency: in.Currency, Amount: amount, ExchangeRate: rate, AmountBase: toBase(amount, in.Currency, rate), PaymentMethodID: paymentID,
		Payee: strings.TrimSpace(in.Payee), Note: note, Status: models.DocCompleted, CreatedByID: actor,
	}
	if err := s.db.WithContext(ctx).Create(&exp).Error; err != nil {
		return nil, err
	}
	return s.GetExpense(ctx, exp.ID, scope)
}

func (s *InventoryService) VoidExpense(ctx context.Context, id uuid.UUID, reason string, scope BranchScope, actor uuid.UUID) (*InventoryExpenseView, error) {
	reason = strings.TrimSpace(reason)
	if len(reason) < 3 {
		return nil, fieldError("reason", "give a reason for voiding")
	}
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var exp models.InventoryExpense
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&exp, "id = ?", id).Error; err != nil {
			return notFound(err)
		}
		if !scope.Has(exp.BranchID) {
			return ErrInventoryNotFound
		}
		if exp.Status != models.DocCompleted {
			return fmt.Errorf("this expense is already %s", exp.Status)
		}
		now := time.Now()
		return tx.Model(&exp).Updates(map[string]interface{}{"status": models.DocVoided, "voided_by_id": actor, "voided_at": now, "void_reason": reason}).Error
	})
	if err != nil {
		return nil, err
	}
	return s.GetExpense(ctx, id, scope)
}

func (s *InventoryService) GetExpense(ctx context.Context, id uuid.UUID, scope BranchScope) (*InventoryExpenseView, error) {
	var exp models.InventoryExpense
	if err := s.db.WithContext(ctx).First(&exp, "id = ?", id).Error; err != nil {
		return nil, notFound(err)
	}
	if !scope.Has(exp.BranchID) {
		return nil, ErrInventoryNotFound
	}
	views := s.expenseViews(ctx, []models.InventoryExpense{exp})
	return &views[0], nil
}

type ExpenseFilter struct {
	DocFilter
	CategoryID *uuid.UUID
}

func (s *InventoryService) ListExpenses(ctx context.Context, f ExpenseFilter, scope BranchScope) ([]InventoryExpenseView, int64, error) {
	q := scope.where(s.db.WithContext(ctx).Model(&models.InventoryExpense{}), "branch_id")
	if f.BranchID != nil {
		q = q.Where("branch_id = ?", *f.BranchID)
	}
	if f.CategoryID != nil {
		q = q.Where("category_id = ?", *f.CategoryID)
	}
	if f.Status != "" {
		q = q.Where("status = ?", f.Status)
	}
	if f.From != nil {
		q = q.Where("expense_date >= ?", *f.From)
	}
	if f.To != nil {
		q = q.Where("expense_date <= ?", *f.To)
	}
	q = docSearch(q, f.Search, "number", "payee", "note")
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	page, size := pageArgs(f.Page, f.PageSize)
	var list []models.InventoryExpense
	if err := q.Order("expense_date DESC, created_at DESC").Limit(size).Offset((page - 1) * size).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return s.expenseViews(ctx, list), total, nil
}

func (s *InventoryService) expenseViews(ctx context.Context, list []models.InventoryExpense) []InventoryExpenseView {
	var users, branches, payments, cats []uuid.UUID
	for _, e := range list {
		users = append(users, e.CreatedByID)
		if e.VoidedByID != nil {
			users = append(users, *e.VoidedByID)
		}
		branches = append(branches, e.BranchID)
		cats = append(cats, e.CategoryID)
		if e.PaymentMethodID != nil {
			payments = append(payments, *e.PaymentMethodID)
		}
	}
	book := s.lookup(ctx, users, branches, nil)
	paymentNames := s.paymentNames(ctx, payments)
	catNames := map[uuid.UUID]string{}
	if len(cats) > 0 {
		var list []models.Category
		s.db.WithContext(ctx).Unscoped().Select("id", "name").Where("id IN ?", cats).Find(&list)
		for _, c := range list {
			catNames[c.ID] = c.Name
		}
	}
	out := make([]InventoryExpenseView, len(list))
	for i, e := range list {
		v := InventoryExpenseView{InventoryExpense: e, BranchCode: book.branches[e.BranchID].Code, BranchName: book.branches[e.BranchID].Name,
			CategoryName: catNames[e.CategoryID], CreatedBy: book.users[e.CreatedByID]}
		if e.PaymentMethodID != nil {
			v.PaymentMethod = paymentNames[*e.PaymentMethodID]
		}
		if e.VoidedByID != nil {
			v.VoidedBy = book.users[*e.VoidedByID]
		}
		out[i] = v
	}
	return out
}

// ExpenseReportRow: inventory expenses in the period by category.
type ExpenseReportRow struct {
	CategoryID uuid.UUID       `json:"category_id"`
	Name       string          `json:"name"`
	Currency   string          `json:"currency"` // USD
	Count      int64           `json:"count"`
	Total      decimal.Decimal `json:"total"`
}

func (s *InventoryService) ExpensesReport(ctx context.Context, f InventoryReportFilter, scope BranchScope) ([]ExpenseReportRow, error) {
	q := s.db.WithContext(ctx).Table("inventory_expenses").
		Joins("LEFT JOIN categories ON categories.id = inventory_expenses.category_id").
		Where("inventory_expenses.status = ? AND inventory_expenses.deleted_at IS NULL", models.DocCompleted)
	q = dateFilter(s.branchFilter(q, "inventory_expenses.branch_id", f, scope), "inventory_expenses.expense_date", f)
	var rows []ExpenseReportRow
	err := q.Select(`inventory_expenses.category_id, COALESCE(categories.name, '') AS name, 'USD' AS currency,
		COUNT(*) AS count, SUM(inventory_expenses.amount_base) AS total`).
		Group("inventory_expenses.category_id, categories.name").Order("total DESC").Scan(&rows).Error
	return rows, err
}
