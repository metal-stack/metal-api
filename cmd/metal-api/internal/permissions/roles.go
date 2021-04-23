package permissions

import (
	"context"
)

type Roles []Role
type Role struct {
	Name        string
	Status      string
	Permissions Permissions
}

// ListRoles lists all roles handled by the permissions handler
func (p *PermissionsHandler) ListRoles(ctx context.Context) (Roles, error) {
	return mockRoles(), nil // FIXME
}

func (rs Roles) MergePermissions() Permissions {
	result := Permissions{}
	for _, r := range rs {
		if r.Status != "active" {
			continue
		}

		for p := range r.Permissions {
			result[p] = true
		}
	}

	return result
}

// FIXME This is only for testing / development
func mockRoles() Roles {
	return Roles{
		{
			Name:   "Metal Image Reader",
			Status: "active",
			Permissions: Permissions{
				"metal.v1.image.get":  true,
				"metal.v1.image.list": true,
			},
		},
		{
			Name:   "Metal Image Writer",
			Status: "active",
			Permissions: Permissions{
				"metal.v1.image.create": true,
				"metal.v1.image.delete": true,
			},
		},
		{
			Name:   "Metal Image Admin Writer",
			Status: "active",
			Permissions: Permissions{
				"metal.v1.image.create.admin": true,
				"metal.v1.image.delete.admin": true,
			},
		},
	}
}
