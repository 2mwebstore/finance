package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// Currency codes supported by the system.
const (
	CurrencyUSD = "USD"
	CurrencyKHR = "KHR"
)

// Transaction types.
const (
	TxnTypeIncome  = "income"
	TxnTypeExpense = "expense"
)

// Transaction statuses.
const (
	TxnStatusPending  = "pending"
	TxnStatusApproved = "approved"
	TxnStatusRejected = "rejected"
	TxnStatusVoided   = "voided"
	TxnStatusReversed = "reversed"
)

// Category groups income or expense classifications.
type Category struct {
	Base
	Code        string `gorm:"size:64;uniqueIndex:idx_category_type_code,priority:2;not null" json:"code"`
	Name        string `gorm:"size:128;not null" json:"name"`
	Type        string `gorm:"size:16;uniqueIndex:idx_category_type_code,priority:1;index;not null" json:"type"` // income | expense
	Description string `gorm:"size:512" json:"description"`
	IsActive    bool   `gorm:"default:true;not null" json:"is_active"`
	IsSystem    bool   `gorm:"default:false;not null" json:"is_system"`
	// BranchID is nil for shared categories (managed by owner/admin) and set for branch-only ones.
	BranchID *uuid.UUID `gorm:"type:char(36);index" json:"branch_id,omitempty"`
}

// PaymentMethod is a configurable payment channel (Cash, Bank, KHQR, etc.).
type PaymentMethod struct {
	Base
	Code     string `gorm:"size:64;uniqueIndex;not null" json:"code"`
	Name     string `gorm:"size:128;not null" json:"name"`
	IsActive bool   `gorm:"default:true;not null" json:"is_active"`
	IsSystem bool   `gorm:"default:false;not null" json:"is_system"`
}

// TransactionNumberSeq tracks per-prefix monthly sequences for safe concurrent numbering.
// Example prefix: INC-202610 → next number 1, 2, 3...
type TransactionNumberSeq struct {
	ID        uuid.UUID `gorm:"type:char(36);primaryKey" json:"id"`
	Prefix    string    `gorm:"size:32;uniqueIndex;not null" json:"prefix"` // e.g. INC-202610
	LastValue int64     `gorm:"not null;default:0" json:"last_value"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (t *TransactionNumberSeq) BeforeCreate(tx *gorm.DB) error {
	if t.ID == uuid.Nil {
		t.ID = uuid.New()
	}
	return nil
}

// Transaction is the core financial record. Amount is stored as DECIMAL via shopspring/decimal.
// Never use float64 for money.
type Transaction struct {
	Base
	TransactionNumber string          `gorm:"size:32;uniqueIndex;not null" json:"transaction_number"`
	Type              string          `gorm:"size:16;index;not null" json:"type"` // income | expense
	Status            string          `gorm:"size:16;index;not null;default:'approved'" json:"status"`
	BranchID          uuid.UUID       `gorm:"type:char(36);index;not null" json:"branch_id"`
	CategoryID        *uuid.UUID      `gorm:"type:char(36);index" json:"category_id,omitempty"`
	PaymentMethodID   uuid.UUID       `gorm:"type:char(36);index;not null" json:"payment_method_id"`
	Currency          string          `gorm:"size:3;index;not null" json:"currency"` // USD | KHR
	Amount            decimal.Decimal `gorm:"type:decimal(19,4);not null" json:"amount"`
	TransactionDate   time.Time       `gorm:"index;not null" json:"transaction_date"`
	Description       string          `gorm:"size:1024" json:"description"`
	Counterparty      string          `gorm:"size:255" json:"counterparty"` // payer or payee
	CreatedByID       uuid.UUID       `gorm:"type:char(36);index;not null" json:"created_by_id"`
	ApprovedByID      *uuid.UUID      `gorm:"type:char(36);index" json:"approved_by_id,omitempty"`
	ApprovedAt        *time.Time      `json:"approved_at,omitempty"`
	VoidedByID        *uuid.UUID      `gorm:"type:char(36);index" json:"voided_by_id,omitempty"`
	VoidedAt          *time.Time      `json:"voided_at,omitempty"`
	VoidReason        string          `gorm:"size:512" json:"void_reason,omitempty"`
	ReversedByTxnID   *uuid.UUID      `gorm:"type:char(36);index" json:"reversed_by_txn_id,omitempty"`
	ReversesTxnID     *uuid.UUID      `gorm:"type:char(36);index" json:"reverses_txn_id,omitempty"`

	Branch        *Branch        `gorm:"foreignKey:BranchID" json:"branch,omitempty"`
	Category      *Category      `gorm:"foreignKey:CategoryID" json:"category,omitempty"`
	PaymentMethod *PaymentMethod `gorm:"foreignKey:PaymentMethodID" json:"payment_method,omitempty"`
}

// IsFinalized returns true if this transaction counts toward financial totals.
func (t *Transaction) IsFinalized() bool {
	return t.Status == TxnStatusApproved
}

// Money helpers — precision rules
// USD: 2 decimal places typical for display; stored with up to 4
// KHR: 0 decimal places typical (whole riel); stored with up to 4

func ValidateCurrency(c string) bool {
	return c == CurrencyUSD || c == CurrencyKHR
}

func ValidateTxnType(t string) bool {
	return t == TxnTypeIncome || t == TxnTypeExpense
}

func ValidateAmount(currency string, amount decimal.Decimal) error {
	if !amount.IsPositive() {
		return gorm.ErrInvalidData // will wrap in service
	}
	// Reject more than 4 decimal places
	if amount.Exponent() < -4 {
		return gorm.ErrInvalidData
	}
	return nil
}

// OpeningBalance records starting balance per branch and currency.
type OpeningBalance struct {
	Base
	BranchID    uuid.UUID       `gorm:"type:char(36);uniqueIndex:idx_ob_branch_currency_date,priority:1;not null" json:"branch_id"`
	Currency    string          `gorm:"size:3;uniqueIndex:idx_ob_branch_currency_date,priority:2;not null" json:"currency"`
	Amount      decimal.Decimal `gorm:"type:decimal(19,4);not null" json:"amount"`
	BalanceDate time.Time       `gorm:"uniqueIndex:idx_ob_branch_currency_date,priority:3;not null" json:"balance_date"` // date only
	Notes       string          `gorm:"size:512" json:"notes"`
	CreatedByID uuid.UUID       `gorm:"type:char(36);not null" json:"created_by_id"`
	Status      string          `gorm:"size:16;not null;default:'approved'" json:"status"` // pending|approved
}

// CashReconciliation records expected vs counted cash per branch/currency/date.
type CashReconciliation struct {
	Base
	BranchID       uuid.UUID       `gorm:"type:char(36);index;not null" json:"branch_id"`
	Currency       string          `gorm:"size:3;index;not null" json:"currency"`
	ReconcileDate  time.Time       `gorm:"index;not null" json:"reconcile_date"`
	ExpectedAmount decimal.Decimal `gorm:"type:decimal(19,4);not null" json:"expected_amount"`
	ActualAmount   decimal.Decimal `gorm:"type:decimal(19,4);not null" json:"actual_amount"`
	Difference     decimal.Decimal `gorm:"type:decimal(19,4);not null" json:"difference"` // actual - expected
	Notes          string          `gorm:"size:1024" json:"notes"`
	CreatedByID    uuid.UUID       `gorm:"type:char(36);not null" json:"created_by_id"`
	Status         string          `gorm:"size:16;not null;default:'completed'" json:"status"`
}

// TransactionApproval stores approval/rejection history entries.
type TransactionApproval struct {
	ID            uuid.UUID `gorm:"type:char(36);primaryKey" json:"id"`
	TransactionID uuid.UUID `gorm:"type:char(36);index;not null" json:"transaction_id"`
	ActorID       uuid.UUID `gorm:"type:char(36);index;not null" json:"actor_id"`
	Action        string    `gorm:"size:16;not null" json:"action"` // submit|approve|reject
	Notes         string    `gorm:"size:512" json:"notes"`
	CreatedAt     time.Time `json:"created_at"`
}

func (a *TransactionApproval) BeforeCreate(tx *gorm.DB) error {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	return nil
}
