package winlose

import (
	"testing"

	"github.com/shopspring/decimal"
)

func d(s string) decimal.Decimal { return decimal.RequireFromString(s) }

func TestV1(t *testing.T) {
	cases := []struct {
		name                      string
		winLose, commission, rate string
		wantAdjusted, wantJamePS  string
	}{
		{"win plus commission at 35%", "1000", "200", "35", "1200", "420"},
		{"a loss", "-1500", "100", "35", "-1400", "-490"},
		{"rounds half away from zero (up)", "5", "0", "50", "5", "3"},      // 2.5 → 3
		{"rounds half away from zero (down)", "-5", "0", "50", "-5", "-3"}, // -2.5 → -3
		{"below half rounds down", "1", "0", "49", "1", "0"},               // 0.49 → 0
		{"decimals in the inputs", "10.25", "0.75", "10", "11", "1"},       // 1.1 → 1
		{"rate 0", "1000", "0", "0", "1000", "0"},
		{"riel amounts", "-410000", "20500", "12.5", "-389500", "-48688"}, // -48687.5 → -48688
	}
	for _, c := range cases {
		got, err := Calculate(V1, Input{WinLose: d(c.winLose), Commission: d(c.commission), RatePct: d(c.rate)})
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		if !got.Adjusted.Equal(d(c.wantAdjusted)) || !got.JamePS.Equal(d(c.wantJamePS)) || got.Version != V1 {
			t.Errorf("%s: got adjusted=%s jame=%s (%s), want %s and %s", c.name, got.Adjusted, got.JamePS, got.Version, c.wantAdjusted, c.wantJamePS)
		}
	}
}

func TestV2(t *testing.T) {
	cases := []struct {
		name                      string
		winLose, commission, rate string
		wantAdjusted, wantJamePS  string
	}{
		{"commission taken away at 10%", "1000", "100", "10", "900", "90"},
		{"a loss", "-1500", "100", "35", "-1600", "-560"},
		{"rounds half away from zero", "5", "0", "50", "5", "3"},          // 2.5 → 3
		{"riel amounts", "-410000", "20500", "12.5", "-430500", "-53813"}, // -53812.5 → -53813
	}
	for _, c := range cases {
		got, err := Calculate(V2, Input{WinLose: d(c.winLose), Commission: d(c.commission), RatePct: d(c.rate)})
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		if !got.Adjusted.Equal(d(c.wantAdjusted)) || !got.JamePS.Equal(d(c.wantJamePS)) || got.Version != V2 {
			t.Errorf("%s: got adjusted=%s jame=%s (%s), want %s and %s", c.name, got.Adjusted, got.JamePS, got.Version, c.wantAdjusted, c.wantJamePS)
		}
	}
}

func TestV3(t *testing.T) {
	cases := []struct {
		name                      string
		winLose, commission, rate string
		wantAdjusted, wantJamePS  string
	}{
		{"the agent's 60% leaves 40%", "1000", "100", "60", "900", "360"},
		{"rate 10% leaves 90%", "1000", "100", "10", "900", "810"},
		{"rate 0 leaves it all", "1000", "100", "0", "900", "900"},
		{"rate 100% leaves nothing", "1000", "100", "100", "900", "0"},
		{"a loss", "-1500", "100", "35", "-1600", "-1040"},
		{"rounds half away from zero", "5", "0", "50", "5", "3"},          // 2.5 → 3
		{"riel amounts", "-410000", "20500", "87.5", "-430500", "-53813"}, // -53812.5 → -53813
	}
	for _, c := range cases {
		got, err := Calculate(V3, Input{WinLose: d(c.winLose), Commission: d(c.commission), RatePct: d(c.rate)})
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		if !got.Adjusted.Equal(d(c.wantAdjusted)) || !got.JamePS.Equal(d(c.wantJamePS)) || got.Version != V3 {
			t.Errorf("%s: got adjusted=%s jame=%s (%s), want %s and %s", c.name, got.Adjusted, got.JamePS, got.Version, c.wantAdjusted, c.wantJamePS)
		}
	}
	if f, _ := Lookup(V3); !f.Rest {
		t.Error("V3: the rate is the agent's share")
	}
}

func TestUnknownVersion(t *testing.T) {
	if _, err := Calculate("v0", Input{}); err != ErrUnknownFormula {
		t.Errorf("an unknown version is refused, got %v", err)
	}
	if _, ok := Lookup(Default); !ok {
		t.Error("the default version must exist")
	}
	if len(Versions()) == 0 || Versions()[0].Description == "" {
		t.Error("versions are listed with a description")
	}
}

func TestKinds(t *testing.T) {
	cases := []struct {
		name                      string
		kind                      Kind
		winLose, commission, rate string
		wantAdjusted, wantJamePS  string
	}{
		{"PO is what is left of 100%", PO, "1000", "100", "60", "900", "360"},
		{"empty kind is PO", "", "1000", "100", "60", "900", "360"},
		{"GI is minus the agent's share", GI, "1000", "100", "60", "900", "-540"},
		{"GI on a loss turns positive", GI, "-1500", "100", "35", "-1600", "560"},
		{"GI rounds half away from zero", GI, "5", "0", "50", "5", "-3"},
		{"GI at 0%", GI, "1000", "0", "0", "1000", "0"},
	}
	for _, c := range cases {
		got, err := CalculateKind(V3, c.kind, Input{WinLose: d(c.winLose), Commission: d(c.commission), RatePct: d(c.rate)})
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		if !got.Adjusted.Equal(d(c.wantAdjusted)) || !got.JamePS.Equal(d(c.wantJamePS)) {
			t.Errorf("%s: got adjusted=%s jame=%s, want %s and %s", c.name, got.Adjusted, got.JamePS, c.wantAdjusted, c.wantJamePS)
		}
	}
}

// A group holds up to 100 agents, each at its own rate and type: every Total must match the
// formula worked out by hand (PO = adjusted × (100% − rate), GI = −(adjusted × rate)).
func TestHundredAgents(t *testing.T) {
	var po, gi decimal.Decimal
	for i := 1; i <= 100; i++ {
		rate := decimal.NewFromInt(int64(i - 1)) // 0% … 99%
		in := Input{WinLose: decimal.NewFromInt(int64(-1000 * i)), Commission: decimal.NewFromInt(2000), RatePct: rate}
		adjusted := in.WinLose.Sub(in.Commission)
		p, err := CalculateKind(V3, PO, in)
		if err != nil {
			t.Fatal(err)
		}
		g, err := CalculateKind(V3, GI, in)
		if err != nil {
			t.Fatal(err)
		}
		if want := adjusted.Mul(decimal.NewFromInt(100).Sub(rate)).Div(decimal.NewFromInt(100)).Round(0); !p.JamePS.Equal(want) {
			t.Fatalf("agent %d PO: got %s want %s", i, p.JamePS, want)
		}
		if want := adjusted.Mul(rate).Div(decimal.NewFromInt(100)).Round(0).Neg(); !g.JamePS.Equal(want) {
			t.Fatalf("agent %d GI: got %s want %s", i, g.JamePS, want)
		}
		if !p.Adjusted.Equal(adjusted) || !g.Adjusted.Equal(adjusted) {
			t.Fatalf("agent %d adjusted: %s / %s want %s", i, p.Adjusted, g.Adjusted, adjusted)
		}
		po, gi = po.Add(p.JamePS), gi.Add(g.JamePS)
	}
	t.Logf("100 agents: PO total %s, GI total %s", po, gi)
}

func TestV4(t *testing.T) {
	cases := []struct {
		name                      string
		kind                      Kind
		winLose, commission, rate string
		wantAdjusted, wantJamePS  string
	}{
		{"PO: commission added, what is left of 100%", PO, "1000", "100", "60", "1100", "440"},
		{"PO at 0% leaves the whole amount", PO, "1000", "100", "0", "1100", "1100"},
		{"PO at 100% leaves nothing", PO, "1000", "100", "100", "1100", "0"},
		{"PO on a loss", PO, "-100000", "10000", "60", "-90000", "-36000"},
		{"GI: minus the share of win/lose plus commission", GI, "1000", "100", "60", "1100", "-660"},
		{"GI on a loss turns positive", GI, "-1500", "100", "35", "-1400", "490"},
		{"GI rounds half away from zero", GI, "4", "1", "50", "5", "-3"},
		{"PO rounds half away from zero", PO, "4", "1", "50", "5", "3"},
	}
	for _, c := range cases {
		got, err := CalculateKind(V4, c.kind, Input{WinLose: d(c.winLose), Commission: d(c.commission), RatePct: d(c.rate)})
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		if !got.Adjusted.Equal(d(c.wantAdjusted)) || !got.JamePS.Equal(d(c.wantJamePS)) || got.Version != V4 {
			t.Errorf("%s: got adjusted=%s jame=%s (%s), want %s and %s", c.name, got.Adjusted, got.JamePS, got.Version, c.wantAdjusted, c.wantJamePS)
		}
	}
	if Default != V4 {
		t.Errorf("default formula is %s, want %s", Default, V4)
	}
	// v3 stays as it was for the weeks saved with it.
	old, _ := CalculateKind(V3, GI, Input{WinLose: d("1000"), Commission: d("100"), RatePct: d("60")})
	if !old.JamePS.Equal(d("-540")) {
		t.Errorf("v3 GI changed: %s", old.JamePS)
	}
}

// 100 agents at rates 0%–99%, PO and GI, under v4: each Total matches the formula by hand.
func TestHundredAgentsV4(t *testing.T) {
	for i := 1; i <= 100; i++ {
		rate := decimal.NewFromInt(int64(i - 1))
		in := Input{WinLose: decimal.NewFromInt(int64(-1000 * i)), Commission: decimal.NewFromInt(2000), RatePct: rate}
		adjusted := in.WinLose.Add(in.Commission)
		p, _ := CalculateKind(V4, PO, in)
		g, _ := CalculateKind(V4, GI, in)
		if want := adjusted.Mul(decimal.NewFromInt(100).Sub(rate)).Div(decimal.NewFromInt(100)).Round(0); !p.JamePS.Equal(want) {
			t.Fatalf("agent %d PO: got %s want %s", i, p.JamePS, want)
		}
		if want := adjusted.Mul(rate).Div(decimal.NewFromInt(100)).Round(0).Neg(); !g.JamePS.Equal(want) {
			t.Fatalf("agent %d GI: got %s want %s", i, g.JamePS, want)
		}
	}
}
