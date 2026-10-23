package controllers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/cambodia-finance/income-expense/internal/accounting"
	"github.com/cambodia-finance/income-expense/internal/audit"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type AuditController struct {
	svc *audit.Service
	db  *gorm.DB
	loc *time.Location
}

func NewAuditController(svc *audit.Service, db *gorm.DB, timezone string) *AuditController {
	return &AuditController{svc: svc, db: db, loc: accounting.LoadLocation(timezone)}
}

func (ac *AuditController) List(c *gin.Context) {
	scope, ok := loadScope(c, ac.db)
	if !ok {
		return
	}
	f := audit.ListFilter{
		Restricted: !scope.Full,
		BranchIDs:  scope.BranchIDs,
		Action:     c.Query("action"),
		EntityType: c.Query("entity_type"),
		ActorID:    c.Query("actor_id"),
		BranchID:   c.Query("branch_id"),
		Outcome:    c.Query("outcome"),
		Search:     c.Query("search"),
	}
	f.Page, _ = strconv.Atoi(c.DefaultQuery("page", "1"))
	f.PageSize, _ = strconv.Atoi(c.DefaultQuery("page_size", "50"))
	f.DateFrom, f.DateTo = dayRange(c, ac.loc)

	items, total, err := ac.svc.List(c.Request.Context(), f)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "list failed"})
		return
	}
	out := make([]gin.H, 0, len(items))
	for _, a := range items {
		out = append(out, gin.H{
			"id":             a.ID,
			"created_at":     a.CreatedAt,
			"actor_id":       a.ActorID,
			"actor_email":    a.ActorEmail,
			"action":         a.Action,
			"entity_type":    a.EntityType,
			"entity_id":      a.EntityID,
			"branch_id":      a.BranchID,
			"ip_address":     a.IPAddress,
			"outcome":        a.Outcome,
			"details":        a.Details,
			"correlation_id": a.Correlation,
		})
	}
	c.JSON(http.StatusOK, gin.H{
		"items": out, "total": total, "page": f.Page, "page_size": f.PageSize,
	})
}

// Cleanup deletes audit events older than the last keep_months (1, 2 or 3) months.
// With dry_run it only reports how many events would be deleted.
func (ac *AuditController) Cleanup(c *gin.Context) {
	var body struct {
		KeepMonths int  `json:"keep_months"`
		DryRun     bool `json:"dry_run"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.KeepMonths < 1 || body.KeepMonths > 3 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "keep_months must be 1, 2 or 3"})
		return
	}
	ctx := c.Request.Context()
	cutoff := time.Now().AddDate(0, -body.KeepMonths, 0)

	if body.DryRun {
		count, err := ac.svc.CountBefore(ctx, cutoff)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "count failed"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"would_delete": count, "cutoff": cutoff, "keep_months": body.KeepMonths})
		return
	}

	deleted, err := ac.svc.DeleteBefore(ctx, cutoff)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "cleanup failed"})
		return
	}
	// Recorded after the delete so the cleanup itself stays in the log.
	ev := audit.Event{
		Action: audit.ActionAuditCleanup, EntityType: "audit_log", Outcome: "success",
		IPAddress: c.ClientIP(), UserAgent: c.Request.UserAgent(),
		Details: map[string]interface{}{"keep_months": body.KeepMonths, "deleted": deleted, "cutoff": cutoff},
	}
	if actorID, ok := getActorID(c); ok {
		ev.ActorID = &actorID
	}
	ac.svc.Record(ctx, ev)
	c.JSON(http.StatusOK, gin.H{"deleted": deleted, "cutoff": cutoff, "keep_months": body.KeepMonths})
}
