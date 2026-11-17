package services

// Shared-or-one-branch lists (products, categories, customers, suppliers, payment methods)
// against a real MySQL database (only with TEST_MYSQL_DSN naming a scratch database whose name
// contains "test"; see platform_integration_test.go). The test branches are TBO1 and TBO2;
// everything it adds is named "TBO …" (codes tbo-…).

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
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestBranchOwnedLists(t *testing.T) {
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
	pms := NewPaymentMethodService(db)

	branches := []models.Branch{{Code: "TBO1", Name: "TBO One", IsActive: true}, {Code: "TBO2", Name: "TBO Two", IsActive: true}}
	for i := range branches {
		if err := db.Where(models.Branch{Code: branches[i].Code}).FirstOrCreate(&branches[i]).Error; err != nil {
			t.Fatal(err)
		}
	}
	a, b := branches[0].ID, branches[1].ID
	clean := func() {
		for _, br := range []uuid.UUID{a, b} {
			db.Exec("DELETE qi FROM quotation_items qi JOIN quotations q ON q.id = qi.quotation_id WHERE q.branch_id = ?", br)
			db.Unscoped().Where("branch_id = ?", br).Delete(&models.Quotation{})
			db.Exec("DELETE si FROM sale_items si JOIN sales s ON s.id = si.sale_id WHERE s.branch_id = ?", br)
			db.Unscoped().Where("branch_id = ?", br).Delete(&models.Sale{})
			db.Exec("DELETE pi FROM purchase_items pi JOIN purchases p ON p.id = pi.purchase_id WHERE p.branch_id = ?", br)
			db.Unscoped().Where("branch_id = ?", br).Delete(&models.Purchase{})
			db.Exec("DELETE ti FROM stock_transfer_items ti JOIN stock_transfers t ON t.id = ti.transfer_id WHERE t.from_branch_id = ?", br)
			db.Unscoped().Where("from_branch_id = ?", br).Delete(&models.StockTransfer{})
			db.Where("branch_id = ?", br).Delete(&models.StockMovement{})
			db.Where("branch_id = ?", br).Delete(&models.StockLevel{})
		}
		db.Unscoped().Where("name LIKE ?", "TBO %").Delete(&models.Product{})
		db.Unscoped().Where("code LIKE ?", "tbo-%").Delete(&models.ProductCategory{})
		db.Unscoped().Where("code LIKE ?", "tbo-%").Delete(&models.PaymentMethod{})
		db.Exec("DELETE FROM customers WHERE name LIKE 'TBO %'")
		db.Exec("DELETE FROM suppliers WHERE name LIKE 'TBO %'")
	}
	clean()
	defer clean()
	user := models.User{Username: "tbo_tester", Email: "tbo_tester@example.test", PasswordHash: "x", FullName: "TBO Tester", IsActive: true}
	if err := db.Where(models.User{Username: user.Username}).FirstOrCreate(&user).Error; err != nil {
		t.Fatal(err)
	}
	actor := user.ID
	all := BranchScope{All: true}
	onlyA, onlyB := BranchScope{IDs: []uuid.UUID{a}}, BranchScope{IDs: []uuid.UUID{b}}
	admin := CatalogScope{Branches: all, Full: true}
	staffA, staffB := CatalogScope{Branches: onlyA}, CatalogScope{Branches: onlyB}
	str := func(s string) *string { return &s }
	wantErr := func(what string, err error, part string) {
		t.Helper()
		if err == nil || !strings.Contains(err.Error(), part) {
			t.Fatalf("%s: want an error with %q, got %v", what, part, err)
		}
	}
	today := time.Now().Format("2006-01-02")

	// --- Product categories ---
	_, err = svc.SaveCategory(ctx, nil, ProductCategoryInput{Code: "tbo-shared-try", Name: "TBO try"}, staffA)
	wantErr("branch staff can't add a shared category", err, "only the owner or an admin")
	shared, err := svc.SaveCategory(ctx, nil, ProductCategoryInput{Code: "tbo-shared", Name: "TBO shared"}, admin)
	if err != nil || shared.BranchID != nil {
		t.Fatalf("shared category: %+v %v", shared, err)
	}
	catA, err := svc.SaveCategory(ctx, nil, ProductCategoryInput{Code: "tbo-a", Name: "TBO A goods", BranchID: str(a.String())}, staffA)
	if err != nil || catA.BranchID == nil || *catA.BranchID != a || catA.BranchName != "TBO One" {
		t.Fatalf("branch category: %+v %v", catA, err)
	}
	if _, err := svc.SaveCategory(ctx, nil, ProductCategoryInput{Code: "tbo-b-try", Name: "TBO B try", BranchID: str(b.String())}, staffA); !errors.Is(err, ErrBranchAccessDenied) {
		t.Fatalf("staff of A adding for B: %v", err)
	}
	codes := func(list []models.ProductCategory) string {
		var out []string
		for _, c := range list {
			if strings.HasPrefix(c.Code, "tbo-") {
				out = append(out, c.Code)
			}
		}
		return strings.Join(out, ",")
	}
	if list, _ := svc.ListCategories(ctx, true, onlyB, OwnerFilter{}); codes(list) != "tbo-shared" {
		t.Fatalf("B sees: %s", codes(list))
	}
	if list, _ := svc.ListCategories(ctx, true, onlyA, OwnerFilter{}); codes(list) != "tbo-a,tbo-shared" {
		t.Fatalf("A sees: %s", codes(list))
	}
	if list, _ := svc.ListCategories(ctx, true, all, OwnerFilter{For: &b}); codes(list) != "tbo-shared" {
		t.Fatalf("admin with B picked sees: %s", codes(list))
	}
	if list, _ := svc.ListCategories(ctx, true, all, OwnerFilter{Only: SharedOnly}); codes(list) != "tbo-shared" {
		t.Fatalf("shared only: %s", codes(list))
	}
	if _, err := svc.SaveCategory(ctx, &shared.ID, ProductCategoryInput{Code: "tbo-shared", Name: "TBO renamed"}, staffA); !errors.Is(err, ErrSharedOnlyFull) {
		t.Fatalf("staff changing a shared category: %v", err)
	}
	if _, err := svc.SaveCategory(ctx, &catA.ID, ProductCategoryInput{Code: "tbo-a", Name: "TBO A renamed"}, staffB); !errors.Is(err, ErrInventoryNotFound) {
		t.Fatalf("B changing A's category: %v", err)
	}

	// --- Products ---
	_, err = svc.SaveProduct(ctx, nil, ProductInput{Name: "TBO Try", CategoryID: shared.ID.String(), SalePrice: "5", CostPrice: "3"}, staffA)
	wantErr("staff can't add a shared product", err, "only the owner or an admin")
	_, err = svc.SaveProduct(ctx, nil, ProductInput{Name: "TBO Try", CategoryID: catA.ID.String(), SalePrice: "5", CostPrice: "3"}, admin)
	wantErr("shared product in A's category", err, "a shared product needs a shared category")
	lamp, err := svc.SaveProduct(ctx, nil, ProductInput{Name: "TBO Lamp", CategoryID: catA.ID.String(), SalePrice: "10", CostPrice: "6", BranchID: str(a.String())}, staffA)
	if err != nil || lamp.BranchID == nil || *lamp.BranchID != a {
		t.Fatalf("A's product: %+v %v", lamp, err)
	}
	cup, err := svc.SaveProduct(ctx, nil, ProductInput{Name: "TBO Cup", CategoryID: shared.ID.String(), SalePrice: "4", CostPrice: "2"}, admin)
	if err != nil || cup.BranchID != nil {
		t.Fatalf("shared product: %+v %v", cup, err)
	}
	names := func(rows []ProductRow) string {
		var out []string
		for _, r := range rows {
			if strings.HasPrefix(r.Name, "TBO ") {
				out = append(out, r.Name)
			}
		}
		return strings.Join(out, ",")
	}
	if rows, _, _ := svc.ListProducts(ctx, ProductFilter{Search: "TBO"}, onlyB); names(rows) != "TBO Cup" {
		t.Fatalf("B's products: %s", names(rows))
	}
	if rows, _, _ := svc.ListProducts(ctx, ProductFilter{Search: "TBO", BranchID: &a}, all); names(rows) != "TBO Cup,TBO Lamp" || rows[1].BranchName != "TBO One" {
		t.Fatalf("A's products: %s %+v", names(rows), rows)
	}
	if rows, _, _ := svc.ListProducts(ctx, ProductFilter{Search: "TBO", BranchID: &a, Owner: SharedOnly}, all); names(rows) != "TBO Cup" {
		t.Fatalf("shared products for a transfer: %s", names(rows))
	}
	if _, err := svc.SaveProduct(ctx, &lamp.ID, ProductInput{Name: "TBO Lamp", CategoryID: catA.ID.String(), SalePrice: "11", CostPrice: "6"}, staffB); !errors.Is(err, ErrInventoryNotFound) {
		t.Fatalf("B changing A's product: %v", err)
	}
	_, err = svc.Adjust(ctx, AdjustInput{BranchID: b.String(), ProductID: lamp.ID.String(), Mode: "add", Quantity: "5", Reason: "count"}, all, actor)
	wantErr("stocking A's product in B", err, "TBO One's own product")
	for _, br := range []uuid.UUID{a, b} {
		if _, err := svc.Adjust(ctx, AdjustInput{BranchID: br.String(), ProductID: cup.ID.String(), Mode: "add", Quantity: "10", UnitCost: "2", Reason: "opening"}, all, actor); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := svc.Adjust(ctx, AdjustInput{BranchID: a.String(), ProductID: lamp.ID.String(), Mode: "add", Quantity: "10", UnitCost: "6", Reason: "opening"}, all, actor); err != nil {
		t.Fatal(err)
	}
	line := func(p *models.Product) []DocItemInput {
		return []DocItemInput{{ProductID: p.ID.String(), Quantity: "1", UnitPrice: "10", UnitCost: "6"}}
	}
	_, err = svc.CreateSale(ctx, SaleInput{BranchID: b.String(), SaleDate: today, Currency: "USD", Items: line(lamp)}, all, actor)
	wantErr("selling A's product in B", err, "another branch can't use it")
	if _, err := svc.CreateSale(ctx, SaleInput{BranchID: a.String(), SaleDate: today, Currency: "USD", Items: line(lamp)}, all, actor); err != nil {
		t.Fatalf("selling A's product in A: %v", err)
	}
	_, err = svc.CreateTransfer(ctx, TransferInput{FromBranchID: a.String(), ToBranchID: b.String(), TransferDate: today, Items: line(lamp)}, all, actor)
	wantErr("moving A's product", err, "only shared products move between branches")
	if _, err := svc.CreateTransfer(ctx, TransferInput{FromBranchID: a.String(), ToBranchID: b.String(), TransferDate: today, Items: line(cup)}, all, actor); err != nil {
		t.Fatalf("moving a shared product: %v", err)
	}
	// Making something one branch's own: only when no other branch has used it.
	_, err = svc.SaveProduct(ctx, &cup.ID, ProductInput{Name: "TBO Cup", CategoryID: shared.ID.String(), SalePrice: "4", CostPrice: "2", BranchID: str(a.String())}, admin)
	wantErr("cup is stocked in both branches", err, "another branch")
	bowl, err := svc.SaveProduct(ctx, nil, ProductInput{Name: "TBO Bowl", CategoryID: shared.ID.String(), SalePrice: "4", CostPrice: "2"}, admin)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Adjust(ctx, AdjustInput{BranchID: a.String(), ProductID: bowl.ID.String(), Mode: "add", Quantity: "3", UnitCost: "2", Reason: "opening"}, all, actor); err != nil {
		t.Fatal(err)
	}
	if bowl, err = svc.SaveProduct(ctx, &bowl.ID, ProductInput{Name: "TBO Bowl", CategoryID: shared.ID.String(), SalePrice: "4", CostPrice: "2", BranchID: str(a.String())}, admin); err != nil || bowl.BranchID == nil || *bowl.BranchID != a {
		t.Fatalf("bowl used only in A becomes A's own: %+v %v", bowl, err)
	}
	// Leaving branch_id out keeps the owner.
	if bowl, err = svc.SaveProduct(ctx, &bowl.ID, ProductInput{Name: "TBO Bowl", CategoryID: shared.ID.String(), SalePrice: "5", CostPrice: "2"}, staffA); err != nil || bowl.BranchID == nil || *bowl.BranchID != a {
		t.Fatalf("A's staff edits A's bowl: %+v %v", bowl, err)
	}
	_, err = svc.SaveCategory(ctx, &shared.ID, ProductCategoryInput{Code: "tbo-shared", Name: "TBO shared", BranchID: str(b.String())}, admin)
	wantErr("a category with shared products can't be B's", err, "can't be TBO Two's own")

	// --- Customers and suppliers ---
	daraA, err := svc.SaveParty(ctx, Customers, nil, PartyInput{Name: "TBO Dara", BranchID: str(a.String())}, staffA)
	if err != nil || daraA.BranchID == nil || *daraA.BranchID != a {
		t.Fatalf("A's customer: %+v %v", daraA, err)
	}
	if _, err := svc.SaveParty(ctx, Customers, nil, PartyInput{Name: "TBO Dara", BranchID: str(b.String())}, staffB); err != nil {
		t.Fatalf("B can have its own of the same name: %v", err)
	}
	_, err = svc.SaveParty(ctx, Customers, nil, PartyInput{Name: "tbo dara", BranchID: str(a.String())}, staffA)
	wantErr("twice in A", err, "already in TBO One's list")
	_, err = svc.SaveParty(ctx, Customers, nil, PartyInput{Name: "TBO Dara"}, admin)
	wantErr("shared one of a branch's name", err, "already in TBO")
	_, err = svc.SaveParty(ctx, Customers, nil, PartyInput{Name: "TBO Someone"}, staffA)
	wantErr("staff adding a shared customer", err, "only the owner or an admin")
	if list, _, _ := svc.ListParties(ctx, Customers, PartyFilter{Search: "TBO"}, onlyB); len(list) != 1 || list[0].BranchID == nil || *list[0].BranchID != b || list[0].BranchName != "TBO Two" {
		t.Fatalf("B's customers: %+v", list)
	}
	_, err = svc.CreateSale(ctx, SaleInput{BranchID: b.String(), CustomerID: daraA.ID.String(), SaleDate: today, Currency: "USD", Items: line(cup)}, all, actor)
	wantErr("B's sale for A's customer", err, "in TBO One's list only")
	if s, err := svc.CreateSale(ctx, SaleInput{BranchID: a.String(), CustomerID: daraA.ID.String(), SaleDate: today, Currency: "USD", Items: line(cup)}, all, actor); err != nil || s.CustomerID == nil || *s.CustomerID != daraA.ID {
		t.Fatalf("A's sale for A's customer: %+v %v", s, err)
	}
	both, err := svc.SaveParty(ctx, Customers, nil, PartyInput{Name: "TBO Both"}, admin)
	if err != nil {
		t.Fatal(err)
	}
	for _, br := range []uuid.UUID{a, b} {
		if _, err := svc.CreateQuotation(ctx, QuotationInput{BranchID: br.String(), CustomerID: both.ID.String(), QuoteDate: today, Currency: "USD", Items: line(cup)}, all, actor); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := svc.SaveParty(ctx, Customers, &both.ID, PartyInput{Name: "TBO Both"}, staffA); !errors.Is(err, ErrSharedOnlyFull) {
		t.Fatalf("staff changing a shared customer: %v", err)
	}
	_, err = svc.SaveParty(ctx, Customers, &both.ID, PartyInput{Name: "TBO Both", BranchID: str(a.String())}, admin)
	wantErr("customer on both branches' quotations", err, "on another branch's documents")
	acme, err := svc.SaveParty(ctx, Suppliers, nil, PartyInput{Name: "TBO Acme", BranchID: str(b.String())}, staffB)
	if err != nil {
		t.Fatal(err)
	}
	_, err = svc.CreatePurchase(ctx, PurchaseInput{BranchID: a.String(), SupplierID: acme.ID.String(), PurchaseDate: today, Currency: "USD", Items: line(cup)}, all, actor)
	wantErr("A buying from B's supplier", err, "in TBO Two's list only")

	// --- Payment methods ---
	_, err = pms.Create(ctx, CreatePaymentMethodInput{Code: "tbo-try", Name: "TBO try"}, staffA)
	wantErr("staff adding a shared method", err, "only the owner or an admin")
	abaA, err := pms.Create(ctx, CreatePaymentMethodInput{Code: "tbo-aba", Name: "TBO ABA", BranchID: a.String()}, staffA)
	if err != nil || abaA.BranchID == nil || abaA.BranchName != "TBO One" {
		t.Fatalf("A's method: %+v %v", abaA, err)
	}
	listB, err := pms.List(ctx, false, CategoryListScope{All: true, ForBranch: &b})
	if err != nil {
		t.Fatal(err)
	}
	for _, pm := range listB {
		if pm.ID == abaA.ID {
			t.Fatal("B sees A's payment method")
		}
	}
	_, err = svc.CreateSale(ctx, SaleInput{BranchID: b.String(), PaymentMethodID: abaA.ID.String(), SaleDate: today, Currency: "USD", Items: line(cup)}, all, actor)
	wantErr("B paid with A's method", err, "TBO One's own payment method")
	if s, err := svc.CreateSale(ctx, SaleInput{BranchID: a.String(), PaymentMethodID: abaA.ID.String(), SaleDate: today, Currency: "USD", Items: line(cup)}, all, actor); err != nil || s.PaymentMethodID == nil || *s.PaymentMethodID != abaA.ID {
		t.Fatalf("A paid with A's method: %v", err)
	}
	if _, err := pms.Update(ctx, abaA.ID, UpdatePaymentMethodInput{BranchID: str(b.String())}, admin); err == nil || !strings.Contains(err.Error(), "another branch's records") {
		t.Fatalf("A's used method moved to B: %v", err)
	}
	if _, err := pms.Update(ctx, abaA.ID, UpdatePaymentMethodInput{Name: str("TBO ABA 2")}, staffB); !errors.Is(err, ErrPaymentMethodNotFound) {
		t.Fatalf("B changing A's method: %v", err)
	}
	system := models.PaymentMethod{Base: models.Base{ID: uuid.New()}, Code: "tbo-sys", Name: "TBO Built-in", IsActive: true, IsSystem: true}
	if err := db.Create(&system).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := pms.Update(ctx, system.ID, UpdatePaymentMethodInput{BranchID: str(a.String())}, admin); err == nil || !strings.Contains(err.Error(), "built in") {
		t.Fatalf("built-in method made A's own: %v", err)
	}
	if _, err := pms.Update(ctx, system.ID, UpdatePaymentMethodInput{Name: str("TBO Built-in 2")}, staffA); !errors.Is(err, ErrSharedOnlyFull) {
		t.Fatalf("staff changing a shared method: %v", err)
	}
}
