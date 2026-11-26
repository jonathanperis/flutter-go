//go:build !flutter_release && !flutter_profile

package listen

// debugMode is whether Dart's assert(() {...}()) blocks run: in debug
// builds only, as foundation's KDebugMode (package:listen can't import
// foundation, which depends on it).
const debugMode = true
