package ipam

import (
	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/metal"
	v1 "git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/service/v1"
)

// A IPAMer is responsible to allocate a IP for a given purpose
// On the other hand it should release the IP.
// Later Implementations should also allocate and release Networks.
type IPAMer interface {
	AllocateIP(prefix metal.Prefix) (string, error)
	ReleaseIP(ip metal.IP) error
	AllocateChildPrefix(parentPrefix metal.Prefix, childLength int) (*metal.Prefix, error)
	CreatePrefix(prefix metal.Prefix) error
	DeletePrefix(prefix metal.Prefix) error
	PrefixUsage(cidr string) (*v1.NetworkUsage, error) // TODO: Also wrap usage, such that it is independent of v1
	PrefixesOverlapping(existingPrefixes metal.Prefixes, newPrefixes metal.Prefixes) error
}
