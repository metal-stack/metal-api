package ipam

import (
	"fmt"
	"net/netip"

	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"

	ipam "github.com/metal-stack/go-ipam"
)

type Ipam struct {
	ip ipam.Ipamer
}

// New creates a new IPAM module.
func New(ip ipam.Ipamer) *Ipam {
	return &Ipam{
		ip: ip,
	}
}

// AllocateChildPrefix creates a child prefix from a parent prefix in the IPAM.
func (i *Ipam) AllocateChildPrefix(parentPrefix metal.Prefix, childLength uint8) (*metal.Prefix, error) {
	ipamParentPrefix := i.ip.PrefixFrom(parentPrefix.String())

	if ipamParentPrefix == nil {
		return nil, fmt.Errorf("error finding parent prefix in ipam: %s", parentPrefix.String())
	}

	ipamPrefix, err := i.ip.AcquireChildPrefix(ipamParentPrefix.Cidr, childLength)
	if err != nil {
		return nil, fmt.Errorf("error creating new prefix in ipam: %w", err)
	}

	prefix, err := metal.NewPrefixFromCIDR(ipamPrefix.Cidr)
	if err != nil {
		return nil, fmt.Errorf("error creating prefix from ipam prefix: %w", err)
	}

	return prefix, nil
}

// ReleaseChildPrefix release a child prefix from a parent prefix in the IPAM.
func (i *Ipam) ReleaseChildPrefix(childPrefix metal.Prefix) error {
	_, err := netip.ParsePrefix(childPrefix.String())
	if err != nil {
		return fmt.Errorf("invalid child prefix: %w", err)
	}

	ipamChildPrefix := i.ip.PrefixFrom(childPrefix.String())
	if ipamChildPrefix == nil {
		// FIXME: unfortunately, go-ipam does not return a proper error here so we cannot deduce if the prefix
		// was already deleted or not, so if the database is down or something we continue with network deletion
		// even though there could be remainings in the go-ipam db
		return nil
	}

	err = i.ip.ReleaseChildPrefix(ipamChildPrefix)
	if err != nil {
		return fmt.Errorf("error releasing child prefix in ipam: %w", err)
	}

	return nil
}

// CreatePrefix creates a prefix in the IPAM.
func (i *Ipam) CreatePrefix(prefix metal.Prefix) error {
	_, err := i.ip.NewPrefix(prefix.String())
	if err != nil {
		return fmt.Errorf("unable to create prefix in ipam: %w", err)
	}
	return nil
}

// DeletePrefix remove a prefix in the IPAM.
func (i *Ipam) DeletePrefix(prefix metal.Prefix) error {
	_, err := i.ip.DeletePrefix(prefix.String())
	if err != nil {
		return err
	}
	return nil
}

// AllocateIP an ip in the IPAM and returns the allocated IP as a string.
func (i *Ipam) AllocateIP(prefix metal.Prefix) (string, error) {
	ipamPrefix := i.ip.PrefixFrom(prefix.String())
	if ipamPrefix == nil {
		return "", fmt.Errorf("error finding prefix in ipam: %s", prefix.String())
	}

	ipamIP, err := i.ip.AcquireIP(ipamPrefix.Cidr)
	if err != nil {
		return "", fmt.Errorf("cannot allocate ip in prefix %s in ipam: %w", prefix.String(), err)
	}
	if ipamIP == nil {
		return "", fmt.Errorf("cannot find free ip to allocate in ipam: %s", prefix.String())
	}

	return ipamIP.IP.String(), nil
}

// AllocateSpecificIP a specific ip in the IPAM and returns the allocated IP as a string.
func (i *Ipam) AllocateSpecificIP(prefix metal.Prefix, specificIP string) (string, error) {
	ipamPrefix := i.ip.PrefixFrom(prefix.String())
	if ipamPrefix == nil {
		return "", fmt.Errorf("error finding prefix in ipam: %s", prefix.String())
	}
	ipamIP, err := i.ip.AcquireSpecificIP(ipamPrefix.Cidr, specificIP)
	if err != nil {
		return "", fmt.Errorf("cannot allocate ip in prefix %s in ipam: %w", prefix.String(), err)
	}
	if ipamIP == nil {
		return "", fmt.Errorf("cannot find free ip to allocate in ipam: %s", prefix.String())
	}

	return ipamIP.IP.String(), nil
}

// ReleaseIP an ip in the IPAM.
func (i *Ipam) ReleaseIP(ip metal.IP) error {
	ipamPrefix := i.ip.PrefixFrom(ip.ParentPrefixCidr)
	if ipamPrefix == nil {
		return fmt.Errorf("error finding parent prefix %s of ip %s in ipam", ip.ParentPrefixCidr, ip.IPAddress)
	}

	err := i.ip.ReleaseIPFromPrefix(ipamPrefix.Cidr, ip.IPAddress)
	if err != nil {
		return fmt.Errorf("error release ip %s in prefix %s: %w", ip.IPAddress, ip.ParentPrefixCidr, err)
	}
	return nil
}

// PrefixUsage calculates the IP and Prefix Usage
func (i *Ipam) PrefixUsage(cidr string) (*metal.NetworkUsage, error) {
	prefix := i.ip.PrefixFrom(cidr)
	if prefix == nil {
		return nil, fmt.Errorf("prefix for cidr:%s not found", cidr)
	}
	usage := prefix.Usage()

	return &metal.NetworkUsage{
		AvailableIPs: usage.AvailableIPs,
		UsedIPs:      usage.AcquiredIPs,
		// FIXME add usage.AvailablePrefixList as already done here https://github.com/metal-stack/metal-api/pull/152/files#diff-fe05f7f1480be933b5c482b74af28c8b9ca7ef2591f8341eb6e6663cbaeda7baR828
		AvailablePrefixes: usage.AvailableSmallestPrefixes,
		UsedPrefixes:      usage.AcquiredPrefixes,
	}, nil
}

// PrefixesOverlapping returns an error if prefixes overlap.
func (i *Ipam) PrefixesOverlapping(existingPrefixes metal.Prefixes, newPrefixes metal.Prefixes) error {
	return i.ip.PrefixesOverlapping(existingPrefixes.String(), newPrefixes.String())
}
