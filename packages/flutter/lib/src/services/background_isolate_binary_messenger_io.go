package services

// Partial port of _background_isolate_binary_messenger_io.dart (Flutter
// 3.50.1; the Go file drops the leading underscore, which would make Go
// ignore it).
//
// ENGINE GAP: a Go program has one isolate (RootIsolateTokenInstance is
// never nil, so platform channels always use the binding's messenger), and
// there are no isolate ports for the engine to reply on
// (PlatformDispatcher.sendPortPlatformMessage). The messenger exists, with
// Flutter's errors, but sending through it panics with an
// UnimplementedError.

import (
	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/src/async"
	"github.com/jonathanperis/flutter-go/src/dart"
)

// BackgroundIsolateBinaryMessenger is the BinaryMessenger of background
// isolates (partial, see the file comment).
type BackgroundIsolateBinaryMessenger struct{}

var backgroundIsolateBinaryMessengerInstance BinaryMessenger

// BackgroundIsolateBinaryMessengerInstance is
// BackgroundIsolateBinaryMessenger.instance: it panics with a StateError
// until BackgroundIsolateBinaryMessengerEnsureInitialized has run.
func BackgroundIsolateBinaryMessengerInstance() BinaryMessenger {
	if backgroundIsolateBinaryMessengerInstance == nil {
		panic(dart.StateError{Message: "The BackgroundIsolateBinaryMessenger.instance value is invalid " +
			"until BackgroundIsolateBinaryMessenger.ensureInitialized is " +
			"executed."})
	}
	return backgroundIsolateBinaryMessengerInstance
}

// BackgroundIsolateBinaryMessengerEnsureInitialized is
// BackgroundIsolateBinaryMessenger.ensureInitialized(token): it creates the
// instance for the isolate token belongs to.
func BackgroundIsolateBinaryMessengerEnsureInitialized(token *ui.RootIsolateToken) {
	if backgroundIsolateBinaryMessengerInstance == nil {
		backgroundIsolateBinaryMessengerInstance = &BackgroundIsolateBinaryMessenger{}
	}
}

// HandlePlatformMessage panics: it is deprecated.
func (m *BackgroundIsolateBinaryMessenger) HandlePlatformMessage(channel string, data []byte, callback ui.PlatformMessageResponseCallback) async.Future[[]byte] {
	panic(dart.UnimplementedError{Message: "handlePlatformMessage is deprecated."})
}

// Send would send the message to the platform through the root isolate
// (ENGINE GAP: it panics, see the file comment).
func (m *BackgroundIsolateBinaryMessenger) Send(channel string, message []byte) async.Future[[]byte] {
	panic(dart.UnimplementedError{Message: "BackgroundIsolateBinaryMessenger.send: Go has no background isolates (ENGINE GAP)"})
}

// SetMessageHandler panics: messages from the host platform always go to
// the root isolate.
func (m *BackgroundIsolateBinaryMessenger) SetMessageHandler(channel string, handler MessageHandler) {
	panic(dart.UnsupportedError{Message: "Background isolates do not support setMessageHandler(). Messages from the host platform always go to the root isolate."})
}
