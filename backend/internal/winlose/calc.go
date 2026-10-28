// Package winlose works out Jame-PS (shown in the app as "Total") from a platform's weekly
// win/lose. The formula lives only here. Each formula is a version: a saved week keeps the
// version, rate and currency it was worked out with, so changing the formula (or a rate) never
// changes weeks already saved.
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
	calc      func(Input) (adjusted, jamePS decimal.Decimal)
}

// V1 is the formula read from the spreadsheet; it still has to be confirmed by the business:
//
//	adjusted = win_lose + commission
//	jame_ps  = ROUND(adjusted × rate, 0)
//
// ROUND is half away from zero, like Excel's (2.5 → 3, -2.5 → -3).
const V1 = "v1"

// Default is the version new weeks use unless the winlose.formula setting picks another.
const Default = V1

var hundred = decimal.NewFromInt(100)

var formulas = map[string]Formula{
	V1: {
		Version:     V1,
		Description: "Total = ROUND((Win/Lose + Commission) × Rate, 0)",
		calc: func(in Input) (decimal.Decimal, decimal.Decimal) {
			adjusted := in.WinLose.Add(in.Commission)
			return adjusted, adjusted.Mul(in.RatePct).Div(hundred).Round(0)
		},
	},
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
