package middleware

import (
	"github.com/Wei-Shaw/sub2api/internal/domain"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

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
