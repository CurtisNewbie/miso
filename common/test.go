package common

import "testing"

// Test if expected == actual, else call t.Fatalf with values printed
func TestEqual[V comparable](t *testing.T, expected V, actual V) {
	if expected != actual {
		t.Fatalf("Expected: %v, actual: %v", expected, actual)
	}
}

// Test if actual is true, else call t.Fatalf with values printed
func TestTrue(t *testing.T, actual bool) {
	TestEqual(t, true, actual)
}

// Test if actual is false, else call t.Fatalf with values printed
func TestFalse(t *testing.T, actual bool) {
	TestEqual(t, false, actual)
}
