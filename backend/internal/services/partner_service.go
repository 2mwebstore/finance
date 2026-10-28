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

// Partners: who shares in branch-platform assignments, each at their own %. Managed with the
// platforms (platforms.manage); their default % is used when they're given an assignment.

var ErrPartnerNotFound = errors.New("partner not found")

type PartnerInput struct {
	Name        string `json:"name" binding:"required,max=128"`
	Phone       string `json:"phone" binding:"max=32"`
	Note        string `json:"note" binding:"max=255"`
	DefaultRate string `json:"default_rate"` // percent: 35 = 35%
	IsActive    *bool  `json:"is_active"`
}

type PartnerRow struct {
	models.Partner
	// ActiveAssignments is how many branch-platform assignments the partner is on now.
	ActiveAssignments int64 `json:"active_assignments"`
}

func (s *PlatformService) ListPartners(ctx context.Context, includeInactive bool) ([]PartnerRow, error) {
	q := s.db.WithContext(ctx).Model(&models.Partner{}).
		Select("partners.*, " + activeAssignmentsOf("partners.id") + " AS active_assignments")
	if !includeInactive {
		q = q.Where("is_active = ?", true)
	}
	var rows []PartnerRow
	err := q.Order("name").Scan(&rows).Error
	return rows, err
}

func (s *PlatformService) SavePartner(ctx context.Context, id *uuid.UUID, in PartnerInput) (*models.Partner, error) {
	var p models.Partner
	if id != nil {
		if err := s.db.WithContext(ctx).First(&p, "id = ?", *id).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, ErrPartnerNotFound
			}
			return nil, err
		}
	} else {
		p = models.Partner{Base: models.Base{ID: uuid.New()}, IsActive: true}
	}
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return nil, fieldError("name", "name is required")
	}
	var taken int64
	if err := s.db.WithContext(ctx).Unscoped().Model(&models.Partner{}).Where("name = ? AND id <> ?", name, p.ID).Count(&taken).Error; err != nil {
		return nil, err
	}
	if taken > 0 {
		return nil, fieldError("name", "another partner already has this name")
	}
	rate, err := parseRate("default_rate", in.DefaultRate)
	if err != nil {
		return nil, err
	}
	if in.IsActive != nil && !*in.IsActive && p.IsActive && id != nil {
		var active int64
		if err := s.db.WithContext(ctx).Raw("SELECT "+activeAssignmentsOf("?"), p.ID).Scan(&active).Error; err != nil {
			return nil, err
		}
		if active > 0 {
			return nil, fieldError("is_active", fmt.Sprintf("%d branch assignments still have this partner; take them off first", active))
		}
	}
	p.Name, p.Phone, p.Note, p.DefaultRate = name, strings.TrimSpace(in.Phone), strings.TrimSpace(in.Note), rate
	if in.IsActive != nil {
		p.IsActive = *in.IsActive
	}
	if err := s.db.WithContext(ctx).Save(&p).Error; err != nil {
		if isDuplicate(err) {
			return nil, fieldError("name", "another partner already has this name")
		}
		return nil, err
	}
	return &p, nil
}

// partnersIn loads partners by id (kept when turned off, so old weeks keep their names).
func (s *PlatformService) partnersIn(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]models.Partner, error) {
	out := map[uuid.UUID]models.Partner{}
	if len(ids) == 0 {
		return out, nil
	}
	var rows []models.Partner
	if err := s.db.WithContext(ctx).Unscoped().Where("id IN ?", ids).Find(&rows).Error; err != nil {
		return nil, err
	}
	for _, p := range rows {
		out[p.ID] = p
	}
	return out, nil
}

// activeAssignmentsOf counts the active assignments a partner is on (SQL for the partner id).
func activeAssignmentsOf(partnerID string) string {
	return "(SELECT COUNT(*) FROM assignment_partners ap JOIN hq_platform_assignments a ON a.id = ap.assignment_id " +
		"WHERE ap.partner_id = " + partnerID + " AND a.active_key = 1 AND a.deleted_at IS NULL)"
}

// activePartner loads a partner that can be given an assignment (field names the request field).
func (s *PlatformService) activePartner(ctx context.Context, field string, id uuid.UUID) (*models.Partner, error) {
	var p models.Partner
	if err := s.db.WithContext(ctx).First(&p, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fieldError(field, "a partner was not found")
		}
		return nil, err
	}
	if !p.IsActive {
		return nil, fieldError(field, p.Name+" is turned off")
	}
	return &p, nil
}
