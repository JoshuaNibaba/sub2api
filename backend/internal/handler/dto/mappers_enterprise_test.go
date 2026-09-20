package dto

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

func TestEnterpriseAccountPoolFromServiceDoesNotExposeSecrets(t *testing.T) {
	reset := time.Now().Add(5 * time.Minute)
	account := &service.Account{
		ID: 42, Name: "shared-openai", Platform: service.PlatformOpenAI,
		Type: service.AccountTypeOAuth, Status: service.StatusActive,
		Schedulable: true, Concurrency: 8, Credentials: map[string]any{
			"access_token": "secret-token", "api_key": "secret-key",
		}, Extra: map[string]any{"base_url": "https://upstream.invalid"},
		RateLimitedAt: timePtr(time.Now().Add(-time.Minute)), RateLimitResetAt: &reset,
	}

	item := EnterpriseAccountPoolFromService(account, time.Now())
	if item == nil || item.Health != "degraded" || !item.RateLimited {
		t.Fatalf("unexpected pool item: %#v", item)
	}
	payload, err := json.Marshal(item)
	if err != nil {
		t.Fatal(err)
	}
	body := string(payload)
	for _, secret := range []string{"secret-token", "secret-key", "upstream.invalid", "credentials", "proxy"} {
		if strings.Contains(body, secret) {
			t.Fatalf("enterprise payload leaked %q: %s", secret, body)
		}
	}
}

func timePtr(value time.Time) *time.Time { return &value }
