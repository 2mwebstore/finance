package access

import (
	"testing"

	"github.com/cambodia-finance/income-expense/internal/models"
	"github.com/google/uuid"
)

var (
	pp = models.Branch{Base: models.Base{ID: uuid.MustParse("11111111-1111-1111-1111-111111111111")}, Code: "PP-01"}
	sr = models.Branch{Base: models.Base{ID: uuid.MustParse("22222222-2222-2222-2222-222222222222")}, Code: "SR-01"}
)

func TestBranchScope(t *testing.T) {
	cases := []struct {
		name    string
		user    models.User
		wantAll bool
		want    []uuid.UUID
	}{
		{"own branches", models.User{Branches: []models.Branch{pp, sr}, Roles: []models.Role{{DataScope: models.DataScopeAssigned}}}, false, []uuid.UUID{pp.ID, sr.ID}},
		// Older rows may still say custom/all until the next start converts them: never shared.
		{"an old custom or all scope shares nothing", models.User{Branches: []models.Branch{pp}, Roles: []models.Role{{DataScope: "custom"}, {DataScope: "all"}}}, false, []uuid.UUID{pp.ID}},
		{"branches.manage means all", models.User{Roles: []models.Role{{Permissions: []models.Permission{{Code: "branches.manage"}}}}}, true, nil},
		{"super admin sees all", models.User{IsSuperAdmin: true}, true, nil},
		{"no branches, nothing", models.User{Roles: []models.Role{{DataScope: models.DataScopeAssigned}}}, false, nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ids, all := BranchScope(&c.user)
			if all != c.wantAll || len(ids) != len(c.want) {
				t.Fatalf("got all=%v ids=%v, want all=%v ids=%v", all, ids, c.wantAll, c.want)
			}
			for i := range ids {
				if ids[i] != c.want[i] {
					t.Fatalf("ids = %v, want %v", ids, c.want)
				}
			}
		})
	}
}

// Only someone whose every branch is inactive is kept from signing in; never the owner or an admin.
// The inactive branch stays in their data scope (its data stays in the reports).
func TestBranchesInactive(t *testing.T) {
	open := models.Branch{Base: models.Base{ID: uuid.MustParse("44444444-4444-4444-4444-444444444444")}, Code: "BR-001", IsActive: true}
	off := models.Branch{Base: models.Base{ID: uuid.MustParse("33333333-3333-3333-3333-333333333333")}, Code: "KP-01"}
	admin := []models.Role{{Permissions: []models.Permission{{Code: "branches.manage"}}}}
	cases := []struct {
		name string
		user models.User
		want bool
	}{
		{"every branch inactive", models.User{Branches: []models.Branch{off}}, true},
		{"one branch still active", models.User{Branches: []models.Branch{off, open}}, false},
		{"no branch at all", models.User{}, false},
		{"an admin in an inactive branch", models.User{Branches: []models.Branch{off}, Roles: admin}, false},
		{"super admin in an inactive branch", models.User{Branches: []models.Branch{off}, IsSuperAdmin: true}, false},
	}
	for _, c := range cases {
		if got := BranchesInactive(&c.user); got != c.want {
			t.Errorf("%s: got %v, want %v", c.name, got, c.want)
		}
	}
	if ids, _ := BranchScope(&models.User{Branches: []models.Branch{off, open}}); len(ids) != 2 {
		t.Errorf("an inactive branch should stay in the data scope, got %v", ids)
	}
}
