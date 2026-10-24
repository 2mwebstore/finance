package controllers

import (
	"testing"

	"github.com/cambodia-finance/income-expense/internal/models"
)

func TestSpecRoleIsCheckedAgainstTheActor(t *testing.T) {
	manager := &accessScope{permissions: map[string]bool{"income.view": true, "roles.manage": true}}
	if !manager.CoversPermissions([]models.Role{specRole([]string{"income.view"})}) {
		t.Error("own permissions should be allowed")
	}
	if manager.CoversPermissions([]models.Role{specRole([]string{"income.view", "users.manage"})}) {
		t.Error("a permission the actor lacks should be refused")
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
