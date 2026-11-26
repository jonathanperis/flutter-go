// Package io is the subset of dart:io that Flutter's API mentions so far:
// the HttpClientResponse that foundation's
// consolidateHttpClientResponseBytes reads (dart:io's http.dart, Dart SDK
// shipped with Flutter 3.50.1). Only the members Flutter uses are declared;
// dart:io's HTTP client itself is not ported (a response is any Go value
// implementing the interface, e.g. an adapter over net/http).
package io

import "github.com/jonathanperis/flutter-go/src/async"

// HttpClientResponseCompressionState describes how a response's body was
// compressed and whether the client decompressed it.
type HttpClientResponseCompressionState int

const (
	// HttpClientResponseCompressionStateNotCompressed: the body was not
	// compressed.
	HttpClientResponseCompressionStateNotCompressed HttpClientResponseCompressionState = iota
	// HttpClientResponseCompressionStateDecompressed: the body was
	// compressed and the client decompressed it (ContentLength is the
	// compressed length).
	HttpClientResponseCompressionStateDecompressed
	// HttpClientResponseCompressionStateCompressed: the body is compressed
	// and was not decompressed.
	HttpClientResponseCompressionStateCompressed
)

func (s HttpClientResponseCompressionState) String() string {
	return [...]string{
		"HttpClientResponseCompressionState.notCompressed",
		"HttpClientResponseCompressionState.decompressed",
		"HttpClientResponseCompressionState.compressed",
	}[s]
}

// HttpClientResponse is an HTTP response: a stream of body chunks, with
// its content length (-1 when unknown) and compression state.
type HttpClientResponse interface {
	async.Stream[[]byte]
	StatusCode() int
	ContentLength() int
	CompressionState() HttpClientResponseCompressionState
}
