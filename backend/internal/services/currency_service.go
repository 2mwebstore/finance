package services

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/cambodia-finance/income-expense/internal/models"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Currencies are managed under Manage → Currency: USD is the base (rate 1, always on) and every
// other currency has a rate in units per 1 USD. Product prices, stock cost and inventory reports
// are in USD; transactions, sales, purchases and expenses can be in any active currency (inventory
// documents keep the rate of the day they were saved). Kept in the settings table: currency.list
// (JSON) and currency.default.

const (
	keyCurrencyList    = "currency.list"
	keyCurrencyDefault = "currency.default"
	keyCurrencyRate    = "currency.rate_khr" // older setting: the KHR rate before the list existed
	defaultKHRRate     = "4100"
	maxCurrencies      = 20
)

var (
	currencyCodeRule = regexp.MustCompile(`^[A-Z]{3}$`)
	maxCurrencyRate  = decimal.RequireFromString("100000000")
)

// Currency is one currency the app can use.
type Currency struct {
	Code     string          `json:"code"`
	Name     string          `json:"name"`
	Symbol   string          `json:"symbol"`
	Decimals int             `json:"decimals"`
	Rate     decimal.Decimal `json:"rate"` // units for 1 USD (USD: 1)
	Active   bool            `json:"active"`
	Base     bool            `json:"base"`
}

type CurrencySettings struct {
	Base            string     `json:"base"` // USD
	Currencies      []Currency `json:"currencies"`
	DefaultCurrency string     `json:"default_currency"` // preselected on forms
	UpdatedAt       *time.Time `json:"updated_at,omitempty"`
	UpdatedBy       string     `json:"updated_by,omitempty"`
}

type CurrencyRowInput struct {
	Code     string `json:"code"`
	Name     string `json:"name"`
	Symbol   string `json:"symbol"`
	Decimals int    `json:"decimals"`
	Rate     string `json:"rate"`
	Active   bool   `json:"active"`
}

type CurrencyInput struct {
	Currencies      []CurrencyRowInput `json:"currencies" binding:"required,min=1"`
	DefaultCurrency string             `json:"default_currency" binding:"required,len=3"`
}

type CurrencyService struct {
	db *gorm.DB
}

func NewCurrencyService(db *gorm.DB) *CurrencyService {
	return &CurrencyService{db: db}
}

func usdCurrency() Currency {
	return Currency{Code: models.CurrencyUSD, Name: "US Dollar", Symbol: "$", Decimals: 2, Rate: decimal.NewFromInt(1), Active: true, Base: true}
}

// loadCurrencies reads the list (USD first); before one is saved: USD and KHR (at the older
// KHR rate setting, or 4100).
func loadCurrencies(ctx context.Context, db *gorm.DB) ([]Currency, map[string]models.Setting) {
	rows := map[string]models.Setting{}
	var list []models.Setting
	db.WithContext(ctx).Where("`key` IN ?", []string{keyCurrencyList, keyCurrencyDefault, keyCurrencyRate}).Find(&list)
	for _, r := range list {
		rows[r.Key] = r
	}
	var out []Currency
	if raw, ok := rows[keyCurrencyList]; ok && json.Unmarshal([]byte(raw.Value), &out) == nil && len(out) > 0 {
		return withBase(out), rows
	}
	khr := decimal.RequireFromString(defaultKHRRate)
	if raw, ok := rows[keyCurrencyRate]; ok {
		if r, err := decimal.NewFromString(raw.Value); err == nil && r.IsPositive() {
			khr = r
		}
	}
	return []Currency{usdCurrency(), {Code: models.CurrencyKHR, Name: "Cambodian Riel", Symbol: "៛", Decimals: 0, Rate: khr, Active: true}}, rows
}

// withBase puts USD first, as the base at rate 1.
func withBase(list []Currency) []Currency {
	out := []Currency{usdCurrency()}
	for _, c := range list {
		if c.Code == models.CurrencyUSD {
			if c.Name != "" {
				out[0].Name = c.Name
			}
			if c.Symbol != "" {
				out[0].Symbol = c.Symbol
			}
			continue
		}
		c.Base = false
		out = append(out, c)
	}
	return out
}

func (s *CurrencyService) Get(ctx context.Context) (CurrencySettings, error) {
	list, rows := loadCurrencies(ctx, s.db)
	out := CurrencySettings{Base: models.CurrencyUSD, Currencies: list, DefaultCurrency: models.CurrencyUSD}
	if d, ok := rows[keyCurrencyDefault]; ok {
		for _, c := range list {
			if c.Code == d.Value && c.Active {
				out.DefaultCurrency = d.Value
			}
		}
	}
	latest, ok := rows[keyCurrencyList]
	if !ok {
		latest, ok = rows[keyCurrencyRate]
	}
	if ok {
		at := latest.UpdatedAt
		out.UpdatedAt = &at
		if latest.UpdatedBy != nil {
			var u models.User
			if s.db.WithContext(ctx).Select("username", "full_name").First(&u, "id = ?", *latest.UpdatedBy).Error == nil {
				out.UpdatedBy = u.FullName
				if out.UpdatedBy == "" {
					out.UpdatedBy = u.Username
				}
			}
		}
	}
	return out, nil
}

// currencyUses counts the records in a currency (a used currency can be turned off, not removed).
func (s *CurrencyService) currencyUses(ctx context.Context, code string) int64 {
	var total int64
	for _, table := range []string{"transactions", "sales", "purchases", "inventory_expenses", "opening_balances", "cash_reconciliations"} {
		if !s.db.Migrator().HasTable(table) || !s.db.Migrator().HasColumn(table, "currency") {
			continue
		}
		var n int64
		s.db.WithContext(ctx).Table(table).Where("currency = ?", code).Count(&n)
		total += n
	}
	return total
}

func (s *CurrencyService) Save(ctx context.Context, in CurrencyInput, actor uuid.UUID) (CurrencySettings, error) {
	if len(in.Currencies) > maxCurrencies {
		return CurrencySettings{}, fieldError("currencies", fmt.Sprintf("at most %d currencies", maxCurrencies))
	}
	seen := map[string]bool{}
	list := []Currency{}
	for i, row := range in.Currencies {
		code := strings.ToUpper(strings.TrimSpace(row.Code))
		line := fmt.Sprintf("row %d", i+1)
		if !currencyCodeRule.MatchString(code) {
			return CurrencySettings{}, fieldError("currencies", line+": the code must be 3 letters, like THB")
		}
		if seen[code] {
			return CurrencySettings{}, fieldError("currencies", code+" is listed twice")
		}
		seen[code] = true
		if code == models.CurrencyUSD {
			continue // the base: always on, rate 1
		}
		name, symbol := strings.TrimSpace(row.Name), strings.TrimSpace(row.Symbol)
		if name == "" || len(name) > 64 {
			return CurrencySettings{}, fieldError("currencies", code+": enter a name (up to 64 characters)")
		}
		if symbol == "" || len([]rune(symbol)) > 6 {
			return CurrencySettings{}, fieldError("currencies", code+": enter a symbol (up to 6 characters)")
		}
		if row.Decimals < 0 || row.Decimals > 4 {
			return CurrencySettings{}, fieldError("currencies", code+": decimals must be 0 to 4")
		}
		rate, err := decimal.NewFromString(strings.TrimSpace(row.Rate))
		if err != nil || !rate.IsPositive() || rate.Exponent() < -6 || rate.GreaterThan(maxCurrencyRate) {
			return CurrencySettings{}, fieldError("currencies", code+": enter how many "+code+" make 1 USD")
		}
		list = append(list, Currency{Code: code, Name: name, Symbol: symbol, Decimals: row.Decimals, Rate: rate, Active: row.Active})
	}
	current, _ := loadCurrencies(ctx, s.db)
	for _, c := range current {
		if !seen[c.Code] && !c.Base {
			if n := s.currencyUses(ctx, c.Code); n > 0 {
				return CurrencySettings{}, fieldError("currencies", fmt.Sprintf("%s is used by %d records; turn it off instead of removing it", c.Code, n))
			}
		}
	}
	list = withBase(list)
	defaultOK := false
	for _, c := range list {
		defaultOK = defaultOK || (c.Code == in.DefaultCurrency && c.Active)
	}
	if !defaultOK {
		return CurrencySettings{}, fieldError("default_currency", "the default currency must be one of the active currencies")
	}
	raw, _ := json.Marshal(list)
	now := time.Now()
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for key, value := range map[string]string{keyCurrencyList: string(raw), keyCurrencyDefault: in.DefaultCurrency} {
			if err := tx.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "key"}},
				DoUpdates: clause.AssignmentColumns([]string{"value", "updated_at", "updated_by"}),
			}).Create(&models.Setting{Key: key, Value: value, UpdatedAt: now, UpdatedBy: &actor}).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return CurrencySettings{}, err
	}
	return s.Get(ctx)
}

// currencyRate checks that code is an active currency and returns its rate (units per 1 USD).
func currencyRate(ctx context.Context, db *gorm.DB, code string) (decimal.Decimal, error) {
	list, _ := loadCurrencies(ctx, db)
	for _, c := range list {
		if c.Code == code {
			if !c.Active {
				return decimal.Zero, fieldError("currency", code+" is turned off under Manage → Currency")
			}
			return c.Rate, nil
		}
	}
	return decimal.Zero, fieldError("currency", "choose one of the currencies set up under Manage → Currency")
}

// validateCurrency checks that code is an active currency.
func validateCurrency(ctx context.Context, db *gorm.DB, code string) error {
	_, err := currencyRate(ctx, db, code)
	return err
}

// toBase converts an amount in currency to USD at rate (units of that currency per 1 USD).
func toBase(amount decimal.Decimal, currency string, rate decimal.Decimal) decimal.Decimal {
	if currency != models.CurrencyUSD && rate.IsPositive() {
		return amount.Div(rate).Round(4)
	}
	return amount
}
