package datastore

import (
	"errors"
	"fmt"
	"net/netip"
	"strconv"

	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
	r "gopkg.in/rethinkdb/rethinkdb-go.v6"
)

// NetworkSearchQuery can be used to search networks.
type NetworkSearchQuery struct {
	ID                  *string           `json:"id" optional:"true"`
	Name                *string           `json:"name" optional:"true"`
	PartitionID         *string           `json:"partitionid" optional:"true"`
	ProjectID           *string           `json:"projectid" optional:"true"`
	Prefixes            []string          `json:"prefixes" optional:"true"`
	DestinationPrefixes []string          `json:"destinationprefixes" optional:"true"`
	Nat                 *bool             `json:"nat" optional:"true"`
	PrivateSuper        *bool             `json:"privatesuper" optional:"true"`
	Underlay            *bool             `json:"underlay" optional:"true"`
	Vrf                 *int64            `json:"vrf" optional:"true"`
	ParentNetworkID     *string           `json:"parentnetworkid" optional:"true"`
	Labels              map[string]string `json:"labels" optional:"true"`
	AddressFamily       *string           `json:"addressfamily" optional:"true" enum:"IPv4|IPv6"`
}

func (p *NetworkSearchQuery) Validate() error {
	var errs []error
	for _, prefix := range p.Prefixes {
		_, err := netip.ParsePrefix(prefix)
		if err != nil {
			errs = append(errs, err)
		}
	}
	for _, prefix := range p.DestinationPrefixes {
		_, err := netip.ParsePrefix(prefix)
		if err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) == 0 {
		return nil
	}
	return errors.Join(errs...)
}

// GenerateTerm generates the project search query term.
func (p *NetworkSearchQuery) generateTerm(rs *RethinkStore) (*r.Term, error) {
	q := *rs.networkTable()

	if p.ID != nil {
		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("id").Eq(*p.ID)
		})
	}

	if p.ProjectID != nil {
		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("projectid").Eq(*p.ProjectID)
		})
	}

	if p.PartitionID != nil {
		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("partitionid").Eq(*p.PartitionID)
		})
	}

	if p.ParentNetworkID != nil {
		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("parentnetworkid").Eq(*p.ParentNetworkID)
		})
	}

	if p.Name != nil {
		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("name").Eq(*p.Name)
		})
	}

	if p.Vrf != nil {
		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("vrf").Eq(*p.Vrf)
		})
	}

	if p.Nat != nil {
		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("nat").Eq(*p.Nat)
		})
	}

	if p.PrivateSuper != nil {
		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("privatesuper").Eq(*p.PrivateSuper)
		})
	}

	if p.Underlay != nil {
		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("underlay").Eq(*p.Underlay)
		})
	}

	for k, v := range p.Labels {
		k := k
		v := v
		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("labels").Field(k).Eq(v)
		})
	}

	for _, prefix := range p.Prefixes {
		pfx, err := netip.ParsePrefix(prefix)
		if err != nil {
			return nil, fmt.Errorf("unable to parse prefix %w", err)
		}
		ip := pfx.Addr()
		length := pfx.Bits()

		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("prefixes").Map(func(p r.Term) r.Term {
				return p.Field("ip")
			}).Contains(r.Expr(ip.String()))
		})

		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("prefixes").Map(func(p r.Term) r.Term {
				return p.Field("length")
			}).Contains(r.Expr(strconv.Itoa(length)))
		})
	}

	for _, destPrefix := range p.DestinationPrefixes {
		pfx, err := netip.ParsePrefix(destPrefix)
		if err != nil {
			return nil, fmt.Errorf("unable to parse prefix %w", err)
		}
		ip := pfx.Addr()
		length := pfx.Bits()

		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("destinationprefixes").Map(func(dp r.Term) r.Term {
				return dp.Field("ip")
			}).Contains(r.Expr(ip.String()))
		})

		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("destinationprefixes").Map(func(dp r.Term) r.Term {
				return dp.Field("length")
			}).Contains(r.Expr(strconv.Itoa(length)))
		})
	}

	// Could simply check for the addressfamilies field match
	if p.AddressFamily != nil {
		var separator string
		af, err := metal.ToAddressFamily(*p.AddressFamily)
		if err != nil {
			return nil, err
		}
		switch af {
		case metal.IPv4AddressFamily:
			separator = "\\."
		case metal.IPv6AddressFamily:
			separator = ":"
		case metal.InvalidAddressFamily:
			return nil, fmt.Errorf("given addressfamily is invalid:%s", af)
		}

		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("prefixes").Contains(func(p r.Term) r.Term {
				return p.Field("ip").Match(separator)
			})
		})
	}

	return &q, nil
}

// FindNetworkByID returns an network of a given id.
func (rs *RethinkStore) FindNetworkByID(id string) (*metal.Network, error) {
	var nw metal.Network
	err := rs.findEntityByID(rs.networkTable(), &nw, id)
	if err != nil {
		return nil, err
	}
	return &nw, nil
}

// FindNetwork returns a machine by the given query, fails if there is no record or multiple records found.
func (rs *RethinkStore) FindNetwork(q *NetworkSearchQuery, n *metal.Network) error {
	term, err := q.generateTerm(rs)
	if err != nil {
		return err
	}
	return rs.findEntity(term, &n)
}

// SearchNetworks returns the networks that match the given properties
func (rs *RethinkStore) SearchNetworks(q *NetworkSearchQuery, ns *metal.Networks) error {
	term, err := q.generateTerm(rs)
	if err != nil {
		return err
	}
	return rs.searchEntities(term, ns)
}

// ListNetworks returns all networks.
func (rs *RethinkStore) ListNetworks() (metal.Networks, error) {
	nws := make(metal.Networks, 0)
	err := rs.listEntities(rs.networkTable(), &nws)
	return nws, err
}

// CreateNetwork creates a new network.
func (rs *RethinkStore) CreateNetwork(nw *metal.Network) error {
	return rs.createEntity(rs.networkTable(), nw)
}

// DeleteNetwork deletes an network.
func (rs *RethinkStore) DeleteNetwork(nw *metal.Network) error {
	return rs.deleteEntity(rs.networkTable(), nw)
}

// UpdateNetwork updates an network.
func (rs *RethinkStore) UpdateNetwork(oldNetwork *metal.Network, newNetwork *metal.Network) error {
	return rs.updateEntity(rs.networkTable(), newNetwork, oldNetwork)
}
