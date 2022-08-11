package v1

type VPNResponse struct {
	Address string `json:"address" description:"Address of VPN's control plane"`
	AuthKey string `json:"auth_key" description:"Auth key to connect to the VPN"`
}
