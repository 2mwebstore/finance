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
	ErrBranchNotFound      = errors.New("branch not found")
	ErrBranchCodeTaken     = errors.New("branch code already exists")
	ErrForbidden           = errors.New("forbidden")
	ErrCannotDeactivateAll = errors.New("cannot perform operation")
)

type BranchService struct {
	db *gorm.DB
}

func NewBranchService(db *gorm.DB) *BranchService {
	return &BranchService{db: db}
}

type CreateBranchInput struct {
	Code      string  `json:"code" binding:"required,min=2,max=32"`
	Name      string  `json:"name" binding:"required,min=2,max=128"`
	Address   string  `json:"address"`
	Phone     string  `json:"phone"`
	Email     string  `json:"email" binding:"omitempty,email,max=255"`
	ManagerID *string `json:"manager_id"`
}

type UpdateBranchInput struct {
	Code      *string `json:"code" binding:"omitempty,min=2,max=32"`
	Name      *string `json:"name" binding:"omitempty,min=2,max=128"`
	Address   *string `json:"address"`
	Phone     *string `json:"phone"`
	Email     *string `json:"email"`
	IsActive  *bool   `json:"is_active"`
	ManagerID *string `json:"manager_id"`
}

func (s *BranchService) Create(ctx context.Context, in CreateBranchInput) (*models.Branch, error) {
	code := strings.ToUpper(strings.TrimSpace(in.Code))
	var existing models.Branch
	if err := s.db.WithContext(ctx).Where("code = ?", code).First(&existing).Error; err == nil {
		return nil, ErrBranchCodeTaken
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	branch := models.Branch{
		Base:     models.Base{ID: uuid.New()},
		Code:     code,
		Name:     strings.TrimSpace(in.Name),
		Address:  in.Address,
		Phone:    in.Phone,
		Email:    in.Email,
		IsActive: true,
	}
	if in.ManagerID != nil && *in.ManagerID != "" {
		mid, err := uuid.Parse(*in.ManagerID)
		if err != nil {
			return nil, fmt.Errorf("invalid manager_id")
		}
		branch.ManagerID = &mid
	}
	if err := s.db.WithContext(ctx).Create(&branch).Error; err != nil {
		return nil, err
	}
	return &branch, nil
}

func (s *BranchService) Update(ctx context.Context, id uuid.UUID, in UpdateBranchInput) (*models.Branch, error) {
	var branch models.Branch
	if err := s.db.WithContext(ctx).First(&branch, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrBranchNotFound
		}
		return nil, err
	}
	updates := map[string]interface{}{}
	if in.Code != nil {
		code := strings.ToUpper(strings.TrimSpace(*in.Code))
		var count int64
		if err := s.db.WithContext(ctx).Model(&models.Branch{}).Where("code = ? AND id <> ?", code, id).Count(&count).Error; err != nil {
			return nil, err
		}
		if count > 0 {
			return nil, ErrBranchCodeTaken
		}
		updates["code"] = code
	}
	if in.Name != nil {
		updates["name"] = strings.TrimSpace(*in.Name)
	}
	if in.Address != nil {
		updates["address"] = *in.Address
	}
	if in.Phone != nil {
		updates["phone"] = *in.Phone
	}
	if in.Email != nil {
		updates["email"] = *in.Email
	}
	if in.IsActive != nil {
		updates["is_active"] = *in.IsActive
	}
	if in.ManagerID != nil {
		if *in.ManagerID == "" {
			updates["manager_id"] = nil
		} else {
			mid, err := uuid.Parse(*in.ManagerID)
			if err != nil {
				return nil, fmt.Errorf("invalid manager_id")
			}
			updates["manager_id"] = mid
		}
	}
	if len(updates) > 0 {
		if err := s.db.WithContext(ctx).Model(&branch).Updates(updates).Error; err != nil {
			return nil, err
		}
	}
	if err := s.db.WithContext(ctx).First(&branch, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &branch, nil
}

// Delete archives a branch so its historical financial records remain linked.
func (s *BranchService) Delete(ctx context.Context, id uuid.UUID) (*models.Branch, error) {
	isActive := false
	return s.Update(ctx, id, UpdateBranchInput{IsActive: &isActive})
}

func (s *BranchService) GetByID(ctx context.Context, id uuid.UUID) (*models.Branch, error) {
	var branch models.Branch
	if err := s.db.WithContext(ctx).First(&branch, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrBranchNotFound
		}
		return nil, err
	}
	return &branch, nil
}

type BranchListFilter struct {
	Search   string
	IsActive *bool
	Page     int
	PageSize int
}

func (s *BranchService) List(ctx context.Context, f BranchListFilter, allowedIDs []uuid.UUID, allBranches bool) ([]models.Branch, int64, error) {
	q := s.db.WithContext(ctx).Model(&models.Branch{})
	if !allBranches {
		if len(allowedIDs) == 0 {
			return []models.Branch{}, 0, nil
		}
		q = q.Where("id IN ?", allowedIDs)
	}
	if strings.TrimSpace(f.Search) != "" {
		like := textsearch.Contains(f.Search)
		q = q.Where("code LIKE ? OR name LIKE ? OR address LIKE ? OR phone LIKE ? OR email LIKE ?", like, like, like, like, like)
	}
	if f.IsActive != nil {
		q = q.Where("is_active = ?", *f.IsActive)
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
	var branches []models.Branch
	err := q.Order("code ASC").Offset((page - 1) * size).Limit(size).Find(&branches).Error
	return branches, total, err
}

func (s *BranchService) AssignUsers(ctx context.Context, branchID uuid.UUID, userIDs []uuid.UUID) error {
	var branch models.Branch
	if err := s.db.WithContext(ctx).First(&branch, "id = ?", branchID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrBranchNotFound
		}
		return err
	}
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Replace assignments for this branch: remove all, then add requested
		if err := tx.Where("branch_id = ?", branchID).Delete(&models.UserBranch{}).Error; err != nil {
			return err
		}
		for _, uid := range userIDs {
			ub := models.UserBranch{UserID: uid, BranchID: branchID}
			if err := tx.Create(&ub).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
