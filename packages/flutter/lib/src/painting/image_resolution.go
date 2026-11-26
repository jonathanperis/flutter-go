package painting

// Port of image_resolution.dart (Flutter 3.50.1): AssetImage, which picks
// the asset variant best suited to the device pixel ratio from the
// bundle's AssetManifest.

import (
	"fmt"
	"sort"

	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/services"
	"github.com/jonathanperis/flutter-go/src/async"
	"github.com/jonathanperis/flutter-go/src/dart"
)

const kLowDprLimit = 2.0

// The default resolution of the main asset (1.0).
const naturalResolution = 1.0

// AssetImage fetches an image from an AssetBundle, choosing the variant
// (e.g. "2.0x/name.png") matching the configuration's device pixel ratio.
type AssetImage struct {
	AssetBundleImageProviderBase
	// AssetName is the name of the main asset.
	AssetName string
	// Bundle is the bundle from which the image will be obtained (nil: the
	// configuration's, or RootBundle).
	Bundle services.AssetBundle
	// Package is the name of the package from which the image is included
	// (nil: none).
	Package *string
}

// AssetImageOptions are AssetImage's named parameters.
type AssetImageOptions struct {
	Bundle  services.AssetBundle
	Package *string
}

// NewAssetImage is AssetImage(assetName, {bundle, package}).
func NewAssetImage(assetName string, opts ...AssetImageOptions) *AssetImage {
	var o AssetImageOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	p := &AssetImage{AssetName: assetName, Bundle: o.Bundle, Package: o.Package}
	p.SetSelf(p)
	return p
}

// KeyName is the name used to generate the key to obtain the asset
// ("packages/<package>/<assetName>" for a package's asset).
func (p *AssetImage) KeyName() string {
	if p.Package == nil {
		return p.AssetName
	}
	return "packages/" + *p.Package + "/" + p.AssetName
}

// ObtainKey loads the bundle's AssetManifest and picks the variant for the
// configuration; the result is synchronous when the manifest is already
// loaded.
func (p *AssetImage) ObtainKey(configuration ImageConfiguration) async.Future[any] {
	chosenBundle := p.Bundle
	if chosenBundle == nil {
		chosenBundle = configuration.Bundle
	}
	if chosenBundle == nil {
		chosenBundle = services.RootBundle
	}
	var completer *async.Completer[any]
	var result async.Future[any]
	keyFuture := services.AssetManifestLoadFromAssetBundle(chosenBundle).ThenAny(func(manifest services.AssetManifest) any {
		candidateVariants := manifest.GetAssetVariants(p.KeyName())
		chosenVariant := p.chooseVariant(p.KeyName(), configuration, candidateVariants)
		scale := naturalResolution
		if chosenVariant.TargetDevicePixelRatio != nil {
			scale = *chosenVariant.TargetDevicePixelRatio
		}
		key := AssetBundleImageKey{Bundle: chosenBundle, Name: chosenVariant.Key, Scale: scale}
		if completer != nil {
			// We already returned from this function, which means we are in
			// the asynchronous mode. Pass the value to the completer. The
			// completer's future is what we returned.
			completer.Complete(key)
		} else {
			// We haven't yet returned, so we must have been called
			// synchronously just after loadStructuredData returned (which
			// means it provided us with a SynchronousFuture). Let's return a
			// SynchronousFuture ourselves.
			result = foundation.NewSynchronousFuture[any](key)
		}
		return nil
	}, nil)
	async.OnComplete(keyFuture, func(any) {}, func(err error) {
		// We had an error. (This guarantees we weren't called synchronously.)
		// Forward the error to the caller.
		dartAssert(completer != nil)
		dartAssert(result == nil)
		completer.CompleteError(err)
	})
	if result != nil {
		// The code above ran synchronously, and came up with an answer.
		// Return the SynchronousFuture that we created above.
		return result
	}
	// The code above hasn't yet run its "then" handler yet. Let's prepare a
	// completer for it to use when it does run.
	completer = async.NewCompleter[any]()
	return completer.Future()
}

func (p *AssetImage) chooseVariant(mainAssetKey string, config ImageConfiguration, candidateVariants []services.AssetMetadata) services.AssetMetadata {
	if len(candidateVariants) == 0 || config.DevicePixelRatio == nil {
		return services.AssetMetadata{Key: mainAssetKey, Main: true}
	}
	// SplayTreeMap<double, AssetMetadata>: keys ordered by
	// double.compareTo (-0.0 before 0.0), a later candidate replacing an
	// earlier one with the same ratio.
	var candidates []dprVariant
	for _, candidate := range candidateVariants {
		dpr := naturalResolution
		if candidate.TargetDevicePixelRatio != nil {
			dpr = *candidate.TargetDevicePixelRatio
		}
		found := false
		for i := range candidates {
			if dart.CompareDoubles(candidates[i].dpr, dpr) == 0 {
				candidates[i].metadata, found = candidate, true
				break
			}
		}
		if !found {
			candidates = append(candidates, dprVariant{dpr, candidate})
		}
	}
	sort.SliceStable(candidates, func(i, j int) bool { return dart.CompareDoubles(candidates[i].dpr, candidates[j].dpr) < 0 })
	return findBestVariant(candidates, *config.DevicePixelRatio)
}

type dprVariant struct {
	dpr      float64
	metadata services.AssetMetadata
}

// findBestVariant returns the candidate whose ratio is closest to value:
// the exact one if any; else, below 2.0 (where images look worse scaled
// up), the next larger one; else the closer neighbor, the larger one past
// the midpoint.
func findBestVariant(candidates []dprVariant, value float64) services.AssetMetadata {
	var lower, upper *dprVariant
	for i := range candidates {
		c := dart.CompareDoubles(candidates[i].dpr, value)
		if c == 0 {
			return candidates[i].metadata
		}
		if c < 0 {
			lower = &candidates[i]
		} else if upper == nil {
			upper = &candidates[i]
		}
	}
	if lower == nil {
		return upper.metadata
	}
	if upper == nil {
		return lower.metadata
	}
	// On screens with low device-pixel ratios the artifacts from
	// upscaling images are more visible than on screens with a higher
	// device-pixel ratios because the physical pixels are larger. Choose
	// the higher resolution image in that case instead of the nearest one.
	if value < kLowDprLimit || value > (lower.dpr+upper.dpr)/2 {
		return upper.metadata
	}
	return lower.metadata
}

// Equal is ==: same key name and bundle.
func (p *AssetImage) Equal(other any) bool {
	o, ok := other.(*AssetImage)
	return ok && o.KeyName() == p.KeyName() && o.Bundle == p.Bundle
}

func (p *AssetImage) String() string {
	bundle := "null"
	if p.Bundle != nil {
		bundle = p.Bundle.String()
	}
	return fmt.Sprintf(`%s(bundle: %s, name: "%s")`, foundation.ObjectRuntimeType(p, "AssetImage"), bundle, p.KeyName())
}
