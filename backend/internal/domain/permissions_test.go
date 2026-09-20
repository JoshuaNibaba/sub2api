package domain

import "testing"

func TestRolePermissionMatrix(t *testing.T) {
	tests := []struct {
		role       string
		permission Permission
		want       bool
	}{
		{RoleSuperAdmin, PermissionAdminSystemWrite, true},
		{RoleAdmin, PermissionAdminUsersWrite, true},
		{RoleAdmin, PermissionAdminSystemWrite, false},
		{RoleAdmin, PermissionAdminOpsRead, true},
		{RoleAdmin, PermissionAdminOpsWrite, false},
		{RoleAdmin, PermissionAdminUsageWrite, false},
		{RoleUser, PermissionAdminPanel, false},
		{RoleEnterpriseUser, PermissionEnterpriseAccountPool, true},
		{RoleEnterpriseUser, PermissionAdminAuditRead, false},
		{RoleSuperAdmin, PermissionAdminAuditWrite, true},
	}

	for _, tt := range tests {
		if got := HasPermission(tt.role, tt.permission); got != tt.want {
			t.Fatalf("HasPermission(%q, %q)=%v, want %v", tt.role, tt.permission, got, tt.want)
		}
	}
}

func TestPermissionsForRoleIsSortedAndDefensive(t *testing.T) {
	permissions := PermissionsForRole(RoleEnterpriseUser)
	if len(permissions) != 3 {
		t.Fatalf("got %d enterprise permissions, want 3", len(permissions))
	}
	permissions[0] = "mutated"
	if HasPermission(RoleEnterpriseUser, PermissionEnterpriseAccountPool) == false {
		t.Fatal("permission lookup changed after mutating returned slice")
	}
}
