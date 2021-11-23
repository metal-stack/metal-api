package v1

type FirewallCreateRequest struct {
	MachineAllocateRequest
	// HA if set to true firewall is created in ha configuration
	//
	// Deprecated: will be removed in the next release
	HA *bool `json:"ha" description:"if set to true, this firewall is set up in a High Available manner" optional:"true"`
}

type FirewallResponse struct {
	MachineResponse
}

type FirewallFindRequest struct {
	MachineFindRequest
}
