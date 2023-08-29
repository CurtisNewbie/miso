package core

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

// Test if expected == actual, else call t.Fatalf with values printed
func TestEqual[V comparable](t *testing.T, expected V, actual V) {
	caller := _callerLine()
	if expected != actual {
		t.Fatalf("[FAIL] %v -> Expected: %v, actual: %v", caller, expected, actual)
	} else {
		t.Logf("[PASS] %v -> Expected: %v, actual: %v", caller, expected, actual)
	}
}

// Test if actual is true, else call t.Fatalf with values printed
func TestTrue(t *testing.T, actual bool) {
	caller := _callerLine()
	if !actual {
		t.Fatalf("[FAIL] %v -> Expected: true, actual: %v", caller, actual)
	} else {
		t.Logf("[PASS] %v -> Expected: true, actual: %v", caller, actual)
	}
}

// Test if actual is false, else call t.Fatalf with values printed
func TestFalse(t *testing.T, actual bool) {
	caller := _callerLine()
	if actual {
		t.Fatalf("[FAIL] %v -> Expected: false, actual: %v", caller, actual)
	} else {
		t.Logf("[PASS] %v -> Expected: false, actual: %v", caller, actual)
	}
}

// Test if value is nil, else call t.Fatalf with values printed
func TestIsNil(t *testing.T, value any) {
	caller := _callerLine()
	if value != nil {
		t.Fatalf("[FAIL] %v -> Expected: <nil>, actual: %v", caller, value)
	} else {
		t.Logf("[PASS] %v -> Expected: <nil>, actual: %v", caller, value)
	}
}

// Test if value is not nil, else call t.Fatalf with values printed
func TestNotNil(t *testing.T, value any) {
	caller := _callerLine()
	if value == nil {
		t.Fatalf("[FAIL] %v -> Should not be nil", caller)
	} else {
		t.Logf("[PASS] %v -> Expected: non-nil, actual: %v", caller, value)
	}
}

func _callerLine() string {
	caller := getCaller(4)
	ftkn := strings.Split(caller.File, string(os.PathSeparator))
	file := ftkn[len(ftkn)-1]
	return fmt.Sprintf("%v:%-4v", file, caller.Line)
}
