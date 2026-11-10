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
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/cambodia-finance/income-expense/internal/database"
	"github.com/cambodia-finance/income-expense/internal/models"
	"github.com/cambodia-finance/income-expense/internal/permissions"
	"github.com/cambodia-finance/income-expense/internal/winlose"
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
	// entries: each HQ's entry for a week once saved (by HQ and Monday), as Data Entry keeps it.
	entries map[string]string
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
	f := &wlFixture{db: db, svc: NewPlatformService(db, "Asia/Phnom_Penh"), ctx: context.Background(), entries: map[string]string{}}
	for _, table := range []string{"weekly_settlement_totals", "weekly_settlements", "weekly_group_results", "weekly_partner_results", "weekly_result_reactions", "weekly_platform_results", "weekly_entry_reactions", "weekly_entries", "assignment_partners", "hq_platform_assignments", "assignment_groups"} {
		db.Exec("DELETE FROM " + table)
	}
	db.Exec("DELETE FROM platforms WHERE code LIKE 'TWL%'")
	// The amounts in these tests are worked out with v1 (W/L + commission); TestFormulaChange has v2 and v3.
	if err := db.Save(&models.AppSetting{Key: FormulaSettingKey, Value: "v1"}).Error; err != nil {
		t.Fatal(err)
	}
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

func (f *wlFixture) entryKey(week time.Time, hq int) string {
	return f.hq[hq].ID.String() + dayString(weekStart(week))
}

// save saves an HQ's week on its entry: the first save makes the entry and the ones after change
// it, as Data Entry keeps the entry it is on.
func (f *wlFixture) save(week time.Time, hq int, can func(string) bool, reason string, rows ...WeekEntryRow) (*WeekSheet, []Correction, error) {
	key := f.entryKey(week, hq)
	sheet, corrections, err := f.svc.SaveWeek(f.ctx, WeekEntryInput{EntryID: f.entries[key], WeekStart: dayString(week), BranchID: f.hq[hq].ID.String(), Title: "TWL week", Rows: rows, Reason: reason}, all, can, f.actor)
	if err == nil && sheet.Entry != nil {
		f.entries[key] = sheet.Entry.ID.String()
	}
	return sheet, corrections, err
}

// sheet is an HQ's entry for the week, or a new one when none was saved.
func (f *wlFixture) sheet(t *testing.T, week time.Time, hq int) *WeekSheet {
	t.Helper()
	s, err := f.svc.EntrySheet(f.ctx, f.entries[f.entryKey(week, hq)], dayString(week), f.hq[hq].ID.String(), all, f.actor)
	if err != nil {
		t.Fatal(err)
	}
	return s
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

func TestEntryReactions(t *testing.T) {
	f := newWLFixture(t)
	f.assign(t, []models.Platform{f.p1}, 0)
	sheet, _, err := f.save(f.week, 0, canAll, "", row(f.p1, "1000", "200"))
	if err != nil {
		t.Fatal(err)
	}
	entry := sheet.Entry.ID
	other := models.User{Username: "twl_other", Email: "twl_other@example.test", PasswordHash: "x", FullName: "TWL Other", IsActive: true}
	if err := f.db.Where(models.User{Username: other.Username}).FirstOrCreate(&other).Error; err != nil {
		t.Fatal(err)
	}
	react := func(who uuid.UUID, reaction, want string, wantList ...string) {
		t.Helper()
		_, now, list, err := f.svc.ReactToEntry(f.ctx, entry, reaction, all, who)
		if err != nil {
			t.Fatal(err)
		}
		got := []string{}
		for _, r := range list {
			got = append(got, r.UserName+" "+r.Reaction)
		}
		// (Two given in the same millisecond can come in either order.)
		sort.Strings(got)
		sort.Strings(wantList)
		if now != want || strings.Join(got, ", ") != strings.Join(wantList, ", ") {
			t.Errorf("%s: now %q with %v, want %q with %v", reaction, now, got, want, wantList)
		}
	}
	react(f.actor, "yes", "yes", "TWL Tester yes")
	react(other.ID, "no", "no", "TWL Tester yes", "TWL Other no")
	react(f.actor, "no", "no", "TWL Other no", "TWL Tester no") // changed: one each
	react(f.actor, "no", "", "TWL Other no")                    // the same again takes it back
	if _, _, _, err := f.svc.ReactToEntry(f.ctx, entry, "maybe", all, f.actor); err == nil {
		t.Error("only yes or no")
	}
	if _, _, _, err := f.svc.ReactToEntry(f.ctx, entry, "yes", f.scope(1), f.actor); err == nil {
		t.Error("an entry of another group can't be reacted to")
	}
	// Data entry's list has them too.
	if list, err := f.svc.ListEntries(f.ctx, EntryFilter{}, all); err != nil || len(list) != 1 || len(list[0].Reactions) != 1 || list[0].Reactions[0].Reaction != "no" {
		t.Errorf("reactions on the list: %v %+v", err, list)
	}
	// The report by entry has them; deleting the entry removes them.
	r, err := f.svc.Report(f.ctx, ReportFilter{DateFrom: dayString(f.week), DateTo: dayString(f.week.AddDate(0, 0, 6)), ByEntry: true}, all, true, f.actor)
	if err != nil || len(r.Entries) != 1 || len(r.Entries[0].Reactions) != 1 || r.Entries[0].Reactions[0].UserName != "TWL Other" {
		t.Fatalf("reactions in the report: %v %+v", err, r)
	}
	if _, _, err := f.svc.DeleteEntry(f.ctx, entry, "wrong week", all); err != nil {
		t.Fatal(err)
	}
	var left int64
	f.db.Model(&models.WeeklyEntryReaction{}).Where("entry_id = ?", entry).Count(&left)
	if left != 0 {
		t.Errorf("%d reactions left after deleting the entry", left)
	}
}

func TestAgentReactions(t *testing.T) {
	f := newWLFixture(t)
	f.assign(t, []models.Platform{f.p1, f.p2}, 0)
	sheet, _, err := f.save(f.week, 0, canAll, "", row(f.p1, "1000", "200"))
	if err != nil {
		t.Fatal(err)
	}
	r1, r2 := sheetRow(t, sheet, f.p1), sheetRow(t, sheet, f.p2)
	if r1.ResultID == nil || r2.ResultID != nil || len(r1.Reactions) != 0 {
		t.Fatalf("a saved agent has its result (and no reactions yet), one not entered none: %+v %+v", r1, r2)
	}
	other := models.User{Username: "twl_other", Email: "twl_other@example.test", PasswordHash: "x", FullName: "TWL Other", IsActive: true}
	if err := f.db.Where(models.User{Username: other.Username}).FirstOrCreate(&other).Error; err != nil {
		t.Fatal(err)
	}
	if _, now, list, err := f.svc.ReactToResult(f.ctx, *r1.ResultID, "yes", all, f.actor); err != nil || now != "yes" || len(list) != 1 {
		t.Fatalf("✅ on the agent: %v %q %v", err, now, list)
	}
	if _, now, _, err := f.svc.ReactToResult(f.ctx, *r1.ResultID, "no", all, other.ID); err != nil || now != "no" {
		t.Fatalf("❌ from someone else: %v %q", err, now)
	}
	if _, _, _, err := f.svc.ReactToResult(f.ctx, *r1.ResultID, "yes", f.scope(1), f.actor); err == nil {
		t.Error("a result of another group can't be reacted to")
	}
	// The entry shows them on the agent's row, apart from the entry's own.
	got := []string{}
	for _, r := range sheetRow(t, f.sheet(t, f.week, 0), f.p1).Reactions {
		got = append(got, r.UserName+" "+r.Reaction)
	}
	sort.Strings(got)
	if strings.Join(got, ", ") != "TWL Other no, TWL Tester yes" {
		t.Errorf("the agent's reactions: %v", got)
	}
	if list, _ := f.svc.ListEntries(f.ctx, EntryFilter{}, all); len(list) != 1 || len(list[0].Reactions) != 0 {
		t.Errorf("the entry itself has none: %+v", list)
	}
	// The reports by entry show them on the agent's line in the entry; lines that add results up
	// have none.
	checkReportReactions(t, f, nil, sheet.Entry.ID, *r1.ResultID, "TWL Other no, TWL Tester yes")
	// A correction keeps them; deleting the entry removes them.
	if _, _, err := f.save(f.week, 0, canAll, "typo", row(f.p1, "1100", "200")); err != nil {
		t.Fatal(err)
	}
	if len(sheetRow(t, f.sheet(t, f.week, 0), f.p1).Reactions) != 2 {
		t.Error("a corrected agent keeps its reactions")
	}
	if _, _, err := f.svc.DeleteEntry(f.ctx, sheet.Entry.ID, "wrong week", all); err != nil {
		t.Fatal(err)
	}
	var left int64
	f.db.Model(&models.WeeklyResultReaction{}).Where("result_id = ?", *r1.ResultID).Count(&left)
	if left != 0 {
		t.Errorf("%d agent reactions left after deleting the entry", left)
	}
}

// checkReportReactions: in the reports by entry (all of it, and one entry; for one group with
// groupID) the agent's line in the entry is its result with these reactions, and the lines that
// add results up have none.
func checkReportReactions(t *testing.T, f *wlFixture, groupID *uuid.UUID, entryID, resultID uuid.UUID, want string) {
	t.Helper()
	for name, filter := range map[string]ReportFilter{
		"all":      {ByEntry: true, GroupID: groupID},
		"an entry": {ByEntry: true, GroupID: groupID, EntryID: &entryID},
	} {
		filter.DateFrom, filter.DateTo = dayString(f.week), dayString(f.week.AddDate(0, 0, 6))
		rep, err := f.svc.Report(f.ctx, filter, all, true, f.actor)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if len(rep.Lines) != 1 || rep.Lines[0].ResultID != nil || rep.Lines[0].Reactions != nil {
			t.Errorf("%s: the added-up line has no result or reactions: %+v", name, rep.Lines)
		}
		if len(rep.Entries) != 1 || len(rep.Entries[0].Lines) != 1 {
			t.Fatalf("%s: one entry with p1: %+v", name, rep.Entries)
		}
		l := rep.Entries[0].Lines[0]
		got := []string{}
		for _, r := range l.Reactions {
			got = append(got, r.UserName+" "+r.Reaction)
		}
		sort.Strings(got)
		if l.ResultID == nil || *l.ResultID != resultID || strings.Join(got, ", ") != want {
			t.Errorf("%s: the entry's line has the agent's result and its reactions: %v %v", name, l.ResultID, got)
		}
	}
}

// Group summary's group: its agents in an entry have their ✅ and ❌ (the same as in Data entry).
func TestAgentReactionsInAGroup(t *testing.T) {
	f := newWLFixture(t)
	f.db.Model(&models.User{}).Where("id = ?", f.actor).Update("default_branch_id", f.hq[0].ID)
	defer f.db.Model(&models.User{}).Where("id = ?", f.actor).Update("default_branch_id", nil)
	g, err := f.svc.CreateGroup(f.ctx, GroupInput{Title: "Reacted", Agents: []GroupAgentInput{{PlatformID: f.p1.ID.String(), CommissionRate: "10"}}, StartsOn: dayString(f.week)}, all, f.actor)
	if err != nil {
		t.Fatal(err)
	}
	sheet, _, err := f.save(f.week, 0, canAll, "", row(f.p1, "1000", "200"))
	if err != nil {
		t.Fatal(err)
	}
	r1 := sheetRow(t, sheet, f.p1)
	if r1.ResultID == nil || len(r1.Groups) != 1 {
		t.Fatalf("p1 is saved in the group: %+v", r1)
	}
	checkReportReactions(t, f, &g.ID, sheet.Entry.ID, *r1.ResultID, "")
	if _, _, _, err := f.svc.ReactToResult(f.ctx, *r1.ResultID, "yes", all, f.actor); err != nil {
		t.Fatal(err)
	}
	checkReportReactions(t, f, &g.ID, sheet.Entry.ID, *r1.ResultID, "TWL Tester yes")
}

func TestDeletingAnAgent(t *testing.T) {
	f := newWLFixture(t)
	f.assign(t, []models.Platform{f.p1}, 0)
	if _, err := f.svc.DeletePlatform(f.ctx, f.p1.ID); err == nil || !strings.Contains(err.Error(), "turn it off") {
		t.Errorf("an agent in a group can't be deleted, got %v", err)
	}
	unused := f.platform(t, "TWL_UNUSED", "0", "USD")
	if p, err := f.svc.DeletePlatform(f.ctx, unused.ID); err != nil || p.Code != "TWL_UNUSED" {
		t.Fatalf("an unused agent is deleted: %v", err)
	}
	var left int64
	f.db.Unscoped().Model(&models.Platform{}).Where("id = ?", unused.ID).Count(&left)
	if left != 0 {
		t.Error("the deleted agent is gone for good, so its user can be used again")
	}
	if _, err := f.svc.DeletePlatform(f.ctx, unused.ID); !errors.Is(err, ErrPlatformNotFound) {
		t.Errorf("deleting it again: %v", err)
	}
}

// The RP permissions that took over from the two "manage" ones (and from Correct Win/Lose for
// deleting) are given to the roles that had those, and the old ones are removed.
func TestRPPermissionsTakeOver(t *testing.T) {
	f := newWLFixture(t)
	perm := func(code string) models.Permission {
		var p models.Permission
		if err := f.db.Where(models.Permission{Code: code}).Attrs(models.Permission{Base: models.Base{ID: uuid.New()}, Name: code, Module: "test"}).FirstOrCreate(&p).Error; err != nil {
			t.Fatal(err)
		}
		return p
	}
	// A role from before: it could manage agents and groups, correct and see Data entry.
	role := models.Role{Base: models.Base{ID: uuid.New()}, Code: "twl_rp_role", Name: "TWL RP"}
	f.db.Unscoped().Where("code = ?", role.Code).Delete(&models.Role{})
	if err := f.db.Create(&role).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		f.db.Exec("DELETE FROM role_permissions WHERE role_id = ?", role.ID)
		f.db.Unscoped().Delete(&role)
	})
	old := []models.Permission{perm("platforms.manage"), perm("platform-assignments.manage"), perm(permissions.WinLoseEdit), perm(permissions.WinLoseView)}
	if err := f.db.Model(&role).Association("Permissions").Replace(old); err != nil {
		t.Fatal(err)
	}
	// The new ones aren't there yet (an upgrade).
	added := []string{permissions.PlatformsCreate, permissions.PlatformsEdit, permissions.PlatformsDelete, permissions.PlatformAssignmentsView,
		permissions.PlatformAssignmentsCreate, permissions.PlatformAssignmentsEdit, permissions.PlatformAssignmentsDelete, permissions.WinLoseDelete, permissions.WinLoseReact}
	for _, code := range added {
		f.db.Exec("DELETE rp FROM role_permissions rp JOIN permissions p ON p.id = rp.permission_id WHERE p.code = ?", code)
		f.db.Unscoped().Where("code = ?", code).Delete(&models.Permission{})
	}
	if err := database.EnsureRequiredPermissions(f.db); err != nil {
		t.Fatal(err)
	}
	var codes []string
	f.db.Table("role_permissions").Joins("JOIN permissions ON permissions.id = role_permissions.permission_id").
		Where("role_permissions.role_id = ?", role.ID).Pluck("permissions.code", &codes)
	have := map[string]bool{}
	for _, c := range codes {
		have[c] = true
	}
	for _, code := range append(added, permissions.WinLoseEdit, permissions.WinLoseView) {
		if !have[code] {
			t.Errorf("the role lost %s (has %v)", code, codes)
		}
	}
	var retired int64
	f.db.Unscoped().Model(&models.Permission{}).Where("code IN ?", []string{"platforms.manage", "platform-assignments.manage"}).Count(&retired)
	if retired != 0 || have["platforms.manage"] || have["platform-assignments.manage"] {
		t.Errorf("the manage permissions are removed: %d left, role has %v", retired, codes)
	}
}

func TestTurningAnAgentOffNeedsAReason(t *testing.T) {
	f := newWLFixture(t)
	on, off := true, false
	in := PlatformInput{Code: f.p1.Code, Name: f.p1.Name, DefaultCurrency: f.p1.DefaultCurrency, IsActive: &off}
	if _, err := f.svc.SavePlatform(f.ctx, &f.p1.ID, in); err == nil || !strings.Contains(err.Error(), "reason") {
		t.Errorf("turning off without a reason must be refused, got %v", err)
	}
	in.Reason = "closed"
	if p, err := f.svc.SavePlatform(f.ctx, &f.p1.ID, in); err != nil || p.IsActive {
		t.Fatalf("turned off with a reason: %v", err)
	}
	in.IsActive, in.Reason = &on, ""
	if p, err := f.svc.SavePlatform(f.ctx, &f.p1.ID, in); err != nil || !p.IsActive {
		t.Errorf("turning on needs no reason: %v", err)
	}
}

func TestPlatformAssignmentsManyToMany(t *testing.T) {
	f := newWLFixture(t)
	// Two platforms to two HQs in one go; one platform also to a third HQ.
	if got := len(f.assign(t, []models.Platform{f.p1, f.p2}, 0, 1)); got != 4 {
		t.Fatalf("%d assignments, want 4 (2 platforms × 2 HQs)", got)
	}
	f.assign(t, []models.Platform{f.p1}, 2)
	sheet, err := f.svc.EntrySheet(f.ctx, "", dayString(f.week.AddDate(0, 0, 3)), f.hq[0].ID.String(), all, f.actor)
	if err != nil {
		t.Fatal(err)
	}
	if sheet.WeekStart != dayString(f.week) || sheet.Entry != nil || len(sheetRow(t, sheet, f.p1).Groups) != 1 {
		t.Errorf("a new entry for HQ1's week (from any day of it) from %s, want %s, with the agent's assignment", sheet.WeekStart, dayString(f.week))
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
	// The unique key stops a second active assignment of an agent in a group.
	one := int8(1)
	grp := models.AssignmentGroup{ID: uuid.New(), BranchID: f.hq[0].ID, Title: "TWL key", StartsOn: f.week, IsActive: true, CreatedByID: f.actor}
	if err := f.db.Create(&grp).Error; err != nil {
		t.Fatal(err)
	}
	first := models.HQPlatformAssignment{Base: models.Base{ID: uuid.New()}, BranchID: f.hq[0].ID, PlatformID: f.p1.ID, GroupID: &grp.ID, CommissionRate: decimal.Zero, Currency: "USD", IsActive: true, ActiveKey: &one, StartsOn: f.week, CreatedByID: f.actor}
	dup := first
	dup.ID = uuid.New()
	if err := f.db.Create(&first).Error; err != nil {
		t.Fatal(err)
	}
	if err := f.db.Create(&dup).Error; !isDuplicate(err) {
		t.Errorf("the unique key must stop a second active assignment in a group, got %v", err)
	}
	f.db.Unscoped().Delete(&first)
	f.db.Delete(&grp)
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
	if !sheetRow(t, f.sheet(t, f.week, 2), f.p1).Saved {
		t.Error("the old HQ's saved week stays")
	}
	if r := sheetRow(t, f.sheet(t, next, 2), f.p1); len(r.Groups) != 0 || !r.CommissionRate.IsZero() {
		t.Error("after the move the agent no longer counts at the old HQ")
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
	entryID := uuid.MustParse(f.entries[f.entryKey(f.week, 0)])
	dup := models.WeeklyPlatformResult{Base: models.Base{ID: uuid.New()}, EntryID: &entryID, WeekStart: f.week, WeekEnd: f.week.AddDate(0, 0, 6), BranchID: f.hq[0].ID, PlatformID: f.p1.ID,
		Currency: "USD", FormulaVersion: "v1", CreatedByID: f.actor}
	if err := f.db.Create(&dup).Error; !isDuplicate(err) {
		t.Errorf("the unique key must stop a second row for the agent on the entry, got %v", err)
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
	if r := sheetRow(t, f.sheet(t, f.week, 3), f.p1); len(r.Groups) != 1 {
		t.Error("an agent assigned on the Wednesday counts in that week")
	}
	// An agent in no group can be entered (its Total is 0); a week that hasn't started can't.
	free, _, err := f.save(f.week, 1, canAll, "", row(f.p1, "1", "1"))
	if err != nil {
		t.Fatalf("an agent in no group can be entered: %v", err)
	}
	if r := sheetRow(t, free, f.p1); !r.JamePS.IsZero() || len(r.Groups) != 0 {
		t.Errorf("an agent in no group has no Total: %s in %d groups", r.JamePS, len(r.Groups))
	}
	if _, _, err := f.save(weekStart(f.svc.today()).AddDate(0, 0, 7), 0, canAll, "", row(f.p1, "1", "1")); err == nil {
		t.Error("a future week must be refused")
	}
}

func TestWinLoseHQAccess(t *testing.T) {
	f := newWLFixture(t)
	f.assign(t, []models.Platform{f.p1}, 0, 1)
	only2 := f.scope(1)
	if _, err := f.svc.EntrySheet(f.ctx, "", dayString(f.week), f.hq[0].ID.String(), only2, f.actor); !errors.Is(err, ErrBranchAccessDenied) {
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
	// Each week's totals, earliest first: 420 USD and -38950 KHR, then 35 USD.
	weeks := []string{}
	for _, w := range two.Weeks {
		for _, s := range w.Totals {
			weeks = append(weeks, w.WeekStart+" "+w.WeekEnd[8:]+" "+s.Currency+" "+s.JamePS.String())
		}
	}
	if want := []string{
		dayString(f.week) + " " + dayString(f.week.AddDate(0, 0, 6))[8:] + " KHR -38950",
		dayString(f.week) + " " + dayString(f.week.AddDate(0, 0, 6))[8:] + " USD 420",
		dayString(f.week.AddDate(0, 0, 7)) + " " + dayString(f.week.AddDate(0, 0, 13))[8:] + " USD 35",
	}; strings.Join(weeks, ", ") != strings.Join(want, ", ") {
		t.Errorf("by week: %v, want %v", weeks, want)
	}
	// By data entry: the week's two entries (one per HQ), each with its agents and totals.
	byEntry, err := f.svc.Report(f.ctx, ReportFilter{DateFrom: dayString(f.week), DateTo: dayString(f.week.AddDate(0, 0, 6)), ByEntry: true}, all, true, f.actor)
	if err != nil {
		t.Fatal(err)
	}
	hq1, hq2 := f.entries[f.entryKey(f.week, 0)], f.entries[f.entryKey(f.week, 1)]
	entryOf := map[string]string{hq1: "HQ1", hq2: "HQ2"}
	got := map[string]bool{}
	lines := 0
	for _, e := range byEntry.Entries {
		for _, l := range e.Lines {
			got[entryOf[e.ID.String()]+" "+l.PlatformCode+" "+l.JamePS.String()] = true
			lines++
		}
		for _, s := range e.Totals {
			got[entryOf[e.ID.String()]+" "+e.Title+" "+e.CreatedBy+" "+s.Currency+" "+s.JamePS.String()] = true
		}
	}
	for _, w := range []string{"HQ1 " + f.p1.Code + " 420", "HQ1 " + f.p2.Code + " -38950", "HQ2 " + f.p1.Code + " -140",
		"HQ1 TWL week TWL Tester USD 420", "HQ1 TWL week TWL Tester KHR -38950", "HQ2 TWL week TWL Tester USD -140"} {
		if !got[w] {
			t.Errorf("by entry: %q missing from %v", w, got)
		}
	}
	// The report's own lines stay by agent (across both entries).
	if lines != 3 || len(byEntry.Entries) != 2 || len(byEntry.Lines) != 2 {
		t.Errorf("by entry: %d entry lines in %d entries and %d lines, want 3 in 2 and 2", lines, len(byEntry.Entries), len(byEntry.Lines))
	}
	// One entry only.
	id := uuid.MustParse(hq2)
	one, err := f.svc.Report(f.ctx, ReportFilter{DateFrom: dayString(f.week), DateTo: dayString(f.week.AddDate(0, 0, 6)), EntryID: &id, ByEntry: true}, all, true, f.actor)
	if err != nil || len(one.Lines) != 1 || !one.Lines[0].JamePS.Equal(decimal.NewFromInt(-140)) || len(one.Entries) != 1 || one.Entries[0].ID.String() != hq2 {
		t.Errorf("one entry: %v %+v", err, one)
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
	// (Next week: this week's Sunday may be today, and a week can be finalized from its Sunday.)
	if _, err := f.svc.FinalizeWeek(f.ctx, FinalizeInput{BranchID: f.hq[0].ID.String(), WeekStart: dayString(weekStart(f.svc.today()).AddDate(0, 0, 7))}, all, f.actor); err == nil {
		t.Error("a week can't be finalized before its Sunday")
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
	if sheet.BranchID != f.hq[1].ID || sheet.Entry == nil || sheet.Entry.Title != "Week 1 TWL2" || !sheetRow(t, sheet, f.p1).Saved {
		t.Fatalf("saved under the default branch with its title: %s %+v", sheet.BranchCode, sheet.Entry)
	}
	// Without a default branch and with several branches there is no group to use.
	f.db.Model(&models.User{}).Where("id = ?", f.actor).Update("default_branch_id", nil)
	f.db.Exec("DELETE FROM user_branches WHERE user_id = ?", f.actor)
	if _, err := f.svc.EntrySheet(f.ctx, "", dayString(f.week), "", all, f.actor); err == nil {
		t.Error("no default branch and no single branch: the own group can't be worked out")
	}
	// Renaming needs Correct Win/Lose; the amounts can be left as they are.
	in.EntryID, in.Title = sheet.Entry.ID.String(), "Week 1 (checked)"
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
	list, err := f.svc.ListEntries(f.ctx, EntryFilter{}, all)
	if err != nil || len(list) != 2 || list[0].BranchID != f.hq[0].ID || list[1].Title != "Week 1 (checked)" {
		t.Errorf("both entries, newest first: %v %+v", err, list)
	}
	if mine, _ := f.svc.ListEntries(f.ctx, EntryFilter{}, f.scope(1)); len(mine) != 1 || mine[0].BranchID != f.hq[1].ID {
		t.Errorf("only the groups in scope: %+v", mine)
	}
	if found, _ := f.svc.ListEntries(f.ctx, EntryFilter{Search: "checked"}, all); len(found) != 1 {
		t.Errorf("search by title: %+v", found)
	}
	// Filter by the day they were created (the business's days).
	today, yesterday := dayString(f.svc.today()), dayString(f.svc.today().AddDate(0, 0, -1))
	if found, _ := f.svc.ListEntries(f.ctx, EntryFilter{CreatedFrom: today, CreatedTo: today}, all); len(found) != 2 {
		t.Errorf("both entries were made today, got %d", len(found))
	}
	if found, _ := f.svc.ListEntries(f.ctx, EntryFilter{CreatedTo: yesterday}, all); len(found) != 0 {
		t.Errorf("none were made by yesterday, got %d", len(found))
	}

	// Deleting an entry (with a reason) removes its amounts for good: the week can be entered again.
	newest := list[0]
	if _, _, err := f.svc.DeleteEntry(f.ctx, newest.ID, "", all); err == nil {
		t.Error("deleting an entry needs a reason")
	}
	entry, removed, err := f.svc.DeleteEntry(f.ctx, newest.ID, "entered twice", all)
	if err != nil || entry.ID != newest.ID || len(removed) != 1 {
		t.Fatalf("delete: %v %d", err, len(removed))
	}
	var left int64
	f.db.Unscoped().Model(&models.WeeklyPlatformResult{}).Where("branch_id = ? AND week_start = ?", newest.BranchID, newest.WeekStart).Count(&left)
	if left != 0 {
		t.Errorf("its amounts are gone, %d left", left)
	}
	delete(f.entries, f.entryKey(f.week.AddDate(0, 0, 7), 0))
	if _, _, err := f.save(f.week.AddDate(0, 0, 7), 0, canAll, "", row(f.p1, "7", "0")); err != nil {
		t.Errorf("the week can be entered again: %v", err)
	}
	// New entry is always a new entry, even for a week that has one; the saves after change only it.
	second, _, err := f.svc.SaveWeek(f.ctx, WeekEntryInput{WeekStart: dayString(f.week), BranchID: f.hq[1].ID.String(), Title: "Week 1 again", Rows: []WeekEntryRow{row(f.p1, "300", "0")}}, all, canAll, f.actor)
	if err != nil || second.Entry == nil || second.Entry.ID == list[1].ID || !sheetRow(t, second, f.p1).WinLose.Equal(decimal.NewFromInt(300)) {
		t.Fatalf("a second entry for the week: %v", err)
	}
	if _, _, err := f.svc.SaveWeek(f.ctx, WeekEntryInput{EntryID: second.Entry.ID.String(), Title: "Week 1 again", Rows: []WeekEntryRow{row(f.p2, "5", "0")}}, all, canAll, f.actor); err != nil {
		t.Fatal(err)
	}
	first, err := f.svc.EntrySheet(f.ctx, list[1].ID.String(), "", "", all, f.actor)
	if err != nil || !sheetRow(t, first, f.p1).WinLose.Equal(decimal.NewFromInt(1000)) || sheetRow(t, first, f.p2).Saved {
		t.Errorf("the first entry is left as it was: %v", err)
	}
	if found, _ := f.svc.ListEntries(f.ctx, EntryFilter{Search: "Week 1"}, all); len(found) != 2 {
		t.Errorf("both entries of the week are listed, got %d", len(found))
	}
	// A finalized week's entry can't be deleted.
	if _, err := f.svc.FinalizeWeek(f.ctx, FinalizeInput{BranchID: f.hq[1].ID.String(), WeekStart: dayString(f.week)}, all, f.actor); err != nil {
		t.Fatal(err)
	}
	if _, _, err := f.svc.DeleteEntry(f.ctx, list[1].ID, "wrong", all); err == nil {
		t.Error("a finalized week's entry can't be deleted")
	}
}

func TestAssignmentGroups(t *testing.T) {
	f := newWLFixture(t)
	f.db.Exec("DELETE FROM partners WHERE name LIKE 'TWL%'")
	f.db.Model(&models.User{}).Where("id = ?", f.actor).Update("default_branch_id", f.hq[0].ID)
	defer f.db.Model(&models.User{}).Where("id = ?", f.actor).Update("default_branch_id", nil)
	agent := func(p models.Platform, rate string) GroupAgentInput {
		return GroupAgentInput{PlatformID: p.ID.String(), CommissionRate: rate}
	}
	rateOf := func(g *GroupRow, p models.Platform) string {
		for _, a := range g.Agents {
			if a.PlatformID == p.ID {
				return a.CommissionRate.String()
			}
		}
		return "-"
	}
	start := dayString(f.week.AddDate(0, 0, -21))

	// A titled group in the person's own branch, each agent at its own rate.
	if _, err := f.svc.CreateGroup(f.ctx, GroupInput{Title: " ", Agents: []GroupAgentInput{agent(f.p1, "10")}}, all, f.actor); err == nil {
		t.Error("a group needs a title")
	}
	if _, err := f.svc.CreateGroup(f.ctx, GroupInput{Title: "No rate", Agents: []GroupAgentInput{agent(f.p1, "")}}, all, f.actor); err == nil {
		t.Error("each agent needs a rate")
	}
	g, err := f.svc.CreateGroup(f.ctx, GroupInput{Title: "  V-series  ", Agents: []GroupAgentInput{agent(f.p1, "20"), agent(f.p2, "10")}, StartsOn: start}, all, f.actor)
	if err != nil || g.Title != "V-series" || g.BranchID != f.hq[0].ID || len(g.Agents) != 2 || len(g.Partners) != 0 || rateOf(g, f.p1) != "20" || rateOf(g, f.p2) != "10" {
		t.Fatalf("group: %v %+v", err, g)
	}
	// An agent can be in several groups: on Data Entry it is one row, counting in each.
	second, err := f.svc.CreateGroup(f.ctx, GroupInput{Title: "Other", Agents: []GroupAgentInput{agent(f.p1, "5")}, StartsOn: start}, all, f.actor)
	if err != nil {
		t.Fatalf("an agent can be in a second group: %v", err)
	}
	var titles []string
	for _, g := range sheetRow(t, f.sheet(t, f.week, 0), f.p1).Groups {
		titles = append(titles, g.GroupTitle)
	}
	if len(titles) != 2 || titles[0] != "Other" || titles[1] != "V-series" {
		t.Errorf("one row in both groups, by title: %v", titles)
	}
	if _, err := f.svc.DeleteGroup(f.ctx, second.ID, all); err != nil {
		t.Fatal(err)
	}
	// The Total is at the agent's rate.
	sheet, _, err := f.save(f.week, 0, canAll, "", row(f.p1, "1000", "0"))
	if err != nil {
		t.Fatal(err)
	}
	if r := sheetRow(t, sheet, f.p1); !r.JamePS.Equal(decimal.NewFromInt(200)) {
		t.Errorf("1000 at 20%% is 200, got %s", r.JamePS)
	}
	// Reports: totals by group, and a filter by group.
	f.assign(t, []models.Platform{f.p1}, 1) // TWL2: an assignment in no group
	if _, _, err := f.save(f.week, 1, canAll, "", row(f.p1, "50", "0")); err != nil {
		t.Fatal(err)
	}
	week := ReportFilter{DateFrom: dayString(f.week), DateTo: dayString(f.week.AddDate(0, 0, 6))}
	r, err := f.svc.Report(f.ctx, week, all, true, f.actor)
	if err != nil || len(r.Groups) != 2 || r.Groups[0].Title != "V-series" || !r.Groups[0].Totals[0].JamePS.Equal(decimal.NewFromInt(200)) || r.Groups[1].GroupID != nil {
		t.Errorf("by group (no group last): %v %+v", err, r.Groups)
	}
	week.GroupID = &g.ID
	if r, err := f.svc.Report(f.ctx, week, all, true, f.actor); err != nil || len(r.Lines) != 1 || !r.Lines[0].WinLose.Equal(decimal.NewFromInt(1000)) {
		t.Errorf("only the group's agents: %v %+v", err, r)
	}

	// Off: its agents leave Data Entry (the saved week stays) and the Agents page's count.
	if _, err := f.svc.SetGroupActive(f.ctx, g.ID, false, all); err != nil {
		t.Fatal(err)
	}
	if r := sheetRow(t, f.sheet(t, f.week, 0), f.p1); !r.Saved || len(r.Groups) != 1 {
		t.Errorf("off: the saved week stays as it was: %+v", r)
	}
	if r := sheetRow(t, f.sheet(t, f.week.AddDate(0, 0, 7), 0), f.p2); len(r.Groups) != 0 {
		t.Errorf("off: its agents count in no group on a new week, got %d", len(r.Groups))
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
	if r := sheetRow(t, f.sheet(t, f.week.AddDate(0, 0, 7), 0), f.p2); len(r.Groups) != 1 {
		t.Errorf("on again: its agents count in it, got %d", len(r.Groups))
	}

	// Update: rename, take out the agent without weeks (removed), a new rate; partners from before go.
	partner, err := f.svc.SavePartner(f.ctx, nil, PartnerInput{Name: "TWL Old Partner", DefaultRate: "5"})
	if err != nil {
		t.Fatal(err)
	}
	f.db.Create(&models.AssignmentPartner{ID: uuid.New(), AssignmentID: g.Agents[0].AssignmentID, PartnerID: partner.ID, CommissionRate: decimal.NewFromInt(5)})
	_, after, err := f.svc.UpdateGroup(f.ctx, g.ID, GroupUpdateInput{Title: "V1 only", Agents: []GroupAgentInput{agent(f.p1, "30")}}, all, f.actor)
	if err != nil || after.Title != "V1 only" || len(after.Agents) != 1 || rateOf(after, f.p1) != "30" || len(after.Partners) != 0 {
		t.Fatalf("update: %v %+v", err, after)
	}
	var gone int64
	f.db.Unscoped().Model(&models.HQPlatformAssignment{}).Where("platform_id = ? AND group_id = ?", f.p2.ID, g.ID).Count(&gone)
	if gone != 0 {
		t.Error("an agent taken out before any week is entered is removed")
	}

	// Delete: only while no week is entered.
	other, err := f.svc.CreateGroup(f.ctx, GroupInput{Title: "P2", Agents: []GroupAgentInput{agent(f.p2, "15")}}, all, f.actor)
	if err != nil {
		t.Fatal(err)
	}
	if found, err := f.svc.GroupByID(f.ctx, other.ID, f.scope(1)); err == nil || found != nil {
		t.Error("another branch's group is not found")
	}
	if _, err := f.svc.DeleteGroup(f.ctx, g.ID, all); err == nil {
		t.Error("a group with weeks entered can't be deleted")
	}
	if _, err := f.svc.DeleteGroup(f.ctx, other.ID, all); err != nil {
		t.Fatalf("a group without weeks can be deleted: %v", err)
	}
	// Taking out an agent with weeks ends it (history); one put in starts today.
	_, after, err = f.svc.UpdateGroup(f.ctx, g.ID, GroupUpdateInput{Title: "V1 only", Agents: []GroupAgentInput{agent(f.p2, "12.5")}}, all, f.actor)
	if err != nil || len(after.Agents) != 1 || after.Agents[0].PlatformID != f.p2.ID || after.Agents[0].StartsOn != dayString(f.svc.today()) || rateOf(after, f.p2) != "12.5" || after.Weeks != 1 {
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
	if found, _ := f.svc.ListGroups(f.ctx, f.hq[0].ID, GroupFilter{Search: "v1 ON"}, all); len(found) != 1 {
		t.Errorf("search by title (any case): %d", len(found))
	}
	if found, _ := f.svc.ListGroups(f.ctx, f.hq[0].ID, GroupFilter{Search: "nothing like it"}, all); len(found) != 0 {
		t.Errorf("a search with no match: %d", len(found))
	}
}

func TestAgentInSeveralGroups(t *testing.T) {
	f := newWLFixture(t)
	f.db.Model(&models.User{}).Where("id = ?", f.actor).Update("default_branch_id", f.hq[0].ID)
	defer f.db.Model(&models.User{}).Where("id = ?", f.actor).Update("default_branch_id", nil)
	start := dayString(f.week.AddDate(0, 0, -21))
	n := func(v int64) decimal.Decimal { return decimal.NewFromInt(v) }
	// The agent's assignment from before groups, ended mid-week, gives way to its groups.
	old := f.assign(t, []models.Platform{f.p1}, 0)[0]
	if _, err := f.svc.EndAssignment(f.ctx, old.ID, AssignmentEndInput{EndsOn: dayString(f.week.AddDate(0, 0, 2)), Reason: "into groups"}, all, f.actor); err != nil {
		t.Fatal(err)
	}
	group := func(title, rate string) *GroupRow {
		t.Helper()
		g, err := f.svc.CreateGroup(f.ctx, GroupInput{Title: title, Agents: []GroupAgentInput{{PlatformID: f.p1.ID.String(), CommissionRate: rate}}, StartsOn: start}, all, f.actor)
		if err != nil {
			t.Fatalf("group %s: %v", title, err)
		}
		return g
	}
	a, b := group("A", "10"), group("B", "20")

	// One row for the agent: its Total in each group at the group's rate, and their sum. An agent
	// in no group (p2 here) can be entered too, with no Total.
	sheet, _, err := f.save(f.week, 0, canAll, "", row(f.p1, "1000", "0"), row(f.p2, "500", "0"))
	if err != nil {
		t.Fatal(err)
	}
	r := sheetRow(t, sheet, f.p1)
	if len(r.Groups) != 2 || r.Groups[0].GroupTitle != "A" || !r.Groups[0].JamePS.Equal(n(100)) || !r.Groups[1].JamePS.Equal(n(200)) ||
		!r.JamePS.Equal(n(300)) || !r.CommissionRate.Equal(n(30)) {
		t.Fatalf("100 in A and 200 in B, 300 in all: %+v", r)
	}
	if r := sheetRow(t, sheet, f.p2); !r.Saved || len(r.Groups) != 0 || !r.JamePS.IsZero() {
		t.Errorf("an agent in no group: %+v", r)
	}
	// A correction works out each group again at the rate saved.
	if _, _, err := f.svc.UpdateGroup(f.ctx, a.ID, GroupUpdateInput{Title: "A", Agents: []GroupAgentInput{{PlatformID: f.p1.ID.String(), CommissionRate: "50"}}}, all, f.actor); err != nil {
		t.Fatal(err)
	}
	sheet, _, err = f.save(f.week, 0, canAll, "fix", row(f.p1, "500", "0"), row(f.p2, "500", "0"))
	if r := sheetRow(t, sheet, f.p1); err != nil || !r.Groups[0].JamePS.Equal(n(50)) || !r.Groups[1].JamePS.Equal(n(100)) || !r.JamePS.Equal(n(150)) {
		t.Errorf("corrected at 10%% and 20%%: %v %+v", err, r)
	}

	// A second entry for the same week adds up with the first in the reports.
	if _, _, err := f.svc.SaveWeek(f.ctx, WeekEntryInput{WeekStart: dayString(f.week), Title: "Late", Rows: []WeekEntryRow{row(f.p1, "100", "0")}}, all, canAll, f.actor); err != nil {
		t.Fatal(err)
	}
	week := ReportFilter{DateFrom: dayString(f.week), DateTo: dayString(f.week.AddDate(0, 0, 6))}
	rep, err := f.svc.Report(f.ctx, week, all, true, f.actor)
	if err != nil {
		t.Fatal(err)
	}
	totals := map[string]string{}
	for _, g := range rep.Groups {
		totals[g.Title] = g.Totals[0].JamePS.String() + " " + g.Totals[0].WinLose.String()
	}
	// A: 50 (500 at 10%) + 50 (100 at 50%); B: 100 + 20; no group: p2's 500 with no Total.
	if totals["A"] != "100 600" || totals["B"] != "120 600" || totals[""] != "0 500" {
		t.Errorf("by group, the agent in both: %v", totals)
	}
	for _, l := range rep.Lines {
		if l.PlatformID == f.p1.ID && (!l.WinLose.Equal(n(600)) || !l.JamePS.Equal(n(220)) || l.Entries != 2) {
			t.Errorf("by agent, both entries and both groups: %+v", l)
		}
	}
	week.GroupID = &b.ID
	if one, err := f.svc.Report(f.ctx, week, all, true, f.actor); err != nil || len(one.Lines) != 1 || !one.Lines[0].JamePS.Equal(n(120)) {
		t.Errorf("one group's report has its part of the Total: %v %+v", err, one)
	}
	// One agent: its part in each group, with its rate there (A's went from 10% to 50% in the week).
	week.GroupID, week.PlatformID = nil, &f.p1.ID
	mine, err := f.svc.Report(f.ctx, week, all, true, f.actor)
	parts := map[string]string{}
	for _, g := range mine.Groups {
		parts[g.Title] = g.RateMin.String() + "-" + g.RateMax.String() + " " + g.Totals[0].JamePS.String() + " " + g.Formula
	}
	if err != nil || len(mine.Lines) != 1 || len(mine.Groups) != 2 || parts["A"] != "10-50 100 v1" || parts["B"] != "20-20 120 v1" {
		t.Errorf("one agent's groups: %v %v", err, parts)
	}
	if g, _ := f.svc.GroupByID(f.ctx, a.ID, all); g.Weeks != 2 {
		t.Errorf("A has the agent's two results, got %d", g.Weeks)
	}
}

func TestFormulaChange(t *testing.T) {
	f := newWLFixture(t) // v1
	n := func(v int64) decimal.Decimal { return decimal.NewFromInt(v) }
	a := f.assign(t, []models.Platform{f.p1}, 0)[0] // 35%
	// Two weeks saved with v1, (1000 + 200) × 35% = 420; the earlier one is finalized.
	open, finalized := f.week, f.week.AddDate(0, 0, -7)
	for _, w := range []time.Time{open, finalized} {
		if _, _, err := f.save(w, 0, canAll, "", row(f.p1, "1000", "200")); err != nil {
			t.Fatal(err)
		}
	}
	st, err := f.svc.FinalizeWeek(f.ctx, FinalizeInput{BranchID: f.hq[0].ID.String(), WeekStart: dayString(finalized)}, all, f.actor)
	if err != nil {
		t.Fatal(err)
	}

	// v2: the commission is taken away. Weeks not finalized follow it.
	if err := f.db.Save(&models.AppSetting{Key: FormulaSettingKey, Value: "v2"}).Error; err != nil {
		t.Fatal(err)
	}
	changed, err := f.svc.RecalculateOpenWeeks(f.ctx)
	if err != nil || changed != 1 {
		t.Fatalf("only the open week is worked out again: %v %d", err, changed)
	}
	if r := sheetRow(t, f.sheet(t, open, 0), f.p1); !r.JamePS.Equal(n(280)) || r.FormulaVersion != "v2" || !r.Groups[0].JamePS.Equal(n(280)) {
		t.Errorf("(1000 − 200) × 35%% = 280: %s %s", r.JamePS, r.FormulaVersion)
	}
	if r := sheetRow(t, f.sheet(t, finalized, 0), f.p1); !r.JamePS.Equal(n(420)) || r.FormulaVersion != "v1" {
		t.Errorf("the finalized week stays as settled: %s %s", r.JamePS, r.FormulaVersion)
	}
	if again, _ := f.svc.RecalculateOpenWeeks(f.ctx); again != 0 {
		t.Errorf("nothing left to work out, got %d", again)
	}
	both := ReportFilter{DateFrom: dayString(finalized), DateTo: dayString(open.AddDate(0, 0, 6))}
	if r, err := f.svc.Report(f.ctx, both, all, true, f.actor); err != nil || len(r.Lines) != 1 || r.Lines[0].Formula != "" {
		t.Errorf("a line over weeks with two formulas names none: %v %+v", err, r)
	}
	// Reopened, the week follows the current formula too.
	if _, err := f.svc.ReopenWeek(f.ctx, st.ID, "late result", all, f.actor); err != nil {
		t.Fatal(err)
	}
	if r := sheetRow(t, f.sheet(t, finalized, 0), f.p1); !r.JamePS.Equal(n(280)) || r.FormulaVersion != "v2" {
		t.Errorf("reopened: %s %s", r.JamePS, r.FormulaVersion)
	}
	if r, _ := f.svc.Report(f.ctx, both, all, true, f.actor); r.Lines[0].Formula != "v2" {
		t.Errorf("the line names its formula, got %q", r.Lines[0].Formula)
	}
	// A new week: (1,000 − 100) × 10% = 90.
	if _, _, err := f.svc.UpdateAssignment(f.ctx, a.ID, AssignmentUpdateInput{CommissionRate: "10", Currency: "USD"}, all); err != nil {
		t.Fatal(err)
	}
	next, _, err := f.save(open.AddDate(0, 0, 7), 0, canAll, "", row(f.p1, "1000", "100"))
	if err != nil {
		t.Fatal(err)
	}
	if r := sheetRow(t, next, f.p1); !r.JamePS.Equal(n(90)) {
		t.Errorf("(1,000 − 100) × 10%% = 90, got %s", r.JamePS)
	}

	// The business's rule, v4: the commission is added and the rate is the agent's share, the Total
	// at what is left.
	f.db.Where("`key` = ?", FormulaSettingKey).Delete(&models.AppSetting{})
	if changed, err := f.svc.RecalculateOpenWeeks(f.ctx); err != nil || changed != 3 {
		t.Fatalf("the three open weeks are worked out again: %v %d", err, changed)
	}
	if r := sheetRow(t, f.sheet(t, open, 0), f.p1); !r.JamePS.Equal(n(780)) || r.FormulaVersion != "v4" || !r.Groups[0].JamePS.Equal(n(780)) {
		t.Errorf("(1000 + 200) × (100%% − 35%%) = 780: %s %s", r.JamePS, r.FormulaVersion)
	}
	if r := sheetRow(t, f.sheet(t, open.AddDate(0, 0, 7), 0), f.p1); !r.JamePS.Equal(n(990)) || !r.CommissionRate.Equal(n(10)) {
		t.Errorf("(1,000 + 100) × (100%% − 10%%) = 990, got %s", r.JamePS)
	}
	// The agent at 60%: (1,000 + 100) × (100% − 60%) = 440.
	if _, _, err := f.svc.UpdateAssignment(f.ctx, a.ID, AssignmentUpdateInput{CommissionRate: "60", Currency: "USD"}, all); err != nil {
		t.Fatal(err)
	}
	later, _, err := f.save(open.AddDate(0, 0, 14), 0, canAll, "", row(f.p1, "1000", "100"))
	if err != nil {
		t.Fatal(err)
	}
	if r := sheetRow(t, later, f.p1); !r.JamePS.Equal(n(440)) || r.FormulaVersion != "v4" {
		t.Errorf("(1,000 + 100) × (100%% − 60%%) = 440, got %s %s", r.JamePS, r.FormulaVersion)
	}
}

func TestRestWithPartners(t *testing.T) {
	// v3, partners at 40% and 10% of 1,000: shares 400 and 100, the Total what is left (500).
	shares := []PartnerShare{{CommissionRate: decimal.NewFromInt(40)}, {CommissionRate: decimal.NewFromInt(10)}}
	r, err := workOut("v3", winlose.PO, decimal.NewFromInt(1000), decimal.Zero, decimal.Zero, shares)
	if err != nil || !r.JamePS.Equal(decimal.NewFromInt(500)) || !shares[0].JamePS.Equal(decimal.NewFromInt(400)) || !shares[1].JamePS.Equal(decimal.NewFromInt(100)) {
		t.Errorf("total %s, shares %v %v: %v", r.JamePS, shares[0].JamePS, shares[1].JamePS, err)
	}
}

// An agent's groups set from its own form (Agents → Add/Edit): put in, rate and type changed and
// taken out as a group's page would, all or nothing with the agent itself.
func TestAgentGroupsFromItsForm(t *testing.T) {
	f := newWLFixture(t)
	f.db.Model(&models.User{}).Where("id = ?", f.actor).Update("default_branch_id", f.hq[0].ID)
	defer f.db.Model(&models.User{}).Where("id = ?", f.actor).Update("default_branch_id", nil)
	start := dayString(f.week.AddDate(0, 0, -21))
	group := func(title string, hq int) *GroupRow {
		t.Helper()
		g, err := f.svc.CreateGroup(f.ctx, GroupInput{Title: title, Agents: []GroupAgentInput{{PlatformID: f.p2.ID.String(), CommissionRate: "10"}}, StartsOn: start, BranchID: f.hq[hq].ID.String()}, all, f.actor)
		if err != nil {
			t.Fatalf("group %s: %v", title, err)
		}
		return g
	}
	a, b, c := group("A", 0), group("B", 0), group("C", 1)
	in := func(id *uuid.UUID, code string, groups ...PlatformGroupInput) PlatformInput {
		list := append([]PlatformGroupInput{}, groups...)
		return PlatformInput{Code: code, DefaultCurrency: "USD", BranchID: f.hq[0].ID.String(), Groups: &list}
	}
	put := func(g *GroupRow, rate, kind string) PlatformGroupInput {
		return PlatformGroupInput{GroupID: g.ID.String(), CommissionRate: rate, CalcType: kind}
	}
	mine := func(g *GroupRow, p uuid.UUID) *GroupAgent {
		t.Helper()
		row, err := f.svc.GroupByID(f.ctx, g.ID, all)
		if err != nil {
			t.Fatal(err)
		}
		for i := range row.Agents {
			if row.Agents[i].PlatformID == p {
				return &row.Agents[i]
			}
		}
		return nil
	}
	scope0 := f.scope(0)

	// A bad rate saves nothing: not even the agent.
	if _, _, err := f.svc.SavePlatformWithGroups(f.ctx, nil, in(nil, "TWL_FORM", put(a, "15", "GI"), put(b, "", "")), scope0, f.actor); err == nil || !strings.Contains(err.Error(), "rate") {
		t.Fatalf("an empty rate is refused: %v", err)
	}
	var made int64
	f.db.Model(&models.Platform{}).Where("code = ?", "TWL_FORM").Count(&made)
	if made != 0 {
		t.Fatalf("the agent was saved without its groups")
	}
	if _, _, err := f.svc.SavePlatformWithGroups(f.ctx, nil, in(nil, "TWL_FORM", put(a, "15", ""), put(a, "20", "")), scope0, f.actor); err == nil || !strings.Contains(err.Error(), "twice") {
		t.Fatalf("a group listed twice is refused: %v", err)
	}
	// Another branch's group is refused for a branch's agent.
	if _, _, err := f.svc.SavePlatformWithGroups(f.ctx, nil, in(nil, "TWL_FORM", put(c, "15", "")), all, f.actor); err == nil || !strings.Contains(err.Error(), "another branch") {
		t.Fatalf("another branch's group is refused: %v", err)
	}
	// Put in A (GI 15%) and B (PO 0%), from today.
	p, changes, err := f.svc.SavePlatformWithGroups(f.ctx, nil, in(nil, "TWL_FORM", put(a, "15", "gi"), put(b, "0", "")), scope0, f.actor)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(changes.Added, ",") != "A,B" || len(changes.Removed)+len(changes.Changed) != 0 {
		t.Errorf("added to A and B: %+v", changes)
	}
	if got := mine(a, p.ID); got == nil || got.CalcType != "GI" || !got.CommissionRate.Equal(decimal.NewFromInt(15)) || got.StartsOn != dayString(f.svc.today()) || got.Currency != "USD" {
		t.Errorf("in A at GI 15%% from today: %+v", got)
	}
	if got := mine(b, p.ID); got == nil || got.CalcType != "PO" || !got.CommissionRate.IsZero() {
		t.Errorf("in B at PO 0%%: %+v", got)
	}
	// Not sending groups leaves them as they are.
	same := PlatformInput{Code: "TWL_FORM", Name: "Renamed", DefaultCurrency: "USD"}
	if _, changes, err := f.svc.SavePlatformWithGroups(f.ctx, &p.ID, same, scope0, f.actor); err != nil || changes.Any() || mine(a, p.ID) == nil || mine(b, p.ID) == nil {
		t.Fatalf("groups not sent stay: %v %+v", err, changes)
	}
	// A's rate changed, B taken out: no week entered there, so it's removed for good.
	if _, changes, err = f.svc.SavePlatformWithGroups(f.ctx, &p.ID, in(&p.ID, "TWL_FORM", put(a, "20", "PO")), scope0, f.actor); err != nil {
		t.Fatal(err)
	}
	if strings.Join(changes.Changed, ",") != "A" || strings.Join(changes.Removed, ",") != "B" || len(changes.Added) != 0 {
		t.Errorf("A changed, B taken out: %+v", changes)
	}
	if got := mine(a, p.ID); got == nil || got.CalcType != "PO" || !got.CommissionRate.Equal(decimal.NewFromInt(20)) {
		t.Errorf("A at PO 20%%: %+v", got)
	}
	var inB int64
	f.db.Unscoped().Model(&models.HQPlatformAssignment{}).Where("platform_id = ? AND group_id = ?", p.ID, b.ID).Count(&inB)
	if inB != 0 {
		t.Errorf("B's unused assignment is removed, %d left", inB)
	}
	// With a week entered, taking it out of A ends the assignment (the week stays).
	if _, _, err := f.save(weekStart(f.svc.today()), 0, canAll, "", row(*p, "1000", "0")); err != nil {
		t.Fatal(err)
	}
	if _, changes, err = f.svc.SavePlatformWithGroups(f.ctx, &p.ID, in(&p.ID, "TWL_FORM"), scope0, f.actor); err != nil || strings.Join(changes.Removed, ",") != "A" {
		t.Fatalf("taken out of A: %v %+v", err, changes)
	}
	var ended models.HQPlatformAssignment
	if err := f.db.Where("platform_id = ? AND group_id = ?", p.ID, a.ID).First(&ended).Error; err != nil || ended.IsActive || ended.EndsOn == nil {
		t.Errorf("A's assignment is ended, not removed: %v %+v", err, ended)
	}
	// A turned-off agent can't be put in a group.
	off := false
	if _, _, err := f.svc.SavePlatformWithGroups(f.ctx, &p.ID, PlatformInput{Code: "TWL_FORM", DefaultCurrency: "USD", IsActive: &off, Reason: "test"}, scope0, f.actor); err != nil {
		t.Fatal(err)
	}
	if _, _, err := f.svc.SavePlatformWithGroups(f.ctx, &p.ID, in(&p.ID, "TWL_FORM", put(b, "5", "")), scope0, f.actor); err == nil || !strings.Contains(err.Error(), "turn the agent on") {
		t.Errorf("a turned-off agent is refused: %v", err)
	}
}

// A week saved before its agent was put in a group counts in the group once it is, from the
// group's first day on (as saving it then would): older weeks and finalized weeks stay as they were.
func TestWeeksSavedBeforeTheGroup(t *testing.T) {
	f := newWLFixture(t)
	f.db.Model(&models.User{}).Where("id = ?", f.actor).Update("default_branch_id", f.hq[0].ID)
	defer f.db.Model(&models.User{}).Where("id = ?", f.actor).Update("default_branch_id", nil)
	n := func(v int64) decimal.Decimal { return decimal.NewFromInt(v) }
	older, settled, now := f.week, f.week.AddDate(0, 0, 7), weekStart(f.svc.today())
	// p1 is in no group yet: its weeks are saved with no Total.
	for _, w := range []time.Time{older, settled, now} {
		if _, _, err := f.save(w, 0, canAll, "", row(f.p1, "1000", "0")); err != nil {
			t.Fatal(err)
		}
	}
	if r := sheetRow(t, f.sheet(t, now, 0), f.p1); len(r.Groups) != 0 || !r.JamePS.IsZero() {
		t.Fatalf("no group yet, no Total: %+v", r)
	}
	if _, err := f.svc.FinalizeWeek(f.ctx, FinalizeInput{BranchID: f.hq[0].ID.String(), WeekStart: dayString(settled)}, all, f.actor); err != nil {
		t.Fatal(err)
	}

	// The group starts on the settled week's Monday: this week counts in it now, the settled week
	// (finalized) and the older one (before its first day) don't.
	g, err := f.svc.CreateGroup(f.ctx, GroupInput{Title: "Later", Agents: []GroupAgentInput{{PlatformID: f.p1.ID.String(), CommissionRate: "10"}}, StartsOn: dayString(settled)}, all, f.actor)
	if err != nil {
		t.Fatal(err)
	}
	if g.CountedWeeks != 1 {
		t.Errorf("one week counted, got %d", g.CountedWeeks)
	}
	r := sheetRow(t, f.sheet(t, now, 0), f.p1)
	if len(r.Groups) != 1 || r.Groups[0].GroupTitle != "Later" || r.Groups[0].JamePS == nil || !r.Groups[0].JamePS.Equal(n(100)) || !r.JamePS.Equal(n(100)) || !r.CommissionRate.Equal(n(10)) {
		t.Fatalf("this week: 100 in Later: %+v", r)
	}
	for _, w := range []time.Time{older, settled} {
		if r := sheetRow(t, f.sheet(t, w, 0), f.p1); len(r.Groups) != 0 || !r.JamePS.IsZero() {
			t.Errorf("week of %s stays with no group: %+v", dayString(w), r)
		}
	}
	// The reports: the group has this week; the other two are still "no group".
	span := ReportFilter{DateFrom: dayString(older), DateTo: dayString(f.svc.today())}
	inGroup := span
	inGroup.GroupID = &g.ID
	rep, err := f.svc.Report(f.ctx, inGroup, all, true, f.actor)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Totals) != 1 || rep.Totals[0].Entries != 1 || !rep.Totals[0].JamePS.Equal(n(100)) {
		t.Errorf("the group's report has this week, 100: %+v", rep.Totals)
	}
	rep, err = f.svc.Report(f.ctx, span, all, true, f.actor)
	if err != nil {
		t.Fatal(err)
	}
	groups := map[string]Sums{}
	for _, row := range rep.Groups {
		groups[row.Title] = row.Totals[0]
	}
	if groups["Later"].Entries != 1 || !groups["Later"].JamePS.Equal(n(100)) || groups[""].Entries != 2 || !groups[""].JamePS.IsZero() {
		t.Errorf("by group: Later 1 week 100, no group 2 weeks: %+v", groups)
	}
	var st models.WeeklySettlement
	f.db.Where("branch_id = ? AND week_start = ?", f.hq[0].ID, dayString(settled)).First(&st)
	var settledTotal models.WeeklySettlementTotal
	f.db.Where("settlement_id = ?", st.ID).First(&settledTotal)
	if !settledTotal.JamePS.IsZero() {
		t.Errorf("the finalized week's totals stay: %+v", settledTotal)
	}

	// Saving the group again counts nothing twice; an agent put in it later brings its week too.
	if _, _, err := f.save(now, 0, canAll, "", row(f.p2, "1000", "0")); err != nil {
		t.Fatal(err)
	}
	_, after, err := f.svc.UpdateGroup(f.ctx, g.ID, GroupUpdateInput{Title: "Later", Agents: []GroupAgentInput{{PlatformID: f.p1.ID.String(), CommissionRate: "10"}, {PlatformID: f.p2.ID.String(), CommissionRate: "20"}}}, all, f.actor)
	if err != nil {
		t.Fatal(err)
	}
	if after.CountedWeeks != 1 {
		t.Errorf("p2's week counted, got %d", after.CountedWeeks)
	}
	sheet := f.sheet(t, now, 0)
	if r := sheetRow(t, sheet, f.p1); len(r.Groups) != 1 || !r.JamePS.Equal(n(100)) {
		t.Errorf("p1 not counted twice: %+v", r)
	}
	if r := sheetRow(t, sheet, f.p2); len(r.Groups) != 1 || !r.JamePS.Equal(n(200)) {
		t.Errorf("p2: 200 in Later: %+v", r)
	}

	// The same from an agent's own form.
	late, _, err := f.svc.SavePlatformWithGroups(f.ctx, nil, PlatformInput{Code: "TWL_LATE", DefaultCurrency: "USD", BranchID: f.hq[0].ID.String()}, all, f.actor)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := f.save(now, 0, canAll, "", row(*late, "1000", "0")); err != nil {
		t.Fatal(err)
	}
	groupsIn := []PlatformGroupInput{{GroupID: g.ID.String(), CommissionRate: "30"}}
	_, changes, err := f.svc.SavePlatformWithGroups(f.ctx, &late.ID, PlatformInput{Code: "TWL_LATE", DefaultCurrency: "USD", Groups: &groupsIn}, all, f.actor)
	if err != nil || changes.Weeks != 1 {
		t.Fatalf("its week counted from the agent form: %v %+v", err, changes)
	}
	if r := sheetRow(t, f.sheet(t, now, 0), *late); len(r.Groups) != 1 || !r.JamePS.Equal(n(300)) {
		t.Errorf("TWL_LATE: 300 in Later: %+v", r)
	}
}
