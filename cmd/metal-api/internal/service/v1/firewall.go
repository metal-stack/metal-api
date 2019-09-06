package v1

type FirewallCreateRequest struct {
	MachineAllocateRequest
	HA *bool `json:"ha" description:"if set to true, this firewall is set up in a High Available manner" optional:"true"`
}

type FirewallResponse struct {
	MachineResponse
}

type FirewallFindRequest struct {
	MachineFindRequest
}
