package controllers

import (
	"net/http"
	"strconv"

	"github.com/cambodia-finance/income-expense/internal/models"
	"github.com/cambodia-finance/income-expense/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UserController struct {
	svc *services.UserService
	db  *gorm.DB
}

func NewUserController(svc *services.UserService, db *gorm.DB) *UserController {
	return &UserController{svc: svc, db: db}
}

const errRoleTooPowerful = "cannot assign a role with more permissions than your own"

// manageableUser loads the target user and checks the actor may manage them. Users outside the
// actor's branches look like they do not exist; owners/admins above the actor are forbidden.
func (uc *UserController) manageableUser(c *gin.Context, scope *accessScope) (*models.User, bool) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return nil, false
	}
	user, err := uc.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		writeServiceError(c, err)
		return nil, false
	}
	if !scope.SharesBranch(user) {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return nil, false
	}
	if !scope.CanManageUser(user) {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return nil, false
	}
	return user, true
}

// checkAssignableRoles rejects roles that would give the target more power than the actor has.
func (uc *UserController) checkAssignableRoles(c *gin.Context, scope *accessScope, codes []string) bool {
	if scope.Full || len(codes) == 0 {
		return true
	}
	var roles []models.Role
	if err := uc.db.Preload("Permissions").Where("code IN ?", codes).Find(&roles).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "role lookup failed"})
		return false
	}
	if !scope.CoversPermissions(roles) {
		c.JSON(http.StatusForbidden, gin.H{"error": errRoleTooPowerful})
		return false
	}
	return true
}

// checkAssignableBranches requires branch-level managers to stay within their own branches.
func (uc *UserController) checkAssignableBranches(c *gin.Context, scope *accessScope, branchIDs []string) bool {
	if scope.Full {
		return true
	}
	for _, raw := range branchIDs {
		id, err := uuid.Parse(raw)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid branch_id: " + raw})
			return false
		}
		if !scope.HasBranch(id) {
			c.JSON(http.StatusForbidden, gin.H{"error": "no access to this branch"})
			return false
		}
	}
	return true
}

func (uc *UserController) Create(c *gin.Context) {
	scope, ok := loadScope(c, uc.db)
	if !ok {
		return
	}
	var in services.CreateUserInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if !scope.Full && len(in.BranchIDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "choose at least one of your branches for the new user"})
		return
	}
	if !uc.checkAssignableBranches(c, scope, in.BranchIDs) || !uc.checkAssignableRoles(c, scope, in.RoleCodes) {
		return
	}
	user, err := uc.svc.Create(c.Request.Context(), in, scope.ActorID, actorIsSuperAdmin(c, uc.db))
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusCreated, userJSON(user))
}

func (uc *UserController) Update(c *gin.Context) {
	scope, ok := loadScope(c, uc.db)
	if !ok {
		return
	}
	target, ok := uc.manageableUser(c, scope)
	if !ok {
		return
	}
	var in services.UpdateUserInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if in.DefaultBranchID != nil && *in.DefaultBranchID != "" && !uc.checkAssignableBranches(c, scope, []string{*in.DefaultBranchID}) {
		return
	}
	user, err := uc.svc.Update(c.Request.Context(), target.ID, in)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, userJSON(user))
}

func (uc *UserController) SetActive(c *gin.Context) {
	scope, ok := loadScope(c, uc.db)
	if !ok {
		return
	}
	target, ok := uc.manageableUser(c, scope)
	if !ok {
		return
	}
	var body struct {
		IsActive *bool `json:"is_active" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if target.ID == scope.ActorID && !*body.IsActive {
		c.JSON(http.StatusForbidden, gin.H{"error": "cannot deactivate your own account"})
		return
	}
	user, err := uc.svc.SetActive(c.Request.Context(), target.ID, *body.IsActive)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, userJSON(user))
}

func (uc *UserController) SetPassword(c *gin.Context) {
	scope, ok := loadScope(c, uc.db)
	if !ok {
		return
	}
	target, ok := uc.manageableUser(c, scope)
	if !ok {
		return
	}
	if target.ID == scope.ActorID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "use change password to update your own password"})
		return
	}
	var body struct {
		Password string `json:"password" binding:"required,len=4"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := uc.svc.SetPassword(c.Request.Context(), target.ID, body.Password); err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "password updated"})
}

func (uc *UserController) Get(c *gin.Context) {
	scope, ok := loadScope(c, uc.db)
	if !ok {
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	user, err := uc.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	if !scope.SharesBranch(user) && user.ID != scope.ActorID {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	c.JSON(http.StatusOK, userJSON(user))
}

func (uc *UserController) List(c *gin.Context) {
	scope, ok := loadScope(c, uc.db)
	if !ok {
		return
	}
	f := services.UserListFilter{
		Search:     c.Query("search"),
		RoleCode:   c.Query("role"),
		Restricted: !scope.Full,
		BranchIDs:  scope.BranchIDs,
	}
	if v := c.Query("is_active"); v != "" {
		b := v == "true" || v == "1"
		f.IsActive = &b
	}
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
		f.BranchID = &branchID
	}
	f.Page, _ = strconv.Atoi(c.DefaultQuery("page", "1"))
	f.PageSize, _ = strconv.Atoi(c.DefaultQuery("page_size", "20"))

	users, total, err := uc.svc.List(c.Request.Context(), f)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "list failed"})
		return
	}
	items := make([]gin.H, 0, len(users))
	for i := range users {
		item := userJSON(&users[i])
		item["can_manage"] = scope.CanManageUser(&users[i])
		items = append(items, item)
	}
	c.JSON(http.StatusOK, gin.H{
		"items":     items,
		"total":     total,
		"page":      f.Page,
		"page_size": f.PageSize,
	})
}

func (uc *UserController) AssignRoles(c *gin.Context) {
	scope, ok := loadScope(c, uc.db)
	if !ok {
		return
	}
	target, ok := uc.manageableUser(c, scope)
	if !ok {
		return
	}
	var body struct {
		RoleCodes []string `json:"role_codes" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// Prevent self role assignment
	if target.ID == scope.ActorID {
		c.JSON(http.StatusForbidden, gin.H{"error": "cannot escalate your own privileges"})
		return
	}
	if !uc.checkAssignableRoles(c, scope, body.RoleCodes) {
		return
	}
	if err := uc.svc.AssignRoles(c.Request.Context(), target.ID, body.RoleCodes, actorIsSuperAdmin(c, uc.db)); err != nil {
		writeServiceError(c, err)
		return
	}
	user, _ := uc.svc.GetByID(c.Request.Context(), target.ID)
	c.JSON(http.StatusOK, userJSON(user))
}

func (uc *UserController) AssignBranches(c *gin.Context) {
	scope, ok := loadScope(c, uc.db)
	if !ok {
		return
	}
	target, ok := uc.manageableUser(c, scope)
	if !ok {
		return
	}
	var body struct {
		BranchIDs []string `json:"branch_ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if !uc.checkAssignableBranches(c, scope, body.BranchIDs) {
		return
	}
	branchIDs := body.BranchIDs
	if !scope.Full {
		// A branch manager only edits their own branches; keep the user's other branches as they are.
		for _, b := range target.Branches {
			if !scope.HasBranch(b.ID) {
				branchIDs = append(branchIDs, b.ID.String())
			}
		}
	}
	if err := uc.svc.AssignBranches(c.Request.Context(), target.ID, branchIDs); err != nil {
		writeServiceError(c, err)
		return
	}
	user, _ := uc.svc.GetByID(c.Request.Context(), target.ID)
	c.JSON(http.StatusOK, userJSON(user))
}
