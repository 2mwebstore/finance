package controllers

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/cambodia-finance/income-expense/internal/accounting"
	"github.com/cambodia-finance/income-expense/internal/services"
	"github.com/gin-gonic/gin"
)

func bindResponse(t *testing.T, dst interface{}, body string) (int, map[string]interface{}) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	if bindJSON(c, dst) {
		return http.StatusOK, nil
	}
	var out map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return w.Code, out
}

func TestBindJSONReportsReadableFieldErrors(t *testing.T) {
	var in services.CreateTransactionInput
	code, out := bindResponse(t, &in, `{"type":"expense","currency":"USD"}`)
	if code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", code)
	}
	fields, _ := out["fields"].(map[string]interface{})
	if fields["branch_id"] != "branch is required" || fields["amount"] != "amount is required" {
		t.Fatalf("fields = %v", fields)
	}
	if out["error"] == "" || strings.Contains(out["error"].(string), "Key:") {
		t.Fatalf("error = %q, want a readable message", out["error"])
	}
}

func TestBindJSONMessages(t *testing.T) {
	cases := []struct {
		name  string
		dst   interface{}
		body  string
		field string
		want  string
	}{
		{"min length", &services.CreateBranchInput{}, `{"code":"A","name":"Main"}`, "code", "code must be at least 2 characters"},
		{"email", &services.CreateUserInput{}, `{"username":"dara","email":"nope","password":"1234","full_name":"Dara"}`, "email", "email must be a valid email address"},
		{"pin digits", &services.CreateUserInput{}, `{"username":"dara","email":"d@x.io","password":"12ab","full_name":"Dara"}`, "password", "PIN must contain digits only"},
		{"pin length", &services.CreateUserInput{}, `{"username":"dara","email":"d@x.io","password":"12","full_name":"Dara"}`, "password", "PIN must be exactly 4 digits"},
		{"oneof", &services.CreateTransactionInput{}, `{"type":"gift","branch_id":"7a1f0c39-6c41-4f0e-9a52-2f3a9f6f4c11","currency":"USD","amount":"1"}`, "type", "type must be income or expense"},
		{"blank update", &services.UpdateBranchInput{}, `{"name":""}`, "name", "name must be at least 2 characters"},
		{"wrong type", &services.CreateTransactionInput{}, `{"amount":12}`, "amount", "amount is not valid"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			code, out := bindResponse(t, tc.dst, tc.body)
			if code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400", code)
			}
			fields, _ := out["fields"].(map[string]interface{})
			if fields[tc.field] != tc.want {
				t.Fatalf("fields[%s] = %v, want %q (all: %v)", tc.field, fields[tc.field], tc.want, fields)
			}
		})
	}
}

func TestBindJSONAcceptsValidBodyAndOmittedUpdates(t *testing.T) {
	if code, out := bindResponse(t, &services.UpdateBranchInput{}, `{"address":"Street 1"}`); code != http.StatusOK {
		t.Fatalf("status = %d (%v), want 200", code, out)
	}
	if code, out := bindResponse(t, &services.CreateBranchInput{}, `{"code":"PP","name":"Main","email":""}`); code != http.StatusOK {
		t.Fatalf("status = %d (%v), want 200", code, out)
	}
}

func TestBindJSONEmptyBody(t *testing.T) {
	code, out := bindResponse(t, &services.CreateBranchInput{}, ``)
	if code != http.StatusBadRequest || out["error"] != "request body is required" {
		t.Fatalf("got %d %v", code, out)
	}
}

func TestErrorFieldsPointAtTheInput(t *testing.T) {
	if _, err := accounting.ParseAmount("abc"); errorFields(err)["amount"] != "amount must be a positive number with at most 4 decimal places" {
		t.Fatalf("amount error fields = %v", errorFields(err))
	}
	if got := errorFields(services.ErrCategoryBranch); got["category_id"] == nil {
		t.Fatalf("category branch error fields = %v", got)
	}
	if got := errorFields(errors.New("something else")); got != nil {
		t.Fatalf("unrelated error fields = %v, want nil", got)
	}
}
