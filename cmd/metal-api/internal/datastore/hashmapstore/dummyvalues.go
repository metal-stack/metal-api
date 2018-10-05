package hashmapstore

import (
	"time"

	"git.f-i-ts.de/cloud-native/maas/metal-api/pkg/metal"
)

var (
	DummyDevices = []*metal.Device{
		// &metal.Device{
		// 	ID:           "1234-1234-1234",
		// 	Name:         "",
		// 	Description:  "",
		// 	Created:      time.Now(),
		// 	Changed:      time.Now(),
		// 	Project:      "",
		// 	MACAddresses: []string{"12:12:12:12:12:12", "34:34:34:34:34:34"},
		// },
		// &metal.Device{
		// 	ID:           "5678-5678-5678",
		// 	Name:         "metal-test-host-1",
		// 	Description:  "A test host.",
		// 	Created:      time.Now(),
		// 	Changed:      time.Now(),
		// 	Project:      "metal",
		// 	Facility:     *DummyFacilities[0],
		// 	Image:        DummyImages[0],
		// 	Size:         *DummySizes[0],
		// 	MACAddresses: []string{"56:56:56:56:56:56", "78:78:78:78:78:78"},
		// },
	}

	DummyFacilities = []*metal.Facility{
		&metal.Facility{
			ID:          "NBG1",
			Name:        "Nuernberg 1",
			Description: "Location number one in NBG",
			Created:     time.Now(),
			Changed:     time.Now(),
		},
		&metal.Facility{
			ID:          "NBG2",
			Name:        "Nuernberg 2",
			Description: "Location number two in NBG",
			Created:     time.Now(),
			Changed:     time.Now(),
		},
		&metal.Facility{
			ID:          "FRA",
			Name:        "Frankfurt",
			Description: "A location in frankfurt",
			Created:     time.Now(),
			Changed:     time.Now(),
		},
	}

	DummyImages = []*metal.Image{
		&metal.Image{
			ID:          "1",
			Name:        "Discovery",
			Description: "Image for initial discovery",
			Url:         "https://registry.maas/discovery/dicoverer:latest",
			Created:     time.Now(),
			Changed:     time.Now(),
		},
		&metal.Image{
			ID:          "2",
			Name:        "Alpine 3.8",
			Description: "Alpine 3.8",
			Url:         "https://registry.maas/alpine/alpine:3.8",
			Created:     time.Now(),
			Changed:     time.Now(),
		},
	}
	DummySizes = []*metal.Size{
		&metal.Size{
			ID:          "t1.small.x86",
			Name:        "t1.small.x86",
			Description: "The Tiny But Mighty!",
			Created:     time.Now(),
			Changed:     time.Now(),
		},
		&metal.Size{
			ID:          "m2.xlarge.x86",
			Name:        "m2.xlarge.x86",
			Description: "The Latest and Greatest",
			Created:     time.Now(),
			Changed:     time.Now(),
		},
		&metal.Size{
			ID:          "c1.large.arm",
			Name:        "c1.large.arm",
			Description: "The Armv8 Beast!",
			Created:     time.Now(),
			Changed:     time.Now(),
		},
	}
)
