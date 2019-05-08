package metal

import (
	"net"
	"time"

	"github.com/pkg/errors"
)

// IP of a machine/firewall.
type IP struct {
	IPAddress        string    `rethinkdb:"id"`
	ParentPrefixCidr string    `rethinkdb:"prefix"`
	Name             string    `rethinkdb:"name"`
	Description      string    `rethinkdb:"description"`
	Created          time.Time `rethinkdb:"created"`
	Changed          time.Time `rethinkdb:"changed"`
	NetworkID        string    `rethinkdb:"networkid"`
	ProjectID        string    `rethinkdb:"projectid"`
}

func (ip *IP) GetID() string {
	return ip.IPAddress
}

func (ip *IP) SetID(id string) {
	ip.IPAddress = id
}

func (ip *IP) GetChanged() time.Time {
	return ip.Changed
}

func (ip *IP) SetChanged(changed time.Time) {
	ip.Changed = changed
}

func (ip *IP) GetCreated() time.Time {
	return ip.Created
}

func (ip *IP) SetCreated(created time.Time) {
	ip.Created = created
}

// ASN calculate a ASN from the ip
// we start to calculate ASNs for machines with the first ASN in the 32bit ASN range and
// add the last 2 octets of the ip of the machine to achieve unique ASNs per vrf
func (ip *IP) ASN() (int64, error) {
	const asnbase = 4200000000

	i := net.ParseIP(ip.IPAddress)
	if i == nil {
		return int64(-1), errors.Errorf("unable to parse ip %s", ip.IPAddress)
	}
	asn := asnbase + int64(i[14])*256 + int64(i[15])
	return asn, nil
}
