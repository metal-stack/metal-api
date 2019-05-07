package metal

import "testing"

func TestIPToASN(t *testing.T) {
	ipaddress := IP{
		IPAddress: "10.0.1.2",
	}

	asn, err := ipaddress.ASN()
	if err != nil {
		t.Errorf("no error expected got:%v", err)
	}

	if asn != 4200000258 {
		t.Errorf("expected 4200000258 got: %d", asn)
	}
}
