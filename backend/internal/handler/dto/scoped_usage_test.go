package dto

import (
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

func TestScopedUsageLogMasksNonOwnedAccountName(t *testing.T) {
	owner := int64(99)
	log := &service.UsageLog{
		ID: 1, UserID: 7, AccountID: 42,
		Account: &service.Account{ID: 42, Name: "provider-account", OwnerUserID: &owner},
	}
	got := ScopedUsageLogFromService(log, 7)
	if got == nil || got.Account == nil {
		t.Fatal("expected account summary")
	}
	if got.Account.Name != "pr****nt" || got.AccountOwnedByUser {
		t.Fatalf("unexpected non-owned account projection: %#v", got)
	}
}

func TestScopedUsageLogKeepsOwnedAccountName(t *testing.T) {
	owner := int64(7)
	log := &service.UsageLog{
		ID: 1, UserID: 7, AccountID: 42,
		Account: &service.Account{ID: 42, Name: "owned-account", OwnerUserID: &owner},
	}
	got := ScopedUsageLogFromService(log, 7)
	if got == nil || got.Account == nil || got.Account.Name != "owned-account" || !got.AccountOwnedByUser {
		t.Fatalf("unexpected owned account projection: %#v", got)
	}
}
