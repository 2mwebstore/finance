package controllers

import (
	"net/http"

	"github.com/cambodia-finance/income-expense/internal/accounting"
	"github.com/cambodia-finance/income-expense/internal/audit"
	"github.com/cambodia-finance/income-expense/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type ReportController struct {
	svc   *services.ReportService
	audit *audit.Service
}

func NewReportController(svc *services.ReportService, auditSvc *audit.Service) *ReportController {
	return &ReportController{svc: svc, audit: auditSvc}
}

func (rc *ReportController) parseBranch(c *gin.Context) (*uuid.UUID, error) {
	if v := c.Query("branch_id"); v != "" {
		id, err := uuid.Parse(v)
		if err != nil {
			return nil, err
		}
		return &id, nil
	}
	return nil, nil
}

func (rc *ReportController) Dashboard(c *gin.Context) {
	actorID, ok := getActorID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	branchID, err := rc.parseBranch(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid branch_id"})
		return
	}
	period := c.DefaultQuery("period", "this_month")
	sum, err := rc.svc.Dashboard(c.Request.Context(), actorID, period, c.Query("date_from"), c.Query("date_to"), branchID)
	if err != nil {
		writeTxnError(c, err)
		return
	}

	recent := make([]gin.H, 0, len(sum.Recent))
	for i := range sum.Recent {
		recent = append(recent, txnJSON(&sum.Recent[i]))
	}
	daily, _ := rc.svc.DailySummary(c.Request.Context(), actorID, period, c.Query("date_from"), c.Query("date_to"), branchID)

	c.JSON(http.StatusOK, gin.H{
		"period":              sum.Period,
		"date_from":           sum.DateFrom,
		"date_to":             sum.DateTo,
		"income_usd":          sum.IncomeUSD.StringFixed(4),
		"expense_usd":         sum.ExpenseUSD.StringFixed(4),
		"net_usd":             sum.NetUSD.StringFixed(4),
		"income_khr":          sum.IncomeKHR.StringFixed(4),
		"expense_khr":         sum.ExpenseKHR.StringFixed(4),
		"net_khr":             sum.NetKHR.StringFixed(4),
		"income_usd_display":  accounting.FormatDisplay("USD", sum.IncomeUSD),
		"expense_usd_display": accounting.FormatDisplay("USD", sum.ExpenseUSD),
		"net_usd_display":     accounting.FormatDisplay("USD", sum.NetUSD),
		"income_khr_display":  accounting.FormatDisplay("KHR", sum.IncomeKHR),
		"expense_khr_display": accounting.FormatDisplay("KHR", sum.ExpenseKHR),
		"net_khr_display":     accounting.FormatDisplay("KHR", sum.NetKHR),
		"transaction_count":   sum.TxnCount,
		"by_currency":         sum.ByCurrency,
		"daily":               daily,
		"income_by_category":  sum.IncomeByCat,
		"expense_by_category": sum.ExpenseByCat,
		"by_branch":           sum.ByBranch,
		"recent_transactions": recent,
	})
}

func (rc *ReportController) Daily(c *gin.Context) {
	actorID, ok := getActorID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	branchID, err := rc.parseBranch(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid branch_id"})
		return
	}
	period := c.DefaultQuery("period", "this_month")
	rows, err := rc.svc.DailySummary(c.Request.Context(), actorID, period, c.Query("date_from"), c.Query("date_to"), branchID)
	if err != nil {
		writeTxnError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"period": period,
		"items":  rows,
	})
}

func (rc *ReportController) Income(c *gin.Context) {
	rc.txnReport(c, "income")
}

func (rc *ReportController) Expenses(c *gin.Context) {
	rc.txnReport(c, "expense")
}

func (rc *ReportController) Transactions(c *gin.Context) {
	rc.txnReport(c, "")
}

func (rc *ReportController) txnReport(c *gin.Context, typ string) {
	actorID, ok := getActorID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	branchID, err := rc.parseBranch(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid branch_id"})
		return
	}
	period := c.DefaultQuery("period", "this_month")
	dr, err := accounting.ParsePeriod(period, c.Query("date_from"), c.Query("date_to"), "Asia/Phnom_Penh")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Reuse transaction list via report export path conceptually — use dashboard totals + list
	sum, err := rc.svc.Dashboard(c.Request.Context(), actorID, period, c.Query("date_from"), c.Query("date_to"), branchID)
	if err != nil {
		writeTxnError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"report_type":         map[string]string{"": "transactions", "income": "income", "expense": "expenses"}[typ],
		"period":              sum.Period,
		"date_from":           sum.DateFrom,
		"date_to":             sum.DateTo,
		"income_usd":          sum.IncomeUSD.StringFixed(4),
		"expense_usd":         sum.ExpenseUSD.StringFixed(4),
		"net_usd":             sum.NetUSD.StringFixed(4),
		"income_khr":          sum.IncomeKHR.StringFixed(4),
		"expense_khr":         sum.ExpenseKHR.StringFixed(4),
		"net_khr":             sum.NetKHR.StringFixed(4),
		"transaction_count":   sum.TxnCount,
		"income_by_category":  sum.IncomeByCat,
		"expense_by_category": sum.ExpenseByCat,
		"by_branch":           sum.ByBranch,
		"range_start":         dr.Start,
		"range_end":           dr.End,
	})
}

func (rc *ReportController) Branches(c *gin.Context) {
	actorID, ok := getActorID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	period := c.DefaultQuery("period", "this_month")
	sum, err := rc.svc.Dashboard(c.Request.Context(), actorID, period, c.Query("date_from"), c.Query("date_to"), nil)
	if err != nil {
		writeTxnError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"period":    sum.Period,
		"date_from": sum.DateFrom,
		"date_to":   sum.DateTo,
		"branches":  sum.ByBranch,
	})
}

func (rc *ReportController) Export(c *gin.Context) {
	actorID, ok := getActorID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	branchID, err := rc.parseBranch(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid branch_id"})
		return
	}
	period := c.DefaultQuery("period", "this_month")
	txnType := c.Query("type")
	csv, err := rc.svc.ExportCSV(c.Request.Context(), actorID, period, c.Query("date_from"), c.Query("date_to"), branchID, txnType)
	if err != nil {
		writeTxnError(c, err)
		return
	}
	if rc.audit != nil {
		rc.audit.Record(c.Request.Context(), audit.Event{
			ActorID: &actorID, Action: audit.ActionReportExport, EntityType: "report",
			Outcome: "success", IPAddress: c.ClientIP(),
			Details: map[string]interface{}{"period": period, "type": txnType},
		})
	}
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", "attachment; filename=report.csv")
	c.String(http.StatusOK, csv)
}
