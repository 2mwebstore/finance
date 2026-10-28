package services

import (
	"context"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// Win/lose reports: the weekly report and the HQ summary both come from one aggregation of the
// weekly results by HQ, platform and currency. A week is in the range when its Monday is (the
// range's first day counts from its own week's Monday). Amounts are never added up across
// currencies: every total is per currency. JamePS is what the app shows as "Total".

type ReportFilter struct {
	DateFrom, DateTo string
	BranchID         *uuid.UUID
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

// ReportLine is one HQ's platform in one currency over the range.
type ReportLine struct {
	BranchID     uuid.UUID `json:"branch_id"`
	BranchCode   string    `json:"branch_code"`
	BranchName   string    `json:"branch_name"`
	PlatformID   uuid.UUID `json:"platform_id"`
	PlatformCode string    `json:"platform_code"`
	PlatformName string    `json:"platform_name"`
	// The rates used (the same unless a rate changed during the range).
	RateMin decimal.Decimal `json:"rate_min"`
	RateMax decimal.Decimal `json:"rate_max"`
	Sums
}

type HQTotals struct {
	BranchID   uuid.UUID `json:"branch_id"`
	BranchCode string    `json:"branch_code"`
	BranchName string    `json:"branch_name"`
	Totals     []Sums    `json:"totals"`
}

type PlatformTotals struct {
	PlatformID   uuid.UUID `json:"platform_id"`
	PlatformCode string    `json:"platform_code"`
	PlatformName string    `json:"platform_name"`
	Totals       []Sums    `json:"totals"`
}

type WinLoseReport struct {
	DateFrom string `json:"date_from"`
	DateTo   string `json:"date_to"`
	// Consolidated: every HQ in the user's scope (Main HQ view); otherwise one HQ.
	Consolidated bool             `json:"consolidated"`
	Lines        []ReportLine     `json:"lines"`
	HQs          []HQTotals       `json:"hqs"`
	Platforms    []PlatformTotals `json:"platforms"`
	Totals       []Sums           `json:"totals"`
}

// Report adds up results over a range. Without consolidated access it covers one HQ: the one
// asked for, or the user's only HQ.
func (s *PlatformService) Report(ctx context.Context, f ReportFilter, scope BranchScope, consolidated bool) (*WinLoseReport, error) {
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
		if scope.All || len(scope.IDs) != 1 {
			return nil, fieldError("branch_id", "choose an HQ (totals across HQs need View Consolidated Win/Lose)")
		}
		only := scope.IDs[0]
		branchID = &only
	}

	q := scope.where(s.db.WithContext(ctx).Table("weekly_platform_results"), "branch_id").
		Select("branch_id, platform_id, currency, MIN(commission_rate) AS rate_min, MAX(commission_rate) AS rate_max, "+
			"SUM(win_lose) AS win_lose, SUM(commission) AS commission, SUM(adjusted_amount) AS adjusted_amount, SUM(jame_ps) AS jame_ps, COUNT(*) AS entries").
		Where("deleted_at IS NULL AND week_start BETWEEN ? AND ?", dayString(weekStart(from)), dayString(to))
	if branchID != nil {
		q = q.Where("branch_id = ?", *branchID)
	}
	var lines []ReportLine
	if err := q.Group("branch_id, platform_id, currency").Scan(&lines).Error; err != nil {
		return nil, err
	}

	var branchIDs, platformIDs []uuid.UUID
	for _, l := range lines {
		branchIDs, platformIDs = append(branchIDs, l.BranchID), append(platformIDs, l.PlatformID)
	}
	branches, err := s.branchesIn(ctx, branchIDs)
	if err != nil {
		return nil, err
	}
	platforms, err := s.platformsIn(ctx, platformIDs)
	if err != nil {
		return nil, err
	}
	for i := range lines {
		b, p := branches[lines[i].BranchID], platforms[lines[i].PlatformID]
		lines[i].BranchCode, lines[i].BranchName, lines[i].PlatformCode, lines[i].PlatformName = b.Code, b.Name, p.Code, p.Name
	}
	sort.Slice(lines, func(i, j int) bool {
		a, b := lines[i], lines[j]
		if a.BranchCode != b.BranchCode {
			return a.BranchCode < b.BranchCode
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
	hqs := map[uuid.UUID]*HQTotals{}
	plats := map[uuid.UUID]*PlatformTotals{}
	total := map[string]*Sums{}
	hqSums := map[uuid.UUID]map[string]*Sums{}
	platSums := map[uuid.UUID]map[string]*Sums{}
	bump := func(m map[string]*Sums, l ReportLine) {
		if m[l.Currency] == nil {
			m[l.Currency] = &Sums{Currency: l.Currency}
		}
		m[l.Currency].add(l.Sums)
	}
	for _, l := range lines {
		if hqs[l.BranchID] == nil {
			hqs[l.BranchID] = &HQTotals{BranchID: l.BranchID, BranchCode: l.BranchCode, BranchName: l.BranchName}
			hqSums[l.BranchID] = map[string]*Sums{}
		}
		if plats[l.PlatformID] == nil {
			plats[l.PlatformID] = &PlatformTotals{PlatformID: l.PlatformID, PlatformCode: l.PlatformCode, PlatformName: l.PlatformName}
			platSums[l.PlatformID] = map[string]*Sums{}
		}
		bump(hqSums[l.BranchID], l)
		bump(platSums[l.PlatformID], l)
		bump(total, l)
	}
	out.HQs, out.Platforms, out.Totals = []HQTotals{}, []PlatformTotals{}, sortedSums(total)
	for id, h := range hqs {
		h.Totals = sortedSums(hqSums[id])
		out.HQs = append(out.HQs, *h)
	}
	for id, p := range plats {
		p.Totals = sortedSums(platSums[id])
		out.Platforms = append(out.Platforms, *p)
	}
	sort.Slice(out.HQs, func(i, j int) bool { return out.HQs[i].BranchCode < out.HQs[j].BranchCode })
	sort.Slice(out.Platforms, func(i, j int) bool { return out.Platforms[i].PlatformCode < out.Platforms[j].PlatformCode })
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
