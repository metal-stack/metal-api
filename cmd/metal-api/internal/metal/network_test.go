package metal

import (
	"reflect"
	"testing"
)

func TestNics_ByMac(t *testing.T) {

	// Create Nics
	var countOfNics = 3
	nicArray := make([]Nic, countOfNics)
	for i := 0; i < countOfNics; i++ {
		nicArray[i] = Nic{
			MacAddress: MacAddress("11:11:1" + string(i)),
			Name:       "swp" + string(i),
			Neighbors:  nil,
		}
	}

	// all have all as Neighbors
	for i := 0; i < countOfNics; i++ {
		nicArray[i].Neighbors = append(nicArray[0:i], nicArray[i+1:countOfNics]...)
	}

	map1 := NicMap{}
	for i, n := range nicArray {
		map1[n.MacAddress] = &nicArray[i]
	}

	tests := []struct {
		name string
		nics Nics
		want NicMap
	}{
		// Test Data Array (only 1 data):
		{
			name: "TestNics_ByMac Test 1",
			nics: nicArray,
			want: map1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.nics.ByMac(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Nics.ByMac() = %v, want %v", got, tt.want)
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
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Prefix{
				IP:     tt.fields.IP,
				Length: tt.fields.Length,
			}
			if got := p.Equals(tt.args.other); got != tt.want {
				t.Errorf("Prefix.Equals() = %v, want %v", got, tt.want)
			}
		})
	}
}
