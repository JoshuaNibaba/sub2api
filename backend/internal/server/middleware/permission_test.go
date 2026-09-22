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
