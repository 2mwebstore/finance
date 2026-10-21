package services

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/cambodia-finance/income-expense/internal/auth"
	"github.com/cambodia-finance/income-expense/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	ErrUserNotFound     = errors.New("user not found")
	ErrUsernameTaken    = errors.New("username already taken")
	ErrEmailTaken       = errors.New("email already taken")
	ErrCannotDemoteSelf = errors.New("cannot remove your own super admin status")
	ErrLastSuperAdmin   = errors.New("cannot demote or deactivate the last super admin")
	ErrSelfPrivilegeEsc = errors.New("cannot escalate your own privileges")
)

type UserService struct {
	db *gorm.DB
}

func NewUserService(db *gorm.DB) *UserService {
	return &UserService{db: db}
}

type CreateUserInput struct {
	Username        string   `json:"username" binding:"required,min=3,max=64"`
	Email           string   `json:"email" binding:"required,email"`
	Password        string   `json:"password" binding:"required,len=4,numeric"`
	FullName        string   `json:"full_name" binding:"required,min=2"`
	RoleCodes       []string `json:"role_codes"`
	BranchIDs       []string `json:"branch_ids"`
	ManagerIDs      []string `json:"manager_ids"`
	DefaultBranchID *string  `json:"default_branch_id"`
	IsActive        *bool    `json:"is_active"`
}

type UpdateUserInput struct {
	Email           *string `json:"email" binding:"omitempty,email"`
	FullName        *string `json:"full_name" binding:"omitempty,min=2"`
	DefaultBranchID *string `json:"default_branch_id"`
}

func (s *UserService) Create(ctx context.Context, in CreateUserInput, actorID uuid.UUID, actorIsSuperAdmin bool) (*models.User, error) {
	username := strings.TrimSpace(in.Username)
	email := strings.ToLower(strings.TrimSpace(in.Email))

	var count int64
	s.db.WithContext(ctx).Model(&models.User{}).Where("username = ?", username).Count(&count)
	if count > 0 {
		return nil, ErrUsernameTaken
	}
	s.db.WithContext(ctx).Model(&models.User{}).Where("email = ?", email).Count(&count)
	if count > 0 {
		return nil, ErrEmailTaken
	}

	hash, err := auth.HashPassword(in.Password)
	if err != nil {
		return nil, err
	}

	active := true
	if in.IsActive != nil {
		active = *in.IsActive
	}

	user := models.User{
		Base:         models.Base{ID: uuid.New()},
		Username:     username,
		Email:        email,
		PasswordHash: hash,
		FullName:     strings.TrimSpace(in.FullName),
		IsActive:     active,
		IsSuperAdmin: false, // only setup or explicit super_admin role assignment by super admin
	}

	if in.DefaultBranchID != nil && *in.DefaultBranchID != "" {
		bid, err := uuid.Parse(*in.DefaultBranchID)
		if err != nil {
			return nil, fmt.Errorf("invalid default_branch_id")
		}
		user.DefaultBranchID = &bid
	}

	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&user).Error; err != nil {
			return err
		}
		if len(in.RoleCodes) > 0 {
			if err := s.setRolesTx(tx, &user, in.RoleCodes, actorIsSuperAdmin); err != nil {
				return err
			}
		}
		if len(in.BranchIDs) > 0 {
			if err := s.setBranchesTx(tx, user.ID, in.BranchIDs); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return s.GetByID(ctx, user.ID)
}

func (s *UserService) setRolesTx(tx *gorm.DB, user *models.User, codes []string, actorIsSuperAdmin bool) error {
	var roles []models.Role
	if err := tx.Where("code IN ?", codes).Find(&roles).Error; err != nil {
		return err
	}
	if len(roles) != len(codes) {
		return fmt.Errorf("one or more role codes are invalid")
	}
	for _, r := range roles {
		if r.Code == "super_admin" && !actorIsSuperAdmin {
			return ErrForbidden
		}
	}
	if err := tx.Model(user).Association("Roles").Replace(roles); err != nil {
		return err
	}
	// Sync is_super_admin flag
	isSA := false
	for _, r := range roles {
		if r.Code == "super_admin" {
			isSA = true
			break
		}
	}
	return tx.Model(user).Update("is_super_admin", isSA).Error
}

func (s *UserService) setBranchesTx(tx *gorm.DB, userID uuid.UUID, branchIDStrs []string) error {
	ids := make([]uuid.UUID, 0, len(branchIDStrs))
	for _, s := range branchIDStrs {
		id, err := uuid.Parse(s)
		if err != nil {
			return fmt.Errorf("invalid branch_id: %s", s)
		}
		ids = append(ids, id)
	}
	if err := tx.Where("user_id = ?", userID).Delete(&models.UserBranch{}).Error; err != nil {
		return err
	}
	for _, bid := range ids {
		if err := tx.Create(&models.UserBranch{UserID: userID, BranchID: bid}).Error; err != nil {
			return err
		}
	}
	return nil
}

func (s *UserService) Update(ctx context.Context, id uuid.UUID, in UpdateUserInput) (*models.User, error) {
	var user models.User
	if err := s.db.WithContext(ctx).Preload("Roles").First(&user, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		updates := map[string]interface{}{}
		if in.Email != nil {
			email := strings.ToLower(strings.TrimSpace(*in.Email))
			var c int64
			tx.Model(&models.User{}).Where("email = ? AND id != ?", email, id).Count(&c)
			if c > 0 {
				return ErrEmailTaken
			}
			updates["email"] = email
		}
		if in.FullName != nil {
			updates["full_name"] = strings.TrimSpace(*in.FullName)
		}
		if in.DefaultBranchID != nil {
			if *in.DefaultBranchID == "" {
				updates["default_branch_id"] = nil
			} else {
				bid, err := uuid.Parse(*in.DefaultBranchID)
				if err != nil {
					return fmt.Errorf("invalid default_branch_id")
				}
				updates["default_branch_id"] = bid
			}
		}
		if len(updates) > 0 {
			if err := tx.Model(&user).Updates(updates).Error; err != nil {
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

func (s *UserService) SetActive(ctx context.Context, id uuid.UUID, active bool) (*models.User, error) {
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var user models.User
		if err := tx.First(&user, "id = ?", id).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrUserNotFound
			}
			return err
		}
		if user.IsSuperAdmin && !active {
			if err := s.ensureNotLastSuperAdmin(tx, id); err != nil {
				return err
			}
		}
		if err := tx.Model(&user).Update("is_active", active).Error; err != nil {
			return err
		}
		if !active {
			now := time.Now()
			return tx.Model(&models.Session{}).
				Where("user_id = ? AND is_revoked = ?", id, false).
				Updates(map[string]interface{}{"is_revoked": true, "revoked_at": now}).Error
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return s.GetByID(ctx, id)
}

func (s *UserService) SetPassword(ctx context.Context, id uuid.UUID, password string) error {
	var user models.User
	if err := s.db.WithContext(ctx).First(&user, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrUserNotFound
		}
		return err
	}
	hash, err := auth.HashPassword(password)
	if err != nil {
		return err
	}
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&user).Update("password_hash", hash).Error; err != nil {
			return err
		}
		now := time.Now()
		return tx.Model(&models.Session{}).
			Where("user_id = ? AND is_revoked = ?", id, false).
			Updates(map[string]interface{}{"is_revoked": true, "revoked_at": now}).Error
	})
}

func (s *UserService) ensureNotLastSuperAdmin(tx *gorm.DB, excludeID uuid.UUID) error {
	var count int64
	tx.Model(&models.User{}).Where("is_super_admin = ? AND is_active = ? AND id != ?", true, true, excludeID).Count(&count)
	if count == 0 {
		return ErrLastSuperAdmin
	}
	return nil
}

func (s *UserService) GetByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	var user models.User
	err := s.db.WithContext(ctx).
		Preload("Roles.Permissions").
		Preload("Branches").
		First(&user, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return &user, nil
}

type UserListFilter struct {
	Search   string
	IsActive *bool
	RoleCode string
	Page     int
	PageSize int
	// When Restricted, only users assigned to one of BranchIDs are listed (branch-level managers).
	Restricted bool
	BranchIDs  []uuid.UUID
	// BranchID narrows the list to users assigned to one branch.
	BranchID *uuid.UUID
}

func (s *UserService) List(ctx context.Context, f UserListFilter) ([]models.User, int64, error) {
	q := s.db.WithContext(ctx).Model(&models.User{})
	if f.Search != "" {
		like := "%" + f.Search + "%"
		q = q.Where("username LIKE ? OR email LIKE ? OR full_name LIKE ?", like, like, like)
	}
	if f.IsActive != nil {
		q = q.Where("is_active = ?", *f.IsActive)
	}
	if f.RoleCode != "" {
		q = q.Joins("JOIN user_roles ur ON ur.user_id = users.id").
			Joins("JOIN roles r ON r.id = ur.role_id").
			Where("r.code = ?", f.RoleCode)
	}
	if f.BranchID != nil {
		q = q.Where("users.id IN (?)", s.db.Model(&models.UserBranch{}).Select("user_id").Where("branch_id = ?", *f.BranchID))
	}
	if f.Restricted {
		if len(f.BranchIDs) == 0 {
			q = q.Where("1 = 0")
		} else {
			q = q.Where("users.id IN (?)", s.db.Model(&models.UserBranch{}).Select("user_id").Where("branch_id IN ?", f.BranchIDs))
		}
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	page := f.Page
	if page < 1 {
		page = 1
	}
	size := f.PageSize
	if size < 1 || size > 100 {
		size = 20
	}
	var users []models.User
	err := q.Preload("Roles.Permissions").Preload("Branches").
		Order("username ASC").
		Offset((page - 1) * size).Limit(size).
		Find(&users).Error
	return users, total, err
}

func (s *UserService) AssignRoles(ctx context.Context, userID uuid.UUID, roleCodes []string, actorIsSuperAdmin bool) error {
	var user models.User
	if err := s.db.WithContext(ctx).Preload("Roles").First(&user, "id = ?", userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrUserNotFound
		}
		return err
	}
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if user.IsSuperAdmin {
			willRemainSuperAdmin := false
			for _, code := range roleCodes {
				if code == "super_admin" {
					willRemainSuperAdmin = true
					break
				}
			}
			if !willRemainSuperAdmin {
				if err := s.ensureNotLastSuperAdmin(tx, userID); err != nil {
					return err
				}
			}
		}
		return s.setRolesTx(tx, &user, roleCodes, actorIsSuperAdmin)
	})
}

func (s *UserService) AssignBranches(ctx context.Context, userID uuid.UUID, branchIDs []string) error {
	var user models.User
	if err := s.db.WithContext(ctx).First(&user, "id = ?", userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrUserNotFound
		}
		return err
	}
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return s.setBranchesTx(tx, userID, branchIDs)
	})
}

// ManagerSummary is the short form of a manager shown on user rows.
type ManagerSummary struct {
	ID       uuid.UUID `json:"id"`
	Username string    `json:"username"`
	FullName string    `json:"full_name"`
}

// ManagersFor returns each user's assigned managers.
func (s *UserService) ManagersFor(ctx context.Context, userIDs []uuid.UUID) (map[uuid.UUID][]ManagerSummary, error) {
	out := map[uuid.UUID][]ManagerSummary{}
	if len(userIDs) == 0 {
		return out, nil
	}
	var rows []struct {
		UserID   uuid.UUID
		ID       uuid.UUID
		Username string
		FullName string
	}
	err := s.db.WithContext(ctx).Table("user_managers um").
		Select("um.user_id, u.id, u.username, u.full_name").
		Joins("JOIN users u ON u.id = um.manager_id AND u.deleted_at IS NULL").
		Where("um.user_id IN ?", userIDs).
		Order("u.full_name").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, r := range rows {
		out[r.UserID] = append(out[r.UserID], ManagerSummary{ID: r.ID, Username: r.Username, FullName: r.FullName})
	}
	return out, nil
}

// SetManagers replaces the managers who approve or reject userID's transactions.
func (s *UserService) SetManagers(ctx context.Context, userID uuid.UUID, managerIDs []uuid.UUID) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("user_id = ?", userID).Delete(&models.UserManager{}).Error; err != nil {
			return err
		}
		for _, m := range managerIDs {
			if err := tx.Create(&models.UserManager{UserID: userID, ManagerID: m}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
