package foundation

// Port of capabilities.dart with _capabilities_io.dart (Flutter 3.50.1):
// flutter-go is never a web build, so these are the dart:io versions, which
// throw (Flutter only reads them under kIsWeb).

import "github.com/jonathanperis/flutter-go/flutter_go/dart"

// IsCanvasKit reports whether the web renderer is CanvasKit. Not
// implemented outside the web: it panics with an UnimplementedError, as on
// dart:io.
func IsCanvasKit() bool {
	panic(dart.UnimplementedError{Message: "isCanvasKit is not implemented for dart:io."})
}

// IsSkwasm reports whether the web renderer is Skwasm. It panics with an
// UnimplementedError, as on dart:io.
func IsSkwasm() bool {
	panic(dart.UnimplementedError{Message: "isSkwasm is not implemented for dart:io."})
}

// IsSkiaWeb is IsCanvasKit() || IsSkwasm() (so it panics too).
func IsSkiaWeb() bool { return IsCanvasKit() || IsSkwasm() }
