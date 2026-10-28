package controllers

import (
	"testing"

	"github.com/cambodia-finance/income-expense/internal/models"
	"github.com/google/uuid"
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
	branch := uuid.New()
	shared := &models.Role{Permissions: []models.Permission{{Code: "income.view"}}}
	if reason, _ := rc.blockedReason(manager, shared); reason == "" {
		t.Error("a role shared by every branch must be left to the owner/admin")
	}
	own := &models.Role{BranchID: &branch, Permissions: []models.Permission{{Code: "income.view"}}}
	if reason, err := rc.blockedReason(manager, own); reason != "" || err != nil {
		t.Errorf("own branch role blocked: %q %v", reason, err)
	}
	other := uuid.New()
	inOneBranch := &accessScope{BranchIDs: []uuid.UUID{branch}, branchSet: map[uuid.UUID]bool{branch: true}, permissions: map[string]bool{"income.view": true}}
	if reason, _ := rc.blockedReason(inOneBranch, &models.Role{BranchID: &other, Permissions: own.Permissions}); reason == "" {
		t.Error("another branch's role must be blocked")
	}
	if reason, _ := rc.blockedReason(&accessScope{Full: true}, &models.Role{IsSystem: true}); reason != "" {
		t.Error("full access may change any role")
	}
}
