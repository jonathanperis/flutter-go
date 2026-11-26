package services

// Port of packages/flutter/lib/src/services/scribe.dart (Flutter 3.50.1).

import (
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	"github.com/jonathanperis/flutter-go/src/async"
)

// scribe is the type of Scribe (Dart's abstract final class of static
// methods).
type scribe struct{}

// Scribe allows access to stylus handwriting input on Android
// (Scribe.isFeatureAvailable() is Scribe.IsFeatureAvailable()...).
var Scribe scribe

func (scribe) channel() *MethodChannel { return SystemChannels.Scribe }

// IsFeatureAvailable returns true if the InputMethodManager supports
// Scribe stylus handwriting input; a null reply fails with a FlutterError.
func (s scribe) IsFeatureAvailable() async.Future[bool] {
	return async.Async(func() bool { return scribeNonNullBool(s.channel().InvokeMethod("Scribe.isFeatureAvailable").Await()) })
}

// IsStylusHandwritingAvailable returns true if the InputMethodManager
// supports Scribe stylus handwriting input (deprecated in favor of
// IsFeatureAvailable on Android 14+).
func (s scribe) IsStylusHandwritingAvailable() async.Future[bool] {
	return async.Async(func() bool {
		return scribeNonNullBool(s.channel().InvokeMethod("Scribe.isStylusHandwritingAvailable").Await())
	})
}

// StartStylusHandwriting tells Android to begin receiving stylus
// handwriting input.
func (s scribe) StartStylusHandwriting() async.Future[async.Void] {
	return async.CastFuture[async.Void](s.channel().InvokeMethod("Scribe.startStylusHandwriting"))
}

func scribeNonNullBool(result any) bool {
	if result == nil {
		panic(foundation.NewFlutterError("MethodChannel.invokeMethod unexpectedly returned null."))
	}
	return result.(bool)
}
