package ipam

import (
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
)

// A IPAMer is responsible to allocate a IP for a given purpose
// On the other hand it should release the IP.
// Later Implementations should also allocate and release Networks.
type IPAMer interface {
	AllocateIP(prefix metal.Prefix) (string, error)
	AllocateSpecificIP(prefix metal.Prefix, specificIP string) (string, error)
	ReleaseIP(ip metal.IP) error
	AllocateChildPrefix(parentPrefix metal.Prefix, childLength uint8) (*metal.Prefix, error)
	ReleaseChildPrefix(childPrefix metal.Prefix) error
	CreatePrefix(prefix metal.Prefix) error
	DeletePrefix(prefix metal.Prefix) error
	PrefixUsage(cidr string) (*metal.NetworkUsage, error)
	// PrefixesOverlapping(existingPrefixes metal.Prefixes, newPrefixes metal.Prefixes) error
}
