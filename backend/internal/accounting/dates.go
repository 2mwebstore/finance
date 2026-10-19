package accounting

import (
	"fmt"
	"time"
)

// DateRange is a half-open interval [Start, End).
type DateRange struct {
	Start time.Time
	End   time.Time
	Label string
}

func LoadLocation(tz string) *time.Location {
	loc, err := time.LoadLocation(tz)
	if err != nil {
		return time.FixedZone("Asia/Phnom_Penh", 7*3600)
	}
	return loc
}

// StartOfDay returns 00:00:00 in the given location.
func StartOfDay(t time.Time, loc *time.Location) time.Time {
	t = t.In(loc)
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, loc)
}

// ParsePeriod resolves preset periods or custom dates.
// period: today | this_week | this_month | custom
// Week starts Monday (ISO).
func ParsePeriod(period, dateFrom, dateTo, tz string) (DateRange, error) {
	loc := LoadLocation(tz)
	now := time.Now().In(loc)

	switch period {
	case "", "today":
		start := StartOfDay(now, loc)
		return DateRange{Start: start, End: start.Add(24 * time.Hour), Label: "today"}, nil
	case "this_week":
		start := StartOfDay(now, loc)
		// Weekday: Sunday=0 ... Saturday=6; Monday-based: shift so Monday=0
		wd := int(start.Weekday())
		if wd == 0 {
			wd = 7
		}
		start = start.AddDate(0, 0, -(wd - 1))
		return DateRange{Start: start, End: start.AddDate(0, 0, 7), Label: "this_week"}, nil
	case "this_month":
		start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, loc)
		end := start.AddDate(0, 1, 0)
		return DateRange{Start: start, End: end, Label: "this_month"}, nil
	case "custom":
		if dateFrom == "" || dateTo == "" {
			return DateRange{}, fmt.Errorf("date_from and date_to required for custom period")
		}
		from, err := time.ParseInLocation("2006-01-02", dateFrom, loc)
		if err != nil {
			return DateRange{}, fmt.Errorf("invalid date_from")
		}
		to, err := time.ParseInLocation("2006-01-02", dateTo, loc)
		if err != nil {
			return DateRange{}, fmt.Errorf("invalid date_to")
		}
		// inclusive end day → half-open end = next day
		end := to.Add(24 * time.Hour)
		if !end.After(from) {
			return DateRange{}, fmt.Errorf("date_to must be on or after date_from")
		}
		// limit to 366 days
		if end.Sub(from) > 366*24*time.Hour {
			return DateRange{}, fmt.Errorf("date range too large (max 366 days)")
		}
		return DateRange{Start: from, End: end, Label: "custom"}, nil
	default:
		return DateRange{}, fmt.Errorf("invalid period: use today, this_week, this_month, or custom")
	}
}
