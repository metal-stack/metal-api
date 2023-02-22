package datastore

import (
	"fmt"

	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
	r "gopkg.in/rethinkdb/rethinkdb-go.v6"
)

// SwitchSearchQuery can be used to search switches.
type SwitchSearchQuery struct {
	ID          *string `json:"id" optional:"true"`
	Name        *string `json:"name" optional:"true"`
	PartitionID *string `json:"partitionid" optional:"true"`
	RackID      *string `json:"rackid" optional:"true"`
	OSVendor    *string `json:"osvendor" optional:"true"`
	OSVersion   *string `json:"osversion" optional:"true"`
}

// GenerateTerm generates the project search query term.
func (p *SwitchSearchQuery) generateTerm(rs *RethinkStore) *r.Term {
	q := *rs.switchTable()

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

	if p.PartitionID != nil {
		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("partitionid").Eq(*p.PartitionID)
		})
	}

	if p.RackID != nil {
		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("rackid").Eq(*p.RackID)
		})
	}

	if p.OSVendor != nil {
		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("os").Field("vendor").Eq(*p.OSVendor)
		})
	}

	if p.OSVersion != nil {
		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("os").Field("version").Eq(*p.OSVersion)
		})
	}

	return &q
}

// FindSwitch returns a switch for a given id.
func (rs *RethinkStore) FindSwitch(id string) (*metal.Switch, error) {
	var s metal.Switch
	err := rs.findEntityByID(rs.switchTable(), &s, id)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// ListSwitches returns all known switches.
func (rs *RethinkStore) ListSwitches() (metal.Switches, error) {
	ss := make(metal.Switches, 0)
	err := rs.listEntities(rs.switchTable(), &ss)
	return ss, err
}

// CreateSwitch creates a new switch.
func (rs *RethinkStore) CreateSwitch(s *metal.Switch) error {
	return rs.createEntity(rs.switchTable(), s)
}

// DeleteSwitch deletes a switch.
func (rs *RethinkStore) DeleteSwitch(s *metal.Switch) error {
	return rs.deleteEntity(rs.switchTable(), s)
}

// UpdateSwitch updates a switch.
func (rs *RethinkStore) UpdateSwitch(oldSwitch *metal.Switch, newSwitch *metal.Switch) error {
	return rs.updateEntity(rs.switchTable(), newSwitch, oldSwitch)
}

// SearchSwitches searches for switches by the given parameters.
func (rs *RethinkStore) SearchSwitches(q *SwitchSearchQuery, ss *metal.Switches) error {
	return rs.searchEntities(q.generateTerm(rs), ss)
}

// SearchSwitchesConnectedToMachine searches switches that are connected to the given machine.
func (rs *RethinkStore) SearchSwitchesConnectedToMachine(m *metal.Machine) (metal.Switches, error) {
	switches := metal.Switches{}

	err := rs.SearchSwitches(&SwitchSearchQuery{RackID: &m.RackID}, &switches)
	if err != nil {
		return nil, err
	}

	res := metal.Switches{}
	for _, sw := range switches {
		if _, has := sw.MachineConnections[m.ID]; has {
			res = append(res, sw)
		}
	}

	return res, nil
}

// SetVrfAtSwitches finds the switches connected to the given machine and puts the switch ports into the given vrf.
// Returns the updated switches.
func (rs *RethinkStore) SetVrfAtSwitches(m *metal.Machine, vrf string) (metal.Switches, error) {
	switches, err := rs.SearchSwitchesConnectedToMachine(m)
	if err != nil {
		return nil, err
	}
	newSwitches := make(metal.Switches, 0)
	for i := range switches {
		sw := switches[i]
		oldSwitch := sw
		sw.SetVrfOfMachine(m, vrf)
		err := rs.UpdateSwitch(&oldSwitch, &sw)
		if err != nil {
			return nil, err
		}
		newSwitches = append(newSwitches, sw)
	}
	return newSwitches, nil
}

func (rs *RethinkStore) ConnectMachineWithSwitches(m *metal.Machine) error {
	switches, err := rs.ListSwitches()
	if err != nil {
		return err
	}

	oldSwitches := metal.Switches{}
	newSwitches := metal.Switches{}
	for _, sw := range switches {
		oldSwitch := sw
		if cons := sw.ConnectMachine(m); cons > 0 {
			oldSwitches = append(oldSwitches, oldSwitch)
			newSwitches = append(newSwitches, sw)
		}
	}

	if len(newSwitches) != 2 {
		return fmt.Errorf("machine %v is not connected to exactly two switches, found connections to %d switches", m.ID, len(newSwitches))
	}

	s1 := newSwitches[0]
	s2 := newSwitches[1]
	cons1 := s1.MachineConnections[m.ID]
	cons2 := s2.MachineConnections[m.ID]
	connectionMapError := fmt.Errorf("twin-switches do not have a connection map that is mirrored crosswise for machine %v, switch %v (connections: %v), switch %v (connections: %v)", m.ID, s1.Name, cons1, s2.Name, cons2)
	if len(cons1) != len(cons2) {
		return connectionMapError
	}

	if s1.RackID != s2.RackID {
		return fmt.Errorf("connected switches of a machine must reside in the same rack, rack of switch %s: %s, rack of switch %s: %s, machine: %s", s1.Name, s1.RackID, s2.Name, s2.RackID, m.ID)
	}
	// We detect the rackID of a machine by connections to leaf switches
	m.RackID = s1.RackID
	m.PartitionID = s1.PartitionID

	byNicName, err := s2.MachineConnections.ByNicName()
	if err != nil {
		return err
	}
	for _, con := range s1.MachineConnections[m.ID] {
		if con2, has := byNicName[con.Nic.Name]; has {
			if con.Nic.Name != con2.Nic.Name {
				return connectionMapError
			}
		} else {
			return connectionMapError
		}
	}

	for i := range oldSwitches {
		err = rs.UpdateSwitch(&oldSwitches[i], &newSwitches[i])
		if err != nil {
			return err
		}
	}

	return nil
}
