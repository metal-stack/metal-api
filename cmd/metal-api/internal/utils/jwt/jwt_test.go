package jwt

import (
	"testing"

	"git.f-i-ts.de/cloud-native/maas/metal-api/metal"
)

func TestSerialization(t *testing.T) {
	e := NewPhoneHomeClaims(&metal.Device{ID: "1"})
	jwt, err := e.JWT()
	if err != nil {
		t.Fatalf("coult not sign %v", err)
	}

	a, err := FromJWT(jwt)
	if err != nil {
		t.Fatalf("coult not parse %v", err)
	}

	if e.Device.ID != a.Device.ID {
		t.Fatalf("got device id %v, expected %v", a.Device.ID, e.Device.ID)
	}
}
