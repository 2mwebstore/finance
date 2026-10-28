package services

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/cambodia-finance/income-expense/internal/accounting"
	"github.com/cambodia-finance/income-expense/internal/models"
	"github.com/cambodia-finance/income-expense/internal/richtext"
	"github.com/cambodia-finance/income-expense/internal/storage"
	"github.com/cambodia-finance/income-expense/internal/textsearch"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// InventoryService: the product catalog (shared by every branch), stock per branch at average
// cost, stock history, adjustments, and the documents that move stock (inventory_documents.go).
// Inventory is separate from the income/expense ledger.
type InventoryService struct {
	db      *gorm.DB
	numbers *accounting.NumberGenerator
	loc     *time.Location
	store   storage.Store // product pictures (R2, or local disk in development); may be nil
}

func NewInventoryService(db *gorm.DB, timezone string, store storage.Store) *InventoryService {
	return &InventoryService{db: db, numbers: accounting.NewNumberGenerator(db, timezone), loc: accounting.LoadLocation(timezone), store: store}
}

// ErrInventoryNotFound is answered with 404.
var ErrInventoryNotFound = errors.New("not found")

// BranchScope is the branches whose stock and documents the actor sees and works in.
type BranchScope struct {
	All bool
	IDs []uuid.UUID
}

func (b BranchScope) Has(id uuid.UUID) bool {
	if b.All {
		return true
	}
	for _, x := range b.IDs {
		if x == id {
			return true
		}
	}
	return false
}

// where limits q to the scope's branches (column is the branch column).
func (b BranchScope) where(q *gorm.DB, column string) *gorm.DB {
	if b.All {
		return q
	}
	if len(b.IDs) == 0 {
		return q.Where("1 = 0")
	}
	return q.Where(column+" IN ?", b.IDs)
}

// --- input helpers ---

var maxQuantity = decimal.RequireFromString("999999999")

// parseQuantity: a positive number with at most 4 decimal places.
func parseQuantity(field, raw string) (decimal.Decimal, error) {
	d, err := decimal.NewFromString(strings.TrimSpace(raw))
	if err != nil || !d.IsPositive() || d.Exponent() < -4 || d.GreaterThan(maxQuantity) {
		return decimal.Zero, fieldError(field, "quantity must be a number above 0 (at most 4 decimals)")
	}
	return d, nil
}

// parseMoney: zero or more, at most 4 decimal places. Empty is zero.
func parseMoney(field, raw string) (decimal.Decimal, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return decimal.Zero, nil
	}
	d, err := decimal.NewFromString(raw)
	if err != nil || d.IsNegative() || d.Exponent() < -4 || d.GreaterThan(accounting.MaxAmount) {
		return decimal.Zero, fieldError(field, "enter an amount of 0 or more (at most 4 decimals)")
	}
	return d, nil
}

// parseDocDate reads YYYY-MM-DD (or RFC3339) in the business time zone; empty is now.
func (s *InventoryService) parseDocDate(field, raw string) (time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Now(), nil
	}
	if t, err := time.ParseInLocation("2006-01-02", raw, s.loc); err == nil {
		// Midday keeps the day the same in any time zone the date is shown in.
		return t.Add(12 * time.Hour), nil
	}
	if t, err := time.Parse(time.RFC3339, raw); err == nil {
		return t, nil
	}
	return time.Time{}, fieldError(field, "date is not valid")
}

// DateRange parses from/to (YYYY-MM-DD, either may be empty) into an inclusive range.
func (s *InventoryService) DateRange(from, to string) (*time.Time, *time.Time, error) {
	var start, end *time.Time
	if from = strings.TrimSpace(from); from != "" {
		t, err := time.ParseInLocation("2006-01-02", from, s.loc)
		if err != nil {
			return nil, nil, fieldError("date_from", "start date is not valid")
		}
		start = &t
	}
	if to = strings.TrimSpace(to); to != "" {
		t, err := time.ParseInLocation("2006-01-02", to, s.loc)
		if err != nil {
			return nil, nil, fieldError("date_to", "end date is not valid")
		}
		t = t.AddDate(0, 0, 1).Add(-time.Nanosecond)
		end = &t
	}
	if start != nil && end != nil && end.Before(*start) {
		return nil, nil, fieldError("date_to", "end date is before the start date")
	}
	return start, end, nil
}

// cleanNote sanitizes a rich-text note (same rules as transaction descriptions).
func cleanNote(raw string) (string, error) {
	out := richtext.Sanitize(raw)
	if len(out) > richtext.MaxHTMLLength || utf8.RuneCountInString(richtext.ToText(out)) > DescriptionMaxChars {
		return "", fieldError("note", "note must be at most 1000 characters")
	}
	if strings.TrimSpace(richtext.ToText(out)) == "" {
		return "", nil
	}
	return out, nil
}

// paymentMethod checks the chosen payment method, or picks the default: Cash, else the first
// active method (nil when none is set up).
func (s *InventoryService) paymentMethod(ctx context.Context, raw string) (*uuid.UUID, error) {
	var pm models.PaymentMethod
	if strings.TrimSpace(raw) != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			return nil, fieldError("payment_method_id", "choose a valid payment method")
		}
		if err := s.db.WithContext(ctx).First(&pm, "id = ? AND is_active = ?", id, true).Error; err != nil {
			return nil, fieldError("payment_method_id", "this payment method was not found or is inactive")
		}
		return &pm.ID, nil
	}
	err := s.db.WithContext(ctx).Where("is_active = ?", true).Order("CASE WHEN code = 'cash' THEN 0 ELSE 1 END, created_at ASC").First(&pm).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &pm.ID, nil
}

func pageArgs(page, size int) (int, int) {
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 500 {
		size = 20
	}
	return page, size
}

// --- product categories ---

type ProductCategoryInput struct {
	Code        string `json:"code" binding:"required,min=2,max=64"`
	Name        string `json:"name" binding:"required,min=2,max=128"`
	Description string `json:"description" binding:"max=512"`
	IsActive    *bool  `json:"is_active"`
}

func (s *InventoryService) ListCategories(ctx context.Context, includeInactive bool) ([]models.ProductCategory, error) {
	q := s.db.WithContext(ctx).Order("name ASC")
	if !includeInactive {
		q = q.Where("is_active = ?", true)
	}
	var out []models.ProductCategory
	return out, q.Find(&out).Error
}

func (s *InventoryService) SaveCategory(ctx context.Context, id *uuid.UUID, in ProductCategoryInput) (*models.ProductCategory, error) {
	code := strings.ToLower(strings.TrimSpace(in.Code))
	var cat models.ProductCategory
	if id != nil {
		if err := s.db.WithContext(ctx).First(&cat, "id = ?", *id).Error; err != nil {
			return nil, notFound(err)
		}
	} else {
		cat = models.ProductCategory{Base: models.Base{ID: uuid.New()}, IsActive: true}
	}
	var clash int64
	q := s.db.WithContext(ctx).Model(&models.ProductCategory{}).Where("code = ?", code)
	if id != nil {
		q = q.Where("id <> ?", *id)
	}
	if err := q.Count(&clash).Error; err != nil {
		return nil, err
	}
	if clash > 0 {
		return nil, fieldError("code", "a product category with this code already exists")
	}
	cat.Code, cat.Name, cat.Description = code, strings.TrimSpace(in.Name), strings.TrimSpace(in.Description)
	if in.IsActive != nil {
		cat.IsActive = *in.IsActive
	}
	if err := s.db.WithContext(ctx).Save(&cat).Error; err != nil {
		return nil, err
	}
	return &cat, nil
}

// --- products ---

// ProductInput: the SKU is generated when a product is added and never changes.
type ProductInput struct {
	Name         string `json:"name" binding:"required,min=2,max=191"`
	CategoryID   string `json:"category_id"`
	Unit         string `json:"unit" binding:"max=32"`
	Barcode      string `json:"barcode" binding:"max=64"`
	SalePrice    string `json:"sale_price"` // prices and costs are in USD
	CostPrice    string `json:"cost_price"`
	ReorderLevel string `json:"reorder_level"`
	Description  string `json:"description" binding:"max=512"`
	// ImageURL: a picture on the web (https://…). Empty keeps an uploaded picture.
	ImageURL string `json:"image_url" binding:"max=1024"`
	IsActive *bool  `json:"is_active"`
}

// ProductRow is a product with its category name and stock (in one branch, or every branch in
// the scope added up).
type ProductRow struct {
	models.Product
	CategoryName string          `json:"category_name"`
	HasImage     bool            `json:"has_image"` // an uploaded picture (GET /inventory/products/:id/image)
	Quantity     decimal.Decimal `json:"quantity"`
	AvgCost      decimal.Decimal `json:"avg_cost"` // one branch only
}

type ProductFilter struct {
	Search     string
	CategoryID *uuid.UUID
	Active     *bool
	BranchID   *uuid.UUID // stock of this branch (else the scope's total)
	Page       int
	PageSize   int
}

func (s *InventoryService) ListProducts(ctx context.Context, f ProductFilter, scope BranchScope) ([]ProductRow, int64, error) {
	q := s.db.WithContext(ctx).Model(&models.Product{})
	if strings.TrimSpace(f.Search) != "" {
		like := textsearch.Contains(f.Search)
		q = q.Where("products.sku LIKE ? OR products.name LIKE ? OR products.barcode LIKE ?", like, like, like)
	}
	if f.CategoryID != nil {
		q = q.Where("products.category_id = ?", *f.CategoryID)
	}
	if f.Active != nil {
		q = q.Where("products.is_active = ?", *f.Active)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	page, size := pageArgs(f.Page, f.PageSize)
	var products []models.Product
	if err := q.Order("products.name ASC").Limit(size).Offset((page - 1) * size).Find(&products).Error; err != nil {
		return nil, 0, err
	}
	rows := make([]ProductRow, len(products))
	ids := make([]uuid.UUID, len(products))
	catIDs := []uuid.UUID{}
	for i, p := range products {
		rows[i] = ProductRow{Product: p, HasImage: p.ImageKey != ""}
		ids[i] = p.ID
		if p.CategoryID != nil {
			catIDs = append(catIDs, *p.CategoryID)
		}
	}
	if len(ids) == 0 {
		return rows, total, nil
	}
	names := map[uuid.UUID]string{}
	if len(catIDs) > 0 {
		var cats []models.ProductCategory
		if err := s.db.WithContext(ctx).Where("id IN ?", catIDs).Find(&cats).Error; err != nil {
			return nil, 0, err
		}
		for _, c := range cats {
			names[c.ID] = c.Name
		}
	}
	type stockSum struct {
		ProductID uuid.UUID
		Quantity  decimal.Decimal
		AvgCost   decimal.Decimal
	}
	var sums []stockSum
	sq := s.db.WithContext(ctx).Model(&models.StockLevel{}).Where("product_id IN ?", ids)
	if f.BranchID != nil {
		sq = sq.Where("branch_id = ?", *f.BranchID).Select("product_id, quantity, avg_cost")
	} else {
		sq = scope.where(sq, "branch_id").Select("product_id, SUM(quantity) AS quantity, 0 AS avg_cost").Group("product_id")
	}
	if err := sq.Scan(&sums).Error; err != nil {
		return nil, 0, err
	}
	stock := map[uuid.UUID]stockSum{}
	for _, x := range sums {
		stock[x.ProductID] = x
	}
	for i := range rows {
		if rows[i].CategoryID != nil {
			rows[i].CategoryName = names[*rows[i].CategoryID]
		}
		rows[i].Quantity = stock[rows[i].ID].Quantity
		rows[i].AvgCost = stock[rows[i].ID].AvgCost
	}
	return rows, total, nil
}

func (s *InventoryService) SaveProduct(ctx context.Context, id *uuid.UUID, in ProductInput) (*models.Product, error) {
	var p models.Product
	if id != nil {
		if err := s.db.WithContext(ctx).First(&p, "id = ?", *id).Error; err != nil {
			return nil, notFound(err)
		}
	} else {
		p = models.Product{Base: models.Base{ID: uuid.New()}, IsActive: true}
		sku, err := s.newSKU(ctx)
		if err != nil {
			return nil, err
		}
		p.SKU = sku
	}
	imageURL := strings.TrimSpace(in.ImageURL)
	if imageURL != "" {
		u, err := url.Parse(imageURL)
		if err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Host == "" {
			return nil, fieldError("image_url", "enter a picture address starting with https://")
		}
	}
	var categoryID *uuid.UUID
	if strings.TrimSpace(in.CategoryID) != "" {
		cid, err := uuid.Parse(in.CategoryID)
		if err != nil {
			return nil, fieldError("category_id", "choose a valid category")
		}
		var cat models.ProductCategory
		if err := s.db.WithContext(ctx).First(&cat, "id = ?", cid).Error; err != nil {
			return nil, fieldError("category_id", "this category was not found")
		}
		categoryID = &cid
	}
	sale, err := parseMoney("sale_price", in.SalePrice)
	if err != nil {
		return nil, err
	}
	cost, err := parseMoney("cost_price", in.CostPrice)
	if err != nil {
		return nil, err
	}
	reorder := decimal.Zero
	if strings.TrimSpace(in.ReorderLevel) != "" {
		reorder, err = decimal.NewFromString(strings.TrimSpace(in.ReorderLevel))
		if err != nil || reorder.IsNegative() || reorder.Exponent() < -4 || reorder.GreaterThan(maxQuantity) {
			return nil, fieldError("reorder_level", "low-stock level must be 0 or more")
		}
	}
	unit := strings.TrimSpace(in.Unit)
	if unit == "" {
		unit = "pcs"
	}
	p.Name, p.CategoryID, p.Unit = strings.TrimSpace(in.Name), categoryID, unit
	p.Barcode, p.Currency, p.SalePrice, p.CostPrice = strings.TrimSpace(in.Barcode), models.CurrencyUSD, sale, cost
	p.ReorderLevel, p.Description = reorder, strings.TrimSpace(in.Description)
	if in.IsActive != nil {
		p.IsActive = *in.IsActive
	}
	// A picture address replaces an uploaded picture.
	oldKey := ""
	if imageURL != "" {
		oldKey, p.ImageKey, p.ImageType = p.ImageKey, "", ""
	}
	p.ImageURL = imageURL
	if err := s.db.WithContext(ctx).Save(&p).Error; err != nil {
		return nil, err
	}
	s.removeStored(ctx, oldKey)
	return &p, nil
}

// newSKU is the next free SKU-000001 style code.
func (s *InventoryService) newSKU(ctx context.Context) (string, error) {
	for i := 0; i < 20; i++ {
		sku, err := s.numbers.NextPlain(ctx, "SKU")
		if err != nil {
			return "", err
		}
		var taken int64
		if err := s.db.WithContext(ctx).Unscoped().Model(&models.Product{}).Where("sku = ?", sku).Count(&taken).Error; err != nil {
			return "", err
		}
		if taken == 0 {
			return sku, nil
		}
	}
	return "", errors.New("could not find a free SKU")
}

// --- product pictures ---

// MaxProductImageBytes caps an uploaded product picture.
const MaxProductImageBytes = 5 << 20

// ProductImageTypes are the picture formats a product may have.
var ProductImageTypes = map[string]string{"image/jpeg": "jpg", "image/png": "png", "image/webp": "webp", "image/gif": "gif"}

// ErrNoProductImage: the product has no uploaded picture.
var ErrNoProductImage = errors.New("this product has no uploaded picture")

func (s *InventoryService) removeStored(ctx context.Context, key string) {
	if key != "" && s.store != nil {
		_ = s.store.Delete(ctx, key)
	}
}

// SetProductImage stores an uploaded picture (replacing an earlier one or a picture address).
func (s *InventoryService) SetProductImage(ctx context.Context, id uuid.UUID, contentType string, size int64, r io.Reader) (*models.Product, error) {
	if s.store == nil {
		return nil, errors.New("file storage is not set up")
	}
	ext, ok := ProductImageTypes[contentType]
	if !ok {
		return nil, fieldError("image", "use a JPG, PNG, WebP or GIF picture")
	}
	var p models.Product
	if err := s.db.WithContext(ctx).First(&p, "id = ?", id).Error; err != nil {
		return nil, notFound(err)
	}
	key := fmt.Sprintf("products/%s/%s.%s", p.ID, uuid.NewString()[:8], ext)
	if err := s.store.Put(ctx, key, r, size, contentType); err != nil {
		return nil, fmt.Errorf("could not store the picture: %w", err)
	}
	oldKey := p.ImageKey
	if err := s.db.WithContext(ctx).Model(&p).Updates(map[string]interface{}{"image_key": key, "image_type": contentType, "image_url": ""}).Error; err != nil {
		s.removeStored(ctx, key)
		return nil, err
	}
	s.removeStored(ctx, oldKey)
	p.ImageKey, p.ImageType, p.ImageURL = key, contentType, ""
	return &p, nil
}

// ClearProductImage removes the picture (uploaded or address).
func (s *InventoryService) ClearProductImage(ctx context.Context, id uuid.UUID) error {
	var p models.Product
	if err := s.db.WithContext(ctx).First(&p, "id = ?", id).Error; err != nil {
		return notFound(err)
	}
	// Updates writes the new values into p, so keep the stored file's key first.
	oldKey := p.ImageKey
	if err := s.db.WithContext(ctx).Model(&p).Updates(map[string]interface{}{"image_key": "", "image_type": "", "image_url": ""}).Error; err != nil {
		return err
	}
	s.removeStored(ctx, oldKey)
	return nil
}

// OpenProductImage opens the uploaded picture.
func (s *InventoryService) OpenProductImage(ctx context.Context, id uuid.UUID) (io.ReadCloser, string, error) {
	var p models.Product
	if err := s.db.WithContext(ctx).Select("id", "image_key", "image_type").First(&p, "id = ?", id).Error; err != nil {
		return nil, "", notFound(err)
	}
	if p.ImageKey == "" || s.store == nil {
		return nil, "", ErrNoProductImage
	}
	rc, err := s.store.Open(ctx, p.ImageKey)
	if err != nil {
		return nil, "", ErrNoProductImage
	}
	return rc, p.ImageType, nil
}

// --- stock ---

type StockFilter struct {
	BranchID   *uuid.UUID
	CategoryID *uuid.UUID
	Search     string
	LowOnly    bool
	Page       int
	PageSize   int
}

// StockRow is one product's stock in one branch.
type StockRow struct {
	BranchID     uuid.UUID       `json:"branch_id"`
	BranchCode   string          `json:"branch_code"`
	BranchName   string          `json:"branch_name"`
	ProductID    uuid.UUID       `json:"product_id"`
	SKU          string          `json:"sku"`
	Name         string          `json:"name"`
	Unit         string          `json:"unit"`
	Currency     string          `json:"currency"`
	CategoryName string          `json:"category_name"`
	Quantity     decimal.Decimal `json:"quantity"`
	AvgCost      decimal.Decimal `json:"avg_cost"`
	Value        decimal.Decimal `json:"value"`
	SalePrice    decimal.Decimal `json:"sale_price"`
	ReorderLevel decimal.Decimal `json:"reorder_level"`
	Low          bool            `json:"low"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

const stockSelect = `stock_levels.branch_id, branches.code AS branch_code, branches.name AS branch_name,
	stock_levels.product_id, products.sku, products.name, products.unit, products.currency,
	COALESCE(product_categories.name, '') AS category_name, stock_levels.quantity, stock_levels.avg_cost,
	products.sale_price, products.reorder_level, stock_levels.updated_at`

func (s *InventoryService) stockQuery(ctx context.Context, f StockFilter, scope BranchScope) *gorm.DB {
	q := s.db.WithContext(ctx).Table("stock_levels").
		Joins("JOIN products ON products.id = stock_levels.product_id AND products.deleted_at IS NULL").
		Joins("JOIN branches ON branches.id = stock_levels.branch_id").
		Joins("LEFT JOIN product_categories ON product_categories.id = products.category_id")
	q = scope.where(q, "stock_levels.branch_id")
	if f.BranchID != nil {
		q = q.Where("stock_levels.branch_id = ?", *f.BranchID)
	}
	if f.CategoryID != nil {
		q = q.Where("products.category_id = ?", *f.CategoryID)
	}
	if strings.TrimSpace(f.Search) != "" {
		like := textsearch.Contains(f.Search)
		q = q.Where("products.sku LIKE ? OR products.name LIKE ? OR products.barcode LIKE ?", like, like, like)
	}
	if f.LowOnly {
		q = q.Where("products.reorder_level > 0 AND stock_levels.quantity <= products.reorder_level")
	}
	return q
}

func finishStockRows(rows []StockRow) []StockRow {
	for i := range rows {
		rows[i].Value = rows[i].Quantity.Mul(rows[i].AvgCost).Round(4)
		rows[i].Low = rows[i].ReorderLevel.IsPositive() && rows[i].Quantity.LessThanOrEqual(rows[i].ReorderLevel)
	}
	return rows
}

func (s *InventoryService) ListStock(ctx context.Context, f StockFilter, scope BranchScope) ([]StockRow, int64, error) {
	var total int64
	if err := s.stockQuery(ctx, f, scope).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	page, size := pageArgs(f.Page, f.PageSize)
	var rows []StockRow
	err := s.stockQuery(ctx, f, scope).Select(stockSelect).
		Order("products.name ASC, branches.code ASC").Limit(size).Offset((page - 1) * size).Scan(&rows).Error
	return finishStockRows(rows), total, err
}

type MovementFilter struct {
	BranchID  *uuid.UUID
	ProductID *uuid.UUID
	Type      string
	From, To  *time.Time
	Page      int
	PageSize  int
}

// MovementRow is a stock movement with names for the history list.
type MovementRow struct {
	models.StockMovement
	BranchCode string `json:"branch_code"`
	SKU        string `json:"sku"`
	Name       string `json:"name"`
	Unit       string `json:"unit"`
	Currency   string `json:"currency"`
	CreatedBy  string `json:"created_by"`
}

func (s *InventoryService) ListMovements(ctx context.Context, f MovementFilter, scope BranchScope) ([]MovementRow, int64, error) {
	q := s.db.WithContext(ctx).Table("stock_movements").
		Joins("JOIN products ON products.id = stock_movements.product_id").
		Joins("JOIN branches ON branches.id = stock_movements.branch_id").
		Joins("LEFT JOIN users ON users.id = stock_movements.created_by_id")
	q = scope.where(q, "stock_movements.branch_id")
	if f.BranchID != nil {
		q = q.Where("stock_movements.branch_id = ?", *f.BranchID)
	}
	if f.ProductID != nil {
		q = q.Where("stock_movements.product_id = ?", *f.ProductID)
	}
	if f.Type != "" {
		q = q.Where("stock_movements.type = ?", f.Type)
	}
	if f.From != nil {
		q = q.Where("stock_movements.created_at >= ?", *f.From)
	}
	if f.To != nil {
		q = q.Where("stock_movements.created_at <= ?", *f.To)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	page, size := pageArgs(f.Page, f.PageSize)
	var rows []MovementRow
	err := q.Select(`stock_movements.*, branches.code AS branch_code, products.sku, products.name, products.unit,
		products.currency, COALESCE(NULLIF(users.full_name, ''), users.username, '') AS created_by`).
		Order("stock_movements.created_at DESC").Limit(size).Offset((page - 1) * size).Scan(&rows).Error
	return rows, total, err
}

// move is one change to a branch's stock of a product.
type move struct {
	branchID, productID uuid.UUID
	qty                 decimal.Decimal // + in, − out
	unitCost            decimal.Decimal // cost of stock coming in (or leaving, with atCost)
	atCost              bool            // stock leaving was brought in at unitCost (voiding a purchase or transfer)
	kind, refType       string
	refID               *uuid.UUID
	refNumber, note     string
	actor               uuid.UUID
}

// applyMove changes the stock inside tx (the stock row is locked) and records the movement. It
// returns the average cost before the change, the cost of stock going out. Stock never goes
// below zero.
func (s *InventoryService) applyMove(tx *gorm.DB, m move) (decimal.Decimal, error) {
	now := time.Now()
	seed := models.StockLevel{ID: uuid.New(), BranchID: m.branchID, ProductID: m.productID, UpdatedAt: now}
	if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&seed).Error; err != nil {
		return decimal.Zero, err
	}
	var level models.StockLevel
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("branch_id = ? AND product_id = ?", m.branchID, m.productID).First(&level).Error; err != nil {
		return decimal.Zero, err
	}
	before := level.AvgCost
	newQty, avg, ok := nextStock(level.Quantity, level.AvgCost, m.qty, m.unitCost, m.atCost)
	if !ok {
		return before, &stockShortError{productID: m.productID, branchID: m.branchID, have: level.Quantity}
	}
	if err := tx.Model(&models.StockLevel{}).Where("id = ?", level.ID).
		Updates(map[string]interface{}{"quantity": newQty, "avg_cost": avg, "updated_at": now}).Error; err != nil {
		return before, err
	}
	unitCost := m.unitCost
	if m.qty.IsNegative() && !m.atCost {
		unitCost = before
	}
	return before, tx.Create(&models.StockMovement{
		ID: uuid.New(), BranchID: m.branchID, ProductID: m.productID, Type: m.kind,
		Quantity: m.qty, UnitCost: unitCost, BalanceAfter: newQty,
		RefType: m.refType, RefID: m.refID, RefNumber: m.refNumber, Note: m.note,
		CreatedByID: m.actor, CreatedAt: now,
	}).Error
}

// nextStock works out the stock after a change, at average cost: stock coming in is re-weighted
// into the average; stock going out leaves at the average (which stays), unless atCost says it
// leaves at unitCost (undoing a purchase or transfer that brought it in). ok is false when the
// stock would go below zero.
func nextStock(qty, avg, delta, unitCost decimal.Decimal, atCost bool) (newQty, newAvg decimal.Decimal, ok bool) {
	newQty = qty.Add(delta)
	if newQty.IsNegative() {
		return qty, avg, false
	}
	newAvg = avg
	if delta.IsPositive() || (atCost && newQty.IsPositive()) {
		value := qty.Mul(avg).Add(delta.Mul(unitCost))
		if value.IsNegative() {
			value = decimal.Zero
		}
		newAvg = value.Div(newQty).Round(4)
	}
	return newQty, newAvg, true
}

// stockShortError: not enough stock; described with names by describeShort.
type stockShortError struct {
	productID, branchID uuid.UUID
	have                decimal.Decimal
}

func (e *stockShortError) Error() string { return "not enough stock" }

// describeShort turns a stock shortage into a field error naming the product and branch.
func (s *InventoryService) describeShort(ctx context.Context, err error) error {
	var short *stockShortError
	if !errors.As(err, &short) {
		return err
	}
	var p models.Product
	var b models.Branch
	s.db.WithContext(ctx).Select("sku", "name", "unit").First(&p, "id = ?", short.productID)
	s.db.WithContext(ctx).Select("code").First(&b, "id = ?", short.branchID)
	return fieldError("items", fmt.Sprintf("not enough %s (%s) in %s: %s %s left", p.Name, p.SKU, b.Code, short.have.String(), p.Unit))
}

// --- adjustments ---

type AdjustInput struct {
	BranchID  string `json:"branch_id" binding:"required,uuid"`
	ProductID string `json:"product_id" binding:"required,uuid"`
	Mode      string `json:"mode" binding:"required,oneof=add remove set"` // add, remove, or set the counted quantity
	Quantity  string `json:"quantity" binding:"required"`
	UnitCost  string `json:"unit_cost"` // cost of stock added (default: current average, else the product's cost)
	Reason    string `json:"reason" binding:"required,min=3,max=512"`
}

func (s *InventoryService) Adjust(ctx context.Context, in AdjustInput, scope BranchScope, actor uuid.UUID) (*models.StockLevel, error) {
	branchID, productID := uuid.MustParse(in.BranchID), uuid.MustParse(in.ProductID)
	if !scope.Has(branchID) {
		return nil, ErrBranchAccessDenied
	}
	var product models.Product
	if err := s.db.WithContext(ctx).First(&product, "id = ?", productID).Error; err != nil {
		return nil, fieldError("product_id", "this product was not found")
	}
	var qty decimal.Decimal
	var err error
	if in.Mode == "set" {
		qty, err = decimal.NewFromString(strings.TrimSpace(in.Quantity))
		if err != nil || qty.IsNegative() || qty.Exponent() < -4 || qty.GreaterThan(maxQuantity) {
			return nil, fieldError("quantity", "the counted quantity must be 0 or more")
		}
	} else if qty, err = parseQuantity("quantity", in.Quantity); err != nil {
		return nil, err
	}
	cost, err := parseMoney("unit_cost", in.UnitCost)
	if err != nil {
		return nil, err
	}
	var level models.StockLevel
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Lock the stock row first, so a count ("set") is measured against the stock as it is now.
		seed := models.StockLevel{ID: uuid.New(), BranchID: branchID, ProductID: productID, UpdatedAt: time.Now()}
		if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&seed).Error; err != nil {
			return err
		}
		var current models.StockLevel
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("branch_id = ? AND product_id = ?", branchID, productID).First(&current).Error; err != nil {
			return err
		}
		delta := qty
		switch in.Mode {
		case "remove":
			delta = qty.Neg()
		case "set":
			delta = qty.Sub(current.Quantity)
		}
		if delta.IsZero() {
			return fieldError("quantity", "the stock is already at this quantity")
		}
		if strings.TrimSpace(in.UnitCost) == "" {
			cost = current.AvgCost
			if cost.IsZero() {
				cost = product.CostPrice
			}
		}
		if _, err := s.applyMove(tx, move{
			branchID: branchID, productID: productID, qty: delta, unitCost: cost,
			kind: models.MoveAdjustment, refType: "adjustment", note: strings.TrimSpace(in.Reason), actor: actor,
		}); err != nil {
			return err
		}
		return tx.Where("branch_id = ? AND product_id = ?", branchID, productID).First(&level).Error
	})
	if err != nil {
		return nil, s.describeShort(ctx, err)
	}
	return &level, nil
}

// --- shared lookups ---

func notFound(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrInventoryNotFound
	}
	return err
}

// actorHas: Super Admin, or one of the actor's roles grants the code.
func (s *InventoryService) actorHas(ctx context.Context, actorID uuid.UUID, code string) bool {
	var actor models.User
	if err := s.db.WithContext(ctx).Preload("Roles.Permissions").First(&actor, "id = ?", actorID).Error; err != nil {
		return false
	}
	if actor.IsSuperAdmin {
		return true
	}
	for _, r := range actor.Roles {
		for _, p := range r.Permissions {
			if p.Code == code {
				return true
			}
		}
	}
	return false
}

// names looks up display names: users (full name or username), branches (code — name) and
// products, for the ids given.
type nameBook struct {
	users    map[uuid.UUID]string
	branches map[uuid.UUID]models.Branch
	products map[uuid.UUID]models.Product
}

func (s *InventoryService) lookup(ctx context.Context, users, branches, products []uuid.UUID) nameBook {
	book := nameBook{users: map[uuid.UUID]string{}, branches: map[uuid.UUID]models.Branch{}, products: map[uuid.UUID]models.Product{}}
	if len(users) > 0 {
		var list []models.User
		s.db.WithContext(ctx).Select("id", "username", "full_name").Where("id IN ?", users).Find(&list)
		for _, u := range list {
			book.users[u.ID] = u.FullName
			if u.FullName == "" {
				book.users[u.ID] = u.Username
			}
		}
	}
	if len(branches) > 0 {
		var list []models.Branch
		s.db.WithContext(ctx).Select("id", "code", "name").Where("id IN ?", branches).Find(&list)
		for _, b := range list {
			book.branches[b.ID] = b
		}
	}
	if len(products) > 0 {
		var list []models.Product
		s.db.WithContext(ctx).Unscoped().Select("id", "sku", "name", "unit", "currency").Where("id IN ?", products).Find(&list)
		for _, p := range list {
			book.products[p.ID] = p
		}
	}
	return book
}
