package services

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func d(s string) decimal.Decimal { return decimal.RequireFromString(s) }

func TestNextStockAverageCost(t *testing.T) {
	cases := []struct {
		name                  string
		qty, avg, delta, cost string
		atCost                bool
		wantQty, wantAvg      string
		wantOK                bool
	}{
		{"first purchase sets the cost", "0", "0", "10", "2", false, "10", "2", true},
		{"second purchase re-weights", "10", "2", "10", "4", false, "20", "3", true},
		{"a sale leaves at the average", "20", "3", "-5", "0", false, "15", "3", true},
		{"selling everything keeps the last average", "15", "3", "-15", "0", false, "0", "3", true},
		{"more than in stock is refused", "5", "3", "-6", "0", false, "5", "3", false},
		{"voiding a purchase takes its cost out", "20", "3", "-10", "4", true, "10", "2", true},
		{"voiding the only purchase empties it", "10", "2", "-10", "2", true, "0", "2", true},
		{"stock coming back after a void sale", "0", "3", "5", "3", false, "5", "3", true},
		{"rounding to 4 places", "3", "1", "1", "2", false, "4", "1.25", true},
	}
	for _, c := range cases {
		q, a, ok := nextStock(d(c.qty), d(c.avg), d(c.delta), d(c.cost), c.atCost)
		if ok != c.wantOK || !q.Equal(d(c.wantQty)) || !a.Equal(d(c.wantAvg)) {
			t.Errorf("%s: got qty=%s avg=%s ok=%v, want qty=%s avg=%s ok=%v", c.name, q, a, ok, c.wantQty, c.wantAvg, c.wantOK)
		}
	}
}

func TestInventoryInputRules(t *testing.T) {
	for _, ok := range []string{"1", "0.5", "12.3456", "999999999"} {
		if _, err := parseQuantity("q", ok); err != nil {
			t.Errorf("quantity %q should be accepted", ok)
		}
	}
	for _, bad := range []string{"", "0", "-1", "1.23456", "abc", "1000000000"} {
		if _, err := parseQuantity("q", bad); err == nil {
			t.Errorf("quantity %q should be refused", bad)
		}
	}
	if v, err := parseMoney("m", ""); err != nil || !v.IsZero() {
		t.Error("an empty price is zero")
	}
	for _, bad := range []string{"-0.01", "1.23456", "x"} {
		if _, err := parseMoney("m", bad); err == nil {
			t.Errorf("price %q should be refused", bad)
		}
	}
}

func TestLockOrderAndScope(t *testing.T) {
	a, b := uuid.MustParse("00000000-0000-0000-0000-00000000000a"), uuid.MustParse("00000000-0000-0000-0000-00000000000b")
	ids := []uuid.UUID{b, a, b}
	order := lockOrder(len(ids), func(i int) uuid.UUID { return ids[i] })
	if ids[order[0]] != a || ids[order[1]] != b || ids[order[2]] != b {
		t.Errorf("lock order = %v", order)
	}
	own := BranchScope{IDs: []uuid.UUID{a}}
	if !own.Has(a) || own.Has(b) {
		t.Error("an own-branch scope sees only its branches")
	}
	if !(BranchScope{All: true}).Has(b) {
		t.Error("an all-branch scope sees every branch")
	}
}

func line(qty, amount, discount string) parsedLine {
	q, a, dd := d(qty), d(amount), d(discount)
	return parsedLine{qty: q, amount: a, discount: dd, lineTotal: q.Mul(a).Sub(dd)}
}

func TestDocTotals(t *testing.T) {
	lines := []parsedLine{line("2", "10", "5"), line("1", "30", "0")} // 15 + 30
	sub, disc, ship, total, err := docTotals(lines, "5", "3")
	if err != nil || !sub.Equal(d("45")) || !disc.Equal(d("5")) || !ship.Equal(d("3")) || !total.Equal(d("43")) {
		t.Fatalf("got %s %s %s %s %v", sub, disc, ship, total, err)
	}
	if _, _, _, _, err := docTotals(lines, "46", ""); err == nil {
		t.Error("a discount above the subtotal is refused")
	}
	if _, _, _, _, err := docTotals(lines, "-1", ""); err == nil {
		t.Error("a negative discount is refused")
	}
}

func TestLandedCostsSpreadShippingAndDiscount(t *testing.T) {
	// Lines worth 60 and 40; shipping 15 and discount 5 → +10 spread 6 / 4.
	lines := []parsedLine{line("6", "10", "0"), line("4", "10", "0")}
	got := landedCosts(lines, d("5"), d("15"))
	if !got[0].Equal(d("11")) || !got[1].Equal(d("11")) {
		t.Errorf("landed = %v, want 11 and 11", got)
	}
	// Unequal values: 100 (10 × 10) and 50 (5 × 10) with shipping 30 → 20 and 10.
	lines = []parsedLine{line("10", "10", "0"), line("5", "10", "0")}
	got = landedCosts(lines, d("0"), d("30"))
	if !got[0].Equal(d("12")) || !got[1].Equal(d("12")) {
		t.Errorf("landed = %v, want 12 and 12", got)
	}
	// Free goods share shipping by quantity.
	lines = []parsedLine{line("1", "0", "0"), line("3", "0", "0")}
	got = landedCosts(lines, d("0"), d("8"))
	if !got[0].Equal(d("2")) || !got[1].Equal(d("2")) {
		t.Errorf("free goods landed = %v, want 2 and 2", got)
	}
}

func TestSaleDiscountShares(t *testing.T) {
	lines := []parsedLine{line("1", "30", "0"), line("1", "10", "0")}
	s := shares(lines, d("8"))
	if !s[0].Equal(d("6")) || !s[1].Equal(d("2")) {
		t.Errorf("shares = %v, want 6 and 2", s)
	}
}

func TestCleanNote(t *testing.T) {
	out, err := cleanNote(`<p>Paid <b>half</b><script>alert(1)</script></p>`)
	if err != nil || strings.Contains(out, "script") || !strings.Contains(out, "<b>half</b>") {
		t.Errorf("cleanNote = %q, %v", out, err)
	}
	if out, _ := cleanNote("<p> </p>"); out != "" {
		t.Errorf("an empty note is stored empty, got %q", out)
	}
	if _, err := cleanNote(strings.Repeat("a", 1001)); err == nil {
		t.Error("a note over 1000 characters is refused")
	}
}

func TestRequiredPrice(t *testing.T) {
	if err := requiredPrice("sale_price", "sale price", "", decimal.Zero); err == nil || !strings.Contains(err.Error(), "required") {
		t.Errorf("an empty price is refused as required, got %v", err)
	}
	if err := requiredPrice("sale_price", "sale price", "0", decimal.Zero); err == nil || !strings.Contains(err.Error(), "greater than 0") {
		t.Errorf("a price of 0 is refused, got %v", err)
	}
	if err := requiredPrice("cost_price", "usual cost", "1.25", d("1.25")); err != nil {
		t.Errorf("a price above 0 is accepted, got %v", err)
	}
}

func TestParseDueDate(t *testing.T) {
	s := &InventoryService{loc: time.FixedZone("ICT", 7*3600)}
	sold, _ := s.parseDocDate("sale_date", "2026-10-28")
	if due, err := s.parseDueDate("", sold); err != nil || due != nil {
		t.Errorf("no due date: got %v, %v", due, err)
	}
	if due, err := s.parseDueDate("2026-10-28", sold); err != nil || due == nil {
		t.Errorf("due the day of the sale should be accepted, got %v", err)
	}
	if due, err := s.parseDueDate("2026-11-27", sold); err != nil || due.Format("2006-01-02") != "2026-11-27" {
		t.Errorf("due in 30 days: got %v, %v", due, err)
	}
	if _, err := s.parseDueDate("2026-10-27", sold); err == nil {
		t.Error("a due date before the sale date is refused")
	}
	if _, err := s.parseDueDate("27/10/2026", sold); err == nil {
		t.Error("a date that isn't YYYY-MM-DD is refused")
	}
}

func TestToBase(t *testing.T) {
	if got := toBase(d("41000"), "KHR", d("4100")); !got.Equal(d("10")) {
		t.Errorf("41000 KHR at 4100 = %s USD, want 10", got)
	}
	if got := toBase(d("12.5"), "USD", d("4100")); !got.Equal(d("12.5")) {
		t.Errorf("USD stays USD, got %s", got)
	}
	if got := toBase(d("1000"), "KHR", d("4100")); !got.Equal(d("0.2439")) {
		t.Errorf("1000 KHR = %s USD, want 0.2439 (4 places)", got)
	}
}
