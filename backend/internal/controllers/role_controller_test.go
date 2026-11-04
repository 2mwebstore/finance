package controllers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/cambodia-finance/income-expense/internal/models"
	"github.com/cambodia-finance/income-expense/internal/services"
	"github.com/gin-gonic/gin"
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

func TestRoleVisible(t *testing.T) {
	mine, other := uuid.New(), uuid.New()
	branchRole := &models.Role{Code: "cashier-bt", BranchID: &mine}
	otherRole := &models.Role{Code: "cashier-pp", BranchID: &other}
	builtIn := &models.Role{Code: "manager", IsSystem: true}
	owner := &models.Role{Code: services.OwnerRoleCode, IsSystem: true}

	branchUser := &accessScope{BranchIDs: []uuid.UUID{mine}, branchSet: map[uuid.UUID]bool{mine: true}}
	ownerUser := &accessScope{Full: true, AllBranches: true}
	superAdmin := &accessScope{Full: true, AllBranches: true, superAdmin: true}

	cases := []struct {
		name  string
		scope *accessScope
		role  *models.Role
		want  bool
	}{
		{"branch user sees their branch's role", branchUser, branchRole, true},
		{"branch user doesn't see another branch's role", branchUser, otherRole, false},
		{"branch user doesn't see a built-in role", branchUser, builtIn, false},
		{"branch user doesn't see the Owner role", branchUser, owner, false},
		{"owner sees a built-in role", ownerUser, builtIn, true},
		{"owner sees any branch's role", ownerUser, otherRole, true},
		{"owner doesn't see the Owner role", ownerUser, owner, false},
		{"Super Admin sees the Owner role", superAdmin, owner, true},
	}
	for _, tc := range cases {
		if got := roleVisible(tc.scope, tc.role); got != tc.want {
			t.Errorf("%s: got %v", tc.name, got)
		}
	}
}

func TestRolesMustHaveABranch(t *testing.T) {
	gin.SetMode(gin.TestMode)
	branch := uuid.New()
	owner := &accessScope{Full: true, AllBranches: true}
	branchUser := &accessScope{BranchIDs: []uuid.UUID{branch}, branchSet: map[uuid.UUID]bool{branch: true}}
	cases := []struct {
		name  string
		scope *accessScope
		raw   string
		want  int
	}{
		{"the owner can't share a role", owner, "", http.StatusBadRequest},
		{"a blank branch is no branch", owner, "   ", http.StatusBadRequest},
		{"the owner may pick any branch", owner, uuid.NewString(), http.StatusOK},
		{"a branch user can't share a role", branchUser, "", http.StatusBadRequest},
		{"a branch user may pick their branch", branchUser, branch.String(), http.StatusOK},
		{"a branch user can't pick another branch", branchUser, uuid.NewString(), http.StatusForbidden},
	}
	for _, tc := range cases {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		allowed := branchOrDeny(c, tc.scope, tc.raw)
		if allowed != (tc.want == http.StatusOK) || (!allowed && w.Code != tc.want) {
			t.Errorf("%s: allowed=%v status=%d, want %d", tc.name, allowed, w.Code, tc.want)
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
