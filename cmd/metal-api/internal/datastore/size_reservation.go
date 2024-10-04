package datastore

import (
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
	r "gopkg.in/rethinkdb/rethinkdb-go.v6"
)

// SizeReservationSearchQuery can be used to search sizes.
type SizeReservationSearchQuery struct {
	ID        *string           `json:"id" optional:"true"`
	SizeID    *string           `json:"sizeid" optional:"true"`
	Name      *string           `json:"name" optional:"true"`
	Labels    map[string]string `json:"labels" optional:"true"`
	Partition *string           `json:"partition" optional:"true"`
	Project   *string           `json:"project" optional:"true"`
}

func (s *SizeReservationSearchQuery) generateTerm(rs *RethinkStore) *r.Term {
	q := *rs.sizeReservationTable()

	if s.ID != nil {
		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("id").Eq(*s.ID)
		})
	}

	if s.SizeID != nil {
		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("sizeid").Eq(*s.SizeID)
		})
	}

	if s.Name != nil {
		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("name").Eq(*s.Name)
		})
	}

	for k, v := range s.Labels {
		k := k
		v := v
		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("labels").Field(k).Eq(v)
		})
	}

	if s.Project != nil {
		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("projectid").Eq(*s.Project)
		})
	}

	if s.Partition != nil {
		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("partitionids").Contains(r.Expr(*s.Partition))
		})
	}

	return &q
}

func (rs *RethinkStore) FindSizeReservation(id string) (*metal.SizeReservation, error) {
	var s metal.SizeReservation
	err := rs.findEntityByID(rs.sizeReservationTable(), &s, id)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (rs *RethinkStore) SearchSizeReservations(q *SizeReservationSearchQuery, rvs *metal.SizeReservations) error {
	return rs.searchEntities(q.generateTerm(rs), rvs)
}

func (rs *RethinkStore) ListSizeReservations() (metal.SizeReservations, error) {
	szs := make(metal.SizeReservations, 0)
	err := rs.listEntities(rs.sizeReservationTable(), &szs)
	return szs, err
}

func (rs *RethinkStore) CreateSizeReservation(rv *metal.SizeReservation) error {
	return rs.createEntity(rs.sizeReservationTable(), rv)
}

func (rs *RethinkStore) DeleteSizeReservation(rv *metal.SizeReservation) error {
	return rs.deleteEntity(rs.sizeReservationTable(), rv)
}

func (rs *RethinkStore) UpdateSizeReservation(oldRv *metal.SizeReservation, newRv *metal.SizeReservation) error {
	return rs.updateEntity(rs.sizeReservationTable(), newRv, oldRv)
}
