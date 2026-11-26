// Port of packages/flutter/test/foundation/consolidate_response_test.dart
// (all 10 tests). MockHttpClientResponse implements io.HttpClientResponse
// with Stream.fromIterable / Stream.fromFuture, as Flutter's mock does.
package foundation

import (
	"bytes"
	"compress/gzip"
	"errors"
	"slices"
	"testing"

	"github.com/jonathanperis/flutter-go/src/async"
	"github.com/jonathanperis/flutter-go/src/io"
)

var (
	chunkOne = []byte{0, 1, 2, 3, 4, 5}
	chunkTwo = []byte{6, 7, 8, 9, 10}
)

type mockHttpClientResponse struct {
	err              error
	chunkOne         []byte
	chunkTwo         []byte
	contentLength    int
	compressionState io.HttpClientResponseCompressionState
}

func (r *mockHttpClientResponse) ContentLength() int { return r.contentLength }
func (r *mockHttpClientResponse) StatusCode() int    { return 200 }
func (r *mockHttpClientResponse) CompressionState() io.HttpClientResponseCompressionState {
	return r.compressionState
}

func (r *mockHttpClientResponse) Listen(onData func([]byte), opts ...async.StreamListenOptions) async.StreamSubscription[[]byte] {
	if r.err != nil {
		return async.StreamFromFuture(async.FutureError[[]byte](r.err)).Listen(onData, opts...)
	}
	return async.StreamFromIterable(slices.Values([][]byte{r.chunkOne, r.chunkTwo})).Listen(onData, opts...)
}

// consolidate awaits ConsolidateHttpClientResponseBytes on a fresh loop.
func consolidate(t *testing.T, response io.HttpClientResponse, opts ...ConsolidateHttpClientResponseBytesOptions) ([]byte, error) {
	t.Helper()
	var result []byte
	err := runLoop(t, func() { result = ConsolidateHttpClientResponseBytes(response, opts...).Await() })
	return result, err
}

var eleven = []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10}

func TestConsolidateWithContentLength(t *testing.T) {
	response := &mockHttpClientResponse{chunkOne: chunkOne, chunkTwo: chunkTwo, contentLength: len(chunkOne) + len(chunkTwo)}
	b, err := consolidate(t, response)
	if err != nil || !bytes.Equal(b, eleven) {
		t.Fatal(b, err)
	}
}

func TestConsolidateCompressedWithContentLength(t *testing.T) {
	response := &mockHttpClientResponse{chunkOne: chunkOne, chunkTwo: chunkTwo, contentLength: len(chunkOne)}
	b, err := consolidate(t, response)
	if err != nil || !bytes.Equal(b, eleven) {
		t.Fatal(b, err)
	}
}

func TestConsolidateWithoutContentLength(t *testing.T) {
	response := &mockHttpClientResponse{chunkOne: chunkOne, chunkTwo: chunkTwo, contentLength: -1}
	b, err := consolidate(t, response)
	if err != nil || !bytes.Equal(b, eleven) {
		t.Fatal(b, err)
	}
}

func recordProgress(records *[]any) BytesReceivedCallback {
	return func(cumulative int, total *int) {
		var tv any
		if total != nil {
			tv = *total
		}
		*records = append(*records, cumulative, tv)
	}
}

func TestConsolidateNotifiesOnBytesReceivedForEveryChunk(t *testing.T) {
	syntheticTotal := (len(chunkOne) + len(chunkTwo)) * 2
	response := &mockHttpClientResponse{chunkOne: chunkOne, chunkTwo: chunkTwo, contentLength: syntheticTotal}
	var records []any
	if _, err := consolidate(t, response, ConsolidateHttpClientResponseBytesOptions{OnBytesReceived: recordProgress(&records)}); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(records, []any{len(chunkOne), syntheticTotal, len(chunkOne) + len(chunkTwo), syntheticTotal}) {
		t.Fatal(records)
	}
}

func TestConsolidateForwardsErrorsFromHttpClientResponse(t *testing.T) {
	testError := errors.New("Exception: Test Error")
	response := &mockHttpClientResponse{err: testError, contentLength: -1}
	if _, err := consolidate(t, response); err != testError {
		t.Fatal(err)
	}
}

func TestConsolidatePropagatesErrorIfOnBytesReceivedThrows(t *testing.T) {
	response := &mockHttpClientResponse{chunkOne: chunkOne, chunkTwo: chunkTwo, contentLength: -1}
	_, err := consolidate(t, response, ConsolidateHttpClientResponseBytesOptions{
		OnBytesReceived: func(int, *int) { panic("misbehaving callback") },
	})
	var pe async.PanicError
	if !errors.As(err, &pe) || pe.Value != "misbehaving callback" {
		t.Fatal(err)
	}
}

func gzipped() []byte {
	var b bytes.Buffer
	w := gzip.NewWriter(&b)
	w.Write(eleven)
	w.Close()
	return b.Bytes()
}

func gzippedResponse() (*mockHttpClientResponse, []byte, []byte, []byte) {
	all := gzipped()
	one, two := all[:len(all)/2], all[len(all)/2:]
	return &mockHttpClientResponse{
		chunkOne: one, chunkTwo: two,
		compressionState: io.HttpClientResponseCompressionStateCompressed,
	}, all, one, two
}

func TestConsolidateUncompressesGzipIfAutoUncompress(t *testing.T) {
	response, all, _, _ := gzippedResponse()
	response.contentLength = len(all)
	b, err := consolidate(t, response)
	if err != nil || !bytes.Equal(b, eleven) {
		t.Fatal(b, err)
	}
}

func TestConsolidateReturnsGzippedBytesIfNotAutoUncompress(t *testing.T) {
	response, all, _, _ := gzippedResponse()
	response.contentLength = len(all)
	b, err := consolidate(t, response, ConsolidateHttpClientResponseBytesOptions{DisableAutoUncompress: true})
	if err != nil || !bytes.Equal(b, all) {
		t.Fatal(b, err)
	}
}

func TestConsolidateNotifiesOnBytesReceivedWithGzippedNumbers(t *testing.T) {
	response, all, one, _ := gzippedResponse()
	response.contentLength = len(all)
	var records []any
	if _, err := consolidate(t, response, ConsolidateHttpClientResponseBytesOptions{OnBytesReceived: recordProgress(&records)}); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(records, []any{len(one), len(all), len(all), len(all)}) {
		t.Fatal(records)
	}
}

func TestConsolidateNotifiesNullTotalIfDecompressed(t *testing.T) {
	response, all, one, _ := gzippedResponse()
	syntheticTotal := (len(chunkOne) + len(chunkTwo)) * 2
	response.compressionState = io.HttpClientResponseCompressionStateDecompressed
	response.contentLength = syntheticTotal
	var records []any
	if _, err := consolidate(t, response, ConsolidateHttpClientResponseBytesOptions{OnBytesReceived: recordProgress(&records)}); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(records, []any{len(one), nil, len(all), nil}) {
		t.Fatal(records)
	}
}

// Not in Flutter's tests: an unknown content length (-1) is reported as a
// null total.
func TestConsolidateUnknownLengthIsNullTotal(t *testing.T) {
	response := &mockHttpClientResponse{chunkOne: chunkOne, chunkTwo: chunkTwo, contentLength: -1}
	var records []any
	if _, err := consolidate(t, response, ConsolidateHttpClientResponseBytesOptions{OnBytesReceived: recordProgress(&records)}); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(records, []any{len(chunkOne), nil, len(chunkOne) + len(chunkTwo), nil}) {
		t.Fatal(records)
	}
}
