package permissions

import "context"

type Permissions map[Permission]bool

type Permission string

// ListPermissions lists all permissions handled by the permissions handler
func (p *PermissionsHandler) ListPermissions(ctx context.Context) ([]string, error) {
	return p.decider.ListPermissions(ctx)
}
