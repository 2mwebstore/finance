package services

// Quotations against a real MySQL database (only with TEST_MYSQL_DSN naming a scratch database
// whose name contains "test"; see platform_integration_test.go). Test branches are TQT1 and TQT2.

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/cambodia-finance/income-expense/internal/database"
	"github.com/cambodia-finance/income-expense/internal/models"
	mysqldriver "github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestQuotations(t *testing.T) {
	dsn := os.Getenv("TEST_MYSQL_DSN")
	if dsn == "" {
		t.Skip("TEST_MYSQL_DSN is not set")
	}
	cfg, err := mysqldriver.ParseDSN(dsn)
	if err != nil || !strings.Contains(cfg.DBName, "test") {
		t.Fatalf("TEST_MYSQL_DSN must name a scratch database with \"test\" in its name, got %q", cfg.DBName)
	}
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	if err := database.AutoMigrate(db); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	svc := NewInventoryService(db, "Asia/Phnom_Penh", nil)

	var branches [2]models.Branch
	for i, code := range []string{"TQT1", "TQT2"} {
		b := models.Branch{Code: code, Name: "Quote test " + code, IsActive: true}
		if err := db.Where(models.Branch{Code: code}).FirstOrCreate(&b).Error; err != nil {
			t.Fatal(err)
		}
		branches[i] = b
	}
	ids := []uuid.UUID{branches[0].ID, branches[1].ID}
	clean := func() {
		db.Exec("DELETE qi FROM quotation_items qi JOIN quotations q ON q.id = qi.quotation_id WHERE q.branch_id IN ?", ids)
		db.Unscoped().Where("branch_id IN ?", ids).Delete(&models.Quotation{})
		db.Exec("DELETE si FROM sale_items si JOIN sales s ON s.id = si.sale_id WHERE s.branch_id IN ?", ids)
		db.Unscoped().Where("branch_id IN ?", ids).Delete(&models.Sale{})
		db.Where("branch_id IN ?", ids).Delete(&models.StockMovement{})
		db.Where("branch_id IN ?", ids).Delete(&models.StockLevel{})
		db.Unscoped().Where("name LIKE ?", "TQT %").Delete(&models.Product{})
	}
	clean()
	defer clean()
	user := models.User{Username: "tqt_tester", Email: "tqt_tester@example.test", PasswordHash: "x", FullName: "TQT Tester", IsActive: true}
	if err := db.Where(models.User{Username: user.Username}).FirstOrCreate(&user).Error; err != nil {
		t.Fatal(err)
	}
	actor := user.ID
	all := BranchScope{All: true}
	admin := CatalogScope{Branches: all, Full: true}
	other := BranchScope{IDs: []uuid.UUID{branches[1].ID}}

	var cat models.ProductCategory
	if err := db.Where(models.ProductCategory{Code: "tqt"}).Attrs(models.ProductCategory{Name: "TQT goods", IsActive: true}).FirstOrCreate(&cat).Error; err != nil {
		t.Fatal(err)
	}
	goods, err := svc.SaveProduct(ctx, nil, ProductInput{Name: "TQT Chair", CategoryID: cat.ID.String(), SalePrice: "10", CostPrice: "6"}, admin)
	if err != nil {
		t.Fatal(err)
	}
	service, err := svc.SaveProduct(ctx, nil, ProductInput{Name: "TQT Delivery", CategoryID: cat.ID.String(), Type: models.ProductService, SalePrice: "15"}, admin)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Adjust(ctx, AdjustInput{BranchID: branches[0].ID.String(), ProductID: goods.ID.String(), Mode: "add", Quantity: "5", UnitCost: "6", Reason: "opening stock"}, all, actor); err != nil {
		t.Fatal(err)
	}
	stock := func() decimal.Decimal {
		var lvl models.StockLevel
		db.Where("branch_id = ? AND product_id = ?", branches[0].ID, goods.ID).First(&lvl)
		return lvl.Quantity
	}
	today := time.Now().Format("2006-01-02")
	input := func(chairs string) QuotationInput {
		return QuotationInput{
			BranchID: branches[0].ID.String(), Customer: "TQT Customer", Contact: "012 345 678", QuoteDate: today, Currency: "USD",
			Discount: "1", Shipping: "4", Note: "<p>Delivery in 3 days</p>",
			Items: []DocItemInput{
				{ProductID: goods.ID.String(), Quantity: chairs, UnitPrice: "10", Discount: "2", Note: "blue"},
				{ProductID: service.ID.String(), Quantity: "1", UnitPrice: "15"},
			},
		}
	}

	// It is for a customer, can't end before it starts, and only in the person's branches.
	nobody := input("3")
	nobody.Customer = "  "
	if _, err := svc.CreateQuotation(ctx, nobody, all, actor); err == nil || !strings.Contains(err.Error(), "customer") {
		t.Fatalf("no customer: %v", err)
	}
	bad := input("3")
	bad.ValidUntil = time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	if _, err := svc.CreateQuotation(ctx, bad, all, actor); err == nil || !strings.Contains(err.Error(), "valid-until") {
		t.Fatalf("valid until before the date: %v", err)
	}
	if _, err := svc.CreateQuotation(ctx, input("3"), other, actor); !errors.Is(err, ErrBranchAccessDenied) {
		t.Fatalf("another branch: %v", err)
	}

	// 3 × 10 − 2 = 28, + 15 = 43 subtotal; − 1 + 4 = 46. No stock moves.
	in := input("3")
	in.ValidUntil = time.Now().AddDate(0, 0, 14).Format("2006-01-02")
	q, err := svc.CreateQuotation(ctx, in, all, actor)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(q.Number, "QUO-") || q.Status != models.QuoteOpen || !q.Subtotal.Equal(d("43")) || !q.Total.Equal(d("46")) || q.Expired ||
		q.Customer != "TQT Customer" || q.Contact != "012 345 678" || len(q.Lines) != 2 || q.Lines[0].Note != "blue" || q.CreatedBy != "TQT Tester" {
		t.Fatalf("new quotation: %+v", q)
	}
	if !stock().Equal(d("5")) {
		t.Fatalf("a quotation moved stock: %s", stock())
	}
	if _, err := svc.GetQuotation(ctx, q.ID, other); !errors.Is(err, ErrInventoryNotFound) {
		t.Fatalf("seen from another branch: %v", err)
	}
	// One quotation carries who it is from, for printing.
	if one, err := svc.GetQuotation(ctx, q.ID, all); err != nil || one.Seller == nil || one.Seller.Company == "" || one.Seller.Branch != branches[0].Name {
		t.Fatalf("seller: %+v %v", one, err)
	}

	// Accepted, then changed: it is open again. A quotation may offer more than is in stock.
	if v, err := svc.SetQuotationStatus(ctx, q.ID, models.QuoteAccepted, all, actor); err != nil || v.Status != models.QuoteAccepted || v.DecidedBy != "TQT Tester" {
		t.Fatalf("accept: %v %+v", err, v)
	}
	if _, err := svc.SetQuotationStatus(ctx, q.ID, models.QuoteAccepted, all, actor); err == nil {
		t.Fatal("accepted twice")
	}
	q, err = svc.UpdateQuotation(ctx, q.ID, input("6"), all, actor)
	if err != nil || q.Status != models.QuoteOpen || q.DecidedByID != nil || !q.Total.Equal(d("76")) || q.ValidUntil != nil {
		t.Fatalf("changed: %v %+v", err, q)
	}

	// Declined can't become a sale; open again it can, once there is enough stock.
	if _, err := svc.SetQuotationStatus(ctx, q.ID, models.QuoteDeclined, all, actor); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.ConvertQuotation(ctx, q.ID, ConvertQuotationInput{}, all, actor); err == nil || !strings.Contains(err.Error(), "declined") {
		t.Fatalf("declined into a sale: %v", err)
	}
	if _, err := svc.SetQuotationStatus(ctx, q.ID, models.QuoteOpen, all, actor); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.ConvertQuotation(ctx, q.ID, ConvertQuotationInput{}, all, actor); err == nil || !strings.Contains(err.Error(), "not enough TQT Chair") {
		t.Fatalf("6 chairs from 5 in stock: %v", err)
	}
	if v, _ := svc.GetQuotation(ctx, q.ID, all); v.Status != models.QuoteOpen || !stock().Equal(d("5")) {
		t.Fatalf("a refused sale changed things: %s, stock %s", v.Status, stock())
	}
	if _, err := svc.ConvertQuotation(ctx, q.ID, ConvertQuotationInput{}, other, actor); !errors.Is(err, ErrInventoryNotFound) {
		t.Fatalf("converted from another branch: %v", err)
	}

	// 4 chairs: the sale has the quotation's prices and takes them out of stock.
	q, err = svc.UpdateQuotation(ctx, q.ID, input("4"), all, actor)
	if err != nil {
		t.Fatal(err)
	}
	sale, err := svc.ConvertQuotation(ctx, q.ID, ConvertQuotationInput{SaleDate: today}, all, actor)
	if err != nil {
		t.Fatal(err)
	}
	// (A sale's lines come back in no set order: find the chairs.)
	chairNote := ""
	for _, l := range sale.Lines {
		if l.ProductID == goods.ID {
			chairNote = l.Note
		}
	}
	if !strings.HasPrefix(sale.Number, "SAL-") || !sale.Total.Equal(d("56")) || !sale.Subtotal.Equal(d("53")) || sale.Customer != "TQT Customer" ||
		sale.QuotationNumber != q.Number || sale.QuotationID == nil || *sale.QuotationID != q.ID || len(sale.Lines) != 2 || chairNote != "blue" ||
		!strings.Contains(sale.Note, "Delivery in 3 days") || !sale.TotalCost.Equal(d("24")) {
		t.Fatalf("the sale: %+v", sale)
	}
	if !stock().Equal(d("1")) {
		t.Fatalf("stock after the sale: %s", stock())
	}
	q, _ = svc.GetQuotation(ctx, q.ID, all)
	if q.Status != models.QuoteConverted || q.SaleID == nil || *q.SaleID != sale.ID || q.SaleNumber != sale.Number || q.ConvertedBy != "TQT Tester" || q.ConvertedAt == nil {
		t.Fatalf("converted quotation: %+v", q)
	}
	// Done: no second sale, no changes, no deleting.
	if _, err := svc.ConvertQuotation(ctx, q.ID, ConvertQuotationInput{}, all, actor); err == nil {
		t.Fatal("converted twice")
	}
	if _, err := svc.UpdateQuotation(ctx, q.ID, input("1"), all, actor); err == nil {
		t.Fatal("a converted quotation changed")
	}
	if _, err := svc.SetQuotationStatus(ctx, q.ID, models.QuoteOpen, all, actor); err == nil {
		t.Fatal("a converted quotation reopened")
	}
	if _, err := svc.DeleteQuotation(ctx, q.ID, all); err == nil {
		t.Fatal("a converted quotation deleted")
	}
	if !stock().Equal(d("1")) {
		t.Fatalf("stock moved again: %s", stock())
	}

	// An open quotation past its valid-until day is expired.
	old := input("1")
	old.Customer = "TQT Old customer"
	old.QuoteDate = time.Now().AddDate(0, 0, -10).Format("2006-01-02")
	old.ValidUntil = time.Now().AddDate(0, 0, -3).Format("2006-01-02")
	expired, err := svc.CreateQuotation(ctx, old, all, actor)
	if err != nil || !expired.Expired {
		t.Fatalf("expired: %v %+v", err, expired)
	}
	list := func(status, search string) []string {
		t.Helper()
		rows, total, err := svc.ListQuotations(ctx, DocFilter{BranchID: &branches[0].ID, Status: status, Search: search}, all)
		if err != nil || int(total) != len(rows) {
			t.Fatalf("list %q: %v %d", status, err, total)
		}
		out := []string{}
		for _, r := range rows {
			out = append(out, r.Number)
		}
		return out
	}
	if got := list("", ""); len(got) != 2 {
		t.Fatalf("all: %v", got)
	}
	if got := list(QuotationStatusExpired, ""); len(got) != 1 || got[0] != expired.Number {
		t.Fatalf("expired: %v", got)
	}
	if got := list(models.QuoteOpen, ""); len(got) != 0 {
		t.Fatalf("open (still valid): %v", got)
	}
	if got := list(models.QuoteConverted, ""); len(got) != 1 || got[0] != q.Number {
		t.Fatalf("converted: %v", got)
	}
	if got := list("", "old cust"); len(got) != 1 || got[0] != expired.Number {
		t.Fatalf("search: %v", got)
	}
	if rows, _, _ := svc.ListQuotations(ctx, DocFilter{}, other); len(rows) != 0 {
		t.Fatalf("another branch's list: %d", len(rows))
	}

	// Deleting: only in the person's branches; then it's gone.
	if _, err := svc.DeleteQuotation(ctx, expired.ID, other); !errors.Is(err, ErrInventoryNotFound) {
		t.Fatalf("deleted from another branch: %v", err)
	}
	if _, err := svc.DeleteQuotation(ctx, expired.ID, all); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.GetQuotation(ctx, expired.ID, all); !errors.Is(err, ErrInventoryNotFound) {
		t.Fatalf("after deleting: %v", err)
	}
	if got := list("", ""); len(got) != 1 {
		t.Fatalf("after deleting: %v", got)
	}
}
