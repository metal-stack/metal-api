package testdata

import (
	"context"
	"fmt"
	"testing"

	"github.com/metal-stack/metal-api/cmd/metal-api/internal/ipam"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
	r "gopkg.in/rethinkdb/rethinkdb-go.v6"
)

// InitMockIpamData initializes mock data to be stored in the IPAM module
func InitMockIpamData(dbMock *r.Mock, withIP bool) (ipam.IPAMer, error) {
	ipamer := ipam.InitTestIpam(&testing.T{})

	ctx := context.Background()

	// start creating the prefixes in the IPAM
	for _, prefix := range prefixesIPAM {
		err := ipamer.CreatePrefix(ctx, prefix)
		if err != nil {
			return nil, fmt.Errorf("error creating ipam mock data: %w", err)
		}
	}
	for _, prefix := range []metal.Prefix{prefix1, prefix2, prefix3, superPrefix, superPrefixV6} {
		err := ipamer.CreatePrefix(ctx, prefix)
		if err != nil {
			return nil, fmt.Errorf("error creating ipam mock data: %w", err)
		}
	}

	NwIPAM = metal.Network{
		Base: metal.Base{
			ID:          "4",
			Name:        "IPAM Network",
			Description: "description IPAM",
		},
		Prefixes:        prefixesIPAM,
		AddressFamilies: metal.AddressFamilies{metal.IPv4AddressFamily: true},
	}

	// now, let's get an ip from the IPAM for IPAMIP
	if withIP {
		ipAddress, err := ipamer.AllocateIP(ctx, prefixesIPAM[0])
		if err != nil {
			return nil, fmt.Errorf("error creating ipam mock data: %w", err)
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
