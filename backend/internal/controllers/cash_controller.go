package controllers

import (
	"net/http"

	"github.com/cambodia-finance/income-expense/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type CashController struct {
	svc *services.CashService
}

func NewCashController(svc *services.CashService) *CashController {
	return &CashController{svc: svc}
}

func (cc *CashController) CreateOpeningBalance(c *gin.Context) {
	var in services.CreateOpeningBalanceInput
	if !bindJSON(c, &in) {
		return
	}
	actorID, ok := getActorID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	ob, err := cc.svc.CreateOpeningBalance(c.Request.Context(), in, actorID)
	if err != nil {
		writeTxnError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"id": ob.ID, "branch_id": ob.BranchID, "currency": ob.Currency,
		"amount": ob.Amount.StringFixed(4), "balance_date": ob.BalanceDate.Format("2006-01-02"),
		"notes": ob.Notes, "status": ob.Status,
	})
}

func (cc *CashController) ListOpeningBalances(c *gin.Context) {
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
	items, err := cc.svc.ListOpeningBalances(c.Request.Context(), actorID, branchID)
	if err != nil {
		writeTxnError(c, err)
		return
	}
	out := make([]gin.H, 0, len(items))
	for _, ob := range items {
		out = append(out, gin.H{
			"id": ob.ID, "branch_id": ob.BranchID, "currency": ob.Currency,
			"amount": ob.Amount.StringFixed(4), "balance_date": ob.BalanceDate.Format("2006-01-02"),
			"notes": ob.Notes, "status": ob.Status,
		})
	}
	c.JSON(http.StatusOK, gin.H{"items": out})
}

func (cc *CashController) CreateReconciliation(c *gin.Context) {
	var in services.CreateReconciliationInput
	if !bindJSON(c, &in) {
		return
	}
	actorID, ok := getActorID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	cr, err := cc.svc.CreateReconciliation(c.Request.Context(), in, actorID)
	if err != nil {
		writeTxnError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"id": cr.ID, "branch_id": cr.BranchID, "currency": cr.Currency,
		"reconcile_date": cr.ReconcileDate.Format("2006-01-02"),
		"expected_amount": cr.ExpectedAmount.StringFixed(4),
		"actual_amount":   cr.ActualAmount.StringFixed(4),
		"difference":      cr.Difference.StringFixed(4),
		"notes": cr.Notes, "status": cr.Status,
	})
}

func (cc *CashController) ListReconciliations(c *gin.Context) {
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
	items, err := cc.svc.ListReconciliations(c.Request.Context(), actorID, branchID)
	if err != nil {
		writeTxnError(c, err)
		return
	}
	out := make([]gin.H, 0, len(items))
	for _, cr := range items {
		out = append(out, gin.H{
			"id": cr.ID, "branch_id": cr.BranchID, "currency": cr.Currency,
			"reconcile_date": cr.ReconcileDate.Format("2006-01-02"),
			"expected_amount": cr.ExpectedAmount.StringFixed(4),
			"actual_amount":   cr.ActualAmount.StringFixed(4),
			"difference":      cr.Difference.StringFixed(4),
			"notes": cr.Notes, "status": cr.Status,
		})
	}
	c.JSON(http.StatusOK, gin.H{"items": out})
}
