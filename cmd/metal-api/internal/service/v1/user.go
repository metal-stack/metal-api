package v1

type User struct {
	EMail   string
	Name    string
	Groups  []string
	Tenant  string
	Issuer  string
	Subject string
}
