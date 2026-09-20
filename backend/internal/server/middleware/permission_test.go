package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

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

