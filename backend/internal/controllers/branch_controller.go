package controllers

import (
	"net/http"
	"strconv"

	"github.com/cambodia-finance/income-expense/internal/middleware"
	"github.com/cambodia-finance/income-expense/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type BranchController struct {
	svc *services.BranchService
	db  *gorm.DB
}

func NewBranchController(svc *services.BranchService, db *gorm.DB) *BranchController {
	return &BranchController{svc: svc, db: db}
}

func (bc *BranchController) Create(c *gin.Context) {
	var in services.CreateBranchInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	branch, err := bc.svc.Create(c.Request.Context(), in)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusCreated, branchJSON(branch))
}

func (bc *BranchController) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var in services.UpdateBranchInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	branch, err := bc.svc.Update(c.Request.Context(), id, in)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, branchJSON(branch))
}

func (bc *BranchController) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	branch, err := bc.svc.Delete(c.Request.Context(), id)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, branchJSON(branch))
}

func (bc *BranchController) Get(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	// Branch access check
	actorID, ok := getActorID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	allowed, all, err := middleware.GetAuthorizedBranchIDs(bc.db, actorID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "authorization check failed"})
		return
	}
	if !all {
		found := false
		for _, aid := range allowed {
			if aid == id {
				found = true
				break
			}
		}
		if !found {
			c.JSON(http.StatusForbidden, gin.H{"error": "no access to this branch"})
			return
		}
	}
	branch, err := bc.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, branchJSON(branch))
}

func (bc *BranchController) List(c *gin.Context) {
	actorID, ok := getActorID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	allowed, all, err := middleware.GetAuthorizedBranchIDs(bc.db, actorID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "authorization check failed"})
		return
	}

	f := services.BranchListFilter{
		Search: c.Query("search"),
	}
	if v := c.Query("is_active"); v != "" {
		b := v == "true" || v == "1"
		f.IsActive = &b
	}
	f.Page, _ = strconv.Atoi(c.DefaultQuery("page", "1"))
	f.PageSize, _ = strconv.Atoi(c.DefaultQuery("page_size", "20"))

	branches, total, err := bc.svc.List(c.Request.Context(), f, allowed, all)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "list failed"})
		return
	}
	items := make([]gin.H, 0, len(branches))
	for i := range branches {
		items = append(items, branchJSON(&branches[i]))
	}
	c.JSON(http.StatusOK, gin.H{
		"items":     items,
		"total":     total,
		"page":      f.Page,
		"page_size": f.PageSize,
	})
}

func (bc *BranchController) AssignUsers(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var body struct {
		UserIDs []string `json:"user_ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	uids := make([]uuid.UUID, 0, len(body.UserIDs))
	for _, s := range body.UserIDs {
		uid, err := uuid.Parse(s)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id: " + s})
			return
		}
		uids = append(uids, uid)
	}
	if err := bc.svc.AssignUsers(c.Request.Context(), id, uids); err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "users assigned"})
}
