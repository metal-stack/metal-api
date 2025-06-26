package headscale_test

import (
	"encoding/json"
	"testing"

	policyv2 "github.com/juanfont/headscale/hscontrol/policy/v2"
	"github.com/metal-stack/metal-lib/pkg/pointer"
	"github.com/stretchr/testify/require"
	"tailscale.com/tailcfg"
)

func TestACLs(t *testing.T) {
	defaultACLs := []policyv2.ACL{
		{
			Action: "accept",
			Sources: policyv2.Aliases{
				pointer.Pointer(policyv2.AutoGroupMember),
			},
			Destinations: []policyv2.AliasWithPorts{
				{
					Alias: pointer.Pointer(policyv2.AutoGroupSelf),
					Ports: []tailcfg.PortRange{tailcfg.PortRangeAny},
				},
			},
		},
	}

	policy := policyv2.Policy{
		ACLs: defaultACLs,
	}

	aclBytes, err := json.Marshal(policy)
	require.NoError(t, err)
	require.JSONEq(t, `
{
    "groups": null,
    "hosts": null,
    "tagOwners": null,
    "acls": [
        {
            "action": "accept",
            "proto": "",
            "src": [
                "autogroup:member"
            ],
            "dst": [
                {
                    "Alias": "autogroup:self",
                    "Ports": [
                        {
                            "First": 0,
                            "Last": 65535
                        }
                    ]
                }
            ]
        }
    ],
    "autoApprovers": {
        "routes": null,
        "exitNode": null
    },
    "ssh": null
}
`, string(aclBytes))
}
