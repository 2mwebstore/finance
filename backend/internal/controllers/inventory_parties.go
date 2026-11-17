package controllers

import (
	"net/http"

	"github.com/cambodia-finance/income-expense/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ===== Customers and suppliers (each one shared, or one branch's own) =====

func (ic *InventoryController) ListCustomers(c *gin.Context) { ic.listParties(c, services.Customers) }
func (ic *InventoryController) ListSuppliers(c *gin.Context) { ic.listParties(c, services.Suppliers) }
func (ic *InventoryController) AddCustomer(c *gin.Context) {
	ic.saveParty(c, services.Customers, false)
}
func (ic *InventoryController) AddSupplier(c *gin.Context) {
	ic.saveParty(c, services.Suppliers, false)
}
func (ic *InventoryController) EditCustomer(c *gin.Context) {
	ic.saveParty(c, services.Customers, true)
}
func (ic *InventoryController) EditSupplier(c *gin.Context) {
	ic.saveParty(c, services.Suppliers, true)
}

func (ic *InventoryController) listParties(c *gin.Context, kind services.PartyKind) {
	_, scope, ok := ic.scope(c)
	if !ok {
		return
	}
	owner, ok := ownerFilter(c)
	if !ok {
		return
	}
	page, size := pageQuery(c)
	list, total, err := ic.svc.ListParties(c.Request.Context(), kind, services.PartyFilter{
		Search: c.Query("search"), Active: c.Query("active"), Owner: owner, Page: page, PageSize: size,
	}, scope)
	if err != nil {
		writeInventoryError(c, err)
		return
	}
	listJSON(c, orEmpty(list), total)
}

func (ic *InventoryController) saveParty(c *gin.Context, kind services.PartyKind, editing bool) {
	var id *uuid.UUID
	if editing {
		parsed, ok := paramID(c)
		if !ok {
			return
		}
		id = &parsed
	}
	var in services.PartyInput
	if !bindJSON(c, &in) {
		return
	}
	scope, ok := ic.catalog(c)
	if !ok {
		return
	}
	p, err := ic.svc.SaveParty(c.Request.Context(), kind, id, in, scope)
	if err != nil {
		writeInventoryError(c, err)
		return
	}
	if actor, ok := getActorID(c); ok {
		action := "inventory." + kind.Label() + ".create"
		if editing {
			action = "inventory." + kind.Label() + ".update"
		}
		ic.record(c, actor, action, p.ID.String(), p.BranchID, map[string]interface{}{"name": p.Name, "active": p.IsActive, "shared": p.BranchID == nil})
	}
	status := http.StatusOK
	if !editing {
		status = http.StatusCreated
	}
	c.JSON(status, p)
}
