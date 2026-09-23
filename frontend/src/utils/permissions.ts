export type UserRole = 'super_admin' | 'admin' | 'user' | 'enterprise_user'

export const Permission = {
  AdminPanel: 'admin.panel',
  AdminDashboardRead: 'admin.dashboard.read',
  AdminUsersRead: 'admin.users.read',
  AdminUsersWrite: 'admin.users.write',
  AdminUsersRoleManage: 'admin.users.role_manage',
  AdminGroupsRead: 'admin.groups.read',
  AdminGroupsWrite: 'admin.groups.write',
  AdminAccountsRead: 'admin.accounts.read',
  AdminAccountsWrite: 'admin.accounts.write',
  AdminAccountsOwnedWrite: 'admin.accounts.owned_write',
  AdminCredentialsRead: 'admin.credentials.read',
  AdminProxiesRead: 'admin.proxies.read',
  AdminProxiesWrite: 'admin.proxies.write',
  AdminUsageRead: 'admin.usage.read',
  AdminUsageWrite: 'admin.usage.write',
  AdminOpsRead: 'admin.ops.read',
  AdminOpsWrite: 'admin.ops.write',
  AdminSettingsRead: 'admin.settings.read',
  AdminSettingsWrite: 'admin.settings.write',
  AdminSystemWrite: 'admin.system.write',
  AdminPaymentRead: 'admin.payment.read',
  AdminPaymentWrite: 'admin.payment.write',
  AdminSecurityRead: 'admin.security.read',
  AdminSecurityWrite: 'admin.security.write',
  AdminPluginsWrite: 'admin.plugins.write',
  AdminAuditRead: 'admin.audit.read',
  AdminAuditWrite: 'admin.audit.write',
  EnterpriseAccountPool: 'enterprise.account_pool.read',
  EnterpriseUsage: 'enterprise.usage.read',
  EnterpriseLogs: 'enterprise.logs.read',
} as const

export type PermissionCode = (typeof Permission)[keyof typeof Permission]

const ROLE_DEFAULT_PERMISSIONS: Record<UserRole, readonly string[]> = {
  super_admin: Object.values(Permission),
  admin: [
    Permission.AdminPanel,
    Permission.AdminDashboardRead,
    Permission.AdminUsersRead,
    Permission.AdminUsersWrite,
    Permission.AdminGroupsRead,
    Permission.AdminGroupsWrite,
    Permission.AdminAccountsRead,
    Permission.AdminAccountsOwnedWrite,
    Permission.AdminUsageRead,
    Permission.AdminOpsRead,
    Permission.AdminProxiesRead,
    Permission.AdminSecurityRead,
    Permission.EnterpriseAccountPool,
    Permission.EnterpriseUsage,
    Permission.EnterpriseLogs,
  ],
  user: [],
  enterprise_user: [
    Permission.EnterpriseAccountPool,
    Permission.EnterpriseUsage,
    Permission.EnterpriseLogs,
  ],
}

export function defaultPermissionsForRole(role: string | undefined): readonly string[] {
  return ROLE_DEFAULT_PERMISSIONS[role as UserRole] ?? []
}

export function isStaffRole(role: string | undefined): boolean {
  return role === 'super_admin' || role === 'admin'
}

export function isSuperAdminRole(role: string | undefined): boolean {
  return role === 'super_admin'
}

// 后端角色值是 snake_case，i18n 键是 camelCase；缺少映射会把原始键直接显示出来。
const ROLE_I18N_KEYS: Record<UserRole, string> = {
  super_admin: 'superAdmin',
  admin: 'admin',
  enterprise_user: 'enterpriseUser',
  user: 'user',
}

/** 返回 admin.users.roles.* 下的完整 i18n 键；未知角色回退到「用户」。 */
export function roleLabelKey(role: string | undefined): string {
  return `admin.users.roles.${ROLE_I18N_KEYS[role as UserRole] ?? ROLE_I18N_KEYS.user}`
}
