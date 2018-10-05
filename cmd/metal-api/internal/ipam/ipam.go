package ipam

import (
	"fmt"
)

var (
	ipPool = map[string]bool{
		"10.0.2.100": true,
		"10.0.2.101": true,
		"10.0.2.102": true,
		"10.0.2.103": true,
		"10.0.2.104": true,
	}
)

func AllocateIP() (string, error) {
	for ip := range ipPool {
		delete(ipPool, ip)
		return ip, nil
	}
	return "", fmt.Errorf("no ip addresses left for allocation")
}

func FreeIP(ip string) error {
	ipPool[ip] = true
	return nil
}
