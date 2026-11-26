package services

// Partial port of packages/flutter/lib/src/services/binary_messenger.dart,
// platform_channel.dart and system_channels.dart (Flutter 3.50.1):
// BinaryMessenger, MessageHandler, BasicMessageChannel and
// SystemChannels.accessibility, what semantics sends events through. Method
// channels, event channels, the other system channels, channel profiling
// and ServicesBinding are not ported yet.
//
// A channel without an explicit messenger uses ServicesBinding.instance.
// defaultBinaryMessenger in Flutter. Until ServicesBinding is ported, that
// is DefaultBinaryMessenger, a variable the binding (or a test, as
// flutter_test's TestDefaultBinaryMessenger) sets; while it is nil, using
// such a channel fails as Flutter fails without a binding.

import (
	"github.com/jonathanperis/flutter-go/src/async"
)

// MessageHandler is a function which takes a platform message and
// asynchronously returns an encoded response (a nil future, or a nil
// result, is Dart's null).
type MessageHandler func(message []byte) async.Future[[]byte]

// BinaryMessenger is a messenger which sends binary data across the Flutter
// platform barrier.
type BinaryMessenger interface {
	// HandlePlatformMessage queues a message (deprecated in Flutter: use
	// ServicesBinding.instance.channelBuffers.push).
	HandlePlatformMessage(channel string, data []byte, callback func(data []byte)) async.Future[async.Void]
	// Send sends a binary message to the platform plugins on the given
	// channel; the future (nil: null) completes with the response.
	Send(channel string, message []byte) async.Future[[]byte]
	// SetMessageHandler sets a callback for receiving messages from the
	// platform plugins on the given channel (nil removes it).
	SetMessageHandler(channel string, handler MessageHandler)
}

// DefaultBinaryMessenger stands in for ServicesBinding.instance.
// defaultBinaryMessenger until ServicesBinding is ported (see the file
// comment).
var DefaultBinaryMessenger BinaryMessenger

// errNoServicesBinding is Flutter's error when no binding is initialized.
type errNoServicesBinding struct{}

func (errNoServicesBinding) Error() string {
	return "Binding has not yet been initialized.\n" +
		"The \"instance\" getter on the ServicesBinding binding mixin is only available once that binding has been initialized."
}

// BasicMessageChannel is a named channel for communicating with platform
// plugins using asynchronous message passing, encoded with Codec.
type BasicMessageChannel[T any] struct {
	// Name is the logical channel on which communication happens.
	Name string
	// Codec is the message codec used by this channel.
	Codec           MessageCodec[T]
	binaryMessenger BinaryMessenger
}

// NewBasicMessageChannel is BasicMessageChannel(name, codec,
// {binaryMessenger}) (a nil or missing messenger is the default one).
func NewBasicMessageChannel[T any](name string, codec MessageCodec[T], binaryMessenger ...BinaryMessenger) *BasicMessageChannel[T] {
	c := &BasicMessageChannel[T]{Name: name, Codec: codec}
	if len(binaryMessenger) > 0 {
		c.binaryMessenger = binaryMessenger[0]
	}
	return c
}

// BinaryMessenger is the messenger which sends the bytes this channel
// encodes (it panics, as Flutter throws, when there is none).
func (c *BasicMessageChannel[T]) BinaryMessenger() BinaryMessenger {
	if c.binaryMessenger != nil {
		return c.binaryMessenger
	}
	if DefaultBinaryMessenger == nil {
		panic(errNoServicesBinding{})
	}
	return DefaultBinaryMessenger
}

// Send sends the specified message to the platform plugins on this
// channel. Returns a future which completes to the received response,
// which may be null (Dart's send is async: errors, even the lookup of the
// messenger, complete the future with an error).
func (c *BasicMessageChannel[T]) Send(message T) async.Future[T] {
	return async.Async(func() T {
		response := c.BinaryMessenger().Send(c.Name, c.Codec.EncodeMessage(message))
		var data []byte
		if response != nil {
			data = response.Await()
		}
		return c.Codec.DecodeMessage(data)
	})
}

// SetMessageHandler sets a callback for receiving messages from the
// platform plugins on this channel (nil removes it).
func (c *BasicMessageChannel[T]) SetMessageHandler(handler func(message T) async.Future[T]) {
	if handler == nil {
		c.BinaryMessenger().SetMessageHandler(c.Name, nil)
		return
	}
	c.BinaryMessenger().SetMessageHandler(c.Name, func(message []byte) async.Future[[]byte] {
		return async.Async(func() []byte {
			return c.Codec.EncodeMessage(handler(c.Codec.DecodeMessage(message)).Await())
		})
	})
}

type systemChannels struct {
	// Accessibility is a BasicMessageChannel for accessibility events
	// ('flutter/accessibility', StandardMessageCodec).
	Accessibility *BasicMessageChannel[any]
}

// SystemChannels holds the platform channels the engine uses (PARTIAL:
// accessibility only so far).
var SystemChannels = systemChannels{
	Accessibility: NewBasicMessageChannel[any]("flutter/accessibility", StandardMessageCodec{}),
}
