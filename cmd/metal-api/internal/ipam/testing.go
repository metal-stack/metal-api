package ipam

import (
	"testing"

	goipam "github.com/metal-pod/go-ipam"
)

func InitTestIpam(t *testing.T) *Ipam {
	ipamInstance := goipam.New()
	ipamer := New(ipamInstance)
	return ipamer
}
