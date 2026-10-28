package controllers

import (
	"net/http"
	"strconv"

	"github.com/cambodia-finance/income-expense/internal/models"
	"github.com/cambodia-finance/income-expense/internal/permissions"
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

// checkAssignableRoles rejects roles that would give the target more power than the actor has,
// and branch roles of a branch the user isn't in.
func (uc *UserController) checkAssignableRoles(c *gin.Context, scope *accessScope, codes []string, userBranches []uuid.UUID) bool {
	if len(codes) == 0 {
		return true
	}
	var roles []models.Role
	if err := uc.db.Preload("Permissions").Where("code IN ?", codes).Find(&roles).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "role lookup failed"})
		return false
	}
	in := map[uuid.UUID]bool{}
	for _, id := range userBranches {
		in[id] = true
	}
	for _, r := range roles {
		if r.BranchID != nil && !in[*r.BranchID] {
			msg := "the role " + r.Name + " is for another branch; add the user to that branch first"
			c.JSON(http.StatusBadRequest, gin.H{"error": msg, "fields": gin.H{"role_codes": msg}})
			return false
		}
	}
	if scope.Full {
		return true
	}
	if !scope.CoversPermissions(roles) {
		c.JSON(http.StatusForbidden, gin.H{"error": errRoleTooPowerful})
		return false
	}
	return true
}

// checkAssignableBranches keeps actors within the branches of their data scope.
func (uc *UserController) checkAssignableBranches(c *gin.Context, scope *accessScope, branchIDs []string) bool {
	if scope.AllBranches {
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

// checkManagers validates manager IDs for a user: existing active users who can approve or reject,
// not the user themself, working in one of the user's branches (owners/admins always qualify),
// and, for branch-level actors, people from the actor's own branches.
func (uc *UserController) checkManagers(c *gin.Context, scope *accessScope, userID uuid.UUID, userBranches []uuid.UUID, raw []string) ([]uuid.UUID, bool) {
	ids := make([]uuid.UUID, 0, len(raw))
	seen := map[uuid.UUID]bool{}
	for _, r := range raw {
		id, err := uuid.Parse(r)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid manager id: " + r})
			return nil, false
		}
		if id == userID {
			c.JSON(http.StatusBadRequest, gin.H{"error": "a user can't be their own manager"})
			return nil, false
		}
		if !seen[id] {
			seen[id] = true
			ids = append(ids, id)
		}
	}
	for _, id := range ids {
		manager, err := uc.svc.GetByID(c.Request.Context(), id)
		if err != nil || !manager.IsActive {
			c.JSON(http.StatusBadRequest, gin.H{"error": "manager not found or inactive"})
			return nil, false
		}
		if !scope.SharesBranch(manager) {
			c.JSON(http.StatusForbidden, gin.H{"error": "managers must come from your branches"})
			return nil, false
		}
		approver, full := manager.IsSuperAdmin, manager.IsSuperAdmin
		for _, role := range manager.Roles {
			for _, p := range role.Permissions {
				switch p.Code {
				case permissions.TransactionsApprove, permissions.TransactionsReject:
					approver = true
				case permissions.BranchesManage:
					full = true
				}
			}
		}
		if !approver {
			c.JSON(http.StatusBadRequest, gin.H{"error": manager.FullName + " can't approve or reject transactions (their role lacks that permission)"})
			return nil, false
		}
		if !full && !sharesAny(manager.Branches, userBranches) {
			c.JSON(http.StatusBadRequest, gin.H{"error": manager.FullName + " doesn't work in any of this user's branches"})
			return nil, false
		}
	}
	return ids, true
}

func sharesAny(branches []models.Branch, ids []uuid.UUID) bool {
	for _, b := range branches {
		for _, id := range ids {
			if b.ID == id {
				return true
			}
		}
	}
	return false
}

// withManagers adds each user's managers to the JSON rows.
func (uc *UserController) withManagers(c *gin.Context, rows []gin.H, ids []uuid.UUID) {
	managers, err := uc.svc.ManagersFor(c.Request.Context(), ids)
	if err != nil {
		return
	}
	for i := range rows {
		list := managers[ids[i]]
		if list == nil {
			list = []services.ManagerSummary{}
		}
		rows[i]["managers"] = list
	}
}

func (uc *UserController) Create(c *gin.Context) {
	scope, ok := loadScope(c, uc.db)
	if !ok {
		return
	}
	var in services.CreateUserInput
	if !bindJSON(c, &in) {
		return
	}
	if !scope.AllBranches && len(in.BranchIDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "choose at least one of your branches for the new user"})
		return
	}
	if !uc.checkAssignableBranches(c, scope, in.BranchIDs) {
		return
	}
	newBranches := make([]uuid.UUID, 0, len(in.BranchIDs))
	for _, raw := range in.BranchIDs {
		if id, err := uuid.Parse(raw); err == nil {
			newBranches = append(newBranches, id)
		}
	}
	if !uc.checkAssignableRoles(c, scope, in.RoleCodes, newBranches) {
		return
	}
	managerIDs, ok := uc.checkManagers(c, scope, uuid.Nil, newBranches, in.ManagerIDs)
	if !ok {
		return
	}
	user, err := uc.svc.Create(c.Request.Context(), in, scope.ActorID, actorIsSuperAdmin(c, uc.db))
	if err != nil {
		writeServiceError(c, err)
		return
	}
	if len(managerIDs) > 0 {
		if err := uc.svc.SetManagers(c.Request.Context(), user.ID, managerIDs); err != nil {
			writeServiceError(c, err)
			return
		}
	}
	rows := []gin.H{userJSON(user)}
	uc.withManagers(c, rows, []uuid.UUID{user.ID})
	c.JSON(http.StatusCreated, rows[0])
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
	if !bindJSON(c, &in) {
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
	if !bindJSON(c, &body) {
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
		Password string `json:"password" binding:"required,min=4,max=6,numeric"`
	}
	if !bindJSON(c, &body) {
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
		Restricted: !scope.AllBranches,
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
	ids := make([]uuid.UUID, 0, len(users))
	for i := range users {
		item := userJSON(&users[i])
		item["can_manage"] = scope.CanManageUser(&users[i])
		items = append(items, item)
		ids = append(ids, users[i].ID)
	}
	uc.withManagers(c, items, ids)
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
	if !bindJSON(c, &body) {
		return
	}
	// Prevent self role assignment
	if target.ID == scope.ActorID {
		c.JSON(http.StatusForbidden, gin.H{"error": "cannot escalate your own privileges"})
		return
	}
	userBranches := make([]uuid.UUID, 0, len(target.Branches))
	for _, b := range target.Branches {
		userBranches = append(userBranches, b.ID)
	}
	if !uc.checkAssignableRoles(c, scope, body.RoleCodes, userBranches) {
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
	if !bindJSON(c, &body) {
		return
	}
	if !uc.checkAssignableBranches(c, scope, body.BranchIDs) {
		return
	}
	branchIDs := body.BranchIDs
	if !scope.AllBranches {
		// A branch manager only edits branches in their scope; keep the user's other branches as they are.
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

// AssignManagers sets who approves or rejects this user's pending transactions.
func (uc *UserController) AssignManagers(c *gin.Context) {
	scope, ok := loadScope(c, uc.db)
	if !ok {
		return
	}
	target, ok := uc.manageableUser(c, scope)
	if !ok {
		return
	}
	if target.ID == scope.ActorID {
		c.JSON(http.StatusForbidden, gin.H{"error": "you can't choose your own managers"})
		return
	}
	var body struct {
		ManagerIDs []string `json:"manager_ids"`
	}
	if !bindJSON(c, &body) {
		return
	}
	targetBranches := make([]uuid.UUID, 0, len(target.Branches))
	for _, b := range target.Branches {
		targetBranches = append(targetBranches, b.ID)
	}
	managerIDs, ok := uc.checkManagers(c, scope, target.ID, targetBranches, body.ManagerIDs)
	if !ok {
		return
	}
	if err := uc.svc.SetManagers(c.Request.Context(), target.ID, managerIDs); err != nil {
		writeServiceError(c, err)
		return
	}
	user, _ := uc.svc.GetByID(c.Request.Context(), target.ID)
	rows := []gin.H{userJSON(user)}
	uc.withManagers(c, rows, []uuid.UUID{user.ID})
	c.JSON(http.StatusOK, rows[0])
}
