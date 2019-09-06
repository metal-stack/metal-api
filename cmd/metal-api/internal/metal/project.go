package metal

// A Project is an entity to bind machine groups, networks and ips to
type Project struct {
	Base
	Tenant string `rethinkdb:"tenant"`
}

// Projects is a list of projects.
type Projects []Project
