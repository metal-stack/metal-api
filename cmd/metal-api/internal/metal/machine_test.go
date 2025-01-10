package metal

import (
	"testing"
)

func TestMachine_HasMAC(t *testing.T) {
	tests := []struct {
		name string
		d    *Machine
		mac  string
		want bool
	}{
		{
			name: "Test 1",
			d: &Machine{
				Base: Base{
					Name:        "1-core/100 B",
					Description: "a machine with 1 core(s) and 100 B of RAM",
					ID:          "5",
				},
				RackID:      "1",
				PartitionID: "1",
				SizeID:      "1",
				Allocation:  nil,
				Hardware: MachineHardware{
					Memory: 100,
					MetalCPUs: []MetalCPU{
						{
							Model:   "Intel Xeon Silver",
							Cores:   1,
							Threads: 1,
						},
					},
					Nics: Nics{
						Nic{
							MacAddress: "11:11:11:11:11:11",
						},
						Nic{
							MacAddress: "21:11:11:11:11:11",
						},
					},
					Disks: []BlockDevice{
						{
							Name: "blockdeviceName",
							Size: 1000000000000,
						},
					},
				},
			},
			mac:  "11:11:11:11:11:11",
			want: true,
		},
		{
			name: "Test 2",
			d: &Machine{
				Base:        Base{ID: "1"},
				PartitionID: "1",
				SizeID:      "1",
				Allocation: &MachineAllocation{
					Name:    "d1",
					ImageID: "1",
				},
			},
			mac:  "11:11:11:11:11:11",
			want: false,
		},
	}

	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.d.HasMAC(tt.mac); got != tt.want {
				t.Errorf("Machine.HasMAC() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMachineNetwork_NetworkType(t *testing.T) {
	type fields struct {
		PrivatePrimary bool
		Private        bool
		Underlay       bool
		Shared         bool
	}
	tests := []struct {
		name    string
		fields  fields
		want    NetworkType
		wantErr bool
	}{
		{
			name: "private primary unshared",
			fields: fields{
				PrivatePrimary: true,
				Private:        true,
				Underlay:       false,
				Shared:         false,
			},
			want: PrivatePrimaryUnshared,
		},
		{
			name: "private primary shared",
			fields: fields{
				PrivatePrimary: true,
				Private:        true,
				Underlay:       false,
				Shared:         true,
			},
			want: PrivatePrimaryShared,
		},
		{
			name: "private secondary shared",
			fields: fields{
				PrivatePrimary: false,
				Private:        true,
				Underlay:       false,
				Shared:         true,
			},
			want: PrivateSecondaryShared,
		},
		{
			name: "public network",
			fields: fields{
				PrivatePrimary: false,
				Private:        false,
				Underlay:       false,
				Shared:         false,
			},
			want: External,
		},
		{
			name: "try to specify a private primary network with private false",
			fields: fields{
				PrivatePrimary: true,
				Private:        false,
				Underlay:       false,
				Shared:         true,
			},
			wantErr: true,
		},
		{
			name: "machine network from old allocation guessed to a primaryprivateunshared",
			fields: fields{
				PrivatePrimary: false,
				Private:        true,
				Underlay:       false,
				Shared:         false,
			},
			wantErr: false,
			want:    PrivatePrimaryUnshared,
		},
		{
			name: "unsupported networktype public shared",
			fields: fields{
				PrivatePrimary: false,
				Private:        false,
				Underlay:       false,
				Shared:         true,
			},
			wantErr: true,
		},
		{
			name: "unsupported networktype underlay private",
			fields: fields{
				PrivatePrimary: false,
				Private:        true,
				Underlay:       true,
				Shared:         true,
			},
			wantErr: true,
		},
		{
			name: "underlay",
			fields: fields{
				Underlay: true,
			},
			want: Underlay,
		},
	}
	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			mn := &MachineNetwork{
				PrivatePrimary: tt.fields.PrivatePrimary,
				Private:        tt.fields.Private,
				Underlay:       tt.fields.Underlay,
				Shared:         tt.fields.Shared,
			}
			got, err := mn.NetworkType()
			if (err != nil) != tt.wantErr {
				t.Errorf("MachineNetwork.NetworkType() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != nil && *got != tt.want {
				t.Errorf("MachineNetwork.NetworkType() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TODO: Write tests for machine allocation

func TestEgressRule_Validate(t *testing.T) {
	tests := []struct {
		name       string
		Protocol   Protocol
		Ports      []int
		To         []string
		Comment    string
		wantErr    bool
		wantErrmsg string
	}{
		{
			name:     "valid egress rule",
			Protocol: ProtocolTCP,
			Ports:    []int{1, 2, 3},
			To:       []string{"1.2.3.0/24", "2.3.4.5/32"},
			Comment:  "allow apt update",
		},
		{
			name:       "wrong protocol",
			Protocol:   Protocol("sctp"),
			Ports:      []int{1, 2, 3},
			To:         []string{"1.2.3.0/24", "2.3.4.5/32"},
			Comment:    "allow apt update",
			wantErr:    true,
			wantErrmsg: "invalid protocol: sctp",
		},
		{
			name:       "wrong port",
			Protocol:   ProtocolTCP,
			Ports:      []int{1, 2, 3, -1},
			To:         []string{"1.2.3.0/24", "2.3.4.5/32"},
			Comment:    "allow apt update",
			wantErr:    true,
			wantErrmsg: "port is out of range",
		},
		{
			name:       "wrong cidr",
			Protocol:   ProtocolTCP,
			Ports:      []int{1, 2, 3},
			To:         []string{"1.2.3.0/24", "2.3.4.5/33"},
			Comment:    "allow apt update",
			wantErr:    true,
			wantErrmsg: "invalid cidr: netip.ParsePrefix(\"2.3.4.5/33\"): prefix length out of range",
		},
		{
			name:       "wrong comment",
			Protocol:   ProtocolTCP,
			Ports:      []int{1, 2, 3},
			To:         []string{"1.2.3.0/24", "2.3.4.5/32"},
			Comment:    "allow apt update\n",
			wantErr:    true,
			wantErrmsg: "illegal character in comment found, only: \"abcdefghijklmnopqrstuvwxyz_- \" allowed",
		},
		{
			name:       "too long comment",
			Protocol:   ProtocolTCP,
			Ports:      []int{1, 2, 3},
			To:         []string{"1.2.3.0/24", "2.3.4.5/32"},
			Comment:    "much too long comment aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			wantErr:    true,
			wantErrmsg: "comments can not exceed 100 characters",
		},
		{
			name:       "mixed address family in cidrs",
			Protocol:   ProtocolTCP,
			Ports:      []int{1, 2, 3},
			To:         []string{"1.2.3.0/24", "2.3.4.5/32", "2001:db8::/32"},
			Comment:    "mixed address family",
			wantErr:    true,
			wantErrmsg: "mixed address family in one rule is not supported:[1.2.3.0/24 2.3.4.5/32 2001:db8::/32]",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := EgressRule{
				Protocol: tt.Protocol,
				Ports:    tt.Ports,
				To:       tt.To,
				Comment:  tt.Comment,
			}
			if err := r.Validate(); (err != nil) != tt.wantErr {
				t.Errorf("EgressRule.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err := r.Validate(); err != nil {
				if tt.wantErrmsg != err.Error() {
					t.Errorf("IngressRule.Validate() error = %v, wantErrmsg %v", err.Error(), tt.wantErrmsg)
				}
			}
		})
	}
}
func TestIngressRule_Validate(t *testing.T) {
	tests := []struct {
		name       string
		Protocol   Protocol
		Ports      []int
		To         []string
		From       []string
		Comment    string
		wantErr    bool
		wantErrmsg string
	}{
		{
			name:     "valid ingress rule",
			Protocol: ProtocolTCP,
			Ports:    []int{1, 2, 3},
			From:     []string{"1.2.3.0/24", "2.3.4.5/32"},
			Comment:  "allow apt update",
		},
		{
			name:     "valid ingress rule",
			Protocol: ProtocolTCP,
			Ports:    []int{1, 2, 3},
			From:     []string{"1.2.3.0/24", "2.3.4.5/32"},
			To:       []string{"100.2.3.0/24", "200.3.4.5/32"},
			Comment:  "allow apt update",
		},
		{
			name:       "invalid ingress rule, mixed address families in to and from",
			Protocol:   ProtocolTCP,
			Ports:      []int{1, 2, 3},
			From:       []string{"1.2.3.0/24", "2.3.4.5/32"},
			To:         []string{"100.2.3.0/24", "2001:db8::/32"},
			Comment:    "allow apt update",
			wantErr:    true,
			wantErrmsg: "mixed address family in one rule is not supported:[100.2.3.0/24 2001:db8::/32]",
		},
		{
			name:       "invalid ingress rule, mixed address families in to and from",
			Protocol:   ProtocolTCP,
			Ports:      []int{1, 2, 3},
			From:       []string{"2.3.4.5/32"},
			To:         []string{"2001:db8::/32"},
			Comment:    "allow apt update",
			wantErr:    true,
			wantErrmsg: "mixed address family in one rule is not supported:[2.3.4.5/32 2001:db8::/32]",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := IngressRule{
				Protocol: tt.Protocol,
				Ports:    tt.Ports,
				To:       tt.To,
				From:     tt.From,
				Comment:  tt.Comment,
			}
			if err := r.Validate(); (err != nil) != tt.wantErr {
				t.Errorf("IngressRule.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err := r.Validate(); err != nil {
				if tt.wantErrmsg != err.Error() {
					t.Errorf("IngressRule.Validate() error = %v, wantErrmsg %v", err.Error(), tt.wantErrmsg)
				}
			}
		})
	}
}
