package common

import (
	"testing"
	"time"
)

func TestLTimeOp(t *testing.T) {
	defer LTimeOp(time.Now(), "myOp")
}
func TestTimeOp(t *testing.T) {
	defer TimeOp(EmptyExecContext(), time.Now(), "myOp")
}

func TestPTimeOp(t *testing.T) {
	defer PTimeOp(time.Now(), "myOp")
}
