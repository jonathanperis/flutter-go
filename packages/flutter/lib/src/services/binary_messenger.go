package services

// Port of binary_messenger.dart (Flutter 3.50.1).

import (
	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/src/async"
)

// MessageHandler takes a platform message and asynchronously returns an
// encoded response. It may return a nil future (Dart's `Future<ByteData?>?`
// null), which means no response (null).
type MessageHandler func(message []byte) async.Future[[]byte]

// BinaryMessenger sends binary data across the Flutter platform barrier,
// and registers handlers for incoming messages.
type BinaryMessenger interface {
	// HandlePlatformMessage queues a message, as if a plugin on the platform
	// thread sent it to the framework on channel (deprecated in Flutter: use
	// ServicesBinding.ChannelBuffers().Push, or a TestDefaultBinaryMessenger
	// in tests). callback, if not nil, eventually receives the response.
	HandlePlatformMessage(channel string, data []byte, callback ui.PlatformMessageResponseCallback) async.Future[[]byte]

	// Send sends a binary message to the platform plugins on channel. The
	// returned future (which may be nil, meaning a null response) completes
	// with the response, undecoded.
	Send(channel string, message []byte) async.Future[[]byte]

	// SetMessageHandler sets the callback receiving messages from the
	// platform plugins on channel, without decoding them; nil removes it.
	// The handler's result, if not nil, is sent as the response.
	SetMessageHandler(channel string, handler MessageHandler)
}

// awaitOrNull is Dart's `await` of a nullable future (`Future<T>?`): a nil
// future is awaited as the value null (which still suspends until a
// microtask, as Dart's `await null` does).
func awaitOrNull[T any](f async.Future[T]) T {
	if f == nil {
		var zero T
		f = async.FutureValue(zero)
	}
	return f.Await()
}
