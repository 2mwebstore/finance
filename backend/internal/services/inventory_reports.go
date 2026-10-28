package services

import (
	"context"
	"time"

	"github.com/cambodia-finance/income-expense/internal/models"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// Inventory reports: stock value, sales and profit by product, purchases by product and
// supplier, expenses, and low stock. Everything is in USD: stock is valued in USD, and each sale,
// purchase or expense in KHR was converted at the rate of the day it was saved.

type InventoryReportFilter struct {
	BranchID   *uuid.UUID
	CategoryID *uuid.UUID
	From, To   *time.Time
}

// InventorySummary: the period's figures in USD (each document converted at its own rate).
type InventorySummary struct {
	Currency         string          `json:"currency"` // USD
	StockValue       decimal.Decimal `json:"stock_value"`
	SalesTotal       decimal.Decimal `json:"sales_total"`
	SalesCost        decimal.Decimal `json:"sales_cost"`
	Profit           decimal.Decimal `json:"profit"`
	SalesCount       int64           `json:"sales_count"`
	PurchasesTotal   decimal.Decimal `json:"purchases_total"`
	PurchasesCount   int64           `json:"purchases_count"`
	PendingPurchases int64           `json:"pending_purchases"`
	ExpensesTotal    decimal.Decimal `json:"expenses_total"`
	ExpensesCount    int64           `json:"expenses_count"`
	NetProfit        decimal.Decimal `json:"net_profit"` // profit − inventory expenses
	Products         int64           `json:"products"`
	LowStock         int64           `json:"low_stock"`
}

func (s *InventoryService) branchFilter(q *gorm.DB, column string, f InventoryReportFilter, scope BranchScope) *gorm.DB {
	q = scope.where(q, column)
	if f.BranchID != nil {
		q = q.Where(column+" = ?", *f.BranchID)
	}
	return q
}

func dateFilter(q *gorm.DB, column string, f InventoryReportFilter) *gorm.DB {
	if f.From != nil {
		q = q.Where(column+" >= ?", *f.From)
	}
	if f.To != nil {
		q = q.Where(column+" <= ?", *f.To)
	}
	return q
}

func (s *InventoryService) Summary(ctx context.Context, f InventoryReportFilter, scope BranchScope) (*InventorySummary, error) {
	out := &InventorySummary{Currency: models.CurrencyUSD}
	type sums struct {
		Amount decimal.Decimal
		Cost   decimal.Decimal
		Count  int64
	}
	var r sums

	q := s.db.WithContext(ctx).Table("stock_levels").Joins("JOIN products ON products.id = stock_levels.product_id AND products.deleted_at IS NULL")
	if err := s.branchFilter(q, "stock_levels.branch_id", f, scope).
		Select("COALESCE(SUM(stock_levels.quantity * stock_levels.avg_cost), 0) AS amount").Scan(&r).Error; err != nil {
		return nil, err
	}
	out.StockValue = r.Amount.Round(4)

	r = sums{}
	q = dateFilter(s.branchFilter(s.db.WithContext(ctx).Model(&models.Sale{}), "branch_id", f, scope), "sale_date", f).Where("status = ?", models.DocCompleted)
	if err := q.Select("COALESCE(SUM(total_base), 0) AS amount, COALESCE(SUM(total_cost), 0) AS cost, COUNT(*) AS count").Scan(&r).Error; err != nil {
		return nil, err
	}
	out.SalesTotal, out.SalesCost, out.SalesCount, out.Profit = r.Amount, r.Cost, r.Count, r.Amount.Sub(r.Cost)

	r = sums{}
	q = dateFilter(s.branchFilter(s.db.WithContext(ctx).Model(&models.Purchase{}), "branch_id", f, scope), "purchase_date", f).Where("status = ?", models.DocApproved)
	if err := q.Select("COALESCE(SUM(total_base), 0) AS amount, COUNT(*) AS count").Scan(&r).Error; err != nil {
		return nil, err
	}
	out.PurchasesTotal, out.PurchasesCount = r.Amount, r.Count

	if err := s.branchFilter(s.db.WithContext(ctx).Model(&models.Purchase{}), "branch_id", f, scope).Where("status = ?", models.DocPending).Count(&out.PendingPurchases).Error; err != nil {
		return nil, err
	}

	r = sums{}
	q = dateFilter(s.branchFilter(s.db.WithContext(ctx).Model(&models.InventoryExpense{}), "branch_id", f, scope), "expense_date", f).Where("status = ?", models.DocCompleted)
	if err := q.Select("COALESCE(SUM(amount_base), 0) AS amount, COUNT(*) AS count").Scan(&r).Error; err != nil {
		return nil, err
	}
	out.ExpensesTotal, out.ExpensesCount = r.Amount, r.Count
	out.NetProfit = out.Profit.Sub(out.ExpensesTotal)

	if err := s.db.WithContext(ctx).Model(&models.Product{}).Where("is_active = ?", true).Count(&out.Products).Error; err != nil {
		return nil, err
	}
	if err := s.stockQuery(ctx, StockFilter{BranchID: f.BranchID, LowOnly: true}, scope).Count(&out.LowStock).Error; err != nil {
		return nil, err
	}
	return out, nil
}

// StockReportRow is a product's stock added up over the chosen branches.
type StockReportRow struct {
	ProductID    uuid.UUID       `json:"product_id"`
	SKU          string          `json:"sku"`
	Name         string          `json:"name"`
	Unit         string          `json:"unit"`
	Currency     string          `json:"currency"`
	CategoryName string          `json:"category_name"`
	Quantity     decimal.Decimal `json:"quantity"`
	Value        decimal.Decimal `json:"value"`
	AvgCost      decimal.Decimal `json:"avg_cost"`
	SalePrice    decimal.Decimal `json:"sale_price"`
	Branches     int64           `json:"branches"`
}

func (s *InventoryService) StockReport(ctx context.Context, f InventoryReportFilter, scope BranchScope) ([]StockReportRow, error) {
	q := s.db.WithContext(ctx).Table("stock_levels").
		Joins("JOIN products ON products.id = stock_levels.product_id AND products.deleted_at IS NULL").
		Joins("LEFT JOIN product_categories ON product_categories.id = products.category_id")
	q = s.branchFilter(q, "stock_levels.branch_id", f, scope)
	if f.CategoryID != nil {
		q = q.Where("products.category_id = ?", *f.CategoryID)
	}
	var rows []StockReportRow
	err := q.Select(`stock_levels.product_id, products.sku, products.name, products.unit, products.currency,
		COALESCE(product_categories.name, '') AS category_name, SUM(stock_levels.quantity) AS quantity,
		SUM(stock_levels.quantity * stock_levels.avg_cost) AS value, products.sale_price,
		SUM(CASE WHEN stock_levels.quantity > 0 THEN 1 ELSE 0 END) AS branches`).
		Group("stock_levels.product_id, products.sku, products.name, products.unit, products.currency, product_categories.name, products.sale_price").
		Order("products.name ASC").Scan(&rows).Error
	for i := range rows {
		rows[i].Value = rows[i].Value.Round(4)
		if rows[i].Quantity.IsPositive() {
			rows[i].AvgCost = rows[i].Value.Div(rows[i].Quantity).Round(4)
		}
	}
	return rows, err
}

// SalesReportRow: one product's sales in the period (voided sales left out), after line and
// sale discounts; shipping charged isn't split by product.
type SalesReportRow struct {
	ProductID uuid.UUID       `json:"product_id"`
	SKU       string          `json:"sku"`
	Name      string          `json:"name"`
	Unit      string          `json:"unit"`
	Currency  string          `json:"currency"`
	Quantity  decimal.Decimal `json:"quantity"`
	Revenue   decimal.Decimal `json:"revenue"`
	Cost      decimal.Decimal `json:"cost"`
	Profit    decimal.Decimal `json:"profit"`
	Margin    decimal.Decimal `json:"margin"` // profit ÷ revenue, in percent
}

func (s *InventoryService) SalesReport(ctx context.Context, f InventoryReportFilter, scope BranchScope) ([]SalesReportRow, error) {
	q := s.db.WithContext(ctx).Table("sale_items").
		Joins("JOIN sales ON sales.id = sale_items.sale_id AND sales.deleted_at IS NULL").
		Joins("JOIN products ON products.id = sale_items.product_id").
		Where("sales.status = ?", models.DocCompleted)
	q = dateFilter(s.branchFilter(q, "sales.branch_id", f, scope), "sales.sale_date", f)
	if f.CategoryID != nil {
		q = q.Where("products.category_id = ?", *f.CategoryID)
	}
	var rows []SalesReportRow
	err := q.Select(`sale_items.product_id, products.sku, products.name, products.unit, 'USD' AS currency,
		SUM(sale_items.quantity) AS quantity, SUM(sale_items.net_total_base) AS revenue,
		SUM(sale_items.quantity * sale_items.unit_cost) AS cost`).
		Group("sale_items.product_id, products.sku, products.name, products.unit").
		Order("revenue DESC").Scan(&rows).Error
	for i := range rows {
		rows[i].Cost = rows[i].Cost.Round(4)
		rows[i].Profit = rows[i].Revenue.Sub(rows[i].Cost)
		if rows[i].Revenue.IsPositive() {
			rows[i].Margin = rows[i].Profit.Div(rows[i].Revenue).Mul(decimal.NewFromInt(100)).Round(1)
		}
	}
	return rows, err
}

// PurchaseReportRow: approved purchases in the period, by product or by supplier.
type PurchaseReportRow struct {
	Key      string          `json:"key"` // product id, or the supplier name
	SKU      string          `json:"sku,omitempty"`
	Name     string          `json:"name"`
	Unit     string          `json:"unit,omitempty"`
	Currency string          `json:"currency"`
	Quantity decimal.Decimal `json:"quantity"`
	Total    decimal.Decimal `json:"total"`
	Count    int64           `json:"count"` // purchases
	AvgCost  decimal.Decimal `json:"avg_cost"`
}

func (s *InventoryService) PurchasesReport(ctx context.Context, f InventoryReportFilter, bySupplier bool, scope BranchScope) ([]PurchaseReportRow, error) {
	var rows []PurchaseReportRow
	if bySupplier {
		q := s.db.WithContext(ctx).Model(&models.Purchase{}).Where("status = ?", models.DocApproved)
		q = dateFilter(s.branchFilter(q, "branch_id", f, scope), "purchase_date", f)
		err := q.Select(`COALESCE(NULLIF(supplier, ''), '(no supplier)') AS ` + "`key`" + `, COALESCE(NULLIF(supplier, ''), '(no supplier)') AS name,
			'USD' AS currency, SUM(total_base) AS total, COUNT(*) AS count`).
			Group("COALESCE(NULLIF(supplier, ''), '(no supplier)')").Order("total DESC").Scan(&rows).Error
		return rows, err
	}
	q := s.db.WithContext(ctx).Table("purchase_items").
		Joins("JOIN purchases ON purchases.id = purchase_items.purchase_id AND purchases.deleted_at IS NULL").
		Joins("JOIN products ON products.id = purchase_items.product_id").
		Where("purchases.status = ?", models.DocApproved)
	q = dateFilter(s.branchFilter(q, "purchases.branch_id", f, scope), "purchases.purchase_date", f)
	if f.CategoryID != nil {
		q = q.Where("products.category_id = ?", *f.CategoryID)
	}
	err := q.Select(`purchase_items.product_id AS ` + "`key`" + `, products.sku, products.name, products.unit, 'USD' AS currency,
		SUM(purchase_items.quantity) AS quantity, SUM(purchase_items.quantity * purchase_items.landed_cost) AS total, COUNT(DISTINCT purchases.id) AS count`).
		Group("purchase_items.product_id, products.sku, products.name, products.unit").
		Order("total DESC").Scan(&rows).Error
	for i := range rows {
		rows[i].Total = rows[i].Total.Round(4)
		if rows[i].Quantity.IsPositive() {
			rows[i].AvgCost = rows[i].Total.Div(rows[i].Quantity).Round(4)
		}
	}
	return rows, err
}

// LowStock lists stock at or under the product's low-stock level, per branch.
func (s *InventoryService) LowStock(ctx context.Context, f InventoryReportFilter, scope BranchScope) ([]StockRow, error) {
	var rows []StockRow
	err := s.stockQuery(ctx, StockFilter{BranchID: f.BranchID, CategoryID: f.CategoryID, LowOnly: true}, scope).
		Select(stockSelect).Order("stock_levels.quantity ASC, products.name ASC").Limit(500).Scan(&rows).Error
	return finishStockRows(rows), err
}
