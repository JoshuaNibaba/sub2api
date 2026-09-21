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

	item := EnterpriseAccountPoolFromService(account, time.Now(), 7, false)
	if item == nil || item.Health != "degraded" || !item.RateLimited {
		t.Fatalf("unexpected pool item: %#v", item)
	}
	if item.Name != "sh****ai" || item.OwnedByViewer {
		t.Fatalf("non-owned account should be masked: %#v", item)
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

func TestEnterpriseAccountPoolFromServiceKeepsOwnedName(t *testing.T) {
	owner := int64(7)
	account := &service.Account{ID: 1, Name: "owned-account", OwnerUserID: &owner, Status: service.StatusActive, Schedulable: true}
	item := EnterpriseAccountPoolFromService(account, time.Now(), owner, false)
	if item == nil || item.Name != "owned-account" || !item.OwnedByViewer {
		t.Fatalf("owned account should remain visible: %#v", item)
	}
}

func timePtr(value time.Time) *time.Time { return &value }
