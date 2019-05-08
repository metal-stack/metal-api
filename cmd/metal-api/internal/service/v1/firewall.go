package v1

// type FirewallNetwork struct {
// 	NetworkID string   `json:"networkid" description:"the networkID of the allocated machine in this vrf"`
// 	IPs       []string `json:"ips" description:"the ip addresses of the allocated machine in this vrf"`
// 	Vrf       uint     `json:"vrf" description:"the vrf of the allocated machine"`
// 	Primary   bool     `json:"primary" description:"this network is the primary vrf of the allocated machine, aka tenant vrf"`
// 	ASN       int      `json:"asn" description:"ASN number for this network in the bgp configuration"`
// }

type FirewallCreateRequest struct {
	MachineAllocateRequest
	NetworkIDs []string `json:"networks" description:"the networks of this firewall"`
	// IPs        []string `json:"ips" description:"the additional ips of this firewall"`
	HA *bool `json:"ha" description:"if set to true, this firewall is set up in a High Available manner" optional:"true"`
}
