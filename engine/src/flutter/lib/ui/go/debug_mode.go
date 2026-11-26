//go:build !flutter_release && !flutter_profile

package ui

// debugMode is whether Dart's assert(() {...}()) blocks run, and whether
// the engine is a debug build (no NDEBUG): in debug builds only, as
// foundation's KDebugMode (dart:ui can't import foundation).
const debugMode = true
