// Package broken has a test file that does not type-check: apiparity
// falls back to the names it mentions.
package broken

// Quiet is mentioned by the broken test.
func Quiet() {}
