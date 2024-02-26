package ipam

import (
	"context"
	"fmt"
	"net/netip"

	"connectrpc.com/connect"
	goipam "github.com/metal-stack/go-ipam"
	apiv1 "github.com/metal-stack/go-ipam/api/v1"
	"github.com/metal-stack/go-ipam/api/v1/apiv1connect"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
)

// A IPAMer is responsible to allocate a IP for a given purpose
// On the other hand it should release the IP.
// Later Implementations should also allocate and release Networks.
type IPAMer interface {
	AllocateIP(ctx context.Context, prefix metal.Prefix) (string, error)
	AllocateSpecificIP(ctx context.Context, prefix metal.Prefix, specificIP string) (string, error)
	ReleaseIP(ctx context.Context, ip metal.IP) error
	AllocateChildPrefix(ctx context.Context, parentPrefix metal.Prefix, childLength uint8) (*metal.Prefix, error)
	ReleaseChildPrefix(ctx context.Context, childPrefix metal.Prefix) error
	CreatePrefix(ctx context.Context, prefix metal.Prefix) error
	DeletePrefix(ctx context.Context, prefix metal.Prefix) error
	PrefixUsage(ctx context.Context, cidr string) (*metal.NetworkUsage, error)
	PrefixesOverlapping(existingPrefixes metal.Prefixes, newPrefixes metal.Prefixes) error
}

type ipam struct {
	ip apiv1connect.IpamServiceClient
}

// New creates a new IPAM module.
func New(ip apiv1connect.IpamServiceClient) IPAMer {
	return &ipam{
		ip: ip,
	}
}

// AllocateChildPrefix creates a child prefix from a parent prefix in the IPAM.
func (i *ipam) AllocateChildPrefix(ctx context.Context, parentPrefix metal.Prefix, childLength uint8) (*metal.Prefix, error) {
	ipamParentPrefix, err := i.ip.GetPrefix(ctx, connect.NewRequest(&apiv1.GetPrefixRequest{
		Cidr: parentPrefix.String(),
	}))
	if err != nil {
		return nil, err
	}

	if ipamParentPrefix == nil {
		return nil, fmt.Errorf("error finding parent prefix in ipam: %s", parentPrefix.String())
	}

	ipamPrefix, err := i.ip.AcquireChildPrefix(ctx, connect.NewRequest(&apiv1.AcquireChildPrefixRequest{
		Cidr:   ipamParentPrefix.Msg.Prefix.Cidr,
		Length: uint32(childLength),
	}))
	if err != nil {
		return nil, fmt.Errorf("error creating new prefix in ipam: %w", err)
	}

	prefix, err := metal.NewPrefixFromCIDR(ipamPrefix.Msg.Prefix.Cidr)
	if err != nil {
		return nil, fmt.Errorf("error creating prefix from ipam prefix: %w", err)
	}

	return prefix, nil
}

// ReleaseChildPrefix release a child prefix from a parent prefix in the IPAM.
<<<<<<< HEAD
func (i *ipam) ReleaseChildPrefix(ctx context.Context, childPrefix metal.Prefix) error {
	ipamChildPrefix, err := i.ip.GetPrefix(ctx, connect.NewRequest(&apiv1.GetPrefixRequest{
		Cidr: childPrefix.String(),
	}))
	if err != nil {
		return err
	}
	if ipamChildPrefix == nil {
		return fmt.Errorf("error finding child prefix in ipam: %s", childPrefix.String())
	}

	_, err = i.ip.ReleaseChildPrefix(ctx, connect.NewRequest(&apiv1.ReleaseChildPrefixRequest{
		Cidr: ipamChildPrefix.Msg.Prefix.Cidr,
	}))
=======
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
>>>>>>> 9858b21434fd5424b042d55fbef9087984cb1fe9
	if err != nil {
		return fmt.Errorf("error releasing child prefix in ipam: %w", err)
	}

	return nil
}

// CreatePrefix creates a prefix in the IPAM.
func (i *ipam) CreatePrefix(ctx context.Context, prefix metal.Prefix) error {
	_, err := i.ip.CreatePrefix(ctx, connect.NewRequest(&apiv1.CreatePrefixRequest{
		Cidr: prefix.String(),
	}))
	if err != nil {
		return fmt.Errorf("unable to create prefix in ipam: %w", err)
	}
	return nil
}

// DeletePrefix remove a prefix in the IPAM.
func (i *ipam) DeletePrefix(ctx context.Context, prefix metal.Prefix) error {
	_, err := i.ip.DeletePrefix(ctx, connect.NewRequest(&apiv1.DeletePrefixRequest{
		Cidr: prefix.String(),
	}))
	if err != nil {
		return err
	}
	return nil
}

// AllocateIP an ip in the IPAM and returns the allocated IP as a string.
func (i *ipam) AllocateIP(ctx context.Context, prefix metal.Prefix) (string, error) {
	ipamPrefix, err := i.ip.GetPrefix(ctx, connect.NewRequest(&apiv1.GetPrefixRequest{
		Cidr: prefix.String(),
	}))
	if err != nil {
		return "", fmt.Errorf("unable to find prefix:%q %w", prefix, err)
	}
	if ipamPrefix == nil {
		return "", fmt.Errorf("error finding prefix in ipam: %s", prefix.String())
	}

	ipamIP, err := i.ip.AcquireIP(ctx, connect.NewRequest(&apiv1.AcquireIPRequest{
		PrefixCidr: ipamPrefix.Msg.Prefix.Cidr,
		Ip:         nil,
	}))
	if err != nil {
		return "", fmt.Errorf("cannot allocate ip in prefix %s in ipam: %w", prefix.String(), err)
	}
	if ipamIP == nil {
		return "", fmt.Errorf("cannot find free ip to allocate in ipam: %s", prefix.String())
	}

	return ipamIP.Msg.Ip.Ip, nil
}

// AllocateSpecificIP a specific ip in the IPAM and returns the allocated IP as a string.
func (i *ipam) AllocateSpecificIP(ctx context.Context, prefix metal.Prefix, specificIP string) (string, error) {
	ipamPrefix, err := i.ip.GetPrefix(ctx, connect.NewRequest(&apiv1.GetPrefixRequest{
		Cidr: prefix.String(),
	}))
	if err != nil {
		return "", err
	}
	if ipamPrefix == nil {
		return "", fmt.Errorf("error finding prefix in ipam: %s", prefix.String())
	}
	ipamIP, err := i.ip.AcquireIP(ctx, connect.NewRequest(&apiv1.AcquireIPRequest{
		PrefixCidr: ipamPrefix.Msg.Prefix.Cidr,
		Ip:         &specificIP,
	}))
	if err != nil {
		return "", fmt.Errorf("cannot allocate ip in prefix %s in ipam: %w", prefix.String(), err)
	}
	if ipamIP == nil {
		return "", fmt.Errorf("cannot find free ip to allocate in ipam: %s", prefix.String())
	}

	return ipamIP.Msg.Ip.Ip, nil
}

// ReleaseIP an ip in the IPAM.
func (i *ipam) ReleaseIP(ctx context.Context, ip metal.IP) error {
	ipamPrefix, err := i.ip.GetPrefix(ctx, connect.NewRequest(&apiv1.GetPrefixRequest{
		Cidr: ip.ParentPrefixCidr,
	}))
	if err != nil {
		return err
	}
	if ipamPrefix == nil {
		return fmt.Errorf("error finding parent prefix %s of ip %s in ipam", ip.ParentPrefixCidr, ip.IPAddress)
	}

	_, err = i.ip.ReleaseIP(ctx, connect.NewRequest(&apiv1.ReleaseIPRequest{
		PrefixCidr: ipamPrefix.Msg.Prefix.Cidr,
		Ip:         ip.IPAddress,
	}))
	if err != nil {
		return fmt.Errorf("error release ip %s in prefix %s: %w", ip.IPAddress, ip.ParentPrefixCidr, err)
	}
	return nil
}

// PrefixUsage calculates the IP and Prefix Usage
func (i *ipam) PrefixUsage(ctx context.Context, cidr string) (*metal.NetworkUsage, error) {
	usage, err := i.ip.PrefixUsage(ctx, connect.NewRequest(&apiv1.PrefixUsageRequest{
		Cidr: cidr,
	}))
	if err != nil {
		return nil, fmt.Errorf("prefix usage for cidr:%s not found %w", cidr, err)
	}
	return &metal.NetworkUsage{
		AvailableIPs: usage.Msg.AvailableIps,
		UsedIPs:      usage.Msg.AcquiredIps,
		// FIXME add usage.AvailablePrefixList as already done here
		// https://github.com/metal-stack/metal-api/pull/152/files#diff-fe05f7f1480be933b5c482b74af28c8b9ca7ef2591f8341eb6e6663cbaeda7baR828
		AvailablePrefixes: usage.Msg.AvailableSmallestPrefixes,
		UsedPrefixes:      usage.Msg.AcquiredPrefixes,
	}, nil
}

// PrefixesOverlapping returns an error if prefixes overlap.
func (i *ipam) PrefixesOverlapping(existingPrefixes metal.Prefixes, newPrefixes metal.Prefixes) error {
	return goipam.PrefixesOverlapping(existingPrefixes.String(), newPrefixes.String())
}
