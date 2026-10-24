package access

import (
	"testing"

	"github.com/cambodia-finance/income-expense/internal/models"
	"github.com/google/uuid"
)

var (
	pp = models.Branch{Base: models.Base{ID: uuid.MustParse("11111111-1111-1111-1111-111111111111")}, Code: "PP-01"}
	sr = models.Branch{Base: models.Base{ID: uuid.MustParse("22222222-2222-2222-2222-222222222222")}, Code: "SR-01"}
	bb = models.Branch{Base: models.Base{ID: uuid.MustParse("33333333-3333-3333-3333-333333333333")}, Code: "BB-01"}
)

func role(scope string, branches ...models.Branch) models.Role {
	return models.Role{DataScope: scope, Branches: branches}
}

func TestBranchScope(t *testing.T) {
	cases := []struct {
		name    string
		user    models.User
		wantAll bool
		want    []uuid.UUID
	}{
		{"assigned: own branches", models.User{Branches: []models.Branch{pp}, Roles: []models.Role{role(models.DataScopeAssigned)}}, false, []uuid.UUID{pp.ID}},
		{"empty scope counts as assigned", models.User{Branches: []models.Branch{pp}, Roles: []models.Role{role("")}}, false, []uuid.UUID{pp.ID}},
		{"custom: only the role's branches", models.User{Branches: []models.Branch{pp}, Roles: []models.Role{role(models.DataScopeCustom, sr, bb)}}, false, []uuid.UUID{sr.ID, bb.ID}},
		{"roles add up", models.User{Branches: []models.Branch{pp}, Roles: []models.Role{role(models.DataScopeAssigned), role(models.DataScopeCustom, sr, pp)}}, false, []uuid.UUID{pp.ID, sr.ID}},
		{"one role with all wins", models.User{Branches: []models.Branch{pp}, Roles: []models.Role{role(models.DataScopeAssigned), role(models.DataScopeAll)}}, true, nil},
		{"branches.manage means all", models.User{Roles: []models.Role{{Permissions: []models.Permission{{Code: "branches.manage"}}}}}, true, nil},
		{"super admin sees all", models.User{IsSuperAdmin: true}, true, nil},
		{"no roles, no branches", models.User{Branches: []models.Branch{pp}}, false, nil},
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

func TestFullAccessIsNotDataScope(t *testing.T) {
	// A role that sees every branch gives no owner/admin authority.
	user := models.User{Roles: []models.Role{role(models.DataScopeAll)}}
	if _, all := BranchScope(&user); !all {
		t.Fatal("scope all should see every branch")
	}
	if IsFullAccess(&user) {
		t.Fatal("data scope all must not grant full access")
	}
}
