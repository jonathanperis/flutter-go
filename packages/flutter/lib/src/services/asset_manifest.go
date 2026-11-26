package services

// Port of asset_manifest.dart (Flutter 3.50.1).

import (
	"regexp"

	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	"github.com/jonathanperis/flutter-go/src/async"
	"github.com/jonathanperis/flutter-go/src/dart"
)

// We use .bin as the extension since it is well-known to represent data
// in some arbitrary binary format.
const kAssetManifestFilename = "AssetManifest.bin"

// AssetManifest contains details about available assets and their
// variants (an abstract class).
type AssetManifest interface {
	// ListAssets lists the keys of all main assets in the application, in
	// the manifest's order (variants already read through
	// GetAssetVariants come last, in the order they were read).
	ListAssets() []string
	// GetAssetVariants retrieves metadata about an asset and its variants;
	// nil (Dart's null) if the asset is not in the manifest.
	GetAssetVariants(key string) []AssetMetadata
}

// AssetManifestLoadFromAssetBundle is AssetManifest.loadFromAssetBundle:
// it loads asset manifest data from bundle and parses it (on the web,
// which flutter-go never is, the manifest is JSON(base64)).
func AssetManifestLoadFromAssetBundle(bundle AssetBundle) async.Future[AssetManifest] {
	if foundation.KIsWeb {
		panic(dart.UnsupportedError{Message: "the web asset manifest is not supported"})
	}
	// On every other platform, the binary file contents are used directly.
	return AssetBundleLoadStructuredBinaryData(bundle, kAssetManifestFilename, assetManifestBinFromStandardMessageCodecMessage)
}

// assetManifestBin is _AssetManifestBin: it lazily parses the binary asset
// manifest (a map of asset keys to lists of variant maps with "asset" and
// "dpr" entries).
type assetManifestBin struct {
	data          *dart.LinkedMap // _data, in the manifest's order
	typeCasted    map[string][]AssetMetadata
	typeCastedOrd []string // _typeCastedData's keys in insertion order
}

func assetManifestBinFromStandardMessageCodecMessage(message []byte) AssetManifest {
	data := newOrderedStandardMessageCodec().DecodeMessage(message)
	entries, ok := data.(*dart.LinkedMap)
	if !ok {
		panic(dart.TypeError{Message: "type '" + foundation.ObjectRuntimeType(data, "Null") +
			"' is not a subtype of type 'Map<Object?, Object?>' in type cast"})
	}
	return &assetManifestBin{data: entries, typeCasted: map[string][]AssetMetadata{}}
}

var hashedExtensionSuffixPattern = regexp.MustCompile(`\.[0-9a-f]{8}((?:\.(?:js|wasm|mjs)\.map)|(?:\.[^./]+))?$`)

func stripContentHash(path string) string {
	loc := hashedExtensionSuffixPattern.FindStringSubmatchIndex(path)
	if loc == nil {
		return path
	}
	group := ""
	if loc[2] >= 0 {
		group = path[loc[2]:loc[3]]
	}
	return path[:loc[0]] + group + path[loc[1]:]
}

func (m *assetManifestBin) GetAssetVariants(key string) []AssetMetadata {
	// We lazily delay typecasting to prevent a performance hiccup when
	// parsing large asset manifests.
	if _, ok := m.typeCasted[key]; !ok {
		variantData := m.data.Lookup(key)
		if variantData == nil {
			return nil
		}
		list, ok := variantData.([]any)
		if !ok {
			panic(dart.TypeError{Message: "type '" + foundation.ObjectRuntimeType(variantData, "Object") +
				"' is not a subtype of type 'Iterable<Object?>' in type cast"})
		}
		variants := make([]AssetMetadata, 0, len(list))
		for _, item := range list {
			variant, ok := item.(*dart.LinkedMap)
			if !ok {
				panic(dart.TypeError{Message: "type '" + foundation.ObjectRuntimeType(item, "Null") +
					"' is not a subtype of type 'Map<Object?, Object?>' in type cast"})
			}
			rawAsset := variant.Lookup("asset")
			if rawAsset == nil {
				panic(dart.TypeError{Message: "Null check operator used on a null value"})
			}
			asset := castElement[string](rawAsset)
			var dpr *float64
			if rawDpr := variant.Lookup("dpr"); rawDpr != nil {
				d, ok := rawDpr.(float64)
				if !ok {
					panic(dart.TypeError{Message: "type '" + foundation.ObjectRuntimeType(rawDpr, "Object") +
						"' is not a subtype of type 'double?' in type cast"})
				}
				dpr = &d
			}
			variants = append(variants, AssetMetadata{
				Key:                    asset,
				TargetDevicePixelRatio: dpr,
				Main:                   key == asset || stripContentHash(asset) == key,
			})
		}
		m.typeCasted[key] = variants
		m.typeCastedOrd = append(m.typeCastedOrd, key)
		m.data.Remove(key)
	}
	return m.typeCasted[key]
}

func (m *assetManifestBin) ListAssets() []string {
	out := make([]string, 0, m.data.Len()+len(m.typeCastedOrd))
	for _, k := range m.data.Keys() {
		out = append(out, castElement[string](k))
	}
	return append(out, m.typeCastedOrd...)
}

// AssetMetadata contains information about an asset (a struct literal).
type AssetMetadata struct {
	// TargetDevicePixelRatio is the device pixel ratio that this asset is
	// most ideal for (nil: the main asset, not a resolution variant).
	TargetDevicePixelRatio *float64
	// Key is the asset's key, which is the path to the asset specified in
	// the pubspec.
	Key string
	// Main is whether or not this is a main asset (not a variant).
	Main bool
}
