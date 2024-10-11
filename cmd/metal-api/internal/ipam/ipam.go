package ipam

import (
	"context"
	"errors"
	"fmt"
	"net/netip"

	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
	"github.com/metal-stack/metal-lib/pkg/healthstatus"

	"connectrpc.com/connect"
	goipam "github.com/metal-stack/go-ipam"
	apiv1 "github.com/metal-stack/go-ipam/api/v1"

	"github.com/metal-stack/go-ipam/api/v1/apiv1connect"
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
	// Required for healthcheck
	Check(ctx context.Context) (healthstatus.HealthResult, error)
	ServiceName() string
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
	ipamPrefix, err := i.ip.AcquireChildPrefix(ctx, connect.NewRequest(&apiv1.AcquireChildPrefixRequest{
		Cidr:   parentPrefix.String(),
		Length: uint32(childLength),
	}))
	if err != nil {
		return nil, fmt.Errorf("error creating new prefix from:%s in ipam: %w", parentPrefix.String(), err)
	}

	prefix, _, err := metal.NewPrefixFromCIDR(ipamPrefix.Msg.Prefix.Cidr)
	if err != nil {
		return nil, fmt.Errorf("error creating prefix from ipam prefix: %w", err)
	}

	return prefix, nil
}

// ReleaseChildPrefix release a child prefix from a parent prefix in the IPAM.
func (i *ipam) ReleaseChildPrefix(ctx context.Context, childPrefix metal.Prefix) error {
	_, err := i.ip.ReleaseChildPrefix(ctx, connect.NewRequest(&apiv1.ReleaseChildPrefixRequest{
		Cidr: childPrefix.String(),
	}))
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
	ipamIP, err := i.ip.AcquireIP(ctx, connect.NewRequest(&apiv1.AcquireIPRequest{
		PrefixCidr: prefix.String(),
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
	ipamIP, err := i.ip.AcquireIP(ctx, connect.NewRequest(&apiv1.AcquireIPRequest{
		PrefixCidr: prefix.String(),
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
	_, err := i.ip.ReleaseIP(ctx, connect.NewRequest(&apiv1.ReleaseIPRequest{
		PrefixCidr: ip.ParentPrefixCidr,
		Ip:         ip.IPAddress,
	}))
	var connectErr *connect.Error
	if errors.As(err, &connectErr) {
		if connectErr.Code() == connect.CodeNotFound {
			return metal.NotFound("ip not found")
		}
	}
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
	pfx, err := netip.ParsePrefix(cidr)
	if err != nil {
		return nil, err
	}
	af := metal.IPv4AddressFamily
	if pfx.Addr().Is6() {
		af = metal.IPv6AddressFamily
	}
	availableIPs := map[metal.AddressFamily]uint64{
		af: usage.Msg.AvailableIps,
	}
	usedIPs := map[metal.AddressFamily]uint64{
		af: usage.Msg.AcquiredIps,
	}
	availablePrefixes := map[metal.AddressFamily]uint64{
		af: usage.Msg.AvailableSmallestPrefixes,
	}
	usedPrefixes := map[metal.AddressFamily]uint64{
		af: usage.Msg.AcquiredPrefixes,
	}
	return &metal.NetworkUsage{
		AvailableIPs: availableIPs,
		UsedIPs:      usedIPs,
		// FIXME add usage.AvailablePrefixList as already done here
		// https://github.com/metal-stack/metal-api/pull/152/files#diff-fe05f7f1480be933b5c482b74af28c8b9ca7ef2591f8341eb6e6663cbaeda7baR828
		AvailablePrefixes: availablePrefixes,
		UsedPrefixes:      usedPrefixes,
	}, nil
}

// PrefixesOverlapping returns an error if prefixes overlap.
func (i *ipam) PrefixesOverlapping(existingPrefixes metal.Prefixes, newPrefixes metal.Prefixes) error {
	return goipam.PrefixesOverlapping(existingPrefixes.String(), newPrefixes.String())
}

func (i *ipam) ServiceName() string {
	return "ipam"
}

func (i *ipam) Check(ctx context.Context) (healthstatus.HealthResult, error) {
	resp, err := i.ip.Version(ctx, connect.NewRequest(&apiv1.VersionRequest{}))

	if err != nil {
		return healthstatus.HealthResult{
			Status: healthstatus.HealthStatusUnhealthy,
		}, err
	}

	return healthstatus.HealthResult{
		Status:  healthstatus.HealthStatusHealthy,
		Message: fmt.Sprintf("connected to ipam service version:%q", resp.Msg.Revision),
	}, nil
}
