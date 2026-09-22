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
// while restricted administrators may create accounts and operate only on the
// accounts they own, through any /admin/accounts/:id... endpoint. Operations
// that address no single account (bulk updates, batch credential writes,
// CRS sync, data import/export) and the credential-minting per-account
// endpoints listed below remain super-admin-only.
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
			if !service.IsSuperAdminRole(role) {
				path := c.FullPath()
				if strings.HasSuffix(path, "/admin/accounts") {
					c.Next()
					return
				}
				if strings.HasSuffix(path, "/admin/accounts/data") {
					AbortWithError(c, http.StatusForbidden, "FORBIDDEN", "Account export requires a super administrator")
					return
				}
				if strings.Contains(path, "/admin/accounts/:id") {
					if !accountRouteOwnerAllowed(c, resolver) {
						return
					}
					c.Next()
					return
				}
				AbortWithError(c, http.StatusForbidden, "FORBIDDEN", "Only owned account details are available")
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
		// Owning an account means being able to run it, not just rename it: the
		// row actions on the account page (test, clear error, recover state,
		// reset quota, schedulability, model sync, probe toggles) all address
		// /admin/accounts/:id/..., and a restricted administrator who can PUT
		// and DELETE an account but cannot clear its rate limit is unable to do
		// the job the role exists for. Bulk, import/export and sync endpoints
		// address no single account and stay out of reach.
		if !strings.Contains(path, "/admin/accounts/:id") {
			AbortWithError(c, http.StatusForbidden, "FORBIDDEN", "Only owned accounts can be modified")
			return
		}
		// Exceptions that stay super-admin-only even for the owner: each one
		// either writes fresh credential material or spawns a second account
		// from the source's credentials, and account creation from credentials
		// is gated on admin.credentials.read, which this role does not hold.
		for _, superAdminOnly := range []string{
			"/admin/accounts/:id/duplicate",
			"/admin/accounts/:id/apply-oauth-credentials",
			"/admin/accounts/:id/shadow",
		} {
			if strings.HasSuffix(path, superAdminOnly) {
				AbortWithError(c, http.StatusForbidden, "FORBIDDEN", "Operation requires a super administrator")
				return
			}
		}
		if resolver == nil {
			AbortWithError(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "Account ownership service unavailable")
			return
		}
		if !accountRouteOwnerAllowed(c, resolver) {
			return
		}
		c.Next()
	}
}

// RequireOwnedAccountParam guards account-scoped routes that live outside the
// /admin/accounts group — the provider-specific quota, balance and plan
// endpoints. Those groups are reachable with admin.accounts.read alone, so
// without this a restricted administrator could read upstream quota, balance
// and scheduled-test state for accounts owned by somebody else, and trigger
// live upstream probes with that owner's credentials. Routes in the group that
// carry no :id (OAuth helpers, runtime checks) are left to the group's own
// permission middleware.
func RequireOwnedAccountParam(resolver AccountOwnershipResolver) gin.HandlerFunc {
	return func(c *gin.Context) {
		role, ok := GetUserRoleFromContext(c)
		if !ok || role == "" {
			AbortWithError(c, http.StatusUnauthorized, "UNAUTHORIZED", "User not found in context")
			return
		}
		if service.IsSuperAdminRole(role) || c.Param("id") == "" {
			c.Next()
			return
		}
		if !accountRouteOwnerAllowed(c, resolver) {
			return
		}
		c.Next()
	}
}

// RouteAccountResolver maps a route's :id to the account that owns the
// addressed resource, for routes where :id is not itself an account id.
type RouteAccountResolver func(ctx context.Context, routeID int64) (int64, error)

// RequireOwnedAccountVia is RequireOwnedAccountParam for routes keyed by a
// child resource — a scheduled-test plan id, for example. The plan itself
// carries no permission of its own, so its results inherit the ownership rule
// of the account it tests.
func RequireOwnedAccountVia(resolve RouteAccountResolver, resolver AccountOwnershipResolver) gin.HandlerFunc {
	return func(c *gin.Context) {
		role, ok := GetUserRoleFromContext(c)
		if !ok || role == "" {
			AbortWithError(c, http.StatusUnauthorized, "UNAUTHORIZED", "User not found in context")
			return
		}
		if service.IsSuperAdminRole(role) {
			c.Next()
			return
		}
		if resolve == nil {
			AbortWithError(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "Account ownership service unavailable")
			return
		}
		routeID, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			AbortWithError(c, http.StatusForbidden, "FORBIDDEN", "Account ownership could not be verified")
			return
		}
		accountID, err := resolve(c.Request.Context(), routeID)
		if err != nil || accountID <= 0 {
			AbortWithError(c, http.StatusForbidden, "FORBIDDEN", "Account ownership could not be verified")
			return
		}
		if !accountOwnerAllowed(c, accountID, resolver) {
			return
		}
		c.Next()
	}
}

func accountRouteOwnerAllowed(c *gin.Context, resolver AccountOwnershipResolver) bool {
	accountID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		AbortWithError(c, http.StatusForbidden, "FORBIDDEN", "Account ownership could not be verified")
		return false
	}
	return accountOwnerAllowed(c, accountID, resolver)
}

func accountOwnerAllowed(c *gin.Context, accountID int64, resolver AccountOwnershipResolver) bool {
	if resolver == nil {
		AbortWithError(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "Account ownership service unavailable")
		return false
	}
	subject, subjectOK := GetAuthSubjectFromContext(c)
	if !subjectOK || subject.UserID <= 0 {
		AbortWithError(c, http.StatusForbidden, "FORBIDDEN", "Account ownership could not be verified")
		return false
	}
	owned, err := resolver.IsAccountOwnedBy(c.Request.Context(), accountID, subject.UserID)
	if err != nil {
		AbortWithError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Account ownership could not be verified")
		return false
	}
	if !owned {
		AbortWithError(c, http.StatusForbidden, "FORBIDDEN", "Only the account owner can access this account")
		return false
	}
	return true
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
