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
func (rs *RethinkStore) Check(ctx context.Context) (rest.HealthStatus, error) {
	err := multi(
		ctx,
		rs.session,
		r.Branch(
			rs.db().TableList().SetIntersection(r.Expr(tables)).Count().Eq(len(tables)),
			r.Expr(true),
			r.Error("required tables are missing"),
		),
	)
	if err != nil {
		return rest.HealthStatusUnhealthy, err
	}

	return rest.HealthStatusHealthy, nil
}
