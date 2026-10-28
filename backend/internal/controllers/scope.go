package controllers

import (
	"net/http"

	"github.com/cambodia-finance/income-expense/internal/access"
	"github.com/cambodia-finance/income-expense/internal/middleware"
	"github.com/cambodia-finance/income-expense/internal/models"
	"github.com/cambodia-finance/income-expense/internal/permissions"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// accessScope is what the acting user may see and manage.
//   - Full: owner/admin/Super Admin authority (branches.manage): shared categories, roles,
//     managing anyone.
//   - AllBranches / BranchIDs: whose data they see (see access.BranchScope): every branch with
//     full access, otherwise only their own branches.
type accessScope struct {
	ActorID     uuid.UUID
	Full        bool
	AllBranches bool
	BranchIDs   []uuid.UUID
	branchSet   map[uuid.UUID]bool
	permissions map[string]bool
	superAdmin  bool
}

// Can reports whether the actor holds the permission (Super Admin holds every one).
func (s *accessScope) Can(code string) bool {
	return s.superAdmin || s.permissions[code]
}

func loadScope(c *gin.Context, db *gorm.DB) (*accessScope, bool) {
	actorID, ok := getActorID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return nil, false
	}
	user, err := middleware.LoadForScope(db, actorID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found"})
		return nil, false
	}
	s := &accessScope{
		ActorID:     actorID,
		branchSet:   map[uuid.UUID]bool{},
		permissions: map[string]bool{},
	}
	for _, role := range user.Roles {
		for _, p := range role.Permissions {
			s.permissions[p.Code] = true
		}
	}
	s.Full = access.IsFullAccess(user)
	s.superAdmin = user.IsSuperAdmin
	s.BranchIDs, s.AllBranches = access.BranchScope(user)
	for _, id := range s.BranchIDs {
		s.branchSet[id] = true
	}
	return s, true
}

// HasBranch reports whether the branch is in the actor's data scope.
func (s *accessScope) HasBranch(id uuid.UUID) bool {
	return s.AllBranches || s.branchSet[id]
}

// SharesBranch reports whether the target user belongs to at least one branch in the actor's scope.
func (s *accessScope) SharesBranch(target *models.User) bool {
	if s.AllBranches {
		return true
	}
	for _, b := range target.Branches {
		if s.branchSet[b.ID] {
			return true
		}
	}
	return false
}

// CoversPermissions reports whether the actor holds every permission in roles, so a branch-level
// manager can never hand out (or manage) more power than they have. Roles need Permissions loaded.
// (Roles carry no branches: everyone sees only their own branches.)
func (s *accessScope) CoversPermissions(roles []models.Role) bool {
	if s.Full {
		return true
	}
	for _, role := range roles {
		if role.Code == "super_admin" {
			return false
		}
		for _, p := range role.Permissions {
			if !s.permissions[p.Code] {
				return false
			}
		}
	}
	return true
}

// CanManageUser: branch-level managers may only manage users in their branches who are
// not owners/admins/Super Admins and hold no permission the manager lacks.
func (s *accessScope) CanManageUser(target *models.User) bool {
	if s.Full {
		return true
	}
	if target.IsSuperAdmin || !s.SharesBranch(target) {
		return false
	}
	for _, role := range target.Roles {
		for _, p := range role.Permissions {
			if p.Code == permissions.BranchesManage {
				return false
			}
		}
	}
	return s.CoversPermissions(target.Roles)
}
