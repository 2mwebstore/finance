package controllers

import (
	"net/http"
	"strconv"

	"github.com/cambodia-finance/income-expense/internal/audit"
	"github.com/cambodia-finance/income-expense/internal/models"
	"github.com/cambodia-finance/income-expense/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// RequestController serves the Requests page: expenses waiting for approval. A request is a
// pending expense, so its files go through the transaction attachment routes.
type RequestController struct {
	svc      *services.TransactionService
	audit    *audit.Service
	telegram *services.BranchTelegramService // the branch's Telegram chat (optional)
}

func NewRequestController(svc *services.TransactionService, auditSvc *audit.Service, telegram *services.BranchTelegramService) *RequestController {
	return &RequestController{svc: svc, audit: auditSvc, telegram: telegram}
}

func (rc *RequestController) List(c *gin.Context) {
	actorID, ok := getActorID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	f := services.RequestListFilter{
		Status:     c.Query("status"),
		BranchID:   c.Query("branch_id"),
		Currency:   c.Query("currency"),
		CategoryID: c.Query("category_id"),
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
	f.DateFrom, f.DateTo = dayRange(c, rc.svc.Location())

	items, total, counts, err := rc.svc.ListRequests(c.Request.Context(), f, actorID)
	if err != nil {
		writeTxnError(c, err)
		return
	}
	decisions, err := rc.svc.Decisions(c.Request.Context(), actorID, items)
	if err != nil {
		writeTxnError(c, err)
		return
	}
	people := rc.svc.People(c.Request.Context(), items)
	out := make([]gin.H, 0, len(items))
	for i := range items {
		item := txnJSON(&items[i])
		d := decisions[items[i].ID]
		item["can_approve"] = d.CanApprove
		item["can_reject"] = d.CanReject
		p := people[items[i].ID]
		item["created_by"] = p.CreatedBy
		item["outcome"] = p.Outcome
		out = append(out, item)
	}
	c.JSON(http.StatusOK, gin.H{
		"items": out, "total": total, "page": f.Page, "page_size": f.PageSize, "counts": counts,
	})
}

// PendingCount is the number on the sidebar's Requests item: the pending requests the user sees,
// in the top bar's branch when branch_id is set.
func (rc *RequestController) PendingCount(c *gin.Context) {
	actorID, ok := getActorID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	n, err := rc.svc.PendingRequestCount(c.Request.Context(), actorID, c.Query("branch_id"))
	if err != nil {
		writeTxnError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"pending": n})
}

func (rc *RequestController) Create(c *gin.Context) {
	var in services.CreateRequestInput
	if !bindJSON(c, &in) {
		return
	}
	actorID, ok := getActorID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	txn, err := rc.svc.CreateRequest(c.Request.Context(), in, actorID)
	if err != nil {
		writeTxnError(c, err)
		return
	}
	rc.record(c, audit.ActionRequestCreate, txn, actorID, map[string]interface{}{
		"number": txn.TransactionNumber, "currency": txn.Currency, "amount": txn.Amount.String(),
	})
	rc.telegram.NotifyTransaction(services.TelegramEventCreated, txn.ID, actorID, "")
	c.JSON(http.StatusCreated, txnJSON(txn))
}

// Approve turns the request into an approved expense (once: a decided request is refused).
func (rc *RequestController) Approve(c *gin.Context) {
	rc.decide(c, true)
}

// Reject keeps the request on the page as rejected; a reason is required.
func (rc *RequestController) Reject(c *gin.Context) {
	rc.decide(c, false)
}

func (rc *RequestController) decide(c *gin.Context, approve bool) {
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
	var txn *models.Transaction
	action, event := audit.ActionRequestApprove, services.TelegramEventApproved
	if approve {
		txn, err = rc.svc.ApproveRequest(c.Request.Context(), id, body.Notes, actorID)
	} else {
		action, event = audit.ActionRequestReject, services.TelegramEventRejected
		txn, err = rc.svc.RejectRequest(c.Request.Context(), id, body.Notes, actorID)
	}
	if err != nil {
		if err == services.ErrInvalidStatus {
			c.JSON(http.StatusConflict, gin.H{"error": "this request was already approved or rejected"})
			return
		}
		writeTxnError(c, err)
		return
	}
	rc.record(c, action, txn, actorID, map[string]interface{}{
		"number": txn.TransactionNumber, "currency": txn.Currency, "amount": txn.Amount.String(), "notes": body.Notes,
	})
	rc.telegram.NotifyTransaction(event, txn.ID, actorID, body.Notes)
	c.JSON(http.StatusOK, txnJSON(txn))
}

func (rc *RequestController) record(c *gin.Context, action string, txn *models.Transaction, actorID uuid.UUID, details map[string]interface{}) {
	if rc.audit == nil {
		return
	}
	bid := txn.BranchID
	rc.audit.Record(c.Request.Context(), audit.Event{
		ActorID: &actorID, Action: action, EntityType: "transaction",
		EntityID: txn.ID.String(), BranchID: &bid, Outcome: "success",
		IPAddress: c.ClientIP(), UserAgent: c.Request.UserAgent(), Details: details,
	})
}
