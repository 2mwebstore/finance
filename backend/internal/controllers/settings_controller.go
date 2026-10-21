package controllers

import (
	"net/http"

	"github.com/cambodia-finance/income-expense/internal/audit"
	"github.com/cambodia-finance/income-expense/internal/services"
	"github.com/gin-gonic/gin"
)

type SettingsController struct {
	svc   *services.SettingsService
	audit *audit.Service
}

func NewSettingsController(svc *services.SettingsService, auditSvc *audit.Service) *SettingsController {
	return &SettingsController{svc: svc, audit: auditSvc}
}

func (sc *SettingsController) Get(c *gin.Context) {
	m, err := sc.svc.GetAll(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load settings"})
		return
	}
	c.JSON(http.StatusOK, m)
}

func (sc *SettingsController) Update(c *gin.Context) {
	var body map[string]string
	if !bindJSON(c, &body) {
		return
	}
	actorID, ok := getActorID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	for k, v := range body {
		if err := sc.svc.Set(c.Request.Context(), k, v, actorID); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if sc.audit != nil {
			sc.audit.Record(c.Request.Context(), audit.Event{
				ActorID: &actorID, Action: audit.ActionSettingsChange, EntityType: "setting",
				EntityID: k, Outcome: "success", IPAddress: c.ClientIP(),
				Details: map[string]interface{}{"key": k},
			})
		}
	}
	m, _ := sc.svc.GetAll(c.Request.Context())
	c.JSON(http.StatusOK, m)
}
