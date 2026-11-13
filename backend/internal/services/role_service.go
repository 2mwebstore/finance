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
	// BranchID: the branch this role is for (the controller requires one; see branchOrDeny).
	BranchID string `json:"branch_id"`
}

type UpdateRoleInput struct {
	Name            *string  `json:"name" binding:"omitempty,min=2,max=128"`
	Description     *string  `json:"description"`
	PermissionCodes []string `json:"permission_codes"`
	BranchID        *string  `json:"branch_id"`
}

// roleBranch checks a role's branch ("" = none: only built-in roles stay without one).
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
		if err := releaseDeletedRoleCode(tx, code); err != nil {
			return err
		}
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

// SuperAdminRoleCode is the built-in Super Admin role, which role lists leave out for all but a
// Super Admin.
const SuperAdminRoleCode = "super_admin"

// RoleListFilter: which roles a list shows. All adds the roles without a branch (the built-in ones)
// to the branch roles; without it, only the roles of BranchIDs. ForBranch is one branch's own roles
// only (the branch picked in the top bar: no built-in roles). HideSuperAdmin leaves out the Super
// Admin role.
type RoleListFilter struct {
	Search         string
	All            bool
	BranchIDs      []uuid.UUID
	ForBranch      *uuid.UUID
	HideSuperAdmin bool
}

// List returns the roles; search (optional) matches the name, code or description.
func (s *RoleService) List(ctx context.Context, f RoleListFilter) ([]models.Role, error) {
	search := f.Search
	q := s.db.WithContext(ctx).Preload("Permissions")
	switch {
	case f.ForBranch != nil:
		q = q.Where("branch_id = ?", *f.ForBranch)
	case !f.All && len(f.BranchIDs) > 0:
		q = q.Where("branch_id IN ?", f.BranchIDs)
	case !f.All:
		q = q.Where("1 = 0")
	}
	if f.HideSuperAdmin {
		q = q.Where("code <> ?", SuperAdminRoleCode)
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

// Delete removes a role that nobody has from the database for good, with its permissions (not just
// hidden: nothing of it is left). A built-in role, or one someone still has, can't be deleted.
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
		return tx.Unscoped().Delete(&role).Error
	})
}

// ErrCloneSuperAdmin: the Super Admin role is the one role that can't be copied (holding it is what
// makes someone the Super Admin).
var ErrCloneSuperAdmin = errors.New("the Super Admin role can't be copied")

// CloneResult: the copies made (with their permissions), and the branches skipped because they
// already have a role of that name.
type CloneResult struct {
	Created []models.Role
	Skipped []models.Branch
}

// Clone copies a role (a built-in one or a branch's) into each branch given, as a role of that
// branch with the same name, description and permissions, ready to give its people. A branch that
// already has a role of that name is skipped. A copy's code is the role's code and the branch's name
// ("cashier-phnompenh"), made unique.
func (s *RoleService) Clone(ctx context.Context, id uuid.UUID, rawBranchIDs []string) (*CloneResult, error) {
	role, err := s.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if role.Code == SuperAdminRoleCode {
		return nil, ErrCloneSuperAdmin
	}
	var ids []uuid.UUID
	seen := map[uuid.UUID]bool{}
	for _, raw := range rawBranchIDs {
		bid, err := uuid.Parse(strings.TrimSpace(raw))
		if err != nil {
			return nil, fieldError("branch_ids", "choose a valid branch")
		}
		if !seen[bid] {
			seen[bid] = true
			ids = append(ids, bid)
		}
	}
	if len(ids) == 0 {
		return nil, fieldError("branch_ids", "choose at least one branch")
	}
	var branches []models.Branch
	if err := s.db.WithContext(ctx).Where("id IN ?", ids).Find(&branches).Error; err != nil {
		return nil, err
	}
	if len(branches) != len(ids) {
		return nil, fieldError("branch_ids", "a branch was not found")
	}
	byID := map[uuid.UUID]models.Branch{}
	for _, b := range branches {
		byID[b.ID] = b
	}

	// A copy of a copy starts from the first one's code ("cashier", not "cashier-phnompenh-kampot").
	base := role.Code
	if role.BranchID != nil {
		var own models.Branch
		if err := s.db.WithContext(ctx).Select("id", "code", "name").First(&own, "id = ?", *role.BranchID).Error; err == nil {
			base = strings.TrimSuffix(base, "-"+codePart(own))
		}
	}

	out := &CloneResult{}
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, bid := range ids {
			b := byID[bid]
			var n int64
			if err := tx.Model(&models.Role{}).Where("branch_id = ? AND LOWER(name) = ?", b.ID, strings.ToLower(role.Name)).Count(&n).Error; err != nil {
				return err
			}
			if n > 0 {
				out.Skipped = append(out.Skipped, b)
				continue
			}
			code, err := freeRoleCode(tx, base+"-"+codePart(b))
			if err != nil {
				return err
			}
			branchID := b.ID
			copyRole := models.Role{
				Base:        models.Base{ID: uuid.New()},
				Code:        code,
				Name:        role.Name,
				Description: role.Description,
				DataScope:   models.DataScopeAssigned,
				BranchID:    &branchID,
			}
			if err := tx.Create(&copyRole).Error; err != nil {
				return err
			}
			if len(role.Permissions) > 0 {
				if err := tx.Model(&copyRole).Association("Permissions").Replace(role.Permissions); err != nil {
					return err
				}
			}
			copyRole.Permissions = role.Permissions
			out.Created = append(out.Created, copyRole)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// codePart is a branch's name for a role code: small letters and digits only ("Phnom Penh" →
// "phnompenh"), else its code in small letters.
func codePart(b models.Branch) string {
	var sb strings.Builder
	for _, r := range strings.ToLower(b.Name) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			sb.WriteRune(r)
		}
	}
	if sb.Len() == 0 {
		return strings.ToLower(b.Code)
	}
	return sb.String()
}

// freeRoleCode is the code, or the code with -2, -3… when it is taken (at most 64 characters).
func freeRoleCode(tx *gorm.DB, code string) (string, error) {
	trim := func(s string, room int) string {
		if len(s) > room {
			return s[:room]
		}
		return s
	}
	try := trim(code, 64)
	for i := 2; i < 1000; i++ {
		if err := releaseDeletedRoleCode(tx, try); err != nil {
			return "", err
		}
		var n int64
		if err := tx.Model(&models.Role{}).Where("code = ?", try).Count(&n).Error; err != nil {
			return "", err
		}
		if n == 0 {
			return try, nil
		}
		suffix := fmt.Sprintf("-%d", i)
		try = trim(code, 64-len(suffix)) + suffix
	}
	return "", ErrRoleCodeTaken
}

// releaseDeletedRoleCode frees a code still held by a role deleted before Delete removed roles for
// good (those were only hidden, and the unique index on code still counts them), so it can be used
// again. The hidden role's code becomes the start of it and its id ("admin-nn~<id>"), which no live
// role can have.
func releaseDeletedRoleCode(tx *gorm.DB, code string) error {
	var gone []models.Role
	if err := tx.Unscoped().Select("id").Where("code = ? AND deleted_at IS NOT NULL", code).Find(&gone).Error; err != nil {
		return err
	}
	for _, r := range gone {
		id := r.ID.String()
		keep := code
		if room := 64 - 1 - len(id); len(keep) > room {
			keep = keep[:room]
		}
		if err := tx.Unscoped().Model(&models.Role{}).Where("id = ?", r.ID).Update("code", keep+"~"+id).Error; err != nil {
			return err
		}
	}
	return nil
}
