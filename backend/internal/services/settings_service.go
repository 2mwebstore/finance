package services

import (
	"context"
	"errors"
	"time"

	"github.com/cambodia-finance/income-expense/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrInvalidSettingKey = errors.New("invalid setting key")

type SettingsService struct {
	db *gorm.DB
}

func NewSettingsService(db *gorm.DB) *SettingsService {
	return &SettingsService{db: db}
}

func (s *SettingsService) GetAll(ctx context.Context) (map[string]string, error) {
	var rows []models.AppSetting
	if err := s.db.WithContext(ctx).Find(&rows).Error; err != nil {
		return nil, err
	}
	out := map[string]string{
		"company_name":            "My Business",
		"timezone":                "Asia/Phnom_Penh",
		"default_currency":        "USD",
		"fiscal_year_start_month": "1",
	}
	for _, r := range rows {
		out[r.Key] = r.Value
	}
	return out, nil
}

func (s *SettingsService) Set(ctx context.Context, key, value string, actorID uuid.UUID) error {
	allowed := map[string]bool{
		"company_name": true, "timezone": true, "default_currency": true,
		"fiscal_year_start_month": true, "support_email": true,
	}
	if !allowed[key] {
		return ErrInvalidSettingKey
	}
	if key == "default_currency" && value != "USD" && value != "KHR" {
		return errors.New("currency must be USD or KHR")
	}
	row := models.AppSetting{
		Key: key, Value: value, UpdatedBy: &actorID, UpdatedAt: time.Now(),
	}
	return s.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "key"}},
		DoUpdates: clause.AssignmentColumns([]string{"value", "updated_by", "updated_at"}),
	}).Create(&row).Error
}
