package controllers

import (
	"net/http"

	"github.com/cambodia-finance/income-expense/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// PaymentMethodController: payment methods are shared by every branch (the owner or an admin
// manages those) or one branch's own (that branch sees and uses it).
type PaymentMethodController struct {
	svc *services.PaymentMethodService
	db  *gorm.DB
}

func NewPaymentMethodController(svc *services.PaymentMethodService, db *gorm.DB) *PaymentMethodController {
	return &PaymentMethodController{svc: svc, db: db}
}

func (pc *PaymentMethodController) catalog(c *gin.Context) (*accessScope, services.CatalogScope, bool) {
	scope, ok := loadScope(c, pc.db)
	if !ok {
		return nil, services.CatalogScope{}, false
	}
	return scope, services.CatalogScope{Branches: services.BranchScope{All: scope.AllBranches, IDs: scope.BranchIDs}, Full: scope.Full}, true
}

func paymentMethodJSON(pm *services.PaymentMethodView) gin.H {
	return gin.H{
		"id": pm.ID, "code": pm.Code, "name": pm.Name, "is_active": pm.IsActive, "is_system": pm.IsSystem,
		"branch_id": pm.BranchID, "branch_name": pm.BranchName,
	}
}

func (pc *PaymentMethodController) List(c *gin.Context) {
	scope, _, ok := pc.catalog(c)
	if !ok {
		return
	}
	activeOnly := c.Query("active") == "true" || c.Query("active") == "1"
	listScope := services.CategoryListScope{All: scope.AllBranches, BranchIDs: scope.BranchIDs}
	if v := c.Query("branch_id"); v != "" {
		branchID, err := uuid.Parse(v)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid branch_id"})
			return
		}
		if !scope.HasBranch(branchID) {
			c.JSON(http.StatusForbidden, gin.H{"error": "no access to this branch"})
			return
		}
		listScope.ForBranch = &branchID
	}
	items, err := pc.svc.List(c.Request.Context(), activeOnly, listScope)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "list failed"})
		return
	}
	out := make([]gin.H, 0, len(items))
	for i := range items {
		out = append(out, paymentMethodJSON(&items[i]))
	}
	c.JSON(http.StatusOK, gin.H{"items": out})
}

func (pc *PaymentMethodController) Create(c *gin.Context) {
	var in services.CreatePaymentMethodInput
	if !bindJSON(c, &in) {
		return
	}
	_, scope, ok := pc.catalog(c)
	if !ok {
		return
	}
	pm, err := pc.svc.Create(c.Request.Context(), in, scope)
	if err != nil {
		writeInventoryError(c, err)
		return
	}
	c.JSON(http.StatusCreated, paymentMethodJSON(pm))
}

func (pc *PaymentMethodController) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var in services.UpdatePaymentMethodInput
	if !bindJSON(c, &in) {
		return
	}
	_, scope, ok := pc.catalog(c)
	if !ok {
		return
	}
	pm, err := pc.svc.Update(c.Request.Context(), id, in, scope)
	if err != nil {
		writeInventoryError(c, err)
		return
	}
	c.JSON(http.StatusOK, paymentMethodJSON(pm))
}
