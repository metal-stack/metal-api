package testdata

import (
	"fmt"
	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/ipam"
	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/metal"
	r "gopkg.in/rethinkdb/rethinkdb-go.v5"

	goipam "github.com/metal-pod/go-ipam"
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
		fmt.Printf("jsk: %#v", TestIPs)
		IPAMIP.IPAddress = ""
		IPAMIP.ParentPrefixCidr = ""
		IPAMIP.NetworkID = ""
	}

	dbMock.On(r.DB("mockdb").Table("ip").Get(IPAMIP.IPAddress)).Return(IPAMIP, nil)
	dbMock.On(r.DB("mockdb").Table("network").Get(NwIPAM.ID)).Return(NwIPAM, nil)

	return ipamer, nil
}
