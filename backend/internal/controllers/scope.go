package controllers

import (
	"net/http"

	"github.com/cambodia-finance/income-expense/internal/models"
	"github.com/cambodia-finance/income-expense/internal/permissions"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// accessScope is what the acting user may manage. Owners, admins and Super Admins
// (anyone with branches.manage) have full access; everyone else is limited to the
// branches they are assigned to.
type accessScope struct {
	ActorID     uuid.UUID
	Full        bool
	BranchIDs   []uuid.UUID
	branchSet   map[uuid.UUID]bool
	permissions map[string]bool
}

func loadScope(c *gin.Context, db *gorm.DB) (*accessScope, bool) {
	actorID, ok := getActorID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return nil, false
	}
	var user models.User
	if err := db.Preload("Roles.Permissions").Preload("Branches").First(&user, "id = ?", actorID).Error; err != nil {
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
	s.Full = user.IsSuperAdmin || s.permissions[permissions.BranchesManage]
	for _, b := range user.Branches {
		s.BranchIDs = append(s.BranchIDs, b.ID)
		s.branchSet[b.ID] = true
	}
	return s, true
}

// HasBranch reports whether the actor may work in the given branch.
func (s *accessScope) HasBranch(id uuid.UUID) bool {
	return s.Full || s.branchSet[id]
}

// SharesBranch reports whether the target user belongs to at least one of the actor's branches.
func (s *accessScope) SharesBranch(target *models.User) bool {
	if s.Full {
		return true
	}
	for _, b := range target.Branches {
		if s.branchSet[b.ID] {
			return true
		}
	}
	return false
}

// CoversPermissions reports whether every permission in roles is one the actor also holds,
// so a branch-level manager can never hand out (or manage) more power than they have.
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
