package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type accountOwnershipStub struct{ owned bool }

func (s accountOwnershipStub) IsAccountOwnedBy(context.Context, int64, int64) (bool, error) {
	return s.owned, nil
}

func TestRequirePermissionUsesRoleMatrix(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tests := []struct {
		name   string
		role   string
		status int
	}{
		{name: "enterprise allowed", role: service.RoleEnterpriseUser, status: http.StatusOK},
		{name: "ordinary user denied", role: service.RoleUser, status: http.StatusForbidden},
		{name: "missing role unauthorized", role: "", status: http.StatusUnauthorized},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			r.Use(func(c *gin.Context) {
				if tt.role != "" {
					c.Set(string(ContextKeyUserRole), tt.role)
				}
				c.Next()
			})
			r.GET("/enterprise", RequirePermission(service.PermissionEnterpriseAccountPool), func(c *gin.Context) {
				c.Status(http.StatusOK)
			})

			w := httptest.NewRecorder()
			r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/enterprise", nil))
			require.Equal(t, tt.status, w.Code)
		})
	}
}

func TestEnterpriseRoleCannotEnterAdminPanelPermission(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(string(ContextKeyUserRole), service.RoleEnterpriseUser)
		c.Next()
	})
	r.GET("/admin", RequirePermission(service.PermissionAdminPanel), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/admin", nil))
	require.Equal(t, http.StatusForbidden, w.Code)
}

func TestRequireAccountRouteAccessRestrictsAdminMutationsToOwnedAccount(t *testing.T) {
	gin.SetMode(gin.TestMode)
	request := func(owned bool) int {
		r := gin.New()
		r.Use(func(c *gin.Context) {
			c.Set(string(ContextKeyUserRole), service.RoleAdmin)
			c.Set(string(ContextKeyUser), AuthSubject{UserID: 7})
			c.Next()
		})
		group := r.Group("/api/v1/admin/accounts")
		group.Use(RequireAccountRouteAccess(accountOwnershipStub{owned: owned}))
		group.PUT("/:id", func(c *gin.Context) { c.Status(http.StatusOK) })
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodPut, "/api/v1/admin/accounts/42", nil))
		return w.Code
	}
	require.Equal(t, http.StatusOK, request(true))
	require.Equal(t, http.StatusForbidden, request(false))
}

// A restricted administrator owns accounts, so the per-account operations the
// account page offers must work on them — while the credential-minting ones and
// anything addressing no single account must not.
func TestRequireAccountRouteAccessAllowsOwnedAccountOperations(t *testing.T) {
	gin.SetMode(gin.TestMode)
	request := func(owned bool, method, route, target string) int {
		r := gin.New()
		r.Use(func(c *gin.Context) {
			c.Set(string(ContextKeyUserRole), service.RoleAdmin)
			c.Set(string(ContextKeyUser), AuthSubject{UserID: 7})
			c.Next()
		})
		group := r.Group("/api/v1/admin/accounts")
		group.Use(RequireAccountRouteAccess(accountOwnershipStub{owned: owned}))
		group.Handle(method, route, func(c *gin.Context) { c.Status(http.StatusOK) })
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(method, "/api/v1/admin/accounts"+target, nil))
		return w.Code
	}
	owned := []struct {
		method string
		route  string
		target string
	}{
		{http.MethodPost, "/:id/test", "/42/test"},
		{http.MethodPost, "/:id/clear-error", "/42/clear-error"},
		{http.MethodPost, "/:id/clear-rate-limit", "/42/clear-rate-limit"},
		{http.MethodPost, "/:id/recover-state", "/42/recover-state"},
		{http.MethodPost, "/:id/reset-quota", "/42/reset-quota"},
		{http.MethodPost, "/:id/schedulable", "/42/schedulable"},
		{http.MethodPost, "/:id/models/sync-upstream", "/42/models/sync-upstream"},
		{http.MethodDelete, "/:id/temp-unschedulable", "/42/temp-unschedulable"},
		{http.MethodPut, "/:id/upstream-billing-probe", "/42/upstream-billing-probe"},
	}
	for _, tt := range owned {
		require.Equal(t, http.StatusOK, request(true, tt.method, tt.route, tt.target), tt.target)
		require.Equal(t, http.StatusForbidden, request(false, tt.method, tt.route, tt.target), tt.target)
	}

	// Credential-minting per-account endpoints stay closed even to the owner.
	require.Equal(t, http.StatusForbidden, request(true, http.MethodPost, "/:id/duplicate", "/42/duplicate"))
	require.Equal(t, http.StatusForbidden, request(true, http.MethodPost, "/:id/apply-oauth-credentials", "/42/apply-oauth-credentials"))
	require.Equal(t, http.StatusForbidden, request(true, http.MethodPost, "/:id/shadow", "/42/shadow"))
	// So do the endpoints that address no single account.
	require.Equal(t, http.StatusForbidden, request(true, http.MethodPost, "/bulk-update", "/bulk-update"))
	require.Equal(t, http.StatusForbidden, request(true, http.MethodPost, "/batch-delete", "/batch-delete"))
	require.Equal(t, http.StatusForbidden, request(true, http.MethodPost, "/sync/crs", "/sync/crs"))
	require.Equal(t, http.StatusForbidden, request(true, http.MethodPost, "/data", "/data"))
	// The read-only preview/batch POSTs the account page needs stay open.
	require.Equal(t, http.StatusOK, request(false, http.MethodPost, "/usage/batch", "/usage/batch"))
	require.Equal(t, http.StatusOK, request(false, http.MethodPost, "/today-stats/batch", "/today-stats/batch"))
}

func TestRequireAccountRouteAccessDeniesExportToRestrictedAdmin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(string(ContextKeyUserRole), service.RoleAdmin)
		c.Set(string(ContextKeyUser), AuthSubject{UserID: 7})
		c.Next()
	})
	group := r.Group("/api/v1/admin/accounts")
	group.Use(RequireAccountRouteAccess(accountOwnershipStub{owned: true}))
	group.GET("/data", func(c *gin.Context) { c.Status(http.StatusOK) })
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/admin/accounts/data", nil))
	require.Equal(t, http.StatusForbidden, w.Code)
	require.Contains(t, w.Body.String(), "super administrator")
}

// The provider groups (openai/grok/cn-providers) are reachable with
// admin.accounts.read alone, so their account-scoped reads need the same
// ownership rule the /admin/accounts group applies.
func TestRequireOwnedAccountParamGuardsProviderRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	request := func(role string, owned bool, path string) int {
		r := gin.New()
		r.Use(func(c *gin.Context) {
			c.Set(string(ContextKeyUserRole), role)
			c.Set(string(ContextKeyUser), AuthSubject{UserID: 7})
			c.Next()
		})
		group := r.Group("/api/v1/admin/cn-providers")
		group.Use(RequireOwnedAccountParam(accountOwnershipStub{owned: owned}))
		group.GET("/accounts/:id/balance", func(c *gin.Context) { c.Status(http.StatusOK) })
		group.GET("/runtime-sanity", func(c *gin.Context) { c.Status(http.StatusOK) })
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/admin/cn-providers"+path, nil))
		return w.Code
	}
	require.Equal(t, http.StatusForbidden, request(service.RoleAdmin, false, "/accounts/42/balance"))
	require.Equal(t, http.StatusOK, request(service.RoleAdmin, true, "/accounts/42/balance"))
	require.Equal(t, http.StatusOK, request(service.RoleSuperAdmin, false, "/accounts/42/balance"))
	// Routes in the same group that address no account keep working.
	require.Equal(t, http.StatusOK, request(service.RoleAdmin, false, "/runtime-sanity"))
}

func TestRequireOwnedAccountViaResolvesParentAccount(t *testing.T) {
	gin.SetMode(gin.TestMode)
	request := func(owned bool, resolve RouteAccountResolver) int {
		r := gin.New()
		r.Use(func(c *gin.Context) {
			c.Set(string(ContextKeyUserRole), service.RoleAdmin)
			c.Set(string(ContextKeyUser), AuthSubject{UserID: 7})
			c.Next()
		})
		r.GET("/plans/:id/results",
			RequireOwnedAccountVia(resolve, accountOwnershipStub{owned: owned}),
			func(c *gin.Context) { c.Status(http.StatusOK) })
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/plans/5/results", nil))
		return w.Code
	}
	toAccount := func(context.Context, int64) (int64, error) { return 42, nil }
	require.Equal(t, http.StatusOK, request(true, toAccount))
	require.Equal(t, http.StatusForbidden, request(false, toAccount))
	// An unresolvable plan must fail closed rather than fall through.
	require.Equal(t, http.StatusForbidden, request(true, func(context.Context, int64) (int64, error) { return 0, nil }))
}
