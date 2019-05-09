package v1

type FirewallCreateRequest struct {
	MachineAllocateRequest
	NetworkIDs []string `json:"networks" description:"the networks of this firewall"`
	// IPs        []string `json:"ips" description:"the additional ips of this firewall"`
	HA *bool `json:"ha" description:"if set to true, this firewall is set up in a High Available manner" optional:"true"`
}

type FirewallListResponse struct {
	MachineListResponse
}

type FirewallDetailResponse struct {
	MachineDetailResponse
}
