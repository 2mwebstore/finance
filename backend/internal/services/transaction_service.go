package services

import (
	"context"
	"errors"
	"fmt"
	"html"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/cambodia-finance/income-expense/internal/accounting"
	"github.com/cambodia-finance/income-expense/internal/middleware"
	"github.com/cambodia-finance/income-expense/internal/models"
	"github.com/cambodia-finance/income-expense/internal/permissions"
	"github.com/cambodia-finance/income-expense/internal/richtext"
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
	ErrNotApprover          = errors.New("only this user's managers can approve or reject it")
	ErrDescriptionTooLong   = errors.New("description must be at most 1000 characters")
)

// DescriptionMaxChars matches the counter on the transaction form.
const DescriptionMaxChars = 1000

// cleanDescription keeps the allowed rich-text subset and enforces the size limit.
func cleanDescription(in string) (string, error) {
	out := richtext.Sanitize(in)
	if len(out) > richtext.MaxHTMLLength || utf8.RuneCountInString(richtext.ToText(out)) > DescriptionMaxChars {
		return "", ErrDescriptionTooLong
	}
	return out, nil
}

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
		return nil, fieldError("branch_id", "choose a valid branch")
	}
	if err := s.assertBranchAccess(ctx, actorID, branchID); err != nil {
		return nil, err
	}

	var categoryID *uuid.UUID
	if in.CategoryID != "" {
		if in.Type == models.TxnTypeIncome {
			return nil, fieldError("category_id", "income transactions cannot have a category")
		}
		parsedCategoryID, parseErr := uuid.Parse(in.CategoryID)
		if parseErr != nil {
			return nil, fieldError("category_id", "choose a valid category")
		}
		categoryID = &parsedCategoryID
	} else if in.Type == models.TxnTypeExpense {
		return nil, fieldError("category_id", "category is required")
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
			return nil, fieldError("category_id", "this category was not found or is inactive")
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
	if strings.TrimSpace(in.TransactionDate) != "" {
		txnDate, err = parseFlexibleDate(in.TransactionDate)
		if err != nil {
			return nil, fieldError("transaction_date", "date is not valid")
		}
	}

	status := models.TxnStatusApproved
	var approvedBy *uuid.UUID
	var approvedAt *time.Time
	// People who can't approve always submit for approval, whatever the request says.
	if in.Pending || !s.canApproveTransactions(ctx, actorID) {
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

	description, err := cleanDescription(in.Description)
	if err != nil {
		return nil, err
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
		Description:       description,
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
			return nil, fieldError("category_id", "income transactions cannot have a category")
		}
		if strings.TrimSpace(*in.CategoryID) == "" {
			return nil, fieldError("category_id", "category is required")
		}
		cid, err := uuid.Parse(*in.CategoryID)
		if err != nil {
			return nil, fieldError("category_id", "choose a valid category")
		}
		var cat models.Category
		if err := s.db.WithContext(ctx).First(&cat, "id = ?", cid).Error; err != nil {
			return nil, fieldError("category_id", "this category was not found")
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
		if strings.TrimSpace(*in.Amount) == "" {
			return nil, fieldError("amount", "amount is required")
		}
		amt, err := accounting.ParseAmount(*in.Amount)
		if err != nil {
			return nil, err
		}
		updates["amount"] = amt
	}
	if in.TransactionDate != nil {
		if strings.TrimSpace(*in.TransactionDate) == "" {
			return nil, fieldError("transaction_date", "date is required")
		}
		d, err := parseFlexibleDate(*in.TransactionDate)
		if err != nil {
			return nil, fieldError("transaction_date", "date is not valid")
		}
		updates["transaction_date"] = d
	}
	if in.Description != nil {
		description, err := cleanDescription(*in.Description)
		if err != nil {
			return nil, err
		}
		updates["description"] = description
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
	if err := s.assertCanDecide(ctx, txn, actorID); err != nil {
		return nil, err
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
	if err := s.assertCanDecide(ctx, txn, actorID); err != nil {
		return nil, err
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
			Description:       html.EscapeString("Reversal of " + orig.TransactionNumber + ": " + reason),
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

// managersOf returns the assigned managers for each of the given users.
func (s *TransactionService) managersOf(ctx context.Context, userIDs []uuid.UUID) (map[uuid.UUID][]uuid.UUID, error) {
	out := map[uuid.UUID][]uuid.UUID{}
	if len(userIDs) == 0 {
		return out, nil
	}
	var links []models.UserManager
	if err := s.db.WithContext(ctx).Where("user_id IN ?", userIDs).Find(&links).Error; err != nil {
		return nil, err
	}
	for _, l := range links {
		out[l.UserID] = append(out[l.UserID], l.ManagerID)
	}
	return out, nil
}

// canDecide: when the creator has assigned managers, only those managers (or someone with
// full access: owner/admin/Super Admin) may approve or reject; otherwise the permission
// and branch rules alone apply. Nobody decides on their own transaction.
func canDecide(creatorID, actorID uuid.UUID, managers []uuid.UUID, actorFull bool) bool {
	if creatorID == actorID {
		return false
	}
	if len(managers) == 0 || actorFull {
		return true
	}
	for _, m := range managers {
		if m == actorID {
			return true
		}
	}
	return false
}

func (s *TransactionService) assertCanDecide(ctx context.Context, txn *models.Transaction, actorID uuid.UUID) error {
	managers, err := s.managersOf(ctx, []uuid.UUID{txn.CreatedByID})
	if err != nil {
		return err
	}
	_, full, err := middleware.GetAuthorizedBranchIDs(s.db, actorID)
	if err != nil {
		return err
	}
	if !canDecide(txn.CreatedByID, actorID, managers[txn.CreatedByID], full) {
		return ErrNotApprover
	}
	return nil
}

// Decision says what the actor may do with a pending transaction.
type Decision struct {
	CanApprove bool
	CanReject  bool
}

// Decisions works out approve/reject rights for the pending items in a list, so the UI only
// shows buttons the API will accept.
func (s *TransactionService) Decisions(ctx context.Context, actorID uuid.UUID, items []models.Transaction) (map[uuid.UUID]Decision, error) {
	out := map[uuid.UUID]Decision{}
	var creators []uuid.UUID
	for _, t := range items {
		if t.Status == models.TxnStatusPending {
			creators = append(creators, t.CreatedByID)
		}
	}
	if len(creators) == 0 {
		return out, nil
	}
	var actor models.User
	if err := s.db.WithContext(ctx).Preload("Roles.Permissions").First(&actor, "id = ?", actorID).Error; err != nil {
		return nil, err
	}
	perms := map[string]bool{}
	for _, r := range actor.Roles {
		for _, p := range r.Permissions {
			perms[p.Code] = true
		}
	}
	full := actor.IsSuperAdmin || perms[permissions.BranchesManage]
	approve := actor.IsSuperAdmin || perms[permissions.TransactionsApprove]
	reject := actor.IsSuperAdmin || perms[permissions.TransactionsReject]
	managers, err := s.managersOf(ctx, creators)
	if err != nil {
		return nil, err
	}
	for _, t := range items {
		if t.Status != models.TxnStatusPending {
			continue
		}
		ok := canDecide(t.CreatedByID, actorID, managers[t.CreatedByID], full)
		out[t.ID] = Decision{CanApprove: ok && approve, CanReject: ok && reject}
	}
	return out, nil
}

// CreatorNames maps user IDs to display names for list rows.
func (s *TransactionService) CreatorNames(ctx context.Context, items []models.Transaction) map[uuid.UUID]string {
	out := map[uuid.UUID]string{}
	ids := make([]uuid.UUID, 0, len(items))
	for _, t := range items {
		ids = append(ids, t.CreatedByID)
	}
	if len(ids) == 0 {
		return out
	}
	var users []models.User
	if err := s.db.WithContext(ctx).Select("id", "full_name", "username").Where("id IN ?", ids).Find(&users).Error; err == nil {
		for _, u := range users {
			name := u.FullName
			if name == "" {
				name = u.Username
			}
			out[u.ID] = name
		}
	}
	return out
}

// CanWriteType reports whether the actor may create or edit transactions of this type
// (used for attachments). Owners/admins/Super Admins always can.
func (s *TransactionService) CanWriteType(ctx context.Context, actorID uuid.UUID, txnType string) bool {
	var actor models.User
	if err := s.db.WithContext(ctx).Preload("Roles.Permissions").First(&actor, "id = ?", actorID).Error; err != nil {
		return false
	}
	if actor.IsSuperAdmin {
		return true
	}
	for _, r := range actor.Roles {
		for _, p := range r.Permissions {
			if p.Code == txnType+".create" || p.Code == txnType+".edit" || p.Code == permissions.BranchesManage {
				return true
			}
		}
	}
	return false
}

// canApproveTransactions: Super Admin or anyone holding transactions.approve.
func (s *TransactionService) canApproveTransactions(ctx context.Context, actorID uuid.UUID) bool {
	var actor models.User
	if err := s.db.WithContext(ctx).Preload("Roles.Permissions").First(&actor, "id = ?", actorID).Error; err != nil {
		return false
	}
	if actor.IsSuperAdmin {
		return true
	}
	for _, r := range actor.Roles {
		for _, p := range r.Permissions {
			if p.Code == permissions.TransactionsApprove {
				return true
			}
		}
	}
	return false
}
