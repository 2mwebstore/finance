package permissions

import (
	"strings"
	"testing"
)

func TestAllPermissionsUniqueAndNonEmpty(t *testing.T) {
	list := AllPermissions()
	if len(list) != 31 {
		t.Fatalf("expected 31 permissions, got %d", len(list))
	}
	seen := map[string]bool{}
	for _, p := range list {
		if p.Code == "" || p.Name == "" || p.Module == "" {
			t.Fatalf("incomplete permission: %+v", p)
		}
		if !strings.Contains(p.Code, ".") {
			t.Errorf("code should be module.action style: %s", p.Code)
		}
		if seen[p.Code] {
			t.Errorf("duplicate code: %s", p.Code)
		}
		seen[p.Code] = true
	}
}

func TestDefaultRolesReferenceKnownPermissions(t *testing.T) {
	known := map[string]bool{}
	for _, p := range AllPermissions() {
		known[p.Code] = true
	}
	roles := DefaultRoles()
	required := []string{"super_admin", "owner", "admin", "manager", "accountant", "cashier", "viewer"}
	for _, code := range required {
		if _, ok := roles[code]; !ok {
			t.Errorf("missing default role: %s", code)
		}
	}
	for roleCode, role := range roles {
		if len(role.Permissions) == 0 {
			t.Errorf("role %s has no permissions", roleCode)
		}
		for _, pc := range role.Permissions {
			if !known[pc] {
				t.Errorf("role %s references unknown permission %s", roleCode, pc)
			}
		}
	}
	for _, roleCode := range []string{"super_admin", "owner"} {
		if len(roles[roleCode].Permissions) != len(known) {
			t.Errorf("%s should have all %d permissions, got %d", roleCode, len(known), len(roles[roleCode].Permissions))
		}
		for code := range known {
			found := false
			for _, assigned := range roles[roleCode].Permissions {
				if assigned == code {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("%s is missing permission %s", roleCode, code)
			}
		}
	}
}

func TestConstValuesMatchCatalog(t *testing.T) {
	// Spot-check critical constants
	checks := map[string]string{
		DashboardView:          "dashboard.view",
		IncomeCreate:           "income.create",
		ExpenseVoid:            "expense.void",
		TransactionsApprove:    "transactions.approve",
		TransactionsReverse:    "transactions.reverse",
		ReportsExport:          "reports.export",
		BranchesManage:         "branches.manage",
		UsersManage:            "users.manage",
		PermissionsManage:      "permissions.manage",
		CurrencySettingsManage: "currency-settings.manage",
		AuditLogsView:          "audit-logs.view",
		SettingsManage:         "settings.manage",
	}
	for got, want := range checks {
		if got != want {
			t.Errorf("const mismatch: got %q want %q", got, want)
		}
	}
}


func TestCodeListMatchesAllPermissions(t *testing.T) {
	if len(CodeList) != Count {
		t.Fatalf("CodeList length %d != Count %d", len(CodeList), Count)
	}
	fromAll := Codes()
	if len(fromAll) != len(CodeList) {
		t.Fatalf("Codes() %d vs CodeList %d", len(fromAll), len(CodeList))
	}
	set := map[string]bool{}
	for _, c := range CodeList {
		set[c] = true
		if !IsValid(c) {
			t.Errorf("IsValid(%q) = false", c)
		}
	}
	if IsValid("not.a.real.permission") {
		t.Error("unknown code should be invalid")
	}
	for _, c := range fromAll {
		if !set[c] {
			t.Errorf("Codes() has %q missing from CodeList", c)
		}
	}
}
