package services

// Port of asset_bundle.dart (Flutter 3.50.1): AssetBundle,
// CachingAssetBundle, PlatformAssetBundle and rootBundle.
// NetworkAssetBundle is not ported (dart:io's HttpClient is not; see
// src/io).
//
// AssetBundle is an abstract class user code extends (tests override
// load), so it is an interface plus the embeddable AssetBundleBase (self
// pattern, API_MAPPING "Mixins and overridable methods"). Dart's generic
// methods loadStructuredData<T>/loadStructuredBinaryData<T> are interface
// methods with T erased to any, and the generic functions
// AssetBundleLoadStructuredData[T]/AssetBundleLoadStructuredBinaryData[T]
// restore T.

import (
	"os"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	"github.com/jonathanperis/flutter-go/src/async"
	"github.com/jonathanperis/flutter-go/src/dart"
)

// AssetBundleLoadStringOptions are loadString's named parameters (Cache
// nil: Dart's default true).
type AssetBundleLoadStringOptions struct {
	Cache *bool
}

func (o AssetBundleLoadStringOptions) cache() bool { return o.Cache == nil || *o.Cache }

// AssetBundle is a collection of resources used by the application
// (images, strings, structured data), looked up by key.
type AssetBundle interface {
	// Load retrieves a binary resource from the asset bundle; the future
	// fails (with a FlutterError) when the asset does not exist.
	Load(key string) async.Future[[]byte]
	// LoadBuffer retrieves a binary resource as an ImmutableBuffer.
	LoadBuffer(key string) async.Future[*ui.ImmutableBuffer]
	// LoadString retrieves a string from the asset bundle (UTF-8),
	// caching it unless Cache is false.
	LoadString(key string, opts ...AssetBundleLoadStringOptions) async.Future[string]
	// LoadStructuredData retrieves a string and returns the result of
	// parsing it (Dart's loadStructuredData<T> with T erased; see
	// AssetBundleLoadStructuredData).
	LoadStructuredData(key string, parser func(value string) async.Future[any]) async.Future[any]
	// LoadStructuredBinaryData retrieves binary data and returns the result
	// of parsing it; parser returns a value or a Future[any] (Dart's
	// FutureOr<T>).
	LoadStructuredBinaryData(key string, parser func(data []byte) any) async.Future[any]
	// Evict removes the given key from the bundle's caches, if any.
	Evict(key string)
	// Clear empties the bundle's caches, if any.
	Clear()
	String() string
}

// AssetBundleLoadStructuredData is bundle.loadStructuredData<T>(key, parser).
func AssetBundleLoadStructuredData[T any](bundle AssetBundle, key string, parser func(value string) async.Future[T]) async.Future[T] {
	return async.Then(bundle.LoadStructuredData(key, func(value string) async.Future[any] {
		return async.EraseFuture(parser(value))
	}), func(v any) T { return castTo[T](v) })
}

// AssetBundleLoadStructuredBinaryData is
// bundle.loadStructuredBinaryData<T>(key, parser).
func AssetBundleLoadStructuredBinaryData[T any](bundle AssetBundle, key string, parser func(data []byte) T) async.Future[T] {
	f := bundle.LoadStructuredBinaryData(key, func(data []byte) any { return parser(data) })
	if s, ok := f.(*foundation.SynchronousFuture[any]); ok {
		var v any
		s.ThenAny(func(x any) any { v = x; return nil }, nil)
		return foundation.NewSynchronousFuture(castTo[T](v))
	}
	return async.Then(f, func(v any) T { return castTo[T](v) })
}

func castTo[T any](v any) T {
	if v == nil {
		var zero T
		return zero
	}
	return v.(T)
}

// AssetBundleBase is AssetBundle's default methods, to embed in a bundle
// (which implements Load and calls SetSelf).
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

// LoadBuffer loads the data with Load and copies it into a buffer.
func (b *AssetBundleBase) LoadBuffer(key string) async.Future[*ui.ImmutableBuffer] {
	self := b.outer()
	return async.Async(func() *ui.ImmutableBuffer {
		data := self.Load(key).Await()
		return ui.ImmutableBufferFromUint8List(data).Await()
	})
}

// LoadString decodes the data Load returns as UTF-8 (on another goroutine,
// through foundation.Compute, when it is 50 KB or more, as Dart does in an
// isolate).
func (b *AssetBundleBase) LoadString(key string, opts ...AssetBundleLoadStringOptions) async.Future[string] {
	self := b.outer()
	return async.Async(func() string {
		data := self.Load(key).Await()
		if len(data) < 50*1024 || foundation.KIsWeb {
			return dart.UTF8Decode(data)
		}
		label := `UTF8 decode for "` + key + `"`
		return foundation.Compute(dart.UTF8Decode, data, foundation.ComputeOptions{DebugLabel: &label}).Await()
	})
}

// LoadStructuredData is parser(await loadString(key)).
func (b *AssetBundleBase) LoadStructuredData(key string, parser func(value string) async.Future[any]) async.Future[any] {
	self := b.outer()
	return async.Async(func() any {
		return parser(self.LoadString(key).Await()).Await()
	})
}

// LoadStructuredBinaryData is parser(await load(key)).
func (b *AssetBundleBase) LoadStructuredBinaryData(key string, parser func(data []byte) any) async.Future[any] {
	self := b.outer()
	return async.Async(func() any {
		v := parser(self.Load(key).Await())
		if f, ok := v.(async.Future[any]); ok {
			return f.Await()
		}
		return v
	})
}

// Evict does nothing (the bundle has no cache).
func (b *AssetBundleBase) Evict(key string) {}

// Clear does nothing (the bundle has no cache).
func (b *AssetBundleBase) Clear() {}

func (b *AssetBundleBase) String() string { return foundation.DescribeIdentity(b.outer()) + "()" }

// CachingAssetBundle is an AssetBundle that permanently caches string and
// structured resources loaded from it (abstract: embed it, implement Load,
// call SetSelf).
type CachingAssetBundle struct {
	AssetBundleBase

	stringCache               map[string]async.Future[string]
	structuredDataCache       map[string]async.Future[any]
	structuredBinaryDataCache map[string]async.Future[any]
}

func (b *CachingAssetBundle) init() {
	if b.stringCache == nil {
		b.stringCache = map[string]async.Future[string]{}
		b.structuredDataCache = map[string]async.Future[any]{}
		b.structuredBinaryDataCache = map[string]async.Future[any]{}
	}
}

// LoadString caches the result unless Cache is false.
func (b *CachingAssetBundle) LoadString(key string, opts ...AssetBundleLoadStringOptions) async.Future[string] {
	b.init()
	var o AssetBundleLoadStringOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	if o.cache() {
		if f, ok := b.stringCache[key]; ok {
			return f
		}
		f := b.AssetBundleBase.LoadString(key)
		b.stringCache[key] = f
		return f
	}
	return b.AssetBundleBase.LoadString(key)
}

// LoadStructuredData caches the parsed result; when the parse completes
// synchronously the cached (and returned) future is a SynchronousFuture.
func (b *CachingAssetBundle) LoadStructuredData(key string, parser func(value string) async.Future[any]) async.Future[any] {
	b.init()
	if f, ok := b.structuredDataCache[key]; ok {
		return f
	}
	f := async.ThenFuture(b.outer().LoadString(key, AssetBundleLoadStringOptions{Cache: new(bool)}), parser)
	return b.cacheStructured(b.structuredDataCache, key, f)
}

// LoadStructuredBinaryData caches the parsed result like
// LoadStructuredData.
func (b *CachingAssetBundle) LoadStructuredBinaryData(key string, parser func(data []byte) any) async.Future[any] {
	b.init()
	if f, ok := b.structuredBinaryDataCache[key]; ok {
		return f
	}
	loaded := b.outer().Load(key)
	f := loaded.ThenAny(func(d []byte) any { return parser(d) }, nil)
	return b.cacheStructured(b.structuredBinaryDataCache, key, f)
}

func (b *CachingAssetBundle) cacheStructured(cache map[string]async.Future[any], key string, parsed async.Future[any]) async.Future[any] {
	var completer *async.Completer[any]
	var synchronousResult async.Future[any]
	async.OnComplete(parsed, func(value any) {
		synchronousResult = foundation.NewSynchronousFuture(value)
		cache[key] = synchronousResult
		if completer != nil {
			completer.Complete(value)
		}
	}, func(err error) {
		if completer == nil {
			panic(dart.NewAssertionError("unexpected synchronous failure"))
		}
		delete(cache, key)
		completer.CompleteError(err)
	})
	if synchronousResult != nil {
		return synchronousResult
	}
	completer = async.NewCompleter[any]()
	cache[key] = completer.Future()
	return completer.Future()
}

// Evict removes key from the caches.
func (b *CachingAssetBundle) Evict(key string) {
	b.init()
	delete(b.stringCache, key)
	delete(b.structuredDataCache, key)
	delete(b.structuredBinaryDataCache, key)
}

// Clear empties the caches.
func (b *CachingAssetBundle) Clear() {
	b.stringCache, b.structuredDataCache, b.structuredBinaryDataCache = nil, nil, nil
	b.init()
}

// LoadBuffer loads the data with Load and copies it into a buffer.
func (b *CachingAssetBundle) LoadBuffer(key string) async.Future[*ui.ImmutableBuffer] {
	return b.AssetBundleBase.LoadBuffer(key)
}

// PlatformAssetBundle is an AssetBundle that loads resources from the
// application's asset bundle: Load asks the platform on the flutter/assets
// channel, LoadBuffer reads the asset through dart:ui
// (ui.ImmutableBufferFromAsset, the embedder's ui.EngineAssetLoader).
type PlatformAssetBundle struct {
	CachingAssetBundle
}

// NewPlatformAssetBundle is PlatformAssetBundle().
func NewPlatformAssetBundle() *PlatformAssetBundle {
	b := &PlatformAssetBundle{}
	b.SetSelf(b)
	return b
}

func errorSummaryWithKey(key string) foundation.DiagnosticsNode {
	return foundation.NewErrorSummary(`Unable to load asset: "` + key + `".`)
}

func assetDoesNotExist(key string) *foundation.FlutterError {
	return foundation.NewFlutterErrorFromParts([]foundation.DiagnosticsNode{
		errorSummaryWithKey(key),
		foundation.NewErrorDescription("The asset does not exist or has empty data."),
	})
}

// Load sends the (URI-encoded) key on the flutter/assets channel; a null
// reply fails with a FlutterError.
func (b *PlatformAssetBundle) Load(key string) async.Future[[]byte] {
	encoded := dart.UTF8Encode(encodeAssetPath(key))
	future := ServicesBindingInstance().DefaultBinaryMessenger().Send("flutter/assets", encoded)
	if future == nil {
		panic(assetDoesNotExist(key))
	}
	return async.Then(future, func(asset []byte) []byte {
		if asset == nil {
			panic(assetDoesNotExist(key))
		}
		return asset
	})
}

// LoadBuffer reads the asset with ui.ImmutableBufferFromAsset (or through
// Load when the UNIT_TEST_ASSETS environment variable is set, in debug
// builds, as flutter_test's asset mocking expects).
func (b *PlatformAssetBundle) LoadBuffer(key string) async.Future[*ui.ImmutableBuffer] {
	return async.Async(func() *ui.ImmutableBuffer {
		debugUsePlatformChannel := false
		if foundation.KDebugMode {
			if _, ok := os.LookupEnv("UNIT_TEST_ASSETS"); ok {
				debugUsePlatformChannel = true
			}
		}
		if debugUsePlatformChannel {
			bytes := b.Load(key).Await()
			return ui.ImmutableBufferFromUint8List(bytes).Await()
		}
		var buffer *ui.ImmutableBuffer
		var failure error
		func() {
			defer func() {
				if r := recover(); r != nil {
					if e, ok := r.(dart.Exception); ok {
						failure = e
						return
					}
					panic(r)
				}
			}()
			buffer = ui.ImmutableBufferFromAsset(key).Await()
		}()
		if failure != nil {
			panic(foundation.NewFlutterErrorFromParts([]foundation.DiagnosticsNode{
				errorSummaryWithKey(key),
				foundation.NewErrorDescription(failure.Error()),
			}))
		}
		return buffer
	})
}

// encodeAssetPath is Uri(path: Uri.encodeFull(key)).path.
func encodeAssetPath(key string) string {
	const hex = "0123456789ABCDEF"
	out := make([]byte, 0, len(key))
	for i := 0; i < len(key); i++ {
		c := key[i]
		if c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' ||
			c != 0 && c < 0x80 && containsByte("!$&'()*+,-./:;=@_~", c) {
			out = append(out, c)
		} else {
			out = append(out, '%', hex[c>>4], hex[c&15])
		}
	}
	return string(out)
}

func containsByte(s string, c byte) bool {
	for i := 0; i < len(s); i++ {
		if s[i] == c {
			return true
		}
	}
	return false
}

// RootBundle is the AssetBundle from which this application was loaded
// (rootBundle): a PlatformAssetBundle.
var RootBundle AssetBundle = NewPlatformAssetBundle()
