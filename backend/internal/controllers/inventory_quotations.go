package controllers

import (
	"net/http"

	"github.com/cambodia-finance/income-expense/internal/models"
	"github.com/cambodia-finance/income-expense/internal/permissions"
	"github.com/cambodia-finance/income-expense/internal/services"
	"github.com/gin-gonic/gin"
)

// ===== Quotations =====

type quotationOut struct {
	*services.QuotationView
	CanEdit    bool `json:"can_edit"`    // change it, or mark it accepted, declined or open again
	CanConvert bool `json:"can_convert"` // turn it into a sale
	CanDelete  bool `json:"can_delete"`
}

func quotationFlags(v *services.QuotationView, scope *accessScope) quotationOut {
	live := v.Status != models.QuoteConverted
	return quotationOut{
		QuotationView: v,
		CanEdit:       live && scope.Can(permissions.QuotationsEdit),
		CanConvert:    (v.Status == models.QuoteOpen || v.Status == models.QuoteAccepted) && scope.Can(permissions.SalesCreate),
		CanDelete:     live && scope.Can(permissions.QuotationsDelete),
	}
}

func (ic *InventoryController) ListQuotations(c *gin.Context) {
	actor, scope, ok := ic.scope(c)
	if !ok {
		return
	}
	f, ok := ic.docFilter(c)
	if !ok {
		return
	}
	list, total, err := ic.svc.ListQuotations(c.Request.Context(), f, scope)
	if err != nil {
		writeInventoryError(c, err)
		return
	}
	out := make([]quotationOut, len(list))
	for i := range list {
		out[i] = quotationFlags(&list[i], actor)
	}
	listJSON(c, out, total)
}

func (ic *InventoryController) GetQuotation(c *gin.Context) {
	actor, scope, ok := ic.scope(c)
	if !ok {
		return
	}
	id, ok := paramID(c)
	if !ok {
		return
	}
	v, err := ic.svc.GetQuotation(c.Request.Context(), id, scope)
	if err != nil {
		writeInventoryError(c, err)
		return
	}
	c.JSON(http.StatusOK, quotationFlags(v, actor))
}

func (ic *InventoryController) CreateQuotation(c *gin.Context) {
	actor, scope, ok := ic.scope(c)
	if !ok {
		return
	}
	var in services.QuotationInput
	if !bindJSON(c, &in) {
		return
	}
	v, err := ic.svc.CreateQuotation(c.Request.Context(), in, scope, actor.ActorID)
	if err != nil {
		writeInventoryError(c, err)
		return
	}
	ic.record(c, actor.ActorID, "inventory.quotation.create", v.ID.String(), &v.BranchID, map[string]interface{}{"number": v.Number, "total": v.Total.String(), "currency": v.Currency})
	c.JSON(http.StatusCreated, quotationFlags(v, actor))
}

// UpdateQuotation changes a quotation that wasn't turned into a sale (it is open again after).
func (ic *InventoryController) UpdateQuotation(c *gin.Context) {
	actor, scope, ok := ic.scope(c)
	if !ok {
		return
	}
	id, ok := paramID(c)
	if !ok {
		return
	}
	var in services.QuotationInput
	if !bindJSON(c, &in) {
		return
	}
	v, err := ic.svc.UpdateQuotation(c.Request.Context(), id, in, scope, actor.ActorID)
	if err != nil {
		writeInventoryError(c, err)
		return
	}
	ic.record(c, actor.ActorID, "inventory.quotation.update", v.ID.String(), &v.BranchID, map[string]interface{}{"number": v.Number, "total": v.Total.String(), "currency": v.Currency})
	c.JSON(http.StatusOK, quotationFlags(v, actor))
}

type quotationStatusBody struct {
	Status string `json:"status" binding:"required,oneof=open accepted declined"`
}

// SetQuotationStatus marks a quotation accepted or declined by the customer, or open again.
func (ic *InventoryController) SetQuotationStatus(c *gin.Context) {
	actor, scope, ok := ic.scope(c)
	if !ok {
		return
	}
	id, ok := paramID(c)
	if !ok {
		return
	}
	var body quotationStatusBody
	if !bindJSON(c, &body) {
		return
	}
	v, err := ic.svc.SetQuotationStatus(c.Request.Context(), id, body.Status, scope, actor.ActorID)
	if err != nil {
		writeInventoryError(c, err)
		return
	}
	ic.record(c, actor.ActorID, "inventory.quotation.status", v.ID.String(), &v.BranchID, map[string]interface{}{"number": v.Number, "status": v.Status})
	c.JSON(http.StatusOK, quotationFlags(v, actor))
}

// ConvertQuotation turns a quotation into a sale (its goods go out of stock); answers the sale.
func (ic *InventoryController) ConvertQuotation(c *gin.Context) {
	actor, scope, ok := ic.scope(c)
	if !ok {
		return
	}
	id, ok := paramID(c)
	if !ok {
		return
	}
	var in services.ConvertQuotationInput
	if !bindJSON(c, &in) {
		return
	}
	sale, err := ic.svc.ConvertQuotation(c.Request.Context(), id, in, scope, actor.ActorID)
	if err != nil {
		writeInventoryError(c, err)
		return
	}
	ic.record(c, actor.ActorID, "inventory.sale.create", sale.ID.String(), &sale.BranchID, map[string]interface{}{"number": sale.Number, "total": sale.Total.String(), "currency": sale.Currency, "quotation": sale.QuotationNumber})
	ic.record(c, actor.ActorID, "inventory.quotation.convert", id.String(), &sale.BranchID, map[string]interface{}{"number": sale.QuotationNumber, "sale": sale.Number})
	c.JSON(http.StatusCreated, saleFlags(sale, actor))
}

func (ic *InventoryController) DeleteQuotation(c *gin.Context) {
	actor, scope, ok := ic.scope(c)
	if !ok {
		return
	}
	id, ok := paramID(c)
	if !ok {
		return
	}
	q, err := ic.svc.DeleteQuotation(c.Request.Context(), id, scope)
	if err != nil {
		writeInventoryError(c, err)
		return
	}
	ic.record(c, actor.ActorID, "inventory.quotation.delete", q.ID.String(), &q.BranchID, map[string]interface{}{"number": q.Number, "total": q.Total.String(), "currency": q.Currency})
	c.Status(http.StatusNoContent)
}
