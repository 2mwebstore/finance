// Package winlose works out Jame-PS (shown in the app as "Total") from a platform's weekly
// win/lose. The formula lives only here. Each formula is a version: a saved week keeps the rate
// and currency it was worked out with; a finalized week also keeps its version, while weeks not
// finalized follow the current one (see PlatformService.RecalculateOpenWeeks).
package winlose

import (
	"errors"
	"sort"

	"github.com/shopspring/decimal"
)

// Input is one HQ's week on one platform.
type Input struct {
	WinLose    decimal.Decimal
	Commission decimal.Decimal
	// RatePct is the commission rate in percent: 35 means 35%.
	RatePct decimal.Decimal
}

// Result is what the formula gives for an Input.
type Result struct {
	Version  string          `json:"formula_version"`
	Adjusted decimal.Decimal `json:"adjusted_amount"`
	JamePS   decimal.Decimal `json:"jame_ps"`
}

// Formula is one version of the calculation.
type Formula struct {
	Version     string `json:"version"`
	Description string `json:"description"`
	// Confirmed is false until the business has checked the formula against real figures.
	Confirmed bool `json:"confirmed"`
	// Rest: the rate is the agent's own share and the Total what is left (100% − rate).
	Rest bool `json:"rest"`
	calc func(Input) (adjusted, jamePS decimal.Decimal)
}

// V1 is the formula read from the spreadsheet; it still has to be confirmed by the business:
//
//	adjusted = win_lose + commission
//	jame_ps  = ROUND(adjusted × rate, 0)
//
// ROUND is half away from zero, like Excel's (2.5 → 3, -2.5 → -3).
const V1 = "v1"

// V2 takes the commission away from the win/lose (the business's rule):
//
//	adjusted = win_lose − commission
//	jame_ps  = ROUND(adjusted × rate, 0)
//
// e.g. (1,000 − 100) × 10% = 90.
const V2 = "v2"

// V3 is V2 with the rate the agent's own share, so the Total is at what is left of 100%:
//
//	adjusted = win_lose − commission
//	jame_ps  = ROUND(adjusted × (100% − rate), 0)
//
// e.g. (1,000 − 100) × (100% − 60%) = 360.
const V3 = "v3"

// V4 is V3 with the commission added to the win/lose instead of taken away (the business's
// rule from November 2026):
//
//	adjusted = win_lose + commission
//	jame_ps  = ROUND(adjusted × (100% − rate), 0)
//
// e.g. (1,000 + 100) × (100% − 60%) = 440. A GI group under it gives −((win_lose + commission) × rate).
const V4 = "v4"

// Default is the version weeks use unless the winlose.formula setting picks another.
const Default = V4

var hundred = decimal.NewFromInt(100)

var formulas = map[string]Formula{
	V1: {
		Version:     V1,
		Description: "Total = ROUND((Win/Lose + Commission) × Partner %, 0)",
		calc: func(in Input) (decimal.Decimal, decimal.Decimal) {
			adjusted := in.WinLose.Add(in.Commission)
			return adjusted, adjusted.Mul(in.RatePct).Div(hundred).Round(0)
		},
	},
	V2: {
		Version:     V2,
		Description: "Total = ROUND((Win/Lose − Commission) × rate, 0)",
		Confirmed:   true,
		calc: func(in Input) (decimal.Decimal, decimal.Decimal) {
			adjusted := in.WinLose.Sub(in.Commission)
			return adjusted, adjusted.Mul(in.RatePct).Div(hundred).Round(0)
		},
	},
	V3: {
		Version:     V3,
		Description: "Total = ROUND((Win/Lose − Commission) × (100% − rate), 0)",
		Confirmed:   true,
		Rest:        true,
		calc: func(in Input) (decimal.Decimal, decimal.Decimal) {
			adjusted := in.WinLose.Sub(in.Commission)
			return adjusted, adjusted.Mul(hundred.Sub(in.RatePct)).Div(hundred).Round(0)
		},
	},
	V4: {
		Version:     V4,
		Description: "Total = ROUND((Win/Lose + Commission) × (100% − rate), 0)",
		Confirmed:   true,
		Rest:        true,
		calc: func(in Input) (decimal.Decimal, decimal.Decimal) {
			adjusted := in.WinLose.Add(in.Commission)
			return adjusted, adjusted.Mul(hundred.Sub(in.RatePct)).Div(hundred).Round(0)
		},
	},
}

// Kind is how an agent splits the adjusted amount (the version's win/lose and commission). It is
// set per agent on the Assign agents page and works on top of the formula version.
type Kind string

const (
	// PO: the agent's rate is its own share and the Total is what is left, as the version says
	// (v4: (W/L + COMM) × (100% − rate); v3: (W/L − COMM) × (100% − rate)).
	PO Kind = "PO"
	// GI: the Total is the agent's share, with the sign turned over: −((W/L + COMM) × rate)
	// (under v3: −((W/L − COMM) × rate)).
	GI Kind = "GI"
)

// ValidKind reports whether k is a kind this build knows.
func ValidKind(k string) bool { return k == string(PO) || k == string(GI) }

// CalculateKind works a week out for an agent of the given kind. An empty or PO kind follows the
// formula version; GI is −ROUND(adjusted × rate, 0), with the adjusted amount of that version
// (win_lose + commission under v4, win_lose − commission under v3).
func CalculateKind(version string, kind Kind, in Input) (Result, error) {
	if kind != GI {
		return Calculate(version, in)
	}
	f, ok := formulas[version]
	if !ok {
		return Result{}, ErrUnknownFormula
	}
	adjusted, _ := f.calc(Input{WinLose: in.WinLose, Commission: in.Commission})
	jame := adjusted.Mul(in.RatePct).Div(hundred).Round(0).Neg()
	return Result{Version: version, Adjusted: adjusted.Round(4), JamePS: jame}, nil
}

// ErrUnknownFormula: a version this build doesn't have.
var ErrUnknownFormula = errors.New("unknown win/lose formula version")

// Lookup returns a formula by version.
func Lookup(version string) (Formula, bool) {
	f, ok := formulas[version]
	return f, ok
}

// Versions lists every formula, oldest first.
func Versions() []Formula {
	out := make([]Formula, 0, len(formulas))
	for _, f := range formulas {
		out = append(out, f)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Version < out[j].Version })
	return out
}

// Calculate works out a week with the given formula version.
func Calculate(version string, in Input) (Result, error) {
	f, ok := formulas[version]
	if !ok {
		return Result{}, ErrUnknownFormula
	}
	adjusted, jame := f.calc(in)
	return Result{Version: version, Adjusted: adjusted.Round(4), JamePS: jame}, nil
}
