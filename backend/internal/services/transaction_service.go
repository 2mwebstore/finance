package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/cambodia-finance/income-expense/internal/accounting"
	"github.com/cambodia-finance/income-expense/internal/middleware"
	"github.com/cambodia-finance/income-expense/internal/models"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

var (
	ErrTransactionNotFound  = errors.New("transaction not found")
	ErrInvalidStatus        = errors.New("invalid transaction status for this operation")
	ErrBranchAccessDenied   = errors.New("no access to this branch")
	ErrCategoryTypeMismatch = errors.New("category type does not match transaction type")
	ErrCategoryBranch       = errors.New("this category belongs to another branch")
)

type TransactionService struct {
	db      *gorm.DB
	numbers *accounting.NumberGenerator
}

func NewTransactionService(db *gorm.DB, timezone string) *TransactionService {
	return &TransactionService{
		db:      db,
		numbers: accounting.NewNumberGenerator(db, timezone),
	}
}

type CreateTransactionInput struct {
	Type            string `json:"type" binding:"required,oneof=income expense"`
	BranchID        string `json:"branch_id" binding:"required,uuid"`
	CategoryID      string `json:"category_id" binding:"omitempty,uuid"`
	PaymentMethodID string `json:"payment_method_id"`
	Currency        string `json:"currency" binding:"required,oneof=USD KHR"`
	Amount          string `json:"amount" binding:"required"` // string to avoid float
	TransactionDate string `json:"transaction_date"`          // RFC3339 or YYYY-MM-DD; default now
	Description     string `json:"description"`
	Counterparty    string `json:"counterparty"`
	// If true, create as pending (requires approval). Default: approved (MVP).
	Pending bool `json:"pending"`
}

type UpdateTransactionInput struct {
	CategoryID      *string `json:"category_id"`
	PaymentMethodID *string `json:"payment_method_id"`
	Amount          *string `json:"amount"`
	TransactionDate *string `json:"transaction_date"`
	Description     *string `json:"description"`
	Counterparty    *string `json:"counterparty"`
}

type TransactionListFilter struct {
	Type       string
	BranchID   string
	Currency   string
	CategoryID string
	Status     string
	DateFrom   *time.Time
	DateTo     *time.Time // exclusive end (half-open)
	Search     string
	Page       int
	PageSize   int
}

func (s *TransactionService) Create(ctx context.Context, in CreateTransactionInput, actorID uuid.UUID) (*models.Transaction, error) {
	if err := accounting.ValidateCurrency(in.Currency); err != nil {
		return nil, err
	}
	if err := accounting.ValidateTxnType(in.Type); err != nil {
		return nil, err
	}
	amount, err := accounting.ParseAmount(in.Amount)
	if err != nil {
		return nil, err
	}

	branchID, err := uuid.Parse(in.BranchID)
	if err != nil {
		return nil, fmt.Errorf("invalid branch_id")
	}
	if err := s.assertBranchAccess(ctx, actorID, branchID); err != nil {
		return nil, err
	}

	var categoryID *uuid.UUID
	if in.CategoryID != "" {
		if in.Type == models.TxnTypeIncome {
			return nil, errors.New("income transactions cannot have a category")
		}
		parsedCategoryID, parseErr := uuid.Parse(in.CategoryID)
		if parseErr != nil {
			return nil, fmt.Errorf("invalid category_id")
		}
		categoryID = &parsedCategoryID
	} else if in.Type == models.TxnTypeExpense {
		return nil, fmt.Errorf("category_id is required for expenses")
	}
	var paymentMethodID uuid.UUID
	if in.PaymentMethodID == "" {
		var pm models.PaymentMethod
		if err := s.db.WithContext(ctx).Where("is_active = ?", true).Order("created_at ASC").First(&pm).Error; err != nil {
			return nil, fmt.Errorf("no active payment method is configured")
		}
		paymentMethodID = pm.ID
	} else {
		paymentMethodID, err = uuid.Parse(in.PaymentMethodID)
		if err != nil {
			return nil, fmt.Errorf("invalid payment_method_id")
		}
	}

	if categoryID != nil {
		var cat models.Category
		if err := s.db.WithContext(ctx).First(&cat, "id = ? AND is_active = ?", *categoryID, true).Error; err != nil {
			return nil, fmt.Errorf("category not found or inactive")
		}
		if cat.Type != in.Type {
			return nil, ErrCategoryTypeMismatch
		}
		if cat.BranchID != nil && *cat.BranchID != branchID {
			return nil, ErrCategoryBranch
		}
	}

	var pm models.PaymentMethod
	if err := s.db.WithContext(ctx).First(&pm, "id = ? AND is_active = ?", paymentMethodID, true).Error; err != nil {
		return nil, fmt.Errorf("payment method not found or inactive")
	}

	txnDate := time.Now()
	if in.TransactionDate != "" {
		txnDate, err = parseFlexibleDate(in.TransactionDate)
		if err != nil {
			return nil, fmt.Errorf("invalid transaction_date: %w", err)
		}
	}

	status := models.TxnStatusApproved
	var approvedBy *uuid.UUID
	var approvedAt *time.Time
	if in.Pending {
		status = models.TxnStatusPending
	} else {
		approvedBy = &actorID
		now := time.Now()
		approvedAt = &now
	}

	number, err := s.numbers.Next(ctx, in.Type)
	if err != nil {
		return nil, fmt.Errorf("generate transaction number: %w", err)
	}

	txn := models.Transaction{
		Base:              models.Base{ID: uuid.New()},
		TransactionNumber: number,
		Type:              in.Type,
		Status:            status,
		BranchID:          branchID,
		CategoryID:        categoryID,
		PaymentMethodID:   paymentMethodID,
		Currency:          in.Currency,
		Amount:            amount,
		TransactionDate:   txnDate,
		Description:       in.Description,
		Counterparty:      in.Counterparty,
		CreatedByID:       actorID,
		ApprovedByID:      approvedBy,
		ApprovedAt:        approvedAt,
	}

	if err := s.db.WithContext(ctx).Create(&txn).Error; err != nil {
		return nil, err
	}
	return s.GetByID(ctx, txn.ID, actorID)
}

func (s *TransactionService) assertBranchAccess(ctx context.Context, actorID, branchID uuid.UUID) error {
	allowed, all, err := middleware.GetAuthorizedBranchIDs(s.db, actorID)
	if err != nil {
		return err
	}
	if all {
		return nil
	}
	for _, id := range allowed {
		if id == branchID {
			return nil
		}
	}
	return ErrBranchAccessDenied
}

func (s *TransactionService) GetByID(ctx context.Context, id, actorID uuid.UUID) (*models.Transaction, error) {
	var txn models.Transaction
	err := s.db.WithContext(ctx).
		Preload("Branch").
		Preload("Category").
		Preload("PaymentMethod").
		First(&txn, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTransactionNotFound
		}
		return nil, err
	}
	if err := s.assertBranchAccess(ctx, actorID, txn.BranchID); err != nil {
		return nil, err
	}
	return &txn, nil
}

func (s *TransactionService) List(ctx context.Context, f TransactionListFilter, actorID uuid.UUID) ([]models.Transaction, int64, error) {
	allowed, all, err := middleware.GetAuthorizedBranchIDs(s.db, actorID)
	if err != nil {
		return nil, 0, err
	}

	q := s.db.WithContext(ctx).Model(&models.Transaction{})
	if !all {
		if len(allowed) == 0 {
			return []models.Transaction{}, 0, nil
		}
		q = q.Where("branch_id IN ?", allowed)
	}
	if f.BranchID != "" {
		bid, err := uuid.Parse(f.BranchID)
		if err != nil {
			return nil, 0, fmt.Errorf("invalid branch_id")
		}
		if !all {
			ok := false
			for _, a := range allowed {
				if a == bid {
					ok = true
					break
				}
			}
			if !ok {
				return nil, 0, ErrBranchAccessDenied
			}
		}
		q = q.Where("branch_id = ?", bid)
	}
	if f.Type != "" {
		q = q.Where("type = ?", f.Type)
	}
	if f.Currency != "" {
		q = q.Where("currency = ?", f.Currency)
	}
	if f.CategoryID != "" {
		q = q.Where("category_id = ?", f.CategoryID)
	}
	if f.Status != "" {
		q = q.Where("status = ?", f.Status)
	}
	if f.DateFrom != nil {
		q = q.Where("transaction_date >= ?", *f.DateFrom)
	}
	if f.DateTo != nil {
		q = q.Where("transaction_date < ?", *f.DateTo)
	}
	if f.Search != "" {
		like := "%" + f.Search + "%"
		q = q.Where("transaction_number LIKE ? OR description LIKE ? OR counterparty LIKE ?", like, like, like)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	page := f.Page
	if page < 1 {
		page = 1
	}
	size := f.PageSize
	if size < 1 || size > 100 {
		size = 10
	}

	var items []models.Transaction
	err = q.Preload("Branch").Preload("Category").Preload("PaymentMethod").
		Order("transaction_date DESC, created_at DESC").
		Offset((page - 1) * size).Limit(size).
		Find(&items).Error
	return items, total, err
}

func (s *TransactionService) Update(ctx context.Context, id uuid.UUID, in UpdateTransactionInput, actorID uuid.UUID) (*models.Transaction, error) {
	txn, err := s.GetByID(ctx, id, actorID)
	if err != nil {
		return nil, err
	}
	// Only pending or approved (non-voided) can be edited; voided/reversed locked
	if txn.Status == models.TxnStatusVoided || txn.Status == models.TxnStatusReversed {
		return nil, ErrInvalidStatus
	}

	updates := map[string]interface{}{}
	if in.CategoryID != nil {
		if txn.Type == models.TxnTypeIncome {
			return nil, errors.New("income transactions cannot have a category")
		}
		cid, err := uuid.Parse(*in.CategoryID)
		if err != nil {
			return nil, fmt.Errorf("invalid category_id")
		}
		var cat models.Category
		if err := s.db.WithContext(ctx).First(&cat, "id = ?", cid).Error; err != nil {
			return nil, fmt.Errorf("category not found")
		}
		if cat.Type != txn.Type {
			return nil, ErrCategoryTypeMismatch
		}
		if cat.BranchID != nil && *cat.BranchID != txn.BranchID {
			return nil, ErrCategoryBranch
		}
		updates["category_id"] = cid
	}
	if in.PaymentMethodID != nil {
		pid, err := uuid.Parse(*in.PaymentMethodID)
		if err != nil {
			return nil, fmt.Errorf("invalid payment_method_id")
		}
		updates["payment_method_id"] = pid
	}
	if in.Amount != nil {
		amt, err := accounting.ParseAmount(*in.Amount)
		if err != nil {
			return nil, err
		}
		updates["amount"] = amt
	}
	if in.TransactionDate != nil {
		d, err := parseFlexibleDate(*in.TransactionDate)
		if err != nil {
			return nil, fmt.Errorf("invalid transaction_date")
		}
		updates["transaction_date"] = d
	}
	if in.Description != nil {
		updates["description"] = *in.Description
	}
	if in.Counterparty != nil {
		updates["counterparty"] = *in.Counterparty
	}
	// Currency and type and branch are immutable after create
	if len(updates) > 0 {
		if err := s.db.WithContext(ctx).Model(txn).Updates(updates).Error; err != nil {
			return nil, err
		}
	}
	return s.GetByID(ctx, id, actorID)
}

func (s *TransactionService) Void(ctx context.Context, id uuid.UUID, reason string, actorID uuid.UUID) (*models.Transaction, error) {
	txn, err := s.GetByID(ctx, id, actorID)
	if err != nil {
		return nil, err
	}
	if txn.Status == models.TxnStatusVoided || txn.Status == models.TxnStatusReversed {
		return nil, ErrInvalidStatus
	}
	now := time.Now()
	err = s.db.WithContext(ctx).Model(txn).Updates(map[string]interface{}{
		"status":       models.TxnStatusVoided,
		"voided_by_id": actorID,
		"voided_at":    now,
		"void_reason":  reason,
	}).Error
	if err != nil {
		return nil, err
	}
	return s.GetByID(ctx, id, actorID)
}

// TotalsByCurrency returns separate income/expense/net for USD and KHR for finalized (approved) txns only.
func (s *TransactionService) TotalsByCurrency(ctx context.Context, actorID uuid.UUID, branchID *uuid.UUID, from, to *time.Time, txnType, currency, search string) (map[string]decimal.Decimal, error) {
	allowed, all, err := middleware.GetAuthorizedBranchIDs(s.db, actorID)
	if err != nil {
		return nil, err
	}

	type row struct {
		Currency string
		Type     string
		Total    decimal.Decimal
	}
	q := s.db.WithContext(ctx).Model(&models.Transaction{}).
		Select("currency, type, COALESCE(SUM(amount), 0) as total").
		Where("status = ?", models.TxnStatusApproved).
		Group("currency, type")

	if !all {
		if len(allowed) == 0 {
			return map[string]decimal.Decimal{
				"income_usd": accounting.Zero(), "expense_usd": accounting.Zero(), "net_usd": accounting.Zero(),
				"income_khr": accounting.Zero(), "expense_khr": accounting.Zero(), "net_khr": accounting.Zero(),
			}, nil
		}
		q = q.Where("branch_id IN ?", allowed)
	}
	if branchID != nil {
		q = q.Where("branch_id = ?", *branchID)
	}
	if from != nil {
		q = q.Where("transaction_date >= ?", *from)
	}
	if to != nil {
		q = q.Where("transaction_date < ?", *to)
	}
	if txnType != "" {
		q = q.Where("type = ?", txnType)
	}
	if currency != "" {
		q = q.Where("currency = ?", currency)
	}
	if search != "" {
		like := "%" + search + "%"
		q = q.Where("transaction_number LIKE ? OR description LIKE ? OR counterparty LIKE ?", like, like, like)
	}

	var rows []row
	if err := q.Scan(&rows).Error; err != nil {
		return nil, err
	}

	out := map[string]decimal.Decimal{
		"income_usd":  accounting.Zero(),
		"expense_usd": accounting.Zero(),
		"income_khr":  accounting.Zero(),
		"expense_khr": accounting.Zero(),
	}
	for _, r := range rows {
		key := r.Type + "_" + toLower(r.Currency)
		if _, ok := out[key]; ok {
			out[key] = r.Total
		}
	}
	out["net_usd"] = out["income_usd"].Sub(out["expense_usd"])
	out["net_khr"] = out["income_khr"].Sub(out["expense_khr"])
	return out, nil
}

func toLower(s string) string {
	b := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		b[i] = c
	}
	return string(b)
}

func parseFlexibleDate(s string) (time.Time, error) {
	layouts := []string{
		time.RFC3339,
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
		"2006-01-02",
	}
	for _, l := range layouts {
		if t, err := time.ParseInLocation(l, s, time.FixedZone("Asia/Phnom_Penh", 7*3600)); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unrecognized date format")
}

func (s *TransactionService) Approve(ctx context.Context, id uuid.UUID, notes string, actorID uuid.UUID) (*models.Transaction, error) {
	txn, err := s.GetByID(ctx, id, actorID)
	if err != nil {
		return nil, err
	}
	if txn.Status != models.TxnStatusPending {
		return nil, ErrInvalidStatus
	}
	// Prevent self-approval
	if txn.CreatedByID == actorID {
		return nil, errors.New("cannot approve your own transaction")
	}
	now := time.Now()
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(txn).Updates(map[string]interface{}{
			"status":         models.TxnStatusApproved,
			"approved_by_id": actorID,
			"approved_at":    now,
		}).Error; err != nil {
			return err
		}
		return tx.Create(&models.TransactionApproval{
			ID: uuid.New(), TransactionID: id, ActorID: actorID,
			Action: "approve", Notes: notes, CreatedAt: now,
		}).Error
	})
	if err != nil {
		return nil, err
	}
	return s.GetByID(ctx, id, actorID)
}

func (s *TransactionService) Reject(ctx context.Context, id uuid.UUID, notes string, actorID uuid.UUID) (*models.Transaction, error) {
	txn, err := s.GetByID(ctx, id, actorID)
	if err != nil {
		return nil, err
	}
	if txn.Status != models.TxnStatusPending {
		return nil, ErrInvalidStatus
	}
	if txn.CreatedByID == actorID {
		return nil, errors.New("cannot reject your own transaction")
	}
	now := time.Now()
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(txn).Updates(map[string]interface{}{
			"status": models.TxnStatusRejected,
		}).Error; err != nil {
			return err
		}
		return tx.Create(&models.TransactionApproval{
			ID: uuid.New(), TransactionID: id, ActorID: actorID,
			Action: "reject", Notes: notes, CreatedAt: now,
		}).Error
	})
	if err != nil {
		return nil, err
	}
	return s.GetByID(ctx, id, actorID)
}

// Reverse creates an opposite transaction and marks the original as reversed.
// Only approved transactions can be reversed. History is preserved.
func (s *TransactionService) Reverse(ctx context.Context, id uuid.UUID, reason string, actorID uuid.UUID) (*models.Transaction, error) {
	orig, err := s.GetByID(ctx, id, actorID)
	if err != nil {
		return nil, err
	}
	if orig.Status != models.TxnStatusApproved {
		return nil, ErrInvalidStatus
	}
	if orig.ReversesTxnID != nil {
		return nil, errors.New("cannot reverse a reversal transaction")
	}
	if orig.ReversedByTxnID != nil {
		return nil, errors.New("transaction already reversed")
	}

	oppType := models.TxnTypeExpense
	if orig.Type == models.TxnTypeExpense {
		oppType = models.TxnTypeIncome
	}

	number, err := s.numbers.Next(ctx, oppType)
	if err != nil {
		return nil, fmt.Errorf("generate transaction number: %w", err)
	}

	now := time.Now()
	var rev *models.Transaction
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		revTxn := models.Transaction{
			Base:              models.Base{ID: uuid.New()},
			TransactionNumber: number,
			Type:              oppType,
			Status:            models.TxnStatusApproved,
			BranchID:          orig.BranchID,
			CategoryID:        orig.CategoryID,
			PaymentMethodID:   orig.PaymentMethodID,
			Currency:          orig.Currency,
			Amount:            orig.Amount,
			TransactionDate:   now,
			Description:       "Reversal of " + orig.TransactionNumber + ": " + reason,
			Counterparty:      orig.Counterparty,
			CreatedByID:       actorID,
			ApprovedByID:      &actorID,
			ApprovedAt:        &now,
			ReversesTxnID:     &orig.ID,
		}
		if err := tx.Create(&revTxn).Error; err != nil {
			return err
		}
		rev = &revTxn
		// Keep original approved so ledger stays balanced (income + opposite expense = net 0).
		// Link via reversed_by_txn_id for audit UI.
		return tx.Model(orig).Updates(map[string]interface{}{
			"reversed_by_txn_id": revTxn.ID,
			"void_reason":        reason,
		}).Error
	})
	if err != nil {
		return nil, err
	}
	return s.GetByID(ctx, rev.ID, actorID)
}
