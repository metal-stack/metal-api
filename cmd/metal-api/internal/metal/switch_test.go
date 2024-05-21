package metal

import (
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
)

var (
	testNics = Nics{
		Nic{
			MacAddress: "11:11:11:11:11:11",
		},
		Nic{
			MacAddress: "21:11:11:11:11:11",
		},
	}

	// Switches
	switch1 = Switch{
		Base: Base{
			ID: "switch1",
		},
		PartitionID: "1",
		RackID:      "1",
		Nics:        testNics,
		MachineConnections: ConnectionMap{
			"1": Connections{
				Connection{
					Nic: Nic{
						MacAddress: MacAddress("11:11:11:11:11:11"),
					},
					MachineID: "1",
				},
				Connection{
					Nic: Nic{
						MacAddress: MacAddress("11:11:11:11:11:22"),
					},
					MachineID: "1",
				},
			},
		},
	}
)

func TestSwitch_ConnectMachine(t *testing.T) {
	type args struct {
		*Machine
	}
	tests := []struct {
		name string
		s    *Switch
		args args
	}{
		// Test-Data List / Test Cases:
		{
			name: "Test 1",
			s:    &switch1,
			args: args{
				Machine: &Machine{
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
						Memory:   100,
						CPUCores: 1,
						Nics:     testNics,
						Disks: []BlockDevice{
							{
								Name: "blockdeviceName",
								Size: 1000000000000,
							},
						},
					},
				},
			},
		},
	}
	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			tt.s.ConnectMachine(tt.args.Machine)
		})
	}
}

// Gerrit and me implemented that monster in a one shot which worked.

func TestSwitch_ConnectMachine2(t *testing.T) {
	type fields struct {
		ID                 string
		Nics               []Nic
		MachineConnections ConnectionMap
		PartitionID        string
		RackID             string
	}

	switchName1 := "switch-1"
	switchName2 := "switch-2"
	tests := []struct {
		name    string
		fields  fields
		machine *Machine
	}{
		{
			name: "simple connection",
			fields: fields{
				ID: switchName1,
				Nics: []Nic{
					{
						Name:       "eth0",
						MacAddress: "00:11:11",
					},
					{
						Name:       "swp1",
						MacAddress: "11:11:11",
					},
					{
						Name:       "swp2",
						MacAddress: "22:11:11",
					},
				},
				PartitionID: "nbg1",
				RackID:      "rack1",
				MachineConnections: ConnectionMap{
					"machine-1": []Connection{
						{
							Nic: Nic{
								Name:       "swp1",
								MacAddress: "11:11:11",
							},
							MachineID: "machine-1",
						},
						{
							Nic: Nic{
								Name:       "swp2",
								MacAddress: "22:11:11",
							},
							MachineID: "machine-1",
						},
					},
				},
			},
			machine: &Machine{
				Base: Base{
					ID: "machine-1",
				},
				Hardware: MachineHardware{
					Nics: []Nic{
						{
							Name: "eth0",
							Neighbors: []Nic{
								{
									MacAddress: "11:11:11",
									Hostname:   switchName1,
								},
								{
									MacAddress: "11:11:12",
									Hostname:   switchName1,
								},
							},
						},
						{
							Name: "eth1",
							Neighbors: []Nic{
								{
									MacAddress: "22:11:11",
									Hostname:   switchName1,
								},
								{
									MacAddress: "11:11:13",
									Hostname:   switchName1,
								},
							},
						},
					},
				},
			},
		},
		{
			name: "multiple switch connection",
			fields: fields{
				ID: switchName1,
				Nics: []Nic{
					{
						Name:       "eth0",
						MacAddress: "00:11:11",
					},
					{
						Name:       "swp1",
						MacAddress: "11:11:11",
					},
					{
						Name:       "swp2",
						MacAddress: "22:11:11",
					},
				},
				PartitionID: "nbg1",
				RackID:      "rack1",
				MachineConnections: ConnectionMap{
					"machine-1": []Connection{
						{
							Nic: Nic{
								Name:       "swp1",
								MacAddress: "11:11:11",
							},
							MachineID: "machine-1",
						},
					},
				},
			},
			machine: &Machine{
				Base: Base{
					ID: "machine-1",
				},
				Hardware: MachineHardware{
					Nics: []Nic{
						{
							Name: "eth0",
							Neighbors: []Nic{
								{
									MacAddress: "11:11:11",
									Hostname:   switchName1,
								},
								{
									MacAddress: "11:11:12",
									Hostname:   switchName1,
								},
							},
						},
						{
							Name: "eth1",
							Neighbors: []Nic{
								{
									MacAddress: "22:11:11",
									Hostname:   switchName2,
								},
								{
									MacAddress: "11:11:13",
									Hostname:   switchName2,
								},
							},
						},
					},
				},
			},
		},
	}
	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			s := Switch{
				Base: Base{
					ID:   tt.fields.ID,
					Name: tt.fields.ID,
				},
				RackID:             tt.fields.RackID,
				Nics:               tt.fields.Nics,
				PartitionID:        tt.fields.PartitionID,
				MachineConnections: ConnectionMap{},
			}
			s.ConnectMachine(tt.machine)
			if !reflect.DeepEqual(s.MachineConnections, tt.fields.MachineConnections) {
				t.Errorf("expected:%v, got:%v", s.MachineConnections, tt.fields.MachineConnections)
			}
		})
	}
}

func TestMapPortNames(t *testing.T) {
	tests := []struct {
		name     string
		ports    []string
		sourceOS SwitchOSVendor
		targetOS SwitchOSVendor
		want     switchPortMapping
		wantErr  error
	}{
		{
			name: "self migration",
			ports: []string{
				"swp0",
				"swp1s2",
			},
			sourceOS: SwitchOSVendorCumulus,
			targetOS: SwitchOSVendorCumulus,
			want: switchPortMapping{
				"swp0":   "swp0",
				"swp1s2": "swp1s2",
			},
			wantErr: nil,
		},
		// TODO: add more test cases
	}
	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			converted, err := MapPortNames(tt.ports, tt.sourceOS, tt.targetOS)
			if err == nil && tt.wantErr != nil || err != nil && tt.wantErr == nil || cmp.Diff(err, tt.wantErr) != "" {
				t.Errorf("expected error: %v, got error: %v", tt.wantErr, err)
			}

			if diff := cmp.Diff(converted, tt.want); diff != "" {
				t.Errorf("diff: %v", diff)
			}
		})
	}
}

func Test_mapSonicPortNamesToLines(t *testing.T) {
	tests := []struct {
		name    string
		ports   []string
		want    switchPortToLine
		wantErr error
	}{
		{
			name:    "no ports",
			ports:   []string{},
			want:    map[string]int{},
			wantErr: nil,
		},
		// TODO: add more test cases
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := mapSonicPortNamesToLines(tt.ports)
			if err != nil && tt.wantErr == nil || err == nil && tt.wantErr != nil || cmp.Diff(err, tt.wantErr) != "" {
				t.Errorf("mapSonicPortNamesToLines() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("mapSonicPortNamesToLines() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_mapCumulusPortNamesToLines(t *testing.T) {
	tests := []struct {
		name    string
		ports   []string
		want    switchPortToLine
		wantErr error
	}{
		{
			name:    "no ports",
			ports:   []string{},
			want:    map[string]int{},
			wantErr: nil,
		},
		// TODO: add more test cases
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := mapCumulusPortNamesToLines(tt.ports)
			if err != nil && tt.wantErr == nil || err == nil && tt.wantErr != nil || cmp.Diff(err, tt.wantErr) != "" {
				t.Errorf("mapSonicPortNamesToLines() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("mapSonicPortNamesToLines() = %v, want %v", got, tt.want)
			}
		})
	}
}
