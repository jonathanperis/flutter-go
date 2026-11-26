package services

// Port of flavor.dart (Flutter 3.50.1).

// appFlavorDefine is the FLUTTER_APP_FLAVOR compile-time define (Dart's
// String.fromEnvironment): set it with
// `go build -ldflags "-X github.com/jonathanperis/flutter-go/packages/flutter/lib/src/services.appFlavorDefine=..."`.
var appFlavorDefine string

// AppFlavor is appFlavor: the flavor this app was built with (the
// FLUTTER_APP_FLAVOR define), nil when there is none or it is empty.
var AppFlavor = func() *string {
	if appFlavorDefine != "" {
		v := appFlavorDefine
		return &v
	}
	return nil
}()
