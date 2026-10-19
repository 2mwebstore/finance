package accounting

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"

	"github.com/cambodia-finance/income-expense/internal/config"
	"github.com/cambodia-finance/income-expense/internal/database"
	"github.com/cambodia-finance/income-expense/internal/models"
)

// Integration-style test: requires DB env vars. Skips if DB unreachable.
func TestConcurrentTransactionNumbers(t *testing.T) {
	cfg := config.Load()
	if cfg.DBPassword == "" && os.Getenv("DB_PASSWORD") == "" {
		// try defaults used in dev
		os.Setenv("DB_HOST", "127.0.0.1")
		os.Setenv("DB_PORT", "3306")
		os.Setenv("DB_USER", "finance")
		os.Setenv("DB_PASSWORD", "finance_dev_password")
		os.Setenv("DB_NAME", "income_expense")
		cfg = config.Load()
	}
	db, err := database.Connect(cfg)
	if err != nil {
		t.Skipf("db not available: %v", err)
	}
	_ = db.AutoMigrate(&models.TransactionNumberSeq{})

	gen := NewNumberGenerator(db, "Asia/Phnom_Penh")
	const n = 50
	results := make([]string, n)
	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func(idx int) {
			defer wg.Done()
			num, err := gen.Next(context.Background(), models.TxnTypeIncome)
			if err != nil {
				t.Errorf("next: %v", err)
				return
			}
			results[idx] = num
		}(i)
	}
	wg.Wait()

	seen := map[string]bool{}
	for i, r := range results {
		if r == "" {
			t.Fatalf("empty number at %d", i)
		}
		if seen[r] {
			t.Fatalf("duplicate transaction number: %s", r)
		}
		seen[r] = true
	}
	if len(seen) != n {
		t.Fatalf("expected %d unique numbers, got %d", n, len(seen))
	}
	fmt.Printf("generated %d unique numbers, sample: %s\n", n, results[0])
}
