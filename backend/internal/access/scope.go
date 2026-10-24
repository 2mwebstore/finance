// Package access holds the rules for who sees which branches' data and who has owner/admin
// authority. They work on a loaded user, so any package can use them without a database.
package access

import (
	"github.com/cambodia-finance/income-expense/internal/models"
	"github.com/google/uuid"
)

// IsFullAccess reports owner/admin authority (Super Admin, or a role with branches.manage) for a
// user loaded with Roles.Permissions.
func IsFullAccess(user *models.User) bool {
	if user.IsSuperAdmin {
		return true
	}
	for _, role := range user.Roles {
		for _, p := range role.Permissions {
			if p.Code == "branches.manage" {
				return true
			}
		}
	}
	return false
}

// BranchScope works out which branches' data a user may see (user loaded with Branches and
// Roles.Permissions). Data is never shared between branches:
//   - Super Admin or branches.manage (owner/admin): every branch (all = true);
//   - everyone else: only the branches they are assigned to.
func BranchScope(user *models.User) (ids []uuid.UUID, all bool) {
	if IsFullAccess(user) {
		return nil, true
	}
	for _, b := range user.Branches {
		ids = append(ids, b.ID)
	}
	return ids, false
}
