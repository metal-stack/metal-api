package metal

import (
	"testing"
)

func TestDevice_HasMAC(t *testing.T) {
	type args struct {
		mac string
	}
	//tests := getAllTestStructsForTestDevice_HasMAC()

	tests := []struct {
		name string
		d    *Device
		args struct {
			mac string
		}
		want bool
	}{
		// Test Data Array (only 1 data):
		{
			name: "TestDevice_HasMAC Test 1",
			d:    &d5,
			args: args{
				mac: "11:11:11:11:11:11",
			},
			want: true,
		},
		{
			name: "TestDevice_HasMAC Test 1",
			d:    &d1,
			args: args{
				mac: "11:11:11:11:11:11",
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.d.HasMAC(tt.args.mac); got != tt.want {
				t.Errorf("Device.HasMAC() = %v, want %v", got, tt.want)
			}
		})
	}
}
