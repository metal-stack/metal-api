package v1

type VPNResponse struct {
	AuthKey string `json:"auth_key" description:"Auth key to connect to the VPN"`
}
