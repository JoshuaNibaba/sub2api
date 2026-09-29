package routes

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/handler"
	adminhandler "github.com/Wei-Shaw/sub2api/internal/handler/admin"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestOpenAIOAuthBootstrapRoutesPermissions(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oauth := service.NewOpenAIOAuthService(nil, nil)
	defer oauth.Stop()
	h := &handler.Handlers{Admin: &handler.AdminHandlers{
		OpenAIOAuth: adminhandler.NewOpenAIOAuthHandler(oauth, nil, nil, nil),
	}}
	for _, role := range []string{service.RoleAdmin, service.RoleSuperAdmin, service.RoleUser, service.RoleEnterpriseUser, ""} {
		for _, path := range []string{"/generate-auth-url", "/exchange-code", "/refresh-token", "/create-from-oauth", "/create-from-codex-pat"} {
			t.Run(role+path, func(t *testing.T) {
				router := gin.New()
				router.Use(func(c *gin.Context) {
					if role != "" {
						c.Set(string(middleware.ContextKeyUserRole), role)
						c.Set(string(middleware.ContextKeyUser), middleware.AuthSubject{UserID: 7})
					}
					c.Next()
				})
				registerOpenAIOAuthRoutes(router.Group("/api/v1/admin"), h)
				w := httptest.NewRecorder()
				router.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/v1/admin/openai"+path, nil))
				want := http.StatusForbidden
				switch role {
				case "":
					want = http.StatusUnauthorized
				case service.RoleAdmin, service.RoleSuperAdmin:
					// An empty request must reach validation instead of permission denial.
					want = http.StatusBadRequest
					if path == "/generate-auth-url" {
						want = http.StatusOK
						require.Contains(t, w.Body.String(), "auth_url")
					}
				}
				require.Equal(t, want, w.Code, w.Body.String())
			})
		}
	}
}

func TestOpenAIOAuthExistingAccountWritesRemainRestricted(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(string(middleware.ContextKeyUserRole), service.RoleAdmin)
		c.Set(string(middleware.ContextKeyUser), middleware.AuthSubject{UserID: 7})
		c.Next()
	})
	h := &handler.Handlers{Admin: &handler.AdminHandlers{
		OpenAIOAuth: adminhandler.NewOpenAIOAuthHandler(nil, nil, nil, nil),
	}}
	registerOpenAIOAuthRoutes(router.Group("/api/v1/admin"), h)
	for _, path := range []string{
		"/accounts/42/refresh", "/accounts/42/quota/refresh", "/accounts/42/reset-quota",
		"/accounts/42/referrals/refresh", "/accounts/42/referrals/invite",
	} {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/v1/admin/openai"+path, nil))
		require.Equal(t, http.StatusForbidden, w.Code, path)
	}
}
