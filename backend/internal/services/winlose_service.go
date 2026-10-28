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

// ===== Week sheet =====

type SettlementState struct {
	ID        uuid.UUID `json:"id"`
	Status    string    `json:"status"`
	WeekStart string    `json:"week_start"`
	WeekEnd   string    `json:"week_end"`
}

// WeekSheetRow is one platform of the HQ in the week: its saved result, or the assignment's rate
// and currency for a new one.
type WeekSheetRow struct {
	PlatformID     uuid.UUID        `json:"platform_id"`
	PlatformCode   string           `json:"platform_code"`
	PlatformName   string           `json:"platform_name"`
	AssignmentID   uuid.UUID        `json:"assignment_id"`
	CommissionRate decimal.Decimal  `json:"commission_rate"`
	Currency       string           `json:"currency"`
	Saved          bool             `json:"saved"`
	WinLose        *decimal.Decimal `json:"win_lose"`
	Commission     *decimal.Decimal `json:"commission"`
	JamePS         *decimal.Decimal `json:"jame_ps"` // shown as "Total"
	FormulaVersion string           `json:"formula_version"`
	UpdatedBy      string           `json:"updated_by"`
	UpdatedAt      *time.Time       `json:"updated_at"`
}

// WeekSheet is an HQ's week (Monday to Sunday): every platform it had that week.
type WeekSheet struct {
	WeekStart  string           `json:"week_start"`
	WeekEnd    string           `json:"week_end"`
	BranchID   uuid.UUID        `json:"branch_id"`
	BranchCode string           `json:"branch_code"`
	BranchName string           `json:"branch_name"`
	Formula    winlose.Formula  `json:"formula"`
	Settlement *SettlementState `json:"settlement"`
	// Locked: the week is finalized, so nothing can be changed.
	Locked bool           `json:"locked"`
	Rows   []WeekSheetRow `json:"rows"`
}

// sheetParts are what a week needs: the assignments it had and the results already saved.
type sheetParts struct {
	branch      models.Branch
	assignments map[uuid.UUID]models.HQPlatformAssignment // by platform
	results     map[uuid.UUID]models.WeeklyPlatformResult // by platform
	platforms   map[uuid.UUID]models.Platform
	settlement  *models.WeeklySettlement
}

// loadWeek reads an HQ's week. A platform the HQ had on any day of the week is on the sheet; if
// it was ended and given again within the week, the later assignment is used.
func (s *PlatformService) loadWeek(tx *gorm.DB, start time.Time, branchID uuid.UUID, lock bool) (*sheetParts, error) {
	end := start.AddDate(0, 0, 6)
	p := &sheetParts{assignments: map[uuid.UUID]models.HQPlatformAssignment{}, results: map[uuid.UUID]models.WeeklyPlatformResult{}}
	if err := tx.First(&p.branch, "id = ?", branchID).Error; err != nil {
		return nil, fieldError("branch_id", "this HQ was not found")
	}
	var assignments []models.HQPlatformAssignment
	if err := tx.Where("branch_id = ? AND starts_on <= ? AND (ends_on IS NULL OR ends_on >= ?)", branchID, dayString(end), dayString(start)).
		Order("starts_on").Find(&assignments).Error; err != nil {
		return nil, err
	}
	for _, a := range assignments {
		p.assignments[a.PlatformID] = a
	}
	q := tx.Where("branch_id = ? AND week_start = ?", branchID, dayString(start))
	if lock {
		q = q.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	var results []models.WeeklyPlatformResult
	if err := q.Find(&results).Error; err != nil {
		return nil, err
	}
	var platformIDs []uuid.UUID
	for _, r := range results {
		p.results[r.PlatformID] = r
		platformIDs = append(platformIDs, r.PlatformID)
	}
	for id := range p.assignments {
		platformIDs = append(platformIDs, id)
	}
	var err error
	if p.platforms, err = s.platformsIn(tx.Statement.Context, platformIDs); err != nil {
		return nil, err
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

func (p *sheetParts) locked() bool {
	return p.settlement != nil && p.settlement.Status == models.SettlementFinalized
}

// WeekSheet returns the HQ's week that contains the given day (any day of it; empty is this week).
func (s *PlatformService) WeekSheet(ctx context.Context, week string, branchID uuid.UUID, scope BranchScope) (*WeekSheet, error) {
	if !scope.Has(branchID) {
		return nil, ErrBranchAccessDenied
	}
	day, err := s.parseDay("week_start", week)
	if err != nil {
		return nil, err
	}
	formula, err := s.Formula(ctx)
	if err != nil {
		return nil, err
	}
	start := weekStart(day)
	parts, err := s.loadWeek(s.db.WithContext(ctx), start, branchID, false)
	if err != nil {
		return nil, err
	}
	return s.sheet(ctx, start, parts, formula), nil
}

func (s *PlatformService) sheet(ctx context.Context, start time.Time, p *sheetParts, formula winlose.Formula) *WeekSheet {
	out := &WeekSheet{WeekStart: dayString(start), WeekEnd: dayString(start.AddDate(0, 0, 6)), BranchID: p.branch.ID, BranchCode: p.branch.Code, BranchName: p.branch.Name,
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
	names := s.userNames(ctx, users)
	seen := map[uuid.UUID]bool{}
	add := func(platformID uuid.UUID) {
		if seen[platformID] {
			return
		}
		seen[platformID] = true
		pl := p.platforms[platformID]
		row := WeekSheetRow{PlatformID: platformID, PlatformCode: pl.Code, PlatformName: pl.Name}
		if r, ok := p.results[platformID]; ok {
			wl, cm, jp, at := r.WinLose, r.Commission, r.JamePS, r.UpdatedAt
			row.AssignmentID, row.CommissionRate, row.Currency, row.FormulaVersion = r.AssignmentID, r.CommissionRate, r.Currency, r.FormulaVersion
			row.Saved, row.WinLose, row.Commission, row.JamePS, row.UpdatedAt = true, &wl, &cm, &jp, &at
			row.UpdatedBy = names[r.CreatedByID]
			if r.UpdatedByID != nil {
				row.UpdatedBy = names[*r.UpdatedByID]
			}
		} else {
			a := p.assignments[platformID]
			row.AssignmentID, row.CommissionRate, row.Currency, row.FormulaVersion = a.ID, a.CommissionRate, a.Currency, formula.Version
		}
		out.Rows = append(out.Rows, row)
	}
	for id := range p.assignments {
		add(id)
	}
	for id := range p.results {
		add(id)
	}
	sort.Slice(out.Rows, func(i, j int) bool { return out.Rows[i].PlatformCode < out.Rows[j].PlatformCode })
	return out
}

// ===== Entering a week =====

type WeekEntryRow struct {
	PlatformID string `json:"platform_id" binding:"required,uuid"`
	WinLose    string `json:"win_lose"`
	Commission string `json:"commission"`
}

type WeekEntryInput struct {
	// Any day of the week (normally its Monday).
	WeekStart string         `json:"week_start" binding:"required"`
	BranchID  string         `json:"branch_id" binding:"required,uuid"`
	Rows      []WeekEntryRow `json:"rows" binding:"required,max=500,dive"`
	// Needed when a saved week is changed (kept in the audit log).
	Reason string `json:"reason" binding:"max=512"`
}

// Correction is a saved week that was changed (for the audit log).
type Correction struct {
	ResultID     uuid.UUID
	PlatformCode string
	Before       map[string]string
	After        map[string]string
}

type rowPlan struct {
	platform        uuid.UUID
	winLose, commis decimal.Decimal
	result          winlose.Result
	rate            decimal.Decimal
	currency        string
	assignment      uuid.UUID
	existing        *models.WeeklyPlatformResult
}

// plan works out every filled-in row: a saved week keeps its own rate, currency and formula
// version; a new one takes the assignment's and the current formula. Blank rows are left out.
func (s *PlatformService) plan(p *sheetParts, rows []WeekEntryRow, formula winlose.Formula) ([]rowPlan, error) {
	var out []rowPlan
	seen := map[uuid.UUID]bool{}
	for i, row := range rows {
		platformID := uuid.MustParse(row.PlatformID)
		if seen[platformID] {
			return nil, fieldError("rows", "a platform is listed twice")
		}
		seen[platformID] = true
		if strings.TrimSpace(row.WinLose) == "" && strings.TrimSpace(row.Commission) == "" {
			continue
		}
		name := p.platforms[platformID].Code
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
		if r, ok := p.results[platformID]; ok {
			r := r
			plan.existing, plan.rate, plan.currency, plan.assignment = &r, r.CommissionRate, r.Currency, r.AssignmentID
			plan.result, err = winlose.Calculate(r.FormulaVersion, winlose.Input{WinLose: wl, Commission: cm, RatePct: r.CommissionRate})
		} else if a, ok := p.assignments[platformID]; ok {
			plan.rate, plan.currency, plan.assignment = a.CommissionRate, a.Currency, a.ID
			plan.result, err = winlose.Calculate(formula.Version, winlose.Input{WinLose: wl, Commission: cm, RatePct: a.CommissionRate})
		} else {
			return nil, fieldError("rows", name+" isn't assigned to "+p.branch.Code+" in this week")
		}
		if err != nil {
			return nil, err
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
	winlose.Result
}

// Preview works out rows without saving them (the entry page shows the Total as people type).
func (s *PlatformService) Preview(ctx context.Context, in WeekEntryInput, scope BranchScope) ([]PreviewRow, error) {
	branchID := uuid.MustParse(in.BranchID)
	if !scope.Has(branchID) {
		return nil, ErrBranchAccessDenied
	}
	day, err := s.parseDay("week_start", in.WeekStart)
	if err != nil {
		return nil, err
	}
	formula, err := s.Formula(ctx)
	if err != nil {
		return nil, err
	}
	parts, err := s.loadWeek(s.db.WithContext(ctx), weekStart(day), branchID, false)
	if err != nil {
		return nil, err
	}
	plans, err := s.plan(parts, in.Rows, formula)
	if err != nil {
		return nil, err
	}
	out := make([]PreviewRow, len(plans))
	for i, p := range plans {
		out[i] = PreviewRow{PlatformID: p.platform, CommissionRate: p.rate, Currency: p.currency, Result: p.result}
	}
	return out, nil
}

// SaveWeek saves an HQ's week in one transaction. New rows need winlose.create; changing a saved
// row needs winlose.edit and a reason. A finalized week can't be changed.
func (s *PlatformService) SaveWeek(ctx context.Context, in WeekEntryInput, scope BranchScope, can func(string) bool, actor uuid.UUID) (*WeekSheet, []Correction, error) {
	branchID := uuid.MustParse(in.BranchID)
	if !scope.Has(branchID) {
		return nil, nil, ErrBranchAccessDenied
	}
	day, err := s.parseDay("week_start", in.WeekStart)
	if err != nil {
		return nil, nil, err
	}
	start := weekStart(day)
	if start.After(s.today()) {
		return nil, nil, fieldError("week_start", "results can't be entered for a week that hasn't started")
	}
	formula, err := s.Formula(ctx)
	if err != nil {
		return nil, nil, err
	}
	reason := strings.TrimSpace(in.Reason)
	var corrections []Correction
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		parts, err := s.loadWeek(tx, start, branchID, true)
		if err != nil {
			return err
		}
		if parts.locked() {
			return fmt.Errorf("the week of %s is finalized for %s; reopen it to make changes", dayString(start), parts.branch.Code)
		}
		plans, err := s.plan(parts, in.Rows, formula)
		if err != nil {
			return err
		}
		now := time.Now()
		for _, p := range plans {
			code := parts.platforms[p.platform].Code
			if p.existing == nil {
				if !can(permissions.WinLoseCreate) {
					return errors.New("you can't enter new results (needs Enter Win/Lose)")
				}
				row := models.WeeklyPlatformResult{
					Base: models.Base{ID: uuid.New()}, WeekStart: start, WeekEnd: start.AddDate(0, 0, 6), BranchID: branchID, PlatformID: p.platform, AssignmentID: p.assignment,
					WinLose: p.winLose, Commission: p.commis, CommissionRate: p.rate, Currency: p.currency,
					AdjustedAmount: p.result.Adjusted, JamePS: p.result.JamePS, FormulaVersion: p.result.Version, CreatedByID: actor,
				}
				if err := tx.Create(&row).Error; err != nil {
					if isDuplicate(err) {
						return errors.New(code + " was just saved for this week by someone else; reload and check it")
					}
					return err
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
				"updated_by_id": actor, "updated_at": now,
			}).Error; err != nil {
				return err
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
	sheet, err := s.WeekSheet(ctx, dayString(start), branchID, scope)
	return sheet, corrections, err
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
	BranchID  string `json:"branch_id" binding:"required,uuid"`
	WeekStart string `json:"week_start" binding:"required"` // a Monday
}

// FinalizeWeek freezes an HQ's week: its totals per currency are kept and the week locked. A
// reopened week can be finalized again (the totals are worked out afresh).
func (s *PlatformService) FinalizeWeek(ctx context.Context, in FinalizeInput, scope BranchScope, actor uuid.UUID) (*SettlementView, error) {
	branchID := uuid.MustParse(in.BranchID)
	if !scope.Has(branchID) {
		return nil, ErrBranchAccessDenied
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
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var st models.WeeklySettlement
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&st, "id = ?", id).Error; err != nil || !scope.Has(st.BranchID) {
			return ErrSettlementNotFound
		}
		if st.Status != models.SettlementFinalized {
			return errors.New("this week is not finalized")
		}
		now := time.Now()
		return tx.Model(&st).Updates(map[string]interface{}{"status": models.SettlementReopened, "reopened_by_id": actor, "reopened_at": now, "reopen_reason": reason, "updated_at": now}).Error
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
