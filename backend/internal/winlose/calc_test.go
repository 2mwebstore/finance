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
	if Default != V2 {
		t.Error("new weeks use V2")
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
