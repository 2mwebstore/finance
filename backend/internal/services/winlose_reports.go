package services

import (
	"context"
	"sort"
	"strings"
	"time"

	"github.com/cambodia-finance/income-expense/internal/models"
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
			Where("r.deleted_at IS NULL AND r.week_start BETWEEN ? AND ?", dayString(weekStart(from)), dayString(to))
		if branchID != nil {
			q = q.Where("r.branch_id = ?", *branchID)
		}
		if f.GroupID != nil {
			q = q.Joins("JOIN weekly_group_results gf ON gf.result_id = r.id AND gf.group_id = ?", *f.GroupID)
		}
		if f.PlatformID != nil {
			q = q.Where("r.platform_id = ?", *f.PlatformID)
		}
		if f.EntryID != nil {
			q = q.Where("r.entry_id = ?", *f.EntryID)
		}
		return q
	}
	// A result's Total is its groups' added up; for one group, its part in that group.
	rate, jame, formula := "r.commission_rate", "r.jame_ps", "r.formula_version"
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
	sort.Slice(out.Platforms, func(i, j int) bool { return agentBefore(out.Platforms[i].PlatformCode, out.Platforms[j].PlatformCode) })
	return out, nil
}

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
	sort.Slice(lines, func(i, j int) bool {
		a, b := lines[i], lines[j]
		if a.WeekStart != b.WeekStart {
			return a.WeekStart < b.WeekStart
		}
		if a.PlatformCode != b.PlatformCode {
			return agentBefore(a.PlatformCode, b.PlatformCode)
		}
		return a.Currency < b.Currency
	})

	return lines, shareRows, nameOf, nil
}

// groupTotals adds the results up by titled group, each at its part in the group (an agent in two
// groups counts in both): by title, weeks of agents in no group last.
func (s *PlatformService) groupTotals(ctx context.Context, results *gorm.DB) ([]GroupTotalsRow, error) {
	var rows []struct {
		GroupID                                                       *uuid.UUID
		Currency                                                      string
		WinLose, Commission, AdjustedAmount, JamePS, RateMin, RateMax decimal.Decimal
		FormulaMin, FormulaMax                                        string
		Entries                                                       int64
	}
	if err := results.Joins("LEFT JOIN weekly_group_results g ON g.result_id = r.id").
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
	sort.Slice(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if (a.GroupID == nil) != (b.GroupID == nil) {
			return b.GroupID == nil
		}
		return a.Title < b.Title
	})
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
	return out, nil
}

func sortedSums(m map[string]*Sums) []Sums {
	out := make([]Sums, 0, len(m))
	for _, s := range m {
		out = append(out, *s)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Currency < out[j].Currency })
	return out
}
