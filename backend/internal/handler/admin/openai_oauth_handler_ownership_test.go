package admin

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/pkg/openai"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type openAIOwnershipAdminStub struct {
	service.AdminService
	input *service.CreateAccountInput
}

func (s *openAIOwnershipAdminStub) CreateAccount(_ context.Context, input *service.CreateAccountInput) (*service.Account, error) {
	s.input = input
	return &service.Account{ID: 42, Name: input.Name, Platform: input.Platform, Type: input.Type, OwnerUserID: input.OwnerUserID}, nil
}

type openAIOwnershipOAuthStub struct{}

func (openAIOwnershipOAuthStub) ExchangeCode(context.Context, string, string, string, string, string) (*openai.TokenResponse, error) {
	return &openai.TokenResponse{AccessToken: "test-access-token", RefreshToken: "test-refresh-token", ExpiresIn: 3600}, nil
}

func (openAIOwnershipOAuthStub) RefreshToken(context.Context, string, string) (*openai.TokenResponse, error) {
	return &openai.TokenResponse{}, nil
}

func (openAIOwnershipOAuthStub) RefreshTokenWithClientID(context.Context, string, string, string) (*openai.TokenResponse, error) {
	return &openai.TokenResponse{}, nil
}

func TestOpenAICreateAccountFromOAuthAssignsOperatorOwnership(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, role := range []string{service.RoleAdmin, service.RoleSuperAdmin} {
		t.Run(role, func(t *testing.T) {
			oauth := service.NewOpenAIOAuthService(nil, openAIOwnershipOAuthStub{})
			defer oauth.Stop()
			auth, err := oauth.GenerateAuthURL(context.Background(), nil, "", service.PlatformOpenAI)
			require.NoError(t, err)
			authURL, err := url.Parse(auth.AuthURL)
			require.NoError(t, err)
			body, err := json.Marshal(map[string]any{
				"session_id": auth.SessionID, "code": "test-code", "state": authURL.Query().Get("state"),
				"owner_user_id": 99,
			})
			require.NoError(t, err)
			admin := &openAIOwnershipAdminStub{}
			h := NewOpenAIOAuthHandler(oauth, admin, nil, nil)
			router := gin.New()
			router.Use(func(c *gin.Context) {
				c.Set(string(middleware.ContextKeyUserRole), role)
				c.Set(string(middleware.ContextKeyUser), middleware.AuthSubject{UserID: 7})
				c.Next()
			})
			router.POST("/api/v1/admin/openai/create-from-oauth", middleware.RequireOpenAIOAuthRouteAccess(), h.CreateAccountFromOAuth)
			w := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodPost, "/api/v1/admin/openai/create-from-oauth", strings.NewReader(string(body)))
			request.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(w, request)
			require.Equal(t, http.StatusOK, w.Code, w.Body.String())
			require.NotNil(t, admin.input)
			require.NotNil(t, admin.input.OwnerUserID)
			require.Equal(t, int64(7), *admin.input.OwnerUserID)
		})
	}
}
