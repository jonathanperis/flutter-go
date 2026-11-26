package broken

import "testing"

// TestBroken does not type-check (Undefined is not declared).
func TestBroken(t *testing.T) {
	Quiet()
	Undefined()
}
