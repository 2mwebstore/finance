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
