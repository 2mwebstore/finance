package controllers

import (
	"net/http"

	"github.com/cambodia-finance/income-expense/internal/notifications"
	"github.com/gin-gonic/gin"
)

type NotificationController struct {
	tg *notifications.TelegramNotifier
}

func NewNotificationController(tg *notifications.TelegramNotifier) *NotificationController {
	return &NotificationController{tg: tg}
}

func (nc *NotificationController) Status(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"telegram_enabled": nc.tg.Enabled(),
		"message":          "Telegram is configured via TELEGRAM_BOT_TOKEN and TELEGRAM_CHAT_ID env vars only",
	})
}

func (nc *NotificationController) Test(c *gin.Context) {
	if !nc.tg.Enabled() {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "telegram not configured (set TELEGRAM_BOT_TOKEN and TELEGRAM_CHAT_ID)",
		})
		return
	}
	nc.tg.NotifyAsync("Income-Expense system: test notification OK")
	c.JSON(http.StatusOK, gin.H{"message": "test notification queued"})
}
