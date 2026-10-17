package routes

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/handler"
	adminhandler "github.com/Wei-Shaw/sub2api/internal/handler/admin"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestAntigravityOAuthBootstrapRoutesPermissions(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oauth := service.NewAntigravityOAuthService(nil)
	defer oauth.Stop()
	h := &handler.Handlers{Admin: &handler.AdminHandlers{
		AntigravityOAuth: adminhandler.NewAntigravityOAuthHandler(oauth),
	}}
	for _, role := range []string{service.RoleAdmin, service.RoleSuperAdmin, service.RoleUser, service.RoleEnterpriseUser, ""} {
		for _, path := range []string{"/auth-url", "/exchange-code", "/refresh-token"} {
			t.Run(role+path, func(t *testing.T) {
				router := gin.New()
				router.Use(func(c *gin.Context) {
					if role != "" {
						c.Set(string(middleware.ContextKeyUserRole), role)
						c.Set(string(middleware.ContextKeyUser), middleware.AuthSubject{UserID: 7})
					}
					c.Next()
				})
				registerAntigravityOAuthRoutes(router.Group("/api/v1/admin"), h)
				w := httptest.NewRecorder()
				req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/antigravity/oauth"+path, strings.NewReader(`{}`))
				req.Header.Set("Content-Type", "application/json")
				router.ServeHTTP(w, req)
				want := http.StatusForbidden
				switch role {
				case "":
					want = http.StatusUnauthorized
				case service.RoleAdmin, service.RoleSuperAdmin:
					// Validation must run after permission checks without calling Google.
					want = http.StatusBadRequest
					if path == "/auth-url" {
						want = http.StatusOK
						require.Contains(t, w.Body.String(), "auth_url")
					}
				}
				require.Equal(t, want, w.Code)
			})
		}
	}
}
