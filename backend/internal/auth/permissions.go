package auth

import "github.com/Exonical/licenseiq/backend/internal/domain"

type Permission struct {
	Resource string
	Action   string
}

var permissionMatrix = map[Permission]domain.Role{
	{Resource: "vendors", Action: "read"}:           domain.RoleViewer,
	{Resource: "vendors", Action: "write"}:          domain.RoleLicenseManager,
	{Resource: "products", Action: "read"}:          domain.RoleViewer,
	{Resource: "products", Action: "write"}:         domain.RoleLicenseManager,
	{Resource: "licenses", Action: "read"}:          domain.RoleViewer,
	{Resource: "licenses", Action: "write"}:         domain.RoleLicenseManager,
	{Resource: "assignments", Action: "read"}:       domain.RoleViewer,
	{Resource: "assignments", Action: "write"}:      domain.RoleLicenseManager,
	{Resource: "attachments", Action: "read"}:       domain.RoleViewer,
	{Resource: "attachments", Action: "write"}:      domain.RoleLicenseManager,
	{Resource: "feature_flags", Action: "read"}:     domain.RoleViewer,
	{Resource: "feature_flags", Action: "write"}:    domain.RoleAdministrator,
	{Resource: "service_accounts", Action: "read"}:  domain.RoleAdministrator,
	{Resource: "service_accounts", Action: "write"}: domain.RoleAdministrator,
	{Resource: "service_accounts", Action: "admin"}: domain.RoleAdministrator,
	{Resource: "api_keys", Action: "read"}:          domain.RoleAdministrator,
	{Resource: "api_keys", Action: "write"}:         domain.RoleAdministrator,
	{Resource: "api_keys", Action: "admin"}:         domain.RoleAdministrator,
	{Resource: "audit_logs", Action: "read"}:        domain.RoleAuditor,
	{Resource: "reports", Action: "read"}:           domain.RoleViewer,
	{Resource: "reports", Action: "financial"}:      domain.RoleFinance,
	{Resource: "notifications", Action: "admin"}:    domain.RoleAdministrator,
}

func RequiredRole(resource, action string) (domain.Role, bool) {
	role, ok := permissionMatrix[Permission{Resource: resource, Action: action}]
	return role, ok
}

func Allows(actor, required domain.Role) bool {
	return rank(actor) >= rank(required)
}

func rank(role domain.Role) int {
	switch role {
	case domain.RoleAdministrator:
		return 5
	case domain.RoleLicenseManager:
		return 4
	case domain.RoleFinance:
		return 3
	case domain.RoleAuditor:
		return 2
	case domain.RoleViewer:
		return 1
	default:
		return 0
	}
}
