package services

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/cambodia-finance/income-expense/internal/models"
	"github.com/cambodia-finance/income-expense/internal/winlose"
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
	// CalcType is this agent's type, PO (the default) or GI, see winlose.Kind.
	CalcType string `json:"calc_type"`
}

type GroupInput struct {
	Title  string            `json:"title" binding:"max=128"`
	Agents []GroupAgentInput `json:"agents" binding:"required,min=1,max=100,dive"`
	// Empty: each agent's default.
	Currency string `json:"currency"`
	// The first day results can be entered; empty is today.
	StartsOn string `json:"starts_on"`
	// BranchID: the branch the group is made in (one the person works in); empty: their own group.
	BranchID string `json:"branch_id" binding:"omitempty,uuid"`
}

// GroupUpdateInput replaces a group's title, agents (with their rates) and currency. Partners a
// group had from before are taken off: each agent's rate is used from then on.
type GroupUpdateInput struct {
	Title  string            `json:"title" binding:"max=128"`
	Agents []GroupAgentInput `json:"agents" binding:"required,min=1,max=100,dive"`
	// Empty: each agent's default.
	Currency string `json:"currency"`
}

// calcType reads an agent's type: PO when empty, else PO or GI.
func calcType(raw string) (string, error) {
	t := strings.ToUpper(strings.TrimSpace(raw))
	if t == "" {
		return string(winlose.PO), nil
	}
	if !winlose.ValidKind(t) {
		return "", fieldError("calc_type", "choose PO or GI")
	}
	return t, nil
}

// agentRates reads a group's agents: each once, each with a rate from 0 to 100 and a type.
func agentRates(agents []GroupAgentInput) ([]string, map[uuid.UUID]decimal.Decimal, map[uuid.UUID]string, error) {
	ids := make([]string, 0, len(agents))
	rates := map[uuid.UUID]decimal.Decimal{}
	kinds := map[uuid.UUID]string{}
	for _, a := range agents {
		id := uuid.MustParse(a.PlatformID)
		if _, twice := rates[id]; twice {
			return nil, nil, nil, fieldError("agents", "an agent is listed twice")
		}
		rate, err := parseRate("agents", a.CommissionRate)
		if err != nil {
			return nil, nil, nil, fieldError("agents", "give each agent a rate from 0 to 100 (at most 4 decimals)")
		}
		kind, err := calcType(a.CalcType)
		if err != nil {
			return nil, nil, nil, fieldError("agents", "choose PO or GI for each agent")
		}
		ids, rates[id], kinds[id] = append(ids, a.PlatformID), rate, kind
	}
	return ids, rates, kinds, nil
}

// GroupAgent is one of a group's agents now.
type GroupAgent struct {
	AssignmentID   uuid.UUID       `json:"assignment_id"`
	PlatformID     uuid.UUID       `json:"platform_id"`
	PlatformCode   string          `json:"platform_code"`
	Currency       string          `json:"currency"`
	CommissionRate decimal.Decimal `json:"commission_rate"`
	CalcType       string          `json:"calc_type"`
	StartsOn       string          `json:"starts_on"`
	Weeks          int64           `json:"weeks"`
}

type GroupRow struct {
	models.AssignmentGroup
	// BranchID and BranchCode: the branch the group is in.
	BranchID   uuid.UUID    `json:"branch_id"`
	BranchCode string       `json:"branch_code"`
	Agents     []GroupAgent `json:"agents"`
	// Partners are the same on every agent (by name).
	Partners []AssignmentPartnerRow `json:"partners"`
	// Weeks is how many weekly results were entered under its agents (taken-out ones too).
	Weeks     int64  `json:"weeks"`
	CreatedBy string `json:"created_by"`
	// CountedWeeks (on a save only): weeks saved before an agent was put in the group that count
	// in it now (see countSavedWeeks).
	CountedWeeks int `json:"counted_weeks,omitempty"`
}

type GroupFilter struct {
	PlatformID *uuid.UUID
	Status     string // all (default) | active | off
	Search     string // words in the title
	// AllBranches: the groups of every branch in the person's scope (else the one branch asked for).
	AllBranches bool
	// PageSize 0: every group; otherwise one page of that size.
	Page, PageSize int
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
	rows, _, err := s.ListGroupsPage(ctx, branchID, f, scope)
	return rows, err
}

// ListGroupsPage lists a branch's groups (one page when f.PageSize is set), with how many groups
// match in all.
func (s *PlatformService) ListGroupsPage(ctx context.Context, branchID uuid.UUID, f GroupFilter, scope BranchScope) ([]GroupRow, int64, error) {
	var q *gorm.DB
	if f.AllBranches {
		q = scope.where(s.db.WithContext(ctx), "branch_id")
	} else {
		if !scope.Has(branchID) {
			return nil, 0, ErrBranchAccessDenied
		}
		q = s.db.WithContext(ctx).Where("branch_id = ?", branchID)
	}
	switch f.Status {
	case "active":
		q = q.Where("is_active = ?", true)
	case "off":
		q = q.Where("is_active = ?", false)
	}
	if search := strings.TrimSpace(f.Search); search != "" {
		q = q.Where("title LIKE ?", likePattern(search))
	}
	// A group with the agent in it now (an agent taken out of it no longer counts).
	if f.PlatformID != nil {
		q = q.Where("id IN (SELECT group_id FROM hq_platform_assignments WHERE platform_id = ? AND is_active = ? AND deleted_at IS NULL)", *f.PlatformID, true)
	}
	var total int64
	if err := q.Session(&gorm.Session{}).Model(&models.AssignmentGroup{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	q = q.Order("is_active DESC, title, created_at")
	if f.PageSize > 0 {
		q = q.Offset((max(f.Page, 1) - 1) * f.PageSize).Limit(f.PageSize)
	}
	var list []models.AssignmentGroup
	if err := q.Find(&list).Error; err != nil {
		return nil, 0, err
	}
	rows, err := s.groupRows(ctx, list)
	if err != nil {
		return nil, 0, err
	}
	return rows, total, nil
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
	branchIDs := make([]uuid.UUID, len(list))
	for i, g := range list {
		branchIDs[i] = g.BranchID
	}
	branches, err := s.branchesIn(ctx, branchIDs)
	if err != nil {
		return nil, err
	}
	for i, g := range list {
		row := GroupRow{AssignmentGroup: g, BranchID: g.BranchID, BranchCode: branches[g.BranchID].Code, Agents: []GroupAgent{}, Partners: []AssignmentPartnerRow{}, Weeks: weeks[g.ID], CreatedBy: names[g.CreatedByID]}
		mine := agents[g.ID]
		sort.Slice(mine, func(a, b int) bool { return agentBefore(mine[a].PlatformCode, mine[b].PlatformCode) })
		for _, a := range mine {
			row.Agents = append(row.Agents, GroupAgent{AssignmentID: a.ID, PlatformID: a.PlatformID, PlatformCode: a.PlatformCode, Currency: a.Currency,
				CommissionRate: a.CommissionRate, CalcType: a.CalcType, StartsOn: dayString(a.StartsOn), Weeks: a.Weeks})
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
	platformIDs, rates, kinds, err := agentRates(in.Agents)
	if err != nil {
		return nil, err
	}
	var branchIDs []string
	if strings.TrimSpace(in.BranchID) != "" {
		branchIDs = []string{strings.TrimSpace(in.BranchID)}
	}
	plan, err := s.planAssign(ctx, AssignmentInput{PlatformIDs: platformIDs, BranchIDs: branchIDs, Currency: in.Currency, StartsOn: in.StartsOn}, scope, actor)
	if err != nil {
		return nil, err
	}
	plan.rates, plan.kinds = rates, kinds
	g := models.AssignmentGroup{ID: uuid.New(), BranchID: plan.branchIDs[0], Title: title, Currency: plan.currency, StartsOn: plan.starts, IsActive: true, CreatedByID: actor}
	counted := 0
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&g).Error; err != nil {
			return err
		}
		made, err := s.applyAssign(tx, plan, actor, &g.ID)
		if err != nil {
			return err
		}
		counted, err = countSavedWeeks(tx, made)
		return err
	})
	if err != nil {
		return nil, err
	}
	row, err := s.groupRow(ctx, g.ID)
	if err != nil {
		return nil, err
	}
	row.CountedWeeks = counted
	return row, nil
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
	platformIDs, rates, kinds, err := agentRates(in.Agents)
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
	counted := 0
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
		plan := &assignPlan{branchIDs: []uuid.UUID{g.BranchID}, platformIDs: ids, rates: rates, kinds: kinds, starts: s.today()}
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
			if err := tx.Model(a).Updates(map[string]interface{}{"commission_rate": rate, "calc_type": kinds[a.PlatformID], "currency": currency, "updated_at": now}).Error; err != nil {
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
			made, err := s.applyAssign(tx, plan, actor, &g.ID)
			if err != nil {
				return err
			}
			if counted, err = countSavedWeeks(tx, made); err != nil {
				return err
			}
		}
		return tx.Model(g).Updates(map[string]interface{}{"title": title, "currency": plan.currency, "updated_at": now}).Error
	})
	if err != nil {
		return nil, nil, err
	}
	if after, err = s.groupRow(ctx, id); err != nil {
		return nil, nil, err
	}
	after.CountedWeeks = counted
	return before, after, nil
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

// GroupChanges says what saving an agent changed in its groups (by title), for the audit log.
type GroupChanges struct {
	Added   []string `json:"added,omitempty"`
	Removed []string `json:"removed,omitempty"`
	Changed []string `json:"changed,omitempty"`
	// Weeks: weeks saved before it was put in a group that count in it now (see countSavedWeeks).
	Weeks int `json:"weeks,omitempty"`
}

// Any reports whether the agent's groups changed.
func (c *GroupChanges) Any() bool {
	return c != nil && len(c.Added)+len(c.Removed)+len(c.Changed) > 0
}

// setPlatformGroups puts an agent in exactly the groups given, as a group's page would: one taken
// out is ended if weeks were entered for it (they stay), else removed; one put in starts today (or
// on the group's first day, if later) in the group's currency (else the agent's own); one it stays
// in gets the rate and type given. Only the person's groups count: another branch's group a shared
// agent is in stays as it is.
func (s *PlatformService) setPlatformGroups(ctx context.Context, tx *gorm.DB, p *models.Platform, groups []PlatformGroupInput, scope BranchScope, actor uuid.UUID) (*GroupChanges, error) {
	type want struct {
		rate decimal.Decimal
		kind string
	}
	wanted := map[uuid.UUID]want{}
	order := make([]uuid.UUID, 0, len(groups))
	for _, g := range groups {
		id := uuid.MustParse(g.GroupID)
		if _, twice := wanted[id]; twice {
			return nil, fieldError("groups", "a group is listed twice")
		}
		rate, err := parseRate("groups", g.CommissionRate)
		if err != nil {
			return nil, fieldError("groups", "give each group a rate from 0 to 100 (at most 4 decimals)")
		}
		kind, err := calcType(g.CalcType)
		if err != nil {
			return nil, fieldError("groups", "choose PO or GI for each group")
		}
		wanted[id] = want{rate: rate, kind: kind}
		order = append(order, id)
	}
	var current []models.HQPlatformAssignment
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("platform_id = ? AND is_active = ? AND group_id IS NOT NULL", p.ID, true).
		Order("created_at").Find(&current).Error; err != nil {
		return nil, err
	}
	changes := &GroupChanges{}
	today, now := s.today(), time.Now()
	have := map[uuid.UUID]bool{}
	for i := range current {
		a := &current[i]
		if !scope.Has(a.BranchID) {
			continue
		}
		g, err := lockGroup(tx, *a.GroupID, scope)
		if err != nil {
			return nil, err
		}
		have[g.ID] = true
		w, keep := wanted[g.ID]
		if !keep {
			if err := takeOut(tx, a, today, actor); err != nil {
				return nil, err
			}
			changes.Removed = append(changes.Removed, g.Title)
			continue
		}
		if !a.CommissionRate.Equal(w.rate) || a.CalcType != w.kind {
			if err := tx.Model(a).Updates(map[string]interface{}{"commission_rate": w.rate, "calc_type": w.kind, "updated_at": now}).Error; err != nil {
				return nil, err
			}
			changes.Changed = append(changes.Changed, g.Title)
		}
	}
	for _, id := range order {
		if have[id] {
			continue
		}
		g, err := lockGroup(tx, id, scope)
		if err != nil {
			return nil, err
		}
		if !p.IsActive {
			return nil, fieldError("groups", "turn the agent on before putting it in a group")
		}
		// An agent of one branch goes only in that branch's groups (a shared one in any).
		if p.BranchID != nil && *p.BranchID != g.BranchID {
			return nil, fieldError("groups", fmt.Sprintf("“%s” is another branch’s group; %s can only go in its own branch’s groups", g.Title, p.Code))
		}
		w := wanted[id]
		plan := &assignPlan{branchIDs: []uuid.UUID{g.BranchID}, platformIDs: []uuid.UUID{p.ID}, platforms: map[uuid.UUID]models.Platform{p.ID: *p},
			rates: map[uuid.UUID]decimal.Decimal{p.ID: w.rate}, kinds: map[uuid.UUID]string{p.ID: w.kind}, currency: g.Currency, starts: today}
		if g.StartsOn.After(plan.starts) {
			plan.starts = g.StartsOn
		}
		if plan.branches, err = s.branchesIn(ctx, plan.branchIDs); err != nil {
			return nil, err
		}
		made, err := s.applyAssign(tx, plan, actor, &g.ID)
		if err != nil {
			return nil, err
		}
		counted, err := countSavedWeeks(tx, made)
		if err != nil {
			return nil, err
		}
		changes.Added, changes.Weeks = append(changes.Added, g.Title), changes.Weeks+counted
	}
	return changes, nil
}

// countSavedWeeks makes the weeks already saved for agents just put in groups count there, as
// saving them now would (an agent counts in every switched-on group it was in on any day of the
// week): each week from the assignment's first day on, saved while the agent wasn't in that group,
// gets its Total in the group at the assignment's rate and type (with the formula the week was
// saved with), and the result's Total and rate become its groups' added up again. Older weeks,
// finalized weeks, weeks of a group that is turned off and weeks saved under an assignment from
// before groups are left as they are. It returns how many weeks now count in a group.
func countSavedWeeks(tx *gorm.DB, made []Assignment) (int, error) {
	counted := 0
	now := time.Now()
	for _, a := range made {
		if a.GroupID == nil {
			continue
		}
		var g models.AssignmentGroup
		if err := tx.First(&g, "id = ?", *a.GroupID).Error; err != nil {
			return counted, err
		}
		if !g.IsActive {
			continue
		}
		q := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("branch_id = ? AND platform_id = ? AND week_end >= ?", a.BranchID, a.PlatformID, dayString(a.StartsOn)).
			Where("NOT EXISTS (SELECT 1 FROM weekly_group_results g WHERE g.result_id = weekly_platform_results.id AND (g.group_id = ? OR g.group_id IS NULL))", *a.GroupID).
			Where(`NOT EXISTS (SELECT 1 FROM weekly_settlements st WHERE st.branch_id = weekly_platform_results.branch_id
				AND st.week_start = weekly_platform_results.week_start AND st.status = ? AND st.deleted_at IS NULL)`, models.SettlementFinalized)
		if a.EndsOn != nil {
			q = q.Where("week_start <= ?", dayString(*a.EndsOn))
		}
		var results []models.WeeklyPlatformResult
		if err := q.Find(&results).Error; err != nil {
			return counted, err
		}
		kind := winlose.PO
		if winlose.Kind(a.CalcType) == winlose.GI {
			kind = winlose.GI
		}
		for _, r := range results {
			res, err := workOut(r.FormulaVersion, kind, r.WinLose, r.Commission, a.CommissionRate, nil)
			if err != nil {
				return counted, err
			}
			if err := tx.Create(&models.WeeklyGroupResult{ID: uuid.New(), ResultID: r.ID, GroupID: a.GroupID, AssignmentID: a.ID,
				CommissionRate: a.CommissionRate, CalcType: string(kind), JamePS: res.JamePS}).Error; err != nil {
				return counted, err
			}
			if err := tx.Model(&models.WeeklyPlatformResult{}).Where("id = ?", r.ID).UpdateColumns(map[string]interface{}{
				"commission_rate": r.CommissionRate.Add(a.CommissionRate), "jame_ps": r.JamePS.Add(res.JamePS), "updated_at": now,
			}).Error; err != nil {
				return counted, err
			}
			counted++
		}
	}
	return counted, nil
}
