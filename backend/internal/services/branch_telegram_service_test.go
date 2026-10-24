package services

import (
	"strings"
	"testing"
	"time"

	"github.com/cambodia-finance/income-expense/internal/models"
	"github.com/shopspring/decimal"
)

func TestFormatTelegramAmount(t *testing.T) {
	cases := map[string][2]string{
		"1,234.50 USD":  {"1234.5", "USD"},
		"12.00 USD":     {"12", "USD"},
		"1,250,000 KHR": {"1250000", "KHR"},
		"-1,000.25 USD": {"-1000.25", "USD"},
		"999 KHR":       {"999.4", "KHR"},
	}
	for want, in := range cases {
		if got := formatTelegramAmount(decimal.RequireFromString(in[0]), in[1]); got != want {
			t.Errorf("formatTelegramAmount(%s %s) = %q, want %q", in[0], in[1], got, want)
		}
	}
}

func TestTransactionMessage(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Phnom_Penh")
	txn := &models.Transaction{
		TransactionNumber: "EXP-2026-000042", Type: models.TxnTypeExpense, Status: models.TxnStatusPending,
		Currency: "USD", Amount: decimal.RequireFromString("1520.5"),
		TransactionDate: time.Date(2026, 10, 24, 3, 0, 0, 0, time.UTC),
		Branch:          &models.Branch{Code: "PP-01", Name: "Phnom Penh"},
		Category:        &models.Category{Name: "Utilities"},
		PaymentMethod:   &models.PaymentMethod{Name: "Cash"},
	}
	created := transactionMessage(TelegramEventCreated, txn, "Dara", "Dara", "", "", loc)
	for _, part := range []string{"New expense · waiting for approval", "Branch: PP-01 · Phnom Penh", "EXP-2026-000042", "Amount: 1,520.50 USD", "Category: Utilities · Cash", "Date: 24 Oct 2026", "By: Dara"} {
		if !strings.Contains(created, part) {
			t.Errorf("created message lacks %q:\n%s", part, created)
		}
	}
	if strings.Contains(created, "Approved by") {
		t.Errorf("created message names an approver:\n%s", created)
	}
	rejected := transactionMessage(TelegramEventRejected, txn, "Dara", "Sokha", "  no receipt  ", "", loc)
	for _, part := range []string{"Expense rejected", "Rejected by: Sokha", "Reason: no receipt"} {
		if !strings.Contains(rejected, part) {
			t.Errorf("rejected message lacks %q:\n%s", part, rejected)
		}
	}
	reversed := transactionMessage(TelegramEventReversed, txn, "Dara", "Sokha", "typo", "EXP-2026-000001", loc)
	if !strings.Contains(reversed, "EXP-2026-000042 (reverses EXP-2026-000001)") {
		t.Errorf("reversed message lacks the original number:\n%s", reversed)
	}
}

func TestTelegramInputRules(t *testing.T) {
	for _, ok := range []string{"-1001234567890", "123456789", "@branch_pp01"} {
		if !telegramChatRule.MatchString(ok) {
			t.Errorf("chat %q should be accepted", ok)
		}
	}
	for _, bad := range []string{"", "abc", "-12", "@ab", "https://t.me/x"} {
		if telegramChatRule.MatchString(bad) {
			t.Errorf("chat %q should be refused", bad)
		}
	}
	if !telegramTokenRule.MatchString("123456789:AAHdqTcvCH1vGWJxfSeofSAs0K5PALDsaw") {
		t.Error("a BotFather token should be accepted")
	}
	if telegramTokenRule.MatchString("not-a-token") {
		t.Error("a malformed token should be refused")
	}
}
