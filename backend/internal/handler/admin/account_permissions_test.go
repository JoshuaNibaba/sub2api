package admin

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
	servermiddleware "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestAccountResponsesMaskNonOwnedProxyForRestrictedAdmin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, role := range []string{service.RoleAdmin, service.RoleSuperAdmin} {
		for _, ownerID := range []int64{0, 7, 9} {
			for _, target := range []string{"/api/v1/admin/accounts/42", "/api/v1/admin/accounts", "/api/v1/admin/accounts?lite=1"} {
				proxyID := int64(27)
				account := service.Account{
					ID: 42, Name: "shared-account", Platform: service.PlatformOpenAI,
					Type: service.AccountTypeOAuth, Status: service.StatusActive,
					ProxyID: &proxyID,
					Proxy: &service.Proxy{
						ID: proxyID, Name: "proxy", Protocol: "socks5h", Host: "proxy.example.com", Port: 1080,
						Username: "proxy-user", Password: "proxy-password",
					},
				}
				if ownerID > 0 {
					account.OwnerUserID = &ownerID
				}
				svc := newStubAdminService()
				svc.accounts = []service.Account{account}
				svc.getAccountResult = &account
				handler := NewAccountHandler(svc, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
				router := gin.New()
				router.Use(func(c *gin.Context) {
					c.Set(string(servermiddleware.ContextKeyUserRole), role)
					c.Set(string(servermiddleware.ContextKeyUser), servermiddleware.AuthSubject{UserID: 7})
					c.Next()
				})
				group := router.Group("/api/v1/admin/accounts", servermiddleware.RequireAccountRouteAccess(handler))
				group.GET("", handler.List)
				group.GET("/:id", handler.GetByID)
				w := httptest.NewRecorder()
				router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, target, nil))
				require.Equal(t, http.StatusOK, w.Code)
				var payload struct {
					Data struct {
						Proxy *dto.Proxy `json:"proxy"`
						Items []struct {
							Proxy *dto.Proxy `json:"proxy"`
						} `json:"items"`
					} `json:"data"`
				}
				require.NoError(t, json.Unmarshal(w.Body.Bytes(), &payload))
				proxy := payload.Data.Proxy
				if len(payload.Data.Items) > 0 {
					proxy = payload.Data.Items[0].Proxy
				}
				require.NotNil(t, proxy, "role=%s owner=%d target=%s", role, ownerID, target)
				if role == service.RoleAdmin && ownerID != 7 {
					require.Equal(t, dto.MaskProxyValue(account.Proxy.Host), proxy.Host)
					require.Equal(t, dto.MaskProxyValue(account.Proxy.Username), proxy.Username)
					require.Empty(t, proxy.Password)
				} else {
					require.Equal(t, account.Proxy.Host, proxy.Host)
				}
				require.Equal(t, "proxy.example.com", account.Proxy.Host, "masking must not mutate the stored account")
			}
		}
	}
}

func TestAntigravityDefaultModelMappingPermissions(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, role := range []string{service.RoleAdmin, service.RoleSuperAdmin, service.RoleUser, service.RoleEnterpriseUser, ""} {
		t.Run(role, func(t *testing.T) {
			router := gin.New()
			router.Use(func(c *gin.Context) {
				c.Set(string(servermiddleware.ContextKeyUserRole), role)
				c.Next()
			})
			handler := &AccountHandler{}
			group := router.Group("/api/v1/admin/accounts", servermiddleware.RequireAccountRouteAccess(nil))
			group.GET("/antigravity/default-model-mapping", handler.GetAntigravityDefaultModelMapping)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/admin/accounts/antigravity/default-model-mapping", nil))
			want := http.StatusForbidden
			if service.IsStaffRole(role) {
				want = http.StatusOK
				require.Contains(t, w.Body.String(), "claude")
			} else if role == "" {
				want = http.StatusUnauthorized
			}
			require.Equal(t, want, w.Code)
		})
	}
}
