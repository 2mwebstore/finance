package controllers

import (
	"testing"

	"github.com/cambodia-finance/income-expense/internal/models"
	"github.com/google/uuid"
)

func TestSpecRoleIsCheckedAgainstTheActor(t *testing.T) {
	mine, other := uuid.New(), uuid.New()
	manager := &accessScope{
		BranchIDs:   []uuid.UUID{mine},
		branchSet:   map[uuid.UUID]bool{mine: true},
		permissions: map[string]bool{"income.view": true, "roles.manage": true},
	}
	cases := []struct {
		name string
		role models.Role
		want bool
	}{
		{"own permissions, assigned", specRole([]string{"income.view"}, models.DataScopeAssigned, nil), true},
		{"own branch", specRole([]string{"income.view"}, models.DataScopeCustom, []string{mine.String()}), true},
		{"another branch", specRole(nil, models.DataScopeCustom, []string{mine.String(), other.String()}), false},
		{"a permission they lack", specRole([]string{"income.view", "users.manage"}, models.DataScopeAssigned, nil), false},
		{"retired all scope", specRole(nil, models.DataScopeAll, nil), false},
		{"bad branch ids are left to the service", specRole(nil, models.DataScopeCustom, []string{"nope"}), true},
	}
	for _, tc := range cases {
		if got := manager.CoversPermissions([]models.Role{tc.role}); got != tc.want {
			t.Errorf("%s: got %v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestBlockedReasonWithoutFullAccess(t *testing.T) {
	manager := &accessScope{AllBranches: true, permissions: map[string]bool{"income.view": true}}
	rc := &RoleController{}
	if reason, _ := rc.blockedReason(manager, &models.Role{IsSystem: true}); reason == "" {
		t.Error("a system role must be blocked")
	}
	wide := &models.Role{Permissions: []models.Permission{{Code: "users.manage"}}}
	if reason, _ := rc.blockedReason(manager, wide); reason == "" {
		t.Error("a role with a permission the actor lacks must be blocked")
	}
	own := &models.Role{Permissions: []models.Permission{{Code: "income.view"}}}
	if reason, err := rc.blockedReason(manager, own); reason != "" || err != nil {
		t.Errorf("own role blocked: %q %v", reason, err)
	}
	if reason, _ := rc.blockedReason(&accessScope{Full: true}, &models.Role{IsSystem: true}); reason != "" {
		t.Error("full access may change any role")
	}
}
