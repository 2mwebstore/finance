package services

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/cambodia-finance/income-expense/internal/models"
	"github.com/cambodia-finance/income-expense/internal/permissions"
	"github.com/cambodia-finance/income-expense/internal/textsearch"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Documents that move stock: purchases (stock in once approved), sales (stock out when saved)
// and transfers between branches. Voiding one puts the stock back as it was.

// DocItemInput is one line of a sale, purchase or transfer.
type DocItemInput struct {
	ProductID string `json:"product_id" binding:"required,uuid"`
	Quantity  string `json:"quantity" binding:"required"`
	UnitPrice string `json:"unit_price"`             // sales
	UnitCost  string `json:"unit_cost"`              // purchases
	Discount  string `json:"discount"`               // off this line, an amount (sales and purchases)
	Note      string `json:"note" binding:"max=255"` // sales: a note on the line
}

// DocLineView is a document line with the product's name.
type DocLineView struct {
	ProductID  uuid.UUID       `json:"product_id"`
	SKU        string          `json:"sku"`
	Name       string          `json:"name"`
	Unit       string          `json:"unit"`
	Quantity   decimal.Decimal `json:"quantity"`
	UnitPrice  decimal.Decimal `json:"unit_price"`
	UnitCost   decimal.Decimal `json:"unit_cost"`
	Discount   decimal.Decimal `json:"discount"`
	LineTotal  decimal.Decimal `json:"line_total"`
	LandedCost decimal.Decimal `json:"landed_cost"`    // purchases: unit cost with shipping and discounts
	Note       string          `json:"note,omitempty"` // sales
}

type DocFilter struct {
	BranchID *uuid.UUID
	Status   string
	Search   string
	From, To *time.Time
	Page     int
	PageSize int
}

// parsedLine is a checked input line.
type parsedLine struct {
	product   models.Product
	qty       decimal.Decimal
	amount    decimal.Decimal // unit price (sale) or unit cost (purchase)
	discount  decimal.Decimal // off the line
	lineTotal decimal.Decimal // qty × amount − discount
	note      string
}

// parseLines checks the lines: active products, quantities above zero, and prices or costs (in
// the document's currency) of zero or more. Services are only sold: purchases and transfers
// (amountField unit_cost) can't have them.
func (s *InventoryService) parseLines(ctx context.Context, items []DocItemInput, amountField string) ([]parsedLine, error) {
	if len(items) == 0 {
		return nil, fieldError("items", "add at least one product")
	}
	if len(items) > 100 {
		return nil, fieldError("items", "a document can have at most 100 lines")
	}
	ids := make([]uuid.UUID, 0, len(items))
	for _, it := range items {
		id, err := uuid.Parse(it.ProductID)
		if err != nil {
			return nil, fieldError("items", "choose a product on every line")
		}
		ids = append(ids, id)
	}
	var products []models.Product
	if err := s.db.WithContext(ctx).Where("id IN ?", ids).Find(&products).Error; err != nil {
		return nil, err
	}
	byID := map[uuid.UUID]models.Product{}
	for _, p := range products {
		byID[p.ID] = p
	}
	out := make([]parsedLine, 0, len(items))
	for i, it := range items {
		p, ok := byID[ids[i]]
		if !ok || !p.IsActive {
			return nil, fieldError("items", fmt.Sprintf("line %d: this product was not found or is inactive", i+1))
		}
		if amountField != "unit_price" && p.Type == models.ProductService {
			return nil, fieldError("items", fmt.Sprintf("line %d (%s): a service has no stock, so it can't be bought or moved", i+1, p.Name))
		}
		qty, err := parseQuantity("items", it.Quantity)
		if err != nil {
			return nil, fieldError("items", fmt.Sprintf("line %d (%s): quantity must be above 0 (at most 4 decimals)", i+1, p.Name))
		}
		raw := it.UnitCost
		if amountField == "unit_price" {
			raw = it.UnitPrice
		}
		amount, err := parseMoney("items", raw)
		if err != nil {
			return nil, fieldError("items", fmt.Sprintf("line %d (%s): the price must be 0 or more", i+1, p.Name))
		}
		discount, err := parseMoney("items", it.Discount)
		if err != nil {
			return nil, fieldError("items", fmt.Sprintf("line %d (%s): the discount must be 0 or more", i+1, p.Name))
		}
		gross := qty.Mul(amount).Round(4)
		if discount.GreaterThan(gross) {
			return nil, fieldError("items", fmt.Sprintf("line %d (%s): the discount is more than the line", i+1, p.Name))
		}
		out = append(out, parsedLine{product: p, qty: qty, amount: amount, discount: discount, lineTotal: gross.Sub(discount), note: strings.TrimSpace(it.Note)})
	}
	return out, nil
}

// docTotals works out subtotal (lines after their discounts) − discount + shipping = total.
func docTotals(lines []parsedLine, discountRaw, shippingRaw string) (subtotal, discount, shipping, total decimal.Decimal, err error) {
	for _, l := range lines {
		subtotal = subtotal.Add(l.lineTotal)
	}
	if discount, err = parseMoney("discount", discountRaw); err != nil {
		return
	}
	if discount.GreaterThan(subtotal) {
		err = fieldError("discount", "the discount is more than the subtotal")
		return
	}
	if shipping, err = parseMoney("shipping", shippingRaw); err != nil {
		return
	}
	total = subtotal.Sub(discount).Add(shipping)
	return
}

// shares splits amount over the lines by value (by quantity when every line is free), so each
// line carries its part of a document's discount or shipping.
func shares(lines []parsedLine, amount decimal.Decimal) []decimal.Decimal {
	out := make([]decimal.Decimal, len(lines))
	if amount.IsZero() {
		return out
	}
	byValue, byQty := decimal.Zero, decimal.Zero
	for _, l := range lines {
		byValue, byQty = byValue.Add(l.lineTotal), byQty.Add(l.qty)
	}
	for i, l := range lines {
		switch {
		case byValue.IsPositive():
			out[i] = amount.Mul(l.lineTotal).Div(byValue)
		case byQty.IsPositive():
			out[i] = amount.Mul(l.qty).Div(byQty)
		}
	}
	return out
}

// landedCosts: each purchase line's unit cost into stock, with its share of shipping added and
// of the purchase discount taken off.
func landedCosts(lines []parsedLine, discount, shipping decimal.Decimal) []decimal.Decimal {
	extra := shares(lines, shipping.Sub(discount))
	out := make([]decimal.Decimal, len(lines))
	for i, l := range lines {
		landed := l.lineTotal.Add(extra[i])
		if landed.IsNegative() {
			landed = decimal.Zero
		}
		out[i] = landed.Div(l.qty).Round(4)
	}
	return out
}

// stockCost is the unit cost a purchase line brought into stock (older lines had no landed cost).
func stockCost(it models.PurchaseItem) decimal.Decimal {
	if it.LandedCost.IsZero() {
		return it.UnitCost
	}
	return it.LandedCost
}

// lockOrder returns line indexes sorted by product, so stock rows are always locked in the same
// order (no deadlocks between documents that share products).
func lockOrder(n int, product func(i int) uuid.UUID) []int {
	idx := make([]int, n)
	for i := range idx {
		idx[i] = i
	}
	sort.SliceStable(idx, func(a, b int) bool { return product(idx[a]).String() < product(idx[b]).String() })
	return idx
}

func docSearch(q *gorm.DB, search string, columns ...string) *gorm.DB {
	if strings.TrimSpace(search) == "" {
		return q
	}
	like := textsearch.Contains(search)
	conds := make([]string, len(columns))
	args := make([]interface{}, len(columns))
	for i, c := range columns {
		conds[i], args[i] = c+" LIKE ?", like
	}
	return q.Where(strings.Join(conds, " OR "), args...)
}

// paymentNames looks up payment method names.
func (s *InventoryService) paymentNames(ctx context.Context, ids []uuid.UUID) map[uuid.UUID]string {
	out := map[uuid.UUID]string{}
	if len(ids) == 0 {
		return out
	}
	var pms []models.PaymentMethod
	s.db.WithContext(ctx).Select("id", "name").Where("id IN ?", ids).Find(&pms)
	for _, pm := range pms {
		out[pm.ID] = pm.Name
	}
	return out
}

// ===== Purchases =====

type PurchaseInput struct {
	BranchID        string         `json:"branch_id" binding:"required,uuid"`
	Supplier        string         `json:"supplier" binding:"max=191"`
	PurchaseDate    string         `json:"purchase_date"`
	Currency        string         `json:"currency" binding:"required,len=3"`
	PaymentMethodID string         `json:"payment_method_id"` // empty: Cash
	Discount        string         `json:"discount"`          // off the whole purchase, an amount
	Shipping        string         `json:"shipping"`
	Note            string         `json:"note"` // rich text
	Items           []DocItemInput `json:"items" binding:"required,dive"`
	// Pending submits for approval even when the creator could approve.
	Pending bool `json:"pending"`
}

type PurchaseView struct {
	models.Purchase
	BranchCode    string        `json:"branch_code"`
	BranchName    string        `json:"branch_name"`
	PaymentMethod string        `json:"payment_method"`
	CreatedBy     string        `json:"created_by"`
	DecidedBy     string        `json:"decided_by"`
	VoidedBy      string        `json:"voided_by"`
	Lines         []DocLineView `json:"lines"`
}

func (s *InventoryService) CreatePurchase(ctx context.Context, in PurchaseInput, scope BranchScope, actor uuid.UUID) (*PurchaseView, error) {
	branchID := uuid.MustParse(in.BranchID)
	if !scope.Has(branchID) {
		return nil, ErrBranchAccessDenied
	}
	date, err := s.parseDocDate("purchase_date", in.PurchaseDate)
	if err != nil {
		return nil, err
	}
	lines, err := s.parseLines(ctx, in.Items, "unit_cost")
	if err != nil {
		return nil, err
	}
	subtotal, discount, shipping, total, err := docTotals(lines, in.Discount, in.Shipping)
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
	number, err := s.numbers.NextFor(ctx, "PUR")
	if err != nil {
		return nil, err
	}
	// People who can approve purchases record them as approved, unless they ask for approval.
	approved := !in.Pending && s.actorHas(ctx, actor, permissions.PurchasesApprove)
	purchase := models.Purchase{
		Base: models.Base{ID: uuid.New()}, Number: number, BranchID: branchID,
		Supplier: strings.TrimSpace(in.Supplier), PurchaseDate: date, Currency: in.Currency,
		Subtotal: subtotal, Discount: discount, Shipping: shipping, Total: total, PaymentMethodID: paymentID,
		Status: models.DocPending, Note: note, CreatedByID: actor,
	}
	// Stock cost is kept in USD: a purchase in another currency is converted at today's rate.
	rate, err := currencyRate(ctx, s.db, in.Currency)
	if err != nil {
		return nil, err
	}
	purchase.ExchangeRate, purchase.TotalBase = rate, toBase(total, in.Currency, rate)
	landed := landedCosts(lines, discount, shipping)
	for i, l := range lines {
		purchase.Items = append(purchase.Items, models.PurchaseItem{
			ID: uuid.New(), PurchaseID: purchase.ID, ProductID: l.product.ID,
			Quantity: l.qty, UnitCost: l.amount, Discount: l.discount, LineTotal: l.lineTotal, LandedCost: toBase(landed[i], in.Currency, rate),
		})
	}
	if approved {
		now := time.Now()
		purchase.Status, purchase.DecidedByID, purchase.DecidedAt = models.DocApproved, &actor, &now
	}
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&purchase).Error; err != nil {
			return err
		}
		if approved {
			return s.receivePurchase(tx, &purchase, actor)
		}
		return nil
	})
	if err != nil {
		return nil, s.describeShort(ctx, err)
	}
	return s.GetPurchase(ctx, purchase.ID, scope)
}

// receivePurchase brings an approved purchase's goods into stock.
func (s *InventoryService) receivePurchase(tx *gorm.DB, p *models.Purchase, actor uuid.UUID) error {
	for _, i := range lockOrder(len(p.Items), func(i int) uuid.UUID { return p.Items[i].ProductID }) {
		it := p.Items[i]
		if _, err := s.applyMove(tx, move{
			branchID: p.BranchID, productID: it.ProductID, qty: it.Quantity, unitCost: stockCost(it),
			kind: models.MovePurchase, refType: "purchase", refID: &p.ID, refNumber: p.Number, actor: actor,
		}); err != nil {
			return err
		}
	}
	return nil
}

// lockPurchase loads a purchase with its items, locked for the rest of tx.
func lockPurchase(tx *gorm.DB, id uuid.UUID, scope BranchScope) (*models.Purchase, error) {
	var p models.Purchase
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Preload("Items").First(&p, "id = ?", id).Error; err != nil {
		return nil, notFound(err)
	}
	if !scope.Has(p.BranchID) {
		return nil, ErrInventoryNotFound
	}
	return &p, nil
}

// DecidePurchase approves (stock goes in) or rejects a pending purchase. Nobody decides on a
// purchase they created.
func (s *InventoryService) DecidePurchase(ctx context.Context, id uuid.UUID, approve bool, note string, scope BranchScope, actor uuid.UUID) (*PurchaseView, error) {
	note = strings.TrimSpace(note)
	if !approve && note == "" {
		return nil, fieldError("note", "give a reason for rejecting")
	}
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		p, err := lockPurchase(tx, id, scope)
		if err != nil {
			return err
		}
		if p.Status != models.DocPending {
			return fmt.Errorf("this purchase is already %s", p.Status)
		}
		if p.CreatedByID == actor {
			return errors.New("you can't approve or reject a purchase you created")
		}
		now := time.Now()
		status := models.DocRejected
		if approve {
			status = models.DocApproved
		}
		if err := tx.Model(p).Updates(map[string]interface{}{
			"status": status, "decided_by_id": actor, "decided_at": now, "decision_note": note,
		}).Error; err != nil {
			return err
		}
		if approve {
			return s.receivePurchase(tx, p, actor)
		}
		return nil
	})
	if err != nil {
		return nil, s.describeShort(ctx, err)
	}
	return s.GetPurchase(ctx, id, scope)
}

// VoidPurchase cancels a pending purchase, or takes an approved purchase's goods back out of
// stock (when they are still there).
func (s *InventoryService) VoidPurchase(ctx context.Context, id uuid.UUID, reason string, scope BranchScope, actor uuid.UUID) (*PurchaseView, error) {
	reason = strings.TrimSpace(reason)
	if len(reason) < 3 {
		return nil, fieldError("reason", "give a reason for voiding")
	}
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		p, err := lockPurchase(tx, id, scope)
		if err != nil {
			return err
		}
		if p.Status != models.DocPending && p.Status != models.DocApproved {
			return fmt.Errorf("a %s purchase can't be voided", p.Status)
		}
		if p.Status == models.DocApproved {
			for _, i := range lockOrder(len(p.Items), func(i int) uuid.UUID { return p.Items[i].ProductID }) {
				it := p.Items[i]
				if _, err := s.applyMove(tx, move{
					branchID: p.BranchID, productID: it.ProductID, qty: it.Quantity.Neg(), unitCost: stockCost(it), atCost: true,
					kind: models.MovePurchaseVoid, refType: "purchase", refID: &p.ID, refNumber: p.Number, note: reason, actor: actor,
				}); err != nil {
					return err
				}
			}
		}
		now := time.Now()
		return tx.Model(p).Updates(map[string]interface{}{"status": models.DocVoided, "voided_by_id": actor, "voided_at": now, "void_reason": reason}).Error
	})
	if err != nil {
		return nil, s.describeShort(ctx, err)
	}
	return s.GetPurchase(ctx, id, scope)
}

func (s *InventoryService) GetPurchase(ctx context.Context, id uuid.UUID, scope BranchScope) (*PurchaseView, error) {
	var p models.Purchase
	if err := s.db.WithContext(ctx).Preload("Items").First(&p, "id = ?", id).Error; err != nil {
		return nil, notFound(err)
	}
	if !scope.Has(p.BranchID) {
		return nil, ErrInventoryNotFound
	}
	views := s.purchaseViews(ctx, []models.Purchase{p})
	return &views[0], nil
}

func (s *InventoryService) ListPurchases(ctx context.Context, f DocFilter, scope BranchScope) ([]PurchaseView, int64, error) {
	q := scope.where(s.db.WithContext(ctx).Model(&models.Purchase{}), "branch_id")
	if f.BranchID != nil {
		q = q.Where("branch_id = ?", *f.BranchID)
	}
	if f.Status != "" {
		q = q.Where("status = ?", f.Status)
	}
	if f.From != nil {
		q = q.Where("purchase_date >= ?", *f.From)
	}
	if f.To != nil {
		q = q.Where("purchase_date <= ?", *f.To)
	}
	q = docSearch(q, f.Search, "number", "supplier", "note")
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	page, size := pageArgs(f.Page, f.PageSize)
	var list []models.Purchase
	if err := q.Preload("Items").Order("purchase_date DESC, created_at DESC").Limit(size).Offset((page - 1) * size).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return s.purchaseViews(ctx, list), total, nil
}

func (s *InventoryService) purchaseViews(ctx context.Context, list []models.Purchase) []PurchaseView {
	var users, branches, products, payments []uuid.UUID
	for _, p := range list {
		if p.PaymentMethodID != nil {
			payments = append(payments, *p.PaymentMethodID)
		}
		users = append(users, p.CreatedByID)
		if p.DecidedByID != nil {
			users = append(users, *p.DecidedByID)
		}
		if p.VoidedByID != nil {
			users = append(users, *p.VoidedByID)
		}
		branches = append(branches, p.BranchID)
		for _, it := range p.Items {
			products = append(products, it.ProductID)
		}
	}
	book := s.lookup(ctx, users, branches, products)
	paymentNames := s.paymentNames(ctx, payments)
	out := make([]PurchaseView, len(list))
	for i, p := range list {
		v := PurchaseView{Purchase: p, BranchCode: book.branches[p.BranchID].Code, BranchName: book.branches[p.BranchID].Name, CreatedBy: book.users[p.CreatedByID]}
		if p.PaymentMethodID != nil {
			v.PaymentMethod = paymentNames[*p.PaymentMethodID]
		}
		if p.DecidedByID != nil {
			v.DecidedBy = book.users[*p.DecidedByID]
		}
		if p.VoidedByID != nil {
			v.VoidedBy = book.users[*p.VoidedByID]
		}
		for _, it := range p.Items {
			prod := book.products[it.ProductID]
			v.Lines = append(v.Lines, DocLineView{ProductID: it.ProductID, SKU: prod.SKU, Name: prod.Name, Unit: prod.Unit, Quantity: it.Quantity, UnitCost: it.UnitCost, Discount: it.Discount, LineTotal: it.LineTotal, LandedCost: stockCost(it)})
		}
		v.Items = nil
		out[i] = v
	}
	return out
}

// ===== Sales =====

type SaleInput struct {
	BranchID        string         `json:"branch_id" binding:"required,uuid"`
	Customer        string         `json:"customer" binding:"max=191"`
	SaleDate        string         `json:"sale_date"`
	DueDate         string         `json:"due_date"` // optional, not before the sale date
	Currency        string         `json:"currency" binding:"required,len=3"`
	PaymentMethodID string         `json:"payment_method_id"` // empty: Cash
	Discount        string         `json:"discount"`          // off the whole sale, an amount
	Shipping        string         `json:"shipping"`          // charged to the customer
	Note            string         `json:"note"`              // rich text
	Items           []DocItemInput `json:"items" binding:"required,dive"`
}

type SaleView struct {
	models.Sale
	BranchCode    string          `json:"branch_code"`
	BranchName    string          `json:"branch_name"`
	PaymentMethod string          `json:"payment_method"`
	Profit        decimal.Decimal `json:"profit"` // USD: total in USD − cost of the goods
	CreatedBy     string          `json:"created_by"`
	VoidedBy      string          `json:"voided_by"`
	Lines         []DocLineView   `json:"lines"`
}

// parseDueDate reads a sale's optional due date, which can't be before the sale date.
func (s *InventoryService) parseDueDate(raw string, saleDate time.Time) (*time.Time, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	due, err := s.parseDocDate("due_date", raw)
	if err != nil {
		return nil, err
	}
	if due.In(s.loc).Format("2006-01-02") < saleDate.In(s.loc).Format("2006-01-02") {
		return nil, fieldError("due_date", "the due date can't be before the sale date")
	}
	return &due, nil
}

func (s *InventoryService) CreateSale(ctx context.Context, in SaleInput, scope BranchScope, actor uuid.UUID) (*SaleView, error) {
	branchID := uuid.MustParse(in.BranchID)
	if !scope.Has(branchID) {
		return nil, ErrBranchAccessDenied
	}
	date, err := s.parseDocDate("sale_date", in.SaleDate)
	if err != nil {
		return nil, err
	}
	due, err := s.parseDueDate(in.DueDate, date)
	if err != nil {
		return nil, err
	}
	paymentID, err := s.paymentMethod(ctx, in.PaymentMethodID)
	if err != nil {
		return nil, err
	}
	lines, err := s.parseLines(ctx, in.Items, "unit_price")
	if err != nil {
		return nil, err
	}
	subtotal, discount, shipping, total, err := docTotals(lines, in.Discount, in.Shipping)
	if err != nil {
		return nil, err
	}
	note, err := cleanNote(in.Note)
	if err != nil {
		return nil, err
	}
	number, err := s.numbers.NextFor(ctx, "SAL")
	if err != nil {
		return nil, err
	}
	sale := models.Sale{
		Base: models.Base{ID: uuid.New()}, Number: number, BranchID: branchID,
		Customer: strings.TrimSpace(in.Customer), SaleDate: date, DueDate: due, Currency: in.Currency, PaymentMethodID: paymentID,
		Subtotal: subtotal, Discount: discount, Shipping: shipping, Total: total,
		Status: models.DocCompleted, Note: note, CreatedByID: actor,
	}
	// Cost and profit are in USD: a sale in another currency is converted at today's rate.
	rate, err := currencyRate(ctx, s.db, in.Currency)
	if err != nil {
		return nil, err
	}
	sale.ExchangeRate, sale.TotalBase = rate, toBase(total, in.Currency, rate)
	discountShares := shares(lines, discount)
	sale.Items = make([]models.SaleItem, len(lines))
	for i, l := range lines {
		net := l.lineTotal.Sub(discountShares[i]).Round(4)
		sale.Items[i] = models.SaleItem{
			ID: uuid.New(), SaleID: sale.ID, ProductID: l.product.ID, Quantity: l.qty,
			UnitPrice: l.amount, Discount: l.discount, LineTotal: l.lineTotal, NetTotal: net, NetTotalBase: toBase(net, in.Currency, rate), Note: l.note,
		}
	}
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Stock goes out first, so each line records the average cost it left at.
		for _, i := range lockOrder(len(sale.Items), func(i int) uuid.UUID { return sale.Items[i].ProductID }) {
			cost, err := s.applyMove(tx, move{
				branchID: branchID, productID: sale.Items[i].ProductID, qty: sale.Items[i].Quantity.Neg(),
				kind: models.MoveSale, refType: "sale", refID: &sale.ID, refNumber: number, actor: actor,
			})
			if err != nil {
				return err
			}
			sale.Items[i].UnitCost = cost
		}
		totalCost := decimal.Zero
		for _, it := range sale.Items {
			totalCost = totalCost.Add(it.Quantity.Mul(it.UnitCost).Round(4))
		}
		sale.TotalCost = totalCost
		return tx.Create(&sale).Error
	})
	if err != nil {
		return nil, s.describeShort(ctx, err)
	}
	return s.GetSale(ctx, sale.ID, scope)
}

// VoidSale cancels a sale and puts its goods back into stock at the cost they left at.
func (s *InventoryService) VoidSale(ctx context.Context, id uuid.UUID, reason string, scope BranchScope, actor uuid.UUID) (*SaleView, error) {
	reason = strings.TrimSpace(reason)
	if len(reason) < 3 {
		return nil, fieldError("reason", "give a reason for voiding")
	}
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var sale models.Sale
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Preload("Items").First(&sale, "id = ?", id).Error; err != nil {
			return notFound(err)
		}
		if !scope.Has(sale.BranchID) {
			return ErrInventoryNotFound
		}
		if sale.Status != models.DocCompleted {
			return fmt.Errorf("this sale is already %s", sale.Status)
		}
		for _, i := range lockOrder(len(sale.Items), func(i int) uuid.UUID { return sale.Items[i].ProductID }) {
			it := sale.Items[i]
			if _, err := s.applyMove(tx, move{
				branchID: sale.BranchID, productID: it.ProductID, qty: it.Quantity, unitCost: it.UnitCost,
				kind: models.MoveSaleVoid, refType: "sale", refID: &sale.ID, refNumber: sale.Number, note: reason, actor: actor,
			}); err != nil {
				return err
			}
		}
		now := time.Now()
		return tx.Model(&sale).Updates(map[string]interface{}{"status": models.DocVoided, "voided_by_id": actor, "voided_at": now, "void_reason": reason}).Error
	})
	if err != nil {
		return nil, s.describeShort(ctx, err)
	}
	return s.GetSale(ctx, id, scope)
}

func (s *InventoryService) GetSale(ctx context.Context, id uuid.UUID, scope BranchScope) (*SaleView, error) {
	var sale models.Sale
	if err := s.db.WithContext(ctx).Preload("Items").First(&sale, "id = ?", id).Error; err != nil {
		return nil, notFound(err)
	}
	if !scope.Has(sale.BranchID) {
		return nil, ErrInventoryNotFound
	}
	views := s.saleViews(ctx, []models.Sale{sale})
	return &views[0], nil
}

func (s *InventoryService) ListSales(ctx context.Context, f DocFilter, scope BranchScope) ([]SaleView, int64, error) {
	q := scope.where(s.db.WithContext(ctx).Model(&models.Sale{}), "branch_id")
	if f.BranchID != nil {
		q = q.Where("branch_id = ?", *f.BranchID)
	}
	if f.Status != "" {
		q = q.Where("status = ?", f.Status)
	}
	if f.From != nil {
		q = q.Where("sale_date >= ?", *f.From)
	}
	if f.To != nil {
		q = q.Where("sale_date <= ?", *f.To)
	}
	q = docSearch(q, f.Search, "number", "customer", "note")
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	page, size := pageArgs(f.Page, f.PageSize)
	var list []models.Sale
	if err := q.Preload("Items").Order("sale_date DESC, created_at DESC").Limit(size).Offset((page - 1) * size).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return s.saleViews(ctx, list), total, nil
}

func (s *InventoryService) saleViews(ctx context.Context, list []models.Sale) []SaleView {
	var users, branches, products, payments []uuid.UUID
	for _, sale := range list {
		users = append(users, sale.CreatedByID)
		if sale.VoidedByID != nil {
			users = append(users, *sale.VoidedByID)
		}
		branches = append(branches, sale.BranchID)
		if sale.PaymentMethodID != nil {
			payments = append(payments, *sale.PaymentMethodID)
		}
		for _, it := range sale.Items {
			products = append(products, it.ProductID)
		}
	}
	book := s.lookup(ctx, users, branches, products)
	paymentNames := s.paymentNames(ctx, payments)
	out := make([]SaleView, len(list))
	for i, sale := range list {
		v := SaleView{Sale: sale, BranchCode: book.branches[sale.BranchID].Code, BranchName: book.branches[sale.BranchID].Name,
			CreatedBy: book.users[sale.CreatedByID], Profit: sale.TotalBase.Sub(sale.TotalCost)}
		if sale.PaymentMethodID != nil {
			v.PaymentMethod = paymentNames[*sale.PaymentMethodID]
		}
		if sale.VoidedByID != nil {
			v.VoidedBy = book.users[*sale.VoidedByID]
		}
		for _, it := range sale.Items {
			prod := book.products[it.ProductID]
			v.Lines = append(v.Lines, DocLineView{ProductID: it.ProductID, SKU: prod.SKU, Name: prod.Name, Unit: prod.Unit, Quantity: it.Quantity, UnitPrice: it.UnitPrice, UnitCost: it.UnitCost, Discount: it.Discount, LineTotal: it.LineTotal, Note: it.Note})
		}
		v.Items = nil
		out[i] = v
	}
	return out
}

// ===== Transfers =====

type TransferInput struct {
	FromBranchID string         `json:"from_branch_id" binding:"required,uuid"`
	ToBranchID   string         `json:"to_branch_id" binding:"required,uuid"`
	TransferDate string         `json:"transfer_date"`
	Note         string         `json:"note"` // rich text
	Items        []DocItemInput `json:"items" binding:"required,dive"`
}

type TransferView struct {
	models.StockTransfer
	FromCode  string        `json:"from_code"`
	FromName  string        `json:"from_name"`
	ToCode    string        `json:"to_code"`
	ToName    string        `json:"to_name"`
	CreatedBy string        `json:"created_by"`
	VoidedBy  string        `json:"voided_by"`
	Lines     []DocLineView `json:"lines"`
}

func (s *InventoryService) CreateTransfer(ctx context.Context, in TransferInput, scope BranchScope, actor uuid.UUID) (*TransferView, error) {
	from, to := uuid.MustParse(in.FromBranchID), uuid.MustParse(in.ToBranchID)
	if from == to {
		return nil, fieldError("to_branch_id", "choose a different branch to send to")
	}
	if !scope.Has(from) || !scope.Has(to) {
		return nil, ErrBranchAccessDenied
	}
	date, err := s.parseDocDate("transfer_date", in.TransferDate)
	if err != nil {
		return nil, err
	}
	lines, err := s.parseLines(ctx, in.Items, "unit_cost")
	if err != nil {
		return nil, err
	}
	note, err := cleanNote(in.Note)
	if err != nil {
		return nil, err
	}
	number, err := s.numbers.NextFor(ctx, "TRF")
	if err != nil {
		return nil, err
	}
	tr := models.StockTransfer{
		Base: models.Base{ID: uuid.New()}, Number: number, FromBranchID: from, ToBranchID: to,
		TransferDate: date, Status: models.DocCompleted, Note: note, CreatedByID: actor,
	}
	tr.Items = make([]models.StockTransferItem, len(lines))
	for i, l := range lines {
		tr.Items[i] = models.StockTransferItem{ID: uuid.New(), TransferID: tr.ID, ProductID: l.product.ID, Quantity: l.qty}
	}
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, i := range lockOrder(len(tr.Items), func(i int) uuid.UUID { return tr.Items[i].ProductID }) {
			it := &tr.Items[i]
			cost, err := s.applyMove(tx, move{
				branchID: from, productID: it.ProductID, qty: it.Quantity.Neg(),
				kind: models.MoveTransferOut, refType: "transfer", refID: &tr.ID, refNumber: number, actor: actor,
			})
			if err != nil {
				return err
			}
			it.UnitCost = cost
			if _, err := s.applyMove(tx, move{
				branchID: to, productID: it.ProductID, qty: it.Quantity, unitCost: cost,
				kind: models.MoveTransferIn, refType: "transfer", refID: &tr.ID, refNumber: number, actor: actor,
			}); err != nil {
				return err
			}
		}
		return tx.Create(&tr).Error
	})
	if err != nil {
		return nil, s.describeShort(ctx, err)
	}
	return s.GetTransfer(ctx, tr.ID, scope)
}

// VoidTransfer moves the goods back to the sending branch (when they are still at the other one).
func (s *InventoryService) VoidTransfer(ctx context.Context, id uuid.UUID, reason string, scope BranchScope, actor uuid.UUID) (*TransferView, error) {
	reason = strings.TrimSpace(reason)
	if len(reason) < 3 {
		return nil, fieldError("reason", "give a reason for voiding")
	}
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var tr models.StockTransfer
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Preload("Items").First(&tr, "id = ?", id).Error; err != nil {
			return notFound(err)
		}
		if !scope.Has(tr.FromBranchID) || !scope.Has(tr.ToBranchID) {
			return ErrInventoryNotFound
		}
		if tr.Status != models.DocCompleted {
			return fmt.Errorf("this transfer is already %s", tr.Status)
		}
		for _, i := range lockOrder(len(tr.Items), func(i int) uuid.UUID { return tr.Items[i].ProductID }) {
			it := tr.Items[i]
			if _, err := s.applyMove(tx, move{
				branchID: tr.ToBranchID, productID: it.ProductID, qty: it.Quantity.Neg(), unitCost: it.UnitCost, atCost: true,
				kind: models.MoveTransferVoid, refType: "transfer", refID: &tr.ID, refNumber: tr.Number, note: reason, actor: actor,
			}); err != nil {
				return err
			}
			if _, err := s.applyMove(tx, move{
				branchID: tr.FromBranchID, productID: it.ProductID, qty: it.Quantity, unitCost: it.UnitCost,
				kind: models.MoveTransferVoid, refType: "transfer", refID: &tr.ID, refNumber: tr.Number, note: reason, actor: actor,
			}); err != nil {
				return err
			}
		}
		now := time.Now()
		return tx.Model(&tr).Updates(map[string]interface{}{"status": models.DocVoided, "voided_by_id": actor, "voided_at": now, "void_reason": reason}).Error
	})
	if err != nil {
		return nil, s.describeShort(ctx, err)
	}
	return s.GetTransfer(ctx, id, scope)
}

// A transfer is visible to people who see either branch.
func transferVisible(tr models.StockTransfer, scope BranchScope) bool {
	return scope.Has(tr.FromBranchID) || scope.Has(tr.ToBranchID)
}

func (s *InventoryService) GetTransfer(ctx context.Context, id uuid.UUID, scope BranchScope) (*TransferView, error) {
	var tr models.StockTransfer
	if err := s.db.WithContext(ctx).Preload("Items").First(&tr, "id = ?", id).Error; err != nil {
		return nil, notFound(err)
	}
	if !transferVisible(tr, scope) {
		return nil, ErrInventoryNotFound
	}
	views := s.transferViews(ctx, []models.StockTransfer{tr})
	return &views[0], nil
}

func (s *InventoryService) ListTransfers(ctx context.Context, f DocFilter, scope BranchScope) ([]TransferView, int64, error) {
	q := s.db.WithContext(ctx).Model(&models.StockTransfer{})
	if !scope.All {
		if len(scope.IDs) == 0 {
			q = q.Where("1 = 0")
		} else {
			q = q.Where("from_branch_id IN ? OR to_branch_id IN ?", scope.IDs, scope.IDs)
		}
	}
	if f.BranchID != nil {
		q = q.Where("from_branch_id = ? OR to_branch_id = ?", *f.BranchID, *f.BranchID)
	}
	if f.Status != "" {
		q = q.Where("status = ?", f.Status)
	}
	if f.From != nil {
		q = q.Where("transfer_date >= ?", *f.From)
	}
	if f.To != nil {
		q = q.Where("transfer_date <= ?", *f.To)
	}
	q = docSearch(q, f.Search, "number", "note")
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	page, size := pageArgs(f.Page, f.PageSize)
	var list []models.StockTransfer
	if err := q.Preload("Items").Order("transfer_date DESC, created_at DESC").Limit(size).Offset((page - 1) * size).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return s.transferViews(ctx, list), total, nil
}

func (s *InventoryService) transferViews(ctx context.Context, list []models.StockTransfer) []TransferView {
	var users, branches, products []uuid.UUID
	for _, tr := range list {
		users = append(users, tr.CreatedByID)
		if tr.VoidedByID != nil {
			users = append(users, *tr.VoidedByID)
		}
		branches = append(branches, tr.FromBranchID, tr.ToBranchID)
		for _, it := range tr.Items {
			products = append(products, it.ProductID)
		}
	}
	book := s.lookup(ctx, users, branches, products)
	out := make([]TransferView, len(list))
	for i, tr := range list {
		v := TransferView{StockTransfer: tr,
			FromCode: book.branches[tr.FromBranchID].Code, FromName: book.branches[tr.FromBranchID].Name,
			ToCode: book.branches[tr.ToBranchID].Code, ToName: book.branches[tr.ToBranchID].Name,
			CreatedBy: book.users[tr.CreatedByID]}
		if tr.VoidedByID != nil {
			v.VoidedBy = book.users[*tr.VoidedByID]
		}
		for _, it := range tr.Items {
			prod := book.products[it.ProductID]
			v.Lines = append(v.Lines, DocLineView{ProductID: it.ProductID, SKU: prod.SKU, Name: prod.Name, Unit: prod.Unit, Quantity: it.Quantity, UnitCost: it.UnitCost, LineTotal: it.Quantity.Mul(it.UnitCost).Round(4)})
		}
		v.Items = nil
		out[i] = v
	}
	return out
}
