package dto

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

func TestAccountFromServiceMaskedDoesNotExposeSecrets(t *testing.T) {
	reset := time.Now().Add(5 * time.Minute)
	proxyID := int64(9)
	notes := "internal note"
	account := &service.Account{
		ID: 42, Name: "shared-openai", Platform: service.PlatformOpenAI,
		Type: service.AccountTypeOAuth, Status: service.StatusActive,
		Schedulable: true, Concurrency: 8, Credentials: map[string]any{
			"access_token": "secret-token", "api_key": "secret-key",
		}, Extra: map[string]any{"base_url": "https://upstream.invalid"},
		ProxyID: &proxyID, Notes: &notes, ErrorMessage: "upstream 401 for user@example.com",
		TempUnschedulableReason: "quota exhausted for user@example.com",
		RateLimitedAt:           timePtr(time.Now().Add(-time.Minute)), RateLimitResetAt: &reset,
	}

	item := AccountFromServiceMasked(account)
	if item == nil || item.Name != "sh****ai" {
		t.Fatalf("non-owned account should be masked: %#v", item)
	}
	if item.OwnerUserID != nil || item.ProxyID != nil || item.Notes != nil {
		t.Fatalf("masked account must not carry owner, proxy or notes: %#v", item)
	}
	if item.ErrorMessage != "" || item.TempUnschedulableReason != "" {
		t.Fatalf("masked account must not carry raw error text: %#v", item)
	}
	if item.RateMultiplier != 0 {
		t.Fatalf("masked account must not carry the billing multiplier: %#v", item)
	}
	// The pool view is only useful if the real operational state survives.
	if !item.Schedulable || item.Concurrency != 8 || item.Status != service.StatusActive {
		t.Fatalf("masked account lost its pool state: %#v", item)
	}
	if item.RateLimitedAt == nil || item.RateLimitResetAt == nil {
		t.Fatalf("masked account lost its rate-limit timers: %#v", item)
	}

	payload, err := json.Marshal(item)
	if err != nil {
		t.Fatal(err)
	}
	body := string(payload)
	for _, secret := range []string{
		"secret-token", "secret-key", "upstream.invalid", "shared-openai",
		"user@example.com", "internal note",
	} {
		if strings.Contains(body, secret) {
			t.Fatalf("masked payload leaked %q: %s", secret, body)
		}
	}
}

func TestAccountListItemFromMaskedAccountStaysMasked(t *testing.T) {
	account := &service.Account{ID: 1, Name: "shared-account", Status: service.StatusActive, Schedulable: true}
	item := AccountListItemFromAccount(AccountFromServiceMasked(account))
	if item == nil || item.Name != "sh****nt" {
		t.Fatalf("list projection should stay masked: %#v", item)
	}
	if item.Proxy != nil || item.ProxyID != nil || item.Credentials != nil {
		t.Fatalf("list projection leaked proxy or credentials: %#v", item)
	}
}

func timePtr(value time.Time) *time.Time { return &value }
