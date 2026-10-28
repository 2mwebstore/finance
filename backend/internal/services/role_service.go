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
	// BranchID: the branch this role is for; empty = shared by every branch.
	BranchID string `json:"branch_id"`
}

type UpdateRoleInput struct {
	Name            *string  `json:"name" binding:"omitempty,min=2,max=128"`
	Description     *string  `json:"description"`
	PermissionCodes []string `json:"permission_codes"`
	BranchID        *string  `json:"branch_id"` // "" makes it shared
}

// roleBranch checks a role's branch ("" = shared).
func (s *RoleService) roleBranch(tx *gorm.DB, raw string) (*uuid.UUID, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		return nil, fieldError("branch_id", "choose a valid branch")
	}
	var b models.Branch
	if err := tx.Select("id").First(&b, "id = ?", id).Error; err != nil {
		return nil, fieldError("branch_id", "this branch was not found")
	}
	return &id, nil
}

func (s *RoleService) Create(ctx context.Context, in CreateRoleInput) (*models.Role, error) {
	code := strings.ToLower(strings.TrimSpace(in.Code))
	var existing models.Role
	if err := s.db.WithContext(ctx).Where("code = ?", code).First(&existing).Error; err == nil {
		return nil, ErrRoleCodeTaken
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	branchID, err := s.roleBranch(s.db.WithContext(ctx), in.BranchID)
	if err != nil {
		return nil, err
	}
	// Every role sees only its users' own branches (data is not shared between branches).
	role := models.Role{
		Base:        models.Base{ID: uuid.New()},
		Code:        code,
		Name:        strings.TrimSpace(in.Name),
		Description: in.Description,
		IsSystem:    false,
		DataScope:   models.DataScopeAssigned,
		BranchID:    branchID,
	}

	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&role).Error; err != nil {
			return err
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
		if in.BranchID != nil {
			if role.IsSystem && *in.BranchID != "" {
				return fieldError("branch_id", "built-in roles stay shared by every branch")
			}
			branchID, err := s.roleBranch(tx, *in.BranchID)
			if err != nil {
				return err
			}
			updates["branch_id"] = branchID
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
	err := s.db.WithContext(ctx).Preload("Permissions").First(&role, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrRoleNotFound
		}
		return nil, err
	}
	return &role, nil
}

// RoleListFilter: which roles a list shows. Without All, only shared roles and the roles of
// BranchIDs; ForBranch narrows to shared roles and that branch's roles.
type RoleListFilter struct {
	Search    string
	All       bool
	BranchIDs []uuid.UUID
	ForBranch *uuid.UUID
}

// List returns the roles; search (optional) matches the name, code or description.
func (s *RoleService) List(ctx context.Context, f RoleListFilter) ([]models.Role, error) {
	search := f.Search
	q := s.db.WithContext(ctx).Preload("Permissions")
	switch {
	case f.ForBranch != nil:
		q = q.Where("branch_id IS NULL OR branch_id = ?", *f.ForBranch)
	case !f.All && len(f.BranchIDs) > 0:
		q = q.Where("branch_id IS NULL OR branch_id IN ?", f.BranchIDs)
	case !f.All:
		q = q.Where("branch_id IS NULL")
	}
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
