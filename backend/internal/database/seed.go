package database

import (
	"fmt"
	"log"

	"github.com/cambodia-finance/income-expense/internal/models"
	"github.com/cambodia-finance/income-expense/internal/permissions"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// SeedPermissionsAndRoles creates the canonical permissions and default system roles.
func SeedPermissionsAndRoles(db *gorm.DB) error {
	permMap := make(map[string]models.Permission)

	for _, p := range permissions.AllPermissions() {
		var existing models.Permission
		err := db.Where("code = ?", p.Code).First(&existing).Error
		if err == gorm.ErrRecordNotFound {
			perm := models.Permission{
				Base:        models.Base{ID: uuid.New()},
				Code:        p.Code,
				Name:        p.Name,
				Module:      p.Module,
				Description: p.Description,
			}
			if err := db.Create(&perm).Error; err != nil {
				return fmt.Errorf("create permission %s: %w", p.Code, err)
			}
			permMap[p.Code] = perm
		} else if err != nil {
			return err
		} else {
			if err := db.Model(&existing).Updates(map[string]interface{}{
				"name": p.Name, "module": p.Module, "description": p.Description,
			}).Error; err != nil {
				return fmt.Errorf("update permission %s: %w", p.Code, err)
			}
			existing.Name = p.Name
			existing.Module = p.Module
			existing.Description = p.Description
			permMap[p.Code] = existing
		}
	}

	for code, def := range permissions.DefaultRoles() {
		var role models.Role
		err := db.Where("code = ?", code).First(&role).Error
		if err == gorm.ErrRecordNotFound {
			role = models.Role{
				Base:        models.Base{ID: uuid.New()},
				Code:        code,
				Name:        def.Name,
				Description: def.Description,
				IsSystem:    true,
				DataScope:   models.DataScopeAssigned,
			}
			if err := db.Create(&role).Error; err != nil {
				return fmt.Errorf("create role %s: %w", code, err)
			}
		} else if err != nil {
			return err
		}

		// Sync permissions for the role
		var perms []models.Permission
		for _, pc := range def.Permissions {
			if p, ok := permMap[pc]; ok {
				perms = append(perms, p)
			}
		}
		if err := db.Model(&role).Association("Permissions").Replace(perms); err != nil {
			return fmt.Errorf("assign permissions to %s: %w", code, err)
		}
	}

	log.Println("seeded permissions and default roles")
	return nil
}

// EnsureRequiredPermissions adds permissions that features depend on when they are missing, and
// grants them to the roles that get every permission (super_admin, owner). It runs on every start,
// even with AUTO_SEED off, so a new feature's page appears without a full seed. It only adds.
//
// A permission that takes over from others (permissions.view used to be part of roles.view) is
// also granted, when it is first added, to every role that has one of them, so nobody loses a page
// or an action they had before. Retired permissions are then removed.
func EnsureRequiredPermissions(db *gorm.DB) error {
	if !db.Migrator().HasTable(&models.Permission{}) {
		return nil
	}
	// code -> the permissions whose roles get it when it is first added.
	required := map[string][]string{
		permissions.BackupsManage:   nil,
		permissions.PermissionsView: {permissions.RolesView},
	}
	// Inventory: owner/admin (Manage Branches) get every inventory permission; other roles get
	// them from the Roles page (or the default roles with AUTO_SEED).
	for _, code := range []string{
		permissions.InventoryView, permissions.InventoryManage, permissions.InventoryAdjust, permissions.InventoryTransfer,
		permissions.SalesView, permissions.SalesCreate, permissions.SalesVoid,
		permissions.PurchasesView, permissions.PurchasesCreate, permissions.PurchasesApprove, permissions.PurchasesVoid,
		permissions.InventoryExpensesView, permissions.InventoryExpensesCreate, permissions.InventoryExpensesVoid,
		permissions.SalesEdit, permissions.PurchasesEdit,
		// RP: agents (platforms), group agent assignments, Data entry and the Report.
		permissions.PlatformsView, permissions.WinLoseView, permissions.WinLoseCreate, permissions.WinLoseEdit,
		permissions.WinLoseReportsView, permissions.WinLoseReportsConsolidated, permissions.WinLoseReportsRates,
	} {
		required[code] = []string{permissions.BranchesManage}
	}
	// RP's view/add/update/delete (and react) took over from the two "manage" permissions, from
	// Correct Win/Lose (deleting an entry) and from seeing agents (their groups).
	for code, from := range map[string][]string{
		permissions.PlatformsCreate:           {retiredPlatformsManage},
		permissions.PlatformsEdit:             {retiredPlatformsManage},
		permissions.PlatformsDelete:           {retiredPlatformsManage},
		permissions.PlatformAssignmentsView:   {permissions.PlatformsView, retiredAssignmentsManage},
		permissions.PlatformAssignmentsCreate: {retiredAssignmentsManage},
		permissions.PlatformAssignmentsEdit:   {retiredAssignmentsManage},
		permissions.PlatformAssignmentsDelete: {retiredAssignmentsManage},
		permissions.WinLoseDelete:             {permissions.WinLoseEdit},
		permissions.WinLoseReportsRates:       {permissions.WinLoseReportsView},
		permissions.WinLoseReact:              {permissions.WinLoseView, permissions.WinLoseCreate, permissions.WinLoseEdit, permissions.WinLoseReportsView},
	} {
		required[code] = append([]string{permissions.BranchesManage}, from...)
	}
	for _, p := range permissions.AllPermissions() {
		from, ok := required[p.Code]
		if !ok {
			continue
		}
		var perm models.Permission
		err := db.Where("code = ?", p.Code).First(&perm).Error
		if err == gorm.ErrRecordNotFound {
			perm = models.Permission{Base: models.Base{ID: uuid.New()}, Code: p.Code, Name: p.Name, Module: p.Module, Description: p.Description}
			if err := db.Create(&perm).Error; err != nil {
				return fmt.Errorf("create permission %s: %w", p.Code, err)
			}
			log.Printf("added permission %s", p.Code)
			for _, code := range from {
				if err := grantToHoldersOf(db, &perm, code); err != nil {
					return err
				}
			}
		} else if err != nil {
			return err
		} else if perm.Name != p.Name || perm.Description != p.Description {
			// The catalog's wording wins: pages and permissions are renamed in the app now and then.
			if err := db.Model(&perm).Updates(map[string]interface{}{"name": p.Name, "description": p.Description}).Error; err != nil {
				return fmt.Errorf("rename permission %s: %w", p.Code, err)
			}
		}
		for _, roleCode := range []string{"super_admin", "owner"} {
			var role models.Role
			if err := db.Preload("Permissions", "code = ?", p.Code).Where("code = ?", roleCode).First(&role).Error; err != nil {
				continue // role not created yet
			}
			if len(role.Permissions) == 0 {
				if err := db.Model(&role).Association("Permissions").Append(&perm); err != nil {
					return fmt.Errorf("grant %s to %s: %w", p.Code, roleCode, err)
				}
			}
		}
	}
	return retirePermissions(db, retiredPlatformsManage, retiredAssignmentsManage, retiredSettlementsFinalize)
}

// Retired permissions: what the first two allowed is now split into view/add/update/delete; weekly
// settlements (finalizing and reopening a week) are gone.
const (
	retiredPlatformsManage     = "platforms.manage"
	retiredAssignmentsManage   = "platform-assignments.manage"
	retiredSettlementsFinalize = "winlose-settlements.finalize"
)

// retirePermissions removes permissions no longer in the catalog, from roles too. Run after the
// permissions that took over from them were added (and given to the same roles).
func retirePermissions(db *gorm.DB, codes ...string) error {
	for _, code := range codes {
		var perm models.Permission
		if err := db.Unscoped().Where("code = ?", code).First(&perm).Error; err == gorm.ErrRecordNotFound {
			continue
		} else if err != nil {
			return err
		}
		if err := db.Exec("DELETE FROM role_permissions WHERE permission_id = ?", perm.ID).Error; err != nil {
			return fmt.Errorf("take %s off roles: %w", code, err)
		}
		if err := db.Unscoped().Delete(&perm).Error; err != nil {
			return fmt.Errorf("remove permission %s: %w", code, err)
		}
		log.Printf("removed retired permission %s", code)
	}
	return nil
}

// grantToHoldersOf gives perm to every role that has the permission with code from.
func grantToHoldersOf(db *gorm.DB, perm *models.Permission, from string) error {
	var roleIDs []uuid.UUID
	err := db.Table("role_permissions").
		Joins("JOIN permissions ON permissions.id = role_permissions.permission_id").
		Where("permissions.code = ?", from).
		Pluck("role_permissions.role_id", &roleIDs).Error
	if err != nil {
		return fmt.Errorf("find roles with %s: %w", from, err)
	}
	for _, id := range roleIDs {
		var role models.Role
		if err := db.First(&role, "id = ?", id).Error; err != nil {
			continue // a deleted role
		}
		if err := db.Model(&role).Association("Permissions").Append(perm); err != nil {
			return fmt.Errorf("grant %s to role %s: %w", perm.Code, id, err)
		}
	}
	if len(roleIDs) > 0 {
		log.Printf("granted %s to %d role(s) that have %s", perm.Code, len(roleIDs), from)
	}
	return nil
}
