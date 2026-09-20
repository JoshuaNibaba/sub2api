package domain

import "sort"

// Permission is an application capability. Roles are only bundles of these
// capabilities; route and service checks should depend on Permission values
// instead of comparing role strings directly.
type Permission string

const (
	PermissionAdminPanel            Permission = "admin.panel"
	PermissionAdminDashboardRead    Permission = "admin.dashboard.read"
	PermissionAdminUsersRead        Permission = "admin.users.read"
	PermissionAdminUsersWrite       Permission = "admin.users.write"
	PermissionAdminUsersRoleManage  Permission = "admin.users.role_manage"
	PermissionAdminGroupsRead       Permission = "admin.groups.read"
	PermissionAdminGroupsWrite      Permission = "admin.groups.write"
	PermissionAdminAccountsRead     Permission = "admin.accounts.read"
	PermissionAdminAccountsWrite    Permission = "admin.accounts.write"
	PermissionAdminCredentialsRead  Permission = "admin.credentials.read"
	PermissionAdminProxiesRead      Permission = "admin.proxies.read"
	PermissionAdminProxiesWrite     Permission = "admin.proxies.write"
	PermissionAdminUsageRead        Permission = "admin.usage.read"
	PermissionAdminUsageWrite       Permission = "admin.usage.write"
	PermissionAdminOpsRead          Permission = "admin.ops.read"
	PermissionAdminOpsWrite         Permission = "admin.ops.write"
	PermissionAdminSettingsRead     Permission = "admin.settings.read"
	PermissionAdminSettingsWrite    Permission = "admin.settings.write"
	PermissionAdminSystemWrite      Permission = "admin.system.write"
	PermissionAdminPaymentRead      Permission = "admin.payment.read"
	PermissionAdminPaymentWrite     Permission = "admin.payment.write"
	PermissionAdminSecurityRead     Permission = "admin.security.read"
	PermissionAdminSecurityWrite    Permission = "admin.security.write"
	PermissionAdminPluginsWrite     Permission = "admin.plugins.write"
	PermissionAdminAuditRead        Permission = "admin.audit.read"
	PermissionAdminAuditWrite       Permission = "admin.audit.write"
	PermissionEnterpriseAccountPool Permission = "enterprise.account_pool.read"
	PermissionEnterpriseUsage       Permission = "enterprise.usage.read"
	PermissionEnterpriseLogs        Permission = "enterprise.logs.read"
)

var allPermissions = []Permission{
	PermissionAdminPanel,
	PermissionAdminDashboardRead,
	PermissionAdminUsersRead,
	PermissionAdminUsersWrite,
	PermissionAdminUsersRoleManage,
	PermissionAdminGroupsRead,
	PermissionAdminGroupsWrite,
	PermissionAdminAccountsRead,
	PermissionAdminAccountsWrite,
	PermissionAdminCredentialsRead,
	PermissionAdminProxiesRead,
	PermissionAdminProxiesWrite,
	PermissionAdminUsageRead,
	PermissionAdminUsageWrite,
	PermissionAdminOpsRead,
	PermissionAdminOpsWrite,
	PermissionAdminSettingsRead,
	PermissionAdminSettingsWrite,
	PermissionAdminSystemWrite,
	PermissionAdminPaymentRead,
	PermissionAdminPaymentWrite,
	PermissionAdminSecurityRead,
	PermissionAdminSecurityWrite,
	PermissionAdminPluginsWrite,
	PermissionAdminAuditRead,
	PermissionAdminAuditWrite,
	PermissionEnterpriseAccountPool,
	PermissionEnterpriseUsage,
	PermissionEnterpriseLogs,
}

// rolePermissions is the single default policy matrix. Changing a role's
// default access is intentionally a local change here; routes and DTOs do not
// need to be rewritten. A future database-backed policy can layer overrides on
// top of this registry without changing callers of HasPermission.
var rolePermissions = map[string]map[Permission]struct{}{
	RoleSuperAdmin: permissionSet(allPermissions...),
	RoleAdmin: permissionSet(
		PermissionAdminPanel,
		PermissionAdminDashboardRead,
		PermissionAdminUsersRead,
		PermissionAdminUsersWrite,
		PermissionAdminGroupsRead,
		PermissionAdminGroupsWrite,
		PermissionAdminAccountsRead,
		PermissionAdminUsageRead,
		PermissionAdminOpsRead,
		PermissionAdminProxiesRead,
		PermissionAdminSecurityRead,
		PermissionEnterpriseAccountPool,
		PermissionEnterpriseUsage,
		PermissionEnterpriseLogs,
	),
	RoleUser: permissionSet(),
	RoleEnterpriseUser: permissionSet(
		PermissionEnterpriseAccountPool,
		PermissionEnterpriseUsage,
		PermissionEnterpriseLogs,
	),
}

func permissionSet(permissions ...Permission) map[Permission]struct{} {
	set := make(map[Permission]struct{}, len(permissions))
	for _, permission := range permissions {
		set[permission] = struct{}{}
	}
	return set
}

// HasPermission reports whether a role is granted a capability.
func HasPermission(role string, permission Permission) bool {
	if role == RoleSuperAdmin {
		return true
	}
	_, ok := rolePermissions[role][permission]
	return ok
}

// PermissionsForRole returns a stable, defensive copy suitable for an API DTO.
func PermissionsForRole(role string) []Permission {
	set := rolePermissions[role]
	permissions := make([]Permission, 0, len(set))
	for permission := range set {
		permissions = append(permissions, permission)
	}
	sort.Slice(permissions, func(i, j int) bool { return permissions[i] < permissions[j] })
	return permissions
}

func IsSuperAdminRole(role string) bool {
	return role == RoleSuperAdmin
}

func IsStaffRole(role string) bool {
	return role == RoleSuperAdmin || role == RoleAdmin
}

func IsEnterpriseRole(role string) bool {
	return role == RoleEnterpriseUser
}
