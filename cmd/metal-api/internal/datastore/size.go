package datastore

import (
	"errors"

	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
	r "gopkg.in/rethinkdb/rethinkdb-go.v6"
)

// SizeSearchQuery can be used to search sizes.
type SizeSearchQuery struct {
	ID          *string           `json:"id" optional:"true"`
	Name        *string           `json:"name" optional:"true"`
	Labels      map[string]string `json:"labels" optional:"true"`
	Reservation Reservation       `json:"reservation" optional:"true"`
}

type Reservation struct {
	Partition *string `json:"partition" optional:"true"`
	Project   *string `json:"project" optional:"true"`
}

// GenerateTerm generates the project search query term.
func (s *SizeSearchQuery) generateTerm(rs *RethinkStore) *r.Term {
	q := *rs.sizeTable()

	if s.ID != nil {
		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("id").Eq(*s.ID)
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

	if s.Reservation.Project != nil {
		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("reservations").Contains(func(p r.Term) r.Term {
				return p.Field("projectid").Eq(r.Expr(*s.Reservation.Project))
			})
		})
	}

	if s.Reservation.Partition != nil {
		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("reservations").Contains(func(p r.Term) r.Term {
				return p.Field("partitionids").Contains(r.Expr(*s.Reservation.Partition))
			})
		})
	}

	return &q
}

// FindSize return a size for a given id.
func (rs *RethinkStore) FindSize(id string) (*metal.Size, error) {
	var s metal.Size
	err := rs.findEntityByID(rs.sizeTable(), &s, id)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// SearchSizes returns the result of the sizes search request query.
func (rs *RethinkStore) SearchSizes(q *SizeSearchQuery, sizes *metal.Sizes) error {
	return rs.searchEntities(q.generateTerm(rs), sizes)
}

// ListSizes returns all sizes.
func (rs *RethinkStore) ListSizes() (metal.Sizes, error) {
	szs := make(metal.Sizes, 0)
	err := rs.listEntities(rs.sizeTable(), &szs)
	return szs, err
}

// CreateSize creates a new size.
func (rs *RethinkStore) CreateSize(size *metal.Size) error {
	return rs.createEntity(rs.sizeTable(), size)
}

// DeleteSize deletes a size.
func (rs *RethinkStore) DeleteSize(size *metal.Size) error {
	return rs.deleteEntity(rs.sizeTable(), size)
}

// UpdateSize updates a size.
func (rs *RethinkStore) UpdateSize(oldSize *metal.Size, newSize *metal.Size) error {
	return rs.updateEntity(rs.sizeTable(), newSize, oldSize)
}

// FromHardware tries to find a size which matches the given hardware specs.
func (rs *RethinkStore) FromHardware(hw metal.MachineHardware) (*metal.Size, error) {
	sz, err := rs.ListSizes()
	if err != nil {
		return nil, err
	}
	if len(sz) < 1 {
		// this should not happen, so we do not return a notfound
		return nil, errors.New("no sizes found in database")
	}
	var sizes metal.Sizes
	for _, s := range sz {
		if len(s.Constraints) < 1 {
			rs.log.Error("missing constraints", "size", s)
			continue
		}
		sizes = append(sizes, s)
	}
	return sizes.FromHardware(hw)
}
