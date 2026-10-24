package controllers

import (
	"testing"

	"github.com/cambodia-finance/income-expense/internal/models"
	"github.com/google/uuid"
)

func TestCoversPermissionsChecksPermissions(t *testing.T) {
	pp := uuid.New()
	manager := &accessScope{
		BranchIDs:   []uuid.UUID{pp},
		branchSet:   map[uuid.UUID]bool{pp: true},
		permissions: map[string]bool{"income.view": true},
	}
	cases := []struct {
		name string
		role models.Role
		want bool
	}{
		{"own permissions", models.Role{Permissions: []models.Permission{{Code: "income.view"}}}, true},
		{"more permissions", models.Role{Permissions: []models.Permission{{Code: "users.manage"}}}, false},
		{"super admin role", models.Role{Code: "super_admin"}, false},
	}
	for _, c := range cases {
		if got := manager.CoversPermissions([]models.Role{c.role}); got != c.want {
			t.Errorf("%s: CoversPermissions = %v, want %v", c.name, got, c.want)
		}
	}
	if !(&accessScope{Full: true}).CoversPermissions([]models.Role{{Code: "super_admin"}}) {
		t.Error("full access covers every role")
	}
}
