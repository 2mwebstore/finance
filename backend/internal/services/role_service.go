package services

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/cambodia-finance/income-expense/internal/models"
	"github.com/cambodia-finance/income-expense/internal/textsearch"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	ErrRoleNotFound  = errors.New("role not found")
	ErrRoleCodeTaken = errors.New("role code already exists")
	ErrSystemRole    = errors.New("cannot modify or delete system role")
	ErrRoleAssigned  = errors.New("role is assigned to one or more users")
)

type RoleService struct {
	db *gorm.DB
}

func NewRoleService(db *gorm.DB) *RoleService {
	return &RoleService{db: db}
}

type CreateRoleInput struct {
	Code            string   `json:"code" binding:"required,min=2,max=64"`
	Name            string   `json:"name" binding:"required,min=2,max=128"`
	Description     string   `json:"description"`
	PermissionCodes []string `json:"permission_codes"`
	// DataScope: which branches' data the role's users see (assigned | custom | all; default assigned).
	DataScope string `json:"data_scope" binding:"omitempty,oneof=assigned custom all"`
	// BranchIDs are the branches of a custom data scope.
	BranchIDs []string `json:"branch_ids"`
}

type UpdateRoleInput struct {
	Name            *string  `json:"name" binding:"omitempty,min=2,max=128"`
	Description     *string  `json:"description"`
	PermissionCodes []string `json:"permission_codes"`
	DataScope       *string  `json:"data_scope" binding:"omitempty,oneof=assigned custom all"`
	BranchIDs       []string `json:"branch_ids"`
}

// scopeBranches checks the branches of a custom data scope (at least one, all existing).
func (s *RoleService) scopeBranches(tx *gorm.DB, scope string, raw []string) ([]models.Branch, error) {
	if scope != models.DataScopeCustom {
		return []models.Branch{}, nil
	}
	ids := make([]uuid.UUID, 0, len(raw))
	for _, r := range raw {
		id, err := uuid.Parse(r)
		if err != nil {
			return nil, fieldError("branch_ids", "choose valid branches")
		}
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		return nil, fieldError("branch_ids", "choose at least one branch for this data scope")
	}
	var branches []models.Branch
	if err := tx.Where("id IN ?", ids).Find(&branches).Error; err != nil {
		return nil, err
	}
	if len(branches) != len(ids) {
		return nil, fieldError("branch_ids", "one or more branches were not found")
	}
	return branches, nil
}

func (s *RoleService) Create(ctx context.Context, in CreateRoleInput) (*models.Role, error) {
	code := strings.ToLower(strings.TrimSpace(in.Code))
	var existing models.Role
	if err := s.db.WithContext(ctx).Where("code = ?", code).First(&existing).Error; err == nil {
		return nil, ErrRoleCodeTaken
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	scope := in.DataScope
	if scope == "" {
		scope = models.DataScopeAssigned
	}
	role := models.Role{
		Base:        models.Base{ID: uuid.New()},
		Code:        code,
		Name:        strings.TrimSpace(in.Name),
		Description: in.Description,
		IsSystem:    false,
		DataScope:   scope,
	}

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		branches, err := s.scopeBranches(tx, scope, in.BranchIDs)
		if err != nil {
			return err
		}
		if err := tx.Create(&role).Error; err != nil {
			return err
		}
		if len(branches) > 0 {
			if err := tx.Model(&role).Association("Branches").Replace(branches); err != nil {
				return err
			}
		}
		if len(in.PermissionCodes) > 0 {
			return s.setPermissionsTx(tx, &role, in.PermissionCodes)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return s.GetByID(ctx, role.ID)
}

func (s *RoleService) setPermissionsTx(tx *gorm.DB, role *models.Role, codes []string) error {
	var perms []models.Permission
	if err := tx.Where("code IN ?", codes).Find(&perms).Error; err != nil {
		return err
	}
	if len(perms) != len(codes) {
		return fmt.Errorf("one or more permission codes are invalid")
	}
	return tx.Model(role).Association("Permissions").Replace(perms)
}

func (s *RoleService) Update(ctx context.Context, id uuid.UUID, in UpdateRoleInput) (*models.Role, error) {
	var role models.Role
	if err := s.db.WithContext(ctx).First(&role, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrRoleNotFound
		}
		return nil, err
	}
	// Allow updating permissions on system roles, but not code/deletion
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		updates := map[string]interface{}{}
		if in.Name != nil {
			if role.IsSystem {
				// allow name update on system roles for display
			}
			updates["name"] = strings.TrimSpace(*in.Name)
		}
		if in.Description != nil {
			updates["description"] = *in.Description
		}
		// Data scope: a new scope, or new branches for the custom scope it already has.
		if in.DataScope != nil || in.BranchIDs != nil {
			scope := role.DataScope
			if in.DataScope != nil {
				scope = *in.DataScope
			}
			if scope == "" {
				scope = models.DataScopeAssigned
			}
			branches, err := s.scopeBranches(tx, scope, in.BranchIDs)
			if err != nil {
				return err
			}
			updates["data_scope"] = scope
			if err := tx.Model(&role).Association("Branches").Replace(branches); err != nil {
				return err
			}
		}
		if len(updates) > 0 {
			if err := tx.Model(&role).Updates(updates).Error; err != nil {
				return err
			}
		}
		if in.PermissionCodes != nil {
			if err := s.setPermissionsTx(tx, &role, in.PermissionCodes); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return s.GetByID(ctx, id)
}

func (s *RoleService) GetByID(ctx context.Context, id uuid.UUID) (*models.Role, error) {
	var role models.Role
	err := s.db.WithContext(ctx).Preload("Permissions").Preload("Branches").First(&role, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrRoleNotFound
		}
		return nil, err
	}
	return &role, nil
}

// List returns the roles; search (optional) matches the name, code or description.
func (s *RoleService) List(ctx context.Context, search string) ([]models.Role, error) {
	q := s.db.WithContext(ctx).Preload("Permissions").Preload("Branches")
	if strings.TrimSpace(search) != "" {
		like := textsearch.Contains(search)
		q = q.Where("name LIKE ? OR code LIKE ? OR description LIKE ?", like, like, like)
	}
	var roles []models.Role
	err := q.Order("code ASC").Find(&roles).Error
	return roles, err
}

func (s *RoleService) ListPermissions(ctx context.Context) ([]models.Permission, error) {
	var perms []models.Permission
	err := s.db.WithContext(ctx).Order("module ASC, code ASC").Find(&perms).Error
	return perms, err
}

func (s *RoleService) SetPermissions(ctx context.Context, roleID uuid.UUID, codes []string) (*models.Role, error) {
	var role models.Role
	if err := s.db.WithContext(ctx).First(&role, "id = ?", roleID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrRoleNotFound
		}
		return nil, err
	}
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return s.setPermissionsTx(tx, &role, codes)
	})
	if err != nil {
		return nil, err
	}
	return s.GetByID(ctx, roleID)
}

func (s *RoleService) Delete(ctx context.Context, roleID uuid.UUID) error {
	var role models.Role
	if err := s.db.WithContext(ctx).First(&role, "id = ?", roleID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrRoleNotFound
		}
		return err
	}
	if role.IsSystem {
		return ErrSystemRole
	}
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var assignments int64
		if err := tx.Model(&models.UserRole{}).Where("role_id = ?", roleID).Count(&assignments).Error; err != nil {
			return err
		}
		if assignments > 0 {
			return ErrRoleAssigned
		}
		if err := tx.Exec("DELETE FROM role_permissions WHERE role_id = ?", roleID).Error; err != nil {
			return err
		}
		return tx.Delete(&role).Error
	})
}
