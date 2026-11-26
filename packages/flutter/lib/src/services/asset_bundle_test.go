package services_test

// Ports of packages/flutter/test/services/asset_bundle_test.dart (11/12)
// and asset_manifest_test.dart (3/3) (Flutter 3.50.1).
//
// Skipped / adapted:
//   - "AssetImage.obtainKey succeeds with ImageConfiguration.empty": needs
//     the painting library (AssetImage), not ported yet.
//   - "NetworkAssetBundle control test": flutter_test's binding answers
//     every HTTP request with 400 (its HttpOverrides); Go has no such
//     override, so a local httptest server answers 400.
//   - The web-only manifest paths (AssetManifest.bin.json) are not
//     exercised: Go is never a web build.

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/services"
	"github.com/jonathanperis/flutter-go/src/async"
	"github.com/jonathanperis/flutter-go/src/dart"
)

type testAssetBundle struct {
	services.CachingAssetBundle
	loadCallCount map[string]int
}

func newTestAssetBundle() *testAssetBundle {
	b := &testAssetBundle{loadCallCount: map[string]int{}}
	b.SetSelf(b)
	return b
}

func (b *testAssetBundle) Load(key string) async.Future[[]byte] {
	return async.Async(func() []byte {
		b.loadCallCount[key]++
		switch key {
		case "AssetManifest.json":
			return []byte(`{"one": ["one"]}`)
		case "AssetManifest.bin":
			return services.NewStandardMessageCodec().EncodeMessage(map[string]any{"one": []any{}})
		case "counter":
			return []byte(dart.ToString(b.loadCallCount[key]))
		case "one":
			return []byte{49}
		}
		panic(foundation.NewFlutterError("key not found"))
	})
}

func TestCachingAssetBundle(t *testing.T) {
	runAsync(t, func() {
		bundle := newTestAssetBundle()
		assetData := bundle.Load("one").Await()
		expect(assetData[0] == 49, "data %v", assetData)
		expect(bundle.loadCallCount["one"] == 1, "count")
		assetString := bundle.LoadString("one").Await()
		expect(assetString == "1", "string %q", assetString)
		expect(bundle.loadCallCount["one"] == 2, "count")
		err := awaitError(bundle.LoadString("foo"))
		var flutterError *foundation.FlutterError
		expect(errors.As(err, &flutterError), "error %v", err)
	})
}

func stringParser(s string) func(string) async.Future[string] {
	return func(string) async.Future[string] { return async.FutureValue(s) }
}

func binaryParser(s string) func([]byte) async.Future[string] {
	return func([]byte) async.Future[string] { return async.FutureValue(s) }
}

// loadStructuredBinaryDataFuture is loadStructuredBinaryData with a parser
// returning a future (Dart's FutureOr<T>).
func loadStructuredBinaryDataFuture(b services.AssetBundle, key string, parser func([]byte) async.Future[string]) async.Future[string] {
	return async.CastFuture[string](b.LoadStructuredBinaryDataAny(key, func(data []byte) any {
		return async.EraseFuture(parser(data))
	}))
}

func TestCachingAssetBundleCaches(t *testing.T) {
	runAsync(t, func() {
		bundle := newTestAssetBundle()
		first := bundle.LoadString("counter").Await()
		second := bundle.LoadString("counter").Await()
		expect(first == "1" && second == "1", "%q %q", first, second)

		first = services.AssetBundleLoadStructuredData(bundle, "AssetManifest.json", stringParser("one")).Await()
		second = services.AssetBundleLoadStructuredData(bundle, "AssetManifest.json", stringParser("two")).Await()
		expect(first == "one" && second == "one", "%q %q", first, second)

		first = loadStructuredBinaryDataFuture(bundle, "AssetManifest.bin", binaryParser("one")).Await()
		second = loadStructuredBinaryDataFuture(bundle, "AssetManifest.bin", binaryParser("two")).Await()
		expect(first == "one" && second == "one", "%q %q", first, second)
	})
}

func TestCachingAssetBundleClear(t *testing.T) {
	runAsync(t, func() {
		bundle := newTestAssetBundle()
		bundle.LoadString("counter").Await()
		bundle.Clear()
		second := bundle.LoadString("counter").Await()
		expect(second == "2", "%q", second)

		services.AssetBundleLoadStructuredData(bundle, "AssetManifest.json", stringParser("one")).Await()
		bundle.Clear()
		second = services.AssetBundleLoadStructuredData(bundle, "AssetManifest.json", stringParser("two")).Await()
		expect(second == "two", "%q", second)

		loadStructuredBinaryDataFuture(bundle, "AssetManifest.bin", binaryParser("one")).Await()
		bundle.Clear()
		second = loadStructuredBinaryDataFuture(bundle, "AssetManifest.bin", binaryParser("two")).Await()
		expect(second == "two", "%q", second)
	})
}

func TestCachingAssetBundleEvict(t *testing.T) {
	runAsync(t, func() {
		bundle := newTestAssetBundle()
		bundle.LoadString("counter").Await()
		bundle.Evict("counter")
		second := bundle.LoadString("counter").Await()
		expect(second == "2", "%q", second)

		services.AssetBundleLoadStructuredData(bundle, "AssetManifest.json", stringParser("one")).Await()
		bundle.Evict("AssetManifest.json")
		second = services.AssetBundleLoadStructuredData(bundle, "AssetManifest.json", stringParser("two")).Await()
		expect(second == "two", "%q", second)

		loadStructuredBinaryDataFuture(bundle, "AssetManifest.bin", binaryParser("one")).Await()
		bundle.Evict("AssetManifest.bin")
		second = loadStructuredBinaryDataFuture(bundle, "AssetManifest.bin", binaryParser("two")).Await()
		expect(second == "two", "%q", second)

		// Evicting one key keeps the others.
		bundle.Evict("unrelated")
		third := services.AssetBundleLoadStructuredData(bundle, "AssetManifest.json", stringParser("three")).Await()
		expect(third == "two", "%q", third)
	})
}

func TestCachingAssetBundleStructuredDataSynchronousAfterFirst(t *testing.T) {
	runAsync(t, func() {
		bundle := newTestAssetBundle()
		services.AssetBundleLoadStructuredData(bundle, "one", func(string) async.Future[int] {
			return foundation.NewSynchronousFuture(1)
		}).Await()
		data := services.AssetBundleLoadStructuredData(bundle, "one", func(string) async.Future[int] {
			return foundation.NewSynchronousFuture(2)
		})
		// Dart: isA<SynchronousFuture<int>>; Go's typed view of it calls back
		// synchronously.
		isSync := false
		async.OnComplete(data, func(int) { isSync = true }, nil)
		expect(isSync, "not synchronous: %T", data)
		expect(data.Await() == 1, "value")
	})
}

func TestCachingAssetBundleStructuredBinaryDataSynchronousAfterFirst(t *testing.T) {
	runAsync(t, func() {
		bundle := newTestAssetBundle()
		services.AssetBundleLoadStructuredBinaryData(bundle, "one", func([]byte) int { return 1 }).Await()
		data := services.AssetBundleLoadStructuredBinaryData(bundle, "one", func([]byte) int { return 2 })
		// Dart: isA<SynchronousFuture<int>>; Go's typed view of it calls back
		// synchronously.
		isSync := false
		async.OnComplete(data, func(int) { isSync = true }, nil)
		expect(isSync, "not synchronous: %T", data)
		expect(data.Await() == 1, "value")
	})
}

func TestCachingAssetBundleStructuredDataErrors(t *testing.T) {
	runAsync(t, func() {
		bundle := newTestAssetBundle()
		err := awaitError(services.AssetBundleLoadStructuredData(bundle, "AssetManifest.json", func(string) async.Future[string] {
			return async.FutureError[string](errors.New("what do they say?"))
		}))
		expect(err != nil && strings.Contains(err.Error(), "what do they say?"), "error %v", err)
		// The failure isn't cached.
		again := services.AssetBundleLoadStructuredData(bundle, "AssetManifest.json", stringParser("ok")).Await()
		expect(again == "ok", "%q", again)
	})
}

func TestCachingAssetBundleStructuredBinaryDataErrors(t *testing.T) {
	runAsync(t, func() {
		bundle := newTestAssetBundle()
		err := awaitError(loadStructuredBinaryDataFuture(bundle, "AssetManifest.bin", func([]byte) async.Future[string] {
			return async.FutureError[string](errors.New("buy more crystals"))
		}))
		expect(err != nil && strings.Contains(err.Error(), "buy more crystals"), "error %v", err)
		again := loadStructuredBinaryDataFuture(bundle, "AssetManifest.bin", binaryParser("ok")).Await()
		expect(again == "ok", "%q", again)
	})
}

func TestNetworkAssetBundle(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()
	uri, _ := url.Parse(server.URL + "/path")
	runAsyncWaiting(t, func() {
		bundle := services.NewNetworkAssetBundle(uri)
		err := awaitError(bundle.Load("key"))
		var flutterError *foundation.FlutterError
		expect(errors.As(err, &flutterError), "error %v", err)
		expect(len(flutterError.Diagnostics) == 2, "diagnostics %v", flutterError.Diagnostics)
		_, isInt := flutterError.Diagnostics[1].(*foundation.IntProperty)
		expect(isInt, "last diagnostic %T", flutterError.Diagnostics[1])
		deep := flutterError.ToStringDeep()
		expect(deep == "FlutterError\n   Unable to load asset: \"key\".\n   HTTP status code: 400\n", "%q", deep)
	})
}

func TestNetworkAssetBundleToString(t *testing.T) {
	uri, _ := url.Parse("http://example.org/path")
	bundle := services.NewNetworkAssetBundle(uri)
	if got, want := bundle.String(), "NetworkAssetBundle#"+foundation.ShortHash(bundle)+"(http://example.org/path)"; got != want {
		t.Fatalf("%q, want %q", got, want)
	}
}

func TestRootBundleLoadMissingAsset(t *testing.T) {
	ensureInitialized()
	runAsync(t, func() {
		err := awaitError(services.RootBundle.Load("not-exists"))
		var flutterError *foundation.FlutterError
		expect(errors.As(err, &flutterError), "error %v", err)
		expect(flutterError.Message() == "Unable to load asset: \"not-exists\".\nThe asset does not exist or has empty data.",
			"message %q", flutterError.Message())
	})
}

func TestRootBundleLoadsThroughFlutterAssets(t *testing.T) {
	ensureInitialized()
	var keys []string
	messenger().SetMockMessageHandler("flutter/assets", func(message []byte) async.Future[[]byte] {
		return async.Async(func() []byte {
			keys = append(keys, string(message))
			return []byte("data")
		})
	})
	defer messenger().SetMockMessageHandler("flutter/assets", nil)
	runAsync(t, func() {
		data := services.RootBundle.Load("assets/sub dir/a#b?.png").Await()
		expect(string(data) == "data", "data %q", data)
		expect(len(keys) == 1 && keys[0] == "assets/sub%20dir/a%23b%3F.png", "keys %q", keys)
	})
}

func TestLoadStructuredBinaryDataLoadsBytes(t *testing.T) {
	runAsync(t, func() {
		bundle := newTestAssetBundle()
		assetManifest := services.AssetBundleLoadStructuredBinaryData(bundle, "AssetManifest.bin", func(data []byte) any {
			return services.NewStandardMessageCodec().DecodeMessage(data)
		}).Await()
		m := assetManifest.(*dart.LinkedMap)
		expect(m.Len() == 1, "manifest %v", m)
		one, ok := m.Lookup("one").([]any)
		expect(ok && len(one) == 0, "manifest %v", m)
	})
}

// manifestBundle is asset_manifest_test's TestAssetBundle: an AssetBundle
// (not caching) serving a binary manifest.
type manifestBundle struct {
	services.AssetBundleBase
	manifest any
}

func newManifestBundle(manifest any) *manifestBundle {
	b := &manifestBundle{manifest: manifest}
	b.SetSelf(b)
	return b
}

func (b *manifestBundle) Load(key string) async.Future[[]byte] {
	return async.Async(func() []byte {
		if key == "AssetManifest.bin" {
			return services.NewStandardMessageCodec().EncodeMessage(b.manifest)
		}
		panic(errors.New("Invalid argument(s): Unexpected key: " + key))
	})
}

func manifestEntry(key string, variants ...[]dart.MapEntry[string, any]) dart.MapEntry[string, any] {
	list := make([]any, len(variants))
	for i, v := range variants {
		list[i] = v
	}
	return dart.MapEntry[string, any]{Key: key, Value: list}
}

func variant(asset string, dpr ...float64) []dart.MapEntry[string, any] {
	v := []dart.MapEntry[string, any]{{Key: "asset", Value: asset}}
	if len(dpr) > 0 {
		v = append(v, dart.MapEntry[string, any]{Key: "dpr", Value: dpr[0]})
	}
	return v
}

func TestAssetManifestLoadFromBundle(t *testing.T) {
	runAsync(t, func() {
		manifest := services.AssetManifestLoadFromAssetBundle(newManifestBundle([]dart.MapEntry[string, any]{
			manifestEntry("assets/foo.png", variant("assets/foo.png"), variant("assets/2x/foo.png", 2.0)),
			manifestEntry("assets/bar.png", variant("assets/bar.png")),
		})).Await()
		assets := manifest.ListAssets()
		expect(strings.Join(assets, ",") == "assets/foo.png,assets/bar.png", "assets %v", assets)

		fooVariants := manifest.GetAssetVariants("assets/foo.png")
		expect(len(fooVariants) == 2, "variants %v", fooVariants)
		expect(fooVariants[0].Key == "assets/foo.png" && fooVariants[0].TargetDevicePixelRatio == nil && fooVariants[0].Main, "%v", fooVariants[0])
		expect(fooVariants[1].Key == "assets/2x/foo.png" && *fooVariants[1].TargetDevicePixelRatio == 2.0 && !fooVariants[1].Main, "%v", fooVariants[1])

		barVariants := manifest.GetAssetVariants("assets/bar.png")
		expect(len(barVariants) == 1 && barVariants[0].Key == "assets/bar.png" && barVariants[0].TargetDevicePixelRatio == nil && barVariants[0].Main, "%v", barVariants)

		// Variants read move to the end of listAssets, in the order read.
		assets = manifest.ListAssets()
		expect(strings.Join(assets, ",") == "assets/foo.png,assets/bar.png", "assets %v", assets)
	})
}

func TestAssetManifestGetAssetVariantsUnknownKey(t *testing.T) {
	runAsync(t, func() {
		manifest := services.AssetManifestLoadFromAssetBundle(newManifestBundle([]dart.MapEntry[string, any]{
			manifestEntry("assets/foo.png", variant("assets/foo.png")),
		})).Await()
		expect(manifest.GetAssetVariants("invalid asset key") == nil, "variants")
	})
}

func TestAssetManifestContentHashedMain(t *testing.T) {
	runAsync(t, func() {
		manifest := services.AssetManifestLoadFromAssetBundle(newManifestBundle([]dart.MapEntry[string, any]{
			manifestEntry("assets/foo.png", variant("assets/foo.9f64a747.png"), variant("assets/2.0x/foo.74f81fe1.png", 2.0)),
			manifestEntry("assets/sub dir/space image.png", variant("assets/sub dir/space image.1a2b3c4d.png")),
		})).Await()
		fooVariants := manifest.GetAssetVariants("assets/foo.png")
		expect(len(fooVariants) == 2, "variants %v", fooVariants)
		expect(fooVariants[0].Key == "assets/foo.9f64a747.png" && fooVariants[0].TargetDevicePixelRatio == nil && fooVariants[0].Main, "%v", fooVariants[0])
		expect(fooVariants[1].Key == "assets/2.0x/foo.74f81fe1.png" && *fooVariants[1].TargetDevicePixelRatio == 2.0 && !fooVariants[1].Main, "%v", fooVariants[1])
		spaced := manifest.GetAssetVariants("assets/sub dir/space image.png")
		expect(len(spaced) == 1 && spaced[0].Key == "assets/sub dir/space image.1a2b3c4d.png" && spaced[0].Main, "%v", spaced)
	})
}
