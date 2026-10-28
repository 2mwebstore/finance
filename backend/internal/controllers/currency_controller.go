package controllers

import (
	"net/http"

	"github.com/cambodia-finance/income-expense/internal/audit"
	"github.com/cambodia-finance/income-expense/internal/services"
	"github.com/gin-gonic/gin"
)

// CurrencyController: the currencies (USD base, others with a rate per 1 USD) and the default
// currency.
type CurrencyController struct {
	svc   *services.CurrencyService
	audit *audit.Service
}

func NewCurrencyController(svc *services.CurrencyService, auditSvc *audit.Service) *CurrencyController {
	return &CurrencyController{svc: svc, audit: auditSvc}
}

// Get is open to every signed-in user: forms convert prices with the rate.
func (cc *CurrencyController) Get(c *gin.Context) {
	settings, err := cc.svc.Get(c.Request.Context())
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, settings)
}

func (cc *CurrencyController) Update(c *gin.Context) {
	actorID, ok := getActorID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	var in services.CurrencyInput
	if !bindJSON(c, &in) {
		return
	}
	settings, err := cc.svc.Save(c.Request.Context(), in, actorID)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	if cc.audit != nil {
		cc.audit.Record(c.Request.Context(), audit.Event{
			ActorID: &actorID, Action: audit.ActionSettingsChange, EntityType: "setting", EntityID: "currency",
			Outcome: "success", IPAddress: c.ClientIP(), UserAgent: c.Request.UserAgent(),
			Details: map[string]interface{}{"currencies": currencySummary(settings.Currencies), "default_currency": settings.DefaultCurrency},
		})
	}
	c.JSON(http.StatusOK, settings)
}

// currencySummary: "KHR 4100 on, THB 36 off" for the audit log.
func currencySummary(list []services.Currency) []string {
	out := make([]string, 0, len(list))
	for _, c := range list {
		state := "on"
		if !c.Active {
			state = "off"
		}
		out = append(out, c.Code+" "+c.Rate.String()+" "+state)
	}
	return out
}
