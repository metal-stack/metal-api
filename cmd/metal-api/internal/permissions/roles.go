package permissions

import "github.com/metal-stack/security"

type Roles []Role
type Role struct {
	Name        string
	Status      string
	Permissions Permissions
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

func mockRoles(_ *security.User) Roles {
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
