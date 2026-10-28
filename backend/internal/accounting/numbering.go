package accounting

import (
	"context"
	"fmt"
	"time"

	"github.com/cambodia-finance/income-expense/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// NumberGenerator issues unique human-readable transaction numbers under concurrency.
// Format: INC-202610-000001 or EXP-202610-000001
type NumberGenerator struct {
	db *gorm.DB
	loc *time.Location
}

func NewNumberGenerator(db *gorm.DB, timezone string) *NumberGenerator {
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		loc = time.FixedZone("Asia/Phnom_Penh", 7*3600)
	}
	return &NumberGenerator{db: db, loc: loc}
}

// Next returns the next transaction number for the given type (income|expense).
// Uses row-level lock on the sequence table so concurrent requests cannot duplicate.
func (g *NumberGenerator) Next(ctx context.Context, txnType string) (string, error) {
	prefixCode := "INC"
	if txnType == models.TxnTypeExpense {
		prefixCode = "EXP"
	}
	return g.NextFor(ctx, prefixCode)
}

// NextFor returns the next number for a document code, e.g. "PUR" → PUR-202610-000001
// (sales SAL, purchases PUR, stock transfers TRF, inventory expenses IEX).
func (g *NumberGenerator) NextFor(ctx context.Context, prefixCode string) (string, error) {
	now := time.Now().In(g.loc)
	return g.next(ctx, fmt.Sprintf("%s-%s", prefixCode, now.Format("200601")))
}

// NextPlain returns the next number of a sequence that never restarts, e.g. "SKU" → SKU-000001.
func (g *NumberGenerator) NextPlain(ctx context.Context, prefix string) (string, error) {
	return g.next(ctx, prefix)
}

func (g *NumberGenerator) next(ctx context.Context, prefix string) (string, error) {
	var number string
	err := g.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var seq models.TransactionNumberSeq
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("prefix = ?", prefix).
			First(&seq).Error

		if err == gorm.ErrRecordNotFound {
			seq = models.TransactionNumberSeq{
				ID:        uuid.New(),
				Prefix:    prefix,
				LastValue: 1,
				UpdatedAt: time.Now(),
			}
			if err := tx.Create(&seq).Error; err != nil {
				// Race: another goroutine inserted — retry lock path
				if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
					Where("prefix = ?", prefix).First(&seq).Error; err != nil {
					return err
				}
				seq.LastValue++
				seq.UpdatedAt = time.Now()
				if err := tx.Save(&seq).Error; err != nil {
					return err
				}
			}
		} else if err != nil {
			return err
		} else {
			seq.LastValue++
			seq.UpdatedAt = time.Now()
			if err := tx.Save(&seq).Error; err != nil {
				return err
			}
		}

		number = fmt.Sprintf("%s-%06d", prefix, seq.LastValue)
		return nil
	})
	return number, err
}
