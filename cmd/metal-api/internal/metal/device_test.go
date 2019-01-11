package metal

import (
	"testing"
)

type testDataStruct struct {
	name string
	d    *Device
	args struct {
		mac string
	}
	want bool
}

func getAllTestStructsForTestDevice_HasMAC() []testDataStruct {
	/*
		Returns an struct Array of all Test Data
		// Create all Test Data
		tests := getAllTestStructs()
	*/
	type args struct {
		mac string
	}
	returnData := make([]testDataStruct, len(TestDeviceArray)*len(TestMacArray))
	index := 0
	for i := 0; i < len(TestDeviceArray); i++ {
		for j := 0; j < len(TestMacArray); j++ {
			want := false
			if TestDeviceArray[i].ID == D5.ID && TestMacArray[j] == "11:11:11:11:11:11" {
				want = true
			} else {
				want = false
			}
			returnData[index] = struct {
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
	return returnData
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
