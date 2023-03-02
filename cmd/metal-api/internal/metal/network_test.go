package metal

import (
	"fmt"
	"reflect"
	"testing"
)

func TestNics_ByIdentifier(t *testing.T) {
	// Create Nics
	countOfNics := 3
	nicArray := make([]Nic, countOfNics)
	for i := 0; i < countOfNics; i++ {
		nicArray[i] = Nic{
			MacAddress: MacAddress("11:11:1" + fmt.Sprintf("%d", i)),
			Name:       "swp" + fmt.Sprintf("%d", i),
			Neighbors:  nil,
		}
	}

	// all have all as Neighbors
	for i := 0; i < countOfNics; i++ {
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
			if got := p.Equals(tt.args.other); got != tt.want {
				t.Errorf("Prefix.Equals() = %v, want %v", got, tt.want)
			}
		})
	}
}
