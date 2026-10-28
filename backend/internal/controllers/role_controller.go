package controllers

import (
	"net/http"
	"strings"

	"github.com/cambodia-finance/income-expense/internal/models"
	"github.com/cambodia-finance/income-expense/internal/permissions"
	"github.com/cambodia-finance/income-expense/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type RoleController struct {
	svc *services.RoleService
	db  *gorm.DB
}

func NewRoleController(svc *services.RoleService, db *gorm.DB) *RoleController {
	return &RoleController{svc: svc, db: db}
}

// Roles are shared by every branch. The owner/admin (full access) may change any of them. Anyone
// else with roles.manage may add roles and change the ones that are theirs to change (see
// blockedReason), and never give a role a permission they don't have themselves.
const msgRoleTooWide = "you can only give a role the permissions you have yourself"

// specRole is a role with these permissions, for CoversPermissions.
func specRole(codes []string) models.Role {
	role := models.Role{DataScope: models.DataScopeAssigned}
	for _, code := range codes {
		role.Permissions = append(role.Permissions, models.Permission{Code: code})
	}
	return role
}

// blockedReason says why the actor may not change or delete the role ("" when they may). Without
// full access: no system role (every branch uses those), no role with a permission the actor
// lacks, and no role held by people outside the actor's branches.
func (rc *RoleController) blockedReason(scope *accessScope, role *models.Role) (string, error) {
	if scope.Full {
		return "", nil
	}
	if role.IsSystem {
		return "only the owner or an admin can change system roles", nil
	}
	// Branch-level managers look after their branches' roles; shared roles are the owner's.
	if role.BranchID == nil {
		return "only the owner or an admin can change roles shared by every branch", nil
	}
	if !scope.HasBranch(*role.BranchID) {
		return "this role belongs to another branch", nil
	}
	if !scope.CoversPermissions([]models.Role{*role}) {
		return "this role has permissions you don't have, so only the owner or an admin can change it", nil
	}
	if scope.AllBranches {
		return "", nil
	}
	var holders []models.User
	err := rc.db.Preload("Branches").
		Joins("JOIN user_roles ON user_roles.user_id = users.id").
		Where("user_roles.role_id = ?", role.ID).
		Find(&holders).Error
	if err != nil {
		return "", err
	}
	for i := range holders {
		if !scope.SharesBranch(&holders[i]) {
			return "people outside your branches have this role, so only the owner or an admin can change it", nil
		}
	}
	return "", nil
}

// loadChangeable loads the role and checks the actor may change it; it answers the request when not.
func (rc *RoleController) loadChangeable(c *gin.Context, scope *accessScope, id uuid.UUID) (*models.Role, bool) {
	role, err := rc.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		writeServiceError(c, err)
		return nil, false
	}
	reason, err := rc.blockedReason(scope, role)
	if err != nil {
		writeServiceError(c, err)
		return nil, false
	}
	if reason != "" {
		c.JSON(http.StatusForbidden, gin.H{"error": reason})
		return nil, false
	}
	return role, true
}

// branchOrDeny checks the branch a role is for ("" = shared): the owner/admin may pick any
// branch or none; others must pick one of their own branches.
func branchOrDeny(c *gin.Context, scope *accessScope, raw string) bool {
	if strings.TrimSpace(raw) == "" {
		if scope.Full {
			return true
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": "choose your branch for this role", "fields": gin.H{"branch_id": "choose your branch for this role"}})
		return false
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "choose a valid branch", "fields": gin.H{"branch_id": "choose a valid branch"}})
		return false
	}
	if !scope.Full && !scope.HasBranch(id) {
		c.JSON(http.StatusForbidden, gin.H{"error": "you can only make roles for your own branches"})
		return false
	}
	return true
}

// coversOrDeny answers 403 when the role would get a permission the actor lacks.
func coversOrDeny(c *gin.Context, scope *accessScope, role models.Role) bool {
	if scope.Full || scope.CoversPermissions([]models.Role{role}) {
		return true
	}
	c.JSON(http.StatusForbidden, gin.H{"error": msgRoleTooWide})
	return false
}

func (rc *RoleController) Create(c *gin.Context) {
	var in services.CreateRoleInput
	if !bindJSON(c, &in) {
		return
	}
	scope, ok := loadScope(c, rc.db)
	if !ok {
		return
	}
	if !branchOrDeny(c, scope, in.BranchID) || !coversOrDeny(c, scope, specRole(in.PermissionCodes)) {
		return
	}
	role, err := rc.svc.Create(c.Request.Context(), in)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusCreated, roleJSON(role))
}

func (rc *RoleController) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var in services.UpdateRoleInput
	if !bindJSON(c, &in) {
		return
	}
	scope, ok := loadScope(c, rc.db)
	if !ok {
		return
	}
	if _, ok := rc.loadChangeable(c, scope, id); !ok {
		return
	}
	if in.PermissionCodes != nil && !coversOrDeny(c, scope, specRole(in.PermissionCodes)) {
		return
	}
	if in.BranchID != nil && !branchOrDeny(c, scope, *in.BranchID) {
		return
	}
	role, err := rc.svc.Update(c.Request.Context(), id, in)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, roleJSON(role))
}

func (rc *RoleController) Get(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	role, err := rc.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, roleJSON(role))
}

// List: the owner/admin see every role; others see shared roles and their branches' roles.
// branch_id narrows to shared roles and that branch's roles (e.g. for a user of that branch).
func (rc *RoleController) List(c *gin.Context) {
	scope, ok := loadScope(c, rc.db)
	if !ok {
		return
	}
	f := services.RoleListFilter{Search: c.Query("search"), All: scope.Full || scope.AllBranches, BranchIDs: scope.BranchIDs}
	if raw := c.Query("branch_id"); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil || !scope.HasBranch(id) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid branch_id"})
			return
		}
		f.ForBranch = &id
	}
	roles, err := rc.svc.List(c.Request.Context(), f)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "list failed"})
		return
	}
	branchNames := map[uuid.UUID]models.Branch{}
	var branchList []models.Branch
	rc.db.Select("id", "code", "name").Find(&branchList)
	for _, b := range branchList {
		branchNames[b.ID] = b
	}
	mayManage := scope.Full || scope.permissions[permissions.RolesManage]
	items := make([]gin.H, 0, len(roles))
	for i := range roles {
		item := roleJSON(&roles[i])
		if b, ok := branchNames[derefID(roles[i].BranchID)]; ok {
			item["branch"] = gin.H{"id": b.ID, "code": b.Code, "name": b.Name}
		}
		// can_manage: whether this user may edit (and, for non-system roles, delete) the role.
		item["can_manage"] = false
		if mayManage {
			reason, err := rc.blockedReason(scope, &roles[i])
			if err != nil {
				writeServiceError(c, err)
				return
			}
			item["can_manage"] = reason == ""
		}
		items = append(items, item)
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (rc *RoleController) ListPermissions(c *gin.Context) {
	perms, err := rc.svc.ListPermissions(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "list failed"})
		return
	}
	items := make([]gin.H, 0, len(perms))
	for _, p := range perms {
		items = append(items, gin.H{
			"id":          p.ID,
			"code":        p.Code,
			"name":        p.Name,
			"module":      p.Module,
			"description": p.Description,
		})
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (rc *RoleController) SetPermissions(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var body struct {
		PermissionCodes []string `json:"permission_codes" binding:"required"`
	}
	if !bindJSON(c, &body) {
		return
	}
	scope, ok := loadScope(c, rc.db)
	if !ok {
		return
	}
	if _, ok := rc.loadChangeable(c, scope, id); !ok {
		return
	}
	if !coversOrDeny(c, scope, specRole(body.PermissionCodes)) {
		return
	}
	role, err := rc.svc.SetPermissions(c.Request.Context(), id, body.PermissionCodes)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, roleJSON(role))
}

func (rc *RoleController) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	scope, ok := loadScope(c, rc.db)
	if !ok {
		return
	}
	if _, ok := rc.loadChangeable(c, scope, id); !ok {
		return
	}
	if err := rc.svc.Delete(c.Request.Context(), id); err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "role deleted"})
}

func derefID(id *uuid.UUID) uuid.UUID {
	if id == nil {
		return uuid.Nil
	}
	return *id
}
