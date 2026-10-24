package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/cambodia-finance/income-expense/internal/models"
	"github.com/cambodia-finance/income-expense/internal/notifications"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Branch Telegram: each branch can send its transaction events to its own Telegram chat (a
// group or channel), with its own bot or the default one (the bot set up for backups, or
// TELEGRAM_BOT_TOKEN). Settings are kept in the settings table as JSON under
// branch.telegram.<branch id>; the bot token never leaves the server.

// Transaction events a branch can send.
const (
	TelegramEventCreated  = "created"
	TelegramEventApproved = "approved"
	TelegramEventRejected = "rejected"
	TelegramEventVoided   = "voided"
	TelegramEventReversed = "reversed"
)

// TelegramEvents lists every event, in the order the page shows them.
var TelegramEvents = []string{TelegramEventCreated, TelegramEventApproved, TelegramEventRejected, TelegramEventVoided, TelegramEventReversed}

var defaultTelegramEvents = []string{TelegramEventCreated, TelegramEventApproved, TelegramEventRejected}

const branchTelegramKeyPrefix = "branch.telegram."

// The key the backups page saves its bot under (see backup.keyBotToken).
const sharedBotTokenKey = "telegram.bot_token"

var (
	// A group or channel id (-100…), a user id, or a public @channel name.
	telegramChatRule = regexp.MustCompile(`^(-?\d{5,20}|@[A-Za-z][A-Za-z0-9_]{4,31})$`)
	// The token BotFather gives: <bot id>:<secret>.
	telegramTokenRule = regexp.MustCompile(`^\d{5,12}:[A-Za-z0-9_-]{30,64}$`)
)

// BranchTelegramSettings is what the branch page shows. The token itself is never sent back.
type BranchTelegramSettings struct {
	Enabled         bool     `json:"enabled"`
	ChatID          string   `json:"chat_id"`
	Events          []string `json:"events"`
	OwnBot          bool     `json:"own_bot"`           // a bot token is saved for this branch
	DefaultBotReady bool     `json:"default_bot_ready"` // the shared bot (backups page or environment) exists
	Ready           bool     `json:"ready"`             // on, with a chat and a bot
}

// BranchTelegramInput is a change from the branch page; nil fields stay as they are.
type BranchTelegramInput struct {
	Enabled    *bool    `json:"enabled"`
	ChatID     *string  `json:"chat_id"`
	BotToken   *string  `json:"bot_token"`
	ClearToken bool     `json:"clear_bot_token"`
	Events     []string `json:"events"`
}

type branchTelegramStored struct {
	Enabled  bool     `json:"enabled"`
	ChatID   string   `json:"chat_id"`
	BotToken string   `json:"bot_token,omitempty"`
	Events   []string `json:"events"`
}

type BranchTelegramService struct {
	db  *gorm.DB
	loc *time.Location
	// send is the Telegram call (replaced in tests).
	send func(ctx context.Context, token, chatID, text string) error
}

func NewBranchTelegramService(db *gorm.DB, timezone string) *BranchTelegramService {
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		loc = time.UTC
	}
	return &BranchTelegramService{db: db, loc: loc, send: notifications.SendTelegramText}
}

func branchTelegramKey(branchID uuid.UUID) string { return branchTelegramKeyPrefix + branchID.String() }

func (s *BranchTelegramService) load(ctx context.Context, branchID uuid.UUID) (branchTelegramStored, error) {
	var row models.Setting
	err := s.db.WithContext(ctx).Where("`key` = ?", branchTelegramKey(branchID)).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return branchTelegramStored{Events: append([]string(nil), defaultTelegramEvents...)}, nil
	}
	if err != nil {
		return branchTelegramStored{}, err
	}
	var st branchTelegramStored
	if json.Unmarshal([]byte(row.Value), &st) != nil {
		return branchTelegramStored{Events: append([]string(nil), defaultTelegramEvents...)}, nil
	}
	return st, nil
}

// sharedToken is the default bot: the one saved on the backups page, else TELEGRAM_BOT_TOKEN.
func (s *BranchTelegramService) sharedToken(ctx context.Context) string {
	var row models.Setting
	if err := s.db.WithContext(ctx).Where("`key` = ?", sharedBotTokenKey).First(&row).Error; err == nil && row.Value != "" {
		return row.Value
	}
	return os.Getenv("TELEGRAM_BOT_TOKEN")
}

func (s *BranchTelegramService) view(ctx context.Context, st branchTelegramStored) BranchTelegramSettings {
	events := st.Events
	if events == nil {
		events = []string{}
	}
	out := BranchTelegramSettings{
		Enabled:         st.Enabled,
		ChatID:          st.ChatID,
		Events:          events,
		OwnBot:          st.BotToken != "",
		DefaultBotReady: s.sharedToken(ctx) != "",
	}
	out.Ready = out.Enabled && out.ChatID != "" && (out.OwnBot || out.DefaultBotReady)
	return out
}

func (s *BranchTelegramService) Get(ctx context.Context, branchID uuid.UUID) (BranchTelegramSettings, error) {
	st, err := s.load(ctx, branchID)
	if err != nil {
		return BranchTelegramSettings{}, err
	}
	return s.view(ctx, st), nil
}

// Save checks and stores the branch's Telegram settings.
func (s *BranchTelegramService) Save(ctx context.Context, branchID uuid.UUID, in BranchTelegramInput, actor uuid.UUID) (BranchTelegramSettings, error) {
	st, err := s.load(ctx, branchID)
	if err != nil {
		return BranchTelegramSettings{}, err
	}
	if in.Enabled != nil {
		st.Enabled = *in.Enabled
	}
	if in.ChatID != nil {
		st.ChatID = strings.TrimSpace(*in.ChatID)
	}
	if in.ClearToken {
		st.BotToken = ""
	}
	if in.BotToken != nil && strings.TrimSpace(*in.BotToken) != "" {
		token := strings.TrimSpace(*in.BotToken)
		if !telegramTokenRule.MatchString(token) {
			return BranchTelegramSettings{}, fieldError("bot_token", "enter the bot token from @BotFather (like 123456789:AA…)")
		}
		st.BotToken = token
	}
	if in.Events != nil {
		allowed := map[string]bool{}
		for _, e := range TelegramEvents {
			allowed[e] = true
		}
		seen := map[string]bool{}
		events := []string{}
		for _, e := range in.Events {
			if !allowed[e] {
				return BranchTelegramSettings{}, fieldError("events", "unknown event: "+e)
			}
			if !seen[e] {
				seen[e] = true
				events = append(events, e)
			}
		}
		st.Events = events
	}
	if st.ChatID != "" && !telegramChatRule.MatchString(st.ChatID) {
		return BranchTelegramSettings{}, fieldError("chat_id", "enter the chat ID (like -1001234567890) or a public @channel name")
	}
	if st.Enabled {
		if st.ChatID == "" {
			return BranchTelegramSettings{}, fieldError("chat_id", "enter the chat ID of the branch's Telegram group")
		}
		if st.BotToken == "" && s.sharedToken(ctx) == "" {
			return BranchTelegramSettings{}, fieldError("bot_token", "add a bot token; no default bot is set up on the Backups page")
		}
		if len(st.Events) == 0 {
			return BranchTelegramSettings{}, fieldError("events", "choose at least one event to send")
		}
	}
	raw, _ := json.Marshal(st)
	err = s.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "key"}},
		DoUpdates: clause.AssignmentColumns([]string{"value", "updated_at", "updated_by"}),
	}).Create(&models.Setting{Key: branchTelegramKey(branchID), Value: string(raw), UpdatedAt: time.Now(), UpdatedBy: &actor}).Error
	if err != nil {
		return BranchTelegramSettings{}, err
	}
	return s.view(ctx, st), nil
}

// EnabledBranches returns the branches that send to Telegram (for the branch list).
func (s *BranchTelegramService) EnabledBranches(ctx context.Context) (map[uuid.UUID]bool, error) {
	var rows []models.Setting
	if err := s.db.WithContext(ctx).Where("`key` LIKE ?", branchTelegramKeyPrefix+"%").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := map[uuid.UUID]bool{}
	for _, row := range rows {
		id, err := uuid.Parse(strings.TrimPrefix(row.Key, branchTelegramKeyPrefix))
		if err != nil {
			continue
		}
		var st branchTelegramStored
		if json.Unmarshal([]byte(row.Value), &st) == nil && st.Enabled && st.ChatID != "" {
			out[id] = true
		}
	}
	return out, nil
}

// target returns the bot and chat to use, or ok=false when the branch doesn't send this event.
func (s *BranchTelegramService) target(ctx context.Context, branchID uuid.UUID, event string) (token, chatID string, ok bool) {
	st, err := s.load(ctx, branchID)
	if err != nil || !st.Enabled || st.ChatID == "" {
		return "", "", false
	}
	if event != "" {
		wanted := false
		for _, e := range st.Events {
			wanted = wanted || e == event
		}
		if !wanted {
			return "", "", false
		}
	}
	token = st.BotToken
	if token == "" {
		token = s.sharedToken(ctx)
	}
	return token, st.ChatID, token != ""
}

// SendTest sends a test message with the branch's saved settings.
func (s *BranchTelegramService) SendTest(ctx context.Context, branch *models.Branch) error {
	st, err := s.load(ctx, branch.ID)
	if err != nil {
		return err
	}
	if st.ChatID == "" {
		return fieldError("chat_id", "save a chat ID first")
	}
	token := st.BotToken
	if token == "" {
		token = s.sharedToken(ctx)
	}
	if token == "" {
		return fieldError("bot_token", "save a bot token first; no default bot is set up on the Backups page")
	}
	text := fmt.Sprintf("✅ Test from Income & Expense\n%s · %s\nThis chat will get the branch's transaction updates.", branch.Code, branch.Name)
	return s.send(ctx, token, st.ChatID, text)
}

// NotifyTransaction sends a transaction event to its branch's chat in the background, when the
// branch is set up for that event. note is the approval/rejection note or the void/reversal
// reason. Failures are logged and never affect the request.
func (s *BranchTelegramService) NotifyTransaction(event string, txnID, actorID uuid.UUID, note string) {
	if s == nil {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()
		var txn models.Transaction
		if err := s.db.WithContext(ctx).Preload("Branch").Preload("Category").Preload("PaymentMethod").First(&txn, "id = ?", txnID).Error; err != nil {
			return
		}
		token, chatID, ok := s.target(ctx, txn.BranchID, event)
		if !ok {
			return
		}
		names := s.userNames(ctx, txn.CreatedByID, actorID)
		reverses := ""
		if txn.ReversesTxnID != nil {
			var orig models.Transaction
			if s.db.WithContext(ctx).Select("transaction_number").First(&orig, "id = ?", *txn.ReversesTxnID).Error == nil {
				reverses = orig.TransactionNumber
			}
		}
		text := transactionMessage(event, &txn, names[txn.CreatedByID], names[actorID], note, reverses, s.loc)
		if err := s.send(ctx, token, chatID, text); err != nil {
			log.Printf("telegram: branch %s, %s %s not sent: %v", txn.BranchID, event, txn.TransactionNumber, err)
		}
	}()
}

func (s *BranchTelegramService) userNames(ctx context.Context, ids ...uuid.UUID) map[uuid.UUID]string {
	var users []models.User
	s.db.WithContext(ctx).Select("id", "username", "full_name").Where("id IN ?", ids).Find(&users)
	out := map[uuid.UUID]string{}
	for _, u := range users {
		out[u.ID] = u.FullName
		if out[u.ID] == "" {
			out[u.ID] = u.Username
		}
	}
	return out
}

// transactionMessage is the plain-text Telegram message for a transaction event.
func transactionMessage(event string, t *models.Transaction, creator, actor, note, reverses string, loc *time.Location) string {
	kind := "Income"
	if t.Type == models.TxnTypeExpense {
		kind = "Expense"
	}
	var head, actorLabel, noteLabel string
	switch event {
	case TelegramEventCreated:
		head = "🆕 New " + strings.ToLower(kind)
		if t.Status == models.TxnStatusPending {
			head += " · waiting for approval"
		}
	case TelegramEventApproved:
		head, actorLabel, noteLabel = "✅ "+kind+" approved", "Approved by", "Note"
	case TelegramEventRejected:
		head, actorLabel, noteLabel = "❌ "+kind+" rejected", "Rejected by", "Reason"
	case TelegramEventVoided:
		head, actorLabel, noteLabel = "🚫 "+kind+" voided", "Voided by", "Reason"
	case TelegramEventReversed:
		head, actorLabel, noteLabel = "↩️ "+kind+" reversed", "Reversed by", "Reason"
	default:
		head = kind + " " + event
	}
	lines := []string{head}
	if t.Branch != nil {
		lines = append(lines, "Branch: "+t.Branch.Code+" · "+t.Branch.Name)
	}
	number := t.TransactionNumber
	if reverses != "" {
		number += " (reverses " + reverses + ")"
	}
	lines = append(lines, number, "Amount: "+formatTelegramAmount(t.Amount, t.Currency))
	var details []string
	if t.Category != nil {
		details = append(details, t.Category.Name)
	}
	if t.PaymentMethod != nil {
		details = append(details, t.PaymentMethod.Name)
	}
	if len(details) > 0 {
		lines = append(lines, "Category: "+strings.Join(details, " · "))
	}
	lines = append(lines, "Date: "+t.TransactionDate.In(loc).Format("02 Jan 2006"))
	if creator != "" {
		lines = append(lines, "By: "+creator)
	}
	if actorLabel != "" && actor != "" {
		lines = append(lines, actorLabel+": "+actor)
	}
	if note = strings.TrimSpace(note); note != "" && noteLabel != "" {
		if r := []rune(note); len(r) > 300 {
			note = string(r[:300]) + "…"
		}
		lines = append(lines, noteLabel+": "+note)
	}
	return strings.Join(lines, "\n")
}

// formatTelegramAmount writes 1234.5 USD as "1,234.50 USD" and riel without decimals.
func formatTelegramAmount(amount decimal.Decimal, currency string) string {
	places := int32(2)
	if currency == "KHR" {
		places = 0
	}
	s := amount.StringFixed(places)
	sign := ""
	if strings.HasPrefix(s, "-") {
		sign, s = "-", s[1:]
	}
	whole, frac := s, ""
	if i := strings.IndexByte(s, '.'); i >= 0 {
		whole, frac = s[:i], s[i:]
	}
	var b strings.Builder
	for i, r := range whole {
		if i > 0 && (len(whole)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(r)
	}
	return sign + b.String() + frac + " " + currency
}
