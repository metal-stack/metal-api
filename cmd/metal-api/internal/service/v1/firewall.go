package v1

type FirewallCreateRequest struct {
	MachineAllocateRequest
	FirewallAllocateRequest
}

type FirewallAllocateRequest struct {
	Egress  []FirewallEgressRule  `json:"egress,omitempty" description:"list of egress rules to be deployed during firewall allocation" optional:"true"`
	Ingress []FirewallIngressRule `json:"ingress,omitempty" description:"list of ingress rules to be deployed during firewall allocation" optional:"true"`
}

type FirewallEgressRule struct {
	Protocol  string   `json:"protocol,omitempty" description:"the protocol for the rule, defaults to tcp" enum:"tcp|udp" optional:"true"`
	Ports     []int    `json:"ports" description:"the ports affected by this rule"`
	FromCIDRs []string `json:"from_cidrs" description:"the cidrs affected by this rule"`
	Comment   string   `json:"comment,omitempty" description:"an optional comment describing what this rule is used for" optional:"true"`
}

type FirewallIngressRule struct {
	Protocol string `json:"protocol,omitempty" description:"the protocol for the rule, defaults to tcp" enum:"tcp|udp" optional:"true"`
	Ports    []int  `json:"ports" description:"the ports affected by this rule"`
	// no ToCIDRs, destination is always the node network
	Comment string `json:"comment,omitempty" description:"an optional comment describing what this rule is used for" optional:"true"`
}

type FirewallResponse struct {
	MachineResponse
}

type FirewallFindRequest struct {
	MachineFindRequest
}
