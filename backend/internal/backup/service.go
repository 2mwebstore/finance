package backup

import (
	"bufio"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/cambodia-finance/income-expense/internal/database"
	"github.com/cambodia-finance/income-expense/internal/models"
	"github.com/cambodia-finance/income-expense/internal/notifications"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// What started a backup; part of the file name.
const (
	TriggerScheduled  = "scheduled"
	TriggerManual     = "manual"
	TriggerPreRestore = "pre-restore"
	TriggerUploaded   = "uploaded"
)

// Limits for files people upload to restore from.
const (
	MaxUploadBytes     = 100 << 20 // compressed or plain, as uploaded
	maxScriptBytes     = 500 << 20 // after decompressing
	defaultBackupTime  = "12:00"
	defaultKeepDays    = 7
	maxKeepDays        = 90
	schedulerCheckTime = time.Minute
)

// Settings keys, in the settings table (not app_settings, which GET /settings shows to everyone
// with dashboard access: the bot token must never be listed there).
const (
	keyEnabled       = "backup.enabled"
	keyTime          = "backup.time"
	keyKeepDays      = "backup.keep_days"
	keyTelegram      = "backup.telegram"
	keyLastScheduled = "backup.last_scheduled"
	keyChatID        = "telegram.chat_id"
	keyBotToken      = "telegram.bot_token"
)

var (
	ErrNotFound    = errors.New("backup not found")
	ErrBusy        = errors.New("another backup or restore is running; try again in a moment")
	ErrBadTime     = errors.New("time must be HH:MM, 00:00 to 23:59")
	ErrBadKeepDays = fmt.Errorf("keep days must be 1 to %d", maxKeepDays)
	fileNameRule   = regexp.MustCompile(`^backup-(\d{8}-\d{6})-([a-z-]+)\.sql\.gz$`)
	timeRule       = regexp.MustCompile(`^([01]\d|2[0-3]):([0-5]\d)$`)
)

// Item is one backup file on the server.
type Item struct {
	Name          string    `json:"name"`
	Size          int64     `json:"size"`
	CreatedAt     time.Time `json:"created_at"`
	Trigger       string    `json:"trigger"`
	Telegram      string    `json:"telegram"` // sent | failed | off | skipped
	TelegramError string    `json:"telegram_error,omitempty"`
	CreatedBy     string    `json:"created_by,omitempty"`
}

// Settings is what the admin page shows and edits. The bot token itself is never sent back.
type Settings struct {
	Enabled          bool       `json:"enabled"`
	Time             string     `json:"time"`
	KeepDays         int        `json:"keep_days"`
	Telegram         bool       `json:"telegram"`
	ChatID           string     `json:"telegram_chat_id"`
	TokenSet         bool       `json:"telegram_token_set"`
	TelegramFromEnv  bool       `json:"telegram_from_env"`
	TelegramReady    bool       `json:"telegram_ready"`
	Timezone         string     `json:"timezone"`
	NextRun          *time.Time `json:"next_run,omitempty"`
	LastScheduledDay string     `json:"last_scheduled_day,omitempty"`
	token            string
}

// SettingsInput is a change from the admin page; nil fields stay as they are.
type SettingsInput struct {
	Enabled    *bool   `json:"enabled"`
	Time       *string `json:"time"`
	KeepDays   *int    `json:"keep_days"`
	Telegram   *bool   `json:"telegram"`
	ChatID     *string `json:"telegram_chat_id"`
	BotToken   *string `json:"telegram_bot_token"`
	ClearToken bool    `json:"clear_telegram_token"`
}

type sidecar struct {
	Trigger       string `json:"trigger"`
	Telegram      string `json:"telegram"`
	TelegramError string `json:"telegram_error,omitempty"`
	CreatedBy     string `json:"created_by,omitempty"`
}

// Service makes, keeps, sends and restores database backups.
type Service struct {
	db      *gorm.DB
	dir     string
	loc     *time.Location
	appName string
	mu      sync.Mutex // one backup or restore at a time
}

func NewService(db *gorm.DB, dir, timezone, appName string) (*Service, error) {
	if err := os.MkdirAll(dir, 0o750); err != nil {
		// Don't stop the API over it: keep backups in the temp folder and say so.
		fallback := filepath.Join(os.TempDir(), "income-expense-backups")
		log.Printf("backup: can't use %s (%v); keeping backups in %s instead", dir, err, fallback)
		if err := os.MkdirAll(fallback, 0o750); err != nil {
			return nil, fmt.Errorf("backup dir: %w", err)
		}
		dir = fallback
	}
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		loc = time.FixedZone("Asia/Phnom_Penh", 7*3600)
	}
	return &Service{db: db, dir: dir, loc: loc, appName: appName}, nil
}

// --- settings ---

func (s *Service) readSettings(ctx context.Context) (map[string]string, error) {
	var rows []models.Setting
	if err := s.db.WithContext(ctx).Where("`key` LIKE ? OR `key` LIKE ?", "backup.%", "telegram.%").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := map[string]string{}
	for _, r := range rows {
		out[r.Key] = r.Value
	}
	return out, nil
}

func (s *Service) writeSetting(tx *gorm.DB, key, value string, actor *uuid.UUID) error {
	return tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "key"}},
		DoUpdates: clause.AssignmentColumns([]string{"value", "updated_at", "updated_by"}),
	}).Create(&models.Setting{Key: key, Value: value, UpdatedAt: time.Now(), UpdatedBy: actor}).Error
}

func (s *Service) Settings(ctx context.Context) (Settings, error) {
	raw, err := s.readSettings(ctx)
	if err != nil {
		return Settings{}, err
	}
	st := Settings{
		Enabled:          raw[keyEnabled] != "false",
		Time:             defaultBackupTime,
		KeepDays:         defaultKeepDays,
		Telegram:         raw[keyTelegram] != "false",
		ChatID:           raw[keyChatID],
		Timezone:         s.loc.String(),
		LastScheduledDay: raw[keyLastScheduled],
		token:            raw[keyBotToken],
	}
	if timeRule.MatchString(raw[keyTime]) {
		st.Time = raw[keyTime]
	}
	if n, err := strconv.Atoi(raw[keyKeepDays]); err == nil && n >= 1 && n <= maxKeepDays {
		st.KeepDays = n
	}
	// Saved settings win; otherwise the environment (TELEGRAM_BOT_TOKEN / TELEGRAM_CHAT_ID).
	if st.token == "" && st.ChatID == "" {
		if envToken, envChat := os.Getenv("TELEGRAM_BOT_TOKEN"), os.Getenv("TELEGRAM_CHAT_ID"); envToken != "" && envChat != "" {
			st.token, st.ChatID, st.TelegramFromEnv = envToken, envChat, true
		}
	}
	st.TokenSet = st.token != ""
	st.TelegramReady = st.TokenSet && st.ChatID != ""
	if st.Enabled {
		next := s.nextRun(st, raw[keyLastScheduled], time.Now())
		st.NextRun = &next
	}
	return st, nil
}

func (s *Service) SaveSettings(ctx context.Context, in SettingsInput, actor uuid.UUID) (Settings, error) {
	if in.Time != nil && !timeRule.MatchString(strings.TrimSpace(*in.Time)) {
		return Settings{}, ErrBadTime
	}
	if in.KeepDays != nil && (*in.KeepDays < 1 || *in.KeepDays > maxKeepDays) {
		return Settings{}, ErrBadKeepDays
	}
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		set := func(key, value string) error { return s.writeSetting(tx, key, value, &actor) }
		if in.Enabled != nil {
			if err := set(keyEnabled, strconv.FormatBool(*in.Enabled)); err != nil {
				return err
			}
		}
		if in.Time != nil {
			if err := set(keyTime, strings.TrimSpace(*in.Time)); err != nil {
				return err
			}
		}
		if in.KeepDays != nil {
			if err := set(keyKeepDays, strconv.Itoa(*in.KeepDays)); err != nil {
				return err
			}
		}
		if in.Telegram != nil {
			if err := set(keyTelegram, strconv.FormatBool(*in.Telegram)); err != nil {
				return err
			}
		}
		if in.ChatID != nil {
			if err := set(keyChatID, strings.TrimSpace(*in.ChatID)); err != nil {
				return err
			}
		}
		if in.ClearToken {
			if err := set(keyBotToken, ""); err != nil {
				return err
			}
		} else if in.BotToken != nil && strings.TrimSpace(*in.BotToken) != "" {
			if err := set(keyBotToken, strings.TrimSpace(*in.BotToken)); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return Settings{}, err
	}
	if in.KeepDays != nil {
		s.prune(*in.KeepDays)
	}
	return s.Settings(ctx)
}

// nextRun is the next scheduled backup: today at the set time if today's hasn't run, else tomorrow.
func (s *Service) nextRun(st Settings, lastDay string, now time.Time) time.Time {
	now = now.In(s.loc)
	hh, _ := strconv.Atoi(st.Time[:2])
	mm, _ := strconv.Atoi(st.Time[3:])
	due := time.Date(now.Year(), now.Month(), now.Day(), hh, mm, 0, 0, s.loc)
	if lastDay == now.Format("2006-01-02") {
		due = due.AddDate(0, 0, 1)
	} else if now.After(due) {
		return now // overdue: the scheduler runs it within a minute
	}
	return due
}

// --- files ---

func (s *Service) path(name string) (string, error) {
	if !fileNameRule.MatchString(name) {
		return "", ErrNotFound
	}
	return filepath.Join(s.dir, name), nil
}

func (s *Service) List() ([]Item, error) {
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		return nil, err
	}
	items := []Item{}
	for _, e := range entries {
		if item, ok := s.item(e.Name()); ok {
			items = append(items, item)
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].CreatedAt.After(items[j].CreatedAt) })
	return items, nil
}

func (s *Service) item(name string) (Item, bool) {
	m := fileNameRule.FindStringSubmatch(name)
	if m == nil {
		return Item{}, false
	}
	info, err := os.Stat(filepath.Join(s.dir, name))
	if err != nil || info.IsDir() {
		return Item{}, false
	}
	created, err := time.ParseInLocation("20060102-150405", m[1], s.loc)
	if err != nil {
		return Item{}, false
	}
	item := Item{Name: name, Size: info.Size(), CreatedAt: created, Trigger: m[2], Telegram: "off"}
	if raw, err := os.ReadFile(filepath.Join(s.dir, name+".json")); err == nil {
		var meta sidecar
		if json.Unmarshal(raw, &meta) == nil {
			item.Telegram, item.TelegramError, item.CreatedBy = meta.Telegram, meta.TelegramError, meta.CreatedBy
		}
	}
	return item, true
}

func (s *Service) writeSidecar(name string, meta sidecar) {
	raw, _ := json.Marshal(meta)
	_ = os.WriteFile(filepath.Join(s.dir, name+".json"), raw, 0o640)
}

// Open returns the backup file for download.
func (s *Service) Open(name string) (*os.File, Item, error) {
	p, err := s.path(name)
	if err != nil {
		return nil, Item{}, err
	}
	item, ok := s.item(name)
	if !ok {
		return nil, Item{}, ErrNotFound
	}
	f, err := os.Open(p)
	return f, item, err
}

func (s *Service) Delete(name string) error {
	p, err := s.path(name)
	if err != nil {
		return err
	}
	if err := os.Remove(p); err != nil {
		if os.IsNotExist(err) {
			return ErrNotFound
		}
		return err
	}
	_ = os.Remove(p + ".json")
	return nil
}

// prune deletes backups older than keepDays.
func (s *Service) prune(keepDays int) {
	items, err := s.List()
	if err != nil {
		return
	}
	cutoff := time.Now().AddDate(0, 0, -keepDays)
	for _, item := range items {
		if item.CreatedAt.Before(cutoff) {
			if err := s.Delete(item.Name); err == nil {
				log.Printf("backup: removed %s (older than %d days)", item.Name, keepDays)
			}
		}
	}
}

// newName picks a free file name for a backup made now.
func (s *Service) newName(trigger string) string {
	now := time.Now().In(s.loc)
	for {
		name := fmt.Sprintf("backup-%s-%s.sql.gz", now.Format("20060102-150405"), trigger)
		if _, err := os.Stat(filepath.Join(s.dir, name)); os.IsNotExist(err) {
			return name
		}
		now = now.Add(time.Second)
	}
}

// --- backup ---

// Run makes a backup now, keeps it on the server, sends it to Telegram (when switched on) and
// removes backups older than the keep period.
func (s *Service) Run(ctx context.Context, trigger, createdBy string) (Item, error) {
	if !s.mu.TryLock() {
		return Item{}, ErrBusy
	}
	defer s.mu.Unlock()
	return s.runLocked(ctx, trigger, createdBy)
}

func (s *Service) runLocked(ctx context.Context, trigger, createdBy string) (Item, error) {
	st, err := s.Settings(ctx)
	if err != nil {
		return Item{}, err
	}
	sqlDB, err := s.db.DB()
	if err != nil {
		return Item{}, err
	}
	name := s.newName(trigger)
	final := filepath.Join(s.dir, name)
	tmp, err := os.CreateTemp(s.dir, ".partial-*")
	if err != nil {
		return Item{}, err
	}
	defer os.Remove(tmp.Name())
	gz := gzip.NewWriter(tmp)
	if err := Dump(ctx, sqlDB, gz); err != nil {
		tmp.Close()
		return Item{}, fmt.Errorf("backup failed: %w", err)
	}
	if err := gz.Close(); err != nil {
		tmp.Close()
		return Item{}, err
	}
	if err := tmp.Close(); err != nil {
		return Item{}, err
	}
	if err := os.Rename(tmp.Name(), final); err != nil {
		return Item{}, err
	}
	meta := sidecar{Trigger: trigger, Telegram: "off", CreatedBy: createdBy}
	if st.Telegram && st.TelegramReady {
		meta.Telegram, meta.TelegramError = s.sendFile(ctx, st, name)
	}
	s.writeSidecar(name, meta)
	s.prune(st.KeepDays)
	item, _ := s.item(name)
	log.Printf("backup: %s (%d bytes, telegram %s)", name, item.Size, meta.Telegram)
	return item, nil
}

func (s *Service) sendFile(ctx context.Context, st Settings, name string) (status, errText string) {
	f, err := os.Open(filepath.Join(s.dir, name))
	if err != nil {
		return "failed", err.Error()
	}
	defer f.Close()
	if info, err := f.Stat(); err == nil && info.Size() > notifications.TelegramMaxUpload {
		return "skipped", "the file is larger than Telegram's 50 MB limit for bots"
	}
	caption := fmt.Sprintf("%s database backup · %s", s.appName, time.Now().In(s.loc).Format("2006-01-02 15:04"))
	sendCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	if err := notifications.SendTelegramDocument(sendCtx, st.token, st.ChatID, name, f, caption); err != nil {
		log.Printf("backup: sending %s to Telegram failed: %v", name, err)
		return "failed", err.Error()
	}
	return "sent", ""
}

// Resend sends an existing backup to Telegram again (e.g. after fixing the bot settings).
func (s *Service) Resend(ctx context.Context, name string) (Item, error) {
	item, ok := s.item(name)
	if !ok || !fileNameRule.MatchString(name) {
		return Item{}, ErrNotFound
	}
	st, err := s.Settings(ctx)
	if err != nil {
		return Item{}, err
	}
	if !st.TelegramReady {
		return Item{}, errors.New("set the Telegram bot token and chat ID first")
	}
	status, errText := s.sendFile(ctx, st, name)
	s.writeSidecar(name, sidecar{Trigger: item.Trigger, Telegram: status, TelegramError: errText, CreatedBy: item.CreatedBy})
	item, _ = s.item(name)
	if status == "failed" {
		return item, errors.New(errText)
	}
	return item, nil
}

// SendTest checks the bot settings with a short message.
func (s *Service) SendTest(ctx context.Context) error {
	st, err := s.Settings(ctx)
	if err != nil {
		return err
	}
	if !st.TelegramReady {
		return errors.New("set the Telegram bot token and chat ID first")
	}
	return notifications.SendTelegramText(ctx, st.token, st.ChatID,
		fmt.Sprintf("%s: Telegram is connected. Database backups will be sent to this chat.", s.appName))
}

// --- upload & restore ---

// readScript returns the SQL text of a backup, decompressing gzip when the bytes say so.
func readScript(r io.Reader) (string, error) {
	buf := bufio.NewReader(r)
	head, _ := buf.Peek(2)
	var src io.Reader = buf
	if len(head) == 2 && head[0] == 0x1f && head[1] == 0x8b {
		gz, err := gzip.NewReader(buf)
		if err != nil {
			return "", fmt.Errorf("the file is not a valid .gz archive")
		}
		defer gz.Close()
		src = gz
	}
	raw, err := io.ReadAll(io.LimitReader(src, maxScriptBytes+1))
	if err != nil {
		return "", fmt.Errorf("could not read the backup file")
	}
	if len(raw) > maxScriptBytes {
		return "", fmt.Errorf("the backup is larger than %d MB once unpacked", maxScriptBytes>>20)
	}
	return string(raw), nil
}

// SaveUpload checks an uploaded .sql or .sql.gz file and keeps it (compressed) next to the other
// backups, so it can be restored from the list.
func (s *Service) SaveUpload(r io.Reader, createdBy string) (Item, error) {
	script, err := readScript(r)
	if err != nil {
		return Item{}, err
	}
	if err := Validate(script); err != nil {
		return Item{}, err
	}
	name := s.newName(TriggerUploaded)
	tmp, err := os.CreateTemp(s.dir, ".partial-*")
	if err != nil {
		return Item{}, err
	}
	defer os.Remove(tmp.Name())
	gz := gzip.NewWriter(tmp)
	if _, err := io.WriteString(gz, script); err != nil {
		tmp.Close()
		return Item{}, err
	}
	if err := gz.Close(); err != nil {
		tmp.Close()
		return Item{}, err
	}
	if err := tmp.Close(); err != nil {
		return Item{}, err
	}
	if err := os.Rename(tmp.Name(), filepath.Join(s.dir, name)); err != nil {
		return Item{}, err
	}
	s.writeSidecar(name, sidecar{Trigger: TriggerUploaded, Telegram: "off", CreatedBy: createdBy})
	item, _ := s.item(name)
	return item, nil
}

// RestoreResult says what a restore did.
type RestoreResult struct {
	Statements int  `json:"statements"`
	SafetyCopy Item `json:"safety_copy"`
}

// Restore replaces the database with a kept backup. A safety copy of the current database is made
// first, and the backup/Telegram settings are kept as they are now.
func (s *Service) Restore(ctx context.Context, name, createdBy string) (RestoreResult, error) {
	if !s.mu.TryLock() {
		return RestoreResult{}, ErrBusy
	}
	defer s.mu.Unlock()
	p, err := s.path(name)
	if err != nil {
		return RestoreResult{}, err
	}
	f, err := os.Open(p)
	if err != nil {
		return RestoreResult{}, ErrNotFound
	}
	script, err := readScript(f)
	f.Close()
	if err != nil {
		return RestoreResult{}, err
	}
	if err := Validate(script); err != nil {
		return RestoreResult{}, err
	}
	keep, err := s.readSettings(ctx)
	if err != nil {
		return RestoreResult{}, err
	}
	safety, err := s.runLocked(ctx, TriggerPreRestore, createdBy)
	if err != nil {
		return RestoreResult{}, fmt.Errorf("could not make a safety copy first, so nothing was restored: %w", err)
	}
	sqlDB, err := s.db.DB()
	if err != nil {
		return RestoreResult{}, err
	}
	n, err := Restore(ctx, sqlDB, script)
	if err != nil {
		return RestoreResult{SafetyCopy: safety}, fmt.Errorf("%w. Restore %s to go back to how it was", err, safety.Name)
	}
	// The restored settings table is from the backup; put this server's backup settings back.
	if err := s.db.Transaction(func(tx *gorm.DB) error {
		for key, value := range keep {
			if err := s.writeSetting(tx, key, value, nil); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		log.Printf("backup: could not keep backup settings after restore: %v", err)
	}
	if err := database.EnsureRequiredTables(s.db); err != nil {
		log.Printf("backup: required tables after restore: %v", err)
	}
	return RestoreResult{Statements: n, SafetyCopy: safety}, nil
}

// --- schedule ---

// StartScheduler checks every minute whether today's backup is due (and catches up after a
// restart that missed it). It stops when ctx ends.
func (s *Service) StartScheduler(ctx context.Context) {
	if st, err := s.Settings(ctx); err == nil {
		s.prune(st.KeepDays)
	}
	go func() {
		ticker := time.NewTicker(schedulerCheckTime)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				s.tick(ctx)
			}
		}
	}()
}

func (s *Service) tick(ctx context.Context) {
	st, err := s.Settings(ctx)
	if err != nil || !st.Enabled || st.NextRun == nil || time.Now().Before(*st.NextRun) {
		return
	}
	today := time.Now().In(s.loc).Format("2006-01-02")
	// Mark the day first so a failing backup is reported once, not every minute.
	if err := s.writeSetting(s.db.WithContext(ctx), keyLastScheduled, today, nil); err != nil {
		return
	}
	runCtx, cancel := context.WithTimeout(ctx, 30*time.Minute)
	defer cancel()
	if _, err := s.Run(runCtx, TriggerScheduled, "schedule"); err != nil {
		log.Printf("backup: scheduled backup failed: %v", err)
		if st.Telegram && st.TelegramReady {
			_ = notifications.SendTelegramText(runCtx, st.token, st.ChatID,
				fmt.Sprintf("%s: the daily database backup FAILED: %v", s.appName, err))
		}
	}
}
