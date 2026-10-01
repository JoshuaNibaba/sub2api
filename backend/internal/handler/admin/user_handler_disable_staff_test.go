package admin

import (
	"net/http"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// 禁用账号的权限门控：不能禁用自己；禁用管理员账号仅限超级管理员。
func setupDisableStaffRouter(t *testing.T, actorRole string, actorID int64) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	router := gin.New()
	adminSvc := newStubAdminService()
	adminSvc.users = append(adminSvc.users,
		service.User{ID: 2, Email: "admin@example.com", Role: service.RoleAdmin, Status: service.StatusActive},
		service.User{ID: 3, Email: "root@example.com", Role: service.RoleSuperAdmin, Status: service.StatusActive},
	)
	h := NewUserHandler(adminSvc, nil, nil, nil, nil, nil, nil)
	router.PUT("/api/v1/admin/users/:id", func(c *gin.Context) {
		c.Set(string(middleware.ContextKeyUserRole), actorRole)
		c.Set(string(middleware.ContextKeyUser), middleware.AuthSubject{UserID: actorID})
		h.Update(c)
	})
	return router
}

func TestUpdateUserSuperAdminCanDisableStaff(t *testing.T) {
	router := setupDisableStaffRouter(t, service.RoleSuperAdmin, 99)

	for _, path := range []string{"/api/v1/admin/users/2", "/api/v1/admin/users/3"} {
		rec := doJSON(t, router, http.MethodPut, path, map[string]any{"status": "disabled"})
		require.Equal(t, http.StatusOK, rec.Code, path)
	}
}

func TestUpdateUserRestrictedAdminCannotDisableStaff(t *testing.T) {
	router := setupDisableStaffRouter(t, service.RoleAdmin, 99)

	for _, path := range []string{"/api/v1/admin/users/2", "/api/v1/admin/users/3"} {
		rec := doJSON(t, router, http.MethodPut, path, map[string]any{"status": "disabled"})
		require.Equal(t, http.StatusForbidden, rec.Code, path)
	}

	rec := doJSON(t, router, http.MethodPut, "/api/v1/admin/users/1", map[string]any{"status": "disabled"})
	require.Equal(t, http.StatusOK, rec.Code, "受限管理员仍可禁用普通用户")
}

func TestUpdateUserCannotDisableSelf(t *testing.T) {
	router := setupDisableStaffRouter(t, service.RoleSuperAdmin, 3)

	rec := doJSON(t, router, http.MethodPut, "/api/v1/admin/users/3", map[string]any{"status": "disabled"})
	require.Equal(t, http.StatusBadRequest, rec.Code)
}
