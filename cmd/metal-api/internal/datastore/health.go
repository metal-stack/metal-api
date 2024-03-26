package datastore

import (
	"context"

	"github.com/metal-stack/metal-lib/rest"
	r "gopkg.in/rethinkdb/rethinkdb-go.v6"
)

func (rs *RethinkStore) ServiceName() string {
	return "rethinkdb"
}

// Check implements the health interface and tests if the database is healthy.
func (rs *RethinkStore) Check(ctx context.Context) (rest.HealthResult, error) {
	t := r.Branch(
		rs.db().TableList().SetIntersection(r.Expr(tables)).Count().Eq(len(tables)),
		r.Expr(true),
		r.Error("required tables are missing"),
	)

	err := t.Exec(rs.session, r.ExecOpts{Context: ctx})
	if err != nil {
		return rest.HealthResult{
			Status: rest.HealthStatusUnhealthy,
		}, err
	}

	return rest.HealthResult{
		Status: rest.HealthStatusHealthy,
	}, nil
}
