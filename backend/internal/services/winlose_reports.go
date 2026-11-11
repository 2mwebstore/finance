package services

import (
	"context"
	"sort"
	"strings"
	"time"

	"github.com/cambodia-finance/income-expense/internal/models"
	"github.com/cambodia-finance/income-expense/internal/winlose"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// Win/lose reports: the Report (weekly), the group summary and the partner statement all come from
// the weekly results added up by agent (platform) and currency (and week, for the statement), with
// each partner's share of the Total. They cover the person's own group (branch), or every group in
// their scope for people who may see consolidated totals; groups aren't shown apart. A week is in the range when its Monday is (the range's first
// day counts from its own week's Monday). Amounts are never added up across currencies: every
// total is per currency. JamePS is what the app shows as "Total".

type ReportFilter struct {
	DateFrom, DateTo string
	BranchID         *uuid.UUID
	// PartnerID: only weeks the partner has a share of, with the Total being their share.
	PartnerID *uuid.UUID
	// GroupID: only the agents of one titled group (Group Agent Assignments).
	GroupID *uuid.UUID
	// PlatformID: only one agent (its part in each of its groups is in Groups).
	PlatformID *uuid.UUID
	// ByWeek gives one line per week as well (the partner statement).
	ByWeek bool
	// ByEntry adds the data entries (Entries), each with its own lines.
	ByEntry bool
	// EntryID: only one data entry's results.
	EntryID *uuid.UUID
	// Everyone: every switched-on group and agent is listed, at 0 when it has no results in the
	// range (the Report and the Group summary). Not with PartnerID, ByWeek or EntryID.
	Everyone bool
}

// Sums are totals in one currency.
type Sums struct {
	Currency       string          `json:"currency"`
	WinLose        decimal.Decimal `json:"win_lose"`
	Commission     decimal.Decimal `json:"commission"`
	AdjustedAmount decimal.Decimal `json:"adjusted_amount"`
	JamePS         decimal.Decimal `json:"jame_ps"`
	// Entries is how many weekly results were added up.
	Entries int64 `json:"entries"`
}

func (s *Sums) add(o Sums) {
	s.WinLose, s.Commission = s.WinLose.Add(o.WinLose), s.Commission.Add(o.Commission)
	s.AdjustedAmount, s.JamePS = s.AdjustedAmount.Add(o.AdjustedAmount), s.JamePS.Add(o.JamePS)
	s.Entries += o.Entries
}

// LineShare is one partner's share of a report line (nil partner: weeks entered with no partner).
type LineShare struct {
	PartnerID   *uuid.UUID      `json:"partner_id"`
	PartnerName string          `json:"partner_name"`
	RateMin     decimal.Decimal `json:"rate_min"`
	RateMax     decimal.Decimal `json:"rate_max"`
	JamePS      decimal.Decimal `json:"jame_ps"`
	Entries     int64           `json:"entries"`
}

// ReportLine is one agent (platform) in one currency over the range (or one week), across the
// groups covered.
type ReportLine struct {
	WeekStart string `json:"week_start,omitempty"`
	// EntryID: the data entry of an entry's line (nil for results saved before entries).
	EntryID      *uuid.UUID `json:"entry_id,omitempty"`
	PlatformID   uuid.UUID  `json:"platform_id"`
	PlatformCode string     `json:"platform_code"`
	PlatformName string     `json:"platform_name"`
	// The Partner % used (the same unless it changed during the range).
	RateMin decimal.Decimal `json:"rate_min"`
	RateMax decimal.Decimal `json:"rate_max"`
	// Partners: who the Total was shared by, by name (empty when no week had partners).
	Partners []LineShare `json:"partners"`
	// Formula: the version every week of the line was worked out with ("" when they differ).
	Formula string `json:"formula"`
	// ResultID: an entry's line is the agent's one result in it (by_entry), with people's ✅ and ❌
	// on it, the same as on the agent's row in Data entry. Left out of lines that add results up.
	ResultID  *uuid.UUID      `json:"result_id,omitempty"`
	Reactions []EntryReaction `json:"reactions,omitempty"`
	Sums
}

type PlatformTotals struct {
	PlatformID   uuid.UUID `json:"platform_id"`
	PlatformCode string    `json:"platform_code"`
	PlatformName string    `json:"platform_name"`
	Totals       []Sums    `json:"totals"`
}

// PartnerTotals: a partner's weeks, with the Total being their share. A week shared by two
// partners counts for each, so these don't add up to the report's totals.
type PartnerTotals struct {
	PartnerID   *uuid.UUID `json:"partner_id"`
	PartnerName string     `json:"partner_name"`
	Totals      []Sums     `json:"totals"`
}

// GroupTotalsRow is one titled group's totals (nil: weeks of agents not in a group). For one agent
// (PlatformID), RateMin and RateMax are its rate in the group (the same unless it changed in the
// range) and Formula the version its weeks were worked out with ("" when they differ).
type GroupTotalsRow struct {
	GroupID *uuid.UUID      `json:"group_id"`
	Title   string          `json:"title"`
	RateMin decimal.Decimal `json:"rate_min"`
	RateMax decimal.Decimal `json:"rate_max"`
	Formula string          `json:"formula"`
	Totals  []Sums          `json:"totals"`
}

// ReportEntry is a data entry of a report by entry (nil ID: results saved before entries), with
// its agents' lines and its totals per currency.
type ReportEntry struct {
	ID        *uuid.UUID   `json:"id"`
	Title     string       `json:"title"`
	WeekStart string       `json:"week_start"`
	CreatedAt *time.Time   `json:"created_at"`
	CreatedBy string       `json:"created_by"`
	Totals    []Sums       `json:"totals"`
	Lines     []ReportLine `json:"lines"`
	// Reactions: people's ✅ and ❌ on the entry.
	Reactions []EntryReaction `json:"reactions"`
}

// WeekTotals is one week's totals (Monday to Sunday), per currency.
type WeekTotals struct {
	WeekStart string `json:"week_start"`
	WeekEnd   string `json:"week_end"`
	Totals    []Sums `json:"totals"`
}

type WinLoseReport struct {
	DateFrom string `json:"date_from"`
	DateTo   string `json:"date_to"`
	// Consolidated: every group in the user's scope; otherwise their own group.
	Consolidated bool             `json:"consolidated"`
	Lines        []ReportLine     `json:"lines"`
	Platforms    []PlatformTotals `json:"platforms"`
	Partners     []PartnerTotals  `json:"partners"`
	Groups       []GroupTotalsRow `json:"groups"`
	// Weeks: the totals of each week in the range that has results, earliest first.
	Weeks []WeekTotals `json:"weeks"`
	// Entries (ByEntry): the data entries in the range, by week and then as they were created.
	Entries []ReportEntry `json:"entries,omitempty"`
	Totals  []Sums        `json:"totals"`
}

// HideRates blanks every rate (and the formula they are read with) for people without View Rates
// in Reports: the Totals stay, so the report still adds up.
func (r *WinLoseReport) HideRates() {
	hide := func(lines []ReportLine) {
		for i := range lines {
			lines[i].RateMin, lines[i].RateMax, lines[i].Formula = decimal.Zero, decimal.Zero, ""
			for j := range lines[i].Partners {
				lines[i].Partners[j].RateMin, lines[i].Partners[j].RateMax = decimal.Zero, decimal.Zero
			}
		}
	}
	hide(r.Lines)
	for i := range r.Entries {
		hide(r.Entries[i].Lines)
	}
	for i := range r.Groups {
		r.Groups[i].RateMin, r.Groups[i].RateMax, r.Groups[i].Formula = decimal.Zero, decimal.Zero, ""
	}
}

type reportRow struct {
	WeekStart                                                     *time.Time
	EntryID                                                       *uuid.UUID
	PlatformID                                                    uuid.UUID
	PartnerID                                                     *uuid.UUID
	Currency                                                      string
	RateMin, RateMax, WinLose, Commission, AdjustedAmount, JamePS decimal.Decimal
	FormulaMin, FormulaMax                                        string
	Entries                                                       int64
}

func (r reportRow) key() string {
	week, entry := "", ""
	if r.WeekStart != nil {
		week = r.WeekStart.Format("2006-01-02")
	}
	if r.EntryID != nil {
		entry = r.EntryID.String()
	}
	return strings.Join([]string{week, entry, r.PlatformID.String(), r.Currency}, "|")
}

func (r reportRow) sums() Sums {
	return Sums{Currency: r.Currency, WinLose: r.WinLose, Commission: r.Commission, AdjustedAmount: r.AdjustedAmount, JamePS: r.JamePS, Entries: r.Entries}
}

// Report adds up results over a range. Without consolidated access it covers one group: the one
// asked for (the API still takes branch_id), else the person's own.
func (s *PlatformService) Report(ctx context.Context, f ReportFilter, scope BranchScope, consolidated bool, actor uuid.UUID) (*WinLoseReport, error) {
	from, err := s.parseDay("date_from", f.DateFrom)
	if err != nil {
		return nil, err
	}
	to, err := s.parseDay("date_to", f.DateTo)
	if err != nil {
		return nil, err
	}
	if to.Before(from) {
		return nil, fieldError("date_to", "the end date is before the start date")
	}
	if to.Sub(from) > 366*24*time.Hour {
		return nil, fieldError("date_to", "choose at most a year")
	}
	branchID := f.BranchID
	if branchID != nil && !scope.Has(*branchID) {
		return nil, ErrBranchAccessDenied
	}
	if branchID == nil && !consolidated {
		own, err := s.OwnGroup(ctx, actor, scope)
		if err != nil {
			return nil, err
		}
		branchID = &own
	}

	group := "r.platform_id, r.currency"
	if f.ByWeek {
		group = "r.week_start, " + group
	}
	results := func() *gorm.DB {
		q := scope.where(s.db.WithContext(ctx).Table("weekly_platform_results r"), "r.branch_id").
			Where("r.deleted_at IS NULL AND r.week_start BETWEEN ? AND ?", dayString(weekStart(from)), dayString(to)).
			Where(shownResult)
		if branchID != nil {
			q = q.Where("r.branch_id = ?", *branchID)
		}
		if f.GroupID != nil {
			// A switched-off group shows nothing.
			q = q.Joins("JOIN weekly_group_results gf ON gf.result_id = r.id AND gf.group_id = ?", *f.GroupID).
				Joins("JOIN assignment_groups gfa ON gfa.id = gf.group_id AND gfa.is_active = 1")
		}
		if f.PlatformID != nil {
			q = q.Where("r.platform_id = ?", *f.PlatformID)
		}
		if f.EntryID != nil {
			q = q.Where("r.entry_id = ?", *f.EntryID)
		}
		return q
	}
	// A result's Total is its switched-on groups' parts added up; for one group, its part in that
	// group.
	rate, jame, formula := shownRate, shownTotal, "r.formula_version"
	if f.GroupID != nil {
		rate, jame, formula = "gf.commission_rate", "gf.jame_ps", kindFormula("gf")
	}
	lines, shareRows, nameOf, err := s.reportLines(ctx, results, group, rate, jame, formula, f.PartnerID)
	if err != nil {
		return nil, err
	}

	out := &WinLoseReport{DateFrom: dayString(weekStart(from)), DateTo: dayString(to), Consolidated: branchID == nil, Lines: lines}
	if out.Lines == nil {
		out.Lines = []ReportLine{}
	}
	plats := map[uuid.UUID]*PlatformTotals{}
	total := map[string]*Sums{}
	platSums := map[uuid.UUID]map[string]*Sums{}
	bump := func(m map[string]*Sums, add Sums) {
		if m[add.Currency] == nil {
			m[add.Currency] = &Sums{Currency: add.Currency}
		}
		m[add.Currency].add(add)
	}
	for _, l := range lines {
		if plats[l.PlatformID] == nil {
			plats[l.PlatformID] = &PlatformTotals{PlatformID: l.PlatformID, PlatformCode: l.PlatformCode, PlatformName: l.PlatformName}
			platSums[l.PlatformID] = map[string]*Sums{}
		}
		bump(platSums[l.PlatformID], l.Sums)
		bump(total, l.Sums)
	}
	parts := map[string]*PartnerTotals{} // "" is no partner
	partSums := map[string]map[string]*Sums{}
	for _, r := range shareRows {
		key := ""
		if r.PartnerID != nil {
			key = r.PartnerID.String()
		}
		if parts[key] == nil {
			parts[key] = &PartnerTotals{PartnerID: r.PartnerID, PartnerName: nameOf(r.PartnerID)}
			partSums[key] = map[string]*Sums{}
		}
		bump(partSums[key], r.sums())
	}
	out.Platforms, out.Partners, out.Totals = []PlatformTotals{}, []PartnerTotals{}, sortedSums(total)
	if out.Groups, err = s.groupTotals(ctx, results()); err != nil {
		return nil, err
	}
	if out.Weeks, err = weekTotals(results(), jame); err != nil {
		return nil, err
	}
	if f.ByEntry {
		entryLines, _, _, err := s.reportLines(ctx, results, "r.entry_id, "+group, rate, jame, formula, f.PartnerID)
		if err != nil {
			return nil, err
		}
		if err := s.withResultReactions(ctx, results, entryLines); err != nil {
			return nil, err
		}
		if out.Entries, err = s.reportEntries(ctx, entryLines); err != nil {
			return nil, err
		}
	}
	for key, p := range parts {
		p.Totals = sortedSums(partSums[key])
		out.Partners = append(out.Partners, *p)
	}
	// Partners by name; weeks with no partner last.
	sort.Slice(out.Partners, func(i, j int) bool {
		a, b := out.Partners[i], out.Partners[j]
		if (a.PartnerID == nil) != (b.PartnerID == nil) {
			return b.PartnerID == nil
		}
		return a.PartnerName < b.PartnerName
	})
	for id, p := range plats {
		p.Totals = sortedSums(platSums[id])
		out.Platforms = append(out.Platforms, *p)
	}
	if f.Everyone && f.PartnerID == nil && !f.ByWeek && f.EntryID == nil {
		if err := s.addEveryone(ctx, out, f, from, to, branchID, scope); err != nil {
			return nil, err
		}
	}
	sort.SliceStable(out.Platforms, func(i, j int) bool { return agentBefore(out.Platforms[i].PlatformCode, out.Platforms[j].PlatformCode) })
	return out, nil
}

// addEveryone lists the switched-on groups and agents that have no results in the range, at 0 in
// their currency: the agents Data entry lists for the branches covered (the branch's own and the
// shared ones, switched on and not only in switched-off groups) and those branches' switched-on
// groups. For one group, its agents (each at its rate there); for one agent, the groups it is in.
// The totals stay what was entered.
func (s *PlatformService) addEveryone(ctx context.Context, out *WinLoseReport, f ReportFilter, from, to time.Time, branchID *uuid.UUID, scope BranchScope) error {
	db := s.db.WithContext(ctx)
	inBranches := func(q *gorm.DB) *gorm.DB {
		if branchID != nil {
			return q.Where("branch_id = ?", *branchID)
		}
		return scope.where(q, "branch_id")
	}
	var groups []models.AssignmentGroup
	if err := inBranches(db.Model(&models.AssignmentGroup{})).Find(&groups).Error; err != nil {
		return err
	}
	groupOf := make(map[uuid.UUID]models.AssignmentGroup, len(groups))
	for _, g := range groups {
		groupOf[g.ID] = g
	}
	// The agents' places in those groups during the range.
	var assignments []models.HQPlatformAssignment
	if err := inBranches(db.Where("group_id IS NOT NULL AND starts_on <= ? AND (ends_on IS NULL OR ends_on >= ?)", dayString(to), dayString(weekStart(from)))).
		Order("starts_on").Find(&assignments).Error; err != nil {
		return err
	}
	agents := db.Where("is_active = ?", true)
	switch {
	case branchID != nil:
		agents = agents.Where("(branch_id = ? OR branch_id IS NULL)", *branchID)
	case !scope.All && len(scope.IDs) > 0:
		agents = agents.Where("(branch_id IN ? OR branch_id IS NULL)", scope.IDs)
	case !scope.All:
		agents = agents.Where("branch_id IS NULL")
	}
	var list []models.Platform
	if err := agents.Find(&list).Error; err != nil {
		return err
	}
	platformOf := make(map[uuid.UUID]models.Platform, len(list))
	for _, p := range list {
		platformOf[p.ID] = p
	}
	// An agent whose groups in the range are all switched off is left out, as in Data entry.
	on, off := map[uuid.UUID]bool{}, map[uuid.UUID]bool{}
	inGroup := map[[2]uuid.UUID][]models.HQPlatformAssignment{} // by group and agent, switched-on groups only
	members := map[uuid.UUID][]uuid.UUID{}                      // a switched-on group's agents
	for _, a := range assignments {
		g, ok := groupOf[*a.GroupID]
		switch {
		case !ok:
		case g.IsActive:
			on[a.PlatformID] = true
			key := [2]uuid.UUID{g.ID, a.PlatformID}
			if len(inGroup[key]) == 0 {
				members[g.ID] = append(members[g.ID], a.PlatformID)
			}
			inGroup[key] = append(inGroup[key], a)
		default:
			off[a.PlatformID] = true
		}
	}
	shown := func(id uuid.UUID) bool {
		_, ok := platformOf[id]
		return ok && (on[id] || !off[id])
	}

	// An agent's rate in a group (the lowest and highest over the range) and the formula its Total
	// would be worked out with.
	var ids []uuid.UUID
	for _, as := range inGroup {
		for _, a := range as {
			ids = append(ids, a.ID)
		}
	}
	links, err := assignmentPartners(db, ids)
	if err != nil {
		return err
	}
	formula, err := s.Formula(ctx)
	if err != nil {
		return err
	}
	rateIn := func(groupID, platformID uuid.UUID) (lo, hi decimal.Decimal, kind string) {
		for i, a := range inGroup[[2]uuid.UUID{groupID, platformID}] {
			var partners []PartnerShare
			for _, l := range links[a.ID] {
				partners = append(partners, PartnerShare{PartnerID: l.PartnerID, CommissionRate: l.CommissionRate})
			}
			rate := shareRate(a.CommissionRate, partners)
			if i == 0 {
				lo, hi = rate, rate
			}
			lo, hi = decimal.Min(lo, rate), decimal.Max(hi, rate)
			k := formula.Version
			if winlose.Kind(a.CalcType) == winlose.GI {
				k = "gi"
			}
			if i == 0 {
				kind = k
			} else if kind != k {
				kind = ""
			}
		}
		return lo, hi, kind
	}

	// Agents: the group's, the one asked for, or every one shown.
	var wanted []uuid.UUID
	switch {
	case f.GroupID != nil:
		if g, ok := groupOf[*f.GroupID]; ok && g.IsActive {
			for _, id := range members[g.ID] {
				if f.PlatformID == nil || id == *f.PlatformID {
					wanted = append(wanted, id)
				}
			}
		}
	case f.PlatformID != nil:
		wanted = []uuid.UUID{*f.PlatformID}
	default:
		for _, p := range list {
			wanted = append(wanted, p.ID)
		}
	}
	listed := map[uuid.UUID]bool{}
	for _, l := range out.Lines {
		listed[l.PlatformID] = true
	}
	for _, id := range wanted {
		if listed[id] || !shown(id) {
			continue
		}
		listed[id] = true
		p := platformOf[id]
		l := ReportLine{PlatformID: id, PlatformCode: p.Code, PlatformName: p.Name, Partners: []LineShare{}, Sums: Sums{Currency: p.DefaultCurrency}}
		if f.GroupID != nil {
			l.RateMin, l.RateMax, l.Formula = rateIn(*f.GroupID, id)
		}
		out.Lines = append(out.Lines, l)
		out.Platforms = append(out.Platforms, PlatformTotals{PlatformID: id, PlatformCode: p.Code, PlatformName: p.Name, Totals: []Sums{{Currency: p.DefaultCurrency}}})
	}
	sortLines(out.Lines)

	// Groups (not for one group): the agent's, or every switched-on one.
	if f.GroupID != nil {
		return nil
	}
	has := map[uuid.UUID]int{}
	for i, g := range out.Groups {
		if g.GroupID != nil {
			has[*g.GroupID] = i
		}
	}
	for _, g := range groups {
		if !g.IsActive {
			continue
		}
		if i, ok := has[g.ID]; ok {
			// A group with results in one currency is at 0 in its other agents' currencies too
			// (results in KHR and USD agents with none: it is in the USD table as well).
			if f.PlatformID == nil {
				out.Groups[i].Totals = withZeroSums(out.Groups[i].Totals, agentCurrencies(members[g.ID], platformOf))
			}
			continue
		}
		id := g.ID
		row := GroupTotalsRow{GroupID: &id, Title: g.Title}
		if f.PlatformID != nil {
			if len(inGroup[[2]uuid.UUID{g.ID, *f.PlatformID}]) == 0 || !shown(*f.PlatformID) {
				continue
			}
			row.RateMin, row.RateMax, row.Formula = rateIn(g.ID, *f.PlatformID)
			row.Totals = []Sums{{Currency: platformOf[*f.PlatformID].DefaultCurrency}}
		} else {
			row.Totals = zeroSums(groupCurrencies(g, members[g.ID], platformOf))
		}
		out.Groups = append(out.Groups, row)
	}
	sortGroups(out.Groups)
	return nil
}

// groupCurrencies are the currencies a group's agents report in (USD first); a group with no
// agents: its own currency, else USD.
func groupCurrencies(g models.AssignmentGroup, agents []uuid.UUID, platformOf map[uuid.UUID]models.Platform) []string {
	if out := agentCurrencies(agents, platformOf); len(out) > 0 {
		return out
	}
	if g.Currency != "" {
		return []string{g.Currency}
	}
	return []string{models.CurrencyUSD}
}

// agentCurrencies are the currencies the agents report in (USD first); none for no agents.
func agentCurrencies(agents []uuid.UUID, platformOf map[uuid.UUID]models.Platform) []string {
	seen := map[string]bool{}
	var out []string
	for _, id := range agents {
		if p, ok := platformOf[id]; ok && !seen[p.DefaultCurrency] {
			seen[p.DefaultCurrency] = true
			out = append(out, p.DefaultCurrency)
		}
	}
	sort.Slice(out, func(i, j int) bool { return currencyBefore(out[i], out[j]) })
	return out
}

func zeroSums(currencies []string) []Sums {
	out := make([]Sums, len(currencies))
	for i, c := range currencies {
		out[i] = Sums{Currency: c}
	}
	return out
}

// withZeroSums adds a 0 for each currency the sums don't have yet (USD first).
func withZeroSums(sums []Sums, currencies []string) []Sums {
	have := map[string]bool{}
	for _, s := range sums {
		have[s.Currency] = true
	}
	for _, c := range currencies {
		if !have[c] {
			have[c] = true
			sums = append(sums, Sums{Currency: c})
		}
	}
	sort.SliceStable(sums, func(i, j int) bool { return currencyBefore(sums[i].Currency, sums[j].Currency) })
	return sums
}

// currencyBefore puts USD before KHR (any other currency after them, A to Z): the order of every
// report's currencies.
func currencyBefore(a, b string) bool {
	rank := func(c string) int {
		switch c {
		case models.CurrencyUSD:
			return 0
		case models.CurrencyKHR:
			return 1
		}
		return 2
	}
	if x, y := rank(a), rank(b); x != y {
		return x < y
	}
	return a < b
}

// sortLines orders lines by week, then agent A to Z, then currency (USD first).
func sortLines(lines []ReportLine) {
	sort.SliceStable(lines, func(i, j int) bool {
		a, b := lines[i], lines[j]
		if a.WeekStart != b.WeekStart {
			return a.WeekStart < b.WeekStart
		}
		if a.PlatformCode != b.PlatformCode {
			return agentBefore(a.PlatformCode, b.PlatformCode)
		}
		return currencyBefore(a.Currency, b.Currency)
	})
}

// sortGroups orders groups by title A to Z (ignoring case), weeks of agents in no group last.
func sortGroups(groups []GroupTotalsRow) {
	sort.SliceStable(groups, func(i, j int) bool {
		a, b := groups[i], groups[j]
		if (a.GroupID == nil) != (b.GroupID == nil) {
			return b.GroupID == nil
		}
		return agentBefore(a.Title, b.Title)
	})
}

// Switched off: an agent, and a group with everything in it, are left out of the reports (and
// Data entry) until switched on again; nothing is deleted. A result is shown when its agent is on
// and it is in no group or in a switched-on one, and its Total and rate are its switched-on groups'
// parts (a result in no group: its own). Settlements still add up every result of the week.
const (
	shownResult = "r.platform_id IN (SELECT id FROM platforms WHERE is_active = 1) AND (" +
		"NOT EXISTS (SELECT 1 FROM weekly_group_results vg WHERE vg.result_id = r.id AND vg.group_id IS NOT NULL) OR " +
		"EXISTS (SELECT 1 FROM weekly_group_results vg JOIN assignment_groups vag ON vag.id = vg.group_id WHERE vg.result_id = r.id AND vag.is_active = 1))"
	shownTotal = "COALESCE((SELECT SUM(vg.jame_ps) FROM weekly_group_results vg JOIN assignment_groups vag ON vag.id = vg.group_id " +
		"WHERE vg.result_id = r.id AND vag.is_active = 1), r.jame_ps)"
	shownRate = "COALESCE((SELECT SUM(vg.commission_rate) FROM weekly_group_results vg JOIN assignment_groups vag ON vag.id = vg.group_id " +
		"WHERE vg.result_id = r.id AND vag.is_active = 1), r.commission_rate)"
)

// kindFormula is the SQL for the formula a group's part was worked out with: "gi" for a GI group,
// else the result's version. The report shows the rate the Total is at from it (utils/winlose.ts).
func kindFormula(alias string) string {
	return "CASE WHEN " + alias + ".calc_type = 'GI' THEN 'gi' ELSE r.formula_version END"
}

// reportLines adds results up by agent and currency (after the columns before them in group): the
// lines, the partners' share rows and the partners' names. rate and jame are the rate's and the
// Total's columns; with partnerID, the lines are that partner's shares.
func (s *PlatformService) reportLines(ctx context.Context, results func() *gorm.DB, group, rate, jame, formula string, partnerID *uuid.UUID) ([]ReportLine, []reportRow, func(*uuid.UUID) string, error) {
	// Each platform's weeks, and each partner's share of them (no partner: weeks entered without
	// partners, at the platform's own %).
	var lineRows, shareRows []reportRow
	if partnerID == nil {
		if err := results().Select(group + ", MIN(" + rate + ") AS rate_min, MAX(" + rate + ") AS rate_max, MIN(" + formula + ") AS formula_min, MAX(" + formula + ") AS formula_max, " +
			"SUM(r.win_lose) AS win_lose, SUM(r.commission) AS commission, SUM(r.adjusted_amount) AS adjusted_amount, SUM(" + jame + ") AS jame_ps, COUNT(*) AS entries").
			Group(group).Scan(&lineRows).Error; err != nil {
			return nil, nil, nil, err
		}
	}
	shares := results().Joins("LEFT JOIN weekly_partner_results s ON s.result_id = r.id")
	if partnerID != nil {
		shares = shares.Where("s.partner_id = ?", *partnerID)
	}
	if err := shares.Select(group + ", s.partner_id, MIN(COALESCE(s.commission_rate, " + rate + ")) AS rate_min, MAX(COALESCE(s.commission_rate, " + rate + ")) AS rate_max, " +
		"SUM(r.win_lose) AS win_lose, SUM(r.commission) AS commission, SUM(r.adjusted_amount) AS adjusted_amount, SUM(COALESCE(s.jame_ps, " + jame + ")) AS jame_ps, COUNT(*) AS entries").
		Group(group + ", s.partner_id").Scan(&shareRows).Error; err != nil {
		return nil, nil, nil, err
	}
	if partnerID != nil {
		// The partner's statement: their share is the line.
		lineRows = shareRows
	}

	var platformIDs, partnerIDs []uuid.UUID
	for _, r := range lineRows {
		platformIDs = append(platformIDs, r.PlatformID)
	}
	for _, r := range shareRows {
		if r.PartnerID != nil {
			partnerIDs = append(partnerIDs, *r.PartnerID)
		}
	}
	platforms, err := s.platformsIn(ctx, platformIDs)
	if err != nil {
		return nil, nil, nil, err
	}
	partners, err := s.partnersIn(ctx, partnerIDs)
	if err != nil {
		return nil, nil, nil, err
	}
	nameOf := func(id *uuid.UUID) string {
		if id == nil {
			return ""
		}
		return partners[*id].Name
	}
	lineShares := map[string][]LineShare{}
	for _, r := range shareRows {
		lineShares[r.key()] = append(lineShares[r.key()], LineShare{PartnerID: r.PartnerID, PartnerName: nameOf(r.PartnerID), RateMin: r.RateMin, RateMax: r.RateMax, JamePS: r.JamePS, Entries: r.Entries})
	}
	lines := make([]ReportLine, len(lineRows))
	for i, r := range lineRows {
		p := platforms[r.PlatformID]
		l := ReportLine{EntryID: r.EntryID, PlatformID: r.PlatformID, PlatformCode: p.Code, PlatformName: p.Name,
			RateMin: r.RateMin, RateMax: r.RateMax, Partners: []LineShare{}, Sums: r.sums()}
		if r.FormulaMin == r.FormulaMax {
			l.Formula = r.FormulaMin
		}
		if r.WeekStart != nil {
			l.WeekStart = r.WeekStart.Format("2006-01-02")
		}
		// Partners only when some week had them; then weeks without any show as "no partner".
		if list := lineShares[r.key()]; len(list) > 1 || (len(list) == 1 && list[0].PartnerID != nil) {
			sort.Slice(list, func(i, j int) bool {
				if (list[i].PartnerID == nil) != (list[j].PartnerID == nil) {
					return list[j].PartnerID == nil
				}
				return list[i].PartnerName < list[j].PartnerName
			})
			l.Partners = list
		}
		lines[i] = l
	}
	sortLines(lines)

	return lines, shareRows, nameOf, nil
}

// groupTotals adds the results up by titled group, each at its part in the group (an agent in two
// groups counts in both): by title A to Z, weeks of agents in no group last. Switched-off groups are left
// out.
func (s *PlatformService) groupTotals(ctx context.Context, results *gorm.DB) ([]GroupTotalsRow, error) {
	var rows []struct {
		GroupID                                                       *uuid.UUID
		Currency                                                      string
		WinLose, Commission, AdjustedAmount, JamePS, RateMin, RateMax decimal.Decimal
		FormulaMin, FormulaMax                                        string
		Entries                                                       int64
	}
	if err := results.Joins("LEFT JOIN weekly_group_results g ON g.result_id = r.id").
		Joins("LEFT JOIN assignment_groups ga ON ga.id = g.group_id").Where("g.group_id IS NULL OR ga.is_active = 1").
		Select("g.group_id, r.currency, SUM(r.win_lose) AS win_lose, SUM(r.commission) AS commission, SUM(r.adjusted_amount) AS adjusted_amount, SUM(COALESCE(g.jame_ps, r.jame_ps)) AS jame_ps, " +
			"COALESCE(MIN(g.commission_rate), 0) AS rate_min, COALESCE(MAX(g.commission_rate), 0) AS rate_max, MIN(" + kindFormula("g") + ") AS formula_min, MAX(" + kindFormula("g") + ") AS formula_max, COUNT(*) AS entries").
		Group("g.group_id, r.currency").Scan(&rows).Error; err != nil {
		return nil, err
	}
	var ids []uuid.UUID
	for _, r := range rows {
		if r.GroupID != nil {
			ids = append(ids, *r.GroupID)
		}
	}
	titles := map[uuid.UUID]string{}
	if len(ids) > 0 {
		var groups []models.AssignmentGroup
		if err := s.db.WithContext(ctx).Where("id IN ?", ids).Find(&groups).Error; err != nil {
			return nil, err
		}
		for _, g := range groups {
			titles[g.ID] = g.Title
		}
	}
	byGroup := map[string]*GroupTotalsRow{}
	sums := map[string]map[string]*Sums{}
	for _, r := range rows {
		key := ""
		if r.GroupID != nil {
			key = r.GroupID.String()
		}
		g := byGroup[key]
		if g == nil {
			g = &GroupTotalsRow{GroupID: r.GroupID, RateMin: r.RateMin, RateMax: r.RateMax, Formula: r.FormulaMin}
			if r.GroupID != nil {
				g.Title = titles[*r.GroupID]
			}
			byGroup[key] = g
			sums[key] = map[string]*Sums{}
		}
		g.RateMin, g.RateMax = decimal.Min(g.RateMin, r.RateMin), decimal.Max(g.RateMax, r.RateMax)
		if r.FormulaMin != r.FormulaMax || r.FormulaMin != g.Formula {
			g.Formula = ""
		}
		sums[key][r.Currency] = &Sums{Currency: r.Currency, WinLose: r.WinLose, Commission: r.Commission, AdjustedAmount: r.AdjustedAmount, JamePS: r.JamePS, Entries: r.Entries}
	}
	out := make([]GroupTotalsRow, 0, len(byGroup))
	for key, g := range byGroup {
		g.Totals = sortedSums(sums[key])
		out = append(out, *g)
	}
	sortGroups(out)
	return out, nil
}

// reportEntries lists the data entries of lines by entry, each with its lines and totals: by week,
// then as they were created; results saved before entries last.
func (s *PlatformService) reportEntries(ctx context.Context, lines []ReportLine) ([]ReportEntry, error) {
	var ids []uuid.UUID
	sums := map[string]map[string]*Sums{}
	linesOf := map[string][]ReportLine{}
	for _, l := range lines {
		key := ""
		if l.EntryID != nil {
			key = l.EntryID.String()
		}
		linesOf[key] = append(linesOf[key], l)
		if sums[key] == nil {
			sums[key] = map[string]*Sums{}
			if l.EntryID != nil {
				ids = append(ids, *l.EntryID)
			}
		}
		if sums[key][l.Currency] == nil {
			sums[key][l.Currency] = &Sums{Currency: l.Currency}
		}
		sums[key][l.Currency].add(l.Sums)
	}
	var list []models.WeeklyEntry
	if len(ids) > 0 {
		if err := s.db.WithContext(ctx).Where("id IN ?", ids).Order("week_start, created_at, id").Find(&list).Error; err != nil {
			return nil, err
		}
	}
	users := make([]uuid.UUID, len(list))
	for i, e := range list {
		users[i] = e.CreatedByID
	}
	names := s.userNames(ctx, users)
	reactions, err := s.reactionsOf(ctx, entryReactions, ids)
	if err != nil {
		return nil, err
	}
	out := make([]ReportEntry, 0, len(sums))
	for _, e := range list {
		id, created := e.ID, e.CreatedAt
		out = append(out, ReportEntry{ID: &id, Title: e.Title, WeekStart: dayString(e.WeekStart), CreatedAt: &created,
			CreatedBy: names[e.CreatedByID], Totals: sortedSums(sums[id.String()]), Lines: linesOf[id.String()], Reactions: orNone(reactions[id])})
	}
	if none, ok := sums[""]; ok {
		out = append(out, ReportEntry{Title: "Saved before entries", Totals: sortedSums(none), Lines: linesOf[""], Reactions: []EntryReaction{}})
	}
	return out, nil
}

// withResultReactions gives each line of a data entry its result (an agent has one in an entry)
// and people's ✅ and ❌ on it. Lines of results saved before entries can add up several weeks, so
// they get none.
func (s *PlatformService) withResultReactions(ctx context.Context, results func() *gorm.DB, lines []ReportLine) error {
	var rows []struct {
		ID, EntryID, PlatformID uuid.UUID
	}
	if err := results().Where("r.entry_id IS NOT NULL").Select("r.id, r.entry_id, r.platform_id").Scan(&rows).Error; err != nil {
		return err
	}
	resultOf := make(map[string]uuid.UUID, len(rows))
	ids := make([]uuid.UUID, len(rows))
	for i, r := range rows {
		resultOf[r.EntryID.String()+"|"+r.PlatformID.String()] = r.ID
		ids[i] = r.ID
	}
	reactions, err := s.reactionsOf(ctx, resultReactions, ids)
	if err != nil {
		return err
	}
	for i := range lines {
		l := &lines[i]
		if l.EntryID == nil || l.Entries != 1 {
			continue
		}
		if id, ok := resultOf[l.EntryID.String()+"|"+l.PlatformID.String()]; ok {
			l.ResultID, l.Reactions = &id, orNone(reactions[id])
		}
	}
	return nil
}

// weekTotals adds the results up by week and currency (jame: the Total's column).
func weekTotals(results *gorm.DB, jame string) ([]WeekTotals, error) {
	var rows []reportRow
	if err := results.Select("r.week_start, r.currency, SUM(r.win_lose) AS win_lose, SUM(r.commission) AS commission, SUM(r.adjusted_amount) AS adjusted_amount, SUM(" + jame + ") AS jame_ps, COUNT(*) AS entries").
		Group("r.week_start, r.currency").Order("r.week_start, r.currency").Scan(&rows).Error; err != nil {
		return nil, err
	}
	out := []WeekTotals{}
	for _, r := range rows {
		if r.WeekStart == nil {
			continue
		}
		if day := dayString(*r.WeekStart); len(out) == 0 || out[len(out)-1].WeekStart != day {
			out = append(out, WeekTotals{WeekStart: day, WeekEnd: dayString(r.WeekStart.AddDate(0, 0, 6))})
		}
		out[len(out)-1].Totals = append(out[len(out)-1].Totals, r.sums())
	}
	for _, w := range out {
		sort.SliceStable(w.Totals, func(i, j int) bool { return currencyBefore(w.Totals[i].Currency, w.Totals[j].Currency) })
	}
	return out, nil
}

func sortedSums(m map[string]*Sums) []Sums {
	out := make([]Sums, 0, len(m))
	for _, s := range m {
		out = append(out, *s)
	}
	sort.Slice(out, func(i, j int) bool { return currencyBefore(out[i].Currency, out[j].Currency) })
	return out
}
