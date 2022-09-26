package util

import (
	"testing"
	"time"
)

func TestScheduleCron(t *testing.T) {
	t.Log("Yo")

	s := ScheduleCron("*/1 * * * * *", func() {
		t.Log("Yo")
	})
	s.StartAsync()

	time.Sleep(time.Second * 5)
}
