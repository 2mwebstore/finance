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
	ErrPaymentMethodNotFound  = errors.New("payment method not found")
	ErrPaymentMethodCodeTaken = errors.New("payment method code already exists")
)

type PaymentMethodService struct {
	db *gorm.DB
}

func NewPaymentMethodService(db *gorm.DB) *PaymentMethodService {
	return &PaymentMethodService{db: db}
}

type CreatePaymentMethodInput struct {
	Code string `json:"code" binding:"required,min=2,max=64"`
	Name string `json:"name" binding:"required,min=2,max=128"`
}

type UpdatePaymentMethodInput struct {
	Name     *string `json:"name" binding:"omitempty,min=2,max=128"`
	IsActive *bool   `json:"is_active"`
}

func (s *PaymentMethodService) Create(ctx context.Context, in CreatePaymentMethodInput) (*models.PaymentMethod, error) {
	code := strings.ToLower(strings.TrimSpace(in.Code))
	var existing models.PaymentMethod
	if err := s.db.WithContext(ctx).Where("code = ?", code).First(&existing).Error; err == nil {
		return nil, ErrPaymentMethodCodeTaken
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	pm := models.PaymentMethod{
		Base:     models.Base{ID: uuid.New()},
		Code:     code,
		Name:     strings.TrimSpace(in.Name),
		IsActive: true,
		IsSystem: false,
	}
	if err := s.db.WithContext(ctx).Create(&pm).Error; err != nil {
		return nil, err
	}
	return &pm, nil
}

func (s *PaymentMethodService) Update(ctx context.Context, id uuid.UUID, in UpdatePaymentMethodInput) (*models.PaymentMethod, error) {
	var pm models.PaymentMethod
	if err := s.db.WithContext(ctx).First(&pm, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPaymentMethodNotFound
		}
		return nil, err
	}
	updates := map[string]interface{}{}
	if in.Name != nil {
		updates["name"] = strings.TrimSpace(*in.Name)
	}
	if in.IsActive != nil {
		// Forms default to Cash, else the first method that is on: keep at least one on.
		if !*in.IsActive && pm.IsActive {
			var others int64
			if err := s.db.WithContext(ctx).Model(&models.PaymentMethod{}).Where("is_active = ? AND id <> ?", true, pm.ID).Count(&others).Error; err != nil {
				return nil, err
			}
			if others == 0 {
				return nil, fieldError("is_active", "keep at least one payment method turned on")
			}
		}
		updates["is_active"] = *in.IsActive
	}
	if len(updates) > 0 {
		if err := s.db.WithContext(ctx).Model(&pm).Updates(updates).Error; err != nil {
			return nil, err
		}
	}
	_ = s.db.WithContext(ctx).First(&pm, "id = ?", id)
	return &pm, nil
}

func (s *PaymentMethodService) List(ctx context.Context, activeOnly bool) ([]models.PaymentMethod, error) {
	q := s.db.WithContext(ctx).Model(&models.PaymentMethod{})
	if activeOnly {
		q = q.Where("is_active = ?", true)
	}
	var items []models.PaymentMethod
	err := q.Order("name ASC").Find(&items).Error
	return items, err
}
