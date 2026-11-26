package painting

// Port of debug.dart (Flutter 3.50.1).

import (
	"fmt"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	"github.com/jonathanperis/flutter-go/src/dart"
	"github.com/jonathanperis/flutter-go/src/io"
)

// DebugDisableShadows makes BoxShadow and PhysicalModel paint solid
// shadows instead of blurred ones (for golden tests).
var DebugDisableShadows = false

// HttpClientProvider is the signature of a method that returns an
// HttpClient.
type HttpClientProvider func() io.HttpClient

// DebugNetworkImageHttpClientProvider provides the HttpClient NetworkImage
// uses, in debug builds (nil: the shared client).
var DebugNetworkImageHttpClientProvider HttpClientProvider

// PaintImageCallback is the signature of DebugOnPaintImage.
type PaintImageCallback func(info ImageSizeInfo)

// ImageSizeInfo tracks the bytes used by a ui.Image compared to the bytes
// needed to paint it. Compare with ==.
type ImageSizeInfo struct {
	// Source is a unique identifier for this image (its debugLabel, or the
	// URL or file it was loaded from); "" stands for Dart's null.
	Source string
	// DisplaySize is the size the image was painted at, in physical pixels.
	DisplaySize ui.Size
	// ImageSize is the size of the decoded image.
	ImageSize ui.Size
}

// DisplaySizeInBytes is the bytes needed to paint the image at its display
// size, with mipmaps.
func (i ImageSizeInfo) DisplaySizeInBytes() int { return sizeToBytes(i.DisplaySize) }

// DecodedSizeInBytes is the bytes used by the decoded image, with mipmaps.
func (i ImageSizeInfo) DecodedSizeInBytes() int { return sizeToBytes(i.ImageSize) }

func sizeToBytes(size ui.Size) int {
	// Assume 4 bytes per pixel and that mipmapping will be used, which adds
	// 4/3.
	return int(float64(float64(size.Width*size.Height)*4) * (4.0 / 3.0))
}

// ToJson is toJson: the info as a JSON-ready ordered map.
func (i ImageSizeInfo) ToJson() []dart.MapEntry[string, any] {
	var source any
	if i.Source != "" {
		source = i.Source
	}
	return []dart.MapEntry[string, any]{
		{Key: "source", Value: source},
		{Key: "displaySize", Value: []dart.MapEntry[string, any]{{Key: "width", Value: i.DisplaySize.Width}, {Key: "height", Value: i.DisplaySize.Height}}},
		{Key: "imageSize", Value: []dart.MapEntry[string, any]{{Key: "width", Value: i.ImageSize.Width}, {Key: "height", Value: i.ImageSize.Height}}},
		{Key: "displaySizeInBytes", Value: i.DisplaySizeInBytes()},
		{Key: "decodedSizeInBytes", Value: i.DecodedSizeInBytes()},
	}
}

func (i ImageSizeInfo) String() string {
	source := "null"
	if i.Source != "" {
		source = i.Source
	}
	return fmt.Sprintf("ImageSizeInfo(%s, imageSize: %v, displaySize: %v)", source, i.ImageSize, i.DisplaySize)
}

// DebugOnPaintImage is called with every image painted by paintImage, in
// debug builds.
var DebugOnPaintImage PaintImageCallback

// DebugInvertOversizedImages makes paintImage invert the colors of images
// decoded much larger than they are painted, and report an error.
var DebugInvertOversizedImages = false

const imageOverheadAllowanceDefault = 128 * 1024

// DebugImageOverheadAllowance is the number of bytes an image may exceed
// its display size by before DebugInvertOversizedImages flags it.
var DebugImageOverheadAllowance = imageOverheadAllowanceDefault

// DebugAssertAllPaintingVarsUnsetOptions are
// debugAssertAllPaintingVarsUnset's named parameters.
type DebugAssertAllPaintingVarsUnsetOptions struct {
	DebugDisableShadowsOverride bool
}

// DebugAssertAllPaintingVarsUnset panics with FlutterError(reason) if a
// painting debug variable was changed (tests call it when they end).
func DebugAssertAllPaintingVarsUnset(reason string, opts ...DebugAssertAllPaintingVarsUnsetOptions) bool {
	var o DebugAssertAllPaintingVarsUnsetOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	if DebugDisableShadows != o.DebugDisableShadowsOverride ||
		DebugNetworkImageHttpClientProvider != nil ||
		DebugOnPaintImage != nil ||
		DebugInvertOversizedImages ||
		DebugImageOverheadAllowance != imageOverheadAllowanceDefault {
		panic(foundation.NewFlutterError(reason))
	}
	return true
}

// ShaderWarmUpPictureCallback is the signature of
// DebugCaptureShaderWarmUpPicture.
type ShaderWarmUpPictureCallback func(picture *ui.Picture) bool

// ShaderWarmUpImageCallback is the signature of
// DebugCaptureShaderWarmUpImage.
type ShaderWarmUpImageCallback func(image *ui.Image) bool

// DebugCaptureShaderWarmUpPicture is called by ShaderWarmUp.Execute with
// the warm-up picture (debug builds).
var DebugCaptureShaderWarmUpPicture ShaderWarmUpPictureCallback = defaultPictureCapture

func defaultPictureCapture(picture *ui.Picture) bool { return true }

// DebugCaptureShaderWarmUpImage is called by ShaderWarmUp.Execute with the
// warm-up image (debug builds).
var DebugCaptureShaderWarmUpImage ShaderWarmUpImageCallback = defaultImageCapture

func defaultImageCapture(image *ui.Image) bool { return true }

// DebugCheckCanResolveTextDirection panics with a FlutterError when
// direction is nil (target names the type being resolved).
func DebugCheckCanResolveTextDirection(direction *ui.TextDirection, target string) bool {
	if direction == nil {
		panic(foundation.NewFlutterErrorFromParts([]foundation.DiagnosticsNode{
			foundation.NewErrorSummary("No TextDirection found."),
			foundation.NewErrorDescription("To resolve " + target + " properties, it must be provided with a TextDirection."),
			foundation.NewErrorHint("This error usually occurs when " + target + " is used in a widget without " +
				"a Directionality ancestor."),
			foundation.NewErrorHint("Typically, the Directionality widget is introduced by the MaterialApp " +
				"or WidgetsApp widget at the top of your application widget tree. It " +
				"determines the ambient reading direction and is used, for example, to " +
				"determine how to lay out text, how to interpret \"start\" and \"end\" " +
				"values, and to resolve EdgeInsetsDirectional, " +
				"AlignmentDirectional, and other *Directional objects."),
		}))
	}
	return true
}
