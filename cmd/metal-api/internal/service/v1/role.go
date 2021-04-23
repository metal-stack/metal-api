package v1

type Role struct {
	Common
	Permissions []Permission `json:"permissions" description:"the permissions associated with this role"`
	Groups      []string     `json:"max" groups:"the (oidc) groups associated with this role"`
}

type Permission struct {
	Name string `json:"name" description:"the name of this permission"`
}
