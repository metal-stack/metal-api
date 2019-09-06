package datastore

// FIXME: THIS WILL BE FACTORED OUT INTO A SEPARATE MICROSERVICE

import (
	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/metal"

	r "gopkg.in/rethinkdb/rethinkdb-go.v5"
)

// ProjectSearchQuery can be used to search projects.
type ProjectSearchQuery struct {
	ID     *string `json:"id"`
	Name   *string `json:"name"`
	Tenant *string `json:"tenant"`
}

// GenerateTerm generates the project search query term.
func (p *ProjectSearchQuery) generateTerm(rs *RethinkStore) *r.Term {
	q := *rs.projectTable()

	if p.ID != nil {
		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("id").Eq(*p.ID)
		})
	}

	if p.Name != nil {
		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("name").Eq(*p.Name)
		})
	}

	if p.Tenant != nil {
		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("tenant").Eq(*p.Tenant)
		})
	}

	return &q
}

// FindProjectByID returns a project for a given id.
func (rs *RethinkStore) FindProjectByID(id string) (*metal.Project, error) {
	var p metal.Project
	err := rs.findEntityByID(rs.projectTable(), &p, id)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// FindProject returns a project by the given query, fails if there is no record or multiple records found.
func (rs *RethinkStore) FindProject(q *ProjectSearchQuery, p *metal.Project) error {
	return rs.findEntity(q.generateTerm(rs), &p)
}

// SearchProjects returns the result of the projects search request query.
func (rs *RethinkStore) SearchProjects(q *ProjectSearchQuery, ps *metal.Projects) error {
	return rs.searchEntities(q.generateTerm(rs), ps)
}

// ListProjects returns all projects.
func (rs *RethinkStore) ListProjects() (metal.Projects, error) {
	ps := make(metal.Projects, 0)
	err := rs.listEntities(rs.projectTable(), &ps)
	return ps, err
}

// CreateProject creates a new project.
func (rs *RethinkStore) CreateProject(m *metal.Project) error {
	return rs.createEntity(rs.projectTable(), m)
}

// DeleteProject deletes a project.
func (rs *RethinkStore) DeleteProject(m *metal.Project) error {
	return rs.deleteEntity(rs.projectTable(), m)
}

// UpdateProject updates a project.
func (rs *RethinkStore) UpdateProject(oldProject *metal.Project, newProject *metal.Project) error {
	return rs.updateEntity(rs.projectTable(), newProject, oldProject)
}
