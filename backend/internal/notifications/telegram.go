package notifications

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
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
