package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/cambodia-finance/income-expense/internal/middleware"
	"github.com/cambodia-finance/income-expense/internal/models"
	"github.com/cambodia-finance/income-expense/internal/permissions"
	"github.com/cambodia-finance/income-expense/internal/textsearch"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// A request is an expense waiting for approval (status pending). Approving it makes it a real,
// approved expense (it moves to Expense); rejecting it keeps it on the Requests page with the
// reason. Requests are the same transactions as everywhere else, so receipts, managers and the
// branch's Telegram updates work the same way.

// Request statuses the Requests page filters by ("" = every request).
const (
	RequestStatusPending  = models.TxnStatusPending
	RequestStatusApproved = models.TxnStatusApproved
	RequestStatusRejected = models.TxnStatusRejected
)

// RequestListFilter narrows the Requests page.
type RequestListFilter struct {
	Status     string // pending | approved | rejected | "" (all)
	BranchID   string
	Currency   string
	CategoryID string
	DateFrom   *time.Time
	DateTo     *time.Time // exclusive end (half-open)
	Search     string
	Page       int
	PageSize   int
}

// RequestCounts is how many requests each status has (with every filter but the status).
type RequestCounts struct {
	Pending  int64 `json:"pending"`
	Approved int64 `json:"approved"`
	Rejected int64 `json:"rejected"`
	All      int64 `json:"all"`
}

// wasApproved: an approval was recorded for the transaction, so it waited as a request before it
// became an expense (one entered by someone who may approve is approved at once and has none).
const wasApproved = "EXISTS (SELECT 1 FROM transaction_approvals ta WHERE ta.transaction_id = transactions.id AND ta.action = 'approve')"

// requestScope is every request the actor may see (see ListRequests), in one branch when branchID
// is set. ok is false when the actor works in no branch, so there is nothing to see.
func (s *TransactionService) requestScope(ctx context.Context, actorID uuid.UUID, branchID string) (q *gorm.DB, ok bool, err error) {
	allowed, all, err := middleware.GetAuthorizedBranchIDs(s.db, actorID)
	if err != nil {
		return nil, false, err
	}
	q = s.db.WithContext(ctx).Model(&models.Transaction{}).
		Where("transactions.type = ?", models.TxnTypeExpense).
		Where("(transactions.status IN ? OR "+wasApproved+")", []string{models.TxnStatusPending, models.TxnStatusRejected})
	if !all {
		if len(allowed) == 0 {
			return nil, false, nil
		}
		q = q.Where("transactions.branch_id IN ?", allowed)
	}
	if !s.hasPermission(ctx, actorID, permissions.RequestsView) && !s.hasPermission(ctx, actorID, permissions.RequestsApprove) {
		q = q.Where("transactions.created_by_id = ?", actorID)
	}
	if branchID != "" {
		bid, err := uuid.Parse(branchID)
		if err != nil {
			return nil, false, fmt.Errorf("invalid branch_id")
		}
		if !all && !containsID(allowed, bid) {
			return nil, false, ErrBranchAccessDenied
		}
		q = q.Where("transactions.branch_id = ?", bid)
	}
	return q, true, nil
}

// PendingRequestCount is the number on the sidebar's Requests item: the pending requests the actor
// sees (the Requests page's Pending chip with no filters), in one branch when branchID is set.
func (s *TransactionService) PendingRequestCount(ctx context.Context, actorID uuid.UUID, branchID string) (int64, error) {
	q, ok, err := s.requestScope(ctx, actorID, branchID)
	if err != nil || !ok {
		return 0, err
	}
	var n int64
	err = q.Where("transactions.status = ?", models.TxnStatusPending).Count(&n).Error
	return n, err
}

// ListRequests lists the requests in the actor's branches: pending and rejected expenses, and the
// approved ones that went through approval. Someone who may only make requests (not see or approve
// them) sees their own.
func (s *TransactionService) ListRequests(ctx context.Context, f RequestListFilter, actorID uuid.UUID) ([]models.Transaction, int64, RequestCounts, error) {
	var counts RequestCounts
	switch f.Status {
	case "", RequestStatusPending, RequestStatusApproved, RequestStatusRejected:
	default:
		return nil, 0, counts, fieldError("status", "choose pending, approved or rejected")
	}

	q, ok, err := s.requestScope(ctx, actorID, f.BranchID)
	if err != nil {
		return nil, 0, counts, err
	}
	if !ok {
		return []models.Transaction{}, 0, counts, nil
	}
	if f.Currency != "" {
		q = q.Where("transactions.currency = ?", f.Currency)
	}
	if f.CategoryID != "" {
		q = q.Where("transactions.category_id = ?", f.CategoryID)
	}
	if f.DateFrom != nil {
		q = q.Where("transactions.transaction_date >= ?", *f.DateFrom)
	}
	if f.DateTo != nil {
		q = q.Where("transactions.transaction_date < ?", *f.DateTo)
	}
	if strings.TrimSpace(f.Search) != "" {
		like := textsearch.Contains(f.Search)
		q = q.Where("(transactions.transaction_number LIKE ? OR transactions.description LIKE ?)", like, like)
	}

	// The chips' numbers: every filter but the status.
	var rows []struct {
		Status string
		N      int64
	}
	if err := q.Session(&gorm.Session{}).Select("transactions.status AS status, COUNT(*) AS n").Group("transactions.status").Scan(&rows).Error; err != nil {
		return nil, 0, counts, err
	}
	for _, r := range rows {
		counts.All += r.N
		switch r.Status {
		case models.TxnStatusPending:
			counts.Pending = r.N
		case models.TxnStatusApproved:
			counts.Approved = r.N
		case models.TxnStatusRejected:
			counts.Rejected = r.N
		}
	}

	if f.Status != "" {
		q = q.Where("transactions.status = ?", f.Status)
	}
	var total int64
	if err := q.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, counts, err
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
	err = withAttachments(q).Preload("Branch").Preload("Category").Preload("PaymentMethod").
		Order("transactions.transaction_date DESC, transactions.created_at DESC").
		Offset((page - 1) * size).Limit(size).
		Find(&items).Error
	return items, total, counts, err
}

// ApproveRequest approves a request: the expense counts from now on. It runs once: a request that
// was already decided (or is being decided by someone else at the same moment) is refused.
func (s *TransactionService) ApproveRequest(ctx context.Context, id uuid.UUID, notes string, actorID uuid.UUID) (*models.Transaction, error) {
	if err := s.assertRequest(ctx, id, actorID); err != nil {
		return nil, err
	}
	return s.Approve(ctx, id, notes, actorID)
}

// RejectRequest rejects a request with a reason; it stays on the Requests page and never counts.
func (s *TransactionService) RejectRequest(ctx context.Context, id uuid.UUID, reason string, actorID uuid.UUID) (*models.Transaction, error) {
	if strings.TrimSpace(reason) == "" {
		return nil, ErrReasonRequired
	}
	if err := s.assertRequest(ctx, id, actorID); err != nil {
		return nil, err
	}
	return s.Reject(ctx, id, strings.TrimSpace(reason), actorID)
}

func (s *TransactionService) assertRequest(ctx context.Context, id, actorID uuid.UUID) error {
	txn, err := s.GetByID(ctx, id, actorID)
	if err != nil {
		return err
	}
	if txn.Type != models.TxnTypeExpense {
		return ErrNotRequest
	}
	return nil
}

func containsID(list []uuid.UUID, id uuid.UUID) bool {
	for _, x := range list {
		if x == id {
			return true
		}
	}
	return false
}
