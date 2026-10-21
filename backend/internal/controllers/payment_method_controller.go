package controllers

import (
	"net/http"

	"github.com/cambodia-finance/income-expense/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type PaymentMethodController struct {
	svc *services.PaymentMethodService
}

func NewPaymentMethodController(svc *services.PaymentMethodService) *PaymentMethodController {
	return &PaymentMethodController{svc: svc}
}

func (pc *PaymentMethodController) List(c *gin.Context) {
	activeOnly := c.Query("active") == "true" || c.Query("active") == "1"
	items, err := pc.svc.List(c.Request.Context(), activeOnly)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "list failed"})
		return
	}
	out := make([]gin.H, 0, len(items))
	for _, pm := range items {
		out = append(out, gin.H{
			"id": pm.ID, "code": pm.Code, "name": pm.Name,
			"is_active": pm.IsActive, "is_system": pm.IsSystem,
		})
	}
	c.JSON(http.StatusOK, gin.H{"items": out})
}

func (pc *PaymentMethodController) Create(c *gin.Context) {
	var in services.CreatePaymentMethodInput
	if !bindJSON(c, &in) {
		return
	}
	pm, err := pc.svc.Create(c.Request.Context(), in)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"id": pm.ID, "code": pm.Code, "name": pm.Name, "is_active": pm.IsActive,
	})
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
	pm, err := pc.svc.Update(c.Request.Context(), id, in)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"id": pm.ID, "code": pm.Code, "name": pm.Name, "is_active": pm.IsActive,
	})
}
