package services

import (
	"context"
	"encoding/csv"
	"strings"
	"time"

	"github.com/cambodia-finance/income-expense/internal/accounting"
	"github.com/cambodia-finance/income-expense/internal/middleware"
	"github.com/cambodia-finance/income-expense/internal/models"
	"github.com/cambodia-finance/income-expense/internal/richtext"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type ReportService struct {
	db       *gorm.DB
	timezone string
}

func NewReportService(db *gorm.DB, timezone string) *ReportService {
	return &ReportService{db: db, timezone: timezone}
}

// CurrencyFlow is income, expense and net in one currency (by_currency maps cover every
// currency, beside the older USD/KHR fields).
type CurrencyFlow struct {
	Income  string `json:"income"`
	Expense string `json:"expense"`
	Net     string `json:"net"`
}

type flowAcc struct{ income, expense decimal.Decimal }

func addFlow(m map[string]*flowAcc, currency, typ string, total decimal.Decimal) {
	f := m[currency]
	if f == nil {
		f = &flowAcc{}
		m[currency] = f
	}
	if typ == models.TxnTypeIncome {
		f.income = f.income.Add(total)
	} else {
		f.expense = f.expense.Add(total)
	}
}

func flows(m map[string]*flowAcc) map[string]CurrencyFlow {
	out := make(map[string]CurrencyFlow, len(m))
	for cur, f := range m {
		out[cur] = CurrencyFlow{Income: f.income.StringFixed(4), Expense: f.expense.StringFixed(4), Net: f.income.Sub(f.expense).StringFixed(4)}
	}
	return out
}

type DashboardSummary struct {
	Period       string                  `json:"period"`
	DateFrom     string                  `json:"date_from"`
	DateTo       string                  `json:"date_to"` // inclusive display
	IncomeUSD    decimal.Decimal         `json:"-"`
	ExpenseUSD   decimal.Decimal         `json:"-"`
	NetUSD       decimal.Decimal         `json:"-"`
	IncomeKHR    decimal.Decimal         `json:"-"`
	ExpenseKHR   decimal.Decimal         `json:"-"`
	NetKHR       decimal.Decimal         `json:"-"`
	TxnCount     int64                   `json:"transaction_count"`
	ByCurrency   map[string]CurrencyFlow `json:"by_currency"`
	IncomeByCat  []CatTotal              `json:"income_by_category"`
	ExpenseByCat []CatTotal              `json:"expense_by_category"`
	ByBranch     []BranchTotal           `json:"by_branch"`
	Recent       []models.Transaction    `json:"-"`
}

type CatTotal struct {
	CategoryID   string          `json:"category_id"`
	CategoryCode string          `json:"category_code"`
	CategoryName string          `json:"category_name"`
	Currency     string          `json:"currency"`
	Total        decimal.Decimal `json:"-"`
	TotalStr     string          `json:"total"`
}

type BranchTotal struct {
	BranchID      string          `json:"branch_id"`
	BranchCode    string          `json:"branch_code"`
	BranchName    string          `json:"branch_name"`
	IncomeUSD     decimal.Decimal `json:"-"`
	ExpenseUSD    decimal.Decimal `json:"-"`
	IncomeKHR     decimal.Decimal `json:"-"`
	ExpenseKHR    decimal.Decimal `json:"-"`
	IncomeUSDStr  string          `json:"income_usd"`
	ExpenseUSDStr string          `json:"expense_usd"`
	NetUSDStr     string          `json:"net_usd"`
	IncomeKHRStr  string          `json:"income_khr"`
	ExpenseKHRStr string          `json:"expense_khr"`
	NetKHRStr     string          `json:"net_khr"`
	// Every currency.
	ByCurrency map[string]CurrencyFlow `json:"by_currency"`
	flows      map[string]*flowAcc
}

type DailyRow struct {
	Date          string          `json:"date"`
	IncomeUSD     decimal.Decimal `json:"-"`
	ExpenseUSD    decimal.Decimal `json:"-"`
	IncomeKHR     decimal.Decimal `json:"-"`
	ExpenseKHR    decimal.Decimal `json:"-"`
	IncomeUSDStr  string          `json:"income_usd"`
	ExpenseUSDStr string          `json:"expense_usd"`
	NetUSDStr     string          `json:"net_usd"`
	IncomeKHRStr  string          `json:"income_khr"`
	ExpenseKHRStr string          `json:"expense_khr"`
	NetKHRStr     string          `json:"net_khr"`
	Count         int64           `json:"count"`
	// Every currency.
	ByCurrency map[string]CurrencyFlow `json:"by_currency"`
	flows      map[string]*flowAcc
}

func (s *ReportService) scopedQuery(ctx context.Context, actorID uuid.UUID, branchID *uuid.UUID) (*gorm.DB, error) {
	allowed, all, err := middleware.GetAuthorizedBranchIDs(s.db, actorID)
	if err != nil {
		return nil, err
	}
	q := s.db.WithContext(ctx).Model(&models.Transaction{}).Where("status = ?", models.TxnStatusApproved)
	if !all {
		if len(allowed) == 0 {
			q = q.Where("1 = 0")
		} else {
			q = q.Where("branch_id IN ?", allowed)
		}
	}
	if branchID != nil {
		if !all {
			ok := false
			for _, a := range allowed {
				if a == *branchID {
					ok = true
					break
				}
			}
			if !ok {
				return nil, ErrBranchAccessDenied
			}
		}
		q = q.Where("branch_id = ?", *branchID)
	}
	return q, nil
}

func (s *ReportService) Dashboard(ctx context.Context, actorID uuid.UUID, period, dateFrom, dateTo string, branchID *uuid.UUID) (*DashboardSummary, error) {
	dr, err := accounting.ParsePeriod(period, dateFrom, dateTo, s.timezone)
	if err != nil {
		return nil, err
	}

	q, err := s.scopedQuery(ctx, actorID, branchID)
	if err != nil {
		return nil, err
	}
	q = q.Where("transaction_date >= ? AND transaction_date < ?", dr.Start, dr.End)

	// Currency totals
	type curRow struct {
		Currency string
		Type     string
		Total    decimal.Decimal
	}
	var curRows []curRow
	if err := q.Session(&gorm.Session{}).
		Select("currency, type, COALESCE(SUM(amount),0) as total").
		Group("currency, type").Scan(&curRows).Error; err != nil {
		return nil, err
	}

	sum := &DashboardSummary{
		Period:    dr.Label,
		DateFrom:  dr.Start.Format("2006-01-02"),
		DateTo:    dr.End.Add(-time.Second).Format("2006-01-02"),
		IncomeUSD: accounting.Zero(), ExpenseUSD: accounting.Zero(),
		IncomeKHR: accounting.Zero(), ExpenseKHR: accounting.Zero(),
	}
	all := map[string]*flowAcc{}
	for _, r := range curRows {
		addFlow(all, r.Currency, r.Type, r.Total)
		if r.Currency == models.CurrencyUSD && r.Type == models.TxnTypeIncome {
			sum.IncomeUSD = r.Total
		} else if r.Currency == models.CurrencyUSD && r.Type == models.TxnTypeExpense {
			sum.ExpenseUSD = r.Total
		} else if r.Currency == models.CurrencyKHR && r.Type == models.TxnTypeIncome {
			sum.IncomeKHR = r.Total
		} else if r.Currency == models.CurrencyKHR && r.Type == models.TxnTypeExpense {
			sum.ExpenseKHR = r.Total
		}
	}
	sum.NetUSD = sum.IncomeUSD.Sub(sum.ExpenseUSD)
	sum.NetKHR = sum.IncomeKHR.Sub(sum.ExpenseKHR)
	sum.ByCurrency = flows(all)

	if err := q.Session(&gorm.Session{}).Count(&sum.TxnCount).Error; err != nil {
		return nil, err
	}

	// Category breakdown (scoped)
	catRows, _ := s.categoryTotals(ctx, actorID, branchID, dr)
	for _, r := range catRows {
		ct := CatTotal{
			CategoryID: r.CategoryID, CategoryCode: r.Code, CategoryName: r.Name,
			Currency: r.Currency, Total: r.Total, TotalStr: r.Total.StringFixed(4),
		}
		if r.Type == models.TxnTypeIncome {
			sum.IncomeByCat = append(sum.IncomeByCat, ct)
		} else {
			sum.ExpenseByCat = append(sum.ExpenseByCat, ct)
		}
	}

	sum.ByBranch, _ = s.branchTotals(ctx, actorID, branchID, dr)

	// Recent transactions
	rq, _ := s.scopedQuery(ctx, actorID, branchID)
	var recent []models.Transaction
	_ = rq.Where("transaction_date >= ? AND transaction_date < ?", dr.Start, dr.End).
		Preload("Branch").Preload("Category").Preload("PaymentMethod").
		Order("transaction_date DESC, created_at DESC").Limit(10).Find(&recent)
	sum.Recent = recent

	return sum, nil
}

type catAgg struct {
	CategoryID string
	Code       string
	Name       string
	Type       string
	Currency   string
	Total      decimal.Decimal
}

func (s *ReportService) categoryTotals(ctx context.Context, actorID uuid.UUID, branchID *uuid.UUID, dr accounting.DateRange) ([]catAgg, error) {
	q, err := s.scopedQuery(ctx, actorID, branchID)
	if err != nil {
		return nil, err
	}
	var rows []catAgg
	err = q.Where("transaction_date >= ? AND transaction_date < ?", dr.Start, dr.End).
		Select("COALESCE(transactions.category_id, '') as category_id, COALESCE(categories.code, '') as code, COALESCE(categories.name, 'Uncategorized') as name, transactions.type as type, transactions.currency as currency, COALESCE(SUM(transactions.amount),0) as total").
		Joins("LEFT JOIN categories ON categories.id = transactions.category_id").
		Group("transactions.category_id, categories.code, categories.name, transactions.type, transactions.currency").
		Scan(&rows).Error
	return rows, err
}

func (s *ReportService) branchTotals(ctx context.Context, actorID uuid.UUID, branchID *uuid.UUID, dr accounting.DateRange) ([]BranchTotal, error) {
	q, err := s.scopedQuery(ctx, actorID, branchID)
	if err != nil {
		return nil, err
	}
	type row struct {
		BranchID string
		Code     string
		Name     string
		Currency string
		Type     string
		Total    decimal.Decimal
	}
	var rows []row
	err = q.Where("transaction_date >= ? AND transaction_date < ?", dr.Start, dr.End).
		Select("transactions.branch_id as branch_id, branches.code as code, branches.name as name, transactions.currency as currency, transactions.type as type, COALESCE(SUM(transactions.amount),0) as total").
		Joins("LEFT JOIN branches ON branches.id = transactions.branch_id").
		Group("transactions.branch_id, branches.code, branches.name, transactions.currency, transactions.type").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	m := map[string]*BranchTotal{}
	for _, r := range rows {
		bt, ok := m[r.BranchID]
		if !ok {
			bt = &BranchTotal{
				BranchID: r.BranchID, BranchCode: r.Code, BranchName: r.Name,
				IncomeUSD: accounting.Zero(), ExpenseUSD: accounting.Zero(),
				IncomeKHR: accounting.Zero(), ExpenseKHR: accounting.Zero(),
				flows: map[string]*flowAcc{},
			}
			m[r.BranchID] = bt
		}
		addFlow(bt.flows, r.Currency, r.Type, r.Total)
		if r.Currency == models.CurrencyUSD && r.Type == models.TxnTypeIncome {
			bt.IncomeUSD = r.Total
		} else if r.Currency == models.CurrencyUSD && r.Type == models.TxnTypeExpense {
			bt.ExpenseUSD = r.Total
		} else if r.Currency == models.CurrencyKHR && r.Type == models.TxnTypeIncome {
			bt.IncomeKHR = r.Total
		} else if r.Currency == models.CurrencyKHR && r.Type == models.TxnTypeExpense {
			bt.ExpenseKHR = r.Total
		}
	}
	out := make([]BranchTotal, 0, len(m))
	for _, bt := range m {
		bt.IncomeUSDStr = bt.IncomeUSD.StringFixed(4)
		bt.ExpenseUSDStr = bt.ExpenseUSD.StringFixed(4)
		bt.NetUSDStr = bt.IncomeUSD.Sub(bt.ExpenseUSD).StringFixed(4)
		bt.IncomeKHRStr = bt.IncomeKHR.StringFixed(4)
		bt.ExpenseKHRStr = bt.ExpenseKHR.StringFixed(4)
		bt.NetKHRStr = bt.IncomeKHR.Sub(bt.ExpenseKHR).StringFixed(4)
		bt.ByCurrency = flows(bt.flows)
		out = append(out, *bt)
	}
	return out, nil
}

func (s *ReportService) DailySummary(ctx context.Context, actorID uuid.UUID, period, dateFrom, dateTo string, branchID *uuid.UUID) ([]DailyRow, error) {
	dr, err := accounting.ParsePeriod(period, dateFrom, dateTo, s.timezone)
	if err != nil {
		return nil, err
	}
	q, err := s.scopedQuery(ctx, actorID, branchID)
	if err != nil {
		return nil, err
	}
	type row struct {
		Day      string
		Currency string
		Type     string
		Total    decimal.Decimal
		Cnt      int64
	}
	var rows []row
	err = q.Where("transaction_date >= ? AND transaction_date < ?", dr.Start, dr.End).
		Select("DATE(transaction_date) as day, currency, type, COALESCE(SUM(amount),0) as total, COUNT(*) as cnt").
		Group("DATE(transaction_date), currency, type").
		Order("day ASC").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	m := map[string]*DailyRow{}
	for _, r := range rows {
		d, ok := m[r.Day]
		if !ok {
			d = &DailyRow{
				Date:      r.Day,
				IncomeUSD: accounting.Zero(), ExpenseUSD: accounting.Zero(),
				IncomeKHR: accounting.Zero(), ExpenseKHR: accounting.Zero(),
				flows: map[string]*flowAcc{},
			}
			m[r.Day] = d
		}
		d.Count += r.Cnt
		addFlow(d.flows, r.Currency, r.Type, r.Total)
		if r.Currency == models.CurrencyUSD && r.Type == models.TxnTypeIncome {
			d.IncomeUSD = r.Total
		} else if r.Currency == models.CurrencyUSD && r.Type == models.TxnTypeExpense {
			d.ExpenseUSD = r.Total
		} else if r.Currency == models.CurrencyKHR && r.Type == models.TxnTypeIncome {
			d.IncomeKHR = r.Total
		} else if r.Currency == models.CurrencyKHR && r.Type == models.TxnTypeExpense {
			d.ExpenseKHR = r.Total
		}
	}
	out := make([]DailyRow, 0, len(m))
	for _, d := range m {
		d.IncomeUSDStr = d.IncomeUSD.StringFixed(4)
		d.ExpenseUSDStr = d.ExpenseUSD.StringFixed(4)
		d.NetUSDStr = d.IncomeUSD.Sub(d.ExpenseUSD).StringFixed(4)
		d.IncomeKHRStr = d.IncomeKHR.StringFixed(4)
		d.ExpenseKHRStr = d.ExpenseKHR.StringFixed(4)
		d.NetKHRStr = d.IncomeKHR.Sub(d.ExpenseKHR).StringFixed(4)
		d.ByCurrency = flows(d.flows)
		out = append(out, *d)
	}
	// sort by date
	for i := 0; i < len(out); i++ {
		for j := i + 1; j < len(out); j++ {
			if out[j].Date < out[i].Date {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	return out, nil
}

// ExportCSV builds a CSV of transactions for the period (authorized only).
func (s *ReportService) ExportCSV(ctx context.Context, actorID uuid.UUID, period, dateFrom, dateTo string, branchID *uuid.UUID, txnType string) (string, error) {
	dr, err := accounting.ParsePeriod(period, dateFrom, dateTo, s.timezone)
	if err != nil {
		return "", err
	}
	q, err := s.scopedQuery(ctx, actorID, branchID)
	if err != nil {
		return "", err
	}
	q = q.Where("transaction_date >= ? AND transaction_date < ?", dr.Start, dr.End)
	if txnType != "" {
		q = q.Where("type = ?", txnType)
	}
	var items []models.Transaction
	if err := q.Preload("Branch").Preload("Category").Preload("PaymentMethod").
		Order("transaction_date ASC, created_at ASC").
		Limit(10000).Find(&items).Error; err != nil {
		return "", err
	}

	var b strings.Builder
	w := csv.NewWriter(&b)
	_ = w.Write([]string{"transaction_number", "type", "status", "date", "branch_code", "category", "payment_method", "currency", "amount", "description", "counterparty"})
	for _, t := range items {
		branchCode, catName, pmName := "", "", ""
		if t.Branch != nil {
			branchCode = t.Branch.Code
		}
		if t.Category != nil {
			catName = t.Category.Name
		}
		if t.PaymentMethod != nil {
			pmName = t.PaymentMethod.Name
		}
		_ = w.Write([]string{
			t.TransactionNumber, t.Type, t.Status,
			t.TransactionDate.In(accounting.LoadLocation(s.timezone)).Format("2006-01-02"),
			csvText(branchCode), csvText(catName), csvText(pmName), t.Currency,
			t.Amount.StringFixed(4),
			csvText(richtext.ToText(t.Description)), csvText(t.Counterparty),
		})
	}
	w.Flush()
	return b.String(), w.Error()
}

// csvText keeps typed text from running as a spreadsheet formula when the export is opened
// in Excel or Sheets (cells starting with = + - @ are formulas there).
func csvText(v string) string {
	if v != "" && strings.ContainsRune("=+-@\t\r", rune(v[0])) {
		return "'" + v
	}
	return v
}
