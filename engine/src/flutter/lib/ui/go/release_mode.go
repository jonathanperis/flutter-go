//go:build flutter_release || flutter_profile

package ui

// debugMode is false in release and profile builds (see debug_mode.go).
const debugMode = false
