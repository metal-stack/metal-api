package datastore

import (
	"strconv"

	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/metal"
	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/utils"
	r "gopkg.in/rethinkdb/rethinkdb-go.v5"
)

// NetworkSearchQuery can be used to search networks.
type NetworkSearchQuery struct {
	ID                  *string           `json:"id"`
	Name                *string           `json:"name"`
	PartitionID         *string           `json:"partitionid"`
	ProjectID           *string           `json:"projectid"`
	Prefixes            []string          `json:"prefixes"`
	DestinationPrefixes []string          `json:"destinationprefixes"`
	Nat                 *bool             `json:"nat"`
	PrivateSuper        *bool             `json:"privatesuper"`
	Underlay            *bool             `json:"underlay"`
	Vrf                 *int64            `json:"vrf"`
	ParentNetworkID     *string           `json:"parentnetworkid"`
	Labels              map[string]string `json:"labels"`
}

// GenerateTerm generates the project search query term.
func (p *NetworkSearchQuery) generateTerm(rs *RethinkStore) *r.Term {
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

	if p.Labels != nil {
		q = q.Filter(map[string]interface{}{"labels": p.Labels})
	}

	for _, prefix := range p.Prefixes {
		ip, length := utils.SplitCIDR(prefix)

		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("prefixes").Map(func(p r.Term) r.Term {
				return p.Field("ip")
			}).Contains(r.Expr(ip))
		})

		if length != nil {
			q = q.Filter(func(row r.Term) r.Term {
				return row.Field("prefixes").Map(func(p r.Term) r.Term {
					return p.Field("length")
				}).Contains(r.Expr(strconv.Itoa(*length)))
			})
		}
	}

	for _, destPrefix := range p.DestinationPrefixes {
		ip, length := utils.SplitCIDR(destPrefix)

		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("destinationprefixes").Map(func(dp r.Term) r.Term {
				return dp.Field("ip")
			}).Contains(r.Expr(ip))
		})

		if length != nil {
			q = q.Filter(func(row r.Term) r.Term {
				return row.Field("destinationprefixes").Map(func(dp r.Term) r.Term {
					return dp.Field("length")
				}).Contains(r.Expr(strconv.Itoa(*length)))
			})
		}
	}

	return &q
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
	return rs.findEntity(q.generateTerm(rs), &n)
}

// SearchNetworks returns the networks that match the given properties
func (rs *RethinkStore) SearchNetworks(q *NetworkSearchQuery, ns *metal.Networks) error {
	return rs.searchEntities(q.generateTerm(rs), ns)
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
