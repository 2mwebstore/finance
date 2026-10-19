package controllers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/cambodia-finance/income-expense/internal/accounting"
	"github.com/cambodia-finance/income-expense/internal/audit"
	"github.com/cambodia-finance/income-expense/internal/models"
	"github.com/cambodia-finance/income-expense/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type TransactionController struct {
	svc   *services.TransactionService
	audit *audit.Service
}

func NewTransactionController(svc *services.TransactionService, auditSvc *audit.Service) *TransactionController {
	return &TransactionController{svc: svc, audit: auditSvc}
}

func (tc *TransactionController) Create(c *gin.Context) {
	var in services.CreateTransactionInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	actorID, ok := getActorID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	txn, err := tc.svc.Create(c.Request.Context(), in, actorID)
	if err != nil {
		writeTxnError(c, err)
		return
	}
	if tc.audit != nil {
		bid := txn.BranchID
		tc.audit.Record(c.Request.Context(), audit.Event{
			ActorID: &actorID, Action: audit.ActionTxnCreate, EntityType: "transaction",
			EntityID: txn.ID.String(), BranchID: &bid, Outcome: "success",
			IPAddress: c.ClientIP(), UserAgent: c.Request.UserAgent(),
			Details: map[string]interface{}{
				"number": txn.TransactionNumber, "type": txn.Type, "currency": txn.Currency,
				"amount": txn.Amount.String(), "status": txn.Status,
			},
		})
	}
	c.JSON(http.StatusCreated, txnJSON(txn))
}

func (tc *TransactionController) Get(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	actorID, ok := getActorID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	txn, err := tc.svc.GetByID(c.Request.Context(), id, actorID)
	if err != nil {
		writeTxnError(c, err)
		return
	}
	c.JSON(http.StatusOK, txnJSON(txn))
}

func (tc *TransactionController) List(c *gin.Context) {
	actorID, ok := getActorID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	f := services.TransactionListFilter{
		Type:       c.Query("type"),
		BranchID:   c.Query("branch_id"),
		Currency:   c.Query("currency"),
		CategoryID: c.Query("category_id"),
		Status:     c.Query("status"),
		Search:     c.Query("search"),
	}
	f.Page, _ = strconv.Atoi(c.DefaultQuery("page", "1"))
	f.PageSize, _ = strconv.Atoi(c.DefaultQuery("page_size", "10"))
	if f.Page < 1 {
		f.Page = 1
	}
	if f.PageSize < 1 {
		f.PageSize = 10
	} else if f.PageSize > 100 {
		f.PageSize = 100
	}
	if v := c.Query("date_from"); v != "" {
		if t, err := time.Parse("2006-01-02", v); err == nil {
			f.DateFrom = &t
		}
	}
	if v := c.Query("date_to"); v != "" {
		if t, err := time.Parse("2006-01-02", v); err == nil {
			// half-open: include entire end day
			end := t.Add(24 * time.Hour)
			f.DateTo = &end
		}
	}

	items, total, err := tc.svc.List(c.Request.Context(), f, actorID)
	if err != nil {
		writeTxnError(c, err)
		return
	}
	out := make([]gin.H, 0, len(items))
	for i := range items {
		out = append(out, txnJSON(&items[i]))
	}
	c.JSON(http.StatusOK, gin.H{
		"items": out, "total": total, "page": f.Page, "page_size": f.PageSize,
	})
}

func (tc *TransactionController) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var in services.UpdateTransactionInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	actorID, ok := getActorID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	txn, err := tc.svc.Update(c.Request.Context(), id, in, actorID)
	if err != nil {
		writeTxnError(c, err)
		return
	}
	c.JSON(http.StatusOK, txnJSON(txn))
}

func (tc *TransactionController) Void(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var body struct {
		Reason string `json:"reason"`
	}
	_ = c.ShouldBindJSON(&body)
	actorID, ok := getActorID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	txn, err := tc.svc.Void(c.Request.Context(), id, body.Reason, actorID)
	if err != nil {
		writeTxnError(c, err)
		return
	}
	if tc.audit != nil {
		bid := txn.BranchID
		tc.audit.Record(c.Request.Context(), audit.Event{
			ActorID: &actorID, Action: audit.ActionTxnVoid, EntityType: "transaction",
			EntityID: txn.ID.String(), BranchID: &bid, Outcome: "success",
			IPAddress: c.ClientIP(), Details: map[string]interface{}{"reason": body.Reason},
		})
	}
	c.JSON(http.StatusOK, txnJSON(txn))
}

func (tc *TransactionController) Totals(c *gin.Context) {
	actorID, ok := getActorID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	var branchID *uuid.UUID
	if v := c.Query("branch_id"); v != "" {
		id, err := uuid.Parse(v)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid branch_id"})
			return
		}
		branchID = &id
	}
	var from, to *time.Time
	if v := c.Query("date_from"); v != "" {
		if t, err := time.Parse("2006-01-02", v); err == nil {
			from = &t
		}
	}
	if v := c.Query("date_to"); v != "" {
		if t, err := time.Parse("2006-01-02", v); err == nil {
			end := t.Add(24 * time.Hour)
			to = &end
		}
	}
	totals, err := tc.svc.TotalsByCurrency(c.Request.Context(), actorID, branchID, from, to, c.Query("type"), c.Query("currency"), c.Query("search"))
	if err != nil {
		writeTxnError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"income_usd":  decStr(totals["income_usd"]),
		"expense_usd": decStr(totals["expense_usd"]),
		"net_usd":     decStr(totals["net_usd"]),
		"income_khr":  decStr(totals["income_khr"]),
		"expense_khr": decStr(totals["expense_khr"]),
		"net_khr":     decStr(totals["net_khr"]),
	})
}

func decStr(d decimal.Decimal) string {
	return d.StringFixed(4)
}

func txnJSON(t *models.Transaction) gin.H {
	h := gin.H{
		"id":                 t.ID,
		"transaction_number": t.TransactionNumber,
		"type":               t.Type,
		"status":             t.Status,
		"branch_id":          t.BranchID,
		"category_id":        t.CategoryID,
		"payment_method_id":  t.PaymentMethodID,
		"currency":           t.Currency,
		"amount":             t.Amount.StringFixed(4),
		"amount_display":     accounting.FormatDisplay(t.Currency, t.Amount),
		"transaction_date":   t.TransactionDate,
		"description":        t.Description,
		"counterparty":       t.Counterparty,
		"created_by_id":      t.CreatedByID,
		"approved_by_id":     t.ApprovedByID,
		"approved_at":        t.ApprovedAt,
		"voided_by_id":       t.VoidedByID,
		"voided_at":          t.VoidedAt,
		"void_reason":        t.VoidReason,
		"created_at":         t.CreatedAt,
		"updated_at":         t.UpdatedAt,
	}
	if t.Branch != nil {
		h["branch"] = gin.H{"id": t.Branch.ID, "code": t.Branch.Code, "name": t.Branch.Name}
	}
	if t.Category != nil {
		h["category"] = gin.H{"id": t.Category.ID, "code": t.Category.Code, "name": t.Category.Name, "type": t.Category.Type}
	}
	if t.PaymentMethod != nil {
		h["payment_method"] = gin.H{"id": t.PaymentMethod.ID, "code": t.PaymentMethod.Code, "name": t.PaymentMethod.Name}
	}
	return h
}

func writeTxnError(c *gin.Context, err error) {
	msg := err.Error()
	switch msg {
	case "transaction not found":
		c.JSON(http.StatusNotFound, gin.H{"error": msg})
	case "no access to this branch", "forbidden", "cannot approve your own transaction", "cannot reject your own transaction":
		c.JSON(http.StatusForbidden, gin.H{"error": msg})
	case "invalid transaction status for this operation", "opening balance already exists for this branch, currency, and date",
		"category type does not match transaction type",
		"currency must be USD or KHR",
		"amount must be a positive number with at most 4 decimal places",
		"type must be income or expense":
		c.JSON(http.StatusBadRequest, gin.H{"error": msg})
	default:
		if len(msg) > 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": msg})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		}
	}
}

func (tc *TransactionController) Approve(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var body struct {
		Notes string `json:"notes"`
	}
	_ = c.ShouldBindJSON(&body)
	actorID, ok := getActorID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	txn, err := tc.svc.Approve(c.Request.Context(), id, body.Notes, actorID)
	if err != nil {
		writeTxnError(c, err)
		return
	}
	if tc.audit != nil {
		bid := txn.BranchID
		tc.audit.Record(c.Request.Context(), audit.Event{
			ActorID: &actorID, Action: audit.ActionTxnApprove, EntityType: "transaction",
			EntityID: txn.ID.String(), BranchID: &bid, Outcome: "success",
			IPAddress: c.ClientIP(), Details: map[string]interface{}{"notes": body.Notes},
		})
	}
	c.JSON(http.StatusOK, txnJSON(txn))
}

func (tc *TransactionController) Reject(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var body struct {
		Notes string `json:"notes"`
	}
	_ = c.ShouldBindJSON(&body)
	actorID, ok := getActorID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	txn, err := tc.svc.Reject(c.Request.Context(), id, body.Notes, actorID)
	if err != nil {
		writeTxnError(c, err)
		return
	}
	if tc.audit != nil {
		bid := txn.BranchID
		tc.audit.Record(c.Request.Context(), audit.Event{
			ActorID: &actorID, Action: audit.ActionTxnReject, EntityType: "transaction",
			EntityID: txn.ID.String(), BranchID: &bid, Outcome: "success",
			IPAddress: c.ClientIP(), Details: map[string]interface{}{"notes": body.Notes},
		})
	}
	c.JSON(http.StatusOK, txnJSON(txn))
}

func (tc *TransactionController) Reverse(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var body struct {
		Reason string `json:"reason"`
	}
	_ = c.ShouldBindJSON(&body)
	if body.Reason == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "reason is required for reversal"})
		return
	}
	actorID, ok := getActorID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	txn, err := tc.svc.Reverse(c.Request.Context(), id, body.Reason, actorID)
	if err != nil {
		writeTxnError(c, err)
		return
	}
	if tc.audit != nil {
		bid := txn.BranchID
		tc.audit.Record(c.Request.Context(), audit.Event{
			ActorID: &actorID, Action: "transaction.reverse", EntityType: "transaction",
			EntityID: txn.ID.String(), BranchID: &bid, Outcome: "success",
			IPAddress: c.ClientIP(), Details: map[string]interface{}{"reason": body.Reason, "reverses": id.String()},
		})
	}
	c.JSON(http.StatusCreated, txnJSON(txn))
}
