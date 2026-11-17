package services

import (
	"testing"

	"github.com/cambodia-finance/income-expense/internal/models"
	"github.com/shopspring/decimal"
)

// A Total keeps its cents: W/L -1000 and COMM 5 at 10% is -99.50, not -100. Totals saved rounded to
// a whole amount before get their cents back once, at start-up.
func TestTotalsToCents(t *testing.T) {
	f := newWLFixture(t) // v1: (W/L + COMM) × rate
	want := decimal.RequireFromString("-99.5")
	a := f.assign(t, []models.Platform{f.p1}, 0)[0]
	if _, _, err := f.svc.UpdateAssignment(f.ctx, a.ID, AssignmentUpdateInput{CommissionRate: "10", Currency: "USD"}, all); err != nil {
		t.Fatal(err)
	}
	if _, _, err := f.save(f.week, 0, canAll, "", row(f.p1, "-1000", "5")); err != nil {
		t.Fatal(err)
	}
	r := sheetRow(t, f.sheet(t, f.week, 0), f.p1)
	if !r.JamePS.Equal(want) || len(r.Groups) != 1 || !r.Groups[0].JamePS.Equal(want) {
		t.Fatalf("(-1000 + 5) × 10%% = -99.50: got %s (groups %+v)", r.JamePS, r.Groups)
	}
	rep, err := f.svc.Report(f.ctx, ReportFilter{DateFrom: dayString(f.week), DateTo: dayString(f.week.AddDate(0, 0, 6))}, all, true, f.actor)
	if err != nil || len(rep.Lines) != 1 || !rep.Lines[0].JamePS.Equal(want) {
		t.Fatalf("the report's Total: %v %+v", err, rep)
	}

	// As saved before Totals kept their cents: -100.
	var saved models.WeeklyPlatformResult
	if err := f.db.Where("platform_id = ? AND branch_id = ?", f.p1.ID, f.hq[0].ID).First(&saved).Error; err != nil {
		t.Fatal(err)
	}
	f.db.Model(&models.WeeklyPlatformResult{}).Where("id = ?", saved.ID).UpdateColumn("jame_ps", -100)
	f.db.Model(&models.WeeklyGroupResult{}).Where("result_id = ?", saved.ID).UpdateColumn("jame_ps", -100)
	f.db.Where("`key` = ?", CentsSettingKey).Delete(&models.AppSetting{})
	defer f.db.Where("`key` = ?", CentsSettingKey).Delete(&models.AppSetting{})

	changed, err := f.svc.RecalculateToCents(f.ctx)
	if err != nil || changed != 1 {
		t.Fatalf("the week is worked out to the cent once: %v %d", err, changed)
	}
	if r := sheetRow(t, f.sheet(t, f.week, 0), f.p1); !r.JamePS.Equal(want) || !r.Groups[0].JamePS.Equal(want) {
		t.Errorf("back to -99.50: %s %+v", r.JamePS, r.Groups)
	}
	// Done once: later start-ups leave the weeks alone.
	f.db.Model(&models.WeeklyPlatformResult{}).Where("id = ?", saved.ID).UpdateColumn("jame_ps", -100)
	if again, err := f.svc.RecalculateToCents(f.ctx); err != nil || again != 0 {
		t.Errorf("worked out again: %v %d", err, again)
	}
	f.db.Model(&models.WeeklyPlatformResult{}).Where("id = ?", saved.ID).UpdateColumn("jame_ps", want)
	// A week already to the cent isn't written again.
	if n, err := f.svc.RecalculateWeeks(f.ctx); err != nil || n != 0 {
		t.Errorf("nothing for the formula to change: %v %d", err, n)
	}
}
