package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// Platforms and weekly win/lose. An HQ is a branch: each branch can have many platforms and each
// platform many branches (hq_platform_assignments). Tables are created from SQL (migration
// 000013 / database.EnsurePlatformTables), not by AutoMigrate, so these structs must match it.
// Rates are percents (35 = 35%); amounts are in the assignment's currency.

// Platform is an outside platform (LCMMJB, 12K_YMLLJB_V1…) whose weekly results HQs report.
type Platform struct {
	Base
	Code            string          `gorm:"size:64;not null" json:"code"`
	Name            string          `gorm:"size:128;not null" json:"name"`
	DefaultRate     decimal.Decimal `gorm:"type:decimal(7,4);not null" json:"default_rate"`
	DefaultCurrency string          `gorm:"size:3;not null" json:"default_currency"`
	IsActive        bool            `gorm:"not null" json:"is_active"`
}

// Partner is someone with a share of branch-platform assignments. Partners are managed in the app
// (Platforms → Partners); DefaultRate fills in their % when they're given an assignment.
type Partner struct {
	Base
	Name        string          `gorm:"size:128;not null" json:"name"`
	Phone       string          `gorm:"size:32;not null" json:"phone"`
	Note        string          `gorm:"size:255;not null" json:"note"`
	DefaultRate decimal.Decimal `gorm:"type:decimal(7,4);not null" json:"default_rate"`
	IsActive    bool            `gorm:"not null" json:"is_active"`
}

// HQPlatformAssignment gives an HQ a platform with its own rate and currency. Ending or moving it
// keeps the row (EndsOn set) so the history stays; ActiveKey is 1 while active and NULL after, and
// a unique key on (branch, platform, active_key) allows one active assignment per pair.
type HQPlatformAssignment struct {
	Base
	BranchID   uuid.UUID `gorm:"type:char(36);not null" json:"branch_id"`
	PlatformID uuid.UUID `gorm:"type:char(36);not null" json:"platform_id"`
	// GroupID is the titled group the assignment belongs to (nil: made before groups, now ended).
	GroupID *uuid.UUID `gorm:"type:char(36)" json:"group_id"`
	// CommissionRate (shown as "Partner %") is used when the assignment has no partners; with
	// partners, each has their own % (AssignmentPartner).
	CommissionRate decimal.Decimal `gorm:"type:decimal(7,4);not null" json:"commission_rate"`
	Currency       string          `gorm:"size:3;not null" json:"currency"`
	IsActive       bool            `gorm:"not null" json:"is_active"`
	ActiveKey      *int8           `json:"-"`
	// StartsOn and EndsOn are the first and last day results can be entered (EndsOn nil: open).
	StartsOn    time.Time  `gorm:"type:date;not null" json:"starts_on"`
	EndsOn      *time.Time `gorm:"type:date" json:"ends_on"`
	CreatedByID uuid.UUID  `gorm:"type:char(36);not null" json:"created_by_id"`
	EndedByID   *uuid.UUID `gorm:"type:char(36)" json:"ended_by_id,omitempty"`
	EndReason   string     `gorm:"size:255" json:"end_reason"`
}

func (HQPlatformAssignment) TableName() string { return "hq_platform_assignments" }

// AssignmentGroup is a titled set of agents (assignments) a branch reports, with the same partners
// and currency. Turned off, its agents leave Data Entry; weeks already saved stay.
type AssignmentGroup struct {
	ID       uuid.UUID `gorm:"type:char(36);primaryKey" json:"id"`
	BranchID uuid.UUID `gorm:"type:char(36);not null" json:"-"`
	Title    string    `gorm:"size:128;not null" json:"title"`
	// Currency: "" is each agent's own default.
	Currency    string    `gorm:"size:3;not null" json:"currency"`
	StartsOn    time.Time `gorm:"type:date;not null" json:"starts_on"`
	IsActive    bool      `gorm:"not null" json:"is_active"`
	CreatedByID uuid.UUID `gorm:"type:char(36);not null" json:"-"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (AssignmentGroup) TableName() string { return "assignment_groups" }

// AssignmentPartner is one of an assignment's partners with their %.
type AssignmentPartner struct {
	ID             uuid.UUID       `gorm:"type:char(36);primaryKey" json:"-"`
	AssignmentID   uuid.UUID       `gorm:"type:char(36);not null" json:"assignment_id"`
	PartnerID      uuid.UUID       `gorm:"type:char(36);not null" json:"partner_id"`
	CommissionRate decimal.Decimal `gorm:"type:decimal(7,4);not null" json:"commission_rate"`
	CreatedAt      time.Time       `json:"-"`
	UpdatedAt      time.Time       `json:"-"`
}

func (WeeklyPlatformResult) TableName() string { return "weekly_platform_results" }

// WeeklyPartnerResult is one partner's share of a saved week: Total at their % (kept as it was
// when the week was entered). The week's own JamePS is the sum of its partners' shares.
type WeeklyPartnerResult struct {
	ID             uuid.UUID       `gorm:"type:char(36);primaryKey" json:"-"`
	ResultID       uuid.UUID       `gorm:"type:char(36);not null" json:"result_id"`
	PartnerID      uuid.UUID       `gorm:"type:char(36);not null" json:"partner_id"`
	CommissionRate decimal.Decimal `gorm:"type:decimal(7,4);not null" json:"commission_rate"`
	JamePS         decimal.Decimal `gorm:"column:jame_ps;type:decimal(19,4);not null" json:"jame_ps"`
	CreatedAt      time.Time       `json:"-"`
	UpdatedAt      time.Time       `json:"-"`
}

// WeeklyPlatformResult is one branch's week (Monday to Sunday) on one platform, entered once a
// week. The rate, currency and formula version used are kept with it, so a later change to the
// assignment or the formula leaves it as it was. JamePS is what the app shows as "Total": with
// partners it is the sum of their shares (WeeklyPartnerResult), and CommissionRate their % added up.
type WeeklyPlatformResult struct {
	Base
	WeekStart      time.Time       `gorm:"type:date;not null" json:"week_start"`
	WeekEnd        time.Time       `gorm:"type:date;not null" json:"week_end"`
	BranchID       uuid.UUID       `gorm:"type:char(36);not null" json:"branch_id"`
	PlatformID     uuid.UUID       `gorm:"type:char(36);not null" json:"platform_id"`
	AssignmentID   uuid.UUID       `gorm:"type:char(36);not null" json:"assignment_id"`
	WinLose        decimal.Decimal `gorm:"type:decimal(19,4);not null" json:"win_lose"`
	Commission     decimal.Decimal `gorm:"type:decimal(19,4);not null" json:"commission"`
	CommissionRate decimal.Decimal `gorm:"type:decimal(7,4);not null" json:"commission_rate"`
	Currency       string          `gorm:"size:3;not null" json:"currency"`
	AdjustedAmount decimal.Decimal `gorm:"type:decimal(19,4);not null" json:"adjusted_amount"`
	JamePS         decimal.Decimal `gorm:"column:jame_ps;type:decimal(19,4);not null" json:"jame_ps"`
	FormulaVersion string          `gorm:"size:16;not null" json:"formula_version"`
	CreatedByID    uuid.UUID       `gorm:"type:char(36);not null" json:"created_by_id"`
	UpdatedByID    *uuid.UUID      `gorm:"type:char(36)" json:"updated_by_id,omitempty"`
}

// Weekly settlement statuses.
const (
	SettlementFinalized = "finalized"
	SettlementReopened  = "reopened"
)

// WeeklySettlement freezes an HQ's week (Monday to Sunday): its totals per currency are kept in
// weekly_settlement_totals and its days can't be changed while it is finalized.
type WeeklySettlement struct {
	Base
	BranchID      uuid.UUID  `gorm:"type:char(36);not null" json:"branch_id"`
	WeekStart     time.Time  `gorm:"type:date;not null" json:"week_start"`
	WeekEnd       time.Time  `gorm:"type:date;not null" json:"week_end"`
	Status        string     `gorm:"size:16;not null" json:"status"`
	FinalizedByID uuid.UUID  `gorm:"type:char(36);not null" json:"finalized_by_id"`
	FinalizedAt   time.Time  `gorm:"not null" json:"finalized_at"`
	ReopenedByID  *uuid.UUID `gorm:"type:char(36)" json:"reopened_by_id,omitempty"`
	ReopenedAt    *time.Time `json:"reopened_at,omitempty"`
	ReopenReason  string     `gorm:"size:512" json:"reopen_reason"`
}

// WeeklyEntry is a group's (branch's) week as people see it on Data Entry: a title, and the week's
// results (weekly_platform_results) for the group. One per group and week.
type WeeklyEntry struct {
	ID          uuid.UUID  `gorm:"type:char(36);primaryKey" json:"id"`
	BranchID    uuid.UUID  `gorm:"type:char(36);not null" json:"branch_id"`
	WeekStart   time.Time  `gorm:"type:date;not null" json:"week_start"`
	Title       string     `gorm:"size:128;not null" json:"title"`
	CreatedByID uuid.UUID  `gorm:"type:char(36);not null" json:"created_by_id"`
	UpdatedByID *uuid.UUID `gorm:"type:char(36)" json:"updated_by_id,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

func (WeeklyEntry) TableName() string { return "weekly_entries" }

// WeeklySettlementTotal is a finalized week's totals in one currency.
type WeeklySettlementTotal struct {
	ID             uuid.UUID       `gorm:"type:char(36);primaryKey" json:"-"`
	SettlementID   uuid.UUID       `gorm:"type:char(36);not null" json:"-"`
	Currency       string          `gorm:"size:3;not null" json:"currency"`
	WinLose        decimal.Decimal `gorm:"type:decimal(19,4);not null" json:"win_lose"`
	Commission     decimal.Decimal `gorm:"type:decimal(19,4);not null" json:"commission"`
	AdjustedAmount decimal.Decimal `gorm:"type:decimal(19,4);not null" json:"adjusted_amount"`
	JamePS         decimal.Decimal `gorm:"column:jame_ps;type:decimal(19,4);not null" json:"jame_ps"`
	Entries        int             `gorm:"not null" json:"entries"`
}
