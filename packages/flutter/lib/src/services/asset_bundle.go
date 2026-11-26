package services

// Port of asset_bundle.dart (Flutter 3.50.1).
//
// AssetBundle's generic methods are interface methods with T erased
// (LoadStructuredDataAny, LoadStructuredBinaryDataAny: what a bundle
// overrides) plus the generic functions AssetBundleLoadStructuredData[T]
// and AssetBundleLoadStructuredBinaryData[T] (what callers use; Go methods
// can't take type parameters).

import (
	"io"
	"net/http"
	"net/url"
	"os"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	"github.com/jonathanperis/flutter-go/src/async"
	"github.com/jonathanperis/flutter-go/src/dart"
)

// AssetBundle is a collection of resources used by the application (an
// abstract class: implement Load, embed AssetBundleBase for the rest and
// call SetSelf).
type AssetBundle interface {
	// Load retrieves a binary resource from the asset bundle as a data
	// stream; the future fails with a FlutterError if the asset is not
	// found.
	Load(key string) async.Future[[]byte]
	// LoadBuffer retrieves a binary resource as an ImmutableBuffer.
	LoadBuffer(key string) async.Future[*ui.ImmutableBuffer]
	// LoadString retrieves a string (UTF-8) from the asset bundle; Cache
	// (default true) lets the bundle cache the result.
	LoadString(key string, opts ...LoadStringOptions) async.Future[string]
	// LoadStructuredDataAny is loadStructuredData<T> with T erased: it
	// retrieves a string and returns parser's result (a Future[any], see
	// AssetBundleLoadStructuredData).
	LoadStructuredDataAny(key string, parser func(value string) async.Future[any]) async.Future[any]
	// LoadStructuredBinaryDataAny is loadStructuredBinaryData<T> with T
	// erased: parser's result is a value or a Future[any] (Dart's
	// FutureOr<T>).
	LoadStructuredBinaryDataAny(key string, parser func(data []byte) any) async.Future[any]
	// Evict removes key and its associated data from the cache, if any.
	Evict(key string)
	// Clear removes all entries from the bundle's cache.
	Clear()
	String() string
}

// LoadStringOptions are AssetBundle.loadString's named parameters.
type LoadStringOptions struct {
	// NoCache is Dart's `cache: false` (the default caches).
	NoCache bool
}

func loadStringCache(opts []LoadStringOptions) bool {
	return len(opts) == 0 || !opts[0].NoCache
}

// AssetBundleLoadStructuredData is bundle.loadStructuredData<T>(key,
// parser): it retrieves a string from the asset bundle, parses it with
// parser and returns the result.
func AssetBundleLoadStructuredData[T any](bundle AssetBundle, key string, parser func(value string) async.Future[T]) async.Future[T] {
	return async.CastFuture[T](bundle.LoadStructuredDataAny(key, func(value string) async.Future[any] {
		return async.EraseFuture(parser(value))
	}))
}

// AssetBundleLoadStructuredBinaryData is
// bundle.loadStructuredBinaryData<T>(key, parser) for a synchronous
// parser: it retrieves binary data and returns parser's result.
func AssetBundleLoadStructuredBinaryData[T any](bundle AssetBundle, key string, parser func(data []byte) T) async.Future[T] {
	return async.CastFuture[T](bundle.LoadStructuredBinaryDataAny(key, func(data []byte) any { return parser(data) }))
}

// AssetBundleBase is AssetBundle's state and default methods, to embed in
// a bundle (call SetSelf with it).
type AssetBundleBase struct {
	self AssetBundle
}

// SetSelf points the base at the bundle embedding it.
func (b *AssetBundleBase) SetSelf(self AssetBundle) { b.self = self }

func (b *AssetBundleBase) outer() AssetBundle {
	if b.self == nil {
		panic("services: AssetBundleBase used before SetSelf; call SetSelf with the bundle when creating it")
	}
	return b.self
}

func (b *AssetBundleBase) LoadBuffer(key string) async.Future[*ui.ImmutableBuffer] {
	return async.Async(func() *ui.ImmutableBuffer {
		data := b.outer().Load(key).Await()
		return ui.ImmutableBufferFromUint8List(data).Await()
	})
}

func utf8decode(data []byte) string { return dart.UTF8Decode(data) }

func (b *AssetBundleBase) LoadString(key string, opts ...LoadStringOptions) async.Future[string] {
	return async.Async(func() string {
		data := b.outer().Load(key).Await()
		// 50 KB of data should take 2-3 ms to parse on a Moto G4, and about
		// 400 μs on a Pixel 4.
		if len(data) < 50*1024 || foundation.KIsWeb {
			return dart.UTF8Decode(data)
		}
		// For strings larger than 50 KB, run the computation in an isolate
		// to avoid causing main thread jank.
		label := `UTF8 decode for "` + key + `"`
		return foundation.Compute(utf8decode, data, foundation.ComputeOptions{DebugLabel: &label}).Await()
	})
}

func (b *AssetBundleBase) LoadStructuredDataAny(key string, parser func(value string) async.Future[any]) async.Future[any] {
	return async.Async(func() any {
		return parser(b.outer().LoadString(key).Await()).Await()
	})
}

func (b *AssetBundleBase) LoadStructuredBinaryDataAny(key string, parser func(data []byte) any) async.Future[any] {
	return async.Async(func() any {
		result := parser(b.outer().Load(key).Await())
		if f, ok := result.(async.Future[any]); ok {
			return f.Await()
		}
		return result
	})
}

func (b *AssetBundleBase) Evict(key string) {}

func (b *AssetBundleBase) Clear() {}

func (b *AssetBundleBase) String() string { return foundation.DescribeIdentity(b.outer()) + "()" }

// NetworkAssetBundle is an AssetBundle that loads resources over the
// network (Go's net/http stands for dart:io's HttpClient; the request runs
// on a goroutine and its result is delivered through the event loop).
type NetworkAssetBundle struct {
	AssetBundleBase
	baseURL *url.URL
	client  *http.Client
}

// NewNetworkAssetBundle is NetworkAssetBundle(baseUrl): keys are resolved
// relative to baseURL.
func NewNetworkAssetBundle(baseURL *url.URL) *NetworkAssetBundle {
	b := &NetworkAssetBundle{baseURL: baseURL, client: &http.Client{}}
	b.SetSelf(b)
	return b
}

func (b *NetworkAssetBundle) urlFromKey(key string) *url.URL {
	ref, err := url.Parse(key)
	if err != nil {
		panic(dart.NewFormatException(err.Error(), key))
	}
	return b.baseURL.ResolveReference(ref)
}

func (b *NetworkAssetBundle) Load(key string) async.Future[[]byte] {
	return async.Async(func() []byte {
		target := b.urlFromKey(key)
		loop := async.Current()
		completer := async.NewCompleter[[]byte]()
		var status int
		go func() {
			var body []byte
			resp, err := b.client.Get(target.String())
			if err == nil {
				status = resp.StatusCode
				body, err = io.ReadAll(resp.Body)
				resp.Body.Close()
			}
			loop.Post(func() {
				if err != nil {
					completer.CompleteError(err)
					return
				}
				completer.Complete(body)
			})
		}()
		bytes := completer.Future().Await()
		if status != http.StatusOK {
			panic(foundation.NewFlutterErrorFromParts([]foundation.DiagnosticsNode{
				errorSummaryWithKey(key),
				foundation.NewIntProperty("HTTP status code", status),
			}))
		}
		return bytes
	})
}

func (b *NetworkAssetBundle) String() string {
	return foundation.DescribeIdentity(b) + "(" + b.baseURL.String() + ")"
}

// CachingAssetBundle is an AssetBundle that permanently caches string and
// structured resources that have been fetched (an abstract class:
// implement Load, embed it and call SetSelf).
type CachingAssetBundle struct {
	AssetBundleBase
	stringCache               map[string]async.Future[string]
	structuredDataCache       map[string]async.Future[any]
	structuredBinaryDataCache map[string]async.Future[any]
}

func (b *CachingAssetBundle) caches() {
	if b.stringCache == nil {
		b.stringCache = map[string]async.Future[string]{}
		b.structuredDataCache = map[string]async.Future[any]{}
		b.structuredBinaryDataCache = map[string]async.Future[any]{}
	}
}

func (b *CachingAssetBundle) LoadString(key string, opts ...LoadStringOptions) async.Future[string] {
	b.caches()
	if loadStringCache(opts) {
		if f, ok := b.stringCache[key]; ok {
			return f
		}
		f := b.AssetBundleBase.LoadString(key)
		b.stringCache[key] = f
		return f
	}
	return b.AssetBundleBase.LoadString(key)
}

// cachedStructured is the body shared by loadStructuredData and
// loadStructuredBinaryData: start (the load then the parser) completes
// either synchronously (a SynchronousFuture chain, as in flutter_test) or
// later; the result is cached as a SynchronousFuture.
func cachedStructured(cache map[string]async.Future[any], key string, start func() async.Future[any]) async.Future[any] {
	if f, ok := cache[key]; ok {
		return f
	}
	var completer *async.Completer[any]     // For async flow.
	var synchronousResult async.Future[any] // For sync flow.
	async.OnComplete(start(),
		func(value any) {
			synchronousResult = foundation.NewSynchronousFuture(value)
			cache[key] = synchronousResult
			// We already returned from the function, which means we are in
			// the asynchronous mode. Pass the value to the completer. The
			// completer's future is what we returned.
			if completer != nil {
				completer.Complete(value)
			}
		},
		func(err error) {
			if completer == nil {
				panic(dart.NewAssertionError("unexpected synchronous failure"))
			}
			// Either loading or parsing failed. We must report the error
			// back to the caller and anyone waiting on this call. We clear
			// the cache for this key, however, because we want future
			// attempts to try again.
			delete(cache, key)
			completer.CompleteError(err)
		})
	if synchronousResult != nil {
		// The above code ran synchronously. We can synchronously return
		// the result.
		return synchronousResult
	}
	// The code above hasn't yet run its "then" handler yet. Let's prepare a
	// completer for it to use when it does run.
	completer = async.NewCompleter[any]()
	cache[key] = completer.Future()
	return completer.Future()
}

func (b *CachingAssetBundle) LoadStructuredDataAny(key string, parser func(value string) async.Future[any]) async.Future[any] {
	b.caches()
	return cachedStructured(b.structuredDataCache, key, func() async.Future[any] {
		// loadString can return a SynchronousFuture in certain cases, like
		// in the flutter_test framework.
		return b.outer().LoadString(key, LoadStringOptions{NoCache: true}).ThenAny(func(value string) any {
			return parser(value)
		}, nil)
	})
}

func (b *CachingAssetBundle) LoadStructuredBinaryDataAny(key string, parser func(data []byte) any) async.Future[any] {
	b.caches()
	return cachedStructured(b.structuredBinaryDataCache, key, func() async.Future[any] {
		// load can return a SynchronousFuture in certain cases, like in the
		// flutter_test framework.
		return b.outer().Load(key).ThenAny(func(data []byte) any { return parser(data) }, nil)
	})
}

func (b *CachingAssetBundle) Evict(key string) {
	b.caches()
	delete(b.stringCache, key)
	delete(b.structuredDataCache, key)
	delete(b.structuredBinaryDataCache, key)
}

func (b *CachingAssetBundle) Clear() {
	b.stringCache = nil
	b.structuredDataCache = nil
	b.structuredBinaryDataCache = nil
	b.caches()
}

func (b *CachingAssetBundle) LoadBuffer(key string) async.Future[*ui.ImmutableBuffer] {
	return async.Async(func() *ui.ImmutableBuffer {
		data := b.outer().Load(key).Await()
		return ui.ImmutableBufferFromUint8List(data).Await()
	})
}

// PlatformAssetBundle is an AssetBundle that loads resources from the
// engine's asset manager (the flutter/assets channel; flutter-go's
// embedders serve it from ui.EngineAssetLoader).
type PlatformAssetBundle struct {
	CachingAssetBundle
}

// NewPlatformAssetBundle is PlatformAssetBundle().
func NewPlatformAssetBundle() *PlatformAssetBundle {
	b := &PlatformAssetBundle{}
	b.SetSelf(b)
	return b
}

func assetNotFound(key string) *foundation.FlutterError {
	return foundation.NewFlutterErrorFromParts([]foundation.DiagnosticsNode{
		errorSummaryWithKey(key),
		foundation.NewErrorDescription("The asset does not exist or has empty data."),
	})
}

func (b *PlatformAssetBundle) Load(key string) async.Future[[]byte] {
	encoded := dart.UTF8Encode(dart.URIPath(dart.URIEncodeFull(key)))
	future := ServicesBindingInstance().DefaultBinaryMessenger().Send("flutter/assets", encoded)
	if future == nil {
		panic(assetNotFound(key))
	}
	return async.Then(future, func(asset []byte) []byte {
		if asset == nil {
			panic(assetNotFound(key))
		}
		return asset
	})
}

func (b *PlatformAssetBundle) LoadBuffer(key string) async.Future[*ui.ImmutableBuffer] {
	return async.Async(func() *ui.ImmutableBuffer {
		debugUsePlatformChannel := false
		if foundation.KDebugMode {
			// Override how assets are loaded in tests so that the old loader
			// behavior that allows tests to load assets from the current
			// package using the package prefix.
			if _, ok := os.LookupEnv("UNIT_TEST_ASSETS"); ok {
				debugUsePlatformChannel = true
			}
		}
		if debugUsePlatformChannel {
			bytes := b.Load(key).Await()
			return ui.ImmutableBufferFromUint8List(bytes).Await()
		}
		buffer, err := awaitCatching(ui.ImmutableBufferFromAsset(key))
		if err != nil {
			if _, ok := err.(dart.Exception); ok {
				panic(foundation.NewFlutterErrorFromParts([]foundation.DiagnosticsNode{
					errorSummaryWithKey(key),
					foundation.NewErrorDescription(err.Error()),
				}))
			}
			panic(err)
		}
		return buffer
	})
}

// awaitCatching awaits f and returns its error instead of panicking.
func awaitCatching[T any](f async.Future[T]) (value T, err error) {
	defer func() {
		if r := recover(); r != nil {
			e, ok := r.(error)
			if !ok {
				panic(r)
			}
			err = e
		}
	}()
	return f.Await(), nil
}

func errorSummaryWithKey(key string) *foundation.ErrorSummary {
	return foundation.NewErrorSummary(`Unable to load asset: "` + key + `".`)
}

// RootBundle is rootBundle: the AssetBundle from which this application
// was loaded (a PlatformAssetBundle).
var RootBundle AssetBundle = NewPlatformAssetBundle()
