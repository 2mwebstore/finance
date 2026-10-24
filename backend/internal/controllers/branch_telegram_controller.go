package controllers

import (
	"errors"
	"net/http"

	"github.com/cambodia-finance/income-expense/internal/audit"
	"github.com/cambodia-finance/income-expense/internal/models"
	"github.com/cambodia-finance/income-expense/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// BranchTelegramController: each branch's Telegram chat for transaction updates. People who can
// edit branches set it up for the branches they see.
type BranchTelegramController struct {
	svc   *services.BranchTelegramService
	db    *gorm.DB
	audit *audit.Service
}

func NewBranchTelegramController(svc *services.BranchTelegramService, db *gorm.DB, auditSvc *audit.Service) *BranchTelegramController {
	return &BranchTelegramController{svc: svc, db: db, audit: auditSvc}
}

// branch loads the branch from the URL when it is in the actor's branches.
func (bc *BranchTelegramController) branch(c *gin.Context) (*models.Branch, *accessScope, bool) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return nil, nil, false
	}
	scope, ok := loadScope(c, bc.db)
	if !ok {
		return nil, nil, false
	}
	var branch models.Branch
	if !scope.HasBranch(id) || bc.db.First(&branch, "id = ?", id).Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "branch not found"})
		return nil, nil, false
	}
	return &branch, scope, true
}

func (bc *BranchTelegramController) Get(c *gin.Context) {
	branch, _, ok := bc.branch(c)
	if !ok {
		return
	}
	settings, err := bc.svc.Get(c.Request.Context(), branch.ID)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"settings": settings, "events": services.TelegramEvents})
}

func (bc *BranchTelegramController) Save(c *gin.Context) {
	branch, scope, ok := bc.branch(c)
	if !ok {
		return
	}
	var in services.BranchTelegramInput
	if !bindJSON(c, &in) {
		return
	}
	settings, err := bc.svc.Save(c.Request.Context(), branch.ID, in, scope.ActorID)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	if bc.audit != nil {
		// Never log the token itself.
		bid := branch.ID
		bc.audit.Record(c.Request.Context(), audit.Event{
			ActorID: &scope.ActorID, Action: "branch.telegram", EntityType: "branch", EntityID: branch.ID.String(),
			BranchID: &bid, Outcome: "success", IPAddress: c.ClientIP(), UserAgent: c.Request.UserAgent(),
			Details: map[string]interface{}{
				"enabled": settings.Enabled, "chat_id": settings.ChatID, "events": settings.Events,
				"token_changed": in.BotToken != nil && *in.BotToken != "" || in.ClearToken,
			},
		})
	}
	c.JSON(http.StatusOK, gin.H{"settings": settings, "events": services.TelegramEvents})
}

func (bc *BranchTelegramController) Test(c *gin.Context) {
	branch, _, ok := bc.branch(c)
	if !ok {
		return
	}
	if err := bc.svc.SendTest(c.Request.Context(), branch); err != nil {
		var fe *services.FieldError
		if errors.As(err, &fe) {
			writeServiceError(c, err)
			return
		}
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "test message sent"})
}
