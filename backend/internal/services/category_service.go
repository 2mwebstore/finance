package services

import (
	"context"
	"errors"
	"strings"

	"github.com/cambodia-finance/income-expense/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	ErrCategoryNotFound  = errors.New("category not found")
	ErrCategoryCodeTaken = errors.New("category code already exists for this type")
)

type CategoryService struct {
	db *gorm.DB
}

func NewCategoryService(db *gorm.DB) *CategoryService {
	return &CategoryService{db: db}
}

type CreateCategoryInput struct {
	Code        string `json:"code" binding:"required,min=2,max=64"`
	Name        string `json:"name" binding:"required,min=2,max=128"`
	Type        string `json:"type" binding:"required,oneof=expense"`
	Description string `json:"description"`
	// BranchID empty = shared category. The controller decides who may create which.
	BranchID *string `json:"branch_id"`
}

type UpdateCategoryInput struct {
	Code        *string `json:"code" binding:"omitempty,min=2,max=64"`
	Name        *string `json:"name" binding:"omitempty,min=2,max=128"`
	Description *string `json:"description"`
	IsActive    *bool   `json:"is_active"`
}

func (s *CategoryService) Create(ctx context.Context, in CreateCategoryInput, branchID *uuid.UUID) (*models.Category, error) {
	code := strings.ToLower(strings.TrimSpace(in.Code))
	var existing models.Category
	if err := s.db.WithContext(ctx).Where("code = ? AND type = ?", code, in.Type).First(&existing).Error; err == nil {
		return nil, ErrCategoryCodeTaken
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	cat := models.Category{
		Base:        models.Base{ID: uuid.New()},
		Code:        code,
		Name:        strings.TrimSpace(in.Name),
		Type:        in.Type,
		Description: in.Description,
		IsActive:    true,
		IsSystem:    false,
		BranchID:    branchID,
	}
	if err := s.db.WithContext(ctx).Create(&cat).Error; err != nil {
		return nil, err
	}
	return &cat, nil
}

func (s *CategoryService) Update(ctx context.Context, id uuid.UUID, in UpdateCategoryInput) (*models.Category, error) {
	var cat models.Category
	if err := s.db.WithContext(ctx).First(&cat, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrCategoryNotFound
		}
		return nil, err
	}
	updates := map[string]interface{}{}
	if in.Code != nil {
		code := strings.ToLower(strings.TrimSpace(*in.Code))
		var count int64
		if err := s.db.WithContext(ctx).Model(&models.Category{}).
			Where("code = ? AND type = ? AND id <> ?", code, cat.Type, id).
			Count(&count).Error; err != nil {
			return nil, err
		}
		if count > 0 {
			return nil, ErrCategoryCodeTaken
		}
		updates["code"] = code
	}
	if in.Name != nil {
		updates["name"] = strings.TrimSpace(*in.Name)
	}
	if in.Description != nil {
		updates["description"] = *in.Description
	}
	if in.IsActive != nil {
		updates["is_active"] = *in.IsActive
	}
	if len(updates) > 0 {
		if err := s.db.WithContext(ctx).Model(&cat).Updates(updates).Error; err != nil {
			return nil, err
		}
	}
	_ = s.db.WithContext(ctx).First(&cat, "id = ?", id)
	return &cat, nil
}

// CategoryListScope limits which categories a user sees: shared ones plus those of
// their branches (All = every branch), or shared plus one branch with ForBranch.
type CategoryListScope struct {
	All       bool
	BranchIDs []uuid.UUID
	ForBranch *uuid.UUID
}

func (s *CategoryService) List(ctx context.Context, typ string, activeOnly bool, scope CategoryListScope) ([]models.Category, error) {
	q := s.db.WithContext(ctx).Model(&models.Category{})
	switch {
	case scope.ForBranch != nil:
		q = q.Where("branch_id IS NULL OR branch_id = ?", *scope.ForBranch)
	case !scope.All && len(scope.BranchIDs) > 0:
		q = q.Where("branch_id IS NULL OR branch_id IN ?", scope.BranchIDs)
	case !scope.All:
		q = q.Where("branch_id IS NULL")
	}
	if typ != "" {
		q = q.Where("type = ?", typ)
	}
	if activeOnly {
		q = q.Where("is_active = ?", true)
	}
	var cats []models.Category
	err := q.Order("type ASC, name ASC").Find(&cats).Error
	return cats, err
}

func (s *CategoryService) GetByID(ctx context.Context, id uuid.UUID) (*models.Category, error) {
	var cat models.Category
	if err := s.db.WithContext(ctx).First(&cat, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrCategoryNotFound
		}
		return nil, err
	}
	return &cat, nil
}
