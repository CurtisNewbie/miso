package miso

import (
	"testing"
	"time"
)

func TestTimeOp(t *testing.T) {
	defer TimeOp(EmptyRail(), time.Now(), "TimeOp, %v", "123")
	defer TimeOp(EmptyRail(), time.Now(), "TimeOp")
	time.Sleep(time.Second)
}
