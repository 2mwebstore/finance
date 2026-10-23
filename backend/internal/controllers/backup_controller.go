package controllers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/cambodia-finance/income-expense/internal/audit"
	"github.com/cambodia-finance/income-expense/internal/backup"
	"github.com/cambodia-finance/income-expense/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// BackupController: database backups (made on a schedule or on demand, kept on the server for a
// set number of days, sent to Telegram), uploads, and restores.
type BackupController struct {
	svc   *backup.Service
	db    *gorm.DB
	audit *audit.Service
}

func NewBackupController(svc *backup.Service, db *gorm.DB, auditSvc *audit.Service) *BackupController {
	return &BackupController{svc: svc, db: db, audit: auditSvc}
}

// actor returns the signed-in user's id and a label for "created by".
func (bc *BackupController) actor(c *gin.Context) (uuid.UUID, string, bool) {
	id, ok := getActorID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return uuid.Nil, "", false
	}
	var user models.User
	label := id.String()
	if err := bc.db.Select("username", "full_name").First(&user, "id = ?", id).Error; err == nil {
		label = user.FullName
		if label == "" {
			label = user.Username
		}
	}
	return id, label, true
}

func (bc *BackupController) record(c *gin.Context, actorID uuid.UUID, action, entity string, details map[string]interface{}) {
	if bc.audit == nil {
		return
	}
	bc.audit.Record(c.Request.Context(), audit.Event{
		ActorID: &actorID, Action: action, EntityType: "backup", EntityID: entity,
		Outcome: "success", IPAddress: c.ClientIP(), UserAgent: c.Request.UserAgent(), Details: details,
	})
}

func writeBackupError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, backup.ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	case errors.Is(err, backup.ErrBusy):
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
	case writeInternalError(c, err):
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	}
}

// List: the kept backups and the backup settings.
func (bc *BackupController) List(c *gin.Context) {
	items, err := bc.svc.List()
	if err != nil {
		writeBackupError(c, err)
		return
	}
	settings, err := bc.svc.Settings(c.Request.Context())
	if err != nil {
		writeBackupError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "settings": settings})
}

// Create makes a backup now ("Backup now").
func (bc *BackupController) Create(c *gin.Context) {
	actorID, label, ok := bc.actor(c)
	if !ok {
		return
	}
	item, err := bc.svc.Run(c.Request.Context(), backup.TriggerManual, label)
	if err != nil {
		writeBackupError(c, err)
		return
	}
	bc.record(c, actorID, "backup.create", item.Name, map[string]interface{}{"size": item.Size, "telegram": item.Telegram})
	c.JSON(http.StatusCreated, item)
}

func (bc *BackupController) UpdateSettings(c *gin.Context) {
	actorID, _, ok := bc.actor(c)
	if !ok {
		return
	}
	var in backup.SettingsInput
	if !bindJSON(c, &in) {
		return
	}
	settings, err := bc.svc.SaveSettings(c.Request.Context(), in, actorID)
	if err != nil {
		field := map[error]string{backup.ErrBadTime: "time", backup.ErrBadKeepDays: "keep_days"}[err]
		if field != "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "fields": gin.H{field: err.Error()}})
			return
		}
		writeBackupError(c, err)
		return
	}
	// Never log the token itself.
	bc.record(c, actorID, "backup.settings", "settings", map[string]interface{}{
		"enabled": settings.Enabled, "time": settings.Time, "keep_days": settings.KeepDays,
		"telegram": settings.Telegram, "token_changed": in.BotToken != nil && *in.BotToken != "" || in.ClearToken,
	})
	c.JSON(http.StatusOK, settings)
}

func (bc *BackupController) TestTelegram(c *gin.Context) {
	if err := bc.svc.SendTest(c.Request.Context()); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "test message sent"})
}

func (bc *BackupController) Download(c *gin.Context) {
	f, item, err := bc.svc.Open(c.Param("name"))
	if err != nil {
		writeBackupError(c, err)
		return
	}
	defer f.Close()
	if actorID, ok := getActorID(c); ok {
		bc.record(c, actorID, "backup.download", item.Name, nil)
	}
	c.DataFromReader(http.StatusOK, item.Size, "application/gzip", f, map[string]string{
		"Content-Disposition": `attachment; filename="` + item.Name + `"`,
	})
}

func (bc *BackupController) Resend(c *gin.Context) {
	item, err := bc.svc.Resend(c.Request.Context(), c.Param("name"))
	if err != nil {
		writeBackupError(c, err)
		return
	}
	c.JSON(http.StatusOK, item)
}

func (bc *BackupController) Delete(c *gin.Context) {
	actorID, _, ok := bc.actor(c)
	if !ok {
		return
	}
	name := c.Param("name")
	if err := bc.svc.Delete(name); err != nil {
		writeBackupError(c, err)
		return
	}
	bc.record(c, actorID, "backup.delete", name, nil)
	c.JSON(http.StatusOK, gin.H{"message": "backup deleted"})
}

// Upload keeps an uploaded .sql or .sql.gz backup in the list (after checking it), ready to restore.
func (bc *BackupController) Upload(c *gin.Context) {
	actorID, label, ok := bc.actor(c)
	if !ok {
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, backup.MaxUploadBytes+(1<<20))
	fh, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "choose a .sql or .sql.gz backup file (up to " + strconv.Itoa(backup.MaxUploadBytes>>20) + " MB)"})
		return
	}
	f, err := fh.Open()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "could not read the file"})
		return
	}
	defer f.Close()
	item, err := bc.svc.SaveUpload(f, label)
	if err != nil {
		writeBackupError(c, err)
		return
	}
	bc.record(c, actorID, "backup.upload", item.Name, map[string]interface{}{"original_name": fh.Filename, "size": fh.Size})
	c.JSON(http.StatusCreated, item)
}

// Restore replaces the database with a kept backup. The body must confirm with "RESTORE".
func (bc *BackupController) Restore(c *gin.Context) {
	actorID, label, ok := bc.actor(c)
	if !ok {
		return
	}
	var body struct {
		Confirm string `json:"confirm" binding:"required"`
	}
	if !bindJSON(c, &body) {
		return
	}
	if body.Confirm != "RESTORE" {
		msg := `type RESTORE to confirm`
		c.JSON(http.StatusBadRequest, gin.H{"error": msg, "fields": gin.H{"confirm": msg}})
		return
	}
	name := c.Param("name")
	result, err := bc.svc.Restore(c.Request.Context(), name, label)
	if err != nil {
		writeBackupError(c, err)
		return
	}
	// The audit table itself was just replaced; this entry is written into the restored one.
	bc.record(c, actorID, "backup.restore", name, map[string]interface{}{"statements": result.Statements, "safety_copy": result.SafetyCopy.Name})
	c.JSON(http.StatusOK, result)
}
