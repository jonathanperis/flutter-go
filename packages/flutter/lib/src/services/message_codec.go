package services

// Port of message_codec.dart (Flutter 3.50.1).
//
// A Dart ByteData is a Go []byte: a nil slice is Dart's null, an empty
// non-nil slice a zero-length ByteData. Codec values (Dart's dynamic) are
// `any`; see message_codecs.go for how Dart's values map to Go's.

import (
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	"github.com/jonathanperis/flutter-go/src/dart"
)

// MessageCodec is a message encoding/decoding mechanism. Both operations
// panic if conversion fails (a programming error). T's nil is Dart's null:
// a codec of nullable strings is a MessageCodec[*string].
type MessageCodec[T any] interface {
	// EncodeMessage encodes message in binary; it returns nil if the
	// message is null.
	EncodeMessage(message T) []byte
	// DecodeMessage decodes message from binary; it returns T's nil if the
	// message is nil.
	DecodeMessage(message []byte) T
}

// MethodCall is a command object representing the invocation of a named
// method. Arguments must be a valid value for the MethodCodec used (nil is
// null, the default).
type MethodCall struct {
	// Method is the name of the method to be called.
	Method string
	// Arguments are the arguments for the method.
	Arguments any
}

// NewMethodCall is MethodCall(method, [arguments]).
func NewMethodCall(method string, arguments ...any) MethodCall {
	var args any
	if len(arguments) > 0 {
		args = arguments[0]
	}
	return MethodCall{Method: method, Arguments: args}
}

func (c MethodCall) String() string {
	return foundation.ObjectRuntimeType(c, "MethodCall") + "(" + c.Method + ", " + dart.ToString(c.Arguments) + ")"
}

// MethodCodec is a codec for method calls and enveloped results. All
// operations panic if conversion fails.
type MethodCodec interface {
	// EncodeMethodCall encodes methodCall into binary.
	EncodeMethodCall(methodCall MethodCall) []byte
	// DecodeMethodCall decodes a method call from binary.
	DecodeMethodCall(methodCall []byte) MethodCall
	// DecodeEnvelope decodes a result envelope from binary: it panics with
	// a *PlatformException if the envelope represents an error, and
	// returns the enveloped result otherwise.
	DecodeEnvelope(envelope []byte) any
	// EncodeSuccessEnvelope encodes a successful result into a binary
	// envelope.
	EncodeSuccessEnvelope(result any) []byte
	// EncodeErrorEnvelope encodes an error result into a binary envelope:
	// the error code, and optionally a human-readable message and details
	// (the fields of PlatformException). Dart's required named `code:` is
	// positional.
	EncodeErrorEnvelope(code string, opts ...EncodeErrorEnvelopeOptions) []byte
}

// EncodeErrorEnvelopeOptions are MethodCodec.EncodeErrorEnvelope's
// optional named parameters (a nil Message is null).
type EncodeErrorEnvelopeOptions struct {
	Message *string
	Details any
}

func errorEnvelopeOptions(opts []EncodeErrorEnvelopeOptions) EncodeErrorEnvelopeOptions {
	if len(opts) > 0 {
		return opts[0]
	}
	return EncodeErrorEnvelopeOptions{}
}

// PlatformException is thrown (as a panic, or a future's error) to indicate
// that a platform interaction failed in the platform plugin. It is used by
// pointer.
type PlatformException struct {
	// Code is an error code.
	Code string
	// Message is a human-readable error message, possibly nil (null).
	Message *string
	// Details are error details, possibly nil.
	Details any
	// Stacktrace is the native stack trace of the error, possibly nil.
	Stacktrace *string
}

// PlatformExceptionOptions are PlatformException's optional named
// parameters.
type PlatformExceptionOptions struct {
	Message    *string
	Details    any
	Stacktrace *string
}

// NewPlatformException is PlatformException(code:, message:, details:,
// stacktrace:) (the required named `code:` is positional).
func NewPlatformException(code string, opts ...PlatformExceptionOptions) *PlatformException {
	var o PlatformExceptionOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	return &PlatformException{Code: code, Message: o.Message, Details: o.Details, Stacktrace: o.Stacktrace}
}

func nullableString(s *string) string {
	if s == nil {
		return "null"
	}
	return *s
}

// Error is PlatformException.toString.
func (e *PlatformException) Error() string {
	return "PlatformException(" + e.Code + ", " + nullableString(e.Message) + ", " + dart.ToString(e.Details) + ", " +
		nullableString(e.Stacktrace) + ")"
}

// MissingPluginException is thrown to indicate that a platform interaction
// failed to find a handling plugin. It is used by pointer.
type MissingPluginException struct {
	// Message is a human-readable error message, possibly nil (null).
	Message *string
}

// NewMissingPluginException is MissingPluginException([message]).
func NewMissingPluginException(message ...string) *MissingPluginException {
	if len(message) > 0 {
		m := message[0]
		return &MissingPluginException{Message: &m}
	}
	return &MissingPluginException{}
}

// Error is MissingPluginException.toString.
func (e *MissingPluginException) Error() string {
	return "MissingPluginException(" + nullableString(e.Message) + ")"
}
