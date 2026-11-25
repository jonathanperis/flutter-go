// Package foundation is Flutter's foundation library
// (package:flutter/foundation.dart): the lowest framework layer.
//
// Ported from packages/flutter/lib/src/foundation (Flutter 3.50.1), file
// by file: each Go file is named after the Dart file it ports.
package foundation

// KReleaseMode and KProfileMode come from build tags here (Dart reads
// dart.vm.product / dart.vm.profile from the compile-time environment):
// build with `-tags flutter_release` or `-tags flutter_profile`.
const KDebugMode = !KReleaseMode && !KProfileMode

// PrecisionErrorTolerance is the epsilon used where Flutter compares doubles
// for "close enough".
const PrecisionErrorTolerance = 1e-10

// KIsWeb and KIsWasm are false: flutter-go doesn't target the web.
const (
	KIsWeb  = false
	KIsWasm = false
)
