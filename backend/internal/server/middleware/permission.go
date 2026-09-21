package middleware

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/domain"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

// AccountOwnershipResolver is implemented by the account handler. Keeping
// this tiny interface here avoids coupling middleware to handler packages.
type AccountOwnershipResolver interface {
	IsAccountOwnedBy(ctx context.Context, accountID, userID int64) (bool, error)
}

// RequireAccountRouteAccess gives super administrators the full account API,
// while restricted administrators may create accounts and mutate only their
// explicitly owned account via the canonical PUT/DELETE endpoint. Other
// account operations (credential imports, bulk actions, probing, refresh,
// export, etc.) remain super-admin-only.
func RequireAccountRouteAccess(resolver AccountOwnershipResolver) gin.HandlerFunc {
	return func(c *gin.Context) {
		role, ok := GetUserRoleFromContext(c)
		if !ok || role == "" {
			AbortWithError(c, http.StatusUnauthorized, "UNAUTHORIZED", "User not found in context")
			return
		}
		if c.Request.Method == http.MethodGet || c.Request.Method == http.MethodHead {
			if !service.HasPermission(role, service.PermissionAdminAccountsRead) {
				AbortWithError(c, http.StatusForbidden, "FORBIDDEN", "Permission denied")
				return
			}
			if !service.IsSuperAdminRole(role) && strings.HasSuffix(c.FullPath(), "/admin/accounts/data") {
				AbortWithError(c, http.StatusForbidden, "FORBIDDEN", "Account export requires a super administrator")
				return
			}
			c.Next()
			return
		}
		if service.HasPermission(role, service.PermissionAdminAccountsWrite) {
			c.Next()
			return
		}
		if !service.HasPermission(role, service.PermissionAdminAccountsOwnedWrite) {
			AbortWithError(c, http.StatusForbidden, "FORBIDDEN", "Permission denied")
			return
		}

		path := c.FullPath()
		// These POST endpoints are read-only previews/batch queries used by the
		// account UI and must remain available to restricted administrators.
		for _, readOnlyPath := range []string{
			"/admin/accounts/check-mixed-channel",
			"/admin/accounts/sync/crs/preview",
			"/admin/accounts/models/sync-upstream-preview",
			"/admin/accounts/usage/batch",
			"/admin/accounts/today-stats/batch",
		} {
			if strings.HasSuffix(path, readOnlyPath) {
				c.Next()
				return
			}
		}
		if c.Request.Method == http.MethodPost && strings.HasSuffix(path, "/admin/accounts") {
			c.Next()
			return
		}
		if (c.Request.Method != http.MethodPut && c.Request.Method != http.MethodDelete) ||
			!strings.HasSuffix(path, "/admin/accounts/:id") {
			AbortWithError(c, http.StatusForbidden, "FORBIDDEN", "Only owned accounts can be modified")
			return
		}
		if resolver == nil {
			AbortWithError(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "Account ownership service unavailable")
			return
		}
		accountID, err := strconv.ParseInt(c.Param("id"), 10, 64)
		subject, subjectOK := GetAuthSubjectFromContext(c)
		if err != nil || !subjectOK || subject.UserID <= 0 {
			AbortWithError(c, http.StatusForbidden, "FORBIDDEN", "Account ownership could not be verified")
			return
		}
		owned, err := resolver.IsAccountOwnedBy(c.Request.Context(), accountID, subject.UserID)
		if err != nil {
			AbortWithError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Account ownership could not be verified")
			return
		}
		if !owned {
			AbortWithError(c, http.StatusForbidden, "FORBIDDEN", "Only the account owner can modify this account")
			return
		}
		c.Next()
	}
}

// RequirePermission enforces a capability after an authentication middleware
// has populated the user's role in the Gin context. Keeping this check separate
// from adminAuth lets restricted administrators and enterprise users share the
// same policy mechanism without widening the admin route surface.
func RequirePermission(permission domain.Permission) gin.HandlerFunc {
	return func(c *gin.Context) {
		role, ok := GetUserRoleFromContext(c)
		if !ok || role == "" {
			AbortWithError(c, 401, "UNAUTHORIZED", "User not found in context")
			return
		}
		if !service.HasPermission(role, permission) {
			AbortWithError(c, 403, "FORBIDDEN", "Permission denied")
			return
		}
		c.Next()
	}
}

// RequireAnyPermission grants access when the current role has at least one of
// the listed capabilities. It is useful for shared read routes exposed to
// different staff profiles.
func RequireAnyPermission(permissions ...domain.Permission) gin.HandlerFunc {
	return func(c *gin.Context) {
		role, ok := GetUserRoleFromContext(c)
		if !ok || role == "" {
			AbortWithError(c, 401, "UNAUTHORIZED", "User not found in context")
			return
		}
		for _, permission := range permissions {
			if service.HasPermission(role, permission) {
				c.Next()
				return
			}
		}
		AbortWithError(c, 403, "FORBIDDEN", "Permission denied")
	}
}

// RequireReadWritePermission selects the read capability for safe methods and
// the write capability for mutations. This keeps one route group readable by
// a restricted administrator without accidentally granting its mutations.
func RequireReadWritePermission(read, write domain.Permission) gin.HandlerFunc {
	return func(c *gin.Context) {
		permission := read
		switch c.Request.Method {
		case "POST", "PUT", "PATCH", "DELETE":
			permission = write
		}
		RequirePermission(permission)(c)
	}
}
