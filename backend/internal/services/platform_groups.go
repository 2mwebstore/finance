package services

import (
	"context"
	"errors"
	"sort"
	"strings"
	"time"

	"github.com/cambodia-finance/income-expense/internal/models"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Groups (the Group Agent Assignments page): a titled set of agents a branch reports, each an
// assignment, with the same partners and currency. A group lives in the person's own branch.
// Turned off, its agents leave Data Entry (weeks already saved stay); it can be deleted only while
// no week has been entered for it.

var ErrGroupNotFound = errors.New("group not found")

// GroupAgentInput is one of a group's agents with its rate (percent: 35 = 35%): the Total is
// worked out at it.
type GroupAgentInput struct {
	PlatformID     string `json:"platform_id" binding:"required,uuid"`
	CommissionRate string `json:"commission_rate"`
}

type GroupInput struct {
	Title  string            `json:"title" binding:"max=128"`
	Agents []GroupAgentInput `json:"agents" binding:"required,min=1,max=100,dive"`
	// Empty: each agent's default.
	Currency string `json:"currency"`
	// The first day results can be entered; empty is today.
	StartsOn string `json:"starts_on"`
}

// GroupUpdateInput replaces a group's title, agents (with their rates) and currency. Partners a
// group had from before are taken off: each agent's rate is used from then on.
type GroupUpdateInput struct {
	Title  string            `json:"title" binding:"max=128"`
	Agents []GroupAgentInput `json:"agents" binding:"required,min=1,max=100,dive"`
	// Empty: each agent's default.
	Currency string `json:"currency"`
}

// agentRates reads a group's agents: each once, each with a rate from 0 to 100.
func agentRates(agents []GroupAgentInput) ([]string, map[uuid.UUID]decimal.Decimal, error) {
	ids := make([]string, 0, len(agents))
	rates := map[uuid.UUID]decimal.Decimal{}
	for _, a := range agents {
		id := uuid.MustParse(a.PlatformID)
		if _, twice := rates[id]; twice {
			return nil, nil, fieldError("agents", "an agent is listed twice")
		}
		rate, err := parseRate("agents", a.CommissionRate)
		if err != nil {
			return nil, nil, fieldError("agents", "give each agent a rate from 0 to 100 (at most 4 decimals)")
		}
		ids, rates[id] = append(ids, a.PlatformID), rate
	}
	return ids, rates, nil
}

// GroupAgent is one of a group's agents now.
type GroupAgent struct {
	AssignmentID   uuid.UUID       `json:"assignment_id"`
	PlatformID     uuid.UUID       `json:"platform_id"`
	PlatformCode   string          `json:"platform_code"`
	Currency       string          `json:"currency"`
	CommissionRate decimal.Decimal `json:"commission_rate"`
	StartsOn       string          `json:"starts_on"`
	Weeks          int64           `json:"weeks"`
}

type GroupRow struct {
	models.AssignmentGroup
	Agents []GroupAgent `json:"agents"`
	// Partners are the same on every agent (by name).
	Partners []AssignmentPartnerRow `json:"partners"`
	// Weeks is how many weekly results were entered under its agents (taken-out ones too).
	Weeks     int64  `json:"weeks"`
	CreatedBy string `json:"created_by"`
}

type GroupFilter struct {
	PlatformID *uuid.UUID
	Status     string // all (default) | active | off
	Search     string // words in the title
}

func groupTitle(raw string) (string, error) {
	title := strings.TrimSpace(raw)
	if title == "" {
		return "", fieldError("title", "title is required")
	}
	return title, nil
}

// ListGroups lists a branch's groups: switched-on first, by title.
func (s *PlatformService) ListGroups(ctx context.Context, branchID uuid.UUID, f GroupFilter, scope BranchScope) ([]GroupRow, error) {
	if !scope.Has(branchID) {
		return nil, ErrBranchAccessDenied
	}
	q := s.db.WithContext(ctx).Where("branch_id = ?", branchID)
	switch f.Status {
	case "active":
		q = q.Where("is_active = ?", true)
	case "off":
		q = q.Where("is_active = ?", false)
	}
	if search := strings.TrimSpace(f.Search); search != "" {
		q = q.Where("title LIKE ?", likePattern(search))
	}
	var list []models.AssignmentGroup
	if err := q.Order("is_active DESC, title, created_at").Find(&list).Error; err != nil {
		return nil, err
	}
	rows, err := s.groupRows(ctx, list)
	if err != nil {
		return nil, err
	}
	if f.PlatformID == nil {
		return rows, nil
	}
	out := rows[:0]
	for _, g := range rows {
		for _, a := range g.Agents {
			if a.PlatformID == *f.PlatformID {
				out = append(out, g)
				break
			}
		}
	}
	return out, nil
}

func (s *PlatformService) groupRows(ctx context.Context, list []models.AssignmentGroup) ([]GroupRow, error) {
	out := make([]GroupRow, len(list))
	if len(list) == 0 {
		return out, nil
	}
	ids, users := make([]uuid.UUID, len(list)), make([]uuid.UUID, len(list))
	for i, g := range list {
		ids[i], users[i] = g.ID, g.CreatedByID
	}
	var current []models.HQPlatformAssignment
	if err := s.db.WithContext(ctx).Where("group_id IN ? AND is_active = ?", ids, true).Find(&current).Error; err != nil {
		return nil, err
	}
	rows, err := s.assignmentRows(ctx, current)
	if err != nil {
		return nil, err
	}
	agents := map[uuid.UUID][]AssignmentRow{}
	for _, r := range rows {
		agents[*r.GroupID] = append(agents[*r.GroupID], r)
	}
	var counts []struct {
		GroupID uuid.UUID
		N       int64
	}
	if err := s.db.WithContext(ctx).Table("weekly_group_results g").Joins("JOIN weekly_platform_results r ON r.id = g.result_id").
		Select("g.group_id, COUNT(*) AS n").Where("g.group_id IN ? AND r.deleted_at IS NULL", ids).Group("g.group_id").Scan(&counts).Error; err != nil {
		return nil, err
	}
	weeks := map[uuid.UUID]int64{}
	for _, c := range counts {
		weeks[c.GroupID] = c.N
	}
	names := s.userNames(ctx, users)
	for i, g := range list {
		row := GroupRow{AssignmentGroup: g, Agents: []GroupAgent{}, Partners: []AssignmentPartnerRow{}, Weeks: weeks[g.ID], CreatedBy: names[g.CreatedByID]}
		mine := agents[g.ID]
		sort.Slice(mine, func(a, b int) bool { return agentBefore(mine[a].PlatformCode, mine[b].PlatformCode) })
		for _, a := range mine {
			row.Agents = append(row.Agents, GroupAgent{AssignmentID: a.ID, PlatformID: a.PlatformID, PlatformCode: a.PlatformCode, Currency: a.Currency,
				CommissionRate: a.CommissionRate, StartsOn: dayString(a.StartsOn), Weeks: a.Weeks})
		}
		if len(mine) > 0 {
			row.Partners = mine[0].Partners
		}
		out[i] = row
	}
	return out, nil
}

func (s *PlatformService) groupRow(ctx context.Context, id uuid.UUID) (*GroupRow, error) {
	var g models.AssignmentGroup
	if err := s.db.WithContext(ctx).First(&g, "id = ?", id).Error; err != nil {
		return nil, ErrGroupNotFound
	}
	rows, err := s.groupRows(ctx, []models.AssignmentGroup{g})
	if err != nil {
		return nil, err
	}
	return &rows[0], nil
}

// lockGroup loads a group for a change (one in scope only).
func lockGroup(tx *gorm.DB, id uuid.UUID, scope BranchScope) (*models.AssignmentGroup, error) {
	var g models.AssignmentGroup
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&g, "id = ?", id).Error; err != nil || !scope.Has(g.BranchID) {
		return nil, ErrGroupNotFound
	}
	return &g, nil
}

// CreateGroup makes a group in the person's own branch with its agents, each at its rate.
func (s *PlatformService) CreateGroup(ctx context.Context, in GroupInput, scope BranchScope, actor uuid.UUID) (*GroupRow, error) {
	title, err := groupTitle(in.Title)
	if err != nil {
		return nil, err
	}
	platformIDs, rates, err := agentRates(in.Agents)
	if err != nil {
		return nil, err
	}
	plan, err := s.planAssign(ctx, AssignmentInput{PlatformIDs: platformIDs, Currency: in.Currency, StartsOn: in.StartsOn}, scope, actor)
	if err != nil {
		return nil, err
	}
	plan.rates = rates
	g := models.AssignmentGroup{ID: uuid.New(), BranchID: plan.branchIDs[0], Title: title, Currency: plan.currency, StartsOn: plan.starts, IsActive: true, CreatedByID: actor}
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&g).Error; err != nil {
			return err
		}
		_, err := s.applyAssign(tx, plan, actor, &g.ID)
		return err
	})
	if err != nil {
		return nil, err
	}
	return s.groupRow(ctx, g.ID)
}

// GroupByID is one group (in scope) for its edit page.
func (s *PlatformService) GroupByID(ctx context.Context, id uuid.UUID, scope BranchScope) (*GroupRow, error) {
	g, err := s.groupRow(ctx, id)
	if err != nil {
		return nil, err
	}
	if !scope.Has(g.BranchID) {
		return nil, ErrGroupNotFound
	}
	return g, nil
}

// UpdateGroup changes a group's title, agents (each at its rate) and currency, and takes off any
// partners it had. An agent taken out is ended if weeks were entered for it (history), else
// removed; one put in starts today (or on the group's first day, if that is later). Weeks already
// saved keep what they were worked out with.
func (s *PlatformService) UpdateGroup(ctx context.Context, id uuid.UUID, in GroupUpdateInput, scope BranchScope, actor uuid.UUID) (before, after *GroupRow, err error) {
	title, err := groupTitle(in.Title)
	if err != nil {
		return nil, nil, err
	}
	platformIDs, rates, err := agentRates(in.Agents)
	if err != nil {
		return nil, nil, err
	}
	ids, err := uniqueIDs(platformIDs, nil)
	if err != nil {
		return nil, nil, err
	}
	if before, err = s.groupRow(ctx, id); err != nil {
		return nil, nil, err
	}
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		g, err := lockGroup(tx, id, scope)
		if err != nil {
			return err
		}
		var current []models.HQPlatformAssignment
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("group_id = ? AND is_active = ?", g.ID, true).Find(&current).Error; err != nil {
			return err
		}
		// Every agent wanted must be on; the currency is the one chosen, else each agent's default.
		plan := &assignPlan{branchIDs: []uuid.UUID{g.BranchID}, platformIDs: ids, rates: rates, starts: s.today()}
		if g.StartsOn.After(plan.starts) {
			plan.starts = g.StartsOn
		}
		if err := s.checkPlatforms(ctx, plan, in.Currency); err != nil {
			return err
		}
		have := map[uuid.UUID]bool{}
		now := time.Now()
		for i := range current {
			a := &current[i]
			have[a.PlatformID] = true
			rate, wanted := rates[a.PlatformID]
			if !wanted {
				if err := takeOut(tx, a, s.today(), actor); err != nil {
					return err
				}
				continue
			}
			currency := plan.currency
			if currency == "" {
				currency = plan.platforms[a.PlatformID].DefaultCurrency
			}
			if err := tx.Model(a).Updates(map[string]interface{}{"commission_rate": rate, "currency": currency, "updated_at": now}).Error; err != nil {
				return err
			}
			if err := tx.Where("assignment_id = ?", a.ID).Delete(&models.AssignmentPartner{}).Error; err != nil {
				return err
			}
		}
		// The agents put in start today (or on the group's first day, if later).
		var added []uuid.UUID
		for _, pid := range ids {
			if !have[pid] {
				added = append(added, pid)
			}
		}
		if len(added) > 0 {
			plan.platformIDs = added
			if plan.branches, err = s.branchesIn(ctx, plan.branchIDs); err != nil {
				return err
			}
			if _, err := s.applyAssign(tx, plan, actor, &g.ID); err != nil {
				return err
			}
		}
		return tx.Model(g).Updates(map[string]interface{}{"title": title, "currency": plan.currency, "updated_at": now}).Error
	})
	if err != nil {
		return nil, nil, err
	}
	after, err = s.groupRow(ctx, id)
	return before, after, err
}

// takeOut removes an agent from its group: ended on the given day when weeks were entered for it
// (they stay), else deleted with its partners.
func takeOut(tx *gorm.DB, a *models.HQPlatformAssignment, day time.Time, actor uuid.UUID) error {
	var weeks int64
	if err := tx.Model(&models.WeeklyGroupResult{}).Where("assignment_id = ?", a.ID).Count(&weeks).Error; err != nil {
		return err
	}
	if weeks == 0 {
		return tx.Unscoped().Delete(&models.HQPlatformAssignment{}, "id = ?", a.ID).Error
	}
	return endOn(tx, a, day, "taken out of the group", actor)
}

// SetGroupActive switches a group on or off. Off, its agents leave Data Entry (they stay in the
// group).
func (s *PlatformService) SetGroupActive(ctx context.Context, id uuid.UUID, active bool, scope BranchScope) (*GroupRow, error) {
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		g, err := lockGroup(tx, id, scope)
		if err != nil {
			return err
		}
		return tx.Model(g).Updates(map[string]interface{}{"is_active": active, "updated_at": time.Now()}).Error
	})
	if err != nil {
		return nil, err
	}
	return s.groupRow(ctx, id)
}

// DeleteGroup removes a group and its agents' assignments, only while no week has been entered
// for it; after that it can be turned off instead.
func (s *PlatformService) DeleteGroup(ctx context.Context, id uuid.UUID, scope BranchScope) (*GroupRow, error) {
	row, err := s.groupRow(ctx, id)
	if err != nil {
		return nil, err
	}
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		g, err := lockGroup(tx, id, scope)
		if err != nil {
			return err
		}
		var weeks int64
		if err := tx.Model(&models.WeeklyGroupResult{}).Where("group_id = ?", g.ID).Count(&weeks).Error; err != nil {
			return err
		}
		if weeks > 0 {
			return errors.New("weeks are already entered for this group, so it can't be deleted; turn it off instead")
		}
		// Assignments go for good (their partners with them).
		if err := tx.Unscoped().Where("group_id = ?", g.ID).Delete(&models.HQPlatformAssignment{}).Error; err != nil {
			return err
		}
		return tx.Delete(g).Error
	})
	if err != nil {
		return nil, err
	}
	return row, nil
}
