package v1

type FirewallCreateRequest struct {
	MachineAllocateRequest
	FirewallAllocateRequest
}

type FirewallAllocateRequest struct {
	FirewallRules *FirewallRules `json:"firewall_rules" description:"optional egress and ingress firewall rules to deploy during firewall allocation" optional:"true"`
}

type FirewallEgressRule struct {
	Protocol string   `json:"protocol,omitempty" description:"the protocol for the rule, defaults to tcp" enum:"tcp|udp" optional:"true"`
	Ports    []int    `json:"ports" description:"the ports affected by this rule"`
	ToCIDRs  []string `json:"to_cidrs" description:"the cidrs affected by this rule"`
	Comment  string   `json:"comment,omitempty" description:"an optional comment describing what this rule is used for" optional:"true"`
}

type FirewallIngressRule struct {
	Protocol  string   `json:"protocol,omitempty" description:"the protocol for the rule, defaults to tcp" enum:"tcp|udp" optional:"true"`
	Ports     []int    `json:"ports" description:"the ports affected by this rule"`
	ToCIDRs   []string `json:"to_cidrs" description:"the cidrs affected by this rule"`
	FromCIDRs []string `json:"from_cidrs" description:"the cidrs affected by this rule"`
	Comment   string   `json:"comment,omitempty" description:"an optional comment describing what this rule is used for" optional:"true"`
}

type FirewallResponse struct {
	MachineResponse
}

type FirewallFindRequest struct {
	MachineFindRequest
}
