package painting

// Port of _network_image_io.dart (Flutter 3.50.1): NetworkImage over
// dart:io's HttpClient (src/io's HttpClient interface; the shared client is
// io.NewHttpClient, over net/http). Go ignores files whose name starts
// with "_", so the file drops the underscore. image_provider.dart's
// abstract NetworkImage (a factory to this class, or to the web one) is
// this concrete type: flutter-go has no web build.

import (
	"iter"
	"maps"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	"github.com/jonathanperis/flutter-go/src/async"
	"github.com/jonathanperis/flutter-go/src/dart"
	"github.com/jonathanperis/flutter-go/src/io"
)

// NetworkImage fetches the given URL from the network, associating it with
// the given scale. Its key is itself (== compares the URL, scale and
// headers).
type NetworkImage struct {
	ImageProviderBase
	// URL is the URL from which the image will be fetched.
	URL string
	// Scale is the scale to place in the ImageInfo object of the image.
	Scale float64
	// Headers are the HTTP headers sent with the request (nil: none).
	Headers map[string]string
	// WebHtmlElementStrategy is used on the web only.
	WebHtmlElementStrategy WebHtmlElementStrategy
}

// NetworkImageOptions are NetworkImage's named parameters (Scale nil:
// 1.0).
type NetworkImageOptions struct {
	Scale                  *float64
	Headers                map[string]string
	WebHtmlElementStrategy WebHtmlElementStrategy
}

// NewNetworkImage is NetworkImage(url, {scale, headers,
// webHtmlElementStrategy}).
func NewNetworkImage(url string, opts ...NetworkImageOptions) *NetworkImage {
	var o NetworkImageOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	p := &NetworkImage{URL: url, Scale: 1.0, Headers: o.Headers, WebHtmlElementStrategy: o.WebHtmlElementStrategy}
	if o.Scale != nil {
		p.Scale = *o.Scale
	}
	p.SetSelf(p)
	return p
}

// ObtainKey returns the provider itself, synchronously.
func (p *NetworkImage) ObtainKey(configuration ImageConfiguration) async.Future[any] {
	return foundation.NewSynchronousFuture[any](p)
}

func (p *NetworkImage) networkCollector(key *NetworkImage) foundation.InformationCollector {
	return func() iter.Seq[foundation.DiagnosticsNode] {
		return slices.Values([]foundation.DiagnosticsNode{
			foundation.NewDiagnosticsProperty[ImageProvider]("Image provider", p),
			foundation.NewDiagnosticsProperty("Image key", key),
		})
	}
}

// LoadBuffer is LoadImage with the deprecated decoder.
func (p *NetworkImage) LoadBuffer(key any, decode DecoderBufferCallback) ImageStreamCompleter {
	k := key.(*NetworkImage)
	// Ownership of this controller is handed off to [_loadAsync]; it is
	// that method's responsibility to close the controller's stream when
	// the image has been loaded or an error is thrown.
	chunkEvents := async.NewStreamController[*ImageChunkEvent]()
	u := k.URL
	return NewMultiFrameImageStreamCompleter(
		p.loadAsync(k, chunkEvents, func(b *ui.ImmutableBuffer) async.Future[ui.Codec] { return decode(b) }),
		k.Scale, MultiFrameImageStreamCompleterOptions{ChunkEvents: chunkEvents.Stream(), DebugLabel: &u,
			InformationCollector: p.networkCollector(k)})
}

// LoadImage fetches and decodes the image, reporting chunk events while
// it downloads.
func (p *NetworkImage) LoadImage(key any, decode ImageDecoderCallback) ImageStreamCompleter {
	k := key.(*NetworkImage)
	chunkEvents := async.NewStreamController[*ImageChunkEvent]()
	u := k.URL
	return NewMultiFrameImageStreamCompleter(
		p.loadAsync(k, chunkEvents, func(b *ui.ImmutableBuffer) async.Future[ui.Codec] { return decode(b) }),
		k.Scale, MultiFrameImageStreamCompleterOptions{ChunkEvents: chunkEvents.Stream(), DebugLabel: &u,
			InformationCollector: p.networkCollector(k)})
}

// Do not access this field directly; use httpClient instead. We set
// `autoUncompress` to false to ensure that we can trust the value of the
// `Content-Length` HTTP header. We automatically uncompress the content in
// our call to [consolidateHttpClientResponseBytes].
var sharedHttpClient io.HttpClient

func httpClient() io.HttpClient {
	var client io.HttpClient
	if foundation.KDebugMode && DebugNetworkImageHttpClientProvider != nil {
		client = DebugNetworkImageHttpClientProvider()
	}
	if client != nil {
		return client
	}
	if sharedHttpClient == nil {
		sharedHttpClient = io.NewHttpClient()
		sharedHttpClient.SetAutoUncompress(false)
	}
	return sharedHttpClient
}

// uriBase is Uri.base: the current working directory as a file URI.
func uriBase() *url.URL {
	wd, _ := os.Getwd()
	p := filepath.ToSlash(wd)
	if !strings.HasSuffix(p, "/") {
		p += "/"
	}
	return &url.URL{Scheme: "file", Path: p}
}

func (p *NetworkImage) loadAsync(key *NetworkImage, chunkEvents *async.StreamController[*ImageChunkEvent], decode simpleDecoderCallback) async.Future[ui.Codec] {
	return async.Async(func() ui.Codec {
		defer func() {
			chunkEvents.Close()
		}()
		var codec ui.Codec
		exception, _, threw := tryCatch(func() {
			dartAssert(key == p)
			ref, err := url.Parse(key.URL)
			if err != nil {
				panic(dart.NewFormatException(err.Error()))
			}
			resolved := uriBase().ResolveReference(ref)
			request := httpClient().GetUrl(resolved).Await()
			for _, name := range sortedKeys(p.Headers) {
				request.Headers().Add(name, p.Headers[name])
			}
			response := request.Close().Await()
			if response.StatusCode() != io.HttpStatusOk {
				// The network may be only temporarily unavailable, or the
				// file will be added on the server later. Avoid having
				// future calls to resolve fail to check the network again.
				drainResponse(response).Await()
				panic(NewNetworkImageLoadException(response.StatusCode(), resolved))
			}
			bytes := foundation.ConsolidateHttpClientResponseBytes(response, foundation.ConsolidateHttpClientResponseBytesOptions{
				OnBytesReceived: func(cumulative int, total *int) {
					chunkEvents.Add(NewImageChunkEvent(cumulative, total))
				},
			}).Await()
			if len(bytes) == 0 {
				panic(dart.NewException("NetworkImage is an empty file: " + resolved.String()))
			}
			codec = decode(ui.ImmutableBufferFromUint8List(bytes).Await()).Await()
		})
		if threw {
			// Depending on where the exception was thrown, the image cache
			// may not have had a chance to track the key in the cache at
			// all. Schedule a microtask to give the cache a chance to add
			// the key.
			async.ScheduleMicrotask(func() { PaintingBindingInstance().ImageCache().Evict(key) })
			panic(exception)
		}
		return codec
	})
}

// drainResponse is response.drain(): a response with its own Drain method
// (Dart's Stream.drain, which fakes override) uses it; otherwise the body
// is listened to and discarded.
func drainResponse(response io.HttpClientResponse) async.Future[async.Void] {
	if d, ok := response.(interface {
		Drain() async.Future[async.Void]
	}); ok {
		return d.Drain()
	}
	return async.Then(async.StreamToList[[]byte](response), func([][]byte) async.Void { return async.Void{} })
}

func sortedKeys(m map[string]string) []string {
	keys := slices.Collect(maps.Keys(m))
	sort.Strings(keys)
	return keys
}

// Equal is ==: same URL, scale and headers.
func (p *NetworkImage) Equal(other any) bool {
	o, ok := other.(*NetworkImage)
	return ok && o.URL == p.URL && o.Scale == p.Scale && (o.Headers == nil) == (p.Headers == nil) && maps.Equal(o.Headers, p.Headers)
}

func (p *NetworkImage) String() string {
	headers := "null"
	if p.Headers != nil {
		parts := make([]string, 0, len(p.Headers))
		for _, k := range sortedKeys(p.Headers) {
			parts = append(parts, k+": "+p.Headers[k])
		}
		headers = "{" + strings.Join(parts, ", ") + "}"
	}
	return foundation.ObjectRuntimeType(p, "NetworkImage") + `("` + p.URL + `", scale: ` + dart.ToStringAsFixed(p.Scale, 1) +
		", webHtmlElementStrategy: " + p.WebHtmlElementStrategy.Name() + ", headers: " + headers + ")"
}
