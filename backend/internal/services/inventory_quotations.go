package services

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/cambodia-finance/income-expense/internal/models"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Quotations: price offers from a branch to a customer. They move no stock; one the customer takes
// is turned into a sale (at its prices), which takes the goods out of stock like any other sale.

type QuotationInput struct {
	BranchID   string         `json:"branch_id" binding:"required,uuid"`
	CustomerID string         `json:"customer_id"`                // from the customers list
	Customer   string         `json:"customer" binding:"max=191"` // without customer_id: the name typed
	Contact    string         `json:"contact" binding:"max=191"`
	QuoteDate  string         `json:"quote_date"`
	ValidUntil string         `json:"valid_until"` // optional, not before the quotation date
	Currency   string         `json:"currency" binding:"required,len=3"`
	Discount   string         `json:"discount"` // off the whole quotation, an amount
	Shipping   string         `json:"shipping"` // charged to the customer
	Note       string         `json:"note"`     // rich text
	Items      []DocItemInput `json:"items" binding:"required,dive"`
}

// ConvertQuotationInput is what a sale needs beyond the quotation: its date, due date and payment.
type ConvertQuotationInput struct {
	SaleDate        string `json:"sale_date"` // empty: now
	DueDate         string `json:"due_date"`
	PaymentMethodID string `json:"payment_method_id"` // empty: Cash
}

type QuotationView struct {
	models.Quotation
	BranchCode  string        `json:"branch_code"`
	BranchName  string        `json:"branch_name"`
	CreatedBy   string        `json:"created_by"`
	DecidedBy   string        `json:"decided_by"`
	ConvertedBy string        `json:"converted_by"`
	SaleNumber  string        `json:"sale_number"`
	Expired     bool          `json:"expired"` // open, and its valid-until day has passed
	Lines       []DocLineView `json:"lines"`
	// Seller is who the offer is from, for the printed quotation (only on a single quotation).
	Seller *QuotationSeller `json:"seller,omitempty"`
}

// QuotationSeller is the business and the branch a quotation comes from.
type QuotationSeller struct {
	Company string `json:"company"`
	Branch  string `json:"branch"`
	Address string `json:"address"`
	Phone   string `json:"phone"`
	Email   string `json:"email"`
}

// QuotationStatusExpired filters open quotations whose valid-until day has passed.
const QuotationStatusExpired = "expired"

// ErrQuotationChanged: the quotation changed between reading it and turning it into a sale.
var ErrQuotationChanged = errors.New("this quotation was just changed by someone else; open it again")

// quotationParts is a checked quotation input.
type quotationParts struct {
	branchID                            uuid.UUID
	customerID                          *uuid.UUID
	customer                            string
	date                                time.Time
	validUntil                          *time.Time
	lines                               []parsedLine
	subtotal, discount, shipping, total decimal.Decimal
	note                                string
}

// checkQuotation checks a quotation's input; current is its customer so far (a turned-off one can stay).
func (s *InventoryService) checkQuotation(ctx context.Context, in QuotationInput, scope BranchScope, current *uuid.UUID) (*quotationParts, error) {
	branchID := uuid.MustParse(in.BranchID)
	if !scope.Has(branchID) {
		return nil, ErrBranchAccessDenied
	}
	customerID, customer, err := s.pickParty(ctx, Customers, in.CustomerID, in.Customer, current, branchID)
	if err != nil {
		return nil, err
	}
	if customer == "" {
		return nil, fieldError("customer_id", "choose the customer the quotation is for")
	}
	date, err := s.parseDocDate("quote_date", in.QuoteDate)
	if err != nil {
		return nil, err
	}
	var validUntil *time.Time
	if strings.TrimSpace(in.ValidUntil) != "" {
		v, err := s.parseDocDate("valid_until", in.ValidUntil)
		if err != nil {
			return nil, err
		}
		if s.day(v) < s.day(date) {
			return nil, fieldError("valid_until", "the valid-until date can't be before the quotation date")
		}
		validUntil = &v
	}
	lines, err := s.parseLines(ctx, in.Items, "unit_price", &branchID)
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
	return &quotationParts{branchID: branchID, customerID: customerID, customer: customer, date: date, validUntil: validUntil, lines: lines, subtotal: subtotal, discount: discount, shipping: shipping, total: total, note: note}, nil
}

// day is t's date in the business time zone (YYYY-MM-DD).
func (s *InventoryService) day(t time.Time) string { return t.In(s.loc).Format("2006-01-02") }

func quotationItems(id uuid.UUID, lines []parsedLine) []models.QuotationItem {
	items := make([]models.QuotationItem, len(lines))
	for i, l := range lines {
		items[i] = models.QuotationItem{
			ID: uuid.New(), QuotationID: id, Position: i, ProductID: l.product.ID, Quantity: l.qty,
			UnitPrice: l.amount, Discount: l.discount, LineTotal: l.lineTotal, Note: l.note,
		}
	}
	return items
}

func (s *InventoryService) CreateQuotation(ctx context.Context, in QuotationInput, scope BranchScope, actor uuid.UUID) (*QuotationView, error) {
	p, err := s.checkQuotation(ctx, in, scope, nil)
	if err != nil {
		return nil, err
	}
	rate, err := currencyRate(ctx, s.db, in.Currency)
	if err != nil {
		return nil, err
	}
	number, err := s.numbers.NextFor(ctx, "QUO")
	if err != nil {
		return nil, err
	}
	q := models.Quotation{
		Base: models.Base{ID: uuid.New()}, Number: number, BranchID: p.branchID,
		CustomerID: p.customerID, Customer: p.customer, Contact: strings.TrimSpace(in.Contact), QuoteDate: p.date, ValidUntil: p.validUntil,
		Currency: in.Currency, Subtotal: p.subtotal, Discount: p.discount, Shipping: p.shipping, Total: p.total,
		ExchangeRate: rate, TotalBase: toBase(p.total, in.Currency, rate),
		Status: models.QuoteOpen, Note: p.note, CreatedByID: actor,
	}
	q.Items = quotationItems(q.ID, p.lines)
	if err := s.db.WithContext(ctx).Create(&q).Error; err != nil {
		return nil, err
	}
	return s.GetQuotation(ctx, q.ID, scope)
}

// inOrder loads a quotation's lines in their order on it.
func inOrder(db *gorm.DB) *gorm.DB { return db.Order("position") }

// lockQuotation reads a quotation for a change (locked until the transaction ends).
func lockQuotation(tx *gorm.DB, id uuid.UUID, scope BranchScope) (*models.Quotation, error) {
	var q models.Quotation
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Preload("Items", inOrder).First(&q, "id = ?", id).Error; err != nil {
		return nil, notFound(err)
	}
	if !scope.Has(q.BranchID) {
		return nil, ErrInventoryNotFound
	}
	return &q, nil
}

// UpdateQuotation changes a quotation that wasn't turned into a sale. A changed offer is open again
// (an accepted or declined one goes back to open).
func (s *InventoryService) UpdateQuotation(ctx context.Context, id uuid.UUID, in QuotationInput, scope BranchScope, actor uuid.UUID) (*QuotationView, error) {
	var current models.Quotation // its customer so far (missing: the lock below answers not found)
	s.db.WithContext(ctx).Select("id", "customer_id").Take(&current, "id = ?", id)
	p, err := s.checkQuotation(ctx, in, scope, current.CustomerID)
	if err != nil {
		return nil, err
	}
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		q, err := lockQuotation(tx, id, scope)
		if err != nil {
			return err
		}
		if q.Status == models.QuoteConverted {
			return errors.New("this quotation was turned into a sale, so it can't be changed")
		}
		rate, err := s.keepRate(ctx, q.Currency, q.ExchangeRate, in.Currency)
		if err != nil {
			return err
		}
		if err := tx.Where("quotation_id = ?", q.ID).Delete(&models.QuotationItem{}).Error; err != nil {
			return err
		}
		items := quotationItems(q.ID, p.lines)
		if err := tx.Create(&items).Error; err != nil {
			return err
		}
		return tx.Model(&models.Quotation{}).Where("id = ?", q.ID).Updates(map[string]interface{}{
			"branch_id": p.branchID, "customer_id": p.customerID, "customer": p.customer, "contact": strings.TrimSpace(in.Contact),
			"quote_date": p.date, "valid_until": p.validUntil, "currency": in.Currency,
			"subtotal": p.subtotal, "discount": p.discount, "shipping": p.shipping, "total": p.total,
			"exchange_rate": rate, "total_base": toBase(p.total, in.Currency, rate), "note": p.note,
			"status": models.QuoteOpen, "decided_by_id": nil, "decided_at": nil, "updated_at": time.Now(),
		}).Error
	})
	if err != nil {
		return nil, err
	}
	return s.GetQuotation(ctx, id, scope)
}

// SetQuotationStatus marks a quotation accepted or declined by the customer, or open again.
func (s *InventoryService) SetQuotationStatus(ctx context.Context, id uuid.UUID, status string, scope BranchScope, actor uuid.UUID) (*QuotationView, error) {
	if status != models.QuoteOpen && status != models.QuoteAccepted && status != models.QuoteDeclined {
		return nil, fieldError("status", "choose open, accepted or declined")
	}
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		q, err := lockQuotation(tx, id, scope)
		if err != nil {
			return err
		}
		if q.Status == models.QuoteConverted {
			return errors.New("this quotation was turned into a sale")
		}
		if q.Status == status {
			return fmt.Errorf("this quotation is already %s", status)
		}
		changes := map[string]interface{}{"status": status, "decided_by_id": nil, "decided_at": nil}
		if status != models.QuoteOpen {
			changes["decided_by_id"], changes["decided_at"] = actor, time.Now()
		}
		return tx.Model(&models.Quotation{}).Where("id = ?", q.ID).Updates(changes).Error
	})
	if err != nil {
		return nil, err
	}
	return s.GetQuotation(ctx, id, scope)
}

// ConvertQuotation turns an open or accepted quotation into a sale at its prices: the goods go out
// of the branch's stock (there must be enough), and the quotation is marked converted with the sale.
func (s *InventoryService) ConvertQuotation(ctx context.Context, id uuid.UUID, in ConvertQuotationInput, scope BranchScope, actor uuid.UUID) (*SaleView, error) {
	var q models.Quotation
	if err := s.db.WithContext(ctx).Preload("Items", inOrder).First(&q, "id = ?", id).Error; err != nil {
		return nil, notFound(err)
	}
	if !scope.Has(q.BranchID) {
		return nil, ErrInventoryNotFound
	}
	if err := convertible(q.Status); err != nil {
		return nil, err
	}
	saleIn := SaleInput{
		BranchID: q.BranchID.String(), Customer: q.Customer, keepCustomer: q.CustomerID, SaleDate: in.SaleDate, DueDate: in.DueDate, Currency: q.Currency,
		PaymentMethodID: in.PaymentMethodID, Discount: q.Discount.String(), Shipping: q.Shipping.String(), Note: q.Note,
	}
	for _, it := range q.Items {
		saleIn.Items = append(saleIn.Items, DocItemInput{
			ProductID: it.ProductID.String(), Quantity: it.Quantity.String(), UnitPrice: it.UnitPrice.String(), Discount: it.Discount.String(), Note: it.Note,
		})
	}
	sale, err := s.newSale(ctx, saleIn, scope, actor)
	if err != nil {
		return nil, err
	}
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		locked, err := lockQuotation(tx, id, scope)
		if err != nil {
			return err
		}
		if err := convertible(locked.Status); err != nil {
			return err
		}
		if !locked.UpdatedAt.Equal(q.UpdatedAt) {
			return ErrQuotationChanged
		}
		if err := s.saveNewSale(tx, sale, actor); err != nil {
			return err
		}
		now := time.Now()
		return tx.Model(&models.Quotation{}).Where("id = ?", id).Updates(map[string]interface{}{
			"status": models.QuoteConverted, "sale_id": sale.ID, "converted_by_id": actor, "converted_at": now, "updated_at": now,
		}).Error
	})
	if err != nil {
		return nil, s.describeShort(ctx, err)
	}
	return s.GetSale(ctx, sale.ID, scope)
}

func convertible(status string) error {
	switch status {
	case models.QuoteOpen, models.QuoteAccepted:
		return nil
	case models.QuoteConverted:
		return errors.New("this quotation was already turned into a sale")
	default:
		return errors.New("a declined quotation can't be turned into a sale; mark it open again first")
	}
}

// DeleteQuotation deletes a quotation that wasn't turned into a sale.
func (s *InventoryService) DeleteQuotation(ctx context.Context, id uuid.UUID, scope BranchScope) (*models.Quotation, error) {
	var deleted models.Quotation
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		q, err := lockQuotation(tx, id, scope)
		if err != nil {
			return err
		}
		if q.Status == models.QuoteConverted {
			return errors.New("this quotation was turned into a sale, so it can't be deleted")
		}
		deleted = *q
		return tx.Delete(&models.Quotation{}, "id = ?", q.ID).Error
	})
	if err != nil {
		return nil, err
	}
	return &deleted, nil
}

func (s *InventoryService) GetQuotation(ctx context.Context, id uuid.UUID, scope BranchScope) (*QuotationView, error) {
	var q models.Quotation
	if err := s.db.WithContext(ctx).Preload("Items", inOrder).First(&q, "id = ?", id).Error; err != nil {
		return nil, notFound(err)
	}
	if !scope.Has(q.BranchID) {
		return nil, ErrInventoryNotFound
	}
	views := s.quotationViews(ctx, []models.Quotation{q})
	views[0].Seller = s.quotationSeller(ctx, q.BranchID)
	return &views[0], nil
}

// quotationSeller is the company name (Settings) and the branch's name and contact details.
func (s *InventoryService) quotationSeller(ctx context.Context, branchID uuid.UUID) *QuotationSeller {
	seller := &QuotationSeller{Company: "My Business"}
	var setting models.AppSetting
	if err := s.db.WithContext(ctx).Where("`key` = ?", "company_name").Take(&setting).Error; err == nil && strings.TrimSpace(setting.Value) != "" {
		seller.Company = strings.TrimSpace(setting.Value)
	}
	var b models.Branch
	if err := s.db.WithContext(ctx).Unscoped().Select("name", "address", "phone", "email").Take(&b, "id = ?", branchID).Error; err == nil {
		seller.Branch, seller.Address, seller.Phone, seller.Email = b.Name, b.Address, b.Phone, b.Email
	}
	return seller
}

// startOfToday is midnight today in the business time zone.
func (s *InventoryService) startOfToday() time.Time {
	now := time.Now().In(s.loc)
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, s.loc)
}

func (s *InventoryService) ListQuotations(ctx context.Context, f DocFilter, scope BranchScope) ([]QuotationView, int64, error) {
	q := scope.where(s.db.WithContext(ctx).Model(&models.Quotation{}), "branch_id")
	if f.BranchID != nil {
		q = q.Where("branch_id = ?", *f.BranchID)
	}
	today := s.startOfToday()
	switch f.Status {
	case "":
	case models.QuoteOpen: // still valid
		q = q.Where("status = ? AND (valid_until IS NULL OR valid_until >= ?)", models.QuoteOpen, today)
	case QuotationStatusExpired:
		q = q.Where("status = ? AND valid_until < ?", models.QuoteOpen, today)
	default:
		q = q.Where("status = ?", f.Status)
	}
	if f.From != nil {
		q = q.Where("quote_date >= ?", *f.From)
	}
	if f.To != nil {
		q = q.Where("quote_date <= ?", *f.To)
	}
	q = docSearch(q, f.Search, "number", "customer", "contact", "note")
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	page, size := pageArgs(f.Page, f.PageSize)
	var list []models.Quotation
	if err := q.Preload("Items", inOrder).Order("quote_date DESC, created_at DESC").Limit(size).Offset((page - 1) * size).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return s.quotationViews(ctx, list), total, nil
}

func (s *InventoryService) quotationViews(ctx context.Context, list []models.Quotation) []QuotationView {
	var users, branches, products, sales []uuid.UUID
	for _, q := range list {
		users = append(users, q.CreatedByID)
		for _, u := range []*uuid.UUID{q.DecidedByID, q.ConvertedByID} {
			if u != nil {
				users = append(users, *u)
			}
		}
		branches = append(branches, q.BranchID)
		if q.SaleID != nil {
			sales = append(sales, *q.SaleID)
		}
		for _, it := range q.Items {
			products = append(products, it.ProductID)
		}
	}
	book := s.lookup(ctx, users, branches, products)
	saleNumbers := map[uuid.UUID]string{}
	if len(sales) > 0 {
		var rows []models.Sale
		s.db.WithContext(ctx).Unscoped().Select("id", "number").Where("id IN ?", sales).Find(&rows)
		for _, r := range rows {
			saleNumbers[r.ID] = r.Number
		}
	}
	today := s.day(time.Now())
	out := make([]QuotationView, len(list))
	for i, q := range list {
		v := QuotationView{Quotation: q, BranchCode: book.branches[q.BranchID].Code, BranchName: book.branches[q.BranchID].Name, CreatedBy: book.users[q.CreatedByID]}
		if q.DecidedByID != nil {
			v.DecidedBy = book.users[*q.DecidedByID]
		}
		if q.ConvertedByID != nil {
			v.ConvertedBy = book.users[*q.ConvertedByID]
		}
		if q.SaleID != nil {
			v.SaleNumber = saleNumbers[*q.SaleID]
		}
		v.Expired = q.Status == models.QuoteOpen && q.ValidUntil != nil && s.day(*q.ValidUntil) < today
		for _, it := range q.Items {
			prod := book.products[it.ProductID]
			v.Lines = append(v.Lines, DocLineView{ProductID: it.ProductID, SKU: prod.SKU, Name: prod.Name, Unit: prod.Unit, Quantity: it.Quantity, UnitPrice: it.UnitPrice, Discount: it.Discount, LineTotal: it.LineTotal, Note: it.Note})
		}
		v.Items = nil
		out[i] = v
	}
	return out
}

// quotationsOfSales: the quotation each of these sales was made from.
func (s *InventoryService) quotationsOfSales(ctx context.Context, list []models.Sale) map[uuid.UUID]models.Quotation {
	out := map[uuid.UUID]models.Quotation{}
	if len(list) == 0 {
		return out
	}
	ids := make([]uuid.UUID, len(list))
	for i, sale := range list {
		ids[i] = sale.ID
	}
	var rows []models.Quotation
	s.db.WithContext(ctx).Select("id", "number", "sale_id").Where("sale_id IN ?", ids).Find(&rows)
	for _, q := range rows {
		if q.SaleID != nil {
			out[*q.SaleID] = q
		}
	}
	return out
}
