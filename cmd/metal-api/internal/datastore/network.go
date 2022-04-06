package datastore

import (
	"context"
	"strconv"

	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/utils"
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

	for k, v := range p.Labels {
		k := k
		v := v
		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("labels").Field(k).Eq(v)
		})
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
func (rs *RethinkStore) FindNetworkByID(ctx context.Context, id string) (*metal.Network, error) {
	var nw metal.Network
	err := rs.findEntityByID(ctx, rs.networkTable(), &nw, id)
	if err != nil {
		return nil, err
	}
	return &nw, nil
}

// FindNetwork returns a machine by the given query, fails if there is no record or multiple records found.
func (rs *RethinkStore) FindNetwork(ctx context.Context, q *NetworkSearchQuery, n *metal.Network) error {
	return rs.findEntity(ctx, q.generateTerm(rs), &n)
}

// SearchNetworks returns the networks that match the given properties
func (rs *RethinkStore) SearchNetworks(ctx context.Context, q *NetworkSearchQuery, ns *metal.Networks) error {
	return rs.searchEntities(ctx, q.generateTerm(rs), ns)
}

// ListNetworks returns all networks.
func (rs *RethinkStore) ListNetworks(ctx context.Context) (metal.Networks, error) {
	nws := make(metal.Networks, 0)
	err := rs.listEntities(ctx, rs.networkTable(), &nws)
	return nws, err
}

// CreateNetwork creates a new network.
func (rs *RethinkStore) CreateNetwork(ctx context.Context, nw *metal.Network) error {
	return rs.createEntity(ctx, rs.networkTable(), nw)
}

// DeleteNetwork deletes an network.
func (rs *RethinkStore) DeleteNetwork(ctx context.Context, nw *metal.Network) error {
	return rs.deleteEntity(ctx, rs.networkTable(), nw)
}

// UpdateNetwork updates an network.
func (rs *RethinkStore) UpdateNetwork(ctx context.Context, oldNetwork *metal.Network, newNetwork *metal.Network) error {
	return rs.updateEntity(ctx, rs.networkTable(), newNetwork, oldNetwork)
}
