package miso

import (
	"testing"
	"time"
)

func TestCheckPortOpened(t *testing.T) {
	err := CheckPortOpened("192.168.0.1", "1087", time.Second)
	t.Logf("Port opened: %v", err)
}
