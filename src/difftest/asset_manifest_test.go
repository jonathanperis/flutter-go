package difftest

// Go twin of flutter/test/asset_manifest_test.dart: binary asset manifests
// parsed by AssetManifestLoadFromAssetBundle, read in various orders, and
// the keys PlatformAssetBundle sends on flutter/assets.

import (
	"errors"
	"strings"
	"testing"

	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/services"
	flutter_test "github.com/jonathanperis/flutter-go/packages/flutter_test/lib/src"
	"github.com/jonathanperis/flutter-go/src/async"
	"github.com/jonathanperis/flutter-go/src/dart"
)

type manifestTestBundle struct {
	services.AssetBundleBase
	manifest any
}

func newManifestTestBundle(manifest any) *manifestTestBundle {
	b := &manifestTestBundle{manifest: manifest}
	b.SetSelf(b)
	return b
}

func (b *manifestTestBundle) Load(key string) async.Future[[]byte] {
	return async.Async(func() []byte {
		if key == "AssetManifest.bin" {
			return services.NewStandardMessageCodec().EncodeMessage(b.manifest)
		}
		panic(errors.New("Invalid argument(s): Unexpected key: " + key))
	})
}

func TestAssetManifestMatchesFlutter(t *testing.T) {
	runLoggedScenes(t, "flutter/build/asset_manifest.json", func(add func(string)) {
		type entry = dart.MapEntry[string, any]
		describe := func(variants []services.AssetMetadata) string {
			if variants == nil {
				return "null"
			}
			parts := make([]string, len(variants))
			for i, v := range variants {
				dpr := "null"
				if v.TargetDevicePixelRatio != nil {
					dpr = dart.DoubleToString(*v.TargetDevicePixelRatio)
				}
				parts[i] = "(" + v.Key + ", " + dpr + ", " + dart.ToString(v.Main) + ")"
			}
			return strings.Join(parts, " ")
		}
		list := func(m services.AssetManifest) string { return "list [" + strings.Join(m.ListAssets(), ", ") + "]" }
		errorName := func(r any) string {
			if _, ok := r.(dart.TypeError); ok {
				return "TypeError"
			}
			return "?" + dart.ToString(r)
		}
		scene := func(name string, manifest any, reads []string) {
			add("scene " + name)
			var m services.AssetManifest
			func() {
				defer func() {
					if r := recover(); r != nil {
						add("load !! " + errorName(r))
					}
				}()
				m = services.AssetManifestLoadFromAssetBundle(newManifestTestBundle(manifest)).Await()
			}()
			if m == nil {
				return
			}
			add(list(m))
			for _, key := range reads {
				func() {
					defer func() {
						if r := recover(); r != nil {
							add("variants " + key + " !! " + errorName(r))
						}
					}()
					add("variants " + key + " " + describe(m.GetAssetVariants(key)))
				}()
				add(list(m))
			}
		}
		v := func(asset string, dpr ...any) []entry {
			e := []entry{{Key: "asset", Value: asset}}
			if len(dpr) > 0 {
				e = append(e, entry{Key: "dpr", Value: dpr[0]})
			}
			return e
		}
		l := func(items ...any) []any { return items }

		scene("flutter test", []entry{
			{Key: "assets/foo.png", Value: l(v("assets/foo.png"), v("assets/2x/foo.png", 2.0))},
			{Key: "assets/bar.png", Value: l(v("assets/bar.png"))},
		}, []string{"assets/bar.png", "assets/foo.png", "assets/bar.png", "nope"})

		scene("resolutions", []entry{
			{Key: "icons/a.png", Value: l(v("icons/a.png"), v("icons/1.5x/a.png", 1.5), v("icons/2.0x/a.png", 2.0),
				v("icons/3.0x/a.png", 3.0), v("icons/4.0x/a.png", 4.0))},
			{Key: "icons/b.png", Value: l(v("icons/2.0x/b.png", 2.0))},
			{Key: "empty", Value: l()},
			{Key: "z", Value: l(v("z"))},
			{Key: "a", Value: l(v("a"))},
			{Key: "m", Value: l(v("m"))},
		}, []string{"m", "icons/b.png", "empty", "a", "icons/a.png", "z", "m"})

		hashed := []entry{
			{Key: "assets/foo.png", Value: l(v("assets/foo.9f64a747.png"), v("assets/2.0x/foo.74f81fe1.png", 2.0))},
			{Key: "assets/sub dir/space image.png", Value: l(v("assets/sub dir/space image.1a2b3c4d.png"))},
			{Key: "a/b.js.map", Value: l(v("a/b.abcdef01.js.map"))},
			{Key: "a/c.wasm.map", Value: l(v("a/c.00000000.wasm.map"))},
			{Key: "a/d.mjs.map", Value: l(v("a/d.12345678.mjs.map"))},
			{Key: "a/e.map", Value: l(v("a/e.12345678.map"))},
			{Key: "x", Value: l(v("x.abcdef01"))},
			{Key: "upper.png", Value: l(v("upper.ABCDEF01.png"))},
			{Key: "short.png", Value: l(v("short.abcdef0.png"))},
			{Key: "long.png", Value: l(v("long.abcdef012.png"))},
			{Key: "nothex.png", Value: l(v("nothex.abcdefgh.png"))},
			{Key: "two.11111111.png", Value: l(v("two.11111111.22222222.png"))},
			{Key: "two.png", Value: l(v("two.11111111.22222222.png"))},
			{Key: "dir.12345678/file.png", Value: l(v("dir.12345678/file.png"))},
			{Key: "dir/file.png", Value: l(v("dir.12345678/file.png"))},
			{Key: "multi.tar.gz", Value: l(v("multi.12345678.tar.gz"))},
			{Key: "multi.gz", Value: l(v("multi.12345678.gz"))},
		}
		keys := make([]string, len(hashed))
		for i, e := range hashed {
			keys[i] = e.Key
		}
		scene("content hashes", hashed, keys)

		scene("malformed", []entry{
			{Key: "not a list", Value: "x"},
			{Key: "null variant", Value: l(nil)},
			{Key: "no asset", Value: l([]entry{{Key: "dpr", Value: 2.0}})},
			{Key: "int dpr", Value: l(v("a", 2))},
			{Key: "string dpr", Value: l(v("a", "2.0"))},
			{Key: "asset not a string", Value: l([]entry{{Key: "asset", Value: 1}})},
			{Key: "null value", Value: nil},
			{Key: "ok", Value: l(v("ok"))},
		}, []string{"not a list", "null variant", "no asset", "int dpr", "string dpr", "asset not a string", "null value", "ok"})

		scene("not a map", l("a"), nil)
		scene("empty", []entry{}, []string{"a"})

		add("scene asset keys")
		var sent []string
		messenger := flutter_test.TestDefaultBinaryMessengerBindingInstance().DefaultBinaryMessenger().(*flutter_test.TestDefaultBinaryMessenger)
		messenger.SetMockMessageHandler("flutter/assets", func(message []byte) async.Future[[]byte] {
			return async.Async(func() []byte {
				sent = append(sent, dart.UTF8Decode(message))
				return []byte{0}
			})
		})
		jsonEncode := dart.JSONEncodeString
		for _, key := range []string{
			"assets/a.png",
			"assets/sub dir/space image.png",
			"ünïcødé/✓.png",
			"q?x=1#frag",
			"percent %41 %zz %",
			"brackets[]{}|^`\"<>",
			`back\slash`,
			"./dot/../a/./b",
			"../up",
			"a:b/c",
			"ab:c",
			"1a:b",
			"/abs/../x",
			"emoji 😀.png",
			"sub-delims !$&'()*+,;=@~_.-",
			"",
		} {
			services.RootBundle.Load(key).Await()
			add("key " + jsonEncode(key) + " -> " + jsonEncode(sent[len(sent)-1]))
		}
	})
}
