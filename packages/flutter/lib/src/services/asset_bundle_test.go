package services_test

// Port of packages/flutter/test/services/asset_bundle_test.dart and
// asset_manifest_test.dart (Flutter 3.50.1): 14 of 16 tests. Skipped:
// "NetworkAssetBundle control test" and "toString works as intended"
// (NetworkAssetBundle is not ported). The web manifest branches
// (AssetManifest.bin.json) are not needed: flutter-go is never a web build.

import (
	"errors"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/painting"
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
			return []byte(strconv.Itoa(b.loadCallCount[key]))
		case "one":
			return []byte{49}
		}
		panic(foundation.NewFlutterError("key not found"))
	})
}

func strFuture(s string) func(string) async.Future[any] {
	return func(string) async.Future[any] { return async.FutureValue[any](s) }
}

func binFuture(s string) func([]byte) any {
	return func([]byte) any { return async.EraseFuture(async.FutureValue(s)) }
}

func TestCachingAssetBundle(t *testing.T) {
	ensureInitialized()
	runAsync(t, func() {
		bundle := newTestAssetBundle()
		assetData := bundle.Load("one").Await()
		expect(assetData[0] == 49, "data %v", assetData)
		expect(bundle.loadCallCount["one"] == 1, "calls %d", bundle.loadCallCount["one"])
		expect(bundle.LoadString("one").Await() == "1", "string")
		expect(bundle.loadCallCount["one"] == 2, "calls %d", bundle.loadCallCount["one"])
		err := awaitError(bundle.LoadString("foo"))
		var fe *foundation.FlutterError
		expect(errors.As(err, &fe), "error %v", err)
	})
}

func TestCachingAssetBundleCachesResults(t *testing.T) {
	ensureInitialized()
	runAsync(t, func() {
		bundle := newTestAssetBundle()
		expect(bundle.LoadString("counter").Await() == "1", "first")
		expect(bundle.LoadString("counter").Await() == "1", "second")
		expect(bundle.LoadStructuredData("AssetManifest.json", strFuture("one")).Await() == "one", "data 1")
		expect(bundle.LoadStructuredData("AssetManifest.json", strFuture("two")).Await() == "one", "data 2")
		expect(bundle.LoadStructuredBinaryData("AssetManifest.bin", binFuture("one")).Await() == "one", "bin 1")
		expect(bundle.LoadStructuredBinaryData("AssetManifest.bin", binFuture("two")).Await() == "one", "bin 2")
	})
}

func TestCachingAssetBundleClearClearsAllCachedValues(t *testing.T) {
	ensureInitialized()
	runAsync(t, func() {
		bundle := newTestAssetBundle()
		bundle.LoadString("counter").Await()
		bundle.Clear()
		expect(bundle.LoadString("counter").Await() == "2", "string")
		bundle.LoadStructuredData("AssetManifest.json", strFuture("one")).Await()
		bundle.Clear()
		expect(bundle.LoadStructuredData("AssetManifest.json", strFuture("two")).Await() == "two", "data")
		bundle.LoadStructuredBinaryData("AssetManifest.bin", binFuture("one")).Await()
		bundle.Clear()
		expect(bundle.LoadStructuredBinaryData("AssetManifest.bin", binFuture("two")).Await() == "two", "bin")
	})
}

func TestCachingAssetBundleEvictEvictsAKey(t *testing.T) {
	ensureInitialized()
	runAsync(t, func() {
		bundle := newTestAssetBundle()
		bundle.LoadString("counter").Await()
		bundle.Evict("counter")
		expect(bundle.LoadString("counter").Await() == "2", "string")
		bundle.LoadStructuredData("AssetManifest.json", strFuture("one")).Await()
		bundle.Evict("AssetManifest.json")
		expect(bundle.LoadStructuredData("AssetManifest.json", strFuture("two")).Await() == "two", "data")
		bundle.LoadStructuredBinaryData("AssetManifest.bin", binFuture("one")).Await()
		bundle.Evict("AssetManifest.bin")
		expect(bundle.LoadStructuredBinaryData("AssetManifest.bin", binFuture("two")).Await() == "two", "bin")
	})
}

func TestCachingAssetBundleStructuredDataSynchronousAfterFirst(t *testing.T) {
	ensureInitialized()
	runAsync(t, func() {
		bundle := newTestAssetBundle()
		bundle.LoadStructuredData("one", func(string) async.Future[any] { return foundation.NewSynchronousFuture[any](1) }).Await()
		data := bundle.LoadStructuredData("one", func(string) async.Future[any] { return foundation.NewSynchronousFuture[any](2) })
		_, sync := data.(*foundation.SynchronousFuture[any])
		expect(sync, "not a SynchronousFuture: %T", data)
		expect(data.Await() == 1, "value")
	})
}

func TestCachingAssetBundleStructuredBinaryDataSynchronousAfterFirst(t *testing.T) {
	ensureInitialized()
	runAsync(t, func() {
		bundle := newTestAssetBundle()
		bundle.LoadStructuredBinaryData("one", func([]byte) any { return 1 }).Await()
		data := bundle.LoadStructuredBinaryData("one", func([]byte) any { return 2 })
		_, sync := data.(*foundation.SynchronousFuture[any])
		expect(sync, "not a SynchronousFuture: %T", data)
		expect(data.Await() == 1, "value")
	})
}

func TestCachingAssetBundleStructuredDataHandlesExceptions(t *testing.T) {
	ensureInitialized()
	runAsync(t, func() {
		bundle := newTestAssetBundle()
		err := awaitError(bundle.LoadStructuredData("AssetManifest.json", func(string) async.Future[any] {
			return async.FutureError[any](async.PanicError{Value: "what do they say?"})
		}))
		expect(err != nil && strings.Contains(err.Error(), "what do they say?"), "error %v", err)
	})
}

func TestCachingAssetBundleStructuredBinaryDataHandlesExceptions(t *testing.T) {
	ensureInitialized()
	runAsync(t, func() {
		bundle := newTestAssetBundle()
		err := awaitError(bundle.LoadStructuredBinaryData("AssetManifest.bin", func([]byte) any {
			return async.FutureError[any](async.PanicError{Value: "buy more crystals"})
		}))
		expect(err != nil && strings.Contains(err.Error(), "buy more crystals"), "error %v", err)
	})
}

func TestAssetImageObtainKeySucceedsWithEmptyConfiguration(t *testing.T) {
	ensureInitialized()
	runAsync(t, func() {
		assetImage := painting.NewAssetImage("one", painting.AssetImageOptions{Bundle: newTestAssetBundle()})
		key := assetImage.ObtainKey(painting.ImageConfigurationEmpty).Await().(painting.AssetBundleImageKey)
		expect(key.Name == "one" && key.Scale == 1.0, "key %v", key)
	})
}

func TestRootBundleThrowsForMissingAsset(t *testing.T) {
	ensureInitialized()
	runAsync(t, func() {
		err := awaitError(services.RootBundle.Load("not-exists"))
		var fe *foundation.FlutterError
		expect(errors.As(err, &fe), "error %v", err)
		expect(fe.Message() == "Unable to load asset: \"not-exists\".\nThe asset does not exist or has empty data.",
			"message %q", fe.Message())
	})
}

func TestLoadStructuredBinaryDataLoadsByteData(t *testing.T) {
	ensureInitialized()
	runAsync(t, func() {
		bundle := newTestAssetBundle()
		manifest := services.AssetBundleLoadStructuredBinaryData(bundle, "AssetManifest.bin", func(data []byte) map[any]any {
			return services.NewStandardMessageCodec().DecodeMessage(data).(map[any]any)
		}).Await()
		expect(len(manifest) == 1, "keys %v", manifest)
		one, ok := manifest["one"].([]any)
		expect(ok && len(one) == 0, "one %v", manifest["one"])
	})
}

// manifestBundle is asset_manifest_test's TestAssetBundle (an AssetBundle,
// not caching).
type manifestBundle struct {
	services.AssetBundleBase
	data map[string]any
}

func newManifestBundle(data map[string]any) *manifestBundle {
	b := &manifestBundle{data: data}
	b.SetSelf(b)
	return b
}

func (b *manifestBundle) Load(key string) async.Future[[]byte] {
	return async.Async(func() []byte {
		if key == "AssetManifest.bin" {
			return services.NewStandardMessageCodec().EncodeMessage(b.data)
		}
		panic(dart.NewArgumentErrorValue(nil, "", "Unexpected key: "+key))
	})
}

var binManifestData = map[string]any{
	"assets/foo.png": []any{
		map[string]any{"asset": "assets/foo.png"},
		map[string]any{"asset": "assets/2x/foo.png", "dpr": 2.0},
	},
	"assets/bar.png": []any{
		map[string]any{"asset": "assets/bar.png"},
	},
}

func TestAssetManifestLoadFromBundleParsesBinaryManifest(t *testing.T) {
	ensureInitialized()
	runAsync(t, func() {
		manifest := services.AssetManifestLoadFromAssetBundle(newManifestBundle(binManifestData)).Await()
		assets := manifest.ListAssets()
		slices.Sort(assets)
		expect(slices.Equal(assets, []string{"assets/bar.png", "assets/foo.png"}), "assets %v", assets)
		foo := manifest.GetAssetVariants("assets/foo.png")
		expect(len(foo) == 2, "foo %v", foo)
		expect(foo[0].Key == "assets/foo.png" && foo[0].TargetDevicePixelRatio == nil && foo[0].Main, "foo[0] %v", foo[0])
		expect(foo[1].Key == "assets/2x/foo.png" && *foo[1].TargetDevicePixelRatio == 2.0 && !foo[1].Main, "foo[1] %v", foo[1])
		bar := manifest.GetAssetVariants("assets/bar.png")
		expect(len(bar) == 1 && bar[0].Key == "assets/bar.png" && bar[0].TargetDevicePixelRatio == nil && bar[0].Main, "bar %v", bar)
	})
}

func TestAssetManifestGetAssetVariantsReturnsNullForUnknownKey(t *testing.T) {
	ensureInitialized()
	runAsync(t, func() {
		manifest := services.AssetManifestLoadFromAssetBundle(newManifestBundle(binManifestData)).Await()
		expect(manifest.GetAssetVariants("invalid asset key") == nil, "not nil")
	})
}

func TestAssetManifestContentHashedMainVariant(t *testing.T) {
	ensureInitialized()
	runAsync(t, func() {
		manifest := services.AssetManifestLoadFromAssetBundle(newManifestBundle(map[string]any{
			"assets/foo.png": []any{
				map[string]any{"asset": "assets/foo.9f64a747.png"},
				map[string]any{"asset": "assets/2.0x/foo.74f81fe1.png", "dpr": 2.0},
			},
			"assets/sub dir/space image.png": []any{
				map[string]any{"asset": "assets/sub dir/space image.1a2b3c4d.png"},
			},
		})).Await()
		foo := manifest.GetAssetVariants("assets/foo.png")
		expect(len(foo) == 2, "foo %v", foo)
		expect(foo[0].Key == "assets/foo.9f64a747.png" && foo[0].TargetDevicePixelRatio == nil && foo[0].Main, "foo[0] %v", foo[0])
		expect(foo[1].Key == "assets/2.0x/foo.74f81fe1.png" && *foo[1].TargetDevicePixelRatio == 2.0 && !foo[1].Main, "foo[1] %v", foo[1])
		spaced := manifest.GetAssetVariants("assets/sub dir/space image.png")
		expect(len(spaced) == 1 && spaced[0].Key == "assets/sub dir/space image.1a2b3c4d.png" && spaced[0].Main, "spaced %v", spaced)
	})
}
