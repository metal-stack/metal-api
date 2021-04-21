package datastore

import (
	"fmt"

	"github.com/emicklei/go-restful/v3"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
	r "gopkg.in/rethinkdb/rethinkdb-go.v6"
)

const (
	// FIXME prevent tenants, projects and resources to be named "*"
	// scopeAllTenants is a scope wildcard for tenant predicates
	scopeAllTenants = "*"
	// scopeAllProjects is a scope wildcard for project predicates
	scopeAllProjects = "*"
	// scopeAllResources is a scope wildcard for resource predicates
	scopeAllResources = "*"
)

// ResourceScope contains predicates for filtering resources on the database.
type ResourceScope struct {
	// tenants includes predicates that need to match the resource's tenant field
	tenants Predicates
	// projects includes predicates that need to match the resource's project field
	projects Predicates
	// resources includes predicates that need to match the resource's id field
	resources Predicates
	// isAdmin decides whether public resources (i.e. resources with empty tenant and project) can be written .
	isAdmin bool

	visibility Predicates
}

func NewResourceScope(tenants Predicates, projects Predicates, resources Predicates, isAdmin bool) ResourceScope {
	return ResourceScope{
		tenants:   tenants,
		projects:  projects,
		resources: resources,
		isAdmin:   isAdmin,
	}
}

// EverythingScope has all permissions on all resources.
// Should only be used for internal, technical purposes.
var EverythingScope = ResourceScope{
	tenants:    Predicates{scopeAllTenants},
	projects:   Predicates{scopeAllProjects},
	resources:  Predicates{scopeAllResources},
	isAdmin:    true,
	visibility: Predicates{VisibilityPrivate, VisibilityShared, VisibilityPublic, VisibilityAdmin},
}

// NewResourceScopeFromRequestAttributes extracts the resource scope from the request attributes.
//
// The attributes are typically defined in the request filter chain before reaching the business logic.
//
// It also extracts a common query parameter "visibility" for additional scoping on resource visibility.
func (rs *RethinkStore) NewResourceScopeFromRequestAttributes(req *restful.Request) (*ResourceScope, error) {
	scope, ok := req.Attribute("scope").(ResourceScope)
	if !ok {
		return nil, fmt.Errorf("no resource scope found in request attributes")
	}

	v, err := visibilityFromName(req.QueryParameter("visibility"))
	if err != nil {
		return nil, err
	}
	scope.visibility = v

	if v.Contains(VisibilityAdmin) && !scope.isAdmin {
		return nil, fmt.Errorf("not allowed to view resources with admin visibility")
	}

	rs.Debugw("created new resource scope", "visibilities", scope.visibility)

	return &scope, nil
}

// Apply adds database filters to the database query in order to exclude resources from
// the result for which a user does not have the required permissions.
func (scope *ResourceScope) Apply(e metal.Entity, q r.Term) r.Term {
	fields := e.GetFieldNames()

	var subTerms []r.Term

	if scope.visibility.Contains(VisibilityPrivate) {
		qPrivate := q

		if !scope.allTenants() {
			qPrivate = qPrivate.Filter(func(row r.Term) r.Term {
				return r.Expr(scope.tenants).Contains(row.Field(fields.TenantID))
			})
		}

		if !scope.allProjects() {
			qPrivate = qPrivate.Filter(func(row r.Term) r.Term {
				return r.Expr(scope.projects).Contains(row.Field(fields.ProjectID))
			})
		}

		if !scope.allResources() {
			qPrivate = qPrivate.Filter(func(row r.Term) r.Term {
				return r.Expr(scope.resources).Contains(row.Field(fields.ID))
			})
		}

		subTerms = append(subTerms, qPrivate)
	}

	if scope.visibility.Contains(VisibilityShared) {
		qShared := q

		qShared = qShared.Filter(func(row r.Term) r.Term {
			// FIXME shared needs field getter
			return row.Field("shared").Eq(true)
		})

		subTerms = append(subTerms, qShared)
	}

	if scope.visibility.Contains(VisibilityPublic) {
		subTerms = append(subTerms, q.Filter(map[string]interface{}{fields.ProjectID: "", fields.TenantID: ""}))
	}

	// if scope.visibility.Contains(VisibilityAdmin) {
	// FIXME what to do here?
	// }

	if len(subTerms) == 0 {
		return r.Expr(map[string]interface{}{})
	}

	res := subTerms[0]
	for i, subTerm := range subTerms {
		if i == 0 {
			continue
		}
		res = res.Union(subTerm)
	}

	return res
}

// AllowsWriteOn returns true if the resource scope allows the given entities for writing.
func (scope *ResourceScope) AllowsWriteOn(es ...metal.Entity) error {
	for _, e := range es {
		if err := scope.allows(e); err != nil {
			return err
		}
	}
	return nil
}

func (scope *ResourceScope) allows(e metal.Entity) error {
	if (e.GetProjectID() == "" || e.GetTenantID() == "") && !scope.isAdmin {
		return fmt.Errorf("no write access on admin resources")
	}

	if !scope.allTenants() && !scope.tenants.Contains(Predicate(e.GetTenantID())) {
		return fmt.Errorf("tenant not allowed")
	}

	if !scope.allProjects() && !scope.projects.Contains(Predicate(e.GetProjectID())) {
		return fmt.Errorf("project not allowed")
	}

	if !scope.allResources() && !scope.resources.Contains(Predicate(e.GetID())) {
		return fmt.Errorf("id not allowed")
	}

	return nil
}

func (scope *ResourceScope) allTenants() bool {
	return scope.tenants.Contains(scopeAllTenants)
}

func (scope *ResourceScope) allProjects() bool {
	return scope.projects.Contains(scopeAllProjects)
}

func (scope *ResourceScope) allResources() bool {
	return scope.resources.Contains(scopeAllResources)
}
