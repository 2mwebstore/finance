package accounting

import (
	"testing"

	"github.com/shopspring/decimal"
)

func TestParseAndValidateAmount(t *testing.T) {
	d, err := ParseAmount("100.50")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if !d.Equal(decimal.RequireFromString("100.50")) {
		t.Fatalf("got %s", d)
	}

	if _, err := ParseAmount("-1"); err == nil {
		t.Fatal("negative should fail")
	}
	if _, err := ParseAmount("0"); err == nil {
		t.Fatal("zero should fail")
	}
	if _, err := ParseAmount("1.12345"); err == nil {
		t.Fatal("5 decimal places should fail")
	}
	if _, err := ParseAmount("999999999999"); err != nil {
		t.Fatalf("maximum amount should pass: %v", err)
	}
	if _, err := ParseAmount("1000000000000"); err != ErrAmountTooLarge {
		t.Fatalf("13-digit amount should be too large, got %v", err)
	}
}

func TestValidateCurrency(t *testing.T) {
	if err := ValidateCurrency("USD"); err != nil {
		t.Fatal(err)
	}
	if err := ValidateCurrency("KHR"); err != nil {
		t.Fatal(err)
	}
	if err := ValidateCurrency("EUR"); err == nil {
		t.Fatal("EUR should be rejected")
	}
}

func TestSumDoesNotMixCurrenciesByDesign(t *testing.T) {
	// SumAmounts only adds numbers — currency separation is caller's responsibility
	a := decimal.RequireFromString("100.00")
	b := decimal.RequireFromString("50.25")
	sum := SumAmounts(a, b)
	if !sum.Equal(decimal.RequireFromString("150.25")) {
		t.Fatalf("sum=%s", sum)
	}
}

func TestFormatDisplay(t *testing.T) {
	usd := decimal.RequireFromString("1234.5")
	if got := FormatDisplay("USD", usd); got != "1234.50" {
		t.Fatalf("USD format: %s", got)
	}
	khr := decimal.RequireFromString("150000.00")
	if got := FormatDisplay("KHR", khr); got != "150000" {
		t.Fatalf("KHR format: %s", got)
	}
}

func TestNoFloatMoney(t *testing.T) {
	// Demonstrate exactness that float would lose
	a := decimal.RequireFromString("0.1")
	b := decimal.RequireFromString("0.2")
	sum := a.Add(b)
	if !sum.Equal(decimal.RequireFromString("0.3")) {
		t.Fatalf("decimal 0.1+0.2 should be 0.3, got %s", sum)
	}
}
