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

// BranchScope works out which branches' data a user may see, from their roles' data scopes
// (user loaded with Branches, Roles.Permissions and Roles.Branches):
//   - Super Admin, branches.manage, or a role with the retired scope "all" (left by an older
//     backup until the next start converts it): every branch (all = true);
//   - otherwise the union of the user's assigned branches (roles with scope "assigned") and the
//     branches picked on roles with scope "custom".
func BranchScope(user *models.User) (ids []uuid.UUID, all bool) {
	if IsFullAccess(user) {
		return nil, true
	}
	seen := map[uuid.UUID]bool{}
	add := func(id uuid.UUID) {
		if !seen[id] {
			seen[id] = true
			ids = append(ids, id)
		}
	}
	for _, role := range user.Roles {
		switch role.DataScope {
		case models.DataScopeAll:
			return nil, true
		case models.DataScopeCustom:
			for _, b := range role.Branches {
				add(b.ID)
			}
		default:
			for _, b := range user.Branches {
				add(b.ID)
			}
		}
	}
	return ids, false
}
