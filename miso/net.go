package miso

import (
	"net"
	"time"
)

// Check whether host's port is opened, connection is always closed.
func CheckPortOpened(host string, port string, timeout time.Duration) error {
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(host, port), timeout)
	if err != nil {
		return err
	}
	if conn != nil {
		defer conn.Close()
	}
	return nil
}
