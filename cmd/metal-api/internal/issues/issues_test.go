package issues

import (
	"fmt"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
	"github.com/stretchr/testify/require"
)

func TestFindIssues(t *testing.T) {
	machineTemplate := func(id string) metal.Machine {
		return metal.Machine{
			Base: metal.Base{
				ID: id,
			},
			PartitionID: "a",
			IPMI: metal.IPMI{
				Address:     "1.2.3.4",
				MacAddress:  "aa:bb:00",
				LastUpdated: time.Now().Add(-1 * time.Minute),
			},
		}
	}
	eventContainerTemplate := func(id string) metal.ProvisioningEventContainer {
		return metal.ProvisioningEventContainer{
			Base: metal.Base{
				ID: id,
			},
			Liveliness: metal.MachineLivelinessAlive,
		}
	}

	tests := []struct {
		name string
		only []Type

		machines        func() metal.Machines
		eventContainers func() metal.ProvisioningEventContainers

		want func(machines metal.Machines) MachineIssues
	}{
		{
			name: "good machine has no issues",
			machines: func() metal.Machines {
				return metal.Machines{
					machineTemplate("good"),
				}
			},
			eventContainers: func() metal.ProvisioningEventContainers {
				return metal.ProvisioningEventContainers{
					eventContainerTemplate("good"),
				}
			},
			want: nil,
		},
		{
			name: "no partition",
			only: []Type{TypeNoPartition},
			machines: func() metal.Machines {
				noPartitionMachine := machineTemplate("no-partition")
				noPartitionMachine.PartitionID = ""

				return metal.Machines{
					noPartitionMachine,
					machineTemplate("good"),
				}
			},
			eventContainers: func() metal.ProvisioningEventContainers {
				return metal.ProvisioningEventContainers{
					eventContainerTemplate("no-partition"),
					eventContainerTemplate("good"),
				}
			},
			want: func(machines metal.Machines) MachineIssues {
				return MachineIssues{
					{
						Machine: &machines[0],
						Issues: Issues{
							toIssue(&issueNoPartition{}),
						},
					},
				}
			},
		},
		{
			name: "liveliness dead",
			only: []Type{TypeLivelinessDead},
			machines: func() metal.Machines {
				return metal.Machines{
					machineTemplate("dead"),
					machineTemplate("good"),
				}
			},
			eventContainers: func() metal.ProvisioningEventContainers {
				dead := eventContainerTemplate("dead")
				dead.Liveliness = metal.MachineLivelinessDead

				return metal.ProvisioningEventContainers{
					dead,
					eventContainerTemplate("good"),
				}
			},
			want: func(machines metal.Machines) MachineIssues {
				return MachineIssues{
					{
						Machine: &machines[0],
						Issues: Issues{
							toIssue(&issueLivelinessDead{}),
						},
					},
				}
			},
		},
		{
			name: "liveliness unknown",
			only: []Type{TypeLivelinessUnknown},
			machines: func() metal.Machines {
				return metal.Machines{
					machineTemplate("unknown"),
					machineTemplate("good"),
				}
			},
			eventContainers: func() metal.ProvisioningEventContainers {
				unknown := eventContainerTemplate("unknown")
				unknown.Liveliness = metal.MachineLivelinessUnknown

				return metal.ProvisioningEventContainers{
					unknown,
					eventContainerTemplate("good"),
				}
			},
			want: func(machines metal.Machines) MachineIssues {
				return MachineIssues{
					{
						Machine: &machines[0],
						Issues: Issues{
							toIssue(&issueLivelinessUnknown{}),
						},
					},
				}
			},
		},
		{
			name: "liveliness not available",
			only: []Type{TypeLivelinessNotAvailable},
			machines: func() metal.Machines {
				return metal.Machines{
					machineTemplate("n/a"),
					machineTemplate("good"),
				}
			},
			eventContainers: func() metal.ProvisioningEventContainers {
				na := eventContainerTemplate("n/a")
				na.Liveliness = metal.MachineLiveliness("")

				return metal.ProvisioningEventContainers{
					na,
					eventContainerTemplate("good"),
				}
			},
			want: func(machines metal.Machines) MachineIssues {
				return MachineIssues{
					{
						Machine: &machines[0],
						Issues: Issues{
							toIssue(&issueLivelinessNotAvailable{}),
						},
					},
				}
			},
		},
		{
			name: "failed machine reclaim",
			only: []Type{TypeFailedMachineReclaim},
			machines: func() metal.Machines {
				failedOld := machineTemplate("failed-old")

				return metal.Machines{
					machineTemplate("good"),
					machineTemplate("failed"),
					failedOld,
				}
			},
			eventContainers: func() metal.ProvisioningEventContainers {
				failed := eventContainerTemplate("failed")
				failed.FailedMachineReclaim = true

				failedOld := eventContainerTemplate("failed-old")
				failedOld.Events = metal.ProvisioningEvents{
					{
						Event: metal.ProvisioningEventPhonedHome,
					},
				}

				return metal.ProvisioningEventContainers{
					failed,
					eventContainerTemplate("good"),
					failedOld,
				}
			},
			want: func(machines metal.Machines) MachineIssues {
				return MachineIssues{
					{
						Machine: &machines[1],
						Issues: Issues{
							toIssue(&issueFailedMachineReclaim{}),
						},
					},
					{
						Machine: &machines[2],
						Issues: Issues{
							toIssue(&issueFailedMachineReclaim{}),
						},
					},
				}
			},
		},
		{
			name: "crashloop",
			only: []Type{TypeCrashLoop},
			machines: func() metal.Machines {
				return metal.Machines{
					machineTemplate("good"),
					machineTemplate("crash"),
				}
			},
			eventContainers: func() metal.ProvisioningEventContainers {
				crash := eventContainerTemplate("crash")
				crash.CrashLoop = true

				return metal.ProvisioningEventContainers{
					crash,
					eventContainerTemplate("good"),
				}
			},
			want: func(machines metal.Machines) MachineIssues {
				return MachineIssues{
					{
						Machine: &machines[1],
						Issues: Issues{
							toIssue(&issueCrashLoop{}),
						},
					},
				}
			},
		},
		// FIXME:
		// {
		// 	name: "last event error",
		// 	only: []IssueType{IssueTypeLastEventError},
		// 	machines: func() metal.Machines {
		// 		lastEventErrorMachine := machineTemplate("last")

		// 		return metal.Machines{
		// 			machineTemplate("good"),
		// 			lastEventErrorMachine,
		// 		}
		// 	},
		// 	eventContainers: func() metal.ProvisioningEventContainers {
		// 		last := eventContainerTemplate("last")
		// 		last.LastErrorEvent = &metal.ProvisioningEvent{
		// 			Time: time.Now().Add(-5 * time.Minute),
		// 		}
		// 		return metal.ProvisioningEventContainers{
		// 			last,
		// 			eventContainerTemplate("good"),
		// 		}
		// 	},
		// 	want: func(machines metal.Machines) MachineIssues {
		// 		return MachineIssues{
		// 			{
		// 				Machine: &machines[1],
		// 				Issues: Issues{
		// 					toIssue(&IssueLastEventError{details: "occurred 5m0s ago"}),
		// 				},
		// 			},
		// 		}
		// 	},
		// },
		{
			name: "bmc without mac",
			only: []Type{TypeBMCWithoutMAC},
			machines: func() metal.Machines {
				noMac := machineTemplate("no-mac")
				noMac.IPMI.MacAddress = ""

				return metal.Machines{
					machineTemplate("good"),
					noMac,
				}
			},
			eventContainers: func() metal.ProvisioningEventContainers {
				crash := eventContainerTemplate("crash")
				crash.CrashLoop = true

				return metal.ProvisioningEventContainers{
					eventContainerTemplate("no-mac"),
					eventContainerTemplate("good"),
				}
			},
			want: func(machines metal.Machines) MachineIssues {
				return MachineIssues{
					{
						Machine: &machines[1],
						Issues: Issues{
							toIssue(&issueBMCWithoutMAC{}),
						},
					},
				}
			},
		},
		{
			name: "bmc without ip",
			only: []Type{TypeBMCWithoutIP},
			machines: func() metal.Machines {
				noIP := machineTemplate("no-ip")
				noIP.IPMI.Address = ""

				return metal.Machines{
					machineTemplate("good"),
					noIP,
				}
			},
			eventContainers: func() metal.ProvisioningEventContainers {
				crash := eventContainerTemplate("crash")
				crash.CrashLoop = true

				return metal.ProvisioningEventContainers{
					eventContainerTemplate("no-ip"),
					eventContainerTemplate("good"),
				}
			},
			want: func(machines metal.Machines) MachineIssues {
				return MachineIssues{
					{
						Machine: &machines[1],
						Issues: Issues{
							toIssue(&issueBMCWithoutIP{}),
						},
					},
				}
			},
		},
		// FIXME:
		// {
		// 	name: "bmc info outdated",
		// 	only: []IssueType{IssueTypeBMCInfoOutdated},
		// 	machines: func() metal.Machines {
		// 		outdated := machineTemplate("outdated")
		// 		outdated.IPMI.LastUpdated = time.Now().Add(-3 * 60 * time.Minute)

		// 		return metal.Machines{
		// 			machineTemplate("good"),
		// 			outdated,
		// 		}
		// 	},
		// 	eventContainers: func() metal.ProvisioningEventContainers {
		// 		return metal.ProvisioningEventContainers{
		// 			eventContainerTemplate("outdated"),
		// 			eventContainerTemplate("good"),
		// 		}
		// 	},
		// 	want: func(machines metal.Machines) MachineIssues {
		// 		return MachineIssues{
		// 			{
		// 				Machine: &machines[1],
		// 				Issues: Issues{
		// 					toIssue(&IssueBMCInfoOutdated{
		// 						details: "last updated 3h0m0s ago",
		// 					}),
		// 				},
		// 			},
		// 		}
		// 	},
		// },
		{
			name: "asn shared",
			only: []Type{TypeASNUniqueness},
			machines: func() metal.Machines {
				shared1 := machineTemplate("shared1")
				shared1.Allocation = &metal.MachineAllocation{
					Role: metal.RoleFirewall,
					MachineNetworks: []*metal.MachineNetwork{
						{
							ASN: 0,
						},
						{
							ASN: 100,
						},
						{
							ASN: 200,
						},
					},
				}

				shared2 := machineTemplate("shared2")
				shared2.Allocation = &metal.MachineAllocation{
					Role: metal.RoleFirewall,
					MachineNetworks: []*metal.MachineNetwork{
						{
							ASN: 1,
						},
						{
							ASN: 100,
						},
						{
							ASN: 200,
						},
					},
				}

				return metal.Machines{
					shared1,
					shared2,
					machineTemplate("good"),
				}
			},
			eventContainers: func() metal.ProvisioningEventContainers {
				return metal.ProvisioningEventContainers{
					eventContainerTemplate("shared1"),
					eventContainerTemplate("shared2"),
					eventContainerTemplate("good"),
				}
			},
			want: func(machines metal.Machines) MachineIssues {
				return MachineIssues{
					{
						Machine: &machines[0],
						Issues: Issues{
							toIssue(&issueASNUniqueness{
								details: fmt.Sprintf("- ASN (100) not unique, shared with [%[1]s]\n- ASN (200) not unique, shared with [%[1]s]", machines[1].ID),
							}),
						},
					},
					{
						Machine: &machines[1],
						Issues: Issues{
							toIssue(&issueASNUniqueness{
								details: fmt.Sprintf("- ASN (100) not unique, shared with [%[1]s]\n- ASN (200) not unique, shared with [%[1]s]", machines[0].ID),
							}),
						},
					},
				}
			},
		},
		{
			name: "non distinct bmc ip",
			only: []Type{TypeNonDistinctBMCIP},
			machines: func() metal.Machines {
				bmc1 := machineTemplate("bmc1")
				bmc1.IPMI.Address = "127.0.0.1"

				bmc2 := machineTemplate("bmc2")
				bmc2.IPMI.Address = "127.0.0.1"

				return metal.Machines{
					bmc1,
					bmc2,
					machineTemplate("good"),
				}
			},
			eventContainers: func() metal.ProvisioningEventContainers {
				return metal.ProvisioningEventContainers{
					eventContainerTemplate("bmc1"),
					eventContainerTemplate("bmc2"),
					eventContainerTemplate("good"),
				}
			},
			want: func(machines metal.Machines) MachineIssues {
				return MachineIssues{
					{
						Machine: &machines[0],
						Issues: Issues{
							toIssue(&issueNonDistinctBMCIP{
								details: fmt.Sprintf("BMC IP (127.0.0.1) not unique, shared with [%[1]s]", machines[1].ID),
							}),
						},
					},
					{
						Machine: &machines[1],
						Issues: Issues{
							toIssue(&issueNonDistinctBMCIP{
								details: fmt.Sprintf("BMC IP (127.0.0.1) not unique, shared with [%[1]s]", machines[0].ID),
							}),
						},
					},
				}
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			ms := tt.machines()

			got, err := FindIssues(&Config{
				Machines:           ms,
				EventContainers:    tt.eventContainers(),
				Only:               tt.only,
				LastErrorThreshold: DefaultLastErrorThreshold(),
			})
			require.NoError(t, err)

			var want MachineIssues
			if tt.want != nil {
				want = tt.want(ms)
			}

			if diff := cmp.Diff(want, got.ToList(), cmp.AllowUnexported(issueLastEventError{}, issueASNUniqueness{}, issueNonDistinctBMCIP{})); diff != "" {
				t.Errorf("diff (+got -want):\n %s", diff)
			}
		})
	}
}

func TestAllIssues(t *testing.T) {
	issuesTypes := map[Type]bool{}
	for _, i := range AllIssues() {
		issuesTypes[i.Type] = true
	}

	for _, ty := range AllIssueTypes() {
		if _, ok := issuesTypes[ty]; !ok {
			t.Errorf("issue of type %s not contained in all issues", ty)
		}
	}
}
