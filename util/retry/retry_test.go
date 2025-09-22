package retry

import (
	"testing"
	"time"

	"github.com/curtisnewbie/miso/util/cli"
	"github.com/curtisnewbie/miso/util/errs"
)

func TestGetOneWithBackoff(t *testing.T) {
	backoff := []time.Duration{time.Second, time.Millisecond * 500, time.Second}
	var gap time.Duration
	now := time.Now()
	err := CallWithBackoff(backoff, func() error {
		gap = time.Since(now)
		cli.TPrintlnf("call, gap: %v", gap)
		now = time.Now()
		return errs.NewErrf("no")
	})
	if err == nil {
		t.Fatal("err should be nil")
	}
}
