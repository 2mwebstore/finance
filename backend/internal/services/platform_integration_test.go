package services

// Platforms and weekly win/lose against a real MySQL database. These run only when TEST_MYSQL_DSN
// names a scratch database whose name contains "test", e.g.
//
//	TEST_MYSQL_DSN='root:pw@tcp(127.0.0.1:3306)/finance_test?charset=utf8mb4&parseTime=True&loc=Asia%2FPhnom_Penh' go test ./internal/services/ -run Platform
//
// Tables are created with database.AutoMigrate; the test HQs, user and platforms are prefixed TWL.

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/cambodia-finance/income-expense/internal/database"
	"github.com/cambodia-finance/income-expense/internal/models"
	"github.com/cambodia-finance/income-expense/internal/permissions"
	mysqldriver "github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type wlFixture struct {
	db    *gorm.DB
	svc   *PlatformService
	ctx   context.Context
	actor uuid.UUID
	hq    [4]models.Branch
	p1    models.Platform // USD, 35%
	p2    models.Platform // KHR, 10%
	week  time.Time       // the Monday two weeks ago
}

func newWLFixture(t *testing.T) *wlFixture {
	t.Helper()
	dsn := os.Getenv("TEST_MYSQL_DSN")
	if dsn == "" {
		t.Skip("TEST_MYSQL_DSN is not set")
	}
	cfg, err := mysqldriver.ParseDSN(dsn)
	if err != nil || !strings.Contains(cfg.DBName, "test") {
		t.Fatalf("TEST_MYSQL_DSN must name a scratch database with \"test\" in its name, got %q", cfg.DBName)
	}
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	if err := database.AutoMigrate(db); err != nil {
		t.Fatal(err)
	}
	f := &wlFixture{db: db, svc: NewPlatformService(db, "Asia/Phnom_Penh"), ctx: context.Background()}
	for _, table := range []string{"weekly_settlement_totals", "weekly_settlements", "weekly_entries", "weekly_partner_results", "weekly_platform_results", "assignment_partners", "hq_platform_assignments", "assignment_groups"} {
		db.Exec("DELETE FROM " + table)
	}
	db.Exec("DELETE FROM platforms WHERE code LIKE 'TWL%'")
	user := models.User{Username: "twl_tester", Email: "twl_tester@example.test", PasswordHash: "x", FullName: "TWL Tester", IsActive: true}
	if err := db.Where(models.User{Username: user.Username}).FirstOrCreate(&user).Error; err != nil {
		t.Fatal(err)
	}
	f.actor = user.ID
	for i := range f.hq {
		code := []string{"TWL1", "TWL2", "TWL3", "TWL4"}[i]
		b := models.Branch{Code: code, Name: "Test HQ " + code, IsActive: true}
		if err := db.Where(models.Branch{Code: code}).FirstOrCreate(&b).Error; err != nil {
			t.Fatal(err)
		}
		f.hq[i] = b
	}
	f.p1 = f.platform(t, "TWL_LCMMJB", "35", "USD")
	f.p2 = f.platform(t, "TWL_12K_YMLLJB_V1", "10", "KHR")
	f.week = weekStart(f.svc.today()).AddDate(0, 0, -14)
	return f
}

func (f *wlFixture) platform(t *testing.T, code, rate, currency string) models.Platform {
	t.Helper()
	p, err := f.svc.SavePlatform(f.ctx, nil, PlatformInput{Code: code, Name: code, DefaultRate: rate, DefaultCurrency: currency})
	if err != nil {
		t.Fatal(err)
	}
	return *p
}

func (f *wlFixture) scope(hqs ...int) BranchScope {
	s := BranchScope{}
	for _, i := range hqs {
		s.IDs = append(s.IDs, f.hq[i].ID)
	}
	return s
}

var all = BranchScope{All: true}

func canAll(string) bool { return true }

func (f *wlFixture) ids(hqs ...int) []string {
	out := []string{}
	for _, i := range hqs {
		out = append(out, f.hq[i].ID.String())
	}
	return out
}

// assign gives platforms to HQs from three weeks before the test week.
func (f *wlFixture) assign(t *testing.T, platforms []models.Platform, hqs ...int) []Assignment {
	t.Helper()
	var pids []string
	for _, p := range platforms {
		pids = append(pids, p.ID.String())
	}
	list, err := f.svc.Assign(f.ctx, AssignmentInput{PlatformIDs: pids, BranchIDs: f.ids(hqs...), StartsOn: dayString(f.week.AddDate(0, 0, -21))}, all, f.actor)
	if err != nil {
		t.Fatal(err)
	}
	return list
}

func (f *wlFixture) save(week time.Time, hq int, can func(string) bool, reason string, rows ...WeekEntryRow) (*WeekSheet, []Correction, error) {
	return f.svc.SaveWeek(f.ctx, WeekEntryInput{WeekStart: dayString(week), BranchID: f.hq[hq].ID.String(), Title: "TWL week", Rows: rows, Reason: reason}, all, can, f.actor)
}

func row(p models.Platform, winLose, commission string) WeekEntryRow {
	return WeekEntryRow{PlatformID: p.ID.String(), WinLose: winLose, Commission: commission}
}

func sheetRow(t *testing.T, s *WeekSheet, p models.Platform) WeekSheetRow {
	t.Helper()
	for _, r := range s.Rows {
		if r.PlatformID == p.ID {
			return r
		}
	}
	t.Fatalf("%s is not on the sheet", p.Code)
	return WeekSheetRow{}
}

func TestPlatformAssignmentsManyToMany(t *testing.T) {
	f := newWLFixture(t)
	// Two platforms to two HQs in one go; one platform also to a third HQ.
	if got := len(f.assign(t, []models.Platform{f.p1, f.p2}, 0, 1)); got != 4 {
		t.Fatalf("%d assignments, want 4 (2 platforms × 2 HQs)", got)
	}
	f.assign(t, []models.Platform{f.p1}, 2)
	sheet, err := f.svc.WeekSheet(f.ctx, dayString(f.week.AddDate(0, 0, 3)), f.hq[0].ID.String(), all, f.actor)
	if err != nil {
		t.Fatal(err)
	}
	if len(sheet.Rows) != 2 || sheet.WeekStart != dayString(f.week) {
		t.Errorf("HQ1's week (from any day of it) has %d platforms from %s, want 2 from %s", len(sheet.Rows), sheet.WeekStart, dayString(f.week))
	}
	if r := sheetRow(t, sheet, f.p2); r.Currency != "KHR" || !r.CommissionRate.Equal(decimal.NewFromInt(10)) {
		t.Errorf("each platform takes its own default rate and currency, got %s %s", r.CommissionRate, r.Currency)
	}

	// No second active assignment of a platform in an HQ; the whole request is refused.
	_, err = f.svc.Assign(f.ctx, AssignmentInput{PlatformIDs: []string{f.p2.ID.String()}, BranchIDs: f.ids(3, 0)}, all, f.actor)
	if err == nil || !strings.Contains(err.Error(), "is already assigned") {
		t.Fatalf("a duplicate active assignment must be refused, got %v", err)
	}
	var n int64
	f.db.Model(&models.HQPlatformAssignment{}).Where("branch_id = ?", f.hq[3].ID).Count(&n)
	if n != 0 {
		t.Error("a refused request must not assign any HQ")
	}
	one := int8(1)
	dup := models.HQPlatformAssignment{Base: models.Base{ID: uuid.New()}, BranchID: f.hq[0].ID, PlatformID: f.p1.ID, CommissionRate: decimal.Zero, Currency: "USD", IsActive: true, ActiveKey: &one, StartsOn: f.week, CreatedByID: f.actor}
	if err := f.db.Create(&dup).Error; !isDuplicate(err) {
		t.Errorf("the unique key must stop a second active assignment, got %v", err)
	}
	// A currency chosen applies to every platform assigned with it; each keeps its own default %.
	list, err := f.svc.Assign(f.ctx, AssignmentInput{PlatformIDs: []string{f.p1.ID.String(), f.p2.ID.String()}, BranchIDs: f.ids(3), Currency: "USD"}, all, f.actor)
	if err != nil || len(list) != 2 || list[0].Currency != "USD" || list[1].Currency != "USD" ||
		!list[0].CommissionRate.Equal(decimal.NewFromInt(35)) || !list[1].CommissionRate.Equal(decimal.NewFromInt(10)) || len(list[0].Partners) != 0 {
		t.Errorf("one currency for both platforms, each at its own %%: %v", err)
	}
}

func TestAssignmentHistoryKept(t *testing.T) {
	f := newWLFixture(t)
	a := f.assign(t, []models.Platform{f.p1}, 2)[0]
	if _, _, err := f.save(f.week, 2, canAll, "", row(f.p1, "100", "0")); err != nil {
		t.Fatal(err)
	}
	// Can't end it before a week that already has results.
	if _, err := f.svc.EndAssignment(f.ctx, a.ID, AssignmentEndInput{EndsOn: dayString(f.week.AddDate(0, 0, -1))}, all, f.actor); err == nil {
		t.Error("ending before a saved week must be refused")
	}
	// Move HQ3's platform to HQ4 from the next Monday.
	next := f.week.AddDate(0, 0, 7)
	ended, created, err := f.svc.MoveAssignment(f.ctx, a.ID, AssignmentMoveInput{BranchID: f.hq[3].ID.String(), On: dayString(next)}, all, f.actor)
	if err != nil {
		t.Fatal(err)
	}
	if ended.IsActive || ended.ActiveKey != nil || dayString(*ended.EndsOn) != dayString(next.AddDate(0, 0, -1)) {
		t.Errorf("the old assignment ends the day before the move, got active=%v ends=%v", ended.IsActive, ended.EndsOn)
	}
	if created.BranchID != f.hq[3].ID || !created.IsActive || !created.CommissionRate.Equal(a.CommissionRate) {
		t.Error("the new assignment is active at the new HQ with the same rate")
	}
	var rows []models.HQPlatformAssignment
	f.db.Where("platform_id = ?", f.p1.ID).Find(&rows)
	if len(rows) != 2 {
		t.Errorf("history: %d assignments, want 2", len(rows))
	}
	sheet, _ := f.svc.WeekSheet(f.ctx, dayString(f.week), f.hq[2].ID.String(), all, f.actor)
	if !sheetRow(t, sheet, f.p1).Saved {
		t.Error("the old HQ's saved week stays")
	}
	later, _ := f.svc.WeekSheet(f.ctx, dayString(next), f.hq[2].ID.String(), all, f.actor)
	if len(later.Rows) != 0 {
		t.Error("after the move the old HQ no longer has the platform")
	}
	if _, err = f.svc.Assign(f.ctx, AssignmentInput{PlatformIDs: []string{f.p1.ID.String()}, BranchIDs: f.ids(2), StartsOn: dayString(f.week)}, all, f.actor); err == nil {
		t.Error("a new assignment overlapping an ended one must be refused")
	}
	if _, err := f.svc.Assign(f.ctx, AssignmentInput{PlatformIDs: []string{f.p1.ID.String()}, BranchIDs: f.ids(2), StartsOn: dayString(next)}, all, f.actor); err != nil {
		t.Errorf("re-assigning after the end is allowed, got %v", err)
	}
}

func TestWeeklyEntryCalculationAndRateHistory(t *testing.T) {
	f := newWLFixture(t)
	a := f.assign(t, []models.Platform{f.p1, f.p2}, 0)[0]
	sheet, _, err := f.save(f.week, 0, canAll, "", row(f.p1, "1000", "200"), row(f.p2, "-410000", "20500"))
	if err != nil {
		t.Fatal(err)
	}
	r1, r2 := sheetRow(t, sheet, f.p1), sheetRow(t, sheet, f.p2)
	if !r1.JamePS.Equal(decimal.NewFromInt(420)) || r1.Currency != "USD" || r1.FormulaVersion != "v1" {
		t.Errorf("USD row: total %s %s %s, want 420 USD v1", r1.JamePS, r1.Currency, r1.FormulaVersion)
	}
	if !r2.JamePS.Equal(decimal.NewFromInt(-38950)) || r2.Currency != "KHR" {
		t.Errorf("KHR row: total %s %s, want -38950 KHR", r2.JamePS, r2.Currency)
	}

	// Entering from another day of the same week updates the same week; no second row.
	if _, _, err := f.save(f.week.AddDate(0, 0, 4), 0, canAll, "", row(f.p1, "1000", "200")); err != nil {
		t.Fatal(err)
	}
	var n int64
	f.db.Model(&models.WeeklyPlatformResult{}).Where("branch_id = ? AND week_start = ?", f.hq[0].ID, dayString(f.week)).Count(&n)
	if n != 2 {
		t.Fatalf("%d rows for the week, want 2", n)
	}
	dup := models.WeeklyPlatformResult{Base: models.Base{ID: uuid.New()}, WeekStart: f.week, WeekEnd: f.week.AddDate(0, 0, 6), BranchID: f.hq[0].ID, PlatformID: f.p1.ID, AssignmentID: a.ID,
		Currency: "USD", FormulaVersion: "v1", CreatedByID: f.actor}
	if err := f.db.Create(&dup).Error; !isDuplicate(err) {
		t.Errorf("the unique key must stop a second row for the week, got %v", err)
	}

	// A new rate applies to new weeks only; the saved week keeps 35%.
	if _, _, err := f.svc.UpdateAssignment(f.ctx, a.ID, AssignmentUpdateInput{CommissionRate: "50", Currency: "USD"}, all); err != nil {
		t.Fatal(err)
	}
	next, _, err := f.save(f.week.AddDate(0, 0, 7), 0, canAll, "", row(f.p1, "1000", "200"))
	if err != nil {
		t.Fatal(err)
	}
	if r := sheetRow(t, next, f.p1); !r.JamePS.Equal(decimal.NewFromInt(600)) || !r.CommissionRate.Equal(decimal.NewFromInt(50)) {
		t.Errorf("the next week uses 50%%: total %s rate %s", r.JamePS, r.CommissionRate)
	}

	// A correction needs the permission and a reason, and is worked out at the week's own rate.
	onlyCreate := func(code string) bool { return code == permissions.WinLoseCreate }
	if _, _, err := f.save(f.week, 0, onlyCreate, "typo", row(f.p1, "2000", "0")); err == nil {
		t.Error("changing a saved week without winlose.edit must be refused")
	}
	if _, _, err := f.save(f.week, 0, canAll, "", row(f.p1, "2000", "0")); err == nil {
		t.Error("changing a saved week without a reason must be refused")
	}
	fixed, corrections, err := f.save(f.week, 0, canAll, "typo in win/lose", row(f.p1, "2000", "0"), row(f.p2, "-410000", "20500"))
	if err != nil {
		t.Fatal(err)
	}
	if r := sheetRow(t, fixed, f.p1); !r.JamePS.Equal(decimal.NewFromInt(700)) || !r.CommissionRate.Equal(decimal.NewFromInt(35)) {
		t.Errorf("corrected at the saved 35%%: total %s rate %s, want 700 at 35", r.JamePS, r.CommissionRate)
	}
	if len(corrections) != 1 || corrections[0].Before["win_lose"] != "1000" || corrections[0].After["total"] != "700" {
		t.Errorf("only the changed row is a correction, with before and after: %+v", corrections)
	}

	// A platform given mid-week is on that week's sheet.
	mid := f.week.AddDate(0, 0, 2)
	if _, err := f.svc.Assign(f.ctx, AssignmentInput{PlatformIDs: []string{f.p1.ID.String()}, BranchIDs: f.ids(3), StartsOn: dayString(mid)}, all, f.actor); err != nil {
		t.Fatal(err)
	}
	if s, _ := f.svc.WeekSheet(f.ctx, dayString(f.week), f.hq[3].ID.String(), all, f.actor); len(s.Rows) != 1 {
		t.Error("a platform assigned on the Wednesday is part of that week")
	}
	// Not assigned, or a week that hasn't started: refused.
	if _, _, err := f.save(f.week, 1, canAll, "", row(f.p1, "1", "1")); err == nil {
		t.Error("a platform not assigned to the HQ must be refused")
	}
	if _, _, err := f.save(weekStart(f.svc.today()).AddDate(0, 0, 7), 0, canAll, "", row(f.p1, "1", "1")); err == nil {
		t.Error("a future week must be refused")
	}
}

func TestWinLoseHQAccess(t *testing.T) {
	f := newWLFixture(t)
	f.assign(t, []models.Platform{f.p1}, 0, 1)
	only2 := f.scope(1)
	if _, err := f.svc.WeekSheet(f.ctx, dayString(f.week), f.hq[0].ID.String(), only2, f.actor); !errors.Is(err, ErrBranchAccessDenied) {
		t.Errorf("another HQ's week must be refused, got %v", err)
	}
	if _, _, err := f.svc.SaveWeek(f.ctx, WeekEntryInput{WeekStart: dayString(f.week), BranchID: f.hq[0].ID.String(), Title: "TWL week", Rows: []WeekEntryRow{row(f.p1, "1", "1")}}, only2, canAll, f.actor); !errors.Is(err, ErrBranchAccessDenied) {
		t.Errorf("saving another HQ's week must be refused, got %v", err)
	}
	if _, err := f.svc.Report(f.ctx, ReportFilter{BranchID: &f.hq[0].ID}, only2, true, f.actor); !errors.Is(err, ErrBranchAccessDenied) {
		t.Errorf("another HQ's report must be refused, got %v", err)
	}
	if list, _ := f.svc.ListAssignments(f.ctx, AssignmentFilter{}, only2); len(list) != 1 || list[0].BranchID != f.hq[1].ID {
		t.Errorf("assignments list only the user's HQs, got %d", len(list))
	}
	if _, err := f.svc.Assign(f.ctx, AssignmentInput{PlatformIDs: []string{f.p2.ID.String()}, BranchIDs: f.ids(0)}, only2, f.actor); !errors.Is(err, ErrBranchAccessDenied) {
		t.Errorf("assigning to another HQ must be refused, got %v", err)
	}
	if _, err := f.svc.Report(f.ctx, ReportFilter{}, f.scope(0, 1), false, f.actor); err == nil {
		t.Error("two groups, no default branch and no consolidated access: there's no group to report on")
	}
	// Without consolidated access the report is the person's own group (their default branch).
	f.db.Model(&models.User{}).Where("id = ?", f.actor).Update("default_branch_id", f.hq[0].ID)
	defer f.db.Model(&models.User{}).Where("id = ?", f.actor).Update("default_branch_id", nil)
	if _, _, err := f.save(f.week, 0, canAll, "", row(f.p1, "100", "0")); err != nil {
		t.Fatal(err)
	}
	if _, _, err := f.save(f.week, 1, canAll, "", row(f.p1, "900", "0")); err != nil {
		t.Fatal(err)
	}
	own, err := f.svc.Report(f.ctx, ReportFilter{DateFrom: dayString(f.week), DateTo: dayString(f.week.AddDate(0, 0, 6))}, f.scope(0, 1), false, f.actor)
	if err != nil || own.Consolidated || len(own.Lines) != 1 || !own.Lines[0].WinLose.Equal(decimal.NewFromInt(100)) {
		t.Errorf("own group only: %v %+v", err, own)
	}
	// Assigning and finalizing without a group go to the person's own group too.
	if list, err := f.svc.Assign(f.ctx, AssignmentInput{PlatformIDs: []string{f.p2.ID.String()}}, all, f.actor); err != nil || len(list) != 1 || list[0].BranchID != f.hq[0].ID {
		t.Errorf("assign to the own group: %v %+v", err, list)
	}
	if st, err := f.svc.FinalizeWeek(f.ctx, FinalizeInput{WeekStart: dayString(f.week)}, all, f.actor); err != nil || st.BranchID != f.hq[0].ID {
		t.Errorf("finalize the own group's week: %v", err)
	}
	if r, err := f.svc.Report(f.ctx, ReportFilter{}, only2, false, f.actor); err != nil || r.Consolidated {
		t.Errorf("a user with one HQ gets that HQ's report, got %v", err)
	}
}

func TestWinLoseReportsByCurrencyAndWeek(t *testing.T) {
	f := newWLFixture(t)
	f.assign(t, []models.Platform{f.p1}, 0, 1)
	f.assign(t, []models.Platform{f.p2}, 0)
	must := func(_ *WeekSheet, _ []Correction, err error) {
		if err != nil {
			t.Fatal(err)
		}
	}
	must(f.save(f.week, 0, canAll, "", row(f.p1, "1000", "200"), row(f.p2, "-410000", "20500"))) // 420 USD, -38950 KHR
	must(f.save(f.week, 1, canAll, "", row(f.p1, "-500", "100")))                                // -140 USD
	must(f.save(f.week.AddDate(0, 0, 7), 0, canAll, "", row(f.p1, "100", "0")))                  // next week: 35 USD

	// A range starting mid-week still counts that week; the next week is left out.
	r, err := f.svc.Report(f.ctx, ReportFilter{DateFrom: dayString(f.week.AddDate(0, 0, 3)), DateTo: dayString(f.week.AddDate(0, 0, 6))}, all, true, f.actor)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{"USD": "280", "KHR": "-38950"}
	if len(r.Totals) != 2 {
		t.Fatalf("totals in %d currencies, want 2 (USD and KHR kept apart)", len(r.Totals))
	}
	for _, s := range r.Totals {
		if !s.JamePS.Equal(decimal.RequireFromString(want[s.Currency])) {
			t.Errorf("%s total %s, want %s", s.Currency, s.JamePS, want[s.Currency])
		}
	}
	// Consolidated: both groups added up by agent (one USD line), from the week's Monday.
	if len(r.Lines) != 2 || !r.Consolidated || r.DateFrom != dayString(f.week) {
		t.Errorf("one line per agent and currency across both groups from the week's Monday, got %d lines from %s", len(r.Lines), r.DateFrom)
	}
	for _, l := range r.Lines {
		if l.Currency == "USD" && (l.Entries != 2 || !l.JamePS.Equal(decimal.NewFromInt(280))) {
			t.Errorf("USD line across both groups: %d entries, total %s; want 2, 280", l.Entries, l.JamePS)
		}
	}
	two, _ := f.svc.Report(f.ctx, ReportFilter{DateFrom: dayString(f.week), DateTo: dayString(f.week.AddDate(0, 0, 13)), BranchID: &f.hq[0].ID}, all, false, f.actor)
	for _, l := range two.Lines {
		if l.Currency == "USD" && (l.Entries != 2 || !l.JamePS.Equal(decimal.NewFromInt(455))) {
			t.Errorf("two weeks of HQ1 USD: %d weeks, total %s; want 2 weeks, 455", l.Entries, l.JamePS)
		}
	}
}

func TestWeeklySettlementLocksTheWeek(t *testing.T) {
	f := newWLFixture(t)
	f.assign(t, []models.Platform{f.p1, f.p2}, 0)
	if _, _, err := f.save(f.week, 0, canAll, "", row(f.p1, "1000", "200"), row(f.p2, "100000", "0")); err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.FinalizeWeek(f.ctx, FinalizeInput{BranchID: f.hq[0].ID.String(), WeekStart: dayString(f.week.AddDate(0, 0, 1))}, all, f.actor); err == nil {
		t.Error("a week starts on a Monday")
	}
	if _, err := f.svc.FinalizeWeek(f.ctx, FinalizeInput{BranchID: f.hq[0].ID.String(), WeekStart: dayString(weekStart(f.svc.today()))}, all, f.actor); err == nil {
		t.Error("the current week can't be finalized before its Sunday")
	}
	st, err := f.svc.FinalizeWeek(f.ctx, FinalizeInput{BranchID: f.hq[0].ID.String(), WeekStart: dayString(f.week)}, all, f.actor)
	if err != nil {
		t.Fatal(err)
	}
	if st.Status != models.SettlementFinalized || len(st.Totals) != 2 {
		t.Fatalf("finalized with totals per currency, got %s and %d totals", st.Status, len(st.Totals))
	}
	if _, _, err := f.save(f.week, 0, canAll, "late", row(f.p1, "1", "0")); err == nil || !strings.Contains(err.Error(), "finalized") {
		t.Errorf("a finalized week can't be changed, got %v", err)
	}
	if _, err := f.svc.ReopenWeek(f.ctx, st.ID, "", all, f.actor); err == nil {
		t.Error("reopening needs a reason")
	}
	if _, err := f.svc.ReopenWeek(f.ctx, st.ID, "late result from HQ", all, f.actor); err != nil {
		t.Fatal(err)
	}
	if _, _, err := f.save(f.week, 0, canAll, "late result from HQ", row(f.p1, "1100", "200")); err != nil {
		t.Errorf("a reopened week can be changed, got %v", err)
	}
	again, err := f.svc.FinalizeWeek(f.ctx, FinalizeInput{BranchID: f.hq[0].ID.String(), WeekStart: dayString(f.week)}, all, f.actor)
	if err != nil {
		t.Fatal(err)
	}
	for _, total := range again.Totals {
		if total.Currency == "USD" && !total.JamePS.Equal(decimal.NewFromInt(455)) { // (1100 + 200) × 35%
			t.Errorf("finalizing again works the totals out afresh: USD %s, want 455", total.JamePS)
		}
	}
}

func TestPartners(t *testing.T) {
	f := newWLFixture(t)
	f.db.Exec("DELETE FROM partners WHERE name LIKE 'TWL%'")
	save := func(name, rate string) models.Partner {
		p, err := f.svc.SavePartner(f.ctx, nil, PartnerInput{Name: name, DefaultRate: rate})
		if err != nil {
			t.Fatal(err)
		}
		return *p
	}
	a, b, c := save("TWL Sok Dara", "40"), save("TWL Chan Lina", "10"), save("TWL Dara Vy", "70")
	if _, err := f.svc.SavePartner(f.ctx, nil, PartnerInput{Name: "TWL Sok Dara", DefaultRate: "5"}); err == nil {
		t.Error("a second partner with the same name must be refused")
	}
	n := func(v int64) decimal.Decimal { return decimal.NewFromInt(v) }

	// Several partners per assignment, each at their default %.
	start := dayString(f.week.AddDate(0, 0, -21))
	la, err := f.svc.Assign(f.ctx, AssignmentInput{PlatformIDs: []string{f.p1.ID.String()}, BranchIDs: f.ids(0), PartnerIDs: []string{a.ID.String(), b.ID.String()}, StartsOn: start}, all, f.actor)
	if err != nil || len(la[0].Partners) != 2 || !la[0].CommissionRate.Equal(n(35)) {
		t.Fatalf("two partners on the assignment: %v", err)
	}
	rates := map[uuid.UUID]string{}
	for _, p := range la[0].Partners {
		rates[p.PartnerID] = p.CommissionRate.String()
	}
	if rates[a.ID] != "40" || rates[b.ID] != "10" {
		t.Errorf("each partner at their default %%, got %v", rates)
	}
	lb, err := f.svc.Assign(f.ctx, AssignmentInput{PlatformIDs: []string{f.p2.ID.String()}, BranchIDs: f.ids(0, 1), PartnerIDs: []string{b.ID.String()}, StartsOn: start}, all, f.actor)
	if err != nil || len(lb) != 2 || len(lb[1].Partners) != 1 {
		t.Fatalf("one partner on two branches: %v", err)
	}
	if _, err := f.svc.Assign(f.ctx, AssignmentInput{PlatformIDs: []string{f.p1.ID.String()}, BranchIDs: f.ids(3), PartnerIDs: []string{a.ID.String(), c.ID.String()}}, all, f.actor); err == nil {
		t.Error("partners sharing more than 100% must be refused")
	}

	// Each partner's Total is at their %, and the platform's Total is the sum.
	sheet, _, err := f.save(f.week, 0, canAll, "", row(f.p1, "1000", "0"), row(f.p2, "100000", "0"))
	if err != nil {
		t.Fatal(err)
	}
	r1 := sheetRow(t, sheet, f.p1)
	if len(r1.Partners) != 2 || !r1.JamePS.Equal(n(500)) || !r1.CommissionRate.Equal(n(50)) {
		t.Fatalf("total 500 at 50%% (40 + 10), got %s at %s with %d partners", r1.JamePS, r1.CommissionRate, len(r1.Partners))
	}
	if p := r1.Partners[0]; p.PartnerName != b.Name || !p.JamePS.Equal(n(100)) { // by name: Chan Lina first
		t.Errorf("Chan Lina's share: %s %v", p.PartnerName, p.JamePS)
	}
	if p := r1.Partners[1]; p.PartnerName != a.Name || !p.JamePS.Equal(n(400)) {
		t.Errorf("Sok Dara's share: %s %v", p.PartnerName, p.JamePS)
	}

	// New partners on the assignment apply to new weeks; the saved week keeps its partners.
	if _, _, err := f.svc.UpdateAssignment(f.ctx, la[0].ID, AssignmentUpdateInput{Partners: []AssignmentPartnerInput{{PartnerID: b.ID.String(), CommissionRate: "25"}}, Currency: "USD"}, all); err != nil {
		t.Fatal(err)
	}
	next, _, err := f.save(f.week.AddDate(0, 0, 7), 0, canAll, "", row(f.p1, "500", "0"))
	if err != nil {
		t.Fatal(err)
	}
	if r := sheetRow(t, next, f.p1); len(r.Partners) != 1 || !r.JamePS.Equal(n(125)) {
		t.Errorf("the next week has Chan Lina alone at 25%%: total %s", r.JamePS)
	}
	// Correcting the saved week works each partner's share out again at their saved %.
	fixed, _, err := f.save(f.week, 0, canAll, "typo", row(f.p1, "1100", "0"), row(f.p2, "100000", "0"))
	if err != nil {
		t.Fatal(err)
	}
	if r := sheetRow(t, fixed, f.p1); len(r.Partners) != 2 || !r.JamePS.Equal(n(550)) || !r.Partners[1].JamePS.Equal(n(440)) {
		t.Errorf("corrected: total %s, want 550 = 110 + 440", r.JamePS)
	}

	// Each line shows its partners' shares; totals per partner; the statement is the partner's share.
	r, err := f.svc.Report(f.ctx, ReportFilter{DateFrom: dayString(f.week), DateTo: dayString(f.week.AddDate(0, 0, 13))}, all, true, f.actor)
	if err != nil {
		t.Fatal(err)
	}
	for _, l := range r.Lines {
		if l.PlatformID == f.p1.ID {
			if !l.JamePS.Equal(n(675)) || !l.WinLose.Equal(n(1600)) || len(l.Partners) != 2 || !l.Partners[0].JamePS.Equal(n(235)) || !l.Partners[1].JamePS.Equal(n(440)) {
				t.Errorf("line: total %s, win/lose %s, shares %+v", l.JamePS, l.WinLose, l.Partners)
			}
		}
	}
	totals := map[string]string{}
	for _, p := range r.Partners {
		for _, s := range p.Totals {
			totals[p.PartnerName+" "+s.Currency] = s.JamePS.String()
		}
	}
	if totals[a.Name+" USD"] != "440" || totals[b.Name+" USD"] != "235" || totals[b.Name+" KHR"] != "10000" || totals[" USD"] != "" {
		t.Errorf("totals per partner: %v", totals)
	}
	st, err := f.svc.Report(f.ctx, ReportFilter{DateFrom: dayString(f.week), DateTo: dayString(f.week.AddDate(0, 0, 13)), PartnerID: &b.ID, ByWeek: true}, all, true, f.actor)
	if err != nil {
		t.Fatal(err)
	}
	weeks := map[string]string{}
	for _, l := range st.Lines {
		if l.PlatformID == f.p1.ID {
			weeks[l.WeekStart] = l.JamePS.String()
		}
	}
	if weeks[dayString(f.week)] != "110" || weeks[dayString(f.week.AddDate(0, 0, 7))] != "125" {
		t.Errorf("the statement has Chan Lina's share each week, got %v", weeks)
	}
	if len(st.Totals) != 2 || st.Totals[1].Currency != "USD" || !st.Totals[1].JamePS.Equal(n(235)) {
		t.Errorf("statement totals: %+v", st.Totals)
	}

	// A partner in use can't be turned off; one no longer used can, and can't be given again.
	off := false
	if _, err := f.svc.SavePartner(f.ctx, &b.ID, PartnerInput{Name: b.Name, DefaultRate: "10", IsActive: &off}); err == nil {
		t.Error("turning off a partner with active assignments must be refused")
	}
	if _, err := f.svc.SavePartner(f.ctx, &a.ID, PartnerInput{Name: a.Name, DefaultRate: "40", IsActive: &off}); err != nil {
		t.Errorf("a partner no longer used can be turned off: %v", err)
	}
	if _, err := f.svc.Assign(f.ctx, AssignmentInput{PlatformIDs: []string{f.p1.ID.String()}, BranchIDs: f.ids(3), PartnerIDs: []string{a.ID.String()}}, all, f.actor); err == nil {
		t.Error("a turned-off partner can't be given to an assignment")
	}
	if _, _, err := f.svc.UpdateAssignment(f.ctx, lb[0].ID, AssignmentUpdateInput{Partners: []AssignmentPartnerInput{{PartnerID: b.ID.String(), CommissionRate: "10"}, {PartnerID: a.ID.String(), CommissionRate: "5"}}, Currency: "KHR"}, all); err == nil {
		t.Error("a turned-off partner can't be added to an assignment")
	}
	// Moving keeps the partners and their %.
	_, moved, err := f.svc.MoveAssignment(f.ctx, lb[1].ID, AssignmentMoveInput{BranchID: f.hq[2].ID.String(), On: dayString(f.week.AddDate(0, 0, 14))}, all, f.actor)
	if err != nil || len(moved.Partners) != 1 || moved.Partners[0].PartnerID != b.ID || !moved.Partners[0].CommissionRate.Equal(n(10)) {
		t.Errorf("moving keeps the partners: %v", err)
	}
	list, err := f.svc.ListPartners(f.ctx, true)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range list {
		if p.ID == b.ID && p.ActiveAssignments != 3 { // hq0 p1, hq0 p2 and the moved one
			t.Errorf("Chan Lina is on 3 active assignments, got %d", p.ActiveAssignments)
		}
	}
	rows, err := f.svc.ListAssignments(f.ctx, AssignmentFilter{BranchID: &f.hq[0].ID}, all)
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range rows {
		if row.PlatformID == f.p1.ID && (len(row.Partners) != 1 || row.Partners[0].PartnerName != b.Name || row.Partners[0].CommissionRate.String() != "25") {
			t.Errorf("the assignment lists its partner: %+v", row.Partners)
		}
	}
}

func TestWeeklyEntries(t *testing.T) {
	f := newWLFixture(t)
	f.assign(t, []models.Platform{f.p1}, 0, 1)
	// The entry goes in the person's own group: their default branch.
	f.db.Model(&models.User{}).Where("id = ?", f.actor).Update("default_branch_id", f.hq[1].ID)
	in := WeekEntryInput{WeekStart: dayString(f.week), Rows: []WeekEntryRow{row(f.p1, "1000", "0")}}
	if _, _, err := f.svc.SaveWeek(f.ctx, in, all, canAll, f.actor); err == nil {
		t.Error("an entry needs a title")
	}
	in.Title = "  Week 1 TWL2  "
	sheet, _, err := f.svc.SaveWeek(f.ctx, in, all, canAll, f.actor)
	if err != nil {
		t.Fatal(err)
	}
	if sheet.BranchID != f.hq[1].ID || sheet.Entry == nil || sheet.Entry.Title != "Week 1 TWL2" || !sheet.Rows[0].Saved {
		t.Fatalf("saved under the default branch with its title: %s %+v", sheet.BranchCode, sheet.Entry)
	}
	// Without a default branch and with several branches there is no group to use.
	f.db.Model(&models.User{}).Where("id = ?", f.actor).Update("default_branch_id", nil)
	f.db.Exec("DELETE FROM user_branches WHERE user_id = ?", f.actor)
	if _, err := f.svc.WeekSheet(f.ctx, dayString(f.week), "", all, f.actor); err == nil {
		t.Error("no default branch and no single branch: the own group can't be worked out")
	}
	// Renaming needs Correct Win/Lose; the amounts can be left as they are.
	in.BranchID, in.Title = f.hq[1].ID.String(), "Week 1 (checked)"
	onlyCreate := func(code string) bool { return code == permissions.WinLoseCreate }
	if _, _, err := f.svc.SaveWeek(f.ctx, in, all, onlyCreate, f.actor); err == nil {
		t.Error("changing the title of a saved entry needs Correct Win/Lose")
	}
	_, corrections, err := f.svc.SaveWeek(f.ctx, in, all, canAll, f.actor)
	if err != nil || len(corrections) != 1 || corrections[0].After["title"] != "Week 1 (checked)" {
		t.Errorf("the rename is one correction for the audit log: %v %+v", err, corrections)
	}
	// The list: newest first, only the groups in scope.
	if _, _, err := f.save(f.week.AddDate(0, 0, 7), 0, canAll, "", row(f.p1, "5", "0")); err != nil {
		t.Fatal(err)
	}
	list, err := f.svc.ListEntries(f.ctx, "", all)
	if err != nil || len(list) != 2 || list[0].BranchID != f.hq[0].ID || list[1].Title != "Week 1 (checked)" {
		t.Errorf("both entries, newest first: %v %+v", err, list)
	}
	if mine, _ := f.svc.ListEntries(f.ctx, "", f.scope(1)); len(mine) != 1 || mine[0].BranchID != f.hq[1].ID {
		t.Errorf("only the groups in scope: %+v", mine)
	}
	if found, _ := f.svc.ListEntries(f.ctx, "checked", all); len(found) != 1 {
		t.Errorf("search by title: %+v", found)
	}
}

func TestAssignmentGroups(t *testing.T) {
	f := newWLFixture(t)
	f.db.Exec("DELETE FROM partners WHERE name LIKE 'TWL%'")
	f.db.Model(&models.User{}).Where("id = ?", f.actor).Update("default_branch_id", f.hq[0].ID)
	defer f.db.Model(&models.User{}).Where("id = ?", f.actor).Update("default_branch_id", nil)
	partner, err := f.svc.SavePartner(f.ctx, nil, PartnerInput{Name: "TWL Group Partner", DefaultRate: "20"})
	if err != nil {
		t.Fatal(err)
	}
	ids := func(ps ...models.Platform) []string {
		out := []string{}
		for _, p := range ps {
			out = append(out, p.ID.String())
		}
		return out
	}
	start := dayString(f.week.AddDate(0, 0, -21))

	// A titled group in the person's own branch, its agents with the partner at their default %.
	if _, err := f.svc.CreateGroup(f.ctx, GroupInput{Title: " ", PlatformIDs: ids(f.p1)}, all, f.actor); err == nil {
		t.Error("a group needs a title")
	}
	g, err := f.svc.CreateGroup(f.ctx, GroupInput{Title: "  V-series  ", PlatformIDs: ids(f.p1, f.p2), PartnerIDs: []string{partner.ID.String()}, StartsOn: start}, all, f.actor)
	if err != nil || g.Title != "V-series" || g.BranchID != f.hq[0].ID || len(g.Agents) != 2 || len(g.Partners) != 1 || g.Partners[0].CommissionRate.String() != "20" {
		t.Fatalf("group: %v %+v", err, g)
	}
	if _, err := f.svc.CreateGroup(f.ctx, GroupInput{Title: "Other", PlatformIDs: ids(f.p1)}, all, f.actor); err == nil || !strings.Contains(err.Error(), "V-series") {
		t.Errorf("an agent is in one group at a time, got %v", err)
	}
	if _, _, err := f.save(f.week, 0, canAll, "", row(f.p1, "1000", "0")); err != nil {
		t.Fatal(err)
	}

	// Off: its agents leave Data Entry (the saved week stays) and the Agents page's count.
	if _, err := f.svc.SetGroupActive(f.ctx, g.ID, false, all); err != nil {
		t.Fatal(err)
	}
	if sheet, _ := f.svc.WeekSheet(f.ctx, dayString(f.week), f.hq[0].ID.String(), all, f.actor); len(sheet.Rows) != 1 || !sheet.Rows[0].Saved {
		t.Errorf("off: only the saved agent stays on its week, got %d rows", len(sheet.Rows))
	}
	if next, _ := f.svc.WeekSheet(f.ctx, dayString(f.week.AddDate(0, 0, 7)), f.hq[0].ID.String(), all, f.actor); len(next.Rows) != 0 {
		t.Errorf("off: no agents on a new week, got %d", len(next.Rows))
	}
	agents, _ := f.svc.ListPlatforms(f.ctx, true)
	for _, a := range agents {
		if a.ID == f.p2.ID && a.ActiveHQs != 0 {
			t.Errorf("an off group doesn't count for the agent, got %d", a.ActiveHQs)
		}
	}
	if _, err := f.svc.SetGroupActive(f.ctx, g.ID, true, all); err != nil {
		t.Fatal(err)
	}
	agents, _ = f.svc.ListPlatforms(f.ctx, true)
	for _, a := range agents {
		if a.ID == f.p2.ID && a.ActiveHQs != 1 {
			t.Errorf("a switched-on group counts for the agent, got %d", a.ActiveHQs)
		}
	}
	if next, _ := f.svc.WeekSheet(f.ctx, dayString(f.week.AddDate(0, 0, 7)), f.hq[0].ID.String(), all, f.actor); len(next.Rows) != 2 {
		t.Errorf("on again: both agents, got %d", len(next.Rows))
	}

	// Update: rename, take out the agent without weeks (removed), new % for the partner.
	_, after, err := f.svc.UpdateGroup(f.ctx, g.ID, GroupUpdateInput{Title: "V1 only", PlatformIDs: ids(f.p1),
		Partners: []AssignmentPartnerInput{{PartnerID: partner.ID.String(), CommissionRate: "30"}}}, all, f.actor)
	if err != nil || after.Title != "V1 only" || len(after.Agents) != 1 || after.Partners[0].CommissionRate.String() != "30" {
		t.Fatalf("update: %v %+v", err, after)
	}
	var gone int64
	f.db.Unscoped().Model(&models.HQPlatformAssignment{}).Where("platform_id = ? AND group_id = ?", f.p2.ID, g.ID).Count(&gone)
	if gone != 0 {
		t.Error("an agent taken out before any week is entered is removed")
	}

	// Delete: only while no week is entered.
	other, err := f.svc.CreateGroup(f.ctx, GroupInput{Title: "P2", PlatformIDs: ids(f.p2)}, all, f.actor)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.DeleteGroup(f.ctx, g.ID, all); err == nil {
		t.Error("a group with weeks entered can't be deleted")
	}
	if _, err := f.svc.DeleteGroup(f.ctx, other.ID, all); err != nil {
		t.Fatalf("a group without weeks can be deleted: %v", err)
	}
	// Taking out an agent with weeks ends it (history); one put in starts today.
	_, after, err = f.svc.UpdateGroup(f.ctx, g.ID, GroupUpdateInput{Title: "V1 only", PlatformIDs: ids(f.p2)}, all, f.actor)
	if err != nil || len(after.Agents) != 1 || after.Agents[0].PlatformID != f.p2.ID || after.Agents[0].StartsOn != dayString(f.svc.today()) || after.Weeks != 1 {
		t.Fatalf("swap agents: %v %+v", err, after)
	}
	var ended models.HQPlatformAssignment
	if err := f.db.Where("platform_id = ? AND group_id = ?", f.p1.ID, g.ID).First(&ended).Error; err != nil || ended.IsActive || ended.EndsOn == nil {
		t.Errorf("the agent with weeks is ended, not removed: %v %+v", err, ended)
	}
	list, err := f.svc.ListGroups(f.ctx, f.hq[0].ID, GroupFilter{}, all)
	if err != nil || len(list) != 1 || list[0].ID != g.ID {
		t.Errorf("one group left: %v %d", err, len(list))
	}
}
