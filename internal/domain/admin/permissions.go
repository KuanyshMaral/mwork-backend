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

	// Organization verification
	PermViewOrganizations   Permission = "organizations.view"
	PermVerifyOrganizations Permission = "organizations.verify"

	// Content moderation
	PermViewContent     Permission = "content.view"
	PermModerateContent Permission = "content.moderate"
	PermDeleteContent   Permission = "content.delete"

	// Subscriptions
	PermViewSubscriptions   Permission = "subscriptions.view"
	PermManageSubscriptions Permission = "subscriptions.manage"
	PermRefundPayments      Permission = "payments.refund"

	// Credits (B3: New permission for admin credit grants)
	PermGrantCredits Permission = "credits.grant"

	// System
	PermViewAnalytics    Permission = "analytics.view"
	PermManageFeatures   Permission = "features.manage"
	PermManageAdmins     Permission = "admins.manage"
	PermViewAuditLogs    Permission = "audit.view"
	PermReconcileCredits Permission = "credits.reconcile"
)

// RolePermissions maps roles to their permissions
var RolePermissions = map[Role][]Permission{
	RoleSuperAdmin: {
		// All permissions
		PermViewUsers, PermBanUsers, PermVerifyUsers, PermDeleteUsers, PermImpersonateUsers,
		PermVerifyOrganizations,
		PermViewOrganizations,
		PermViewContent, PermModerateContent, PermDeleteContent,
		PermViewSubscriptions, PermManageSubscriptions, PermRefundPayments,
		PermGrantCredits, // B3: SuperAdmin can grant credits
		PermViewAnalytics, PermManageFeatures, PermManageAdmins, PermViewAuditLogs, PermReconcileCredits,
	},
	RoleAdmin: {
		PermViewUsers, PermBanUsers, PermVerifyUsers,
		PermViewContent, PermModerateContent, PermDeleteContent,
		PermViewSubscriptions, PermManageSubscriptions,
		PermGrantCredits, // B3: Admin can grant credits
		PermViewAnalytics, PermManageFeatures, PermViewAuditLogs, PermReconcileCredits,
		PermViewOrganizations,
		PermVerifyOrganizations,
	},
	RoleModerator: {
		PermViewUsers, PermBanUsers,
		PermViewContent, PermModerateContent,
		PermViewAnalytics,
		PermViewOrganizations,
		PermVerifyOrganizations,
	},
	RoleSupport: {
		PermViewUsers,
		PermViewContent,
		PermViewSubscriptions,
		PermViewOrganizations,
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
