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
	// ResultID is the saved row's result; Reactions people's ✅ and ❌ on it.
	ResultID  *uuid.UUID      `json:"result_id,omitempty"`
	Reactions []EntryReaction `json:"reactions"`
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
	Rows       []WeekSheetRow   `json:"rows"`
}

// share is an assignment a result counts in: an agent in a group, at the group's rate for it.
type share struct {
	group      *uuid.UUID
	assignment uuid.UUID
	kind       winlose.Kind    // PO or GI: how the group splits the amount
	rate       decimal.Decimal // with partners, their % added up
	partners   []PartnerShare  // assignments (and weeks) from before groups had their own rates
	existing   *models.WeeklyGroupResult
	total      decimal.Decimal
}

// entryParts are what an entry needs: its results with their groups, and the groups each agent
// is in that week (for the agents not entered yet).
type entryParts struct {
	branch    models.Branch
	start     time.Time
	entry     *models.WeeklyEntry                         // nil: a new entry
	results   map[uuid.UUID]models.WeeklyPlatformResult   // by platform
	saved     map[uuid.UUID][]models.WeeklyGroupResult    // by result
	partnerOf map[uuid.UUID][]models.WeeklyPartnerResult  // by result
	ofAgent   map[uuid.UUID][]models.HQPlatformAssignment // by platform: the assignments it counts in
	links     map[uuid.UUID][]models.AssignmentPartner    // by assignment
	groups    map[uuid.UUID]models.AssignmentGroup
	partners  map[uuid.UUID]models.Partner
	platforms map[uuid.UUID]models.Platform // every agent that is on, and those saved
	// offOnly: agents in groups that week, all of them switched off (left out of the entry).
	offOnly map[uuid.UUID]bool
}

// loadEntry reads an entry (nil: a new one) of a branch's week. An agent counts in a group (one
// that is switched on) when it was in it on any day of the week; taken out and put in again within
// the week, the later assignment is used. An assignment made before groups counts only when the
// agent is in no group that week.
func (s *PlatformService) loadEntry(tx *gorm.DB, branchID uuid.UUID, start time.Time, entry *models.WeeklyEntry, lock bool) (*entryParts, error) {
	end := start.AddDate(0, 0, 6)
	p := &entryParts{start: start, entry: entry, results: map[uuid.UUID]models.WeeklyPlatformResult{}, saved: map[uuid.UUID][]models.WeeklyGroupResult{},
		partnerOf: map[uuid.UUID][]models.WeeklyPartnerResult{}, ofAgent: map[uuid.UUID][]models.HQPlatformAssignment{}, groups: map[uuid.UUID]models.AssignmentGroup{},
		offOnly: map[uuid.UUID]bool{}}
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
	inOff := map[uuid.UUID]bool{}                             // by agent: in a switched-off group
	for _, a := range assignments {
		if a.GroupID == nil {
			before[a.PlatformID] = a
		} else if g, ok := p.groups[*a.GroupID]; ok && g.IsActive {
			inGroup[[2]uuid.UUID{a.PlatformID, g.ID}] = a
		} else if ok {
			inOff[a.PlatformID] = true
		}
	}
	for key, a := range inGroup {
		p.ofAgent[key[0]] = append(p.ofAgent[key[0]], a)
	}
	for platformID := range inOff {
		if !hasGroup(p.ofAgent[platformID]) {
			p.offOnly[platformID] = true
		}
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
	// Only this branch's agents and the shared ones can be entered here.
	if err := tx.Where("is_active = ? AND (branch_id = ? OR branch_id IS NULL)", true, branchID).Find(&active).Error; err != nil {
		return nil, err
	}
	for _, pl := range active {
		p.platforms[pl.ID] = pl
	}
	return p, nil
}

// hasGroup: one of the assignments is in a group (not one made before groups).
func hasGroup(list []models.HQPlatformAssignment) bool {
	for _, a := range list {
		if a.GroupID != nil {
			return true
		}
	}
	return false
}

// hidden: an agent left out of the entry (and the reports): switched off, or only in switched-off
// groups (a saved result: in groups, none of them switched on; one not entered yet: in groups that
// week, all switched off). An agent in no group is listed.
func (p *entryParts) hidden(platformID uuid.UUID) bool {
	if pl, ok := p.platforms[platformID]; !ok || !pl.IsActive {
		return true
	}
	r, saved := p.results[platformID]
	if !saved {
		return p.offOnly[platformID]
	}
	inGroups, on := false, false
	for _, g := range p.saved[r.ID] {
		if g.GroupID != nil {
			inGroups = true
			on = on || p.groups[*g.GroupID].IsActive
		}
	}
	return inGroups && !on
}

// newShares are the groups an agent is in that week, each at the group's rate for it.
func (p *entryParts) newShares(platformID uuid.UUID) []share {
	var out []share
	for _, a := range p.ofAgent[platformID] {
		var partners []PartnerShare
		for _, l := range p.links[a.ID] {
			partners = append(partners, PartnerShare{PartnerID: l.PartnerID, CommissionRate: l.CommissionRate})
		}
		kind := winlose.PO
		if winlose.Kind(a.CalcType) == winlose.GI {
			kind = winlose.GI
		}
		out = append(out, share{group: a.GroupID, assignment: a.ID, kind: kind, rate: shareRate(a.CommissionRate, partners), partners: partners})
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
		sh := share{group: list[i].GroupID, assignment: list[i].AssignmentID, kind: winlose.Kind(list[i].CalcType), rate: list[i].CommissionRate, existing: &list[i]}
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
// reports without consolidated access): their default branch, else their only branch. Those pages don't ask for one.
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
	sheet := s.sheet(ctx, parts, formula)
	// The reactions on its saved agents (in the entry).
	var ids []uuid.UUID
	for _, r := range sheet.Rows {
		if r.ResultID != nil {
			ids = append(ids, r.PlatformID)
		}
	}
	reactions := map[uuid.UUID][]EntryReaction{}
	if sheet.Entry != nil && sheet.Entry.ID != uuid.Nil {
		if reactions, err = s.reactionsWhere(ctx, agentReactions, ids, map[string]interface{}{"entry_id": sheet.Entry.ID}); err != nil {
			return nil, err
		}
	}
	for i, r := range sheet.Rows {
		if r.ResultID != nil {
			sheet.Rows[i].Reactions = orNone(reactions[r.PlatformID])
		} else {
			sheet.Rows[i].Reactions = []EntryReaction{}
		}
	}
	return sheet, nil
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
		Formula: formula, Rows: []WeekSheetRow{}}
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
		if p.hidden(platformID) {
			continue
		}
		row := WeekSheetRow{PlatformID: platformID, PlatformCode: pl.Code, PlatformName: pl.Name}
		if r, ok := p.results[platformID]; ok {
			wl, cm, jp, at := r.WinLose, r.Commission, r.JamePS, r.UpdatedAt
			row.CommissionRate, row.Currency, row.FormulaVersion = r.CommissionRate, r.Currency, r.FormulaVersion
			id := r.ID
			row.Saved, row.WinLose, row.Commission, row.JamePS, row.UpdatedAt, row.ResultID = true, &wl, &cm, &jp, &at, &id
			row.Groups, row.Partners = p.groupShares(p.savedShares(r)), p.partnerShares(p.partnerOf[r.ID], nil)
			row.UpdatedBy = names[r.CreatedByID]
			if r.UpdatedByID != nil {
				row.UpdatedBy = names[*r.UpdatedByID]
			}
		} else {
			shares := p.newShares(platformID)
			row.Currency, row.FormulaVersion = pl.DefaultCurrency, formula.Version
			row.Groups, row.Partners = p.groupShares(shares), p.partnerShares(nil, shares)
			for _, sh := range shares {
				row.CommissionRate = row.CommissionRate.Add(sh.rate)
			}
		}
		out.Rows = append(out.Rows, row)
	}
	sort.Slice(out.Rows, func(i, j int) bool { return agentBefore(out.Rows[i].PlatformCode, out.Rows[j].PlatformCode) })
	return out
}

// kindOf is the kind as stored: PO unless GI.
func kindOf(k winlose.Kind) string {
	if k == winlose.GI {
		return string(winlose.GI)
	}
	return string(winlose.PO)
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
// Total their sum; without, the Total at the given %. When the rate is the agent's own share
// (Rest), a partner's share is their % and the Total what is left of 100% after all of them.
func workOut(version string, kind winlose.Kind, wl, cm, rate decimal.Decimal, shares []PartnerShare) (winlose.Result, error) {
	if len(shares) == 0 {
		return winlose.CalculateKind(version, kind, winlose.Input{WinLose: wl, Commission: cm, RatePct: rate})
	}
	if f, ok := winlose.Lookup(version); ok && f.Rest {
		total, err := winlose.Calculate(version, winlose.Input{WinLose: wl, Commission: cm, RatePct: shareRate(rate, shares)})
		if err != nil {
			return winlose.Result{}, err
		}
		for i := range shares {
			jp := total.Adjusted.Mul(shares[i].CommissionRate).Div(decimal.NewFromInt(100)).Round(0)
			shares[i].JamePS = &jp
		}
		return total, nil
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
// out. (Saved weeks follow the current formula.)
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
		if !known || p.hidden(platformID) {
			return nil, fieldError("rows", name+" was not found, is turned off or is only in switched-off groups")
		}
		if r, ok := p.results[platformID]; ok {
			r := r
			plan.existing, plan.currency, plan.shares = &r, r.Currency, p.savedShares(r)
		} else {
			plan.currency, plan.shares = pl.DefaultCurrency, p.newShares(platformID)
		}
		base, err := winlose.Calculate(version, winlose.Input{WinLose: wl, Commission: cm})
		if err != nil {
			return nil, err
		}
		plan.result = winlose.Result{Version: version, Adjusted: base.Adjusted}
		for j := range plan.shares {
			sh := &plan.shares[j]
			r, err := workOut(version, sh.kind, wl, cm, sh.rate, sh.partners)
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
		// groupShares lists the groups by title, not in the order of the shares: each Total goes to
		// its own group (an agent in two groups has one share in each).
		groups := parts.groupShares(p.shares)
		for j := range groups {
			for _, sh := range p.shares {
				if (sh.group == nil && groups[j].GroupID == nil) || (sh.group != nil && groups[j].GroupID != nil && *sh.group == *groups[j].GroupID) {
					total := sh.total
					groups[j].JamePS = &total
					break
				}
			}
		}
		out[i] = PreviewRow{PlatformID: p.platform, CommissionRate: p.rate, Currency: p.currency, Groups: groups, Result: p.result}
	}
	return out, nil
}

// SaveWeek saves an entry in one transaction: its title and its agents' results, each with its
// Total in every group the agent is in. With no entry id it makes a new entry (a new entry's first
// save); the sheet returned has its id for the saves after. A new entry and new rows need
// winlose.create; changing a saved row or the title needs winlose.edit (a saved row also a
// reason).
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
					if err := tx.Create(&models.WeeklyGroupResult{ID: uuid.New(), ResultID: row.ID, GroupID: sh.group, AssignmentID: sh.assignment, CommissionRate: sh.rate, CalcType: kindOf(sh.kind), JamePS: sh.total}).Error; err != nil {
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

// RecalculateWeeks works every saved week out again with the current formula when it was saved with
// another, so a formula change applies to it. Run at start-up. It returns how many results changed.
func (s *PlatformService) RecalculateWeeks(ctx context.Context) (int, error) {
	formula, err := s.Formula(ctx)
	if err != nil {
		return 0, err
	}
	var results []models.WeeklyPlatformResult
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("formula_version <> ?", formula.Version).
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
			res, err := workOut(formula.Version, winlose.Kind(g.CalcType), r.WinLose, r.Commission, g.CommissionRate, shares)
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

// applyCurrentRates puts saved weeks on their agents' current rates: where a result's part in a
// group is at a rate or type (PO / GI) that is no longer its assignment's, the result's parts take
// their assignments' and its Total and rate are worked out again (with the formula it was saved
// with). Results with partners (from before groups had their own rates) keep theirs. narrow limits which ones (by g, the group
// part, a, its assignment, and r, the result; nil: all). It returns how many results changed.
func applyCurrentRates(tx *gorm.DB, narrow func(*gorm.DB) *gorm.DB) (int, error) {
	q := tx.Table("weekly_group_results g").
		Joins("JOIN hq_platform_assignments a ON a.id = g.assignment_id").
		Joins("JOIN weekly_platform_results r ON r.id = g.result_id").
		Where("r.deleted_at IS NULL").
		Where("(g.commission_rate <> a.commission_rate OR g.calc_type <> (CASE WHEN a.calc_type = 'GI' THEN 'GI' ELSE 'PO' END))").
		Where("NOT EXISTS (SELECT 1 FROM weekly_partner_results wp WHERE wp.result_id = r.id)").
		Where("NOT EXISTS (SELECT 1 FROM weekly_group_results g2 JOIN assignment_partners l ON l.assignment_id = g2.assignment_id WHERE g2.result_id = r.id)")
	if narrow != nil {
		q = narrow(q)
	}
	var ids []uuid.UUID
	if err := q.Distinct("g.result_id").Pluck("g.result_id", &ids).Error; err != nil {
		return 0, err
	}
	if len(ids) == 0 {
		return 0, nil
	}
	var results []models.WeeklyPlatformResult
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id IN ?", ids).Find(&results).Error; err != nil {
		return 0, err
	}
	var parts []struct {
		ID, ResultID uuid.UUID
		Rate         decimal.Decimal
		Kind         string
	}
	if err := tx.Table("weekly_group_results g").Joins("JOIN hq_platform_assignments a ON a.id = g.assignment_id").
		Select("g.id, g.result_id, a.commission_rate AS rate, a.calc_type AS kind").Where("g.result_id IN ?", ids).Scan(&parts).Error; err != nil {
		return 0, err
	}
	partsOf := map[uuid.UUID][]int{}
	for i, p := range parts {
		partsOf[p.ResultID] = append(partsOf[p.ResultID], i)
	}
	now := time.Now()
	for _, r := range results {
		total, rate := decimal.Zero, decimal.Zero
		for _, i := range partsOf[r.ID] {
			p := parts[i]
			kind := winlose.PO
			if winlose.Kind(p.Kind) == winlose.GI {
				kind = winlose.GI
			}
			res, err := workOut(r.FormulaVersion, kind, r.WinLose, r.Commission, p.Rate, nil)
			if err != nil {
				return 0, err
			}
			if err := tx.Model(&models.WeeklyGroupResult{}).Where("id = ?", p.ID).UpdateColumns(map[string]interface{}{
				"commission_rate": p.Rate, "calc_type": string(kind), "jame_ps": res.JamePS, "updated_at": now,
			}).Error; err != nil {
				return 0, err
			}
			total, rate = total.Add(res.JamePS), rate.Add(p.Rate)
		}
		if err := tx.Model(&models.WeeklyPlatformResult{}).Where("id = ?", r.ID).UpdateColumns(map[string]interface{}{
			"commission_rate": rate, "jame_ps": total, "updated_at": now,
		}).Error; err != nil {
			return 0, err
		}
	}
	return len(results), nil
}

// ApplyCurrentRates puts every saved week on its agents' current rates (applyCurrentRates). Run at start-up, for weeks saved before a rate change that came before this.
func (s *PlatformService) ApplyCurrentRates(ctx context.Context) (int, error) {
	n := 0
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var err error
		n, err = applyCurrentRates(tx, nil)
		return err
	})
	return n, err
}

// ===== Entries =====

// DeleteEntry removes an entry and its agents' results (all or nothing). It needs a reason (audit
// log). It returns what was removed.
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
			if err := tx.Where("result_id IN ?", ids).Delete(&models.WeeklyResultReaction{}).Error; err != nil {
				return err
			}
			if err := tx.Unscoped().Where("id IN ?", ids).Delete(&models.WeeklyPlatformResult{}).Error; err != nil {
				return err
			}
		}
		if err := tx.Where("entry_id = ?", entry.ID).Delete(&models.WeeklyEntryReaction{}).Error; err != nil {
			return err
		}
		if err := tx.Where("entry_id = ?", entry.ID).Delete(&models.WeeklyGroupReaction{}).Error; err != nil {
			return err
		}
		if err := tx.Where("entry_id = ?", entry.ID).Delete(&models.WeeklyAgentReaction{}).Error; err != nil {
			return err
		}
		return tx.Delete(entry).Error
	})
	if err != nil {
		return nil, nil, err
	}
	return entry, removed, nil
}

// ===== Reactions =====

// EntryReaction is a person's ✅ (yes), ❌ (no) or 📤 (sent: the report was sent) on an entry, or on
// a group or agent in it. A person has at most one of ✅ / ❌, and 📤 beside it.
type EntryReaction struct {
	Reaction string    `json:"reaction"`
	UserID   uuid.UUID `json:"user_id"`
	UserName string    `json:"user_name"`
	At       time.Time `json:"at"`
}

// ReactToEntry sets the person's reaction on an entry they can see: the same one again takes it
// back; ✅ replaces their ❌ and the other way round, while 📤 is given or taken back on its own. It
// returns the entry, the person's reaction of that kind now ("" when taken back) and the entry's
// reactions.
func (s *PlatformService) ReactToEntry(ctx context.Context, id uuid.UUID, reaction string, scope BranchScope, actor uuid.UUID) (*models.WeeklyEntry, string, []EntryReaction, error) {
	if err := checkReaction(reaction); err != nil {
		return nil, "", nil, err
	}
	var entry *models.WeeklyEntry
	var now string
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var err error
		if entry, err = findEntry(tx, id.String(), scope, false); err != nil {
			return err
		}
		now, err = toggleReaction(tx, entryReactions, entry.ID, actor, reaction)
		return err
	})
	if err != nil {
		return nil, "", nil, err
	}
	list, err := s.reactionsOf(ctx, entryReactions, []uuid.UUID{entry.ID})
	return entry, now, orNone(list[entry.ID]), err
}

// ReactToResult sets the person's reaction on an agent's result in an entry they can see, the same
// way: it is the agent's in that entry (ReactToAgent). It returns the result, the reaction now and
// the agent's reactions in the entry.
func (s *PlatformService) ReactToResult(ctx context.Context, id uuid.UUID, reaction string, scope BranchScope, actor uuid.UUID) (*models.WeeklyPlatformResult, string, []EntryReaction, error) {
	if err := checkReaction(reaction); err != nil {
		return nil, "", nil, err
	}
	var result models.WeeklyPlatformResult
	if err := s.db.WithContext(ctx).First(&result, "id = ?", id).Error; err != nil || !scope.Has(result.BranchID) || result.EntryID == nil {
		return nil, "", nil, ErrEntryNotFound
	}
	_, _, now, list, err := s.ReactToAgent(ctx, *result.EntryID, result.PlatformID, reaction, scope, actor)
	if err != nil {
		return nil, "", nil, err
	}
	return &result, now, list, nil
}

// ReactToAgent sets the person's reaction on an agent in an entry they can see (its result there,
// or 0 when it has none, as Group summary lists it), the same way. The agent has a result in the
// entry, or is a switched-on one of the entry's branch (or shared). It returns the entry, the
// agent, the reaction now and the agent's reactions in the entry.
func (s *PlatformService) ReactToAgent(ctx context.Context, entryID, platformID uuid.UUID, reaction string, scope BranchScope, actor uuid.UUID) (*models.WeeklyEntry, *models.Platform, string, []EntryReaction, error) {
	if err := checkReaction(reaction); err != nil {
		return nil, nil, "", nil, err
	}
	var entry *models.WeeklyEntry
	var agent models.Platform
	var now string
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var err error
		if entry, err = findEntry(tx, entryID.String(), scope, false); err != nil {
			return err
		}
		if err := tx.First(&agent, "id = ?", platformID).Error; err != nil {
			return ErrPlatformNotFound
		}
		if !agent.IsActive || (agent.BranchID != nil && *agent.BranchID != entry.BranchID) {
			var n int64
			if err := tx.Model(&models.WeeklyPlatformResult{}).Where("entry_id = ? AND platform_id = ?", entry.ID, agent.ID).Count(&n).Error; err != nil {
				return err
			}
			if n == 0 {
				return ErrPlatformNotFound // not one of the entry's agents
			}
		}
		now, err = toggleReactionWhere(tx, agentReactions, map[string]interface{}{"entry_id": entry.ID, "platform_id": agent.ID}, actor, reaction)
		return err
	})
	if err != nil {
		return nil, nil, "", nil, err
	}
	list, err := s.reactionsWhere(ctx, agentReactions, []uuid.UUID{agent.ID}, map[string]interface{}{"entry_id": entry.ID})
	return entry, &agent, now, orNone(list[agent.ID]), err
}

// ReactToGroup sets the person's reaction on a group in an entry they can see (its agents' results
// there, or 0 when it has none, as Group summary lists it), the same way. The group has results in
// the entry, or is a switched-on group of the entry's branch. It returns the entry, the group, the
// reaction now and the group's reactions in the entry.
func (s *PlatformService) ReactToGroup(ctx context.Context, entryID, groupID uuid.UUID, reaction string, scope BranchScope, actor uuid.UUID) (*models.WeeklyEntry, *models.AssignmentGroup, string, []EntryReaction, error) {
	if err := checkReaction(reaction); err != nil {
		return nil, nil, "", nil, err
	}
	var entry *models.WeeklyEntry
	var group models.AssignmentGroup
	var now string
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var err error
		if entry, err = findEntry(tx, entryID.String(), scope, false); err != nil {
			return err
		}
		if err := tx.First(&group, "id = ?", groupID).Error; err != nil {
			return ErrGroupNotFound
		}
		if !group.IsActive || group.BranchID != entry.BranchID {
			var n int64
			if err := tx.Table("weekly_group_results AS g").Joins("JOIN weekly_platform_results r ON r.id = g.result_id").
				Where("r.entry_id = ? AND g.group_id = ? AND r.deleted_at IS NULL", entry.ID, group.ID).Count(&n).Error; err != nil {
				return err
			}
			if n == 0 {
				return ErrGroupNotFound // not one of the entry's groups
			}
		}
		now, err = toggleReactionWhere(tx, groupReactions, map[string]interface{}{"entry_id": entry.ID, "group_id": group.ID}, actor, reaction)
		return err
	})
	if err != nil {
		return nil, nil, "", nil, err
	}
	list, err := s.reactionsWhere(ctx, groupReactions, []uuid.UUID{group.ID}, map[string]interface{}{"entry_id": entry.ID})
	return entry, &group, now, orNone(list[group.ID]), err
}

// The reactions' tables: on entries, and on groups and agents in them (by group_id or platform_id,
// within an entry_id).
type reactionTable struct{ table, column string }

var (
	entryReactions = reactionTable{"weekly_entry_reactions", "entry_id"}
	groupReactions = reactionTable{"weekly_group_reactions", "group_id"}
	agentReactions = reactionTable{"weekly_agent_reactions", "platform_id"}
)

func checkReaction(reaction string) error {
	if reaction != models.ReactionYes && reaction != models.ReactionNo && reaction != models.ReactionSent {
		return fieldError("reaction", "react with yes (✅), no (❌) or sent (📤)")
	}
	return nil
}

// toggleReaction sets the person's reaction on id: the same one again takes it back, ✅ and ❌
// replace each other (one of them each), and 📤 is given or taken back on its own (sent_at, on the
// same row). It returns the reaction of that kind now ("" when taken back).
func toggleReaction(tx *gorm.DB, t reactionTable, id, actor uuid.UUID, reaction string) (string, error) {
	return toggleReactionWhere(tx, t, map[string]interface{}{t.column: id}, actor, reaction)
}

// toggleReactionWhere is toggleReaction on what keys names (column → value; a group's reaction is
// on its group_id in an entry_id).
func toggleReactionWhere(tx *gorm.DB, t reactionTable, keys map[string]interface{}, actor uuid.UUID, reaction string) (string, error) {
	var mine struct {
		ID       uuid.UUID
		Reaction string
		SentAt   *time.Time
	}
	q := tx.Table(t.table).Clauses(clause.Locking{Strength: "UPDATE"}).Select("id, reaction, sent_at").Where("user_id = ?", actor)
	for column, value := range keys {
		q = q.Where(column+" = ?", value)
	}
	err := q.Take(&mine).Error
	now := time.Now()
	sent := reaction == models.ReactionSent
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		row := map[string]interface{}{"id": uuid.New(), "user_id": actor, "reaction": reaction, "created_at": now, "updated_at": now}
		if sent {
			row["reaction"], row["sent_at"] = "", now // 📤 only
		}
		for column, value := range keys {
			row[column] = value
		}
		return reaction, tx.Table(t.table).Create(row).Error
	case err != nil:
		return "", err
	case sent && mine.SentAt == nil:
		return reaction, tx.Table(t.table).Where("id = ?", mine.ID).Update("sent_at", now).Error
	case sent && mine.Reaction != "":
		return "", tx.Table(t.table).Where("id = ?", mine.ID).Update("sent_at", nil).Error
	case !sent && mine.Reaction == reaction && mine.SentAt != nil:
		return "", tx.Table(t.table).Where("id = ?", mine.ID).Updates(map[string]interface{}{"reaction": "", "updated_at": now}).Error
	case sent || mine.Reaction == reaction:
		// Nothing of theirs is left on it.
		return "", tx.Exec("DELETE FROM "+t.table+" WHERE id = ?", mine.ID).Error
	default:
		return reaction, tx.Table(t.table).Where("id = ?", mine.ID).Updates(map[string]interface{}{"reaction": reaction, "updated_at": now}).Error
	}
}

// reactionsOfRow is what a person's row of a reactions table shows: their ✅ or ❌ (as last given),
// and their 📤.
func reactionsOfRow(reaction string, at time.Time, sentAt *time.Time, user uuid.UUID, name string) []EntryReaction {
	var out []EntryReaction
	if reaction != "" {
		out = append(out, EntryReaction{Reaction: reaction, UserID: user, UserName: name, At: at})
	}
	if sentAt != nil {
		out = append(out, EntryReaction{Reaction: models.ReactionSent, UserID: user, UserName: name, At: *sentAt})
	}
	return out
}

// byWhen puts each list of reactions in the order they were given (a 📤 by when it was given, not
// by its row's ✅ / ❌).
func byWhen[K comparable](lists map[K][]EntryReaction) {
	for _, list := range lists {
		sort.SliceStable(list, func(i, j int) bool { return list[i].At.Before(list[j].At) })
	}
}

// reactionsOf lists the reactions on ids by id, as they were last given.
func (s *PlatformService) reactionsOf(ctx context.Context, t reactionTable, ids []uuid.UUID) (map[uuid.UUID][]EntryReaction, error) {
	return s.reactionsWhere(ctx, t, ids, nil)
}

// reactionsWhere is reactionsOf with more to match (column → value: the entry of groups' reactions).
func (s *PlatformService) reactionsWhere(ctx context.Context, t reactionTable, ids []uuid.UUID, also map[string]interface{}) (map[uuid.UUID][]EntryReaction, error) {
	out := map[uuid.UUID][]EntryReaction{}
	if len(ids) == 0 {
		return out, nil
	}
	var rows []struct {
		Target    uuid.UUID
		UserID    uuid.UUID
		Reaction  string
		SentAt    *time.Time
		UpdatedAt time.Time
	}
	q := s.db.WithContext(ctx).Table(t.table).Select(t.column+" AS target, user_id, reaction, sent_at, updated_at").Where(t.column+" IN ?", ids)
	for column, value := range also {
		q = q.Where(column+" = ?", value)
	}
	if err := q.Order("updated_at, id").Scan(&rows).Error; err != nil {
		return nil, err
	}
	users := make([]uuid.UUID, len(rows))
	for i, r := range rows {
		users[i] = r.UserID
	}
	names := s.userNames(ctx, users)
	for _, r := range rows {
		out[r.Target] = append(out[r.Target], reactionsOfRow(r.Reaction, r.UpdatedAt, r.SentAt, r.UserID, names[r.UserID])...)
	}
	byWhen(out)
	return out, nil
}

// orNone is a list that shows as [] rather than null.
func orNone(list []EntryReaction) []EntryReaction {
	if list == nil {
		return []EntryReaction{}
	}
	return list
}

// EntryRow is one saved week on Data Entry's list.
type EntryRow struct {
	ID         uuid.UUID `json:"id"`
	Title      string    `json:"title"`
	WeekStart  string    `json:"week_start"`
	WeekEnd    string    `json:"week_end"`
	BranchID   uuid.UUID `json:"branch_id"`
	BranchCode string    `json:"branch_code"`
	BranchName string    `json:"branch_name"`
	CreatedAt  time.Time `json:"created_at"`
	CreatedBy  string    `json:"created_by"`
	// Reactions: people's ✅ and ❌ on the entry.
	Reactions []EntryReaction `json:"reactions"`
}

// EntryFilter narrows Data Entry's list: words in the title, and the days it was created on
// (YYYY-MM-DD, either may be empty).
type EntryFilter struct {
	// BranchID: only this branch's entries (it must be one the person works in); nil: every branch
	// in their scope.
	BranchID               *uuid.UUID
	Search                 string
	CreatedFrom, CreatedTo string
	// PageSize 0: the newest 500; otherwise one page of that size.
	Page, PageSize int
}

// likePattern matches text anywhere in a column (LIKE's own wildcards taken literally).
func likePattern(text string) string {
	return "%" + strings.NewReplacer(`\`, `\\`, "%", `\%`, "_", `\_`).Replace(text) + "%"
}

// ListEntries lists the entries of the groups in scope, newest first (at most 500).
func (s *PlatformService) ListEntries(ctx context.Context, f EntryFilter, scope BranchScope) ([]EntryRow, error) {
	rows, _, err := s.ListEntriesPage(ctx, f, scope)
	return rows, err
}

// ListEntriesPage lists entries newest first: one page when f.PageSize is set (with how many match
// in all), else the newest 500.
func (s *PlatformService) ListEntriesPage(ctx context.Context, f EntryFilter, scope BranchScope) ([]EntryRow, int64, error) {
	q := scope.where(s.db.WithContext(ctx).Model(&models.WeeklyEntry{}), "branch_id")
	if f.BranchID != nil {
		if !scope.Has(*f.BranchID) {
			return nil, 0, ErrBranchAccessDenied
		}
		q = q.Where("branch_id = ?", *f.BranchID)
	}
	if search := strings.TrimSpace(f.Search); search != "" {
		q = q.Where("title LIKE ?", likePattern(search))
	}
	// Created days are the business's days (its time zone), from midnight to midnight.
	if strings.TrimSpace(f.CreatedFrom) != "" {
		day, err := s.parseDay("created_from", f.CreatedFrom)
		if err != nil {
			return nil, 0, err
		}
		q = q.Where("created_at >= ?", time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, s.loc))
	}
	if strings.TrimSpace(f.CreatedTo) != "" {
		day, err := s.parseDay("created_to", f.CreatedTo)
		if err != nil {
			return nil, 0, err
		}
		q = q.Where("created_at < ?", time.Date(day.Year(), day.Month(), day.Day()+1, 0, 0, 0, 0, s.loc))
	}
	var total int64
	if err := q.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	q = q.Order("created_at DESC, id").Limit(500)
	if f.PageSize > 0 {
		q = q.Offset((max(f.Page, 1) - 1) * f.PageSize).Limit(f.PageSize)
	}
	var list []models.WeeklyEntry
	if err := q.Find(&list).Error; err != nil {
		return nil, 0, err
	}
	var branchIDs, users []uuid.UUID
	for _, e := range list {
		branchIDs, users = append(branchIDs, e.BranchID), append(users, e.CreatedByID)
	}
	branches, err := s.branchesIn(ctx, branchIDs)
	if err != nil {
		return nil, 0, err
	}
	names := s.userNames(ctx, users)
	ids := make([]uuid.UUID, len(list))
	for i, e := range list {
		ids[i] = e.ID
	}
	reactions, err := s.reactionsOf(ctx, entryReactions, ids)
	if err != nil {
		return nil, 0, err
	}
	out := make([]EntryRow, len(list))
	for i, e := range list {
		out[i] = EntryRow{ID: e.ID, Title: e.Title, WeekStart: dayString(e.WeekStart), WeekEnd: dayString(e.WeekStart.AddDate(0, 0, 6)),
			BranchID: e.BranchID, BranchCode: branches[e.BranchID].Code, BranchName: branches[e.BranchID].Name, CreatedAt: e.CreatedAt, CreatedBy: names[e.CreatedByID],
			Reactions: orNone(reactions[e.ID])}
	}
	return out, total, nil
}
