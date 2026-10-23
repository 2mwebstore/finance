package database

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/cambodia-finance/income-expense/internal/config"
	"gorm.io/gorm"
)

func traceAll(cfg *config.Config) string {
	var out bytes.Buffer
	l := newLogger(cfg, &out)
	ctx := context.Background()
	sql := func(q string) func() (string, int64) { return func() (string, int64) { return q, 1 } }
	now := time.Now()
	l.Trace(ctx, now, sql("SELECT ok"), nil)
	l.Trace(ctx, now, sql("SELECT missing"), gorm.ErrRecordNotFound)
	l.Trace(ctx, now, sql("SELECT broken"), errors.New("boom"))
	l.Trace(ctx, now.Add(-time.Second), sql("SELECT slow"), nil)
	return out.String()
}

// Hosts like Railway drop log lines past a per-second limit, so ordinary statements stay quiet.
func TestLoggerPrintsOnlyErrorsAndSlowQueriesByDefault(t *testing.T) {
	got := traceAll(&config.Config{})
	for _, quiet := range []string{"SELECT ok", "SELECT missing"} {
		if strings.Contains(got, quiet) {
			t.Errorf("%q should not be logged:\n%s", quiet, got)
		}
	}
	for _, loud := range []string{"SELECT broken", "SELECT slow"} {
		if !strings.Contains(got, loud) {
			t.Errorf("%q should be logged:\n%s", loud, got)
		}
	}
	if strings.Contains(got, "\x1b[") {
		t.Errorf("log has colour codes:\n%q", got)
	}
}

func TestLoggerPrintsEveryStatementWhenAsked(t *testing.T) {
	if got := traceAll(&config.Config{LogSQL: true}); !strings.Contains(got, "SELECT ok") {
		t.Fatalf("DB_LOG_SQL should print every statement:\n%s", got)
	}
}
