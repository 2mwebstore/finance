package services

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/cambodia-finance/income-expense/internal/models"
	"github.com/cambodia-finance/income-expense/internal/permissions"
	"github.com/cambodia-finance/income-expense/internal/winlose"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// FormulaSettingKey picks the formula version new weeks are worked out with (app_settings).
const FormulaSettingKey = "winlose.formula"

// Formula is the version new weeks use: the winlose.formula setting, else winlose.Default.
func (s *PlatformService) Formula(ctx context.Context) (winlose.Formula, error) {
	version := winlose.Default
	var setting models.AppSetting
	if err := s.db.WithContext(ctx).First(&setting, "`key` = ?", FormulaSettingKey).Error; err == nil && strings.TrimSpace(setting.Value) != "" {
		version = strings.TrimSpace(setting.Value)
	}
	f, ok := winlose.Lookup(version)
	if !ok {
		return winlose.Formula{}, fmt.Errorf("the win/lose formula setting names an unknown version (%s)", version)
	}
	return f, nil
}

// ===== Entries (Data Entry) =====
//
// An entry is a titled set of agents' results for one week (Monday to Sunday) of a group (branch);
// a week can have several entries. Every agent that is on can be entered, whatever its groups. A
// result's Total is worked out in each group the agent is in that week, at the group's rate for
// it (an agent in two groups counts in both; in none, its Total is 0), and kept with the rates.

type SettlementState struct {
	ID        uuid.UUID `json:"id"`
	Status    string    `json:"status"`
	WeekStart string    `json:"week_start"`
	WeekEnd   string    `json:"week_end"`
}

// PartnerShare is one partner's part of a Total: at their %. Partners are only on assignments (and
// weeks) from before each group had its own rate for an agent.
type PartnerShare struct {
	PartnerID      uuid.UUID        `json:"partner_id"`
	PartnerName    string           `json:"partner_name"`
	CommissionRate decimal.Decimal  `json:"commission_rate"`
	JamePS         *decimal.Decimal `json:"jame_ps"` // nil until the week is entered
}

// GroupShare is an agent's part in one group: its Total at the group's rate for it.
type GroupShare struct {
	// GroupID nil: an assignment made before groups.
	GroupID        *uuid.UUID       `json:"group_id"`
	GroupTitle     string           `json:"group_title"`
	CommissionRate decimal.Decimal  `json:"commission_rate"`
	JamePS         *decimal.Decimal `json:"jame_ps"` // nil until saved
}

// WeekSheetRow is one agent on an entry: its saved result, or the groups it is in that week.
type WeekSheetRow struct {
	PlatformID   uuid.UUID `json:"platform_id"`
	PlatformCode string    `json:"platform_code"`
	PlatformName string    `json:"platform_name"`
	// Groups: its Total in each group, by title (none: it is in no group and its Total is 0).
	Groups []GroupShare `json:"groups"`
	// Partners: who shares the Total, by name (only on assignments from before groups had rates).
	Partners []PartnerShare `json:"partners"`
	// CommissionRate is the groups' rates added up.
	CommissionRate decimal.Decimal  `json:"commission_rate"`
	Currency       string           `json:"currency"`
	Saved          bool             `json:"saved"`
	WinLose        *decimal.Decimal `json:"win_lose"`
	Commission     *decimal.Decimal `json:"commission"`
	JamePS         *decimal.Decimal `json:"jame_ps"` // shown as "Total": the groups' added up
	FormulaVersion string           `json:"formula_version"`
	UpdatedBy      string           `json:"updated_by"`
	UpdatedAt      *time.Time       `json:"updated_at"`
}

// EntryState is the entry as Data Entry lists it (nil for a new entry until its first save).
type EntryState struct {
	ID        uuid.UUID `json:"id"`
	Title     string    `json:"title"`
	CreatedAt time.Time `json:"created_at"`
	CreatedBy string    `json:"created_by"`
}

// WeekSheet is an entry (or a new one) with every agent that can be entered: those that are on,
// and any saved on it.
type WeekSheet struct {
	WeekStart  string           `json:"week_start"`
	WeekEnd    string           `json:"week_end"`
	BranchID   uuid.UUID        `json:"branch_id"`
	BranchCode string           `json:"branch_code"`
	BranchName string           `json:"branch_name"`
	Entry      *EntryState      `json:"entry"`
	Formula    winlose.Formula  `json:"formula"`
	Settlement *SettlementState `json:"settlement"`
	// Locked: the week is finalized, so nothing can be changed.
	Locked bool           `json:"locked"`
	Rows   []WeekSheetRow `json:"rows"`
}

// share is an assignment a result counts in: an agent in a group, at the group's rate for it.
type share struct {
	group      *uuid.UUID
	assignment uuid.UUID
	rate       decimal.Decimal // with partners, their % added up
	partners   []PartnerShare  // assignments (and weeks) from before groups had their own rates
	existing   *models.WeeklyGroupResult
	total      decimal.Decimal
}

// entryParts are what an entry needs: its results with their groups, and the groups each agent
// is in that week (for the agents not entered yet).
type entryParts struct {
	branch     models.Branch
	start      time.Time
	entry      *models.WeeklyEntry                         // nil: a new entry
	results    map[uuid.UUID]models.WeeklyPlatformResult   // by platform
	saved      map[uuid.UUID][]models.WeeklyGroupResult    // by result
	partnerOf  map[uuid.UUID][]models.WeeklyPartnerResult  // by result
	ofAgent    map[uuid.UUID][]models.HQPlatformAssignment // by platform: the assignments it counts in
	links      map[uuid.UUID][]models.AssignmentPartner    // by assignment
	groups     map[uuid.UUID]models.AssignmentGroup
	partners   map[uuid.UUID]models.Partner
	platforms  map[uuid.UUID]models.Platform // every agent that is on, and those saved
	settlement *models.WeeklySettlement
}

// loadEntry reads an entry (nil: a new one) of a branch's week. An agent counts in a group (one
// that is switched on) when it was in it on any day of the week; taken out and put in again within
// the week, the later assignment is used. An assignment made before groups counts only when the
// agent is in no group that week.
func (s *PlatformService) loadEntry(tx *gorm.DB, branchID uuid.UUID, start time.Time, entry *models.WeeklyEntry, lock bool) (*entryParts, error) {
	end := start.AddDate(0, 0, 6)
	p := &entryParts{start: start, entry: entry, results: map[uuid.UUID]models.WeeklyPlatformResult{}, saved: map[uuid.UUID][]models.WeeklyGroupResult{},
		partnerOf: map[uuid.UUID][]models.WeeklyPartnerResult{}, ofAgent: map[uuid.UUID][]models.HQPlatformAssignment{}, groups: map[uuid.UUID]models.AssignmentGroup{}}
	if err := tx.First(&p.branch, "id = ?", branchID).Error; err != nil {
		return nil, fieldError("branch_id", "this group was not found")
	}
	var platformIDs, resultIDs, groupIDs []uuid.UUID
	if entry != nil {
		q := tx.Where("entry_id = ?", entry.ID)
		if lock {
			q = q.Clauses(clause.Locking{Strength: "UPDATE"})
		}
		var results []models.WeeklyPlatformResult
		if err := q.Find(&results).Error; err != nil {
			return nil, err
		}
		for _, r := range results {
			p.results[r.PlatformID] = r
			platformIDs, resultIDs = append(platformIDs, r.PlatformID), append(resultIDs, r.ID)
		}
	}
	if len(resultIDs) > 0 {
		var saved []models.WeeklyGroupResult
		if err := tx.Where("result_id IN ?", resultIDs).Find(&saved).Error; err != nil {
			return nil, err
		}
		for _, g := range saved {
			p.saved[g.ResultID] = append(p.saved[g.ResultID], g)
			if g.GroupID != nil {
				groupIDs = append(groupIDs, *g.GroupID)
			}
		}
		var partners []models.WeeklyPartnerResult
		if err := tx.Where("result_id IN ?", resultIDs).Find(&partners).Error; err != nil {
			return nil, err
		}
		for _, sh := range partners {
			p.partnerOf[sh.ResultID] = append(p.partnerOf[sh.ResultID], sh)
		}
	}
	var assignments []models.HQPlatformAssignment
	if err := tx.Where("branch_id = ? AND starts_on <= ? AND (ends_on IS NULL OR ends_on >= ?)", branchID, dayString(end), dayString(start)).
		Order("starts_on").Find(&assignments).Error; err != nil {
		return nil, err
	}
	for _, a := range assignments {
		if a.GroupID != nil {
			groupIDs = append(groupIDs, *a.GroupID)
		}
	}
	if len(groupIDs) > 0 {
		var groups []models.AssignmentGroup
		if err := tx.Where("id IN ?", groupIDs).Find(&groups).Error; err != nil {
			return nil, err
		}
		for _, g := range groups {
			p.groups[g.ID] = g
		}
	}
	inGroup := map[[2]uuid.UUID]models.HQPlatformAssignment{} // by agent and group
	before := map[uuid.UUID]models.HQPlatformAssignment{}     // by agent: made before groups
	for _, a := range assignments {
		if a.GroupID == nil {
			before[a.PlatformID] = a
		} else if g, ok := p.groups[*a.GroupID]; ok && g.IsActive {
			inGroup[[2]uuid.UUID{a.PlatformID, g.ID}] = a
		}
	}
	for key, a := range inGroup {
		p.ofAgent[key[0]] = append(p.ofAgent[key[0]], a)
	}
	for platformID, a := range before {
		if len(p.ofAgent[platformID]) == 0 {
			p.ofAgent[platformID] = []models.HQPlatformAssignment{a}
		}
	}
	var assignmentIDs []uuid.UUID
	for _, list := range p.ofAgent {
		for _, a := range list {
			assignmentIDs = append(assignmentIDs, a.ID)
		}
	}
	var err error
	if p.links, err = assignmentPartners(tx, assignmentIDs); err != nil {
		return nil, err
	}
	var partnerIDs []uuid.UUID
	for _, links := range p.links {
		for _, l := range links {
			partnerIDs = append(partnerIDs, l.PartnerID)
		}
	}
	for _, list := range p.partnerOf {
		for _, sh := range list {
			partnerIDs = append(partnerIDs, sh.PartnerID)
		}
	}
	if p.partners, err = s.partnersIn(tx.Statement.Context, partnerIDs); err != nil {
		return nil, err
	}
	if p.platforms, err = s.platformsIn(tx.Statement.Context, platformIDs); err != nil {
		return nil, err
	}
	var active []models.Platform
	if err := tx.Where("is_active = ?", true).Find(&active).Error; err != nil {
		return nil, err
	}
	for _, pl := range active {
		p.platforms[pl.ID] = pl
	}
	var st models.WeeklySettlement
	err = tx.Where("branch_id = ? AND week_start = ?", branchID, dayString(start)).First(&st).Error
	if err == nil {
		p.settlement = &st
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	return p, nil
}

func (p *entryParts) locked() bool {
	return p.settlement != nil && p.settlement.Status == models.SettlementFinalized
}

// newShares are the groups an agent is in that week, each at the group's rate for it.
func (p *entryParts) newShares(platformID uuid.UUID) []share {
	var out []share
	for _, a := range p.ofAgent[platformID] {
		var partners []PartnerShare
		for _, l := range p.links[a.ID] {
			partners = append(partners, PartnerShare{PartnerID: l.PartnerID, CommissionRate: l.CommissionRate})
		}
		out = append(out, share{group: a.GroupID, assignment: a.ID, rate: shareRate(a.CommissionRate, partners), partners: partners})
	}
	return out
}

// partnerShares lists who shares a row's Total, by name: a saved row's partners with their
// Totals, else its shares' partners.
func (p *entryParts) partnerShares(saved []models.WeeklyPartnerResult, shares []share) []PartnerShare {
	out := []PartnerShare{}
	for _, sh := range saved {
		total := sh.JamePS
		out = append(out, PartnerShare{PartnerID: sh.PartnerID, PartnerName: p.partners[sh.PartnerID].Name, CommissionRate: sh.CommissionRate, JamePS: &total})
	}
	if len(saved) == 0 {
		for _, sh := range shares {
			for _, ps := range sh.partners {
				out = append(out, PartnerShare{PartnerID: ps.PartnerID, PartnerName: p.partners[ps.PartnerID].Name, CommissionRate: ps.CommissionRate})
			}
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].PartnerName < out[j].PartnerName })
	return out
}

// partnersOf adds a new result's partners up over its shares (a partner in two of its groups once).
func partnersOf(shares []share) []models.WeeklyPartnerResult {
	var out []models.WeeklyPartnerResult
	at := map[uuid.UUID]int{}
	for _, sh := range shares {
		for _, ps := range sh.partners {
			i, ok := at[ps.PartnerID]
			if !ok {
				i = len(out)
				at[ps.PartnerID] = i
				out = append(out, models.WeeklyPartnerResult{PartnerID: ps.PartnerID})
			}
			out[i].CommissionRate = out[i].CommissionRate.Add(ps.CommissionRate)
			out[i].JamePS = out[i].JamePS.Add(*ps.JamePS)
		}
	}
	return out
}

// savedShares are a saved result's groups at the rates it was worked out with (a week from before
// groups had their own rates keeps its partners too).
func (p *entryParts) savedShares(r models.WeeklyPlatformResult) []share {
	list := p.saved[r.ID]
	var out []share
	for i := range list {
		sh := share{group: list[i].GroupID, assignment: list[i].AssignmentID, rate: list[i].CommissionRate, existing: &list[i]}
		if len(list) == 1 {
			for _, pr := range p.partnerOf[r.ID] {
				sh.partners = append(sh.partners, PartnerShare{PartnerID: pr.PartnerID, CommissionRate: pr.CommissionRate})
			}
		}
		out = append(out, sh)
	}
	return out
}

// groupShares shows shares by group title (their Totals once saved).
func (p *entryParts) groupShares(list []share) []GroupShare {
	out := make([]GroupShare, 0, len(list))
	for _, sh := range list {
		g := GroupShare{GroupID: sh.group, CommissionRate: sh.rate}
		if sh.group != nil {
			g.GroupTitle = p.groups[*sh.group].Title
		}
		if sh.existing != nil {
			total := sh.existing.JamePS
			g.JamePS = &total
		}
		out = append(out, g)
	}
	sort.Slice(out, func(i, j int) bool { return strings.ToLower(out[i].GroupTitle) < strings.ToLower(out[j].GroupTitle) })
	return out
}

// OwnGroup is the group (branch) a person works in on the PLATFORMS pages (entries, assignments,
// settlements, reports without consolidated access): their default branch, else their only
// branch. Those pages don't ask for one.
func (s *PlatformService) OwnGroup(ctx context.Context, actor uuid.UUID, scope BranchScope) (uuid.UUID, error) {
	var user models.User
	if err := s.db.WithContext(ctx).Preload("Branches").First(&user, "id = ?", actor).Error; err != nil {
		return uuid.Nil, err
	}
	if user.DefaultBranchID != nil && scope.Has(*user.DefaultBranchID) {
		return *user.DefaultBranchID, nil
	}
	var mine []uuid.UUID
	for _, b := range user.Branches {
		if scope.Has(b.ID) {
			mine = append(mine, b.ID)
		}
	}
	if len(mine) == 1 {
		return mine[0], nil
	}
	if !scope.All && len(scope.IDs) == 1 {
		return scope.IDs[0], nil
	}
	return uuid.Nil, fieldError("branch_id", "your account has no default branch, so there's no group to work in; ask an admin to set one in Manage → Users")
}

// groupFor is the group asked for, else the person's own.
func (s *PlatformService) groupFor(ctx context.Context, raw string, scope BranchScope, actor uuid.UUID) (uuid.UUID, error) {
	if strings.TrimSpace(raw) == "" {
		return s.OwnGroup(ctx, actor, scope)
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		return uuid.Nil, fieldError("branch_id", "choose a group")
	}
	if !scope.Has(id) {
		return uuid.Nil, ErrBranchAccessDenied
	}
	return id, nil
}

var ErrEntryNotFound = errors.New("entry not found")

// findEntry loads an entry in scope (locked, for a change).
func findEntry(tx *gorm.DB, raw string, scope BranchScope, lock bool) (*models.WeeklyEntry, error) {
	id, err := uuid.Parse(strings.TrimSpace(raw))
	if err != nil {
		return nil, ErrEntryNotFound
	}
	if lock {
		tx = tx.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	var e models.WeeklyEntry
	if err := tx.First(&e, "id = ?", id).Error; err != nil || !scope.Has(e.BranchID) {
		return nil, ErrEntryNotFound
	}
	return &e, nil
}

// EntrySheet is an entry with every agent that can be entered or, with no entry id, a new entry
// for the week of the given day (empty: this week) in the group asked for (empty: the person's own).
func (s *PlatformService) EntrySheet(ctx context.Context, entryID, week, group string, scope BranchScope, actor uuid.UUID) (*WeekSheet, error) {
	formula, err := s.Formula(ctx)
	if err != nil {
		return nil, err
	}
	parts, err := s.entryFor(ctx, s.db.WithContext(ctx), WeekEntryInput{EntryID: entryID, WeekStart: week, BranchID: group}, scope, actor, false)
	if err != nil {
		return nil, err
	}
	return s.sheet(ctx, parts, formula), nil
}

// entryFor loads the entry asked for or, with no entry id, the week a new one is for (in the group
// asked for, else the person's own).
func (s *PlatformService) entryFor(ctx context.Context, tx *gorm.DB, in WeekEntryInput, scope BranchScope, actor uuid.UUID, lock bool) (*entryParts, error) {
	if strings.TrimSpace(in.EntryID) != "" {
		entry, err := findEntry(tx, in.EntryID, scope, lock)
		if err != nil {
			return nil, err
		}
		return s.loadEntry(tx, entry.BranchID, entry.WeekStart, entry, lock)
	}
	branchID, err := s.groupFor(ctx, in.BranchID, scope, actor)
	if err != nil {
		return nil, err
	}
	day, err := s.parseDay("week_start", in.WeekStart)
	if err != nil {
		return nil, err
	}
	start := weekStart(day)
	if start.After(s.today()) {
		return nil, fieldError("week_start", "results can't be entered for a week that hasn't started")
	}
	return s.loadEntry(tx, branchID, start, nil, lock)
}

func (s *PlatformService) sheet(ctx context.Context, p *entryParts, formula winlose.Formula) *WeekSheet {
	out := &WeekSheet{WeekStart: dayString(p.start), WeekEnd: dayString(p.start.AddDate(0, 0, 6)), BranchID: p.branch.ID, BranchCode: p.branch.Code, BranchName: p.branch.Name,
		Formula: formula, Locked: p.locked(), Rows: []WeekSheetRow{}}
	if p.settlement != nil {
		out.Settlement = &SettlementState{ID: p.settlement.ID, Status: p.settlement.Status, WeekStart: dayString(p.settlement.WeekStart), WeekEnd: dayString(p.settlement.WeekEnd)}
	}
	var users []uuid.UUID
	for _, r := range p.results {
		users = append(users, r.CreatedByID)
		if r.UpdatedByID != nil {
			users = append(users, *r.UpdatedByID)
		}
	}
	if p.entry != nil {
		users = append(users, p.entry.CreatedByID)
	}
	names := s.userNames(ctx, users)
	if p.entry != nil {
		out.Entry = &EntryState{ID: p.entry.ID, Title: p.entry.Title, CreatedAt: p.entry.CreatedAt, CreatedBy: names[p.entry.CreatedByID]}
	}
	for platformID, pl := range p.platforms {
		row := WeekSheetRow{PlatformID: platformID, PlatformCode: pl.Code, PlatformName: pl.Name}
		if r, ok := p.results[platformID]; ok {
			wl, cm, jp, at := r.WinLose, r.Commission, r.JamePS, r.UpdatedAt
			row.CommissionRate, row.Currency, row.FormulaVersion = r.CommissionRate, r.Currency, r.FormulaVersion
			row.Saved, row.WinLose, row.Commission, row.JamePS, row.UpdatedAt = true, &wl, &cm, &jp, &at
			row.Groups, row.Partners = p.groupShares(p.savedShares(r)), p.partnerShares(p.partnerOf[r.ID], nil)
			row.UpdatedBy = names[r.CreatedByID]
			if r.UpdatedByID != nil {
				row.UpdatedBy = names[*r.UpdatedByID]
			}
		} else if pl.IsActive {
			shares := p.newShares(platformID)
			row.Currency, row.FormulaVersion = pl.DefaultCurrency, formula.Version
			row.Groups, row.Partners = p.groupShares(shares), p.partnerShares(nil, shares)
			for _, sh := range shares {
				row.CommissionRate = row.CommissionRate.Add(sh.rate)
			}
		} else {
			continue
		}
		out.Rows = append(out.Rows, row)
	}
	sort.Slice(out.Rows, func(i, j int) bool { return agentBefore(out.Rows[i].PlatformCode, out.Rows[j].PlatformCode) })
	return out
}

// shareRate is a share's rate: its partners' % added up, or its own when it has none.
func shareRate(own decimal.Decimal, partners []PartnerShare) decimal.Decimal {
	if len(partners) == 0 {
		return own
	}
	sum := decimal.Zero
	for _, p := range partners {
		sum = sum.Add(p.CommissionRate)
	}
	return sum
}

// ===== Saving an entry =====

type WeekEntryRow struct {
	PlatformID string `json:"platform_id" binding:"required,uuid"`
	WinLose    string `json:"win_lose"`
	Commission string `json:"commission"`
}

type WeekEntryInput struct {
	// EntryID is the entry to change; empty makes a new one (a new entry's first save).
	EntryID string `json:"entry_id" binding:"omitempty,uuid"`
	// A new entry's week: any day of it (empty: this week).
	WeekStart string `json:"week_start"`
	// A new entry's group; empty: the person's own.
	BranchID string         `json:"branch_id" binding:"omitempty,uuid"`
	Title    string         `json:"title" binding:"max=128"`
	Rows     []WeekEntryRow `json:"rows" binding:"required,max=500,dive"`
	// Needed when a saved row is changed (kept in the audit log).
	Reason string `json:"reason" binding:"max=512"`
}

// Correction is a saved row that was changed (for the audit log). A title change has no platform
// code; ResultID is then the entry's.
type Correction struct {
	ResultID     uuid.UUID
	PlatformCode string
	Before       map[string]string
	After        map[string]string
}

type rowPlan struct {
	platform        uuid.UUID
	winLose, commis decimal.Decimal
	// result: the Total is the shares' added up, rate their rates.
	result   winlose.Result
	rate     decimal.Decimal
	currency string
	shares   []share
	existing *models.WeeklyPlatformResult
}

// workOut gives a Total: with partners, each one's share at their % (filled into shares) and the
// Total their sum; without, the Total at the given %.
func workOut(version string, wl, cm, rate decimal.Decimal, shares []PartnerShare) (winlose.Result, error) {
	if len(shares) == 0 {
		return winlose.Calculate(version, winlose.Input{WinLose: wl, Commission: cm, RatePct: rate})
	}
	total := winlose.Result{}
	for i := range shares {
		r, err := winlose.Calculate(version, winlose.Input{WinLose: wl, Commission: cm, RatePct: shares[i].CommissionRate})
		if err != nil {
			return winlose.Result{}, err
		}
		jp := r.JamePS
		shares[i].JamePS = &jp
		total.Version, total.Adjusted, total.JamePS = r.Version, r.Adjusted, total.JamePS.Add(r.JamePS)
	}
	return total, nil
}

// plan works out every filled-in row with the current formula: a saved one in the groups and at
// the rates it was worked out with, a new one in the agent's groups that week. Blank rows are left
// out. (Only a week that isn't finalized can be changed, and those follow the current formula.)
func (s *PlatformService) plan(p *entryParts, rows []WeekEntryRow, formula winlose.Formula) ([]rowPlan, error) {
	var out []rowPlan
	seen := map[uuid.UUID]bool{}
	for i, row := range rows {
		platformID := uuid.MustParse(row.PlatformID)
		if seen[platformID] {
			return nil, fieldError("rows", "an agent is listed twice")
		}
		seen[platformID] = true
		if strings.TrimSpace(row.WinLose) == "" && strings.TrimSpace(row.Commission) == "" {
			continue
		}
		pl, known := p.platforms[platformID]
		name := pl.Code
		if name == "" {
			name = fmt.Sprintf("row %d", i+1)
		}
		wl, err := parseSignedAmount("rows", row.WinLose)
		if err != nil {
			return nil, fieldError("rows", name+": win/lose must be an amount (at most 4 decimals)")
		}
		cm, err := parseSignedAmount("rows", row.Commission)
		if err != nil {
			return nil, fieldError("rows", name+": commission must be an amount (at most 4 decimals)")
		}
		plan := rowPlan{platform: platformID, winLose: wl, commis: cm}
		version := formula.Version
		if r, ok := p.results[platformID]; ok {
			r := r
			plan.existing, plan.currency, plan.shares = &r, r.Currency, p.savedShares(r)
		} else if known && pl.IsActive {
			plan.currency, plan.shares = pl.DefaultCurrency, p.newShares(platformID)
		} else {
			return nil, fieldError("rows", name+" was not found or is turned off")
		}
		base, err := winlose.Calculate(version, winlose.Input{WinLose: wl, Commission: cm})
		if err != nil {
			return nil, err
		}
		plan.result = winlose.Result{Version: version, Adjusted: base.Adjusted}
		for j := range plan.shares {
			sh := &plan.shares[j]
			r, err := workOut(version, wl, cm, sh.rate, sh.partners)
			if err != nil {
				return nil, err
			}
			sh.total = r.JamePS
			plan.rate, plan.result.JamePS = plan.rate.Add(sh.rate), plan.result.JamePS.Add(sh.total)
		}
		out = append(out, plan)
	}
	return out, nil
}

// PreviewRow is what a filled-in row would save as.
type PreviewRow struct {
	PlatformID     uuid.UUID       `json:"platform_id"`
	CommissionRate decimal.Decimal `json:"commission_rate"`
	Currency       string          `json:"currency"`
	Groups         []GroupShare    `json:"groups"`
	winlose.Result
}

// Preview works out rows without saving them.
func (s *PlatformService) Preview(ctx context.Context, in WeekEntryInput, scope BranchScope, actor uuid.UUID) ([]PreviewRow, error) {
	formula, err := s.Formula(ctx)
	if err != nil {
		return nil, err
	}
	parts, err := s.entryFor(ctx, s.db.WithContext(ctx), in, scope, actor, false)
	if err != nil {
		return nil, err
	}
	plans, err := s.plan(parts, in.Rows, formula)
	if err != nil {
		return nil, err
	}
	out := make([]PreviewRow, len(plans))
	for i, p := range plans {
		groups := parts.groupShares(p.shares)
		for j, sh := range p.shares {
			total := sh.total
			groups[j].JamePS = &total
		}
		out[i] = PreviewRow{PlatformID: p.platform, CommissionRate: p.rate, Currency: p.currency, Groups: groups, Result: p.result}
	}
	return out, nil
}

// SaveWeek saves an entry in one transaction: its title and its agents' results, each with its
// Total in every group the agent is in. With no entry id it makes a new entry (a new entry's first
// save); the sheet returned has its id for the saves after. A new entry and new rows need
// winlose.create; changing a saved row or the title needs winlose.edit (a saved row also a
// reason). A finalized week can't be changed.
func (s *PlatformService) SaveWeek(ctx context.Context, in WeekEntryInput, scope BranchScope, can func(string) bool, actor uuid.UUID) (*WeekSheet, []Correction, error) {
	title := strings.TrimSpace(in.Title)
	if title == "" {
		return nil, nil, fieldError("title", "title is required")
	}
	formula, err := s.Formula(ctx)
	if err != nil {
		return nil, nil, err
	}
	reason := strings.TrimSpace(in.Reason)
	var corrections []Correction
	var entryID uuid.UUID
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		parts, err := s.entryFor(ctx, tx, in, scope, actor, true)
		if err != nil {
			return err
		}
		if parts.locked() {
			return fmt.Errorf("the week of %s is finalized for %s; reopen it to make changes", dayString(parts.start), parts.branch.Code)
		}
		plans, err := s.plan(parts, in.Rows, formula)
		if err != nil {
			return err
		}
		now := time.Now()
		entry := parts.entry
		switch {
		case entry == nil:
			if !can(permissions.WinLoseCreate) {
				return errors.New("you can't add entries (needs Enter Win/Lose)")
			}
			entry = &models.WeeklyEntry{ID: uuid.New(), BranchID: parts.branch.ID, WeekStart: parts.start, Title: title, CreatedByID: actor}
			if err := tx.Create(entry).Error; err != nil {
				return err
			}
		case entry.Title != title:
			if !can(permissions.WinLoseEdit) {
				return errors.New("you can't change a saved entry's title (needs Correct Win/Lose)")
			}
			if err := tx.Model(entry).Updates(map[string]interface{}{"title": title, "updated_by_id": actor, "updated_at": now}).Error; err != nil {
				return err
			}
			corrections = append(corrections, Correction{ResultID: entry.ID, Before: map[string]string{"title": entry.Title}, After: map[string]string{"title": title}})
		}
		entryID = entry.ID
		for _, p := range plans {
			code := parts.platforms[p.platform].Code
			if p.existing == nil {
				if !can(permissions.WinLoseCreate) {
					return errors.New("you can't enter new results (needs Enter Win/Lose)")
				}
				row := models.WeeklyPlatformResult{
					Base: models.Base{ID: uuid.New()}, EntryID: &entry.ID, WeekStart: parts.start, WeekEnd: parts.start.AddDate(0, 0, 6), BranchID: parts.branch.ID, PlatformID: p.platform,
					WinLose: p.winLose, Commission: p.commis, CommissionRate: p.rate, Currency: p.currency,
					AdjustedAmount: p.result.Adjusted, JamePS: p.result.JamePS, FormulaVersion: p.result.Version, CreatedByID: actor,
				}
				if err := tx.Create(&row).Error; err != nil {
					if isDuplicate(err) {
						return errors.New(code + " was just saved on this entry by someone else; reload and check it")
					}
					return err
				}
				for _, sh := range p.shares {
					if err := tx.Create(&models.WeeklyGroupResult{ID: uuid.New(), ResultID: row.ID, GroupID: sh.group, AssignmentID: sh.assignment, CommissionRate: sh.rate, JamePS: sh.total}).Error; err != nil {
						return err
					}
				}
				for _, pr := range partnersOf(p.shares) {
					pr.ID, pr.ResultID = uuid.New(), row.ID
					if err := tx.Create(&pr).Error; err != nil {
						return err
					}
				}
				continue
			}
			old := *p.existing
			if old.WinLose.Equal(p.winLose) && old.Commission.Equal(p.commis) {
				continue
			}
			if !can(permissions.WinLoseEdit) {
				return errors.New("you can't change a saved week (needs Correct Win/Lose)")
			}
			if len(reason) < 3 {
				return fieldError("reason", "give a reason for changing a saved week")
			}
			if err := tx.Model(&models.WeeklyPlatformResult{}).Where("id = ?", old.ID).Updates(map[string]interface{}{
				"win_lose": p.winLose, "commission": p.commis, "adjusted_amount": p.result.Adjusted, "jame_ps": p.result.JamePS,
				"formula_version": p.result.Version, "updated_by_id": actor, "updated_at": now,
			}).Error; err != nil {
				return err
			}
			for _, sh := range p.shares {
				if err := tx.Model(&models.WeeklyGroupResult{}).Where("id = ?", sh.existing.ID).Updates(map[string]interface{}{"jame_ps": sh.total, "updated_at": now}).Error; err != nil {
					return err
				}
				for _, ps := range sh.partners {
					if err := tx.Model(&models.WeeklyPartnerResult{}).Where("result_id = ? AND partner_id = ?", old.ID, ps.PartnerID).
						Updates(map[string]interface{}{"jame_ps": *ps.JamePS, "updated_at": now}).Error; err != nil {
						return err
					}
				}
			}
			corrections = append(corrections, Correction{
				ResultID: old.ID, PlatformCode: code,
				Before: map[string]string{"win_lose": old.WinLose.String(), "commission": old.Commission.String(), "total": old.JamePS.String()},
				After:  map[string]string{"win_lose": p.winLose.String(), "commission": p.commis.String(), "total": p.result.JamePS.String()},
			})
		}
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	sheet, err := s.EntrySheet(ctx, entryID.String(), "", "", scope, actor)
	return sheet, corrections, err
}

// RecalculateOpenWeeks works every saved week that isn't finalized out again with the current
// formula when it was saved with another, so a formula change applies to it; finalized weeks keep
// what they were settled with. Run at start-up. It returns how many results changed.
func (s *PlatformService) RecalculateOpenWeeks(ctx context.Context) (int, error) {
	formula, err := s.Formula(ctx)
	if err != nil {
		return 0, err
	}
	var results []models.WeeklyPlatformResult
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("formula_version <> ?", formula.Version).
			Where(`NOT EXISTS (SELECT 1 FROM weekly_settlements st WHERE st.branch_id = weekly_platform_results.branch_id
				AND st.week_start = weekly_platform_results.week_start AND st.status = ? AND st.deleted_at IS NULL)`, models.SettlementFinalized).
			Find(&results).Error; err != nil {
			return err
		}
		return recalculate(tx, formula, results)
	})
	return len(results), err
}

// recalculate works saved results out again with the formula: each in its groups at the rates it
// was saved with (a week from before groups had their own rates, with its partners).
func recalculate(tx *gorm.DB, formula winlose.Formula, results []models.WeeklyPlatformResult) error {
	if len(results) == 0 {
		return nil
	}
	ids := make([]uuid.UUID, len(results))
	for i, r := range results {
		ids[i] = r.ID
	}
	var groups []models.WeeklyGroupResult
	if err := tx.Where("result_id IN ?", ids).Find(&groups).Error; err != nil {
		return err
	}
	var partners []models.WeeklyPartnerResult
	if err := tx.Where("result_id IN ?", ids).Find(&partners).Error; err != nil {
		return err
	}
	groupsOf, partnersOf := map[uuid.UUID][]models.WeeklyGroupResult{}, map[uuid.UUID][]PartnerShare{}
	for _, g := range groups {
		groupsOf[g.ResultID] = append(groupsOf[g.ResultID], g)
	}
	for _, p := range partners {
		partnersOf[p.ResultID] = append(partnersOf[p.ResultID], PartnerShare{PartnerID: p.PartnerID, CommissionRate: p.CommissionRate})
	}
	for _, r := range results {
		base, err := winlose.Calculate(formula.Version, winlose.Input{WinLose: r.WinLose, Commission: r.Commission})
		if err != nil {
			return err
		}
		total := decimal.Zero
		for _, g := range groupsOf[r.ID] {
			var shares []PartnerShare
			if len(groupsOf[r.ID]) == 1 {
				shares = partnersOf[r.ID]
			}
			res, err := workOut(formula.Version, r.WinLose, r.Commission, g.CommissionRate, shares)
			if err != nil {
				return err
			}
			if err := tx.Model(&models.WeeklyGroupResult{}).Where("id = ?", g.ID).UpdateColumn("jame_ps", res.JamePS).Error; err != nil {
				return err
			}
			for _, sh := range shares {
				if err := tx.Model(&models.WeeklyPartnerResult{}).Where("result_id = ? AND partner_id = ?", r.ID, sh.PartnerID).UpdateColumn("jame_ps", *sh.JamePS).Error; err != nil {
					return err
				}
			}
			total = total.Add(res.JamePS)
		}
		if err := tx.Model(&models.WeeklyPlatformResult{}).Where("id = ?", r.ID).UpdateColumns(map[string]interface{}{
			"adjusted_amount": base.Adjusted, "jame_ps": total, "formula_version": formula.Version,
		}).Error; err != nil {
			return err
		}
	}
	return nil
}

// ===== Entries =====

// DeleteEntry removes an entry and its agents' results (all or nothing). It needs a reason (audit
// log); a finalized week can't be changed. It returns what was removed.
func (s *PlatformService) DeleteEntry(ctx context.Context, id uuid.UUID, reason string, scope BranchScope) (*models.WeeklyEntry, []models.WeeklyPlatformResult, error) {
	reason = strings.TrimSpace(reason)
	if len(reason) < 3 {
		return nil, nil, fieldError("reason", "give a reason for deleting the entry")
	}
	var entry *models.WeeklyEntry
	var removed []models.WeeklyPlatformResult
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var err error
		if entry, err = findEntry(tx, id.String(), scope, true); err != nil {
			return err
		}
		week := dayString(entry.WeekStart)
		var finalized int64
		if err := tx.Model(&models.WeeklySettlement{}).Where("branch_id = ? AND week_start = ? AND status = ?", entry.BranchID, week, models.SettlementFinalized).
			Count(&finalized).Error; err != nil {
			return err
		}
		if finalized > 0 {
			return fmt.Errorf("the week of %s is finalized; reopen it on the weekly report to delete the entry", week)
		}
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("entry_id = ?", entry.ID).Find(&removed).Error; err != nil {
			return err
		}
		// For good (with each group's and partner's share).
		if len(removed) > 0 {
			ids := make([]uuid.UUID, len(removed))
			for i, r := range removed {
				ids[i] = r.ID
			}
			if err := tx.Where("result_id IN ?", ids).Delete(&models.WeeklyGroupResult{}).Error; err != nil {
				return err
			}
			if err := tx.Where("result_id IN ?", ids).Delete(&models.WeeklyPartnerResult{}).Error; err != nil {
				return err
			}
			if err := tx.Unscoped().Where("id IN ?", ids).Delete(&models.WeeklyPlatformResult{}).Error; err != nil {
				return err
			}
		}
		return tx.Delete(entry).Error
	})
	if err != nil {
		return nil, nil, err
	}
	return entry, removed, nil
}

// EntryRow is one saved week on Data Entry's list.
type EntryRow struct {
	ID         uuid.UUID `json:"id"`
	Title      string    `json:"title"`
	WeekStart  string    `json:"week_start"`
	WeekEnd    string    `json:"week_end"`
	BranchID   uuid.UUID `json:"branch_id"`
	BranchCode string    `json:"branch_code"`
	CreatedAt  time.Time `json:"created_at"`
	CreatedBy  string    `json:"created_by"`
	// Locked: the week is finalized.
	Locked bool `json:"locked"`
}

// EntryFilter narrows Data Entry's list: words in the title, and the days it was created on
// (YYYY-MM-DD, either may be empty).
type EntryFilter struct {
	Search                 string
	CreatedFrom, CreatedTo string
}

// likePattern matches text anywhere in a column (LIKE's own wildcards taken literally).
func likePattern(text string) string {
	return "%" + strings.NewReplacer(`\`, `\\`, "%", `\%`, "_", `\_`).Replace(text) + "%"
}

// ListEntries lists the entries of the groups in scope, newest first (at most 500).
func (s *PlatformService) ListEntries(ctx context.Context, f EntryFilter, scope BranchScope) ([]EntryRow, error) {
	q := scope.where(s.db.WithContext(ctx).Model(&models.WeeklyEntry{}), "branch_id")
	if search := strings.TrimSpace(f.Search); search != "" {
		q = q.Where("title LIKE ?", likePattern(search))
	}
	// Created days are the business's days (its time zone), from midnight to midnight.
	if strings.TrimSpace(f.CreatedFrom) != "" {
		day, err := s.parseDay("created_from", f.CreatedFrom)
		if err != nil {
			return nil, err
		}
		q = q.Where("created_at >= ?", time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, s.loc))
	}
	if strings.TrimSpace(f.CreatedTo) != "" {
		day, err := s.parseDay("created_to", f.CreatedTo)
		if err != nil {
			return nil, err
		}
		q = q.Where("created_at < ?", time.Date(day.Year(), day.Month(), day.Day()+1, 0, 0, 0, 0, s.loc))
	}
	var list []models.WeeklyEntry
	if err := q.Order("created_at DESC, id").Limit(500).Find(&list).Error; err != nil {
		return nil, err
	}
	var branchIDs, users []uuid.UUID
	for _, e := range list {
		branchIDs, users = append(branchIDs, e.BranchID), append(users, e.CreatedByID)
	}
	branches, err := s.branchesIn(ctx, branchIDs)
	if err != nil {
		return nil, err
	}
	names := s.userNames(ctx, users)
	locked := map[string]bool{}
	if len(branchIDs) > 0 {
		var settled []models.WeeklySettlement
		if err := s.db.WithContext(ctx).Where("branch_id IN ? AND status = ?", branchIDs, models.SettlementFinalized).Find(&settled).Error; err != nil {
			return nil, err
		}
		for _, st := range settled {
			locked[st.BranchID.String()+dayString(st.WeekStart)] = true
		}
	}
	out := make([]EntryRow, len(list))
	for i, e := range list {
		out[i] = EntryRow{ID: e.ID, Title: e.Title, WeekStart: dayString(e.WeekStart), WeekEnd: dayString(e.WeekStart.AddDate(0, 0, 6)),
			BranchID: e.BranchID, BranchCode: branches[e.BranchID].Code, CreatedAt: e.CreatedAt, CreatedBy: names[e.CreatedByID],
			Locked: locked[e.BranchID.String()+dayString(e.WeekStart)]}
	}
	return out, nil
}

// ===== Weekly settlements =====

type SettlementView struct {
	models.WeeklySettlement
	BranchCode  string                         `json:"branch_code"`
	BranchName  string                         `json:"branch_name"`
	FinalizedBy string                         `json:"finalized_by"`
	ReopenedBy  string                         `json:"reopened_by"`
	Totals      []models.WeeklySettlementTotal `json:"totals"`
}

// Settlements lists the week's settlements for the HQs in scope (or one HQ).
func (s *PlatformService) Settlements(ctx context.Context, week string, branchID *uuid.UUID, scope BranchScope) ([]SettlementView, error) {
	day, err := s.parseDay("week_start", week)
	if err != nil {
		return nil, err
	}
	q := scope.where(s.db.WithContext(ctx).Model(&models.WeeklySettlement{}), "branch_id").Where("week_start = ?", dayString(weekStart(day)))
	if branchID != nil {
		if !scope.Has(*branchID) {
			return nil, ErrBranchAccessDenied
		}
		q = q.Where("branch_id = ?", *branchID)
	}
	var list []models.WeeklySettlement
	if err := q.Find(&list).Error; err != nil {
		return nil, err
	}
	return s.settlementViews(ctx, list)
}

func (s *PlatformService) settlementViews(ctx context.Context, list []models.WeeklySettlement) ([]SettlementView, error) {
	var branchIDs, userIDs, ids []uuid.UUID
	for _, st := range list {
		branchIDs, ids = append(branchIDs, st.BranchID), append(ids, st.ID)
		userIDs = append(userIDs, st.FinalizedByID)
		if st.ReopenedByID != nil {
			userIDs = append(userIDs, *st.ReopenedByID)
		}
	}
	branches, err := s.branchesIn(ctx, branchIDs)
	if err != nil {
		return nil, err
	}
	users := s.userNames(ctx, userIDs)
	totals := map[uuid.UUID][]models.WeeklySettlementTotal{}
	if len(ids) > 0 {
		var rows []models.WeeklySettlementTotal
		if err := s.db.WithContext(ctx).Where("settlement_id IN ?", ids).Order("currency").Find(&rows).Error; err != nil {
			return nil, err
		}
		for _, t := range rows {
			totals[t.SettlementID] = append(totals[t.SettlementID], t)
		}
	}
	out := make([]SettlementView, len(list))
	for i, st := range list {
		v := SettlementView{WeeklySettlement: st, BranchCode: branches[st.BranchID].Code, BranchName: branches[st.BranchID].Name,
			FinalizedBy: users[st.FinalizedByID], Totals: totals[st.ID]}
		if v.Totals == nil {
			v.Totals = []models.WeeklySettlementTotal{}
		}
		if st.ReopenedByID != nil {
			v.ReopenedBy = users[*st.ReopenedByID]
		}
		out[i] = v
	}
	sort.Slice(out, func(i, j int) bool { return out[i].BranchCode < out[j].BranchCode })
	return out, nil
}

type FinalizeInput struct {
	// Empty: the person's own group.
	BranchID  string `json:"branch_id" binding:"omitempty,uuid"`
	WeekStart string `json:"week_start" binding:"required"` // a Monday
}

// FinalizeWeek freezes an HQ's week: its totals per currency are kept and the week locked. A
// reopened week can be finalized again (the totals are worked out afresh).
func (s *PlatformService) FinalizeWeek(ctx context.Context, in FinalizeInput, scope BranchScope, actor uuid.UUID) (*SettlementView, error) {
	branchID, err := s.groupFor(ctx, in.BranchID, scope, actor)
	if err != nil {
		return nil, err
	}
	start, err := s.parseDay("week_start", in.WeekStart)
	if err != nil {
		return nil, err
	}
	if start.Weekday() != time.Monday {
		return nil, fieldError("week_start", "a week starts on a Monday")
	}
	end := start.AddDate(0, 0, 6)
	if end.After(s.today()) {
		return nil, fieldError("week_start", "a week can be finalized from its Sunday ("+dayString(end)+")")
	}
	var id uuid.UUID
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var st models.WeeklySettlement
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("branch_id = ? AND week_start = ?", branchID, dayString(start)).First(&st).Error
		switch {
		case err == nil && st.Status == models.SettlementFinalized:
			return errors.New("this week is already finalized")
		case err == nil:
		case errors.Is(err, gorm.ErrRecordNotFound):
			st = models.WeeklySettlement{Base: models.Base{ID: uuid.New()}, BranchID: branchID, WeekStart: start, WeekEnd: end}
		default:
			return err
		}
		sums, err := s.sumByCurrency(tx, branchID, start)
		if err != nil {
			return err
		}
		st.Status, st.FinalizedByID, st.FinalizedAt = models.SettlementFinalized, actor, time.Now()
		if err := tx.Save(&st).Error; err != nil {
			if isDuplicate(err) {
				return errors.New("this week was just finalized by someone else")
			}
			return err
		}
		if err := tx.Where("settlement_id = ?", st.ID).Delete(&models.WeeklySettlementTotal{}).Error; err != nil {
			return err
		}
		for _, t := range sums {
			t.ID, t.SettlementID = uuid.New(), st.ID
			if err := tx.Create(&t).Error; err != nil {
				return err
			}
		}
		id = st.ID
		return nil
	})
	if err != nil {
		return nil, err
	}
	return s.settlementByID(ctx, id)
}

// ReopenWeek unlocks a finalized week so it can be corrected; it must be finalized again.
func (s *PlatformService) ReopenWeek(ctx context.Context, id uuid.UUID, reason string, scope BranchScope, actor uuid.UUID) (*SettlementView, error) {
	reason = strings.TrimSpace(reason)
	if len(reason) < 3 {
		return nil, fieldError("reason", "give a reason for reopening the week")
	}
	formula, err := s.Formula(ctx)
	if err != nil {
		return nil, err
	}
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var st models.WeeklySettlement
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&st, "id = ?", id).Error; err != nil || !scope.Has(st.BranchID) {
			return ErrSettlementNotFound
		}
		if st.Status != models.SettlementFinalized {
			return errors.New("this week is not finalized")
		}
		now := time.Now()
		if err := tx.Model(&st).Updates(map[string]interface{}{"status": models.SettlementReopened, "reopened_by_id": actor, "reopened_at": now, "reopen_reason": reason, "updated_at": now}).Error; err != nil {
			return err
		}
		// Reopened, the week follows the current formula again.
		var results []models.WeeklyPlatformResult
		if err := tx.Where("branch_id = ? AND week_start = ? AND formula_version <> ?", st.BranchID, dayString(st.WeekStart), formula.Version).Find(&results).Error; err != nil {
			return err
		}
		return recalculate(tx, formula, results)
	})
	if err != nil {
		return nil, err
	}
	return s.settlementByID(ctx, id)
}

func (s *PlatformService) settlementByID(ctx context.Context, id uuid.UUID) (*SettlementView, error) {
	var st models.WeeklySettlement
	if err := s.db.WithContext(ctx).First(&st, "id = ?", id).Error; err != nil {
		return nil, ErrSettlementNotFound
	}
	views, err := s.settlementViews(ctx, []models.WeeklySettlement{st})
	if err != nil {
		return nil, err
	}
	return &views[0], nil
}

// sumByCurrency totals an HQ's week per currency (never mixing currencies).
func (s *PlatformService) sumByCurrency(tx *gorm.DB, branchID uuid.UUID, start time.Time) ([]models.WeeklySettlementTotal, error) {
	var rows []models.WeeklySettlementTotal
	err := tx.Model(&models.WeeklyPlatformResult{}).
		Select("currency, SUM(win_lose) AS win_lose, SUM(commission) AS commission, SUM(adjusted_amount) AS adjusted_amount, SUM(jame_ps) AS jame_ps, COUNT(*) AS entries").
		Where("branch_id = ? AND week_start = ?", branchID, dayString(start)).
		Group("currency").Order("currency").Scan(&rows).Error
	return rows, err
}
