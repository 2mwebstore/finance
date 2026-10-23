package notifications

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"time"
)

// TelegramNotifier sends optional operational alerts.
// Credentials come only from environment (never from client).
// Delivery failures never block business operations.
type TelegramNotifier struct {
	botToken string
	chatID   string
	enabled  bool
	client   *http.Client
}

func NewTelegramNotifierFromEnv() *TelegramNotifier {
	token := os.Getenv("TELEGRAM_BOT_TOKEN")
	chat := os.Getenv("TELEGRAM_CHAT_ID")
	enabled := token != "" && chat != ""
	return &TelegramNotifier{
		botToken: token,
		chatID:   chat,
		enabled:  enabled,
		client:   &http.Client{Timeout: 5 * time.Second},
	}
}

func (n *TelegramNotifier) Enabled() bool {
	return n.enabled
}

// Send posts a message asynchronously-safe (caller may go-routine).
func (n *TelegramNotifier) Send(ctx context.Context, text string) error {
	if !n.enabled {
		return nil
	}
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", n.botToken)
	body, _ := json.Marshal(map[string]string{
		"chat_id": n.chatID,
		"text":    text,
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := n.client.Do(req)
	if err != nil {
		log.Printf("telegram send failed: %v", err)
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		log.Printf("telegram status %d", resp.StatusCode)
		return fmt.Errorf("telegram status %d", resp.StatusCode)
	}
	return nil
}

// NotifyAsync fire-and-forget so API latency is unaffected.
func (n *TelegramNotifier) NotifyAsync(text string) {
	if !n.enabled {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		defer cancel()
		_ = n.Send(ctx, text)
	}()
}

// TelegramMaxUpload is the largest file a bot may send (Telegram Bot API limit).
const TelegramMaxUpload = 50 << 20

// SendTelegramText sends a message with the given bot and chat (used where the credentials come
// from saved settings rather than the environment).
func SendTelegramText(ctx context.Context, botToken, chatID, text string) error {
	body, _ := json.Marshal(map[string]string{"chat_id": chatID, "text": text})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, telegramURL(botToken, "sendMessage"), bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	return doTelegram(req)
}

// SendTelegramDocument uploads a file to the chat, with an optional caption.
func SendTelegramDocument(ctx context.Context, botToken, chatID, fileName string, file io.Reader, caption string) error {
	var buf bytes.Buffer
	form := multipart.NewWriter(&buf)
	_ = form.WriteField("chat_id", chatID)
	if caption != "" {
		_ = form.WriteField("caption", caption)
	}
	part, err := form.CreateFormFile("document", fileName)
	if err != nil {
		return err
	}
	if _, err := io.Copy(part, file); err != nil {
		return err
	}
	if err := form.Close(); err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, telegramURL(botToken, "sendDocument"), &buf)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", form.FormDataContentType())
	return doTelegram(req)
}

func telegramURL(botToken, method string) string {
	return fmt.Sprintf("https://api.telegram.org/bot%s/%s", botToken, method)
}

var telegramClient = &http.Client{Timeout: 2 * time.Minute}

// doTelegram sends the request and turns Telegram's {"ok":false,"description":...} into an error.
// The bot token is part of the URL, so errors never include the URL.
func doTelegram(req *http.Request) error {
	resp, err := telegramClient.Do(req)
	if err != nil {
		return fmt.Errorf("could not reach Telegram")
	}
	defer resp.Body.Close()
	var answer struct {
		OK          bool   `json:"ok"`
		Description string `json:"description"`
	}
	_ = json.NewDecoder(io.LimitReader(resp.Body, 1<<16)).Decode(&answer)
	if resp.StatusCode >= 300 || !answer.OK {
		if answer.Description != "" {
			return fmt.Errorf("telegram: %s", answer.Description)
		}
		return fmt.Errorf("telegram answered with status %d", resp.StatusCode)
	}
	return nil
}
