package foundation

// Port of consolidate_response.dart (Flutter 3.50.1).

import (
	"bytes"
	"compress/gzip"
	goio "io"

	"github.com/jonathanperis/flutter-go/flutter_go/async"
	"github.com/jonathanperis/flutter-go/flutter_go/io"
)

// BytesReceivedCallback reports download progress: the bytes received so
// far and the total expected (nil when unknown).
type BytesReceivedCallback func(cumulative int, total *int)

// ConsolidateHttpClientResponseBytesOptions are the optional parameters of
// ConsolidateHttpClientResponseBytes.
type ConsolidateHttpClientResponseBytesOptions struct {
	// DisableAutoUncompress is autoUncompress: false (the default, true,
	// gunzips a compressed response).
	DisableAutoUncompress bool
	OnBytesReceived       BytesReceivedCallback
}

// ConsolidateHttpClientResponseBytes collects the response's body into one
// byte slice, gunzipping it when its compression state is compressed (and
// autoUncompress is on), and reporting progress (in received, possibly
// compressed, bytes) to OnBytesReceived. The future fails with the
// response's first error, or with OnBytesReceived's panic (which cancels
// the subscription).
//
// Divergence: Dart's gzip decoder decodes chunk by chunk; this one decodes
// when the response is done (the result and the progress reports are the
// same).
func ConsolidateHttpClientResponseBytes(response io.HttpClientResponse, opts ...ConsolidateHttpClientResponseBytesOptions) async.Future[[]byte] {
	var o ConsolidateHttpClientResponseBytesOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	autoUncompress := !o.DisableAutoUncompress
	completer := async.NewSyncCompleter[[]byte]()
	output := &outputBuffer{}
	var sink byteSink = output
	var expectedContentLength *int
	if n := response.ContentLength(); n != -1 {
		expectedContentLength = &n
	}
	switch response.CompressionState() {
	case io.HttpClientResponseCompressionStateCompressed:
		if autoUncompress {
			// We need to un-compress the bytes as they come in.
			sink = &gzipSink{output: output}
		}
	case io.HttpClientResponseCompressionStateDecompressed:
		// response.contentLength will not match our bytes stream, so we
		// declare that we don't know the expected content length.
		expectedContentLength = nil
	case io.HttpClientResponseCompressionStateNotCompressed:
	}
	bytesReceived := 0
	var subscription async.StreamSubscription[[]byte]
	subscription = response.Listen(func(chunk []byte) {
		sink.add(chunk)
		if o.OnBytesReceived != nil {
			bytesReceived += len(chunk)
			if err := callBytesReceived(o.OnBytesReceived, bytesReceived, expectedContentLength); err != nil {
				completer.CompleteError(err)
				subscription.Cancel()
				return
			}
		}
	}, async.StreamListenOptions{
		OnDone: func() {
			sink.close()
			completer.Complete(output.bytes())
		},
		OnError:       completer.CompleteError,
		CancelOnError: true,
	})
	return completer.Future()
}

func callBytesReceived(cb BytesReceivedCallback, cumulative int, total *int) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = panicAsError(r)
		}
	}()
	cb(cumulative, total)
	return nil
}

type byteSink interface {
	add(chunk []byte)
	close()
}

// outputBuffer is _OutputBuffer: chunks concatenated on close.
type outputBuffer struct {
	chunks        [][]byte
	contentLength int
	result        []byte
	closed        bool
}

func (b *outputBuffer) add(chunk []byte) {
	dartAssert(!b.closed, nil)
	b.chunks = append(b.chunks, chunk)
	b.contentLength += len(chunk)
}

func (b *outputBuffer) close() {
	if b.closed {
		// We've already been closed; this is a no-op.
		return
	}
	b.result = make([]byte, b.contentLength)
	offset := 0
	for _, chunk := range b.chunks {
		offset += copy(b.result[offset:], chunk)
	}
	b.chunks = nil
	b.closed = true
}

func (b *outputBuffer) bytes() []byte {
	dartAssert(b.closed, nil)
	return b.result
}

// gzipSink is gzip.decoder.startChunkedConversion(output).
type gzipSink struct {
	output     *outputBuffer
	compressed bytes.Buffer
}

func (s *gzipSink) add(chunk []byte) { s.compressed.Write(chunk) }

func (s *gzipSink) close() {
	r, err := gzip.NewReader(&s.compressed)
	if err != nil {
		panic(err)
	}
	decoded, err := goio.ReadAll(r)
	if err != nil {
		panic(err)
	}
	s.output.add(decoded)
	s.output.close()
}
