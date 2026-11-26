package painting_test

// Port of packages/flutter/test/painting/image_provider_network_image_test.dart
// (Flutter 3.50.1): all 8 tests. HttpOverrides (which makes `HttpClient()`
// throw) has no Go counterpart: io.NewHttpClient is never called because
// DebugNetworkImageHttpClientProvider is always set, as the Dart tearDown
// checks with createHttpClientCalls. The zone of "Propagates http client
// errors during resolve()" is the loop's UncaughtError hook.

import (
	"math"
	"net/url"
	"os"
	"testing"

	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	. "github.com/jonathanperis/flutter-go/packages/flutter/lib/src/painting"
	flutter_test "github.com/jonathanperis/flutter-go/packages/flutter_test/lib/src"
	"github.com/jonathanperis/flutter-go/src/async"
	"github.com/jonathanperis/flutter-go/src/dart"
	"github.com/jonathanperis/flutter-go/src/io"
)

type fakeHttpClient struct {
	request     *fakeHttpClientRequest
	thrownError any
}

func newFakeHttpClient() *fakeHttpClient {
	return &fakeHttpClient{request: &fakeHttpClientRequest{response: &fakeHttpClientResponse{statusCode: io.HttpStatusOk}}}
}

func (c *fakeHttpClient) SetAutoUncompress(bool) {}
func (c *fakeHttpClient) GetUrl(*url.URL) async.Future[io.HttpClientRequest] {
	return async.Async(func() io.HttpClientRequest {
		if c.thrownError != nil {
			panic(c.thrownError)
		}
		return c.request
	})
}

type fakeHttpClientRequest struct{ response *fakeHttpClientResponse }

func (r *fakeHttpClientRequest) Headers() io.HttpHeaders { return fakeHttpHeaders{} }
func (r *fakeHttpClientRequest) Close() async.Future[io.HttpClientResponse] {
	return async.Async(func() io.HttpClientResponse { return r.response })
}

type fakeHttpHeaders struct{}

func (fakeHttpHeaders) Add(string, any) {}

type fakeHttpClientResponse struct {
	drained       bool
	statusCode    int
	contentLength int
	content       [][]byte
}

func (r *fakeHttpClientResponse) StatusCode() int    { return r.statusCode }
func (r *fakeHttpClientResponse) ContentLength() int { return r.contentLength }
func (r *fakeHttpClientResponse) CompressionState() io.HttpClientResponseCompressionState {
	return io.HttpClientResponseCompressionStateNotCompressed
}
func (r *fakeHttpClientResponse) Listen(onData func([]byte), opts ...async.StreamListenOptions) async.StreamSubscription[[]byte] {
	return async.StreamFromIterable(func(yield func([]byte) bool) {
		for _, c := range r.content {
			if !yield(c) {
				return
			}
		}
	}).Listen(onData, opts...)
}
func (r *fakeHttpClientResponse) Drain() async.Future[async.Void] {
	r.drained = true
	return async.FutureValue(async.Void{})
}

// socketException stands for dart:io's SocketException.
type socketException struct{ message string }

func (e socketException) Error() string { return "SocketException: " + e.message }

func newNetworkTester(t *testing.T) (*flutter_test.Tester, *fakeHttpClient) {
	tester := flutter_test.New(t)
	httpClient := newFakeHttpClient()
	DebugNetworkImageHttpClientProvider = func() io.HttpClient { return httpClient }
	t.Cleanup(func() {
		DebugNetworkImageHttpClientProvider = nil
		ImageCacheInstance().Clear()
		ImageCacheInstance().ClearLiveImages()
	})
	return tester, httpClient
}

func baseResolve(ref string) string {
	wd, _ := os.Getwd()
	u, _ := url.Parse(ref)
	return (&url.URL{Scheme: "file", Path: wd + "/"}).ResolveReference(u).String()
}

func checkStatusCodeError(t *testing.T, tester *flutter_test.Tester, httpClient *fakeHttpClient, imageProvider ImageProvider, checkPending bool) {
	t.Helper()
	const errorStatusCode = io.HttpStatusNotFound
	const requestURL = "foo-url"
	httpClient.request.response.statusCode = errorStatusCode
	caughtError := async.NewCompleter[any]()
	c := ImageCacheInstance()
	if c.PendingImageCount() != 0 || !c.StatusForKey(imageProvider).Untracked() {
		t.Fatal("tracked")
	}
	result := imageProvider.Resolve(ImageConfigurationEmpty)
	if c.PendingImageCount() != 1 {
		t.Fatal("not pending")
	}
	if checkPending && !c.StatusForKey(imageProvider).Pending {
		t.Fatal("not pending")
	}
	result.AddListener(NewImageStreamListener(func(*ImageInfo, bool) {},
		ImageStreamListenerOptions{OnError: func(err any, _ dart.StackTrace) { caughtError.Complete(err) }}))
	err := await(tester, caughtError.Future())
	await(tester, async.FutureDelayed(0, func() int { return 0 }))
	if c.PendingImageCount() != 0 || !c.StatusForKey(imageProvider).Untracked() {
		t.Error("still tracked")
	}
	e, ok := err.(*NetworkImageLoadException)
	if !ok || e.StatusCode != errorStatusCode || e.URI.String() != baseResolve(requestURL) {
		t.Errorf("error %v", err)
	}
	if !httpClient.request.response.drained {
		t.Error("not drained")
	}
}

func TestNetworkImageThrownExceptionWithStatusCode(t *testing.T) {
	tester, httpClient := newNetworkTester(t)
	checkStatusCodeError(t, tester, httpClient, NewNetworkImage("foo-url"), true)
}

func TestNetworkImageThrownExceptionWithStatusCodeResizeImage(t *testing.T) {
	tester, httpClient := newNetworkTester(t)
	checkStatusCodeError(t, tester, httpClient, NewResizeImage(NewNetworkImage("foo-url"), ResizeImageOptions{Width: ip(5), Height: ip(5)}), false)
}

func TestNetworkImageUsesDebugHttpClientProvider(t *testing.T) {
	tester, httpClient := newNetworkTester(t)
	httpClient.thrownError = "client1"
	var captured []any
	loadNetworkImage := func() {
		networkImage := NewNetworkImage("foo")
		completer := networkImage.LoadBuffer(networkImage, basicDecoder)
		completer.AddListener(NewImageStreamListener(func(*ImageInfo, bool) {},
			ImageStreamListenerOptions{OnError: func(err any, _ dart.StackTrace) { captured = append(captured, err) }}))
		tester.Loop().FlushMicrotasks()
	}
	loadNetworkImage()
	if len(captured) != 1 || captured[0] != "client1" {
		t.Fatalf("captured %v", captured)
	}
	client2 := newFakeHttpClient()
	client2.thrownError = "client2"
	DebugNetworkImageHttpClientProvider = func() io.HttpClient { return client2 }
	loadNetworkImage()
	if len(captured) != 2 || captured[1] != "client2" {
		t.Fatalf("captured %v", captured)
	}
}

func TestNetworkImagePropagatesHttpClientErrors(t *testing.T) {
	tester, httpClient := newNetworkTester(t)
	httpClient.thrownError = dartError{}
	uncaught := false
	tester.Loop().UncaughtError = func(error) { uncaught = true }
	defer func() { tester.Loop().UncaughtError = nil }()
	old := foundation.FlutterErrorOnError
	defer func() { foundation.FlutterErrorOnError = old }()
	imageProvider := NewNetworkImage("asdasdasdas")
	caughtError := async.NewCompleter[bool]()
	foundation.FlutterErrorOnError = func(*foundation.FlutterErrorDetails) { panic(dartError{}) }
	result := imageProvider.Resolve(ImageConfigurationEmpty)
	result.AddListener(NewImageStreamListener(func(*ImageInfo, bool) {},
		ImageStreamListenerOptions{OnError: func(any, dart.StackTrace) { caughtError.Complete(true) }}))
	if !await(tester, caughtError.Future()) {
		t.Error("not caught")
	}
	tester.Loop().FlushMicrotasks()
	if uncaught {
		t.Error("uncaught error")
	}
}

func transparentImageChunks(chunkSize int) [][]byte {
	var chunks [][]byte
	for offset := 0; offset < len(kTransparentImage); offset += chunkSize {
		end := min(offset+chunkSize, len(kTransparentImage))
		chunks = append(chunks, append([]byte{}, kTransparentImage[offset:end]...))
	}
	return chunks
}

func TestNetworkImageNotifiesListenersOfChunkEvents(t *testing.T) {
	tester, httpClient := newNetworkTester(t)
	const chunkSize = 8
	chunks := transparentImageChunks(chunkSize)
	imageAvailable := async.NewCompleter[async.Void]()
	r := httpClient.request.response
	r.statusCode, r.contentLength, r.content = io.HttpStatusOk, len(kTransparentImage), chunks
	result := NewNetworkImage("foo").Resolve(ImageConfigurationEmpty)
	var events []*ImageChunkEvent
	result.AddListener(NewImageStreamListener(func(*ImageInfo, bool) { imageAvailable.Complete(async.Void{}) },
		ImageStreamListenerOptions{
			OnChunk: func(e *ImageChunkEvent) { events = append(events, e) },
			OnError: func(err any, _ dart.StackTrace) { imageAvailable.CompleteError(async.PanicError{Value: err}) },
		}))
	await(tester, imageAvailable.Future())
	if len(events) != len(chunks) {
		t.Fatalf("%d events, want %d", len(events), len(chunks))
	}
	for i, e := range events {
		if e.CumulativeBytesLoaded != int(math.Min(float64((i+1)*chunkSize), float64(len(kTransparentImage)))) ||
			*e.ExpectedTotalBytes != len(kTransparentImage) {
			t.Errorf("event %d: %d/%d", i, e.CumulativeBytesLoaded, *e.ExpectedTotalBytes)
		}
	}
}

func TestNetworkImageEvictedOnSocketException(t *testing.T) {
	tester, _ := newNetworkTester(t)
	mockHttpClient := newFakeHttpClient()
	mockHttpClient.thrownError = socketException{"test exception"}
	DebugNetworkImageHttpClientProvider = func() io.HttpClient { return mockHttpClient }
	imageProvider := NewNetworkImage("testing.url")
	c := ImageCacheInstance()
	if c.PendingImageCount() != 0 || !c.StatusForKey(imageProvider).Untracked() {
		t.Fatal("tracked")
	}
	result := imageProvider.Resolve(ImageConfigurationEmpty)
	if c.PendingImageCount() != 1 || !c.StatusForKey(imageProvider).Pending {
		t.Fatal("not pending")
	}
	caughtError := async.NewCompleter[any]()
	result.AddListener(NewImageStreamListener(func(*ImageInfo, bool) {},
		ImageStreamListenerOptions{OnError: func(err any, _ dart.StackTrace) { caughtError.Complete(err) }}))
	err := await(tester, caughtError.Future())
	if _, ok := err.(socketException); !ok {
		t.Errorf("error %v", err)
	}
	if c.PendingImageCount() != 0 || !c.StatusForKey(imageProvider).Untracked() || c.ContainsKey(result) {
		t.Error("still tracked")
	}
}

func TestNetworkImageWithSameArgumentsUsesCache(t *testing.T) {
	tester, _ := newNetworkTester(t)
	mockHttpClient := newFakeHttpClient()
	DebugNetworkImageHttpClientProvider = func() io.HttpClient { return mockHttpClient }
	r := mockHttpClient.request.response
	r.statusCode, r.contentLength, r.content = io.HttpStatusOk, len(kTransparentImage), [][]byte{append([]byte{}, kTransparentImage...)}
	imageAvailable1 := async.NewCompleter[async.Void]()
	imageProvider1 := NewNetworkImage("testing.url", NetworkImageOptions{Headers: map[string]string{"key": "value"}})
	if ImageCacheInstance().LiveImageCount() != 0 {
		t.Fatal("live")
	}
	imageProvider1.Resolve(ImageConfigurationEmpty).AddListener(NewImageStreamListener(func(*ImageInfo, bool) {
		imageAvailable1.Complete(async.Void{})
	}))
	await(tester, imageAvailable1.Future())
	if ImageCacheInstance().LiveImageCount() != 1 {
		t.Fatal("not live")
	}
	// NetworkImage should not make another request.
	r.statusCode = 400
	imageAvailable2 := async.NewCompleter[async.Void]()
	imageProvider2 := NewNetworkImage("testing.url", NetworkImageOptions{Headers: map[string]string{"key": "value"}})
	imageProvider2.Resolve(ImageConfigurationEmpty).AddListener(NewImageStreamListener(func(*ImageInfo, bool) {
		imageAvailable2.Complete(async.Void{})
	}))
	await(tester, imageAvailable2.Future())
	if ImageCacheInstance().LiveImageCount() != 1 {
		t.Error("live count")
	}
}

func TestNetworkImageSetsTag(t *testing.T) {
	_, httpClient := newNetworkTester(t)
	const u = "http://test.png"
	r := httpClient.request.response
	r.statusCode, r.contentLength, r.content = io.HttpStatusOk, len(kTransparentImage), transparentImageChunks(8)
	provider := NewNetworkImage(u)
	completer := provider.LoadBuffer(provider, noOpDecoderBufferCallback)
	if l := completer.DebugLabel(); l == nil || *l != u {
		t.Errorf("label %v", l)
	}
}
