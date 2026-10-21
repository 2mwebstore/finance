package accounting

import (
	"errors"
	"strings"

	"github.com/cambodia-finance/income-expense/internal/models"
	"github.com/shopspring/decimal"
)

var (
	ErrInvalidCurrency = errors.New("currency must be USD or KHR")
	ErrInvalidAmount   = errors.New("amount must be a positive number with at most 4 decimal places")
	ErrInvalidTxnType  = errors.New("type must be income or expense")
	ErrAmountTooLarge  = errors.New("amount is too large (maximum 999,999,999,999)")
)

// MaxAmount keeps amounts well inside DECIMAL(19,4) and exact in browser number formatting.
var MaxAmount = decimal.RequireFromString("999999999999")

// ParseAmount parses a string amount into decimal.Decimal and validates it.
func ParseAmount(s string) (decimal.Decimal, error) {
	d, err := decimal.NewFromString(strings.TrimSpace(s))
	if err != nil {
		return decimal.Zero, ErrInvalidAmount
	}
	return d, ValidateAmount(d)
}

// ValidateAmount ensures amount is positive and has at most 4 decimal places.
func ValidateAmount(amount decimal.Decimal) error {
	if !amount.IsPositive() {
		return ErrInvalidAmount
	}
	// Exponent is negative of decimal places; e.g. 1.2345 has exponent -4
	if amount.Exponent() < -4 {
		return ErrInvalidAmount
	}
	if amount.GreaterThan(MaxAmount) {
		return ErrAmountTooLarge
	}
	return nil
}

// ValidateCurrency checks supported currencies.
func ValidateCurrency(c string) error {
	if !models.ValidateCurrency(c) {
		return ErrInvalidCurrency
	}
	return nil
}

// ValidateTxnType checks income|expense.
func ValidateTxnType(t string) error {
	if !models.ValidateTxnType(t) {
		return ErrInvalidTxnType
	}
	return nil
}

// FormatDisplay formats for UI display.
// USD → 2 decimal places; KHR → 0 decimal places (whole riel).
func FormatDisplay(currency string, amount decimal.Decimal) string {
	switch currency {
	case models.CurrencyUSD:
		return amount.StringFixed(2)
	case models.CurrencyKHR:
		return amount.StringFixed(0)
	default:
		return amount.String()
	}
}

// Zero returns decimal zero (avoid float literals).
func Zero() decimal.Decimal {
	return decimal.Zero
}

// SumAmounts adds amounts that share the same currency.
// Caller MUST ensure all amounts are the same currency — this does not convert.
func SumAmounts(amounts ...decimal.Decimal) decimal.Decimal {
	sum := decimal.Zero
	for _, a := range amounts {
		sum = sum.Add(a)
	}
	return sum
}
