package painting_test

// Port of packages/flutter/test/painting/image_resolution_test.dart
// (Flutter 3.50.1): all 14 tests. The web manifest branch of
// TestAssetBundle.load (AssetManifest.bin.json) is not needed: flutter-go
// is never a web build. expectAsync1 is a callback that must run once.

import (
	"testing"

	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	. "github.com/jonathanperis/flutter-go/packages/flutter/lib/src/painting"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/services"
	"github.com/jonathanperis/flutter-go/src/async"
)

// manifestAssetBundle is image_resolution_test's TestAssetBundle.
type manifestAssetBundle struct {
	services.CachingAssetBundle
	assetBundleMap map[string]any
	loadCallCount  map[string]int
}

func newManifestAssetBundle(assetBundleMap map[string]any) *manifestAssetBundle {
	b := &manifestAssetBundle{assetBundleMap: assetBundleMap, loadCallCount: map[string]int{}}
	b.SetSelf(b)
	return b
}

func (b *manifestAssetBundle) Load(key string) async.Future[[]byte] {
	return async.Async(func() []byte {
		if key == "AssetManifest.bin" {
			return services.NewStandardMessageCodec().EncodeMessage(b.assetBundleMap)
		}
		b.loadCallCount[key]++
		if key == "one" {
			return []byte{49}
		}
		panic(foundation.NewFlutterError("key not found"))
	})
}

// expectKey checks the key obtained, once, asynchronously.
func expectKey(t *testing.T, f async.Future[any], name string, scale float64) func() {
	t.Helper()
	called := 0
	async.Then(f, func(k any) any {
		called++
		key := k.(AssetBundleImageKey)
		if key.Name != name || key.Scale != scale {
			t.Errorf("key %v, want %s @ %v", key, name, scale)
		}
		return nil
	})
	return func() {
		t.Helper()
		if called != 1 {
			t.Errorf("callback called %d times", called)
		}
	}
}

func buildAndTestWithOneAsset(t *testing.T, mainAssetPath string) {
	tester := newImageTester(t)
	assetImage := NewAssetImage(mainAssetPath, AssetImageOptions{Bundle: newManifestAssetBundle(map[string]any{})})
	check := expectKey(t, assetImage.ObtainKey(ImageConfigurationEmpty), mainAssetPath, 1.0)
	tester.Loop().Elapse(0)
	check()
}

func TestAssetImageMainVariantScaleIsOne(t *testing.T) {
	for _, p := range []string{
		"assets/normalFolder/normalFile.png",
		"assets/parentFolder/3.0x/normalFile.png",
		"assets/parentFolder/__3.0x__/leafFolder/normalFile.png",
		"assets/parentFolder/__3.0x_leaf_folder_/normalFile.png",
		"assets/parentFolder/__3.0x__/leafFolder/normalFile.png",
		"assets/parentFolder/3.0x/leafFolder/normalFile.png",
	} {
		t.Run(p, func(t *testing.T) { buildAndTestWithOneAsset(t, p) })
	}
}

func fp(v float64) *float64 { return &v }

func TestAssetImageNotMainVariantScaleIsNotOne(t *testing.T) {
	tester := newImageTester(t)
	const mainAssetPath = "assets/normalFolder/normalFile.png"
	const variantPath = "assets/normalFolder/3.0x/normalFile.png"
	testAssetBundle := newManifestAssetBundle(map[string]any{
		mainAssetPath: []any{map[any]any{"asset": variantPath, "dpr": 3.0}},
	})
	assetImage := NewAssetImage(mainAssetPath, AssetImageOptions{Bundle: testAssetBundle})
	check1 := expectKey(t, assetImage.ObtainKey(ImageConfigurationEmpty), mainAssetPath, 1.0)
	check2 := expectKey(t, assetImage.ObtainKey(ImageConfiguration{Bundle: testAssetBundle, DevicePixelRatio: fp(3.0)}), variantPath, 3.0)
	tester.Loop().Elapse(0)
	check1()
	check2()
}

func TestAssetImageHighResAssetNotPresentReturnsMain(t *testing.T) {
	tester := newImageTester(t)
	const mainAssetPath = "assets/normalFolder/normalFile.png"
	assetBundleMap := map[string]any{mainAssetPath: []any{}}
	testAssetBundle := newManifestAssetBundle(assetBundleMap)
	assetImage := NewAssetImage(mainAssetPath, AssetImageOptions{Bundle: newManifestAssetBundle(assetBundleMap)})
	check1 := expectKey(t, assetImage.ObtainKey(ImageConfigurationEmpty), mainAssetPath, 1.0)
	check2 := expectKey(t, assetImage.ObtainKey(ImageConfiguration{Bundle: testAssetBundle, DevicePixelRatio: fp(3.0)}), mainAssetPath, 1.0)
	tester.Loop().Elapse(0)
	check1()
	check2()
}

func TestAssetImageRangeOfScales(t *testing.T) {
	const mainAssetPath = "assets/normalFolder/normalFile.png"
	const variantPath = "assets/normalFolder/3.0x/normalFile.png"
	for _, c := range []struct {
		name                     string
		deviceRatio, chosenRatio float64
		expectedAssetPath        string
	}{
		{"Obvious case 1.0 - we have exact asset", 1.0, 1.0, mainAssetPath},
		{"Obvious case 3.0 - we have exact asset", 3.0, 3.0, variantPath},
		{"Typical case 2.0", 2.0, 1.0, mainAssetPath},
		{"Borderline case 2.01", 2.01, 3.0, variantPath},
		{"Borderline case 2.9", 2.9, 3.0, variantPath},
		{"Typical case 4.0", 4.0, 3.0, variantPath},
	} {
		t.Run(c.name, func(t *testing.T) {
			tester := newImageTester(t)
			testAssetBundle := newManifestAssetBundle(map[string]any{
				mainAssetPath: []any{
					map[any]any{"asset": mainAssetPath},
					map[any]any{"asset": variantPath, "dpr": 3.0},
				},
			})
			assetImage := NewAssetImage(mainAssetPath, AssetImageOptions{Bundle: testAssetBundle})
			check := expectKey(t, assetImage.ObtainKey(ImageConfiguration{Bundle: testAssetBundle, DevicePixelRatio: &c.deviceRatio}),
				c.expectedAssetPath, c.chosenRatio)
			tester.Loop().Elapse(0)
			check()
		})
	}
}
