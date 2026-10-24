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
