package services

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/cambodia-finance/income-expense/internal/accounting"
	"github.com/cambodia-finance/income-expense/internal/models"
	"github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// PlatformService: platforms, their assignment to HQs, weekly win/lose, weekly settlements and
// the win/lose reports. An HQ is a branch, so HQ access is the branch scope: people work only in
// their own branches (owner/admin in every branch).
type PlatformService struct {
	db  *gorm.DB
	loc *time.Location
}

func NewPlatformService(db *gorm.DB, timezone string) *PlatformService {
	return &PlatformService{db: db, loc: accounting.LoadLocation(timezone)}
}

var (
	ErrPlatformNotFound   = errors.New("platform not found")
	ErrAssignmentNotFound = errors.New("assignment not found")
	ErrSettlementNotFound = errors.New("settlement not found")
)

var (
	platformCode = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.-]{0,63}$`)
	hundred      = decimal.NewFromInt(100)
)

// --- shared helpers ---

// A day is kept as midday UTC: written to a DATE column it stays the same day whatever time
// zone the database connection uses. Days are compared and queried as "2006-01-02".
func civilDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 12, 0, 0, 0, time.UTC)
}

func dayString(t time.Time) string { return t.Format("2006-01-02") }

func (s *PlatformService) today() time.Time { return civilDay(time.Now().In(s.loc)) }

// parseDay reads YYYY-MM-DD; empty is today.
func (s *PlatformService) parseDay(field, raw string) (time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return s.today(), nil
	}
	t, err := time.Parse("2006-01-02", raw)
	if err != nil {
		return time.Time{}, fieldError(field, "enter a date as YYYY-MM-DD")
	}
	return civilDay(t), nil
}

// weekStart is the Monday of t's week (weeks run Monday to Sunday).
func weekStart(t time.Time) time.Time {
	return t.AddDate(0, 0, -((int(t.Weekday()) + 6) % 7))
}

// parseRate reads a commission rate in percent: 0 to 100, at most 4 decimals.
func parseRate(field, raw string) (decimal.Decimal, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return decimal.Zero, fieldError(field, "rate is required")
	}
	r, err := decimal.NewFromString(raw)
	if err != nil || r.IsNegative() || r.GreaterThan(hundred) || r.Exponent() < -4 {
		return decimal.Zero, fieldError(field, "enter a rate from 0 to 100 (at most 4 decimals)")
	}
	return r, nil
}

// parseSignedAmount reads an amount that may be negative (a loss); empty is 0.
func parseSignedAmount(field, raw string) (decimal.Decimal, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return decimal.Zero, nil
	}
	v, err := decimal.NewFromString(raw)
	if err != nil || v.Exponent() < -4 || v.Abs().GreaterThan(accounting.MaxAmount) {
		return decimal.Zero, fieldError(field, "enter an amount (at most 4 decimals)")
	}
	return v, nil
}

func isDuplicate(err error) bool {
	var my *mysql.MySQLError
	return errors.As(err, &my) && my.Number == 1062
}

// branchesIn loads branches by id (code and name for messages and reports).
func (s *PlatformService) branchesIn(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]models.Branch, error) {
	out := map[uuid.UUID]models.Branch{}
	if len(ids) == 0 {
		return out, nil
	}
	var rows []models.Branch
	if err := s.db.WithContext(ctx).Where("id IN ?", ids).Find(&rows).Error; err != nil {
		return nil, err
	}
	for _, b := range rows {
		out[b.ID] = b
	}
	return out, nil
}

func (s *PlatformService) platformsIn(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]models.Platform, error) {
	out := map[uuid.UUID]models.Platform{}
	if len(ids) == 0 {
		return out, nil
	}
	var rows []models.Platform
	if err := s.db.WithContext(ctx).Unscoped().Where("id IN ?", ids).Find(&rows).Error; err != nil {
		return nil, err
	}
	for _, p := range rows {
		out[p.ID] = p
	}
	return out, nil
}

func (s *PlatformService) userNames(ctx context.Context, ids []uuid.UUID) map[uuid.UUID]string {
	out := map[uuid.UUID]string{}
	if len(ids) == 0 {
		return out
	}
	var users []models.User
	s.db.WithContext(ctx).Unscoped().Select("id", "full_name", "username").Where("id IN ?", ids).Find(&users)
	for _, u := range users {
		out[u.ID] = u.FullName
		if out[u.ID] == "" {
			out[u.ID] = u.Username
		}
	}
	return out
}

// ===== Platforms =====

type PlatformInput struct {
	Code            string `json:"code" binding:"required,max=64"`
	Name            string `json:"name" binding:"required,max=128"`
	DefaultRate     string `json:"default_rate"` // percent: 35 = 35%
	DefaultCurrency string `json:"default_currency" binding:"required,len=3"`
	IsActive        *bool  `json:"is_active"`
}

type PlatformRow struct {
	models.Platform
	// ActiveHQs is how many HQs have the platform now.
	ActiveHQs int64 `json:"active_hqs"`
}

func (s *PlatformService) ListPlatforms(ctx context.Context, includeInactive bool) ([]PlatformRow, error) {
	q := s.db.WithContext(ctx).Model(&models.Platform{}).
		Select("platforms.*, (SELECT COUNT(*) FROM hq_platform_assignments a WHERE a.platform_id = platforms.id AND a.active_key = 1 AND a.deleted_at IS NULL) AS active_hqs")
	if !includeInactive {
		q = q.Where("is_active = ?", true)
	}
	var rows []PlatformRow
	err := q.Order("code").Scan(&rows).Error
	return rows, err
}

func (s *PlatformService) SavePlatform(ctx context.Context, id *uuid.UUID, in PlatformInput) (*models.Platform, error) {
	var p models.Platform
	if id != nil {
		if err := s.db.WithContext(ctx).First(&p, "id = ?", *id).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, ErrPlatformNotFound
			}
			return nil, err
		}
	} else {
		p = models.Platform{Base: models.Base{ID: uuid.New()}, IsActive: true}
	}
	code := strings.TrimSpace(in.Code)
	if !platformCode.MatchString(code) {
		return nil, fieldError("code", "use letters, digits, _ . or - (e.g. 12K_YMLLJB_V1)")
	}
	var taken int64
	if err := s.db.WithContext(ctx).Unscoped().Model(&models.Platform{}).Where("code = ? AND id <> ?", code, p.ID).Count(&taken).Error; err != nil {
		return nil, err
	}
	if taken > 0 {
		return nil, fieldError("code", "another platform already has this code")
	}
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return nil, fieldError("name", "name is required")
	}
	rate, err := parseRate("default_rate", in.DefaultRate)
	if err != nil {
		return nil, err
	}
	currency := strings.ToUpper(strings.TrimSpace(in.DefaultCurrency))
	if err := validateCurrency(ctx, s.db, currency); err != nil {
		return nil, fieldError("default_currency", err.Error())
	}
	if in.IsActive != nil && !*in.IsActive && p.IsActive && id != nil {
		var active int64
		if err := s.db.WithContext(ctx).Model(&models.HQPlatformAssignment{}).Where("platform_id = ? AND active_key = 1", p.ID).Count(&active).Error; err != nil {
			return nil, err
		}
		if active > 0 {
			return nil, fieldError("is_active", fmt.Sprintf("%d HQs still have this platform; end those assignments first", active))
		}
	}
	p.Code, p.Name, p.DefaultRate, p.DefaultCurrency = code, name, rate, currency
	if in.IsActive != nil {
		p.IsActive = *in.IsActive
	}
	if err := s.db.WithContext(ctx).Save(&p).Error; err != nil {
		if isDuplicate(err) {
			return nil, fieldError("code", "another platform already has this code")
		}
		return nil, err
	}
	return &p, nil
}

// ===== HQ-platform assignments =====

// AssignmentInput gives one or more platforms to one or more HQs (every platform to every HQ).
type AssignmentInput struct {
	PlatformIDs []string `json:"platform_ids" binding:"required,min=1,max=100,dive,uuid"`
	BranchIDs   []string `json:"branch_ids" binding:"required,min=1,max=200,dive,uuid"`
	// Empty: each platform's default rate (percent) and currency.
	CommissionRate string `json:"commission_rate"`
	Currency       string `json:"currency"`
	// The first day it applies (results are entered per week); empty is today.
	StartsOn string `json:"starts_on"`
}

type AssignmentUpdateInput struct {
	CommissionRate string `json:"commission_rate" binding:"required"`
	Currency       string `json:"currency" binding:"required,len=3"`
}

type AssignmentEndInput struct {
	EndsOn string `json:"ends_on"` // the last day results can be entered; empty is today
	Reason string `json:"reason" binding:"max=255"`
}

type AssignmentMoveInput struct {
	BranchID string `json:"branch_id" binding:"required,uuid"`
	// The new HQ's first day; the old HQ's last day is the day before. Empty is today.
	On     string `json:"on"`
	Reason string `json:"reason" binding:"max=255"`
}

type AssignmentFilter struct {
	BranchID   *uuid.UUID
	PlatformID *uuid.UUID
	Status     string // active (default) | ended | all
}

type AssignmentRow struct {
	models.HQPlatformAssignment
	BranchCode   string `json:"branch_code"`
	BranchName   string `json:"branch_name"`
	PlatformCode string `json:"platform_code"`
	PlatformName string `json:"platform_name"`
	CreatedBy    string `json:"created_by"`
	EndedBy      string `json:"ended_by"`
	// Weeks is how many weekly results were entered under this assignment.
	Weeks int64 `json:"weeks"`
}

func (s *PlatformService) ListAssignments(ctx context.Context, f AssignmentFilter, scope BranchScope) ([]AssignmentRow, error) {
	q := scope.where(s.db.WithContext(ctx).Model(&models.HQPlatformAssignment{}), "branch_id")
	if f.BranchID != nil {
		if !scope.Has(*f.BranchID) {
			return nil, ErrBranchAccessDenied
		}
		q = q.Where("branch_id = ?", *f.BranchID)
	}
	if f.PlatformID != nil {
		q = q.Where("platform_id = ?", *f.PlatformID)
	}
	switch f.Status {
	case "all":
	case "ended":
		q = q.Where("is_active = ?", false)
	default:
		q = q.Where("is_active = ?", true)
	}
	var list []models.HQPlatformAssignment
	if err := q.Order("is_active DESC, starts_on DESC, created_at DESC").Limit(2000).Find(&list).Error; err != nil {
		return nil, err
	}
	return s.assignmentRows(ctx, list)
}

func (s *PlatformService) assignmentRows(ctx context.Context, list []models.HQPlatformAssignment) ([]AssignmentRow, error) {
	var branchIDs, platformIDs, userIDs, ids []uuid.UUID
	for _, a := range list {
		branchIDs, platformIDs, ids = append(branchIDs, a.BranchID), append(platformIDs, a.PlatformID), append(ids, a.ID)
		userIDs = append(userIDs, a.CreatedByID)
		if a.EndedByID != nil {
			userIDs = append(userIDs, *a.EndedByID)
		}
	}
	branches, err := s.branchesIn(ctx, branchIDs)
	if err != nil {
		return nil, err
	}
	platforms, err := s.platformsIn(ctx, platformIDs)
	if err != nil {
		return nil, err
	}
	users := s.userNames(ctx, userIDs)
	weeks := map[uuid.UUID]int64{}
	if len(ids) > 0 {
		var counts []struct {
			AssignmentID uuid.UUID
			N            int64
		}
		if err := s.db.WithContext(ctx).Model(&models.WeeklyPlatformResult{}).Select("assignment_id, COUNT(*) AS n").
			Where("assignment_id IN ?", ids).Group("assignment_id").Scan(&counts).Error; err != nil {
			return nil, err
		}
		for _, c := range counts {
			weeks[c.AssignmentID] = c.N
		}
	}
	out := make([]AssignmentRow, len(list))
	for i, a := range list {
		row := AssignmentRow{HQPlatformAssignment: a, BranchCode: branches[a.BranchID].Code, BranchName: branches[a.BranchID].Name,
			PlatformCode: platforms[a.PlatformID].Code, PlatformName: platforms[a.PlatformID].Name, CreatedBy: users[a.CreatedByID], Weeks: weeks[a.ID]}
		if a.EndedByID != nil {
			row.EndedBy = users[*a.EndedByID]
		}
		out[i] = row
	}
	return out, nil
}

// overlapping finds the HQ's other assignment of the platform that is open on or after day:
// one HQ can't have the same platform twice on a day.
func overlapping(tx *gorm.DB, branchID, platformID uuid.UUID, day time.Time, except *uuid.UUID) (*models.HQPlatformAssignment, error) {
	q := tx.Where("branch_id = ? AND platform_id = ? AND (ends_on IS NULL OR ends_on >= ?)", branchID, platformID, dayString(day))
	if except != nil {
		q = q.Where("id <> ?", *except)
	}
	var a models.HQPlatformAssignment
	err := q.Clauses(clause.Locking{Strength: "UPDATE"}).First(&a).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &a, err
}

// Assign gives each platform to each HQ (all or nothing). A rate or currency typed in applies to
// every platform; left empty, each platform's own default is used.
func (s *PlatformService) Assign(ctx context.Context, in AssignmentInput, scope BranchScope, actor uuid.UUID) ([]models.HQPlatformAssignment, error) {
	var rate *decimal.Decimal
	if strings.TrimSpace(in.CommissionRate) != "" {
		r, err := parseRate("commission_rate", in.CommissionRate)
		if err != nil {
			return nil, err
		}
		rate = &r
	}
	currency := strings.ToUpper(strings.TrimSpace(in.Currency))
	if currency != "" {
		if err := validateCurrency(ctx, s.db, currency); err != nil {
			return nil, fieldError("currency", err.Error())
		}
	}
	starts, err := s.parseDay("starts_on", in.StartsOn)
	if err != nil {
		return nil, err
	}
	platformIDs, err := uniqueIDs(in.PlatformIDs, nil)
	if err != nil {
		return nil, err
	}
	branchIDs, err := uniqueIDs(in.BranchIDs, &scope)
	if err != nil {
		return nil, err
	}
	platforms, err := s.platformsIn(ctx, platformIDs)
	if err != nil {
		return nil, err
	}
	for _, id := range platformIDs {
		p, ok := platforms[id]
		if !ok || !p.IsActive || p.DeletedAt.Valid {
			return nil, fieldError("platform_ids", "a platform was not found or is turned off")
		}
		if currency == "" {
			if err := validateCurrency(ctx, s.db, p.DefaultCurrency); err != nil {
				return nil, fieldError("currency", p.Code+": "+err.Error())
			}
		}
	}
	branches, err := s.branchesIn(ctx, branchIDs)
	if err != nil {
		return nil, err
	}
	one := int8(1)
	out := make([]models.HQPlatformAssignment, 0, len(platformIDs)*len(branchIDs))
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, branchID := range branchIDs {
			b, ok := branches[branchID]
			if !ok || !b.IsActive {
				return fieldError("branch_ids", "an HQ was not found or is turned off")
			}
			for _, platformID := range platformIDs {
				platform := platforms[platformID]
				clash, err := overlapping(tx, branchID, platformID, starts, nil)
				if err != nil {
					return err
				}
				if clash != nil {
					if clash.IsActive {
						return fieldError("branch_ids", fmt.Sprintf("%s already has %s; change or end that assignment instead", b.Code, platform.Code))
					}
					return fieldError("starts_on", fmt.Sprintf("%s had %s until %s; start the new assignment after that day", b.Code, platform.Code, dayString(*clash.EndsOn)))
				}
				a := models.HQPlatformAssignment{
					Base: models.Base{ID: uuid.New()}, BranchID: branchID, PlatformID: platformID, CommissionRate: platform.DefaultRate, Currency: platform.DefaultCurrency,
					IsActive: true, ActiveKey: &one, StartsOn: starts, CreatedByID: actor,
				}
				if rate != nil {
					a.CommissionRate = *rate
				}
				if currency != "" {
					a.Currency = currency
				}
				if err := tx.Create(&a).Error; err != nil {
					if isDuplicate(err) {
						return fieldError("branch_ids", fmt.Sprintf("%s already has %s; change or end that assignment instead", b.Code, platform.Code))
					}
					return err
				}
				out = append(out, a)
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// uniqueIDs parses ids (already checked as UUIDs by the binding), drops repeats and, with a
// scope, refuses branches outside it.
func uniqueIDs(raw []string, scope *BranchScope) ([]uuid.UUID, error) {
	seen := map[uuid.UUID]bool{}
	var out []uuid.UUID
	for _, r := range raw {
		id := uuid.MustParse(r)
		if scope != nil && !scope.Has(id) {
			return nil, ErrBranchAccessDenied
		}
		if !seen[id] {
			seen[id] = true
			out = append(out, id)
		}
	}
	return out, nil
}

func (s *PlatformService) activeAssignment(tx *gorm.DB, id uuid.UUID, scope BranchScope) (*models.HQPlatformAssignment, error) {
	var a models.HQPlatformAssignment
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&a, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrAssignmentNotFound
		}
		return nil, err
	}
	if !scope.Has(a.BranchID) {
		return nil, ErrAssignmentNotFound
	}
	if !a.IsActive {
		return nil, errors.New("this assignment has ended; assign the platform again instead")
	}
	return &a, nil
}

// UpdateAssignment changes the rate and currency of an active assignment. Days already saved keep
// the rate and currency they were worked out with.
func (s *PlatformService) UpdateAssignment(ctx context.Context, id uuid.UUID, in AssignmentUpdateInput, scope BranchScope) (before, after *models.HQPlatformAssignment, err error) {
	rate, err := parseRate("commission_rate", in.CommissionRate)
	if err != nil {
		return nil, nil, err
	}
	currency := strings.ToUpper(strings.TrimSpace(in.Currency))
	if err := validateCurrency(ctx, s.db, currency); err != nil {
		return nil, nil, fieldError("currency", err.Error())
	}
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		a, err := s.activeAssignment(tx, id, scope)
		if err != nil {
			return err
		}
		old := *a
		before = &old
		a.CommissionRate, a.Currency = rate, currency
		after = a
		return tx.Model(a).Updates(map[string]interface{}{"commission_rate": rate, "currency": currency, "updated_at": time.Now()}).Error
	})
	return before, after, err
}

// endOn closes an assignment on its last day (kept as history). Weeks starting after that day
// must have no results; an end before the first day means it never applied.
func endOn(tx *gorm.DB, a *models.HQPlatformAssignment, last time.Time, reason string, actor uuid.UUID) error {
	if last.Before(a.StartsOn.AddDate(0, 0, -1)) {
		return fieldError("ends_on", "the last day can't be before the assignment started ("+dayString(a.StartsOn)+")")
	}
	var later models.WeeklyPlatformResult
	err := tx.Where("assignment_id = ? AND week_start > ?", a.ID, dayString(last)).Order("week_start").First(&later).Error
	if err == nil {
		return fieldError("ends_on", "results are already entered for the week of "+dayString(later.WeekStart)+"; end it on or after that day")
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	return tx.Model(a).Updates(map[string]interface{}{
		"is_active": false, "active_key": nil, "ends_on": dayString(last), "ended_by_id": actor,
		"end_reason": strings.TrimSpace(reason), "updated_at": time.Now(),
	}).Error
}

// EndAssignment takes the platform off the HQ from the day after EndsOn (today at the latest).
func (s *PlatformService) EndAssignment(ctx context.Context, id uuid.UUID, in AssignmentEndInput, scope BranchScope, actor uuid.UUID) (*models.HQPlatformAssignment, error) {
	last, err := s.parseDay("ends_on", in.EndsOn)
	if err != nil {
		return nil, err
	}
	if last.After(s.today()) {
		return nil, fieldError("ends_on", "the last day can't be in the future")
	}
	var out models.HQPlatformAssignment
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		a, err := s.activeAssignment(tx, id, scope)
		if err != nil {
			return err
		}
		if err := endOn(tx, a, last, in.Reason, actor); err != nil {
			return err
		}
		return tx.First(&out, "id = ?", id).Error
	})
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// MoveAssignment ends the platform at one HQ and gives it to another from the given day, with the
// same rate and currency (one transaction; the old assignment stays as history).
func (s *PlatformService) MoveAssignment(ctx context.Context, id uuid.UUID, in AssignmentMoveInput, scope BranchScope, actor uuid.UUID) (ended, created *models.HQPlatformAssignment, err error) {
	on, err := s.parseDay("on", in.On)
	if err != nil {
		return nil, nil, err
	}
	if on.After(s.today()) {
		return nil, nil, fieldError("on", "the move date can't be in the future")
	}
	target := uuid.MustParse(in.BranchID)
	if !scope.Has(target) {
		return nil, nil, ErrBranchAccessDenied
	}
	branches, err := s.branchesIn(ctx, []uuid.UUID{target})
	if err != nil {
		return nil, nil, err
	}
	b, ok := branches[target]
	if !ok || !b.IsActive {
		return nil, nil, fieldError("branch_id", "this HQ was not found or is turned off")
	}
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		a, err := s.activeAssignment(tx, id, scope)
		if err != nil {
			return err
		}
		if a.BranchID == target {
			return fieldError("branch_id", "choose another HQ")
		}
		reason := strings.TrimSpace(in.Reason)
		if reason == "" {
			reason = "moved to " + b.Code
		}
		if err := endOn(tx, a, on.AddDate(0, 0, -1), reason, actor); err != nil {
			if fe, ok := err.(*FieldError); ok {
				return fieldError("on", fe.Message)
			}
			return err
		}
		clash, err := overlapping(tx, target, a.PlatformID, on, nil)
		if err != nil {
			return err
		}
		if clash != nil {
			return fieldError("branch_id", b.Code+" already has this platform on that day")
		}
		one := int8(1)
		next := models.HQPlatformAssignment{
			Base: models.Base{ID: uuid.New()}, BranchID: target, PlatformID: a.PlatformID, CommissionRate: a.CommissionRate,
			Currency: a.Currency, IsActive: true, ActiveKey: &one, StartsOn: on, CreatedByID: actor,
		}
		if err := tx.Create(&next).Error; err != nil {
			if isDuplicate(err) {
				return fieldError("branch_id", b.Code+" already has this platform")
			}
			return err
		}
		var old models.HQPlatformAssignment
		if err := tx.First(&old, "id = ?", a.ID).Error; err != nil {
			return err
		}
		ended, created = &old, &next
		return nil
	})
	return ended, created, err
}
