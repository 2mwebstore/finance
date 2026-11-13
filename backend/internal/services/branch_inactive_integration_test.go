package services

import (
	"testing"
	"time"

	"github.com/cambodia-finance/income-expense/internal/models"
	"github.com/google/uuid"
)

// An inactive (deleted) branch: its people with no other active branch are signed out (and
// access.BranchesInactive keeps them from signing in again: auth.Login and Refresh); someone who
// also works in an active branch stays signed in; its data stays in the reports. Needs
// TEST_MYSQL_DSN (see platform_integration_test.go).
func TestInactiveBranchSignsOutItsPeople(t *testing.T) {
	f := newWLFixture(t)
	branches := NewBranchService(f.db)
	b := models.Branch{Code: "TWLOFF", Name: "To switch off", IsActive: true}
	if err := f.db.Where(models.Branch{Code: b.Code}).FirstOrCreate(&b).Error; err != nil {
		t.Fatal(err)
	}
	f.db.Model(&b).Update("is_active", true)
	t.Cleanup(func() {
		f.db.Exec("DELETE FROM weekly_platform_results WHERE branch_id = ?", b.ID)
		f.db.Model(&models.Branch{}).Where("id = ?", b.ID).Update("is_active", true)
	})

	// A week entered while it is active.
	p, err := f.svc.SavePlatform(f.ctx, nil, PlatformInput{Code: "TWL_OFF_AG", Name: "x", DefaultRate: "10", DefaultCurrency: "USD", BranchID: b.ID.String()})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.CreateGroup(f.ctx, GroupInput{Title: "TWL off group", Agents: []GroupAgentInput{{PlatformID: p.ID.String(), CommissionRate: "10"}}, StartsOn: dayString(f.week), BranchID: b.ID.String()}, all, f.actor); err != nil {
		t.Fatal(err)
	}
	week := WeekEntryInput{WeekStart: dayString(f.week), BranchID: b.ID.String(), Title: "TWL off week", Rows: []WeekEntryRow{{PlatformID: p.ID.String(), WinLose: "100", Commission: "0"}}}
	if _, _, err := f.svc.SaveWeek(f.ctx, week, all, canAll, f.actor); err != nil {
		t.Fatal(err)
	}

	// Two people, each signed in: one only in it, one also in TWL1.
	person := func(name string, in ...models.Branch) models.Session {
		t.Helper()
		f.db.Unscoped().Where("username = ?", name).Delete(&models.User{})
		u := models.User{Username: name, Email: name + "@example.test", PasswordHash: "x", FullName: name, IsActive: true, Branches: in}
		if err := f.db.Create(&u).Error; err != nil {
			t.Fatal(err)
		}
		s := models.Session{UserID: u.ID, RefreshTokenHash: uuid.NewString(), ExpiresAt: time.Now().Add(time.Hour)}
		if err := f.db.Create(&s).Error; err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { f.db.Unscoped().Delete(&u) })
		return s
	}
	onlySession := person("twl_off_only", b)
	bothSession := person("twl_off_both", b, f.hq[0])

	if _, err := branches.Delete(f.ctx, b.ID); err != nil {
		t.Fatal(err)
	}
	var s1, s2 models.Session
	f.db.First(&s1, "id = ?", onlySession.ID)
	f.db.First(&s2, "id = ?", bothSession.ID)
	if !s1.IsRevoked || s2.IsRevoked {
		t.Errorf("only the person with no active branch left is signed out: only=%v both=%v", s1.IsRevoked, s2.IsRevoked)
	}

	// Its week is still in the reports, on its own and with every branch.
	period := ReportFilter{DateFrom: dayString(f.week), DateTo: dayString(f.week.AddDate(0, 0, 6))}
	one := period
	one.BranchID = &b.ID
	if rep, err := f.svc.Report(f.ctx, one, all, true, f.actor); err != nil || len(rep.Lines) != 1 || rep.Lines[0].PlatformID != p.ID {
		t.Errorf("its report still shows its week: %+v %v", rep, err)
	}
	every, err := f.svc.Report(f.ctx, period, all, true, f.actor)
	found := false
	if err == nil {
		for _, l := range every.Lines {
			found = found || l.PlatformID == p.ID
		}
	}
	if !found {
		t.Errorf("all branches' report still has its week: %v", err)
	}
}
