package util

import (
	"net"
	"strings"
)

const (
	LoopbackLocalHost = "localhost"
	Loopback127       = "127.0.0.1"
	LocalIpAny        = "0.0.0.0"
)

// Get local ipv4 address (excluding loopback address)
func GetLocalIPV4() string {
	// src: https://stackoverflow.com/questions/23558425/how-do-i-get-the-local-ip-address-in-go
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}
	for _, address := range addrs {
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}
	return ""
}

// Check whether the address is local (localhost/127.0.0.1)
func IsLocalAddress(address string) bool {
	return address == Loopback127 || strings.ToLower(address) == LoopbackLocalHost
}
