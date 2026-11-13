package services

import (
	"errors"
	"sort"
	"testing"

	"github.com/cambodia-finance/income-expense/internal/models"
)

// Cloning a role makes a role of each chosen branch with the same name, description and
// permissions; a branch that already has a role of that name is skipped; a copy of a copy keeps the
// first code; the Super Admin role can't be copied. Needs TEST_MYSQL_DSN (see
// platform_integration_test.go).
func TestRoleClone(t *testing.T) {
	f := newWLFixture(t)
	roles := NewRoleService(f.db)
	for _, code := range []string{"twl.clone.a", "twl.clone.b"} {
		p := models.Permission{Code: code, Name: code, Module: "Test"}
		if err := f.db.Where(models.Permission{Code: code}).FirstOrCreate(&p).Error; err != nil {
			t.Fatal(err)
		}
	}
	f.db.Unscoped().Where("code LIKE ?", "twl_clone%").Delete(&models.Role{})
	t.Cleanup(func() { f.db.Unscoped().Where("code LIKE ?", "twl_clone%").Delete(&models.Role{}) })

	// A role of TWL1, copied to TWL1 (skipped: it has it) and TWL2.
	src, err := roles.Create(f.ctx, CreateRoleInput{Code: "twl_clone", Name: "TWL Clone Cashier", Description: "till", PermissionCodes: []string{"twl.clone.a", "twl.clone.b"}, BranchID: f.hq[0].ID.String()})
	if err != nil {
		t.Fatal(err)
	}
	res, err := roles.Clone(f.ctx, src.ID, []string{f.hq[0].ID.String(), f.hq[1].ID.String()})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Created) != 1 || len(res.Skipped) != 1 || res.Skipped[0].ID != f.hq[0].ID {
		t.Fatalf("want one copy and TWL1 skipped, got created=%d skipped=%+v", len(res.Created), res.Skipped)
	}
	copied, err := roles.GetByID(f.ctx, res.Created[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, p := range copied.Permissions {
		got = append(got, p.Code)
	}
	sort.Strings(got)
	if copied.Name != src.Name || copied.Description != "till" || copied.IsSystem || copied.BranchID == nil || *copied.BranchID != f.hq[1].ID ||
		copied.Code != "twl_clone-testhqtwl2" || len(got) != 2 || got[0] != "twl.clone.a" || got[1] != "twl.clone.b" {
		t.Errorf("the copy: %+v permissions %v", copied, got)
	}

	// Again to TWL2: skipped. The copy into TWL3 keeps the first code.
	res, err = roles.Clone(f.ctx, src.ID, []string{f.hq[1].ID.String()})
	if err != nil || len(res.Created) != 0 || len(res.Skipped) != 1 {
		t.Errorf("TWL2 already has it: %+v %v", res, err)
	}
	res, err = roles.Clone(f.ctx, copied.ID, []string{f.hq[2].ID.String()})
	if err != nil || len(res.Created) != 1 || res.Created[0].Code != "twl_clone-testhqtwl3" {
		t.Errorf("a copy of the copy: %+v %v", res, err)
	}

	// Deleted (nobody has it): gone from the database with its permissions, so cloning again gives the
	// code back.
	if err := roles.Delete(f.ctx, copied.ID); err != nil {
		t.Fatal(err)
	}
	var left, leftPerms int64
	f.db.Unscoped().Model(&models.Role{}).Where("id = ?", copied.ID).Count(&left)
	f.db.Model(&models.RolePermission{}).Where("role_id = ?", copied.ID).Count(&leftPerms)
	if left != 0 || leftPerms != 0 {
		t.Errorf("a deleted role is removed for good: role rows %d, permission rows %d", left, leftPerms)
	}
	res, err = roles.Clone(f.ctx, src.ID, []string{f.hq[1].ID.String()})
	if err != nil || len(res.Created) != 1 || res.Created[0].Code != "twl_clone-testhqtwl2" {
		t.Fatalf("cloned again after a delete: %+v %v", res, err)
	}
	again := res.Created[0]

	// A role someone has can't be deleted.
	f.db.Unscoped().Where("username = ?", "twl_clone_holder").Delete(&models.User{})
	holder := models.User{Username: "twl_clone_holder", Email: "twl_clone_holder@example.test", PasswordHash: "x", FullName: "x", IsActive: true, Roles: []models.Role{again}}
	if err := f.db.Create(&holder).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		f.db.Exec("DELETE FROM user_roles WHERE user_id = ?", holder.ID)
		f.db.Unscoped().Delete(&holder)
	})
	if err := roles.Delete(f.ctx, again.ID); !errors.Is(err, ErrRoleAssigned) {
		t.Errorf("a role someone has: want ErrRoleAssigned, got %v", err)
	}
	f.db.Unscoped().Model(&models.Role{}).Where("id = ?", again.ID).Count(&left)
	if left != 1 {
		t.Errorf("a role someone has stays: %d", left)
	}

	// A role deleted before (only hidden) still holds its code: it is freed when the code is used again.
	f.db.Exec("DELETE FROM user_roles WHERE user_id = ?", holder.ID)
	if err := f.db.Delete(&models.Role{}, "id = ?", again.ID).Error; err != nil {
		t.Fatal(err)
	}
	if made, err := roles.Create(f.ctx, CreateRoleInput{Code: "twl_clone-testhqtwl2", Name: "Made by hand", BranchID: f.hq[1].ID.String()}); err != nil || made.Code != "twl_clone-testhqtwl2" {
		t.Errorf("a hidden role's code by hand: %+v %v", made, err)
	}
	var held int64
	f.db.Unscoped().Model(&models.Role{}).Where("code LIKE ? AND deleted_at IS NOT NULL", "twl_clone-testhqtwl2~%").Count(&held)
	if held != 1 {
		t.Errorf("the hidden role keeps a freed code: %d", held)
	}

	// A taken code gets -2.
	f.db.Unscoped().Where("code = ?", "twl_clone-testhqtwl4").Delete(&models.Role{})
	if _, err := roles.Create(f.ctx, CreateRoleInput{Code: "twl_clone-testhqtwl4", Name: "Something else", BranchID: f.hq[0].ID.String()}); err != nil {
		t.Fatal(err)
	}
	res, err = roles.Clone(f.ctx, src.ID, []string{f.hq[3].ID.String()})
	if err != nil || len(res.Created) != 1 || res.Created[0].Code != "twl_clone-testhqtwl4-2" {
		t.Errorf("a taken code: %+v %v", res, err)
	}

	// No branch, an unknown branch, the Super Admin role.
	if _, err := roles.Clone(f.ctx, src.ID, nil); err == nil {
		t.Error("no branch: want an error")
	}
	if _, err := roles.Clone(f.ctx, src.ID, []string{"00000000-0000-0000-0000-000000000009"}); err == nil {
		t.Error("an unknown branch: want an error")
	}
	super := models.Role{Code: SuperAdminRoleCode, Name: "Super Admin", IsSystem: true, DataScope: models.DataScopeAssigned}
	if err := f.db.Where(models.Role{Code: SuperAdminRoleCode}).FirstOrCreate(&super).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := roles.Clone(f.ctx, super.ID, []string{f.hq[1].ID.String()}); !errors.Is(err, ErrCloneSuperAdmin) {
		t.Errorf("the Super Admin role: want ErrCloneSuperAdmin, got %v", err)
	}
}
