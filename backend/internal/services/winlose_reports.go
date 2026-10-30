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

// Win/lose reports: the weekly report, the group summary and the partner statement all come from
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
	// ByWeek gives one line per week as well (the partner statement).
	ByWeek bool
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
	WeekStart    string    `json:"week_start,omitempty"`
	PlatformID   uuid.UUID `json:"platform_id"`
	PlatformCode string    `json:"platform_code"`
	PlatformName string    `json:"platform_name"`
	// The Partner % used (the same unless it changed during the range).
	RateMin decimal.Decimal `json:"rate_min"`
	RateMax decimal.Decimal `json:"rate_max"`
	// Partners: who the Total was shared by, by name (empty when no week had partners).
	Partners []LineShare `json:"partners"`
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

// GroupTotalsRow is one titled group's totals (nil: weeks of agents not in a group).
type GroupTotalsRow struct {
	GroupID *uuid.UUID `json:"group_id"`
	Title   string     `json:"title"`
	Totals  []Sums     `json:"totals"`
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
	Totals       []Sums           `json:"totals"`
}

type reportRow struct {
	WeekStart                                                     *time.Time
	PlatformID                                                    uuid.UUID
	PartnerID                                                     *uuid.UUID
	Currency                                                      string
	RateMin, RateMax, WinLose, Commission, AdjustedAmount, JamePS decimal.Decimal
	Entries                                                       int64
}

func (r reportRow) key() string {
	week := ""
	if r.WeekStart != nil {
		week = r.WeekStart.Format("2006-01-02")
	}
	return strings.Join([]string{week, r.PlatformID.String(), r.Currency}, "|")
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
			q = q.Where("r.assignment_id IN (SELECT id FROM hq_platform_assignments WHERE group_id = ?)", *f.GroupID)
		}
		return q
	}
	// Each platform's weeks, and each partner's share of them (no partner: weeks entered without
	// partners, at the platform's own %).
	var lineRows, shareRows []reportRow
	if f.PartnerID == nil {
		if err := results().Select(group + ", MIN(r.commission_rate) AS rate_min, MAX(r.commission_rate) AS rate_max, " +
			"SUM(r.win_lose) AS win_lose, SUM(r.commission) AS commission, SUM(r.adjusted_amount) AS adjusted_amount, SUM(r.jame_ps) AS jame_ps, COUNT(*) AS entries").
			Group(group).Scan(&lineRows).Error; err != nil {
			return nil, err
		}
	}
	shares := results().Joins("LEFT JOIN weekly_partner_results s ON s.result_id = r.id")
	if f.PartnerID != nil {
		shares = shares.Where("s.partner_id = ?", *f.PartnerID)
	}
	if err := shares.Select(group + ", s.partner_id, MIN(COALESCE(s.commission_rate, r.commission_rate)) AS rate_min, MAX(COALESCE(s.commission_rate, r.commission_rate)) AS rate_max, " +
		"SUM(r.win_lose) AS win_lose, SUM(r.commission) AS commission, SUM(r.adjusted_amount) AS adjusted_amount, SUM(COALESCE(s.jame_ps, r.jame_ps)) AS jame_ps, COUNT(*) AS entries").
		Group(group + ", s.partner_id").Scan(&shareRows).Error; err != nil {
		return nil, err
	}
	if f.PartnerID != nil {
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
		return nil, err
	}
	partners, err := s.partnersIn(ctx, partnerIDs)
	if err != nil {
		return nil, err
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
		l := ReportLine{PlatformID: r.PlatformID, PlatformCode: p.Code, PlatformName: p.Name,
			RateMin: r.RateMin, RateMax: r.RateMax, Partners: []LineShare{}, Sums: r.sums()}
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
			return a.PlatformCode < b.PlatformCode
		}
		return a.Currency < b.Currency
	})

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
	sort.Slice(out.Platforms, func(i, j int) bool { return out.Platforms[i].PlatformCode < out.Platforms[j].PlatformCode })
	return out, nil
}

// groupTotals adds the results up by the titled group of the assignment each was entered under:
// by title, weeks of agents in no group last.
func (s *PlatformService) groupTotals(ctx context.Context, results *gorm.DB) ([]GroupTotalsRow, error) {
	var rows []struct {
		GroupID                                     *uuid.UUID
		Currency                                    string
		WinLose, Commission, AdjustedAmount, JamePS decimal.Decimal
		Entries                                     int64
	}
	if err := results.Joins("JOIN hq_platform_assignments a ON a.id = r.assignment_id").
		Select("a.group_id, r.currency, SUM(r.win_lose) AS win_lose, SUM(r.commission) AS commission, SUM(r.adjusted_amount) AS adjusted_amount, SUM(r.jame_ps) AS jame_ps, COUNT(*) AS entries").
		Group("a.group_id, r.currency").Scan(&rows).Error; err != nil {
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
		if byGroup[key] == nil {
			byGroup[key] = &GroupTotalsRow{GroupID: r.GroupID}
			if r.GroupID != nil {
				byGroup[key].Title = titles[*r.GroupID]
			}
			sums[key] = map[string]*Sums{}
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

func sortedSums(m map[string]*Sums) []Sums {
	out := make([]Sums, 0, len(m))
	for _, s := range m {
		out = append(out, *s)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Currency < out[j].Currency })
	return out
}
