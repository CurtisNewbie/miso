package common

import (
	"testing"
	"time"
)

func TestTimeOp(t *testing.T) {
	defer TimeOp(EmptyExecContext(), time.Now(), "myOp")
}

func TestPTimeOp(t *testing.T) {
	defer PTimeOp(time.Now(), "myOp")
}
