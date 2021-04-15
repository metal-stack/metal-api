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

func mockRoles(u *security.User) Roles {
	return Roles{
		{
			Name:   "Metal Image Lister",
			Status: "active",
			Permissions: Permissions{
				"metal.v1.image.list": true,
			},
		},
		{
			Name:   "Metal Image Getter",
			Status: "active",
			Permissions: Permissions{
				"metal.v1.image.get": true,
			},
		},
	}
}
