package datastore

import (
	"context"
	"fmt"

	"github.com/metal-stack/metal-lib/pkg/healthstatus"
	r "gopkg.in/rethinkdb/rethinkdb-go.v6"
)

func (rs *RethinkStore) ServiceName() string {
	return "rethinkdb"
}

// Check implements the health interface and tests if the database is healthy.
func (rs *RethinkStore) Check(ctx context.Context) (healthstatus.HealthResult, error) {
	var version string

	returnStatus := func(err error) (healthstatus.HealthResult, error) {
		if err != nil {
			return healthstatus.HealthResult{
				Status: healthstatus.HealthStatusUnhealthy,
			}, err
		}

		return healthstatus.HealthResult{
			Status:  healthstatus.HealthStatusHealthy,
			Message: fmt.Sprintf("connected to rethinkdb version: %s", version),
		}, nil
	}

	t := r.Branch(
		rs.db().TableList().SetIntersection(r.Expr(tables)).Count().Eq(len(tables)),
		r.Expr(true),
		r.Error("required tables are missing"),
	)

	err := t.Exec(rs.session, r.ExecOpts{Context: ctx})
	if err != nil {
		return returnStatus(err)
	}

	cursor, err := r.DB("rethinkdb").Table("server_status").Field("process").Field("version").Run(rs.session, r.RunOpts{Context: ctx})
	if err != nil {
		return returnStatus(err)
	}

	err = cursor.One(&version)
	if err != nil {
		return returnStatus(err)
	}

	return returnStatus(err)

}
