package metal

import (
	"github.com/metal-stack/metal-lib/pkg/tag"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestAddMachineId(t *testing.T) {
	tests := []struct {
		name         string
		ip           IP
		expectedTags []string
	}{
		{
			name:         "ip without machine tag",
			ip:           IP{},
			expectedTags: []string{IpTag(tag.MachineID, "123")},
		},
		{
			name: "ip with empty machine tag",
			ip: IP{
				Tags: []string{tag.MachineID},
			},
			expectedTags: []string{IpTag(tag.MachineID, "123")},
		},
		{
			name: "ip with other machine tag",
			ip: IP{
				Tags: []string{IpTag(tag.MachineID, "1")},
			},
			expectedTags: []string{IpTag(tag.MachineID, "1"), IpTag(tag.MachineID, "123")},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.ip.AddMachineId("123")
			if got := tt.ip.Tags; !cmp.Equal(got, tt.expectedTags) {
				t.Errorf("%v", cmp.Diff(got, tt.expectedTags))
			}
		})
	}
}
func TestRemoveMachineId(t *testing.T) {
	tests := []struct {
		name         string
		ip           IP
		expectedTags []string
	}{
		{
			name:         "ip without machine tag",
			ip:           IP{},
			expectedTags: nil,
		},
		{
			name: "ip with empty machine tag",
			ip: IP{
				Tags: []string{tag.MachineID},
			},
			expectedTags: []string{tag.MachineID},
		},
		{
			name: "ip with other machine tag",
			ip: IP{
				Tags: []string{IpTag(tag.MachineID, "1")},
			},
			expectedTags: []string{IpTag(tag.MachineID, "1")},
		},
		{
			name: "ip with matching machine tag",
			ip: IP{
				Tags: []string{IpTag(tag.MachineID, "123")},
			},
			expectedTags: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.ip.RemoveMachineId("123")
			if got := tt.ip.Tags; !cmp.Equal(got, tt.expectedTags) {
				t.Errorf("%v", cmp.Diff(got, tt.expectedTags))
			}
		})
	}
}

func TestGetScope(t *testing.T) {
	tests := []struct {
		name          string
		ip            IP
		expectedScope IPScope
	}{
		{
			name: "empty scope ip",
			ip: IP{
				Tags: []string{IpTag(tag.MachineID, "102")},
			},
			expectedScope: ScopeEmpty,
		},
		{
			name: "machine ip",
			ip: IP{
				ProjectID: "1",
				Tags:      []string{IpTag(tag.MachineID, "102")},
			},
			expectedScope: ScopeMachine,
		},
		{
			name: "project ip",
			ip: IP{
				ProjectID: "1",
				Tags:      nil,
			},
			expectedScope: ScopeProject,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.ip.GetScope(); got != tt.expectedScope {
				t.Errorf("IP.GetScope = %v, want %v", got, tt.expectedScope)
			}
		})
	}
}

func TestIPToASN(t *testing.T) {
	ipaddress := IP{
		IPAddress: "10.0.1.2",
	}

	asn, err := ipaddress.ASN()
	if err != nil {
		t.Errorf("no error expected got:%v", err)
	}

	if asn != 4200000258 {
		t.Errorf("expected 4200000258 got: %d", asn)
	}
}
