//go:build integration

package tests

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"testing"
	"time"
)

// Integration tests against a running API (default http://127.0.0.1:8080).
// Run with: go test -tags=integration ./tests/ -count=1
// Requires MySQL + API already started with SETUP_TOKEN set.

func baseURL() string {
	if u := os.Getenv("API_BASE_URL"); u != "" {
		return u
	}
	return "http://127.0.0.1:8080"
}

func doJSON(t *testing.T, method, path string, body interface{}, token string, cookies []*http.Cookie) (*http.Response, map[string]interface{}) {
	t.Helper()
	var rdr io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		rdr = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, baseURL()+path, rdr)
	if err != nil {
		t.Fatal(err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	for _, c := range cookies {
		req.AddCookie(c)
	}
	client := &http.Client{Timeout: 10 * time.Second, CheckRedirect: func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("request %s %s: %v", method, path, err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	var out map[string]interface{}
	_ = json.Unmarshal(raw, &out)
	if out == nil {
		out = map[string]interface{}{"_raw": string(raw)}
	}
	return resp, out
}

func TestHealth(t *testing.T) {
	resp, body := doJSON(t, "GET", "/health", nil, "", nil)
	if resp.StatusCode != 200 {
		t.Fatalf("health status %d body %v", resp.StatusCode, body)
	}
	if body["status"] != "ok" {
		t.Fatalf("unexpected health body: %v", body)
	}
}

func TestReady(t *testing.T) {
	resp, body := doJSON(t, "GET", "/ready", nil, "", nil)
	if resp.StatusCode != 200 {
		t.Fatalf("ready status %d body %v", resp.StatusCode, body)
	}
}

func TestAnonymousCannotAccessMe(t *testing.T) {
	resp, _ := doJSON(t, "GET", "/api/v1/auth/me", nil, "", nil)
	if resp.StatusCode != 401 {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}
}

func TestLoginRejectsBadPassword(t *testing.T) {
	resp, body := doJSON(t, "POST", "/api/v1/auth/login", map[string]string{
		"identifier": "superadmin",
		"password":   "definitely-wrong",
	}, "", nil)
	if resp.StatusCode != 401 {
		t.Fatalf("expected 401, got %d body %v", resp.StatusCode, body)
	}
}

func TestLoginAndMe(t *testing.T) {
	resp, body := doJSON(t, "POST", "/api/v1/auth/login", map[string]string{
		"identifier": "superadmin",
		"password":   "SuperSecure1!",
	}, "", nil)
	if resp.StatusCode != 200 {
		t.Fatalf("login failed %d %v", resp.StatusCode, body)
	}
	token, _ := body["access_token"].(string)
	if token == "" {
		t.Fatal("missing access_token")
	}

	resp2, me := doJSON(t, "GET", "/api/v1/auth/me", nil, token, nil)
	if resp2.StatusCode != 200 {
		t.Fatalf("me failed %d %v", resp2.StatusCode, me)
	}
	if me["username"] != "superadmin" {
		t.Fatalf("unexpected user: %v", me)
	}
	if me["is_super_admin"] != true {
		t.Fatalf("expected super admin: %v", me)
	}
	perms, ok := me["permissions"].([]interface{})
	if !ok || len(perms) < 10 {
		t.Fatalf("expected many permissions, got %v", me["permissions"])
	}
}

func TestLogoutRevokesSession(t *testing.T) {
	resp, body := doJSON(t, "POST", "/api/v1/auth/login", map[string]string{
		"identifier": "superadmin",
		"password":   "SuperSecure1!",
	}, "", nil)
	if resp.StatusCode != 200 {
		t.Fatalf("login failed %d %v", resp.StatusCode, body)
	}
	token := body["access_token"].(string)

	resp2, _ := doJSON(t, "POST", "/api/v1/auth/logout", nil, token, nil)
	if resp2.StatusCode != 200 {
		t.Fatalf("logout failed %d", resp2.StatusCode)
	}

	resp3, _ := doJSON(t, "GET", "/api/v1/auth/me", nil, token, nil)
	if resp3.StatusCode != 401 {
		t.Fatalf("expected 401 after logout, got %d", resp3.StatusCode)
	}
}

func TestSetupDisabledAfterFirstAdmin(t *testing.T) {
	resp, body := doJSON(t, "POST", "/api/v1/auth/setup", map[string]string{
		"setup_token": "dev-setup-token-change-me",
		"username":    "anotheradmin",
		"email":       "another@example.com",
		"password":    "SuperSecure1!",
		"full_name":   "Another Admin",
	}, "", nil)
	if resp.StatusCode != 403 {
		t.Fatalf("expected 403 setup disabled, got %d %v", resp.StatusCode, body)
	}
}

func TestInvalidSetupToken(t *testing.T) {
	resp, body := doJSON(t, "POST", "/api/v1/auth/setup", map[string]string{
		"setup_token": "totally-wrong",
		"username":    "hackeruser",
		"email":       "h@example.com",
		"password":    "SuperSecure1!",
		"full_name":   "Hacker",
	}, "", nil)
	// After first admin, setup is disabled (403). Before first admin, invalid token is 401.
	if resp.StatusCode != 403 && resp.StatusCode != 401 {
		t.Fatalf("expected 401 or 403, got %d %v", resp.StatusCode, body)
	}
}
