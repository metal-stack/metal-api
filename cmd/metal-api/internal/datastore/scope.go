package datastore

import (
	"fmt"
	"strings"

	"github.com/emicklei/go-restful/v3"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
	r "gopkg.in/rethinkdb/rethinkdb-go.v6"
)

const (
	// FIXME prevent tenants, projects and resources to be named "*"
	ScopeAllTenants   = "*"
	ScopeAllProjects  = "*"
	ScopeAllResources = "*"
)

type Predicate string

func (p Predicate) String() string {
	return string(p)
}

type Predicates []Predicate

type ResourceScope struct {
	Tenants   Predicates
	Projects  Predicates
	Resources Predicates

	IsAdmin bool

	visibility Predicates
}

var EverythingScope = ResourceScope{
	Tenants:   Predicates{ScopeAllTenants},
	Projects:  Predicates{ScopeAllProjects},
	Resources: Predicates{ScopeAllResources},
	IsAdmin:   true,

	visibility: Predicates{VisibilityAdmin},
}

const (
	// VisibilityPrivate includes resources that are bound to a specific tenants and projects.
	// (e.g. private machine networks)
	VisibilityPrivate Predicate = "private"
	// VisibilityShared includes resources that are bound to a specific tenants and projects when
	// they have set a special "shared" flag. These entities can be read and used by other
	// tenants in other projects. (e.g. private shared networks)
	VisibilityShared Predicate = "shared"
	// VisibilityPublic includes resources provided by administrators that can be read and used
	// by arbitrary tenants in arbitrary projects. (e.g. public machine images)
	VisibilityPublic Predicate = "public"
	// VisibilityAdmin includes resources that can only be read and used by administrators.
	// (e.g. IPMI console data, machine firmware management, ...)
	VisibilityAdmin Predicate = "admin"
)

func visibilityFromName(input string) (Predicates, error) {
	if input == "" {
		return Predicates{VisibilityPrivate, VisibilityShared}, nil
	}

	names := strings.Split(input, ",")

	ps := map[Predicate]bool{}
	for _, name := range names {
		switch name {
		case VisibilityPrivate.String():
			ps[VisibilityPrivate] = true
		case VisibilityShared.String():
			ps[VisibilityShared] = true
		case VisibilityPublic.String():
			ps[VisibilityPublic] = true
		case VisibilityAdmin.String():
			ps[VisibilityAdmin] = true
		default:
			return nil, fmt.Errorf("unsupported visibility: %s", name)
		}
	}

	var result Predicates
	for p := range ps {
		result = append(result, p)
	}

	return result, nil
}

// NewResourceScopeFromRequestAttributes extracts the resource scope from the request attributes.
//
// The attributes are typically defined in the request filter chain before reaching the business logic.
//
// It also extracts a common query parameter for including sight of resources with public visibility.
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

	if v.Contains(VisibilityAdmin) && !scope.IsAdmin {
		return nil, fmt.Errorf("not allowed to view resources with admin visibility")
	}

	rs.Debugw("created new resource scope", "predicates", scope, "visibilities", scope.visibility)

	return &scope, nil
}

// Apply adds database filters to the database query in order to exclude resources from
// the result for which a user does not have the required permissions.
func (scope *ResourceScope) Apply(e metal.Entity, q r.Term) r.Term {
	fields := e.GetFieldNames()

	var subTerms []r.Term
	for _, v := range scope.visibility {
		switch v {
		case VisibilityPrivate:
			base := q
			if !scope.allTenants() {
				base = base.Filter(func(row r.Term) r.Term {
					return r.Expr(scope.Tenants).Contains(row.Field(fields.TenantID))
				})
			}

			if !scope.allProjects() {
				base = base.Filter(func(row r.Term) r.Term {
					return r.Expr(scope.Projects).Contains(row.Field(fields.ProjectID))
				})
			}

			if !scope.allResources() {
				base = base.Filter(func(row r.Term) r.Term {
					return r.Expr(scope.Resources).Contains(row.Field(fields.ID))
				})
			}

			subTerms = append(subTerms, base)
		case VisibilityShared:
			// FIXME: How to do this?

		case VisibilityPublic:
			subTerms = append(subTerms, q.Filter(map[string]interface{}{fields.ProjectID: "", fields.TenantID: ""}))

		case VisibilityAdmin:
			// FIXME: How to do this?
		}

	}

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

// Allows returns true if the resource scope allows the given entities for writing.
func (scope *ResourceScope) AllowsWriteOn(es ...metal.Entity) error {
	for _, e := range es {
		if err := scope.allows(e); err != nil {
			return err
		}
	}
	return nil
}

func (scope *ResourceScope) allows(e metal.Entity) error {
	if (e.GetProjectID() == "" || e.GetTenantID() == "") && !scope.IsAdmin {
		return fmt.Errorf("no write access on admin resources")
	}

	if !scope.allTenants() && !scope.Tenants.Contains(Predicate(e.GetTenantID())) {
		return fmt.Errorf("tenant not allowed")
	}

	if !scope.allProjects() && !scope.Projects.Contains(Predicate(e.GetProjectID())) {
		return fmt.Errorf("project not allowed")
	}

	if !scope.allResources() && !scope.Resources.Contains(Predicate(e.GetID())) {
		return fmt.Errorf("id not allowed")
	}

	return nil
}

func (scope *ResourceScope) allTenants() bool {
	return scope.Tenants.Contains(ScopeAllTenants)
}

func (scope *ResourceScope) allProjects() bool {
	return scope.Projects.Contains(ScopeAllProjects)
}

func (scope *ResourceScope) allResources() bool {
	return scope.Resources.Contains(ScopeAllResources)
}

// Contains returns true when a given predicate is contained in the slice of predicates.
func (ps Predicates) Contains(p Predicate) bool {
	for _, pp := range ps {
		if pp == p {
			return true
		}
	}
	return false
}

func (p Predicate) Is(pp Predicate) bool {
	return string(p) == string(pp)
}
