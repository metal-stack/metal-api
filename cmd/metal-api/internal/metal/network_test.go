package metal

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/metal-stack/metal-lib/pkg/pointer"
)

func TestNics_ByIdentifier(t *testing.T) {
	// Create Nics
	countOfNics := 3
	nicArray := make([]Nic, countOfNics)
	for i := range countOfNics {
		nicArray[i] = Nic{
			MacAddress: MacAddress("11:11:1" + fmt.Sprintf("%d", i)),
			Name:       "swp" + fmt.Sprintf("%d", i),
			Neighbors:  nil,
		}
	}

	// all have all as Neighbors
	for i := range countOfNics {
		nicArray[i].Neighbors = append(nicArray[0:i], nicArray[i+1:countOfNics]...)
	}

	map1 := map[string]*Nic{}
	for i, n := range nicArray {
		map1[string(n.MacAddress)] = &nicArray[i]
	}

	tests := []struct {
		name string
		nics Nics
		want map[string]*Nic
	}{
		{
			name: "TestNics_ByIdentifier Test 1",
			nics: nicArray,
			want: map1,
		},
	}
	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.nics.ByIdentifier(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Nics.ByIdentifier() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPrefix_Equals(t *testing.T) {
	type fields struct {
		IP     string
		Length string
	}
	type args struct {
		other *Prefix
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name: "test prefix equals",
			fields: fields{
				IP:     "10.0.0.0",
				Length: "18",
			},
			args: args{
				other: &Prefix{
					IP:     "10.0.0.0",
					Length: "18",
				},
			},
			want: true,
		},
		{
			name: "test prefix not equals 1",
			fields: fields{
				IP:     "10.0.0.0",
				Length: "18",
			},
			args: args{
				other: &Prefix{
					IP:     "10.0.0.0",
					Length: "20",
				},
			},
			want: false,
		},
		{
			name: "test prefix not equals 2",
			fields: fields{
				IP:     "10.0.0.1",
				Length: "18",
			},
			args: args{
				other: &Prefix{
					IP:     "10.0.0.0",
					Length: "18",
				},
			},
			want: false,
		},
	}
	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			p := &Prefix{
				IP:     tt.fields.IP,
				Length: tt.fields.Length,
			}
			if got := p.equals(tt.args.other); got != tt.want {
				t.Errorf("Prefix.Equals() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNicState_WantState(t *testing.T) {
	up := SwitchPortStatusUp
	down := SwitchPortStatusDown
	unknown := SwitchPortStatusUnknown

	tests := []struct {
		name    string
		nic     *NicState
		arg     SwitchPortStatus
		want    NicState
		changed bool
	}{
		{
			name: "up to desired down",
			nic: &NicState{
				Desired: nil,
				Actual:  down,
			},
			arg: up,
			want: NicState{
				Desired: &up,
				Actual:  down,
			},
			changed: true,
		},
		{
			name: "up to up with empty desired",
			nic: &NicState{
				Desired: nil,
				Actual:  up,
			},
			arg: up,
			want: NicState{
				Desired: nil,
				Actual:  up,
			},
			changed: false,
		},
		{
			name: "up to up with other desired",
			nic: &NicState{
				Desired: &down,
				Actual:  up,
			},
			arg: up,
			want: NicState{
				Desired: nil,
				Actual:  up,
			},
			changed: true,
		},
		{
			name: "nil to up",
			nic:  nil,
			arg:  up,
			want: NicState{
				Desired: &up,
				Actual:  unknown,
			},
			changed: true,
		},
		{
			name: "different actual with same desired",
			nic: &NicState{
				Desired: &down,
				Actual:  up,
			},
			arg: down,
			want: NicState{
				Desired: &down,
				Actual:  up,
			},
			changed: false,
		},
		{
			name: "different actual with other desired",
			nic: &NicState{
				Desired: &up,
				Actual:  up,
			},
			arg: down,
			want: NicState{
				Desired: &down,
				Actual:  up,
			},
			changed: true,
		},
		{
			name: "different actual with empty desired",
			nic: &NicState{
				Desired: nil,
				Actual:  up,
			},
			arg: down,
			want: NicState{
				Desired: &down,
				Actual:  up,
			},
			changed: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			got, got1 := tt.nic.WantState(tt.arg)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NicState.WantState() got = %+v, want %+v", got, tt.want)
			}
			if got1 != tt.changed {
				t.Errorf("NicState.WantState() got1 = %v, want %v", got1, tt.changed)
			}
		})
	}
}

func TestNicState_SetState(t *testing.T) {
	up := SwitchPortStatusUp
	down := SwitchPortStatusDown
	unknown := SwitchPortStatusUnknown

	tests := []struct {
		name    string
		nic     *NicState
		arg     SwitchPortStatus
		want    NicState
		changed bool
	}{
		{
			name: "different actual with empty desired",
			nic: &NicState{
				Desired: nil,
				Actual:  up,
			},
			arg: down,
			want: NicState{
				Desired: nil,
				Actual:  down,
			},
			changed: true,
		},
		{
			name: "different actual with same state in desired",
			nic: &NicState{
				Desired: &down,
				Actual:  up,
			},
			arg: down,
			want: NicState{
				Desired: nil,
				Actual:  down,
			},
			changed: true,
		},
		{
			name: "different actual with other state in desired",
			nic: &NicState{
				Desired: &unknown,
				Actual:  up,
			},
			arg: down,
			want: NicState{
				Desired: &unknown,
				Actual:  down,
			},
			changed: true,
		},
		{
			name: "nil nic",
			nic:  nil,
			arg:  down,
			want: NicState{
				Desired: nil,
				Actual:  down,
			},
			changed: true,
		},
		{
			name: "same state with same desired",
			nic: &NicState{
				Desired: &down,
				Actual:  down,
			},
			arg: down,
			want: NicState{
				Desired: nil,
				Actual:  down,
			},
			changed: true,
		},
		{
			name: "same state with other desired",
			nic: &NicState{
				Desired: &up,
				Actual:  down,
			},
			arg: down,
			want: NicState{
				Desired: &up,
				Actual:  down,
			},
			changed: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := tt.nic.SetState(tt.arg)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NicState.SetState() got = %+v, want %+v", got, tt.want)
			}
			if got1 != tt.changed {
				t.Errorf("NicState.SetState() got1 = %v, want %v", got1, tt.changed)
			}
		})
	}
}

func Test_getAddressFamily(t *testing.T) {
	tests := []struct {
		name     string
		prefixes Prefixes
		want     *AddressFamily
		wantErr  bool
	}{
		{
			name:     "ipv4",
			prefixes: Prefixes{{IP: "10.0.0.0", Length: "8"}},
			want:     pointer.Pointer(IPv4AddressFamily),
		},
		{
			name:     "ipv6",
			prefixes: Prefixes{{IP: "2001::", Length: "64"}},
			want:     pointer.Pointer(IPv6AddressFamily),
		},
		{
			name:     "empty prefixes",
			prefixes: Prefixes{},
			want:     nil,
			wantErr:  false,
		},
		{
			name:     "malformed ipv4",
			prefixes: Prefixes{{IP: "10.0.0.0.0", Length: "6"}},
			want:     nil,
			wantErr:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetAddressFamily(tt.prefixes)
			if (err != nil) != tt.wantErr {
				t.Errorf("getAddressFamily() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getAddressFamily() = %v, want %v", got, tt.want)
			}
		})
	}
}
