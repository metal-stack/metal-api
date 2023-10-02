package issues

import (
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
		only []IssueType

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
			only: []IssueType{IssueTypeNoPartition},
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
							toIssue(&IssueNoPartition{}),
						},
					},
				}
			},
		},
		{
			name: "liveliness dead",
			only: []IssueType{IssueTypeLivelinessDead},
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
							toIssue(&IssueLivelinessDead{}),
						},
					},
				}
			},
		},
		{
			name: "liveliness unknown",
			only: []IssueType{IssueTypeLivelinessUnknown},
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
							toIssue(&IssueLivelinessUnknown{}),
						},
					},
				}
			},
		},
		{
			name: "liveliness not available",
			only: []IssueType{IssueTypeLivelinessNotAvailable},
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
							toIssue(&IssueLivelinessNotAvailable{}),
						},
					},
				}
			},
		},
		{
			name: "failed machine reclaim",
			only: []IssueType{IssueTypeFailedMachineReclaim},
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
							toIssue(&IssueFailedMachineReclaim{}),
						},
					},
					{
						Machine: &machines[2],
						Issues: Issues{
							toIssue(&IssueFailedMachineReclaim{}),
						},
					},
				}
			},
		},
		{
			name: "crashloop",
			only: []IssueType{IssueTypeCrashLoop},
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
							toIssue(&IssueCrashLoop{}),
						},
					},
				}
			},
		},
		// {
		// 	name: "last event error",
		// 	only: []IssueType{IssueTypeLastEventError},
		// 	machines: func() metal.Machines {
		// 		lastEventErrorMachine := machineTemplate("last")
		// 		lastEventErrorMachine.Events = &models.V1MachineRecentProvisioningEvents{
		// 			LastErrorEvent: &models.V1MachineProvisioningEvent{
		// 				Time: strfmt.DateTime(testTime.Add(-5 * time.Minute)),
		// 			},
		// 		}

		// 		return metal.Machines{
		// 			machineTemplate("0"),
		// 			lastEventErrorMachine,
		// 		}
		// 	},
		// 	want: func(machines metal.Machines) MachineIssues {
		// 		return MachineIssues{
		// 			{
		// 				Machine: machines[1],
		// 				Issues: Issues{
		// 					toIssue(&IssueLastEventError{details: "occurred 5m0s ago"}),
		// 				},
		// 			},
		// 		}
		// 	},
		// },
		// {
		// 	name: "bmc without mac",
		// 	only: []IssueType{IssueTypeBMCWithoutMAC},
		// 	machines: func() metal.Machines {
		// 		bmcWithoutMacMachine := machineTemplate("no-mac")
		// 		bmcWithoutMacMachine.Ipmi.Mac = nil

		// 		return metal.Machines{
		// 			machineTemplate("0"),
		// 			bmcWithoutMacMachine,
		// 		}
		// 	},
		// 	want: func(machines metal.Machines) MachineIssues {
		// 		return MachineIssues{
		// 			{
		// 				Machine: machines[1],
		// 				Issues: Issues{
		// 					toIssue(&IssueBMCWithoutMAC{}),
		// 				},
		// 			},
		// 		}
		// 	},
		// },
		// {
		// 	name: "bmc without ip",
		// 	only: []IssueType{IssueTypeBMCWithoutIP},
		// 	machines: func() metal.Machines {
		// 		bmcWithoutMacMachine := machineTemplate("no-ip")
		// 		bmcWithoutMacMachine.Ipmi.Address = nil

		// 		return metal.Machines{
		// 			machineTemplate("0"),
		// 			bmcWithoutMacMachine,
		// 		}
		// 	},
		// 	want: func(machines metal.Machines) MachineIssues {
		// 		return MachineIssues{
		// 			{
		// 				Machine: machines[1],
		// 				Issues: Issues{
		// 					toIssue(&IssueBMCWithoutIP{}),
		// 				},
		// 			},
		// 		}
		// 	},
		// },
		// {
		// 	name: "bmc info outdated",
		// 	only: []IssueType{IssueTypeBMCInfoOutdated},
		// 	machines: func() metal.Machines {
		// 		bmcOutdatedMachine := machineTemplate("outdated")
		// 		bmcOutdatedMachine.Ipmi.LastUpdated = pointer.Pointer(strfmt.DateTime(testTime.Add(-3 * 60 * time.Minute)))

		// 		return metal.Machines{
		// 			machineTemplate("0"),
		// 			bmcOutdatedMachine,
		// 		}
		// 	},
		// 	want: func(machines metal.Machines) MachineIssues {
		// 		return MachineIssues{
		// 			{
		// 				Machine: machines[1],
		// 				Issues: Issues{
		// 					toIssue(&IssueBMCInfoOutdated{
		// 						details: "last updated 3h0m0s ago",
		// 					}),
		// 				},
		// 			},
		// 		}
		// 	},
		// },
		// {
		// 	name: "asn shared",
		// 	only: []IssueType{IssueTypeASNUniqueness},
		// 	machines: func() metal.Machines {
		// 		asnSharedMachine1 := machineTemplate("shared1")
		// 		asnSharedMachine1.Allocation = &models.V1MachineAllocation{
		// 			Role: pointer.Pointer(models.V1MachineAllocationRoleFirewall),
		// 			Networks: []*models.V1MachineNetwork{
		// 				{
		// 					Asn: pointer.Pointer(int64(0)),
		// 				},
		// 				{
		// 					Asn: pointer.Pointer(int64(100)),
		// 				},
		// 				{
		// 					Asn: pointer.Pointer(int64(200)),
		// 				},
		// 			},
		// 		}

		// 		asnSharedMachine2 := machineTemplate("shared2")
		// 		asnSharedMachine2.Allocation = &models.V1MachineAllocation{
		// 			Role: pointer.Pointer(models.V1MachineAllocationRoleFirewall),
		// 			Networks: []*models.V1MachineNetwork{
		// 				{
		// 					Asn: pointer.Pointer(int64(1)),
		// 				},
		// 				{
		// 					Asn: pointer.Pointer(int64(100)),
		// 				},
		// 				{
		// 					Asn: pointer.Pointer(int64(200)),
		// 				},
		// 			},
		// 		}

		// 		return metal.Machines{
		// 			asnSharedMachine1,
		// 			asnSharedMachine2,
		// 			machineTemplate("0"),
		// 		}
		// 	},
		// 	want: func(machines metal.Machines) MachineIssues {
		// 		return MachineIssues{
		// 			{
		// 				Machine: machines[0],
		// 				Issues: Issues{
		// 					toIssue(&IssueASNUniqueness{
		// 						details: fmt.Sprintf("- ASN (100) not unique, shared with [%[1]s]\n- ASN (200) not unique, shared with [%[1]s]", *machines[1].ID),
		// 					}),
		// 				},
		// 			},
		// 			{
		// 				Machine: machines[1],
		// 				Issues: Issues{
		// 					toIssue(&IssueASNUniqueness{
		// 						details: fmt.Sprintf("- ASN (100) not unique, shared with [%[1]s]\n- ASN (200) not unique, shared with [%[1]s]", *machines[0].ID),
		// 					}),
		// 				},
		// 			},
		// 		}
		// 	},
		// },
		// {
		// 	name: "non distinct bmc ip",
		// 	only: []IssueType{IssueTypeNonDistinctBMCIP},
		// 	machines: func() metal.Machines {
		// 		nonDistinctBMCMachine1 := machineTemplate("bmc1")
		// 		nonDistinctBMCMachine1.Ipmi.Address = pointer.Pointer("127.0.0.1")

		// 		nonDistinctBMCMachine2 := machineTemplate("bmc2")
		// 		nonDistinctBMCMachine2.Ipmi.Address = pointer.Pointer("127.0.0.1")

		// 		return metal.Machines{
		// 			nonDistinctBMCMachine1,
		// 			nonDistinctBMCMachine2,
		// 			machineTemplate("0"),
		// 		}
		// 	},
		// 	want: func(machines metal.Machines) MachineIssues {
		// 		return MachineIssues{
		// 			{
		// 				Machine: machines[0],
		// 				Issues: Issues{
		// 					toIssue(&IssueNonDistinctBMCIP{
		// 						details: fmt.Sprintf("BMC IP (127.0.0.1) not unique, shared with [%[1]s]", *machines[1].ID),
		// 					}),
		// 				},
		// 			},
		// 			{
		// 				Machine: machines[1],
		// 				Issues: Issues{
		// 					toIssue(&IssueNonDistinctBMCIP{
		// 						details: fmt.Sprintf("BMC IP (127.0.0.1) not unique, shared with [%[1]s]", *machines[0].ID),
		// 					}),
		// 				},
		// 			},
		// 		}
		// 	},
		// },
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			ms := tt.machines()

			got, err := FindIssues(&IssueConfig{
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

			if diff := cmp.Diff(want, got, cmp.AllowUnexported(IssueLastEventError{}, IssueASNUniqueness{}, IssueNonDistinctBMCIP{})); diff != "" {
				t.Errorf("diff (+got -want):\n %s", diff)
			}
		})
	}
}

func TestAllIssues(t *testing.T) {
	issuesTypes := map[IssueType]bool{}
	for _, i := range AllIssues() {
		issuesTypes[i.Type] = true
	}

	for _, ty := range AllIssueTypes() {
		if _, ok := issuesTypes[ty]; !ok {
			t.Errorf("issue of type %s not contained in all issues", ty)
		}
	}
}
