package jwt

import (
	"testing"

	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/metal"
)

func TestSerialization(t *testing.T) {
	e := NewPhoneHomeClaims(&metal.Machine{Base: metal.Base{ID: "1"}})
	jwt, err := e.JWT()
	if err != nil {
		t.Fatalf("coult not sign %v", err)
	}

	a, err := FromJWT(jwt)
	if err != nil {
		t.Fatalf("coult not parse %v", err)
	}

	if e.Machine.ID != a.Machine.ID {
		t.Fatalf("got machine id %v, expected %v", a.Machine.ID, e.Machine.ID)
	}
}
