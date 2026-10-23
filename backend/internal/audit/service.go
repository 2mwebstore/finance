package audit

import (
	"context"
	"encoding/json"
	"log"
	"strings"
	"time"

	"github.com/cambodia-finance/income-expense/internal/models"
	"github.com/cambodia-finance/income-expense/internal/textsearch"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Action constants
const (
	ActionLogin           = "auth.login"
	ActionLogout          = "auth.logout"
	ActionLoginFailed     = "auth.login_failed"
	ActionSetupAdmin      = "auth.setup_admin"
	ActionPasswordChange  = "auth.password_change"
	ActionTxnCreate       = "transaction.create"
	ActionTxnUpdate       = "transaction.update"
	ActionTxnVoid         = "transaction.void"
	ActionTxnApprove      = "transaction.approve"
	ActionTxnReject       = "transaction.reject"
	ActionUserCreate      = "user.create"
	ActionUserUpdate      = "user.update"
	ActionRoleAssign      = "user.assign_roles"
	ActionBranchAssign    = "user.assign_branches"
	ActionBranchCreate    = "branch.create"
	ActionBranchUpdate    = "branch.update"
	ActionOpeningBalance  = "cash.opening_balance"
	ActionReconciliation  = "cash.reconciliation"
	ActionReportExport    = "report.export"
	ActionSettingsChange  = "settings.change"
	ActionAuditCleanup    = "audit.cleanup"
)

type Service struct {
	db *gorm.DB
}

func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

type Event struct {
	ActorID     *uuid.UUID
	ActorEmail  string
	Action      string
	EntityType  string
	EntityID    string
	BranchID    *uuid.UUID
	IPAddress   string
	UserAgent   string
	Outcome     string // success | failure
	Details     map[string]interface{} // never include passwords/tokens
	Correlation string
}

// Record writes an immutable audit event. Failures are logged but do not fail the business operation.
func (s *Service) Record(ctx context.Context, ev Event) {
	if ev.Outcome == "" {
		ev.Outcome = "success"
	}
	details := ""
	if ev.Details != nil {
		// Strip any sensitive keys defensively
		safe := map[string]interface{}{}
		for k, v := range ev.Details {
			lk := k
			if lk == "password" || lk == "token" || lk == "access_token" || lk == "refresh_token" || lk == "password_hash" {
				continue
			}
			safe[k] = v
		}
		if b, err := json.Marshal(safe); err == nil {
			details = string(b)
		}
	}
	entry := models.AuditLog{
		ID:          uuid.New(),
		CreatedAt:   time.Now(),
		ActorID:     ev.ActorID,
		ActorEmail:  ev.ActorEmail,
		Action:      ev.Action,
		EntityType:  ev.EntityType,
		EntityID:    ev.EntityID,
		BranchID:    ev.BranchID,
		IPAddress:   ev.IPAddress,
		UserAgent:   ev.UserAgent,
		Outcome:     ev.Outcome,
		Details:     details,
		Correlation: ev.Correlation,
	}
	if err := s.db.WithContext(ctx).Create(&entry).Error; err != nil {
		log.Printf("audit write failed: %v", err)
	}
}

type ListFilter struct {
	Action     string
	EntityType string
	ActorID    string
	BranchID   string
	Outcome    string
	// Search matches the action, actor email, entity, or IP address.
	Search     string
	DateFrom   *time.Time
	DateTo     *time.Time
	Page       int
	PageSize   int
	// When Restricted, only events from BranchIDs or by users of those branches are listed.
	Restricted bool
	BranchIDs  []uuid.UUID
}

func (s *Service) List(ctx context.Context, f ListFilter) ([]models.AuditLog, int64, error) {
	q := s.db.WithContext(ctx).Model(&models.AuditLog{})
	if strings.TrimSpace(f.Search) != "" {
		like := textsearch.Contains(f.Search)
		q = q.Where("action LIKE ? OR actor_email LIKE ? OR entity_type LIKE ? OR entity_id LIKE ? OR ip_address LIKE ?", like, like, like, like, like)
	}
	if f.Action != "" {
		q = q.Where("action = ?", f.Action)
	}
	if f.EntityType != "" {
		q = q.Where("entity_type = ?", f.EntityType)
	}
	if f.ActorID != "" {
		q = q.Where("actor_id = ?", f.ActorID)
	}
	if f.BranchID != "" {
		// Events in the branch plus anything done by people assigned to it (e.g. their sign-ins).
		branchUsers := s.db.Model(&models.UserBranch{}).Select("user_id").Where("branch_id = ?", f.BranchID)
		q = q.Where("branch_id = ? OR actor_id IN (?)", f.BranchID, branchUsers)
	}
	if f.Outcome != "" {
		q = q.Where("outcome = ?", f.Outcome)
	}
	if f.DateFrom != nil {
		q = q.Where("created_at >= ?", *f.DateFrom)
	}
	if f.DateTo != nil {
		q = q.Where("created_at < ?", *f.DateTo)
	}
	if f.Restricted {
		if len(f.BranchIDs) == 0 {
			q = q.Where("1 = 0")
		} else {
			branchUsers := s.db.Model(&models.UserBranch{}).Select("user_id").Where("branch_id IN ?", f.BranchIDs)
			q = q.Where("branch_id IN ? OR actor_id IN (?)", f.BranchIDs, branchUsers)
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
		size = 50
	}
	var items []models.AuditLog
	err := q.Order("created_at DESC").Offset((page - 1) * size).Limit(size).Find(&items).Error
	return items, total, err
}

// CountBefore reports how many audit events are older than cutoff.
func (s *Service) CountBefore(ctx context.Context, cutoff time.Time) (int64, error) {
	var n int64
	err := s.db.WithContext(ctx).Model(&models.AuditLog{}).Where("created_at < ?", cutoff).Count(&n).Error
	return n, err
}

// DeleteBefore permanently removes audit events older than cutoff (retention cleanup).
func (s *Service) DeleteBefore(ctx context.Context, cutoff time.Time) (int64, error) {
	res := s.db.WithContext(ctx).Where("created_at < ?", cutoff).Delete(&models.AuditLog{})
	return res.RowsAffected, res.Error
}
