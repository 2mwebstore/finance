package accounting

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"time"

	"github.com/cambodia-finance/income-expense/internal/models"
	"github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// NumberGenerator issues unique human-readable transaction numbers under concurrency.
// Format: INC-202610-000001 or EXP-202610-000001
type NumberGenerator struct {
	db  *gorm.DB
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

// maxNumberAttempts is how many times a number is asked for when MySQL picks the transaction as a
// deadlock victim (see isDeadlock).
const maxNumberAttempts = 6

// isDeadlock: MySQL rolled the transaction back because it deadlocked with another (1213), or gave
// up waiting for a lock (1205). Either way nothing was written, so asking again is safe.
func isDeadlock(err error) bool {
	var me *mysql.MySQLError
	return errors.As(err, &me) && (me.Number == 1213 || me.Number == 1205)
}

// next issues the next number of a sequence. The first number of a new sequence (e.g. the first
// transaction of a month) is created by whoever gets there first; requests arriving at the same
// moment can deadlock on the empty row, and MySQL rolls one of them back whole. That one asks again
// after a short random wait, by which time the row exists and the normal row lock serves it.
func (g *NumberGenerator) next(ctx context.Context, prefix string) (string, error) {
	var lastErr error
	for attempt := 1; attempt <= maxNumberAttempts; attempt++ {
		number, err := g.nextOnce(ctx, prefix)
		if err == nil {
			return number, nil
		}
		if !isDeadlock(err) {
			return "", err
		}
		lastErr = err
		wait := time.Duration(5+rand.Intn(20)*attempt) * time.Millisecond
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(wait):
		}
	}
	return "", lastErr
}

func (g *NumberGenerator) nextOnce(ctx context.Context, prefix string) (string, error) {
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
				// A deadlock rolled the whole transaction back: let next() ask again.
				if isDeadlock(err) {
					return err
				}
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
