package v1

import "time"

type VPNResponse struct {
	Address string `json:"address" description:"address of VPN's control plane"`
	AuthKey string `json:"auth_key" description:"auth key to connect to the VPN"`
}

type VPNRequest struct {
	Pid        string         `json:"pid" description:"project ID"`
	Expiration *time.Duration `json:"expiration" description:"expiration time" optional:"true"`
}
