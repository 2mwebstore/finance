package accounting

import (
	"testing"
	"time"
)

func TestParsePeriodToday(t *testing.T) {
	dr, err := ParsePeriod("today", "", "", "Asia/Phnom_Penh")
	if err != nil {
		t.Fatal(err)
	}
	if dr.End.Sub(dr.Start) != 24*time.Hour {
		t.Fatalf("today range should be 24h, got %v", dr.End.Sub(dr.Start))
	}
}

func TestParsePeriodThisWeekMondayStart(t *testing.T) {
	dr, err := ParsePeriod("this_week", "", "", "Asia/Phnom_Penh")
	if err != nil {
		t.Fatal(err)
	}
	if dr.Start.Weekday() != time.Monday {
		t.Fatalf("week should start Monday, got %v", dr.Start.Weekday())
	}
	if dr.End.Sub(dr.Start) != 7*24*time.Hour {
		t.Fatalf("week should be 7 days")
	}
}

func TestParsePeriodCustom(t *testing.T) {
	dr, err := ParsePeriod("custom", "2026-10-01", "2026-10-17", "Asia/Phnom_Penh")
	if err != nil {
		t.Fatal(err)
	}
	if dr.Start.Format("2006-01-02") != "2026-10-01" {
		t.Fatal(dr.Start)
	}
	// half-open: end is next day after date_to
	if dr.End.Format("2006-01-02") != "2026-10-18" {
		t.Fatalf("end=%s", dr.End)
	}
}

func TestParsePeriodCustomInvalid(t *testing.T) {
	if _, err := ParsePeriod("custom", "2026-10-17", "2026-10-01", "Asia/Phnom_Penh"); err == nil {
		t.Fatal("expected error for inverted range")
	}
}
