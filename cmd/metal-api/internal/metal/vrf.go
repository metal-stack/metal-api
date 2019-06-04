package metal

import (
	"crypto/sha256"
	"fmt"
	"strconv"
)

// A Vrf ...
type Vrf struct {
	Base
	Tenant    string `rethinkdb:"tenant"`
	ProjectID string `rethinkdb:"projectid"`
}

// GenerateVrfID generates a unique ID for a given unique input string.
func GenerateVrfID(i string) (string, error) {
	sha := sha256.Sum256([]byte(i))
	// cut four bytes of hash
	hexTrunc := fmt.Sprintf("%x", sha)[:4]
	hash, err := strconv.ParseUint(hexTrunc, 16, 16)
	if err != nil {
		return "", err
	}
	return strconv.FormatUint(hash, 10), nil
}

// ToUint converts the VrfID to an unsigned integer.
func (v *Vrf) ToUint() (uint, error) {
	id, err := strconv.ParseUint(v.ID, 10, 64)
	if err != nil {
		return 0, err
	}

	return uint(id), nil
}
