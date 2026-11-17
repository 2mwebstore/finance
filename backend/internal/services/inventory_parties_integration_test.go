package services

// The customer and supplier lists against a real MySQL database (only with TEST_MYSQL_DSN naming
// a scratch database whose name contains "test"; see platform_integration_test.go). The test
// branch is TPC1; everyone it adds is named "TPC …".

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

func TestCustomersAndSuppliers(t *testing.T) {
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

	branch := models.Branch{Code: "TPC1", Name: "Party test TPC1", IsActive: true}
	if err := db.Where(models.Branch{Code: branch.Code}).FirstOrCreate(&branch).Error; err != nil {
		t.Fatal(err)
	}
	clean := func() {
		db.Exec("DELETE qi FROM quotation_items qi JOIN quotations q ON q.id = qi.quotation_id WHERE q.branch_id = ?", branch.ID)
		db.Unscoped().Where("branch_id = ?", branch.ID).Delete(&models.Quotation{})
		db.Exec("DELETE si FROM sale_items si JOIN sales s ON s.id = si.sale_id WHERE s.branch_id = ?", branch.ID)
		db.Unscoped().Where("branch_id = ?", branch.ID).Delete(&models.Sale{})
		db.Exec("DELETE pi FROM purchase_items pi JOIN purchases p ON p.id = pi.purchase_id WHERE p.branch_id = ?", branch.ID)
		db.Unscoped().Where("branch_id = ?", branch.ID).Delete(&models.Purchase{})
		db.Where("branch_id = ?", branch.ID).Delete(&models.StockMovement{})
		db.Where("branch_id = ?", branch.ID).Delete(&models.StockLevel{})
		db.Unscoped().Where("name LIKE ?", "TPC %").Delete(&models.Product{})
		db.Exec("DELETE FROM customers WHERE name LIKE 'TPC %'")
		db.Exec("DELETE FROM suppliers WHERE name LIKE 'TPC %'")
	}
	clean()
	defer clean()
	user := models.User{Username: "tpc_tester", Email: "tpc_tester@example.test", PasswordHash: "x", FullName: "TPC Tester", IsActive: true}
	if err := db.Where(models.User{Username: user.Username}).FirstOrCreate(&user).Error; err != nil {
		t.Fatal(err)
	}
	actor := user.ID
	all := BranchScope{All: true}
	admin := CatalogScope{Branches: all, Full: true}
	var cat models.ProductCategory
	if err := db.Where(models.ProductCategory{Code: "tpc"}).Attrs(models.ProductCategory{Name: "TPC goods", IsActive: true}).FirstOrCreate(&cat).Error; err != nil {
		t.Fatal(err)
	}
	goods, err := svc.SaveProduct(ctx, nil, ProductInput{Name: "TPC Lamp", CategoryID: cat.ID.String(), SalePrice: "10", CostPrice: "6"}, admin)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Adjust(ctx, AdjustInput{BranchID: branch.ID.String(), ProductID: goods.ID.String(), Mode: "add", Quantity: "20", UnitCost: "6", Reason: "opening stock"}, all, actor); err != nil {
		t.Fatal(err)
	}
	today := time.Now().Format("2006-01-02")
	items := []DocItemInput{{ProductID: goods.ID.String(), Quantity: "1", UnitPrice: "10", UnitCost: "6"}}
	sale := func(customerID, name string) (*SaleView, error) {
		return svc.CreateSale(ctx, SaleInput{BranchID: branch.ID.String(), CustomerID: customerID, Customer: name, SaleDate: today, Currency: "USD", Items: items}, all, actor)
	}
	purchase := func(supplierID, name string) (*PurchaseView, error) {
		return svc.CreatePurchase(ctx, PurchaseInput{BranchID: branch.ID.String(), SupplierID: supplierID, Supplier: name, PurchaseDate: today, Currency: "USD", Items: items}, all, actor)
	}

	// Names typed before the lists existed: new lists start with them, linked to their documents.
	legacySale, err := sale("", "  TPC Legacy Buyer ")
	if err != nil || legacySale.CustomerID != nil || legacySale.Customer != "TPC Legacy Buyer" {
		t.Fatalf("typed customer: %+v %v", legacySale, err)
	}
	legacyPurchase, err := purchase("", "TPC Old Supplier")
	if err != nil || legacyPurchase.SupplierID != nil {
		t.Fatalf("typed supplier: %+v %v", legacyPurchase, err)
	}
	if err := db.Migrator().DropTable("customers", "suppliers"); err != nil {
		t.Fatal(err)
	}
	if err := database.EnsureRequiredTables(db); err != nil {
		t.Fatal(err)
	}
	var imported models.Party
	if err := db.Table("customers").Where("name = ?", "TPC Legacy Buyer").Take(&imported).Error; err != nil || !imported.IsActive {
		t.Fatalf("customer not imported: %+v %v", imported, err)
	}
	if got, _ := svc.GetSale(ctx, legacySale.ID, all); got.CustomerID == nil || *got.CustomerID != imported.ID {
		t.Fatalf("legacy sale not linked: %+v", got.CustomerID)
	}
	var importedSupplier models.Party
	if err := db.Table("suppliers").Where("name = ?", "TPC Old Supplier").Take(&importedSupplier).Error; err != nil {
		t.Fatalf("supplier not imported: %v", err)
	}
	if got, _ := svc.GetPurchase(ctx, legacyPurchase.ID, all); got.SupplierID == nil || *got.SupplierID != importedSupplier.ID {
		t.Fatalf("legacy purchase not linked: %+v", got.SupplierID)
	}

	// Adding someone: spaces are tidied, and a name is only once in a list.
	dara, err := svc.SaveParty(ctx, Customers, nil, PartyInput{Name: "  TPC   Sok  Dara ", Phone: " 012 345 678 ", Email: "dara@example.test", Address: "Phnom Penh"}, admin)
	if err != nil || dara.Name != "TPC Sok Dara" || dara.Phone != "012 345 678" || !dara.IsActive {
		t.Fatalf("add customer: %+v %v", dara, err)
	}
	if _, err := svc.SaveParty(ctx, Customers, nil, PartyInput{Name: "tpc sok dara"}, admin); err == nil || !strings.Contains(err.Error(), "already in the list") {
		t.Fatalf("duplicate customer: %v", err)
	}
	// The same name is fine in the other list.
	acme, err := svc.SaveParty(ctx, Suppliers, nil, PartyInput{Name: "TPC Sok Dara"}, admin)
	if err != nil {
		t.Fatalf("supplier named like a customer: %v", err)
	}

	// Documents pick someone from the list and take their name.
	s1, err := sale(dara.ID.String(), "ignored")
	if err != nil || s1.CustomerID == nil || *s1.CustomerID != dara.ID || s1.Customer != "TPC Sok Dara" {
		t.Fatalf("sale with customer: %+v %v", s1, err)
	}
	quote, err := svc.CreateQuotation(ctx, QuotationInput{BranchID: branch.ID.String(), CustomerID: dara.ID.String(), QuoteDate: today, Currency: "USD", Items: items}, all, actor)
	if err != nil || quote.CustomerID == nil || quote.Customer != "TPC Sok Dara" {
		t.Fatalf("quotation with customer: %+v %v", quote, err)
	}
	if _, err := svc.CreateQuotation(ctx, QuotationInput{BranchID: branch.ID.String(), QuoteDate: today, Currency: "USD", Items: items}, all, actor); err == nil || !strings.Contains(err.Error(), "customer") {
		t.Fatalf("quotation without customer: %v", err)
	}
	p1, err := purchase(acme.ID.String(), "")
	if err != nil || p1.SupplierID == nil || *p1.SupplierID != acme.ID || p1.Supplier != "TPC Sok Dara" {
		t.Fatalf("purchase with supplier: %+v %v", p1, err)
	}
	if _, err := sale("not-an-id", ""); err == nil || !strings.Contains(err.Error(), "from the list") {
		t.Fatalf("bad customer id: %v", err)
	}
	if _, err := sale(uuid.NewString(), ""); err == nil || !strings.Contains(err.Error(), "isn't in the list") {
		t.Fatalf("unknown customer: %v", err)
	}

	// Renaming someone renames them on their documents.
	dara, err = svc.SaveParty(ctx, Customers, &dara.ID, PartyInput{Name: "TPC Sok Dara Ltd", Phone: dara.Phone, Email: dara.Email, Address: dara.Address}, admin)
	if err != nil || dara.Name != "TPC Sok Dara Ltd" {
		t.Fatalf("rename: %+v %v", dara, err)
	}
	if got, _ := svc.GetSale(ctx, s1.ID, all); got.Customer != "TPC Sok Dara Ltd" {
		t.Fatalf("sale keeps the old name: %q", got.Customer)
	}
	if got, _ := svc.GetQuotation(ctx, quote.ID, all); got.Customer != "TPC Sok Dara Ltd" {
		t.Fatalf("quotation keeps the old name: %q", got.Customer)
	}

	// The list: search, how many documents name them, on/off filter.
	list, total, err := svc.ListParties(ctx, Customers, PartyFilter{Search: "Sok Dara"}, all)
	if err != nil || total != 1 || list[0].ID != dara.ID || list[0].Documents != 2 {
		t.Fatalf("list: %+v %d %v", list, total, err)
	}

	// Turned off: new documents can't pick them; their own documents keep them.
	off := false
	if _, err := svc.SaveParty(ctx, Customers, &dara.ID, PartyInput{Name: dara.Name, IsActive: &off}, admin); err != nil {
		t.Fatal(err)
	}
	if _, err := sale(dara.ID.String(), ""); err == nil || !strings.Contains(err.Error(), "turned off") {
		t.Fatalf("turned-off customer on a new sale: %v", err)
	}
	if _, err := svc.UpdateSale(ctx, s1.ID, SaleInput{BranchID: branch.ID.String(), CustomerID: dara.ID.String(), SaleDate: today, Currency: "USD",
		Items: []DocItemInput{{ProductID: goods.ID.String(), Quantity: "2", UnitPrice: "10"}}}, all, actor); err != nil {
		t.Fatalf("changing a sale of a turned-off customer: %v", err)
	}
	if _, err := svc.SaveParty(ctx, Customers, nil, PartyInput{Name: "TPC Sok Dara Ltd"}, admin); err == nil || !strings.Contains(err.Error(), "turn them on") {
		t.Fatalf("adding a turned-off name again: %v", err)
	}
	offList, _, _ := svc.ListParties(ctx, Customers, PartyFilter{Active: "false", Search: "TPC"}, all)
	if len(offList) != 1 || offList[0].ID != dara.ID {
		t.Fatalf("turned-off filter: %+v", offList)
	}
	// A quotation of theirs still becomes a sale for them.
	converted, err := svc.ConvertQuotation(ctx, quote.ID, ConvertQuotationInput{SaleDate: today}, all, actor)
	if err != nil || converted.CustomerID == nil || *converted.CustomerID != dara.ID || converted.Customer != "TPC Sok Dara Ltd" {
		t.Fatalf("convert for a turned-off customer: %+v %v", converted, err)
	}
	// Changing someone that isn't there.
	if _, err := svc.SaveParty(ctx, Suppliers, &dara.ID, PartyInput{Name: "TPC Nobody"}, admin); !errors.Is(err, ErrInventoryNotFound) {
		t.Fatalf("customer id in the suppliers list: %v", err)
	}
}
