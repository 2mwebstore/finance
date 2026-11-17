package services

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/cambodia-finance/income-expense/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	ErrPaymentMethodNotFound  = errors.New("payment method not found")
	ErrPaymentMethodCodeTaken = errors.New("payment method code already exists")
)

// PaymentMethodService: payment methods for transactions, sales, purchases and expenses. Each is
// shared by every branch (built-in ones always are) or one branch's own (see catalog_owner.go).
type PaymentMethodService struct {
	db *gorm.DB
}

func NewPaymentMethodService(db *gorm.DB) *PaymentMethodService {
	return &PaymentMethodService{db: db}
}

type CreatePaymentMethodInput struct {
	Code string `json:"code" binding:"required,min=2,max=64"`
	Name string `json:"name" binding:"required,min=2,max=128"`
	// BranchID: "" shared (owner/admin only) or one branch.
	BranchID string `json:"branch_id"`
}

type UpdatePaymentMethodInput struct {
	Name     *string `json:"name" binding:"omitempty,min=2,max=128"`
	IsActive *bool   `json:"is_active"`
	// BranchID: "" shared or one branch; left out keeps it.
	BranchID *string `json:"branch_id"`
}

// PaymentMethodView is a payment method with its branch's name.
type PaymentMethodView struct {
	models.PaymentMethod
	BranchName string `json:"branch_name,omitempty"`
}

func (s *PaymentMethodService) view(ctx context.Context, pm models.PaymentMethod) *PaymentMethodView {
	v := &PaymentMethodView{PaymentMethod: pm}
	if pm.BranchID != nil {
		v.BranchName = BranchName(ctx, s.db, *pm.BranchID)
	}
	return v
}

func (s *PaymentMethodService) Create(ctx context.Context, in CreatePaymentMethodInput, scope CatalogScope) (*PaymentMethodView, error) {
	owner, err := ParseOwner(ctx, s.db, in.BranchID, scope)
	if err != nil {
		return nil, err
	}
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
		BranchID: owner,
	}
	if err := s.db.WithContext(ctx).Create(&pm).Error; err != nil {
		return nil, err
	}
	return s.view(ctx, pm), nil
}

func (s *PaymentMethodService) Update(ctx context.Context, id uuid.UUID, in UpdatePaymentMethodInput, scope CatalogScope) (*PaymentMethodView, error) {
	var pm models.PaymentMethod
	if err := s.db.WithContext(ctx).First(&pm, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPaymentMethodNotFound
		}
		return nil, err
	}
	if err := MayChange(pm.BranchID, scope); err != nil {
		if errors.Is(err, ErrInventoryNotFound) {
			return nil, ErrPaymentMethodNotFound
		}
		return nil, err
	}
	updates := map[string]interface{}{}
	if in.Name != nil {
		updates["name"] = strings.TrimSpace(*in.Name)
	}
	if in.BranchID != nil {
		owner, err := ParseOwner(ctx, s.db, *in.BranchID, scope)
		if err != nil {
			return nil, err
		}
		if !sameOwner(owner, pm.BranchID) {
			if owner != nil {
				if pm.IsSystem {
					return nil, fieldError("branch_id", pm.Name+" is built in, so every branch shares it")
				}
				used, err := countAny(s.db.WithContext(ctx), []string{
					"SELECT COUNT(*) FROM transactions WHERE payment_method_id = ? AND branch_id <> ?",
					"SELECT COUNT(*) FROM sales WHERE payment_method_id = ? AND branch_id <> ?",
					"SELECT COUNT(*) FROM purchases WHERE payment_method_id = ? AND branch_id <> ?",
					"SELECT COUNT(*) FROM inventory_expenses WHERE payment_method_id = ? AND branch_id <> ?",
				}, pm.ID, *owner)
				if err != nil {
					return nil, err
				}
				if used {
					return nil, fieldError("branch_id", fmt.Sprintf("%s is on another branch's records, so it can't be %s's own", pm.Name, BranchName(ctx, s.db, *owner)))
				}
				if err := s.keepOneShared(ctx, pm); err != nil {
					return nil, err
				}
			}
			updates["branch_id"] = owner
		}
	}
	if in.IsActive != nil {
		// Forms default to Cash, else the first method that is on: keep a shared one on, so every
		// branch has one.
		if !*in.IsActive && pm.IsActive {
			if err := s.keepOneShared(ctx, pm); err != nil {
				return nil, err
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
	return s.view(ctx, pm), nil
}

// keepOneShared refuses to take the last shared method that is on away from every branch.
func (s *PaymentMethodService) keepOneShared(ctx context.Context, pm models.PaymentMethod) error {
	if pm.BranchID != nil || !pm.IsActive {
		return nil
	}
	var others int64
	if err := s.db.WithContext(ctx).Model(&models.PaymentMethod{}).
		Where("is_active = ? AND branch_id IS NULL AND id <> ?", true, pm.ID).Count(&others).Error; err != nil {
		return err
	}
	if others == 0 {
		return fieldError("is_active", "keep at least one shared payment method turned on")
	}
	return nil
}

// List: shared payment methods and those of the scope's branches (see CategoryListScope).
func (s *PaymentMethodService) List(ctx context.Context, activeOnly bool, scope CategoryListScope) ([]PaymentMethodView, error) {
	q := s.db.WithContext(ctx).Model(&models.PaymentMethod{})
	switch {
	case scope.ForBranch != nil:
		q = q.Where("(branch_id IS NULL OR branch_id = ?)", *scope.ForBranch)
	case !scope.All && len(scope.BranchIDs) > 0:
		q = q.Where("(branch_id IS NULL OR branch_id IN ?)", scope.BranchIDs)
	case !scope.All:
		q = q.Where("branch_id IS NULL")
	}
	if activeOnly {
		q = q.Where("is_active = ?", true)
	}
	var items []models.PaymentMethod
	if err := q.Order("name ASC").Find(&items).Error; err != nil {
		return nil, err
	}
	names := BranchNames(ctx, s.db)
	out := make([]PaymentMethodView, len(items))
	for i, pm := range items {
		out[i] = PaymentMethodView{PaymentMethod: pm, BranchName: ownerName(names, pm.BranchID)}
	}
	return out, nil
}
