package controllers

import (
	"net/http"

	"github.com/cambodia-finance/income-expense/internal/models"
	"github.com/cambodia-finance/income-expense/internal/permissions"
	"github.com/cambodia-finance/income-expense/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type CategoryController struct {
	svc *services.CategoryService
	db  *gorm.DB
}

func NewCategoryController(svc *services.CategoryService, db *gorm.DB) *CategoryController {
	return &CategoryController{svc: svc, db: db}
}

func (cc *CategoryController) List(c *gin.Context) {
	scope, ok := loadScope(c, cc.db)
	if !ok {
		return
	}
	typ := c.Query("type")
	activeOnly := c.Query("active") == "true" || c.Query("active") == "1"
	listScope := services.CategoryListScope{All: scope.AllBranches, BranchIDs: scope.BranchIDs, Search: c.Query("search")}
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
	cats, err := cc.svc.List(c.Request.Context(), typ, activeOnly, listScope)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "list failed"})
		return
	}
	names := cc.branchNames()
	items := make([]gin.H, 0, len(cats))
	for i := range cats {
		items = append(items, categoryJSON(&cats[i], names))
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (cc *CategoryController) Create(c *gin.Context) {
	scope, ok := loadScope(c, cc.db)
	if !ok {
		return
	}
	var in services.CreateCategoryInput
	if !bindJSON(c, &in) {
		return
	}
	if !canManageCategoryType(c, scope, in.Type) {
		return
	}
	var branchID *uuid.UUID
	if in.BranchID != nil && *in.BranchID != "" {
		id, err := uuid.Parse(*in.BranchID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid branch_id"})
			return
		}
		if !scope.HasBranch(id) {
			c.JSON(http.StatusForbidden, gin.H{"error": "no access to this branch"})
			return
		}
		branchID = &id
	}
	if branchID == nil && !scope.Full {
		c.JSON(http.StatusForbidden, gin.H{"error": "only the owner or an admin can create shared categories; choose your branch"})
		return
	}
	cat, err := cc.svc.Create(c.Request.Context(), in, branchID)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusCreated, categoryJSON(cat, cc.branchNames()))
}

func (cc *CategoryController) Update(c *gin.Context) {
	scope, ok := loadScope(c, cc.db)
	if !ok {
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	existing, err := cc.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	if !canManageCategoryType(c, scope, existing.Type) {
		return
	}
	if !scope.Full {
		if existing.BranchID == nil {
			c.JSON(http.StatusForbidden, gin.H{"error": "only the owner or an admin can change shared categories"})
			return
		}
		if !scope.HasBranch(*existing.BranchID) {
			c.JSON(http.StatusNotFound, gin.H{"error": "category not found"})
			return
		}
	}
	var in services.UpdateCategoryInput
	if !bindJSON(c, &in) {
		return
	}
	cat, err := cc.svc.Update(c.Request.Context(), id, in)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, categoryJSON(cat, cc.branchNames()))
}

func (cc *CategoryController) branchNames() map[uuid.UUID]string {
	var branches []models.Branch
	names := map[uuid.UUID]string{}
	if err := cc.db.Select("id", "code", "name").Find(&branches).Error; err == nil {
		for _, b := range branches {
			names[b.ID] = b.Code + " — " + b.Name
		}
	}
	return names
}

func categoryJSON(cat *models.Category, branchNames map[uuid.UUID]string) gin.H {
	out := gin.H{
		"id":          cat.ID,
		"code":        cat.Code,
		"name":        cat.Name,
		"type":        cat.Type,
		"description": cat.Description,
		"is_active":   cat.IsActive,
		"is_system":   cat.IsSystem,
		"branch_id":   nil,
		"branch_name": "",
	}
	if cat.BranchID != nil {
		out["branch_id"] = cat.BranchID
		out["branch_name"] = branchNames[*cat.BranchID]
	}
	return out
}

// canManageCategoryType: financial (ledger) expense categories need categories.manage; inventory
// expense categories need inventory.manage. They are managed on separate pages.
func canManageCategoryType(c *gin.Context, scope *accessScope, typ string) bool {
	need := permissions.CategoriesManage
	if typ == models.CategoryTypeInventoryExpense {
		need = permissions.InventoryManage
	}
	if scope.Can(need) {
		return true
	}
	c.JSON(http.StatusForbidden, gin.H{"error": "you don't have permission to manage these categories"})
	return false
}
