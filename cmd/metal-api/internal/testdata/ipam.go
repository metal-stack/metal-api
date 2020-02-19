package testdata

import (
	"fmt"

	"github.com/metal-stack/metal-api/cmd/metal-api/internal/ipam"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
	r "gopkg.in/rethinkdb/rethinkdb-go.v5"

	goipam "github.com/metal-stack/go-ipam"
)

// InitMockIpamData initializes mock data to be stored in the IPAM module
func InitMockIpamData(dbMock *r.Mock, withIP bool) (*ipam.Ipam, error) {
	ip := goipam.New()
	ipamer := ipam.New(ip)

	// start creating the prefixes in the IPAM
	for _, prefix := range prefixesIPAM {
		err := ipamer.CreatePrefix(prefix)
		if err != nil {
			return nil, fmt.Errorf("error creating ipam mock data: %v", err)
		}
	}
	for _, prefix := range []metal.Prefix{prefix1, prefix2, prefix3} {
		err := ipamer.CreatePrefix(prefix)
		if err != nil {
			return nil, fmt.Errorf("error creating ipam mock data: %v", err)
		}
	}

	NwIPAM = metal.Network{
		Base: metal.Base{
			ID:          "4",
			Name:        "IPAM Network",
			Description: "description IPAM",
		},
		Prefixes: prefixesIPAM,
	}

	// now, let's get an ip from the IPAM for IPAMIP
	if withIP {
		ipAddress, err := ipamer.AllocateIP(prefixesIPAM[0])
		if err != nil {
			return nil, fmt.Errorf("error creating ipam mock data: %v", err)
		}
		IPAMIP.IPAddress = ipAddress
		IPAMIP.ParentPrefixCidr = prefixesIPAM[0].String()
		IPAMIP.NetworkID = NwIPAM.ID
		TestIPs = append(TestIPs, IPAMIP)
	} else {
		if len(TestIPs) > 3 {
			TestIPs = TestIPs[:3]
		}
		IPAMIP.IPAddress = ""
		IPAMIP.ParentPrefixCidr = ""
		IPAMIP.NetworkID = ""
	}

	dbMock.On(r.DB("mockdb").Table("ip").Get(IPAMIP.IPAddress)).Return(IPAMIP, nil)
	dbMock.On(r.DB("mockdb").Table("network").Get(NwIPAM.ID)).Return(NwIPAM, nil)

	return ipamer, nil
}
