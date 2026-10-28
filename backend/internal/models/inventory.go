package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// Inventory: a shared product catalog, stock kept per branch at average cost, and the documents
// that move stock (sales, purchases, transfers between branches, adjustments). It is separate
// from the income/expense ledger.
//
// The tables only point at older tables by ID (no foreign keys to them), so creating them on a
// running database can never fail on an older table's collation.

// ProductCategory groups products (shared by every branch).
type ProductCategory struct {
	Base
	Code        string `gorm:"size:64;uniqueIndex;not null" json:"code"`
	Name        string `gorm:"size:128;not null" json:"name"`
	Description string `gorm:"size:512" json:"description"`
	IsActive    bool   `gorm:"default:true;not null" json:"is_active"`
}

// Product is an item the business buys and sells (shared by every branch). Prices and costs are
// in USD (the base currency); a sale or purchase in KHR converts them at the exchange rate. The
// SKU is generated.
// Its picture is an uploaded file (ImageKey, in R2 or local storage) or a web address (ImageURL).
type Product struct {
	Base
	SKU          string          `gorm:"column:sku;size:64;uniqueIndex;not null" json:"sku"`
	ImageKey     string          `gorm:"size:255" json:"-"`
	ImageType    string          `gorm:"size:64" json:"-"`
	ImageURL     string          `gorm:"size:1024" json:"image_url"`
	Name         string          `gorm:"size:191;index;not null" json:"name"`
	CategoryID   *uuid.UUID      `gorm:"type:char(36);index" json:"category_id,omitempty"`
	Unit         string          `gorm:"size:32;not null;default:'pcs'" json:"unit"`
	Barcode      string          `gorm:"size:64;index" json:"barcode"`
	Currency     string          `gorm:"size:3;not null" json:"currency"`
	SalePrice    decimal.Decimal `gorm:"type:decimal(19,4);not null;default:0" json:"sale_price"`
	CostPrice    decimal.Decimal `gorm:"type:decimal(19,4);not null;default:0" json:"cost_price"` // usual purchase cost (a suggestion)
	ReorderLevel decimal.Decimal `gorm:"type:decimal(19,4);not null;default:0" json:"reorder_level"`
	Description  string          `gorm:"size:512" json:"description"`
	IsActive     bool            `gorm:"default:true;not null" json:"is_active"`
}

// StockLevel is a product's stock in one branch, valued at average cost.
type StockLevel struct {
	ID        uuid.UUID       `gorm:"type:char(36);primaryKey" json:"id"`
	BranchID  uuid.UUID       `gorm:"type:char(36);not null;uniqueIndex:idx_stock_branch_product,priority:1" json:"branch_id"`
	ProductID uuid.UUID       `gorm:"type:char(36);not null;uniqueIndex:idx_stock_branch_product,priority:2;index" json:"product_id"`
	Quantity  decimal.Decimal `gorm:"type:decimal(19,4);not null;default:0" json:"quantity"`
	AvgCost   decimal.Decimal `gorm:"type:decimal(19,4);not null;default:0" json:"avg_cost"`
	UpdatedAt time.Time       `json:"updated_at"`
}

// Stock movement types.
const (
	MoveSale         = "sale"
	MoveSaleVoid     = "sale_void"
	MovePurchase     = "purchase"
	MovePurchaseVoid = "purchase_void"
	MoveTransferOut  = "transfer_out"
	MoveTransferIn   = "transfer_in"
	MoveTransferVoid = "transfer_void"
	MoveAdjustment   = "adjustment"
)

// StockMovement is one change to a branch's stock of a product (the stock history).
type StockMovement struct {
	ID           uuid.UUID       `gorm:"type:char(36);primaryKey" json:"id"`
	BranchID     uuid.UUID       `gorm:"type:char(36);index:idx_move_branch_product,priority:1;not null" json:"branch_id"`
	ProductID    uuid.UUID       `gorm:"type:char(36);index:idx_move_branch_product,priority:2;index;not null" json:"product_id"`
	Type         string          `gorm:"size:24;index;not null" json:"type"`
	Quantity     decimal.Decimal `gorm:"type:decimal(19,4);not null" json:"quantity"` // + in, − out
	UnitCost     decimal.Decimal `gorm:"type:decimal(19,4);not null;default:0" json:"unit_cost"`
	BalanceAfter decimal.Decimal `gorm:"type:decimal(19,4);not null" json:"balance_after"`
	RefType      string          `gorm:"size:16;index:idx_move_ref,priority:1" json:"ref_type"` // sale | purchase | transfer | adjustment
	RefID        *uuid.UUID      `gorm:"type:char(36);index:idx_move_ref,priority:2" json:"ref_id,omitempty"`
	RefNumber    string          `gorm:"size:32" json:"ref_number"`
	Note         string          `gorm:"size:512" json:"note"`
	CreatedByID  uuid.UUID       `gorm:"type:char(36);not null" json:"created_by_id"`
	CreatedAt    time.Time       `gorm:"index" json:"created_at"`
}

// Purchase statuses (sales and transfers are completed or voided).
const (
	DocPending   = "pending"
	DocApproved  = "approved"
	DocRejected  = "rejected"
	DocCompleted = "completed"
	DocVoided    = "voided"
)

// Purchase is goods bought from a supplier for a branch. Stock goes in once it is approved.
type Purchase struct {
	Base
	Number       string    `gorm:"size:32;uniqueIndex;not null" json:"number"`
	BranchID     uuid.UUID `gorm:"type:char(36);index;not null" json:"branch_id"`
	Supplier     string    `gorm:"size:191;index" json:"supplier"`
	PurchaseDate time.Time `gorm:"index;not null" json:"purchase_date"`
	Currency     string    `gorm:"size:3;not null" json:"currency"`
	// Subtotal (lines after their discounts) − Discount + Shipping = Total. Discount and shipping
	// are spread over the lines into their landed cost.
	Subtotal        decimal.Decimal `gorm:"type:decimal(19,4);not null;default:0" json:"subtotal"`
	Discount        decimal.Decimal `gorm:"type:decimal(19,4);not null;default:0" json:"discount"`
	Shipping        decimal.Decimal `gorm:"type:decimal(19,4);not null;default:0" json:"shipping"`
	Total           decimal.Decimal `gorm:"type:decimal(19,4);not null" json:"total"`
	ExchangeRate    decimal.Decimal `gorm:"type:decimal(19,4);not null;default:0" json:"exchange_rate"` // units of Currency per 1 USD when saved
	TotalBase       decimal.Decimal `gorm:"type:decimal(19,4);not null;default:0" json:"total_base"`    // Total in USD
	PaymentMethodID *uuid.UUID      `gorm:"type:char(36)" json:"payment_method_id,omitempty"`
	Status          string          `gorm:"size:16;index;not null" json:"status"`
	Note            string          `gorm:"type:text" json:"note"` // sanitized rich text
	CreatedByID     uuid.UUID       `gorm:"type:char(36);index;not null" json:"created_by_id"`
	DecidedByID     *uuid.UUID      `gorm:"type:char(36)" json:"decided_by_id,omitempty"` // approved or rejected by
	DecidedAt       *time.Time      `json:"decided_at,omitempty"`
	DecisionNote    string          `gorm:"size:512" json:"decision_note"`
	VoidedByID      *uuid.UUID      `gorm:"type:char(36)" json:"voided_by_id,omitempty"`
	VoidedAt        *time.Time      `json:"voided_at,omitempty"`
	VoidReason      string          `gorm:"size:512" json:"void_reason"`
	Items           []PurchaseItem  `gorm:"foreignKey:PurchaseID" json:"items,omitempty"`
}

type PurchaseItem struct {
	ID         uuid.UUID       `gorm:"type:char(36);primaryKey" json:"id"`
	PurchaseID uuid.UUID       `gorm:"type:char(36);index;not null" json:"purchase_id"`
	ProductID  uuid.UUID       `gorm:"type:char(36);index;not null" json:"product_id"`
	Quantity   decimal.Decimal `gorm:"type:decimal(19,4);not null" json:"quantity"`
	UnitCost   decimal.Decimal `gorm:"type:decimal(19,4);not null" json:"unit_cost"`
	Discount   decimal.Decimal `gorm:"type:decimal(19,4);not null;default:0" json:"discount"`
	LineTotal  decimal.Decimal `gorm:"type:decimal(19,4);not null" json:"line_total"` // quantity × unit cost − discount
	// LandedCost is the unit cost (in USD) that goes into stock: the line with its share of the
	// purchase's discount and shipping, per unit.
	LandedCost decimal.Decimal `gorm:"type:decimal(19,4);not null;default:0" json:"landed_cost"`
}

// Sale is goods sold from a branch. Stock goes out when it is saved.
type Sale struct {
	Base
	Number          string     `gorm:"size:32;uniqueIndex;not null" json:"number"`
	BranchID        uuid.UUID  `gorm:"type:char(36);index;not null" json:"branch_id"`
	Customer        string     `gorm:"size:191;index" json:"customer"`
	SaleDate        time.Time  `gorm:"index;not null" json:"sale_date"`
	Currency        string     `gorm:"size:3;not null" json:"currency"`
	PaymentMethodID *uuid.UUID `gorm:"type:char(36)" json:"payment_method_id,omitempty"`
	// Subtotal (lines after their discounts) − Discount + Shipping (charged to the customer) = Total.
	Subtotal     decimal.Decimal `gorm:"type:decimal(19,4);not null;default:0" json:"subtotal"`
	Discount     decimal.Decimal `gorm:"type:decimal(19,4);not null;default:0" json:"discount"`
	Shipping     decimal.Decimal `gorm:"type:decimal(19,4);not null;default:0" json:"shipping"`
	Total        decimal.Decimal `gorm:"type:decimal(19,4);not null" json:"total"`
	ExchangeRate decimal.Decimal `gorm:"type:decimal(19,4);not null;default:0" json:"exchange_rate"` // units of Currency per 1 USD when saved
	TotalBase    decimal.Decimal `gorm:"type:decimal(19,4);not null;default:0" json:"total_base"`    // Total in USD
	TotalCost    decimal.Decimal `gorm:"type:decimal(19,4);not null" json:"total_cost"`              // USD: average cost of the goods sold
	Status       string          `gorm:"size:16;index;not null" json:"status"`
	Note         string          `gorm:"type:text" json:"note"` // sanitized rich text
	CreatedByID  uuid.UUID       `gorm:"type:char(36);index;not null" json:"created_by_id"`
	VoidedByID   *uuid.UUID      `gorm:"type:char(36)" json:"voided_by_id,omitempty"`
	VoidedAt     *time.Time      `json:"voided_at,omitempty"`
	VoidReason   string          `gorm:"size:512" json:"void_reason"`
	Items        []SaleItem      `gorm:"foreignKey:SaleID" json:"items,omitempty"`
}

type SaleItem struct {
	ID        uuid.UUID       `gorm:"type:char(36);primaryKey" json:"id"`
	SaleID    uuid.UUID       `gorm:"type:char(36);index;not null" json:"sale_id"`
	ProductID uuid.UUID       `gorm:"type:char(36);index;not null" json:"product_id"`
	Quantity  decimal.Decimal `gorm:"type:decimal(19,4);not null" json:"quantity"`
	UnitPrice decimal.Decimal `gorm:"type:decimal(19,4);not null" json:"unit_price"`
	UnitCost  decimal.Decimal `gorm:"type:decimal(19,4);not null" json:"unit_cost"` // USD: average cost when sold
	Discount  decimal.Decimal `gorm:"type:decimal(19,4);not null;default:0" json:"discount"`
	LineTotal decimal.Decimal `gorm:"type:decimal(19,4);not null" json:"line_total"` // quantity × unit price − discount
	// NetTotal is the line after its share of the sale's discount (for sales by product).
	NetTotal     decimal.Decimal `gorm:"type:decimal(19,4);not null;default:0" json:"net_total"`
	NetTotalBase decimal.Decimal `gorm:"type:decimal(19,4);not null;default:0" json:"net_total_base"` // NetTotal in USD
}

// StockTransfer moves stock from one branch to another (at the sending branch's average cost).
type StockTransfer struct {
	Base
	Number       string              `gorm:"size:32;uniqueIndex;not null" json:"number"`
	FromBranchID uuid.UUID           `gorm:"type:char(36);index;not null" json:"from_branch_id"`
	ToBranchID   uuid.UUID           `gorm:"type:char(36);index;not null" json:"to_branch_id"`
	TransferDate time.Time           `gorm:"index;not null" json:"transfer_date"`
	Status       string              `gorm:"size:16;index;not null" json:"status"`
	Note         string              `gorm:"type:text" json:"note"` // sanitized rich text
	CreatedByID  uuid.UUID           `gorm:"type:char(36);index;not null" json:"created_by_id"`
	VoidedByID   *uuid.UUID          `gorm:"type:char(36)" json:"voided_by_id,omitempty"`
	VoidedAt     *time.Time          `json:"voided_at,omitempty"`
	VoidReason   string              `gorm:"size:512" json:"void_reason"`
	Items        []StockTransferItem `gorm:"foreignKey:TransferID" json:"items,omitempty"`
}

type StockTransferItem struct {
	ID         uuid.UUID       `gorm:"type:char(36);primaryKey" json:"id"`
	TransferID uuid.UUID       `gorm:"type:char(36);index;not null" json:"transfer_id"`
	ProductID  uuid.UUID       `gorm:"type:char(36);index;not null" json:"product_id"`
	Quantity   decimal.Decimal `gorm:"type:decimal(19,4);not null" json:"quantity"`
	UnitCost   decimal.Decimal `gorm:"type:decimal(19,4);not null" json:"unit_cost"`
}

// CategoryTypeInventoryExpense is the category type of inventory expenses (kept apart from the
// income/expense categories).
const CategoryTypeInventoryExpense = "inv_expense"

// InventoryExpense is a cost of running the inventory side (shipping, packaging, storage…), in
// one of the inventory expense categories. Saved at once; voiding cancels it. Reports take it
// off the sales profit (net profit).
type InventoryExpense struct {
	Base
	Number          string          `gorm:"size:32;uniqueIndex;not null" json:"number"`
	BranchID        uuid.UUID       `gorm:"type:char(36);index;not null" json:"branch_id"`
	CategoryID      uuid.UUID       `gorm:"type:char(36);index;not null" json:"category_id"`
	ExpenseDate     time.Time       `gorm:"index;not null" json:"expense_date"`
	Currency        string          `gorm:"size:3;not null" json:"currency"`
	Amount          decimal.Decimal `gorm:"type:decimal(19,4);not null" json:"amount"`
	ExchangeRate    decimal.Decimal `gorm:"type:decimal(19,4);not null;default:0" json:"exchange_rate"` // units of Currency per 1 USD when saved
	AmountBase      decimal.Decimal `gorm:"type:decimal(19,4);not null;default:0" json:"amount_base"`   // Amount in USD
	PaymentMethodID *uuid.UUID      `gorm:"type:char(36)" json:"payment_method_id,omitempty"`
	Payee           string          `gorm:"size:191" json:"payee"`
	Note            string          `gorm:"type:text" json:"note"` // sanitized rich text
	Status          string          `gorm:"size:16;index;not null" json:"status"`
	CreatedByID     uuid.UUID       `gorm:"type:char(36);index;not null" json:"created_by_id"`
	VoidedByID      *uuid.UUID      `gorm:"type:char(36)" json:"voided_by_id,omitempty"`
	VoidedAt        *time.Time      `json:"voided_at,omitempty"`
	VoidReason      string          `gorm:"size:512" json:"void_reason"`
}

// InventoryModels lists the inventory tables (for AutoMigrate and the start-up check).
func InventoryModels() []interface{} {
	return []interface{}{
		&ProductCategory{}, &Product{}, &StockLevel{}, &StockMovement{},
		&Purchase{}, &PurchaseItem{}, &Sale{}, &SaleItem{}, &StockTransfer{}, &StockTransferItem{},
		&InventoryExpense{},
	}
}

// InventoryColumns are columns added after the inventory tables first shipped; when one is
// missing the start-up check brings the tables up to date.
var InventoryColumns = map[interface{}][]string{
	&Product{}:          {"image_key", "image_type", "image_url"},
	&Purchase{}:         {"subtotal", "discount", "shipping", "payment_method_id", "exchange_rate", "total_base"},
	&PurchaseItem{}:     {"discount", "landed_cost"},
	&Sale{}:             {"subtotal", "discount", "shipping", "exchange_rate", "total_base"},
	&SaleItem{}:         {"discount", "net_total", "net_total_base"},
	&InventoryExpense{}: {"exchange_rate", "amount_base"},
}
