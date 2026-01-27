package admin

// Permission represents an admin permission
type Permission string

const (
	// User management
	PermViewUsers        Permission = "users.view"
	PermBanUsers         Permission = "users.ban"
	PermVerifyUsers      Permission = "users.verify"
	PermDeleteUsers      Permission = "users.delete"
	PermImpersonateUsers Permission = "users.impersonate"

	// Content moderation
	PermViewContent     Permission = "content.view"
	PermModerateContent Permission = "content.moderate"
	PermDeleteContent   Permission = "content.delete"

	// Subscriptions
	PermViewSubscriptions   Permission = "subscriptions.view"
	PermManageSubscriptions Permission = "subscriptions.manage"
	PermRefundPayments      Permission = "payments.refund"

	// System
	PermViewAnalytics  Permission = "analytics.view"
	PermManageFeatures Permission = "features.manage"
	PermManageAdmins   Permission = "admins.manage"
	PermViewAuditLogs  Permission = "audit.view"
)

// RolePermissions maps roles to their permissions
var RolePermissions = map[Role][]Permission{
	RoleSuperAdmin: {
		// All permissions
		PermViewUsers, PermBanUsers, PermVerifyUsers, PermDeleteUsers, PermImpersonateUsers,
		PermViewContent, PermModerateContent, PermDeleteContent,
		PermViewSubscriptions, PermManageSubscriptions, PermRefundPayments,
		PermViewAnalytics, PermManageFeatures, PermManageAdmins, PermViewAuditLogs,
	},
	RoleAdmin: {
		PermViewUsers, PermBanUsers, PermVerifyUsers,
		PermViewContent, PermModerateContent, PermDeleteContent,
		PermViewSubscriptions, PermManageSubscriptions,
		PermViewAnalytics, PermManageFeatures, PermViewAuditLogs,
	},
	RoleModerator: {
		PermViewUsers, PermBanUsers,
		PermViewContent, PermModerateContent,
		PermViewAnalytics,
	},
	RoleSupport: {
		PermViewUsers,
		PermViewContent,
		PermViewSubscriptions,
	},
}

// RoleHierarchy defines role levels (higher = more permissions)
var RoleHierarchy = map[Role]int{
	RoleSuperAdmin: 100,
	RoleAdmin:      80,
	RoleModerator:  60,
	RoleSupport:    40,
}

// CanManage checks if role1 can manage role2
func CanManage(role1, role2 Role) bool {
	return RoleHierarchy[role1] > RoleHierarchy[role2]
}
