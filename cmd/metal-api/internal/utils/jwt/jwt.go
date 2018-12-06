package jwt

import (
	"errors"
	"fmt"

	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/metal"
	"github.com/dgrijalva/jwt-go"
)

var phoneHomeHmacSecret = []byte("someSortOfSecret0812")

// PhoneHomeClaims contains the structue of the JWT payload
type PhoneHomeClaims struct {
	Device *metal.Device `json:"device"`
	jwt.StandardClaims
}

// New creates PhoneHomeClaims with a device
func NewPhoneHomeClaims(d *metal.Device) *PhoneHomeClaims {
	c := &PhoneHomeClaims{}
	c.Device = d
	c.StandardClaims = jwt.StandardClaims{}
	return c
}

// Valid validates PhoneHomeClaims
func (c *PhoneHomeClaims) Valid() error {
	return nil
}

// SerializeJWT creates, signs and serializes a JWT (based on the claims)
func (c *PhoneHomeClaims) JWT() (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, c)
	signed, err := token.SignedString(phoneHomeHmacSecret)
	if err != nil {
		return "", err
	}
	return signed, nil
}

// FromJWT parses a JWT-String, validates the JWT and returns the contained claims
func FromJWT(t string) (*PhoneHomeClaims, error) {
	token, err := jwt.ParseWithClaims(t, &PhoneHomeClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
		}
		return phoneHomeHmacSecret, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*PhoneHomeClaims); ok && token.Valid {
		return claims, nil
	}
	return nil, errors.New("claims could not be parsed")
}
