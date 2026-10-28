package controllers

import (
	"bytes"
	"errors"
	"io"
	"mime"
	"net/http"
	"strconv"
	"strings"

	"github.com/cambodia-finance/income-expense/internal/audit"
	"github.com/cambodia-finance/income-expense/internal/models"
	"github.com/cambodia-finance/income-expense/internal/permissions"
	"github.com/cambodia-finance/income-expense/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// InventoryController: products, stock per branch, sales, purchases, transfers and inventory
// reports. Everything is limited to the actor's own branches (owner/admin see every branch).
type InventoryController struct {
	svc   *services.InventoryService
	db    *gorm.DB
	audit *audit.Service
}

func NewInventoryController(svc *services.InventoryService, db *gorm.DB, auditSvc *audit.Service) *InventoryController {
	return &InventoryController{svc: svc, db: db, audit: auditSvc}
}

func (ic *InventoryController) scope(c *gin.Context) (*accessScope, services.BranchScope, bool) {
	scope, ok := loadScope(c, ic.db)
	if !ok {
		return nil, services.BranchScope{}, false
	}
	return scope, services.BranchScope{All: scope.AllBranches, IDs: scope.BranchIDs}, true
}

func writeInventoryError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, services.ErrInventoryNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
	case errors.Is(err, services.ErrBranchAccessDenied):
		c.JSON(http.StatusForbidden, gin.H{"error": "you can only work in your own branches"})
	default:
		writeServiceError(c, err)
	}
}

func (ic *InventoryController) record(c *gin.Context, actor uuid.UUID, action, entity string, branch *uuid.UUID, details map[string]interface{}) {
	if ic.audit == nil {
		return
	}
	ic.audit.Record(c.Request.Context(), audit.Event{
		ActorID: &actor, Action: action, EntityType: "inventory", EntityID: entity, BranchID: branch,
		Outcome: "success", IPAddress: c.ClientIP(), UserAgent: c.Request.UserAgent(), Details: details,
	})
}

// queryID reads an optional UUID query parameter; it answers 400 when the value is not one.
func queryID(c *gin.Context, name string) (*uuid.UUID, bool) {
	raw := strings.TrimSpace(c.Query(name))
	if raw == "" {
		return nil, true
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": name + " is not valid", "fields": gin.H{name: "not valid"}})
		return nil, false
	}
	return &id, true
}

func paramID(c *gin.Context) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return uuid.Nil, false
	}
	return id, true
}

func pageQuery(c *gin.Context) (int, int) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	return page, size
}

// orEmpty keeps an empty result a JSON list ([]) rather than null.
func orEmpty[T any](items []T) []T {
	if items == nil {
		return []T{}
	}
	return items
}

func listJSON(c *gin.Context, items interface{}, total int64) {
	page, size := pageQuery(c)
	c.JSON(http.StatusOK, gin.H{"items": items, "total": total, "page": page, "page_size": size})
}

// docFilter reads the shared list filters (branch, status, search, dates, page).
func (ic *InventoryController) docFilter(c *gin.Context) (services.DocFilter, bool) {
	branchID, ok := queryID(c, "branch_id")
	if !ok {
		return services.DocFilter{}, false
	}
	from, to, err := ic.svc.DateRange(c.Query("date_from"), c.Query("date_to"))
	if err != nil {
		writeInventoryError(c, err)
		return services.DocFilter{}, false
	}
	page, size := pageQuery(c)
	return services.DocFilter{BranchID: branchID, Status: c.Query("status"), Search: c.Query("search"), From: from, To: to, Page: page, PageSize: size}, true
}

// ===== Catalog =====

func (ic *InventoryController) ListCategories(c *gin.Context) {
	cats, err := ic.svc.ListCategories(c.Request.Context(), c.Query("include_inactive") == "true")
	if err != nil {
		writeInventoryError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": orEmpty(cats)})
}

func (ic *InventoryController) SaveCategory(c *gin.Context) {
	var id *uuid.UUID
	if c.Param("id") != "" {
		parsed, ok := paramID(c)
		if !ok {
			return
		}
		id = &parsed
	}
	var in services.ProductCategoryInput
	if !bindJSON(c, &in) {
		return
	}
	cat, err := ic.svc.SaveCategory(c.Request.Context(), id, in)
	if err != nil {
		writeInventoryError(c, err)
		return
	}
	if actor, ok := getActorID(c); ok {
		ic.record(c, actor, "inventory.category.save", cat.ID.String(), nil, map[string]interface{}{"code": cat.Code, "name": cat.Name})
	}
	status := http.StatusOK
	if id == nil {
		status = http.StatusCreated
	}
	c.JSON(status, cat)
}

func (ic *InventoryController) ListProducts(c *gin.Context) {
	_, scope, ok := ic.scope(c)
	if !ok {
		return
	}
	categoryID, ok := queryID(c, "category_id")
	if !ok {
		return
	}
	branchID, ok := queryID(c, "branch_id")
	if !ok {
		return
	}
	if branchID != nil && !scope.Has(*branchID) {
		writeInventoryError(c, services.ErrBranchAccessDenied)
		return
	}
	f := services.ProductFilter{Search: c.Query("search"), CategoryID: categoryID, BranchID: branchID}
	if v := c.Query("active"); v != "" {
		active := v == "true"
		f.Active = &active
	}
	f.Page, f.PageSize = pageQuery(c)
	rows, total, err := ic.svc.ListProducts(c.Request.Context(), f, scope)
	if err != nil {
		writeInventoryError(c, err)
		return
	}
	listJSON(c, orEmpty(rows), total)
}

func (ic *InventoryController) SaveProduct(c *gin.Context) {
	var id *uuid.UUID
	if c.Param("id") != "" {
		parsed, ok := paramID(c)
		if !ok {
			return
		}
		id = &parsed
	}
	var in services.ProductInput
	if !bindJSON(c, &in) {
		return
	}
	p, err := ic.svc.SaveProduct(c.Request.Context(), id, in)
	if err != nil {
		writeInventoryError(c, err)
		return
	}
	if actor, ok := getActorID(c); ok {
		ic.record(c, actor, "inventory.product.save", p.ID.String(), nil, map[string]interface{}{"sku": p.SKU, "name": p.Name, "active": p.IsActive})
	}
	status := http.StatusOK
	if id == nil {
		status = http.StatusCreated
	}
	c.JSON(status, p)
}

// --- product pictures ---

// UploadProductImage stores a picture for the product (JPG, PNG, WebP or GIF, up to 5 MB).
func (ic *InventoryController) UploadProductImage(c *gin.Context) {
	id, ok := paramID(c)
	if !ok {
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, services.MaxProductImageBytes+(1<<20))
	fh, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "choose a picture (up to 5 MB)"})
		return
	}
	if fh.Size > services.MaxProductImageBytes {
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "the picture is larger than 5 MB"})
		return
	}
	f, err := fh.Open()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "could not read the file"})
		return
	}
	defer f.Close()
	head := make([]byte, 512)
	n, err := io.ReadFull(f, head)
	if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) && !errors.Is(err, io.EOF) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "could not read the file"})
		return
	}
	// Trust the bytes, not the file name or the browser's content type.
	contentType, _, _ := mime.ParseMediaType(http.DetectContentType(head[:n]))
	if _, ok := services.ProductImageTypes[contentType]; !ok {
		c.JSON(http.StatusUnsupportedMediaType, gin.H{"error": "use a JPG, PNG, WebP or GIF picture"})
		return
	}
	p, err := ic.svc.SetProductImage(c.Request.Context(), id, contentType, fh.Size, io.MultiReader(bytes.NewReader(head[:n]), f))
	if err != nil {
		writeInventoryError(c, err)
		return
	}
	if actor, ok := getActorID(c); ok {
		ic.record(c, actor, "inventory.product.image", p.ID.String(), nil, map[string]interface{}{"sku": p.SKU, "type": contentType, "size": fh.Size})
	}
	c.JSON(http.StatusOK, gin.H{"id": p.ID, "has_image": true, "image_url": ""})
}

func (ic *InventoryController) DeleteProductImage(c *gin.Context) {
	id, ok := paramID(c)
	if !ok {
		return
	}
	if err := ic.svc.ClearProductImage(c.Request.Context(), id); err != nil {
		writeInventoryError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": id, "has_image": false, "image_url": ""})
}

func (ic *InventoryController) ProductImage(c *gin.Context) {
	id, ok := paramID(c)
	if !ok {
		return
	}
	rc, contentType, err := ic.svc.OpenProductImage(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, services.ErrNoProductImage) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		writeInventoryError(c, err)
		return
	}
	defer rc.Close()
	c.DataFromReader(http.StatusOK, -1, contentType, rc, map[string]string{
		"X-Content-Type-Options": "nosniff",
		"Cache-Control":          "private, max-age=600",
	})
}

// ===== Stock =====

func (ic *InventoryController) ListStock(c *gin.Context) {
	_, scope, ok := ic.scope(c)
	if !ok {
		return
	}
	branchID, ok := queryID(c, "branch_id")
	if !ok {
		return
	}
	categoryID, ok := queryID(c, "category_id")
	if !ok {
		return
	}
	page, size := pageQuery(c)
	rows, total, err := ic.svc.ListStock(c.Request.Context(), services.StockFilter{
		BranchID: branchID, CategoryID: categoryID, Search: c.Query("search"), LowOnly: c.Query("low") == "true", Page: page, PageSize: size,
	}, scope)
	if err != nil {
		writeInventoryError(c, err)
		return
	}
	listJSON(c, orEmpty(rows), total)
}

func (ic *InventoryController) ListMovements(c *gin.Context) {
	_, scope, ok := ic.scope(c)
	if !ok {
		return
	}
	branchID, ok := queryID(c, "branch_id")
	if !ok {
		return
	}
	productID, ok := queryID(c, "product_id")
	if !ok {
		return
	}
	from, to, err := ic.svc.DateRange(c.Query("date_from"), c.Query("date_to"))
	if err != nil {
		writeInventoryError(c, err)
		return
	}
	page, size := pageQuery(c)
	rows, total, err := ic.svc.ListMovements(c.Request.Context(), services.MovementFilter{
		BranchID: branchID, ProductID: productID, Type: c.Query("type"), From: from, To: to, Page: page, PageSize: size,
	}, scope)
	if err != nil {
		writeInventoryError(c, err)
		return
	}
	listJSON(c, orEmpty(rows), total)
}

func (ic *InventoryController) Adjust(c *gin.Context) {
	actor, scope, ok := ic.scope(c)
	if !ok {
		return
	}
	var in services.AdjustInput
	if !bindJSON(c, &in) {
		return
	}
	level, err := ic.svc.Adjust(c.Request.Context(), in, scope, actor.ActorID)
	if err != nil {
		writeInventoryError(c, err)
		return
	}
	ic.record(c, actor.ActorID, "inventory.adjust", in.ProductID, &level.BranchID, map[string]interface{}{"mode": in.Mode, "quantity": in.Quantity, "reason": in.Reason, "balance": level.Quantity.String()})
	c.JSON(http.StatusOK, level)
}

// ===== Purchases =====

type purchaseOut struct {
	*services.PurchaseView
	CanApprove bool `json:"can_approve"`
	CanVoid    bool `json:"can_void"`
}

func purchaseFlags(v *services.PurchaseView, scope *accessScope) purchaseOut {
	return purchaseOut{
		PurchaseView: v,
		CanApprove:   v.Status == models.DocPending && scope.Can(permissions.PurchasesApprove) && v.CreatedByID != scope.ActorID,
		CanVoid:      (v.Status == models.DocPending || v.Status == models.DocApproved) && scope.Can(permissions.PurchasesVoid),
	}
}

func (ic *InventoryController) ListPurchases(c *gin.Context) {
	actor, scope, ok := ic.scope(c)
	if !ok {
		return
	}
	f, ok := ic.docFilter(c)
	if !ok {
		return
	}
	list, total, err := ic.svc.ListPurchases(c.Request.Context(), f, scope)
	if err != nil {
		writeInventoryError(c, err)
		return
	}
	out := make([]purchaseOut, len(list))
	for i := range list {
		out[i] = purchaseFlags(&list[i], actor)
	}
	listJSON(c, out, total)
}

func (ic *InventoryController) GetPurchase(c *gin.Context) {
	actor, scope, ok := ic.scope(c)
	if !ok {
		return
	}
	id, ok := paramID(c)
	if !ok {
		return
	}
	v, err := ic.svc.GetPurchase(c.Request.Context(), id, scope)
	if err != nil {
		writeInventoryError(c, err)
		return
	}
	c.JSON(http.StatusOK, purchaseFlags(v, actor))
}

func (ic *InventoryController) CreatePurchase(c *gin.Context) {
	actor, scope, ok := ic.scope(c)
	if !ok {
		return
	}
	var in services.PurchaseInput
	if !bindJSON(c, &in) {
		return
	}
	v, err := ic.svc.CreatePurchase(c.Request.Context(), in, scope, actor.ActorID)
	if err != nil {
		writeInventoryError(c, err)
		return
	}
	ic.record(c, actor.ActorID, "inventory.purchase.create", v.ID.String(), &v.BranchID, map[string]interface{}{"number": v.Number, "total": v.Total.String(), "currency": v.Currency, "status": v.Status})
	c.JSON(http.StatusCreated, purchaseFlags(v, actor))
}

type decisionBody struct {
	Note string `json:"note" binding:"max=512"`
}

func (ic *InventoryController) decidePurchase(c *gin.Context, approve bool) {
	actor, scope, ok := ic.scope(c)
	if !ok {
		return
	}
	id, ok := paramID(c)
	if !ok {
		return
	}
	var body decisionBody
	if c.Request.ContentLength > 0 && !bindJSON(c, &body) {
		return
	}
	v, err := ic.svc.DecidePurchase(c.Request.Context(), id, approve, body.Note, scope, actor.ActorID)
	if err != nil {
		writeInventoryError(c, err)
		return
	}
	action := "inventory.purchase.reject"
	if approve {
		action = "inventory.purchase.approve"
	}
	ic.record(c, actor.ActorID, action, v.ID.String(), &v.BranchID, map[string]interface{}{"number": v.Number, "note": body.Note})
	c.JSON(http.StatusOK, purchaseFlags(v, actor))
}

func (ic *InventoryController) ApprovePurchase(c *gin.Context) { ic.decidePurchase(c, true) }
func (ic *InventoryController) RejectPurchase(c *gin.Context)  { ic.decidePurchase(c, false) }

type voidBody struct {
	Reason string `json:"reason" binding:"required,min=3,max=512"`
}

func (ic *InventoryController) VoidPurchase(c *gin.Context) {
	actor, scope, ok := ic.scope(c)
	if !ok {
		return
	}
	id, ok := paramID(c)
	if !ok {
		return
	}
	var body voidBody
	if !bindJSON(c, &body) {
		return
	}
	v, err := ic.svc.VoidPurchase(c.Request.Context(), id, body.Reason, scope, actor.ActorID)
	if err != nil {
		writeInventoryError(c, err)
		return
	}
	ic.record(c, actor.ActorID, "inventory.purchase.void", v.ID.String(), &v.BranchID, map[string]interface{}{"number": v.Number, "reason": body.Reason})
	c.JSON(http.StatusOK, purchaseFlags(v, actor))
}

// ===== Sales =====

type saleOut struct {
	*services.SaleView
	CanVoid bool `json:"can_void"`
}

func saleFlags(v *services.SaleView, scope *accessScope) saleOut {
	return saleOut{SaleView: v, CanVoid: v.Status == models.DocCompleted && scope.Can(permissions.SalesVoid)}
}

func (ic *InventoryController) ListSales(c *gin.Context) {
	actor, scope, ok := ic.scope(c)
	if !ok {
		return
	}
	f, ok := ic.docFilter(c)
	if !ok {
		return
	}
	list, total, err := ic.svc.ListSales(c.Request.Context(), f, scope)
	if err != nil {
		writeInventoryError(c, err)
		return
	}
	out := make([]saleOut, len(list))
	for i := range list {
		out[i] = saleFlags(&list[i], actor)
	}
	listJSON(c, out, total)
}

func (ic *InventoryController) GetSale(c *gin.Context) {
	actor, scope, ok := ic.scope(c)
	if !ok {
		return
	}
	id, ok := paramID(c)
	if !ok {
		return
	}
	v, err := ic.svc.GetSale(c.Request.Context(), id, scope)
	if err != nil {
		writeInventoryError(c, err)
		return
	}
	c.JSON(http.StatusOK, saleFlags(v, actor))
}

func (ic *InventoryController) CreateSale(c *gin.Context) {
	actor, scope, ok := ic.scope(c)
	if !ok {
		return
	}
	var in services.SaleInput
	if !bindJSON(c, &in) {
		return
	}
	v, err := ic.svc.CreateSale(c.Request.Context(), in, scope, actor.ActorID)
	if err != nil {
		writeInventoryError(c, err)
		return
	}
	ic.record(c, actor.ActorID, "inventory.sale.create", v.ID.String(), &v.BranchID, map[string]interface{}{"number": v.Number, "total": v.Total.String(), "currency": v.Currency})
	c.JSON(http.StatusCreated, saleFlags(v, actor))
}

func (ic *InventoryController) VoidSale(c *gin.Context) {
	actor, scope, ok := ic.scope(c)
	if !ok {
		return
	}
	id, ok := paramID(c)
	if !ok {
		return
	}
	var body voidBody
	if !bindJSON(c, &body) {
		return
	}
	v, err := ic.svc.VoidSale(c.Request.Context(), id, body.Reason, scope, actor.ActorID)
	if err != nil {
		writeInventoryError(c, err)
		return
	}
	ic.record(c, actor.ActorID, "inventory.sale.void", v.ID.String(), &v.BranchID, map[string]interface{}{"number": v.Number, "reason": body.Reason})
	c.JSON(http.StatusOK, saleFlags(v, actor))
}

// ===== Transfers =====

type transferOut struct {
	*services.TransferView
	CanVoid bool `json:"can_void"`
}

func transferFlags(v *services.TransferView, scope *accessScope) transferOut {
	return transferOut{TransferView: v, CanVoid: v.Status == models.DocCompleted && scope.Can(permissions.InventoryTransfer) &&
		scope.HasBranch(v.FromBranchID) && scope.HasBranch(v.ToBranchID)}
}

func (ic *InventoryController) ListTransfers(c *gin.Context) {
	actor, scope, ok := ic.scope(c)
	if !ok {
		return
	}
	f, ok := ic.docFilter(c)
	if !ok {
		return
	}
	list, total, err := ic.svc.ListTransfers(c.Request.Context(), f, scope)
	if err != nil {
		writeInventoryError(c, err)
		return
	}
	out := make([]transferOut, len(list))
	for i := range list {
		out[i] = transferFlags(&list[i], actor)
	}
	listJSON(c, out, total)
}

func (ic *InventoryController) CreateTransfer(c *gin.Context) {
	actor, scope, ok := ic.scope(c)
	if !ok {
		return
	}
	var in services.TransferInput
	if !bindJSON(c, &in) {
		return
	}
	v, err := ic.svc.CreateTransfer(c.Request.Context(), in, scope, actor.ActorID)
	if err != nil {
		writeInventoryError(c, err)
		return
	}
	ic.record(c, actor.ActorID, "inventory.transfer.create", v.ID.String(), &v.FromBranchID, map[string]interface{}{"number": v.Number, "to": v.ToCode})
	c.JSON(http.StatusCreated, transferFlags(v, actor))
}

func (ic *InventoryController) VoidTransfer(c *gin.Context) {
	actor, scope, ok := ic.scope(c)
	if !ok {
		return
	}
	id, ok := paramID(c)
	if !ok {
		return
	}
	var body voidBody
	if !bindJSON(c, &body) {
		return
	}
	v, err := ic.svc.VoidTransfer(c.Request.Context(), id, body.Reason, scope, actor.ActorID)
	if err != nil {
		writeInventoryError(c, err)
		return
	}
	ic.record(c, actor.ActorID, "inventory.transfer.void", v.ID.String(), &v.FromBranchID, map[string]interface{}{"number": v.Number, "reason": body.Reason})
	c.JSON(http.StatusOK, transferFlags(v, actor))
}

// ===== Inventory expenses =====

type expenseOut struct {
	*services.InventoryExpenseView
	CanVoid bool `json:"can_void"`
}

func expenseFlags(v *services.InventoryExpenseView, scope *accessScope) expenseOut {
	return expenseOut{InventoryExpenseView: v, CanVoid: v.Status == models.DocCompleted && scope.Can(permissions.InventoryExpensesVoid)}
}

func (ic *InventoryController) ListExpenses(c *gin.Context) {
	actor, scope, ok := ic.scope(c)
	if !ok {
		return
	}
	f, ok := ic.docFilter(c)
	if !ok {
		return
	}
	categoryID, ok := queryID(c, "category_id")
	if !ok {
		return
	}
	list, total, err := ic.svc.ListExpenses(c.Request.Context(), services.ExpenseFilter{DocFilter: f, CategoryID: categoryID}, scope)
	if err != nil {
		writeInventoryError(c, err)
		return
	}
	out := make([]expenseOut, len(list))
	for i := range list {
		out[i] = expenseFlags(&list[i], actor)
	}
	listJSON(c, out, total)
}

func (ic *InventoryController) CreateExpense(c *gin.Context) {
	actor, scope, ok := ic.scope(c)
	if !ok {
		return
	}
	var in services.InventoryExpenseInput
	if !bindJSON(c, &in) {
		return
	}
	v, err := ic.svc.CreateExpense(c.Request.Context(), in, scope, actor.ActorID)
	if err != nil {
		writeInventoryError(c, err)
		return
	}
	ic.record(c, actor.ActorID, "inventory.expense.create", v.ID.String(), &v.BranchID, map[string]interface{}{"number": v.Number, "amount": v.Amount.String(), "currency": v.Currency, "category": v.CategoryName})
	c.JSON(http.StatusCreated, expenseFlags(v, actor))
}

func (ic *InventoryController) VoidExpense(c *gin.Context) {
	actor, scope, ok := ic.scope(c)
	if !ok {
		return
	}
	id, ok := paramID(c)
	if !ok {
		return
	}
	var body voidBody
	if !bindJSON(c, &body) {
		return
	}
	v, err := ic.svc.VoidExpense(c.Request.Context(), id, body.Reason, scope, actor.ActorID)
	if err != nil {
		writeInventoryError(c, err)
		return
	}
	ic.record(c, actor.ActorID, "inventory.expense.void", v.ID.String(), &v.BranchID, map[string]interface{}{"number": v.Number, "reason": body.Reason})
	c.JSON(http.StatusOK, expenseFlags(v, actor))
}

// ===== Reports =====

func (ic *InventoryController) reportFilter(c *gin.Context) (services.InventoryReportFilter, services.BranchScope, bool) {
	_, scope, ok := ic.scope(c)
	if !ok {
		return services.InventoryReportFilter{}, scope, false
	}
	branchID, ok := queryID(c, "branch_id")
	if !ok {
		return services.InventoryReportFilter{}, scope, false
	}
	categoryID, ok := queryID(c, "category_id")
	if !ok {
		return services.InventoryReportFilter{}, scope, false
	}
	from, to, err := ic.svc.DateRange(c.Query("date_from"), c.Query("date_to"))
	if err != nil {
		writeInventoryError(c, err)
		return services.InventoryReportFilter{}, scope, false
	}
	return services.InventoryReportFilter{BranchID: branchID, CategoryID: categoryID, From: from, To: to}, scope, true
}

func (ic *InventoryController) ReportSummary(c *gin.Context) {
	f, scope, ok := ic.reportFilter(c)
	if !ok {
		return
	}
	out, err := ic.svc.Summary(c.Request.Context(), f, scope)
	if err != nil {
		writeInventoryError(c, err)
		return
	}
	c.JSON(http.StatusOK, out)
}

func (ic *InventoryController) ReportStock(c *gin.Context) {
	f, scope, ok := ic.reportFilter(c)
	if !ok {
		return
	}
	rows, err := ic.svc.StockReport(c.Request.Context(), f, scope)
	if err != nil {
		writeInventoryError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": orEmpty(rows)})
}

func (ic *InventoryController) ReportSales(c *gin.Context) {
	f, scope, ok := ic.reportFilter(c)
	if !ok {
		return
	}
	rows, err := ic.svc.SalesReport(c.Request.Context(), f, scope)
	if err != nil {
		writeInventoryError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": orEmpty(rows)})
}

func (ic *InventoryController) ReportPurchases(c *gin.Context) {
	f, scope, ok := ic.reportFilter(c)
	if !ok {
		return
	}
	rows, err := ic.svc.PurchasesReport(c.Request.Context(), f, c.Query("group") == "supplier", scope)
	if err != nil {
		writeInventoryError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": orEmpty(rows)})
}

func (ic *InventoryController) ReportExpenses(c *gin.Context) {
	f, scope, ok := ic.reportFilter(c)
	if !ok {
		return
	}
	rows, err := ic.svc.ExpensesReport(c.Request.Context(), f, scope)
	if err != nil {
		writeInventoryError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": orEmpty(rows)})
}

func (ic *InventoryController) ReportLowStock(c *gin.Context) {
	f, scope, ok := ic.reportFilter(c)
	if !ok {
		return
	}
	rows, err := ic.svc.LowStock(c.Request.Context(), f, scope)
	if err != nil {
		writeInventoryError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": orEmpty(rows)})
}
