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
// A permission that takes over from another one (permissions.view used to be part of roles.view)
// is also granted, when it is first added, to every role that has the old one, so nobody loses a
// page they could open before.
func EnsureRequiredPermissions(db *gorm.DB) error {
	if !db.Migrator().HasTable(&models.Permission{}) {
		return nil
	}
	// code -> the permission whose roles get it when it is first added ("" for none).
	required := map[string]string{
		permissions.BackupsManage:   "",
		permissions.PermissionsView: permissions.RolesView,
	}
	// Inventory: owner/admin (Manage Branches) get every inventory permission; other roles get
	// them from the Roles page (or the default roles with AUTO_SEED).
	for _, code := range []string{
		permissions.InventoryView, permissions.InventoryManage, permissions.InventoryAdjust, permissions.InventoryTransfer,
		permissions.SalesView, permissions.SalesCreate, permissions.SalesVoid,
		permissions.PurchasesView, permissions.PurchasesCreate, permissions.PurchasesApprove, permissions.PurchasesVoid,
		permissions.InventoryExpensesView, permissions.InventoryExpensesCreate, permissions.InventoryExpensesVoid,
		permissions.SalesEdit, permissions.PurchasesEdit,
		// Platforms (shown as agents) and weekly win/lose.
		permissions.PlatformsView, permissions.PlatformsManage, permissions.PlatformAssignmentsManage,
		permissions.WinLoseView, permissions.WinLoseCreate, permissions.WinLoseEdit,
		permissions.WinLoseReportsView, permissions.WinLoseReportsConsolidated, permissions.WinLoseSettlementsFinalize,
	} {
		required[code] = permissions.BranchesManage
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
			if from != "" {
				if err := grantToHoldersOf(db, &perm, from); err != nil {
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
