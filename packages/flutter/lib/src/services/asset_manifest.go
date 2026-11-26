package services

// Port of asset_manifest.dart (Flutter 3.50.1): AssetManifest (the
// flutter tool's AssetManifest.bin, a StandardMessageCodec map from asset
// keys to their variants) and AssetMetadata.

import (
	"regexp"
	"sort"

	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	"github.com/jonathanperis/flutter-go/src/async"
	"github.com/jonathanperis/flutter-go/src/dart"
)

const (
	assetManifestFilename    = "AssetManifest.bin"
	assetManifestWebFilename = "AssetManifest.bin.json"
)

// AssetManifest contains details about available assets and their
// variants.
type AssetManifest interface {
	// ListAssets lists the keys of all main assets (not variants). The
	// keys not yet looked up with GetAssetVariants come first, sorted (Go
	// maps lose the manifest's order), then the looked-up ones in lookup
	// order.
	ListAssets() []string
	// GetAssetVariants returns the asset's available variants, the main
	// asset included; nil when key is not a main asset.
	GetAssetVariants(key string) []AssetMetadata
}

// AssetManifestLoadFromAssetBundle is AssetManifest.loadFromAssetBundle:
// it loads and parses the manifest of bundle (cached by
// CachingAssetBundle's LoadStructuredBinaryData).
func AssetManifestLoadFromAssetBundle(bundle AssetBundle) async.Future[AssetManifest] {
	if foundation.KIsWeb {
		panic(dart.UnsupportedError{Message: assetManifestWebFilename + " is web-only"})
	}
	return AssetBundleLoadStructuredBinaryData(bundle, assetManifestFilename,
		func(data []byte) AssetManifest { return assetManifestBinFromStandardMessageCodecMessage(data) })
}

type assetManifestBin struct {
	data           map[any]any
	typeCastedData map[string][]AssetMetadata
	typeCastedKeys []string
}

func assetManifestBinFromStandardMessageCodecMessage(message []byte) *assetManifestBin {
	data := NewStandardMessageCodec().DecodeMessage(message)
	m, _ := data.(map[any]any)
	if m == nil && data != nil {
		panic(dart.NewException("AssetManifest.bin is not a map"))
	}
	return &assetManifestBin{data: m, typeCastedData: map[string][]AssetMetadata{}}
}

var hashedExtensionSuffixPattern = regexp.MustCompile(`\.[0-9a-f]{8}((?:\.(?:js|wasm|mjs)\.map)|(?:\.[^./]+))?$`)

func stripContentHash(p string) string {
	loc := hashedExtensionSuffixPattern.FindStringSubmatchIndex(p)
	if loc == nil {
		return p
	}
	group := ""
	if loc[2] >= 0 {
		group = p[loc[2]:loc[3]]
	}
	return p[:loc[0]] + group + p[loc[1]:]
}

func (m *assetManifestBin) GetAssetVariants(key string) []AssetMetadata {
	if _, ok := m.typeCastedData[key]; !ok {
		variantData := m.data[key]
		if variantData == nil {
			return nil
		}
		var variants []AssetMetadata
		for _, d := range variantData.([]any) {
			data := d.(map[any]any)
			asset := data["asset"].(string)
			var dpr *float64
			if v, ok := data["dpr"].(float64); ok {
				dpr = &v
			}
			variants = append(variants, AssetMetadata{
				Key:                    asset,
				TargetDevicePixelRatio: dpr,
				Main:                   key == asset || stripContentHash(asset) == key,
			})
		}
		if variants == nil {
			variants = []AssetMetadata{}
		}
		m.typeCastedData[key] = variants
		m.typeCastedKeys = append(m.typeCastedKeys, key)
		delete(m.data, key)
	}
	return m.typeCastedData[key]
}

func (m *assetManifestBin) ListAssets() []string {
	keys := make([]string, 0, len(m.data)+len(m.typeCastedKeys))
	for k := range m.data {
		keys = append(keys, k.(string))
	}
	sort.Strings(keys)
	return append(keys, m.typeCastedKeys...)
}

// AssetMetadata contains information about an asset.
type AssetMetadata struct {
	// TargetDevicePixelRatio is the device pixel ratio this asset is most
	// ideal for (nil: unspecified).
	TargetDevicePixelRatio *float64
	// Key is the asset's key, its path in the bundle.
	Key string
	// Main is whether this is the main asset rather than a variant.
	Main bool
}
