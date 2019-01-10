package metal

import (
	"testing"
)

func getAllTestStructsForTestDevice_HasMAC() []struct {
	name string
	d    *Device
	args struct {
		mac string
	}
	want bool
} {
	/*
		Returns an struct Array of all Test Data
		// Create all Test Data
		tests := getAllTestStructs()
	*/
	type args struct {
		mac string
	}
	structArray := make([]struct {
		name string
		d    *Device
		args struct {
			mac string
		}
		want bool
	}, len(TestDeviceArray)*len(TestMacArray))
	index := 0
	for i := 0; i < len(TestDeviceArray); i++ {
		for j := 0; j < len(TestMacArray); j++ {
			want := false
			if TestDeviceArray[i].ID == D5.ID && TestMacArray[j] == "11:11:11:11:11:11" {
				want = true
			} else {
				want = false
			}
			structArray[index] = struct {
				name string
				d    *Device
				args struct {
					mac string
				}
				want bool
			}{
				name: "TestDevice_HasMAC Test " + string(i),
				d:    &TestDeviceArray[i],
				args: args{
					mac: TestMacArray[j],
				},
				want: want,
			}
			index++
		}
	}
	return structArray
}

func TestDevice_HasMAC(t *testing.T) {
	type args struct {
		mac string
	}
	tests := getAllTestStructsForTestDevice_HasMAC()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.d.HasMAC(tt.args.mac); got != tt.want {
				t.Errorf("Device.HasMAC() = %v, want %v", got, tt.want)
			}
		})
	}
}
