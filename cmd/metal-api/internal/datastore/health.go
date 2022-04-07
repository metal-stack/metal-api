package datastore

import (
	"context"
	"time"

	"github.com/metal-stack/metal-lib/rest"
	r "gopkg.in/rethinkdb/rethinkdb-go.v6"
)

func (rs *RethinkStore) ServiceName() string {
	return "rethinkdb"
}

// Check implements the health interface and tests if the database is healthy.
func (rs *RethinkStore) Check(ctx context.Context) (rest.HealthStatus, error) {
	healthCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	t := r.Branch(
		rs.db().TableList().SetIntersection(r.Expr(tables)).Count().Eq(len(tables)),
		r.Expr(true),
		r.Error("required tables are missing"),
	)

	err := t.Exec(rs.session, r.ExecOpts{Context: healthCtx})
	if err != nil {
		return rest.HealthStatusUnhealthy, err
	}

	return rest.HealthStatusHealthy, nil
}
