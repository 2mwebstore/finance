package controllers

import (
	"testing"

	"github.com/cambodia-finance/income-expense/internal/models"
	"github.com/google/uuid"
)

func TestCoversPermissionsChecksDataScope(t *testing.T) {
	pp, sr := uuid.New(), uuid.New()
	manager := &accessScope{
		BranchIDs:   []uuid.UUID{pp},
		branchSet:   map[uuid.UUID]bool{pp: true},
		permissions: map[string]bool{"income.view": true},
	}
	perm := []models.Permission{{Code: "income.view"}}
	cases := []struct {
		name string
		role models.Role
		want bool
	}{
		{"own branches", models.Role{DataScope: models.DataScopeAssigned, Permissions: perm}, true},
		{"custom inside the manager's branches", models.Role{DataScope: models.DataScopeCustom, Permissions: perm, Branches: []models.Branch{{Base: models.Base{ID: pp}}}}, true},
		{"custom reaching another branch", models.Role{DataScope: models.DataScopeCustom, Permissions: perm, Branches: []models.Branch{{Base: models.Base{ID: sr}}}}, false},
		{"all branches", models.Role{DataScope: models.DataScopeAll, Permissions: perm}, false},
		{"more permissions", models.Role{DataScope: models.DataScopeAssigned, Permissions: []models.Permission{{Code: "users.manage"}}}, false},
	}
	for _, c := range cases {
		if got := manager.CoversPermissions([]models.Role{c.role}); got != c.want {
			t.Errorf("%s: CoversPermissions = %v, want %v", c.name, got, c.want)
		}
	}
	// Someone who already sees every branch may hand out an all-branches role.
	wide := &accessScope{AllBranches: true, branchSet: map[uuid.UUID]bool{}, permissions: map[string]bool{"income.view": true}}
	if !wide.CoversPermissions([]models.Role{{DataScope: models.DataScopeAll, Permissions: perm}}) {
		t.Error("an all-branches actor should cover an all-branches role")
	}
}
