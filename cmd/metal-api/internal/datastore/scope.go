package datastore

import (
	"fmt"

	"github.com/emicklei/go-restful/v3"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
	r "gopkg.in/rethinkdb/rethinkdb-go.v6"
)

const (
	ScopeRequestAttributeKey = "scope"

	// FIXME prevent tenants, projects and resources to be named "*"
	ScopeAllTenants   = "*"
	ScopeAllProjects  = "*"
	ScopeAllResources = "*"
)

type Predicate string

type Predicates []Predicate

type ResourceScope struct {
	Tenants   Predicates
	Projects  Predicates
	Resources Predicates

	Visibility metal.Visibility
}

var EverythingScope = ResourceScope{
	Tenants:   Predicates{ScopeAllTenants},
	Projects:  Predicates{ScopeAllProjects},
	Resources: Predicates{ScopeAllResources},
}

func NewResourceScopeFromRequestAttributes(req *restful.Request, v metal.Visibility) (*ResourceScope, error) {
	scope, ok := req.Attribute(ScopeRequestAttributeKey).(ResourceScope)
	if !ok {
		return nil, fmt.Errorf("no resource scope found in request attributes")
	}

	scope.Visibility = v

	return &scope, nil
}

func (scope *ResourceScope) Apply(q r.Term) r.Term {
	base := q

	if !scope.allTenants() {
		q = q.Filter(func(row r.Term) r.Term {
			return r.Expr(scope.Tenants).Contains(row.Field("tenantid"))
		})
	}

	if !scope.allProjects() {
		q = q.Filter(func(row r.Term) r.Term {
			return r.Expr(scope.Projects).Contains(row.Field("projectid"))
		})
	}

	if !scope.allResources() {
		q = q.Filter(func(row r.Term) r.Term {
			return r.Expr(scope.Resources).Contains(row.Field("id"))
		})
	}

	if scope.Visibility == metal.VisibilityPublic {
		q = q.Union(base.Filter(map[string]interface{}{"visibility": metal.VisibilityPublic}))
	}

	return q
}

func (scope *ResourceScope) Allows(e metal.Entity) bool {
	if !scope.allTenants() && !scope.Tenants.Contains(Predicate(e.GetTenantID())) {
		return false
	}

	if !scope.allProjects() && !scope.Projects.Contains(Predicate(e.GetProjectID())) {
		return false
	}

	if !scope.allResources() && !scope.Resources.Contains(Predicate(e.GetID())) {
		return false
	}

	// FIXME check visibility

	return true
}

func (scope *ResourceScope) allTenants() bool {
	for _, s := range scope.Tenants {
		if s == ScopeAllTenants {
			return true
		}
	}
	return false
}

func (scope *ResourceScope) allProjects() bool {
	for _, s := range scope.Projects {
		if s == ScopeAllProjects {
			return true
		}
	}
	return false
}

func (scope *ResourceScope) allResources() bool {
	for _, s := range scope.Resources {
		if s == ScopeAllResources {
			return true
		}
	}
	return false
}

func (ps Predicates) Contains(p Predicate) bool {
	for _, pp := range ps {
		if pp == p {
			return true
		}
	}
	return false
}
