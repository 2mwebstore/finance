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
	ErrOpeningBalanceExists   = errors.New("opening balance already exists for this branch, currency, and date")
	ErrReconciliationNotFound = errors.New("reconciliation not found")
)

type CashService struct {
	db *gorm.DB
}

func NewCashService(db *gorm.DB) *CashService {
	return &CashService{db: db}
}

type CreateOpeningBalanceInput struct {
	BranchID    string `json:"branch_id" binding:"required,uuid"`
	Currency    string `json:"currency" binding:"required,oneof=USD KHR"`
	Amount      string `json:"amount" binding:"required"`
	BalanceDate string `json:"balance_date" binding:"required"`
	Notes       string `json:"notes"`
}

func (s *CashService) CreateOpeningBalance(ctx context.Context, in CreateOpeningBalanceInput, actorID uuid.UUID) (*models.OpeningBalance, error) {
	if err := accounting.ValidateCurrency(in.Currency); err != nil {
		return nil, err
	}
	amount, err := parseNonNegativeAmount(in.Amount)
	if err != nil {
		return nil, err
	}
	branchID, err := uuid.Parse(in.BranchID)
	if err != nil {
		return nil, fmt.Errorf("invalid branch_id")
	}
	if err := (&TransactionService{db: s.db}).assertBranchAccess(ctx, actorID, branchID); err != nil {
		return nil, err
	}
	balDate, err := time.ParseInLocation("2006-01-02", in.BalanceDate, time.FixedZone("Asia/Phnom_Penh", 7*3600))
	if err != nil {
		return nil, fmt.Errorf("invalid balance_date")
	}
	var existing models.OpeningBalance
	err = s.db.WithContext(ctx).Where("branch_id = ? AND currency = ? AND balance_date = ?", branchID, in.Currency, balDate).First(&existing).Error
	if err == nil {
		return nil, ErrOpeningBalanceExists
	}
	ob := models.OpeningBalance{
		Base: models.Base{ID: uuid.New()}, BranchID: branchID, Currency: in.Currency,
		Amount: amount, BalanceDate: balDate, Notes: in.Notes, CreatedByID: actorID,
		Status: models.TxnStatusApproved,
	}
	if err := s.db.WithContext(ctx).Create(&ob).Error; err != nil {
		return nil, err
	}
	return &ob, nil
}

func (s *CashService) ListOpeningBalances(ctx context.Context, actorID uuid.UUID, branchID *uuid.UUID) ([]models.OpeningBalance, error) {
	allowed, all, err := middleware.GetAuthorizedBranchIDs(s.db, actorID)
	if err != nil {
		return nil, err
	}
	q := s.db.WithContext(ctx).Model(&models.OpeningBalance{})
	if !all {
		if len(allowed) == 0 {
			return []models.OpeningBalance{}, nil
		}
		q = q.Where("branch_id IN ?", allowed)
	}
	if branchID != nil {
		q = q.Where("branch_id = ?", *branchID)
	}
	var items []models.OpeningBalance
	err = q.Order("balance_date DESC").Find(&items).Error
	return items, err
}

type CreateReconciliationInput struct {
	BranchID       string `json:"branch_id" binding:"required,uuid"`
	Currency       string `json:"currency" binding:"required,oneof=USD KHR"`
	ReconcileDate  string `json:"reconcile_date" binding:"required"`
	ExpectedAmount string `json:"expected_amount" binding:"required"`
	ActualAmount   string `json:"actual_amount" binding:"required"`
	Notes          string `json:"notes"`
}

func (s *CashService) CreateReconciliation(ctx context.Context, in CreateReconciliationInput, actorID uuid.UUID) (*models.CashReconciliation, error) {
	if err := accounting.ValidateCurrency(in.Currency); err != nil {
		return nil, err
	}
	expected, err := parseNonNegativeAmount(in.ExpectedAmount)
	if err != nil {
		return nil, err
	}
	actual, err := parseNonNegativeAmount(in.ActualAmount)
	if err != nil {
		return nil, err
	}
	branchID, err := uuid.Parse(in.BranchID)
	if err != nil {
		return nil, fmt.Errorf("invalid branch_id")
	}
	if err := (&TransactionService{db: s.db}).assertBranchAccess(ctx, actorID, branchID); err != nil {
		return nil, err
	}
	recDate, err := time.ParseInLocation("2006-01-02", in.ReconcileDate, time.FixedZone("Asia/Phnom_Penh", 7*3600))
	if err != nil {
		return nil, fmt.Errorf("invalid reconcile_date")
	}
	diff := actual.Sub(expected)
	cr := models.CashReconciliation{
		Base: models.Base{ID: uuid.New()}, BranchID: branchID, Currency: in.Currency,
		ReconcileDate: recDate, ExpectedAmount: expected, ActualAmount: actual,
		Difference: diff, Notes: in.Notes, CreatedByID: actorID, Status: "completed",
	}
	if err := s.db.WithContext(ctx).Create(&cr).Error; err != nil {
		return nil, err
	}
	return &cr, nil
}

func (s *CashService) ListReconciliations(ctx context.Context, actorID uuid.UUID, branchID *uuid.UUID) ([]models.CashReconciliation, error) {
	allowed, all, err := middleware.GetAuthorizedBranchIDs(s.db, actorID)
	if err != nil {
		return nil, err
	}
	q := s.db.WithContext(ctx).Model(&models.CashReconciliation{})
	if !all {
		if len(allowed) == 0 {
			return []models.CashReconciliation{}, nil
		}
		q = q.Where("branch_id IN ?", allowed)
	}
	if branchID != nil {
		q = q.Where("branch_id = ?", *branchID)
	}
	var items []models.CashReconciliation
	err = q.Order("reconcile_date DESC").Find(&items).Error
	return items, err
}

func parseNonNegativeAmount(s string) (decimal.Decimal, error) {
	d, err := decimal.NewFromString(s)
	if err != nil {
		return decimal.Zero, accounting.ErrInvalidAmount
	}
	if d.IsNegative() {
		return decimal.Zero, accounting.ErrInvalidAmount
	}
	if d.Exponent() < -4 {
		return decimal.Zero, accounting.ErrInvalidAmount
	}
	if d.GreaterThan(accounting.MaxAmount) {
		return decimal.Zero, accounting.ErrAmountTooLarge
	}
	return d, nil
}
