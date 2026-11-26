package painting

// Port of decoration_image.dart (Flutter 3.50.1): ImageRepeat,
// DecorationImage, DecorationImagePainter, paintImage and
// debugFlushLastFrameImageSizeInfo.
//
// DecorationImage is a Go interface: DecorationImage.lerp returns the
// private _BlendedDecorationImage, which implements DecorationImage, so
// Dart's public fields are getter methods (Image(), Fit()...).
// NewDecorationImage(DecorationImageOptions{...}) builds the plain one.

import (
	"math"
	"strconv"
	"strings"
	"time"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/scheduler"
	"github.com/jonathanperis/flutter-go/src/dart"
	"github.com/jonathanperis/flutter-go/src/developer"
)

// ImageRepeat is how to paint any portions of a box not covered by an
// image.
type ImageRepeat int

const (
	// ImageRepeatRepeat repeats the image in both the x and y directions.
	ImageRepeatRepeat ImageRepeat = iota
	// ImageRepeatRepeatX repeats the image in the x direction only.
	ImageRepeatRepeatX
	// ImageRepeatRepeatY repeats the image in the y direction only.
	ImageRepeatRepeatY
	// ImageRepeatNoRepeat leaves uncovered portions of the box transparent.
	ImageRepeatNoRepeat
)

// ImageRepeatValues is ImageRepeat.values.
var ImageRepeatValues = []ImageRepeat{ImageRepeatRepeat, ImageRepeatRepeatX, ImageRepeatRepeatY, ImageRepeatNoRepeat}

// Name is the Dart enum value's name.
func (r ImageRepeat) Name() string {
	return [...]string{"repeat", "repeatX", "repeatY", "noRepeat"}[r]
}

func (r ImageRepeat) String() string { return "ImageRepeat." + r.Name() }

// DecorationImage is an image for a box decoration.
type DecorationImage interface {
	// Image is the image to be painted into the decoration.
	Image() ImageProvider
	// Placeholder is painted until Image is available (nil: none).
	Placeholder() ImageProvider
	// OnError is called when an error occurs loading the image (nil: none).
	OnError() ImageErrorListener
	// ColorFilter is applied to the image before painting it (nil: none).
	ColorFilter() *ui.ColorFilter
	// Fit is how the image should be inscribed into the box (nil: null,
	// scaleDown, or fill with a centerSlice).
	Fit() *BoxFit
	// Alignment is how to align the image within its bounds.
	Alignment() AlignmentGeometry
	// CenterSlice is the center slice for a nine-patch image (nil: none).
	CenterSlice() *ui.Rect
	// Repeat is how to paint any portions of the box not covered by the
	// image.
	Repeat() ImageRepeat
	// MatchTextDirection says whether to paint the image in the direction
	// of the TextDirection.
	MatchTextDirection() bool
	// Scale is the image's scale (image pixels per logical pixel).
	Scale() float64
	// Opacity is the opacity applied to the image (0.0 to 1.0).
	Opacity() float64
	// FilterQuality is the quality of image sampling.
	FilterQuality() ui.FilterQuality
	// InvertColors says whether the colors of the image are inverted when
	// drawn.
	InvertColors() bool
	// IsAntiAlias says whether to paint the image with anti-aliasing.
	IsAntiAlias() bool
	// CreatePainter creates a DecorationImagePainter for this image.
	CreatePainter(onChanged func()) DecorationImagePainter
	// Equal is ==.
	Equal(other DecorationImage) bool
	String() string
}

// DecorationImageOptions are DecorationImage's named parameters (nil
// Alignment: Alignment.center; nil Repeat: noRepeat; nil Scale/Opacity:
// 1.0; nil FilterQuality: medium).
type DecorationImageOptions struct {
	Image              ImageProvider
	Placeholder        ImageProvider
	OnError            ImageErrorListener
	ColorFilter        *ui.ColorFilter
	Fit                *BoxFit
	Alignment          AlignmentGeometry
	CenterSlice        *ui.Rect
	Repeat             *ImageRepeat
	MatchTextDirection bool
	Scale              *float64
	Opacity            *float64
	FilterQuality      *ui.FilterQuality
	InvertColors       bool
	IsAntiAlias        bool
}

type decorationImage struct {
	image              ImageProvider
	placeholder        ImageProvider
	onError            ImageErrorListener
	colorFilter        *ui.ColorFilter
	fit                *BoxFit
	alignment          AlignmentGeometry
	centerSlice        *ui.Rect
	repeat             ImageRepeat
	matchTextDirection bool
	scale              float64
	opacity            float64
	filterQuality      ui.FilterQuality
	invertColors       bool
	isAntiAlias        bool
}

// NewDecorationImage is DecorationImage({required image, ...}).
func NewDecorationImage(o DecorationImageOptions) DecorationImage {
	d := &decorationImage{image: o.Image, placeholder: o.Placeholder, onError: o.OnError, colorFilter: o.ColorFilter,
		fit: o.Fit, alignment: o.Alignment, centerSlice: o.CenterSlice, repeat: ImageRepeatNoRepeat,
		matchTextDirection: o.MatchTextDirection, scale: 1.0, opacity: 1.0, filterQuality: ui.FilterQualityMedium,
		invertColors: o.InvertColors, isAntiAlias: o.IsAntiAlias}
	if d.alignment == nil {
		d.alignment = AlignmentCenter
	}
	if o.Repeat != nil {
		d.repeat = *o.Repeat
	}
	if o.Scale != nil {
		d.scale = *o.Scale
	}
	if o.Opacity != nil {
		d.opacity = *o.Opacity
	}
	if o.FilterQuality != nil {
		d.filterQuality = *o.FilterQuality
	}
	return d
}

func (d *decorationImage) Image() ImageProvider            { return d.image }
func (d *decorationImage) Placeholder() ImageProvider      { return d.placeholder }
func (d *decorationImage) OnError() ImageErrorListener     { return d.onError }
func (d *decorationImage) ColorFilter() *ui.ColorFilter    { return d.colorFilter }
func (d *decorationImage) Fit() *BoxFit                    { return d.fit }
func (d *decorationImage) Alignment() AlignmentGeometry    { return d.alignment }
func (d *decorationImage) CenterSlice() *ui.Rect           { return d.centerSlice }
func (d *decorationImage) Repeat() ImageRepeat             { return d.repeat }
func (d *decorationImage) MatchTextDirection() bool        { return d.matchTextDirection }
func (d *decorationImage) Scale() float64                  { return d.scale }
func (d *decorationImage) Opacity() float64                { return d.opacity }
func (d *decorationImage) FilterQuality() ui.FilterQuality { return d.filterQuality }
func (d *decorationImage) InvertColors() bool              { return d.invertColors }
func (d *decorationImage) IsAntiAlias() bool               { return d.isAntiAlias }

// CreatePainter creates a DecorationImagePainter for this image.
func (d *decorationImage) CreatePainter(onChanged func()) DecorationImagePainter {
	return newDecorationImagePainter(d, onChanged)
}

func providersEqual(a, b ImageProvider) bool {
	if isNilValue(a) || isNilValue(b) {
		return isNilValue(a) && isNilValue(b)
	}
	return KeysEqual(a, b)
}

func colorFiltersEqual(a, b *ui.ColorFilter) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return a.Equal(b)
}

func ptrEq[T comparable](a, b *T) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}

// Equal is ==.
func (d *decorationImage) Equal(other DecorationImage) bool {
	o, ok := other.(*decorationImage)
	if !ok || o == nil {
		return false
	}
	return d == o || (providersEqual(o.image, d.image) && providersEqual(o.placeholder, d.placeholder) &&
		colorFiltersEqual(o.colorFilter, d.colorFilter) && ptrEq(o.fit, d.fit) &&
		alignmentGeometryEqual(o.alignment, d.alignment) && ptrEq(o.centerSlice, d.centerSlice) &&
		o.repeat == d.repeat && o.matchTextDirection == d.matchTextDirection && o.scale == d.scale &&
		o.opacity == d.opacity && o.filterQuality == d.filterQuality && o.invertColors == d.invertColors &&
		o.isAntiAlias == d.isAntiAlias)
}

func stringOf(v any) string {
	if isNilValue(v) {
		return "null"
	}
	if s, ok := v.(interface{ String() string }); ok {
		return s.String()
	}
	return "Instance of '" + foundation.ObjectRuntimeType(v, "Object") + "'"
}

func (d *decorationImage) String() string {
	p := []string{stringOf(d.image)}
	if d.placeholder != nil {
		p = append(p, "placeholder: "+stringOf(d.placeholder))
	}
	if d.colorFilter != nil {
		p = append(p, d.colorFilter.String())
	}
	if d.fit != nil && !(*d.fit == BoxFitFill && d.centerSlice != nil) && !(*d.fit == BoxFitScaleDown && d.centerSlice == nil) {
		p = append(p, d.fit.String())
	}
	p = append(p, d.alignment.String())
	if d.centerSlice != nil {
		p = append(p, "centerSlice: "+d.centerSlice.String())
	}
	if d.repeat != ImageRepeatNoRepeat {
		p = append(p, d.repeat.String())
	}
	if d.matchTextDirection {
		p = append(p, "match text direction")
	}
	p = append(p, "scale "+dart.ToStringAsFixed(d.scale, 1), "opacity "+dart.ToStringAsFixed(d.opacity, 1),
		d.filterQuality.String())
	if d.invertColors {
		p = append(p, "invert colors")
	}
	if d.isAntiAlias {
		p = append(p, "use anti-aliasing")
	}
	return runtimeTypeName("DecorationImage", "DecorationImage") + "(" + strings.Join(p, ", ") + ")"
}

// DecorationImageLerp is DecorationImage.lerp: a blend of a and b at t
// (nil is null).
func DecorationImageLerp(a, b DecorationImage, t float64) DecorationImage {
	if isNilValue(a) {
		a = nil
	}
	if isNilValue(b) {
		b = nil
	}
	if sameObject(a, b) || t == 0.0 {
		return a
	}
	if t == 1.0 {
		return b
	}
	return &blendedDecorationImage{a, b, t}
}

// DecorationImagePainter is the painter for a DecorationImage.
type DecorationImagePainter interface {
	// Paint draws the image onto canvas in rect, clipped to clipPath (nil:
	// no clip). blend and blendMode are Dart's named parameters (1.0 and
	// srcOver by default, see DecorationImagePainterPaintOptions).
	Paint(canvas ui.Canvas, rect ui.Rect, clipPath *ui.Path, configuration ImageConfiguration,
		opts ...DecorationImagePainterPaintOptions)
	// Dispose releases the cache's handles on the image(s).
	Dispose()
}

// DecorationImagePainterPaintOptions are paint's named parameters (nil
// Blend: 1.0; nil BlendMode: srcOver).
type DecorationImagePainterPaintOptions struct {
	Blend     *float64
	BlendMode *ui.BlendMode
}

func (o DecorationImagePainterPaintOptions) resolve() (float64, ui.BlendMode) {
	blend, mode := 1.0, ui.BlendModeSrcOver
	if o.Blend != nil {
		blend = *o.Blend
	}
	if o.BlendMode != nil {
		mode = *o.BlendMode
	}
	return blend, mode
}

func firstPaintOptions(opts []DecorationImagePainterPaintOptions) DecorationImagePainterPaintOptions {
	if len(opts) > 0 {
		return opts[0]
	}
	return DecorationImagePainterPaintOptions{}
}

type decorationImagePainter struct {
	details          DecorationImage
	onChanged        func()
	imageStream      *ImageStream
	image            *ImageInfo
	placeholderStr   *ImageStream
	placeholderImage *ImageInfo
}

func newDecorationImagePainter(details DecorationImage, onChanged func()) *decorationImagePainter {
	p := &decorationImagePainter{details: details, onChanged: onChanged}
	dartAssert(foundation.DebugMaybeDispatchCreated("painting", "_DecorationImagePainter", p))
	return p
}

func (p *decorationImagePainter) Paint(canvas ui.Canvas, rect ui.Rect, clipPath *ui.Path,
	configuration ImageConfiguration, opts ...DecorationImagePainterPaintOptions) {
	blend, blendMode := firstPaintOptions(opts).resolve()
	flipHorizontally := false
	if p.details.MatchTextDirection() {
		if foundation.KDebugMode && configuration.TextDirection == nil {
			errStyle := foundation.DiagnosticsTreeStyleErrorProperty
			panic(foundation.NewFlutterErrorFromParts([]foundation.DiagnosticsNode{
				foundation.NewErrorSummary("DecorationImage.matchTextDirection can only be used when a TextDirection is available."),
				foundation.NewErrorDescription("When DecorationImagePainter.paint() was called, there was no text direction provided " +
					"in the ImageConfiguration object to match."),
				foundation.NewDiagnosticsProperty("The DecorationImage was", p.details,
					foundation.DiagnosticsPropertyOptions{Style: &errStyle}),
				foundation.NewDiagnosticsProperty("The ImageConfiguration was", configuration,
					foundation.DiagnosticsPropertyOptions{Style: &errStyle}),
			}))
		}
		if configuration.TextDirection != nil && *configuration.TextDirection == ui.TextDirectionRtl {
			flipHorizontally = true
		}
	}

	if p.image == nil && !isNilValue(p.details.Placeholder()) {
		newPlaceholderStream := p.details.Placeholder().Resolve(configuration)
		if p.placeholderStr == nil || !sameObject(newPlaceholderStream.Key(), p.placeholderStr.Key()) {
			listener := NewImageStreamListener(p.handlePlaceholderImage, ImageStreamListenerOptions{OnError: p.details.OnError()})
			if p.placeholderStr != nil {
				p.placeholderStr.RemoveListener(listener)
			}
			p.placeholderStr = newPlaceholderStream
			p.placeholderStr.AddListener(listener)
		}
	}

	newImageStream := p.details.Image().Resolve(configuration)
	if p.imageStream == nil || !sameObject(newImageStream.Key(), p.imageStream.Key()) {
		listener := NewImageStreamListener(p.handleImage, ImageStreamListenerOptions{OnError: p.details.OnError()})
		if p.imageStream != nil {
			p.imageStream.RemoveListener(listener)
		}
		p.imageStream = newImageStream
		p.imageStream.AddListener(listener)
	}
	imageInfo := p.image
	if imageInfo == nil {
		imageInfo = p.placeholderImage
	}
	if imageInfo == nil {
		return
	}

	if clipPath != nil {
		canvas.Save()
		canvas.ClipPath(clipPath)
	}

	alignment := p.details.Alignment().Resolve(configuration.TextDirection)
	scale := float64(p.details.Scale() * imageInfo.Scale)
	opacity := float64(p.details.Opacity() * blend)
	repeat := p.details.Repeat()
	fq := p.details.FilterQuality()
	PaintImage(canvas, rect, imageInfo.Image, PaintImageOptions{DebugImageLabel: imageInfo.DebugLabel, Scale: &scale,
		ColorFilter: p.details.ColorFilter(), Fit: p.details.Fit(), Alignment: alignment,
		CenterSlice: p.details.CenterSlice(), Repeat: &repeat, FlipHorizontally: flipHorizontally, Opacity: &opacity,
		FilterQuality: &fq, InvertColors: p.details.InvertColors(), IsAntiAlias: p.details.IsAntiAlias(),
		BlendMode: &blendMode})

	if clipPath != nil {
		canvas.Restore()
	}
}

func imageInfoEqual(a, b *ImageInfo) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return a.Self().Equal(b)
}

func (p *decorationImagePainter) handleImage(value *ImageInfo, synchronousCall bool) {
	if imageInfoEqual(p.image, value) {
		return
	}
	if p.image != nil && p.image.IsCloneOf(value) {
		value.Dispose()
		return
	}
	if p.image != nil {
		p.image.Dispose()
	}
	p.image = value
	p.disposePlaceholder()
	if !synchronousCall {
		p.onChanged()
	}
}

func (p *decorationImagePainter) handlePlaceholderImage(value *ImageInfo, synchronousCall bool) {
	if p.image != nil {
		value.Dispose()
		return
	}
	if imageInfoEqual(p.placeholderImage, value) {
		return
	}
	if p.placeholderImage != nil && p.placeholderImage.IsCloneOf(value) {
		value.Dispose()
		return
	}
	if p.placeholderImage != nil {
		p.placeholderImage.Dispose()
	}
	p.placeholderImage = value
	if !synchronousCall {
		p.onChanged()
	}
}

func (p *decorationImagePainter) disposePlaceholder() {
	if p.placeholderStr != nil {
		p.placeholderStr.RemoveListener(NewImageStreamListener(p.handlePlaceholderImage,
			ImageStreamListenerOptions{OnError: p.details.OnError()}))
	}
	p.placeholderStr = nil
	if p.placeholderImage != nil {
		p.placeholderImage.Dispose()
	}
	p.placeholderImage = nil
}

func (p *decorationImagePainter) Dispose() {
	dartAssert(foundation.DebugMaybeDispatchDisposed(p))
	if p.imageStream != nil {
		p.imageStream.RemoveListener(NewImageStreamListener(p.handleImage,
			ImageStreamListenerOptions{OnError: p.details.OnError()}))
	}
	if p.image != nil {
		p.image.Dispose()
	}
	p.image = nil
	p.disposePlaceholder()
}

func (p *decorationImagePainter) String() string {
	var image string
	if p.image == nil {
		image = "null"
	} else {
		image = p.image.Self().String()
	}
	return runtimeTypeName("_DecorationImagePainter", "DecorationImagePainter") + "(stream: " + stringOf(p.imageStream) +
		", image: " + image + ") for " + p.details.String()
}

// The image sizes painted in the current frame (by source, in insertion
// order) and in the last one.
var (
	pendingImageSizeInfo     = map[string]ImageSizeInfo{}
	pendingImageSizeInfoKeys []string
	lastFrameImageSizeInfo   = map[ImageSizeInfo]struct{}{}
)

// DebugFlushLastFrameImageSizeInfo flushes the record of the image sizes
// painted in the last frame (debug builds; for tests).
func DebugFlushLastFrameImageSizeInfo() {
	if foundation.KDebugMode {
		lastFrameImageSizeInfo = map[ImageSizeInfo]struct{}{}
	}
}

// PaintImageOptions are paintImage's optional named parameters (nil
// Scale/Opacity: 1.0; the zero Alignment is Alignment.center; nil Repeat:
// noRepeat; nil FilterQuality: medium; nil BlendMode: srcOver; nil
// DebugImageLabel, ColorFilter, Fit, CenterSlice: null).
type PaintImageOptions struct {
	DebugImageLabel  *string
	Scale            *float64
	Opacity          *float64
	ColorFilter      *ui.ColorFilter
	Fit              *BoxFit
	Alignment        Alignment
	CenterSlice      *ui.Rect
	Repeat           *ImageRepeat
	FlipHorizontally bool
	InvertColors     bool
	FilterQuality    *ui.FilterQuality
	IsAntiAlias      bool
	BlendMode        *ui.BlendMode
}

var invertMatrix = []float64{-1, 0, 0, 0, 255, 0, -1, 0, 0, 255, 0, 0, -1, 0, 255, 0, 0, 0, 1, 0}

// PaintImage paints an image into the given rectangle on the canvas
// (paintImage; the required named canvas, rect and image are positional).
func PaintImage(canvas ui.Canvas, rect ui.Rect, image *ui.Image, opts ...PaintImageOptions) {
	var o PaintImageOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	scale, opacity := 1.0, 1.0
	if o.Scale != nil {
		scale = *o.Scale
	}
	if o.Opacity != nil {
		opacity = *o.Opacity
	}
	repeat := ImageRepeatNoRepeat
	if o.Repeat != nil {
		repeat = *o.Repeat
	}
	filterQuality := ui.FilterQualityMedium
	if o.FilterQuality != nil {
		filterQuality = *o.FilterQuality
	}
	blendMode := ui.BlendModeSrcOver
	if o.BlendMode != nil {
		blendMode = *o.BlendMode
	}
	alignment, centerSlice := o.Alignment, o.CenterSlice

	if st := image.DebugGetOpenHandleStackTraces(); st != nil && len(st) == 0 {
		panic(dart.NewAssertionError("Cannot paint an image that is disposed.\n" +
			"The caller of paintImage is expected to wait to dispose the image until " +
			"after painting has completed."))
	}
	if rect.IsEmpty() {
		return
	}
	outputSize := rect.Size()
	inputSize := ui.NewSize(float64(image.Width), float64(image.Height))
	var sliceBorder ui.Offset
	sliceFits := true
	if centerSlice != nil {
		sliceBorder = inputSize.Div(scale).Sub(centerSlice.Size()).(ui.Offset)
		outputSize = outputSize.Sub(sliceBorder).(ui.Size)
		inputSize = inputSize.Sub(sliceBorder.Mul(scale)).(ui.Size)
		sliceFits = outputSize.Width > 0.0 && outputSize.Height > 0.0
	}
	var fit BoxFit
	switch {
	case o.Fit != nil:
		fit = *o.Fit
	case centerSlice == nil:
		fit = BoxFitScaleDown
	default:
		fit = BoxFitFill
	}
	if !(centerSlice == nil || (fit != BoxFitNone && fit != BoxFitCover)) {
		panic(dart.NewAssertionError(nil))
	}
	fittedSizes := ApplyBoxFit(fit, inputSize.Div(scale), outputSize)
	sourceSize := fittedSizes.Source.Mul(scale)
	destinationSize := fittedSizes.Destination
	if centerSlice != nil {
		outputSize = outputSize.Add(sliceBorder)
		destinationSize = destinationSize.Add(sliceBorder)
		sourceInputDelta := sourceSize.Sub(inputSize).(ui.Offset)
		if !(!sliceFits || (math.Abs(sourceInputDelta.Dx) <= foundation.PrecisionErrorTolerance &&
			math.Abs(sourceInputDelta.Dy) <= foundation.PrecisionErrorTolerance)) {
			panic(dart.NewAssertionError("centerSlice was used with a BoxFit that does not guarantee that the image is fully visible."))
		}
	}
	if repeat != ImageRepeatNoRepeat && destinationSize == outputSize {
		// There's no need to repeat the image because we're exactly filling
		// the output rect with the image.
		repeat = ImageRepeatNoRepeat
	}
	paint := ui.NewPaint()
	paint.SetIsAntiAlias(o.IsAntiAlias)
	if o.ColorFilter != nil {
		paint.SetColorFilter(o.ColorFilter)
	}
	paint.SetColor(ui.NewColorFromRGBO(0, 0, 0, ui.ClampDouble(opacity, 0.0, 1.0)))
	paint.SetFilterQuality(filterQuality)
	paint.SetInvertColors(o.InvertColors)
	paint.SetBlendMode(blendMode)
	halfWidthDelta := (outputSize.Width - destinationSize.Width) / 2.0
	halfHeightDelta := (outputSize.Height - destinationSize.Height) / 2.0
	ax := alignment.X
	if o.FlipHorizontally {
		ax = -alignment.X
	}
	dx := halfWidthDelta + float64(ax*halfWidthDelta)
	dy := halfHeightDelta + float64(alignment.Y*halfHeightDelta)
	destinationPosition := rect.TopLeft().Translate(dx, dy)
	destinationRect := destinationPosition.And(destinationSize)

	// Set to true if we added a saveLayer to the canvas to invert/flip the
	// image.
	invertedCanvas := false
	if !foundation.KReleaseMode {
		// The only view flutter-go has is the implicit one (Dart folds over
		// every view's devicePixelRatio).
		maxDevicePixelRatio := 0.0
		if v := PaintingBindingInstance().PlatformDispatcher().ImplicitView(); v != nil {
			maxDevicePixelRatio = math.Max(maxDevicePixelRatio, v.DevicePixelRatio())
		}
		source := "<Unknown Image(" + itoa(image.Width) + "×" + itoa(image.Height) + ")>"
		if o.DebugImageLabel != nil {
			source = *o.DebugImageLabel
		}
		sizeInfo := ImageSizeInfo{Source: source, ImageSize: ui.NewSize(float64(image.Width), float64(image.Height)),
			DisplaySize: outputSize.Mul(maxDevicePixelRatio)}
		if foundation.KDebugMode && DebugInvertOversizedImages &&
			sizeInfo.DecodedSizeInBytes() > sizeInfo.DisplaySizeInBytes()+DebugImageOverheadAllowance {
			overheadInKilobytes := (sizeInfo.DecodedSizeInBytes() - sizeInfo.DisplaySizeInBytes()) / 1024
			outputWidth := dartToInt(sizeInfo.DisplaySize.Width)
			outputHeight := dartToInt(sizeInfo.DisplaySize.Height)
			label := "null"
			if o.DebugImageLabel != nil {
				label = *o.DebugImageLabel
			}
			lib := "painting library"
			foundation.FlutterErrorReportError(foundation.NewFlutterErrorDetails(
				"Image "+label+" has a display size of "+
					itoa(outputWidth)+"×"+itoa(outputHeight)+" but a decode size of "+
					itoa(image.Width)+"×"+itoa(image.Height)+", which uses an additional "+
					itoa(overheadInKilobytes)+"KB (assuming a device pixel ratio of "+
					dart.DoubleToString(maxDevicePixelRatio)+").\n\n"+
					"Consider resizing the asset ahead of time, supplying a cacheWidth "+
					"parameter of "+itoa(outputWidth)+", a cacheHeight parameter of "+
					itoa(outputHeight)+", or using a ResizeImage.",
				foundation.FlutterErrorDetailsOptions{Library: &lib,
					Context: foundation.NewErrorDescription("while painting an image")}))
			// Invert the colors of the canvas.
			layerPaint := ui.NewPaint()
			layerPaint.SetColorFilter(ui.NewColorFilterMatrix(invertMatrix))
			canvas.SaveLayer(&destinationRect, layerPaint)
			// Flip the canvas vertically.
			dy := -(rect.Top + rect.Height()/2.0)
			canvas.Translate(0.0, -dy)
			canvas.Scale(1.0, -1.0)
			canvas.Translate(0.0, dy)
			invertedCanvas = true
		}
		if _, ok := lastFrameImageSizeInfo[sizeInfo]; !ok {
			existing, found := pendingImageSizeInfo[sizeInfo.Source]
			if !found || existing.DisplaySizeInBytes() < sizeInfo.DisplaySizeInBytes() {
				if !found {
					pendingImageSizeInfoKeys = append(pendingImageSizeInfoKeys, sizeInfo.Source)
				}
				pendingImageSizeInfo[sizeInfo.Source] = sizeInfo
			}
			if DebugOnPaintImage != nil {
				DebugOnPaintImage(sizeInfo)
			}
			scheduler.SchedulerBindingInstance().AddPostFrameCallback(func(time.Duration) {
				lastFrameImageSizeInfo = map[ImageSizeInfo]struct{}{}
				for _, k := range pendingImageSizeInfoKeys {
					lastFrameImageSizeInfo[pendingImageSizeInfo[k]] = struct{}{}
				}
				if len(pendingImageSizeInfo) == 0 {
					return
				}
				event := map[string]any{}
				for _, k := range pendingImageSizeInfoKeys {
					event[k] = pendingImageSizeInfo[k].ToJson()
				}
				developer.PostEvent("Flutter.ImageSizesForFrame", event)
				pendingImageSizeInfo = map[string]ImageSizeInfo{}
				pendingImageSizeInfoKeys = nil
			}, scheduler.AddPostFrameCallbackOptions{DebugLabel: "paintImage.recordImageSizes"})
		}
	}

	needSave := centerSlice != nil || repeat != ImageRepeatNoRepeat || o.FlipHorizontally
	if needSave {
		canvas.Save()
	}
	if repeat != ImageRepeatNoRepeat {
		canvas.ClipRect(rect)
	}
	if o.FlipHorizontally {
		dx := -(rect.Left + rect.Width()/2.0)
		canvas.Translate(-dx, 0.0)
		canvas.Scale(-1.0, 1.0)
		canvas.Translate(dx, 0.0)
	}
	if centerSlice == nil {
		sourceRect := alignment.Inscribe(sourceSize, ui.OffsetZero.And(inputSize))
		if repeat == ImageRepeatNoRepeat {
			canvas.DrawImageRect(image, sourceRect, destinationRect, paint)
		} else {
			for _, tileRect := range generateImageTileRects(rect, destinationRect, repeat) {
				canvas.DrawImageRect(image, sourceRect, tileRect, paint)
			}
		}
	} else {
		canvas.Scale(1 / scale)
		if repeat == ImageRepeatNoRepeat {
			canvas.DrawImageNine(image, scaleRect(*centerSlice, scale), scaleRect(destinationRect, scale), paint)
		} else {
			for _, tileRect := range generateImageTileRects(rect, destinationRect, repeat) {
				canvas.DrawImageNine(image, scaleRect(*centerSlice, scale), scaleRect(tileRect, scale), paint)
			}
		}
	}
	if needSave {
		canvas.Restore()
	}
	if invertedCanvas {
		canvas.Restore()
	}
}

func itoa(i int) string { return strconv.Itoa(i) }

// dartToInt is double.toInt() (truncation; the values here are finite).
func dartToInt(v float64) int { return int(v) }

func generateImageTileRects(outputRect, fundamentalRect ui.Rect, repeat ImageRepeat) []ui.Rect {
	startX, startY, stopX, stopY := 0, 0, 0, 0
	strideX := fundamentalRect.Width()
	strideY := fundamentalRect.Height()
	if repeat == ImageRepeatRepeat || repeat == ImageRepeatRepeatX {
		startX = int(math.Floor((outputRect.Left - fundamentalRect.Left) / strideX))
		stopX = int(math.Ceil((outputRect.Right - fundamentalRect.Right) / strideX))
	}
	if repeat == ImageRepeatRepeat || repeat == ImageRepeatRepeatY {
		startY = int(math.Floor((outputRect.Top - fundamentalRect.Top) / strideY))
		stopY = int(math.Ceil((outputRect.Bottom - fundamentalRect.Bottom) / strideY))
	}
	var out []ui.Rect
	for i := startX; i <= stopX; i++ {
		for j := startY; j <= stopY; j++ {
			out = append(out, fundamentalRect.Shift(ui.NewOffset(float64(float64(i)*strideX), float64(float64(j)*strideY))))
		}
	}
	return out
}

func scaleRect(rect ui.Rect, scale float64) ui.Rect {
	return ui.NewRectFromLTRB(float64(rect.Left*scale), float64(rect.Top*scale), float64(rect.Right*scale), float64(rect.Bottom*scale))
}

// blendedDecorationImage is _BlendedDecorationImage: DecorationImage.lerp's
// result, painting a at 1 - t and b at t.
type blendedDecorationImage struct {
	a, b DecorationImage
	t    float64
}

func (d *blendedDecorationImage) pick() DecorationImage {
	if d.b != nil {
		return d.b
	}
	return d.a
}

func (d *blendedDecorationImage) Image() ImageProvider { return d.pick().Image() }

// Placeholder, OnError, ColorFilter, Fit and CenterSlice are Dart's
// `b?.x ?? a!.x`: b's value, unless b is null or its value is.
func (d *blendedDecorationImage) Placeholder() ImageProvider {
	if d.b != nil && !isNilValue(d.b.Placeholder()) {
		return d.b.Placeholder()
	}
	return d.a.Placeholder()
}
func (d *blendedDecorationImage) OnError() ImageErrorListener {
	if d.b != nil && d.b.OnError() != nil {
		return d.b.OnError()
	}
	return d.a.OnError()
}
func (d *blendedDecorationImage) ColorFilter() *ui.ColorFilter {
	if d.b != nil && d.b.ColorFilter() != nil {
		return d.b.ColorFilter()
	}
	return d.a.ColorFilter()
}
func (d *blendedDecorationImage) Fit() *BoxFit {
	if d.b != nil && d.b.Fit() != nil {
		return d.b.Fit()
	}
	return d.a.Fit()
}
func (d *blendedDecorationImage) Alignment() AlignmentGeometry { return d.pick().Alignment() }
func (d *blendedDecorationImage) CenterSlice() *ui.Rect {
	if d.b != nil && d.b.CenterSlice() != nil {
		return d.b.CenterSlice()
	}
	return d.a.CenterSlice()
}
func (d *blendedDecorationImage) Repeat() ImageRepeat             { return d.pick().Repeat() }
func (d *blendedDecorationImage) MatchTextDirection() bool        { return d.pick().MatchTextDirection() }
func (d *blendedDecorationImage) Scale() float64                  { return d.pick().Scale() }
func (d *blendedDecorationImage) Opacity() float64                { return d.pick().Opacity() }
func (d *blendedDecorationImage) FilterQuality() ui.FilterQuality { return d.pick().FilterQuality() }
func (d *blendedDecorationImage) InvertColors() bool              { return d.pick().InvertColors() }
func (d *blendedDecorationImage) IsAntiAlias() bool               { return d.pick().IsAntiAlias() }

func (d *blendedDecorationImage) CreatePainter(onChanged func()) DecorationImagePainter {
	p := &blendedDecorationImagePainter{t: d.t}
	if d.a != nil {
		p.a = d.a.CreatePainter(onChanged)
	}
	if d.b != nil {
		p.b = d.b.CreatePainter(onChanged)
	}
	dartAssert(foundation.DebugMaybeDispatchCreated("painting", "_BlendedDecorationImagePainter", p))
	return p
}

func decorationImagesEqual(a, b DecorationImage) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return a.Equal(b)
}

func (d *blendedDecorationImage) Equal(other DecorationImage) bool {
	o, ok := other.(*blendedDecorationImage)
	if !ok || o == nil {
		return false
	}
	return d == o || (decorationImagesEqual(o.a, d.a) && decorationImagesEqual(o.b, d.b) && o.t == d.t)
}

func (d *blendedDecorationImage) String() string {
	return runtimeTypeName("_BlendedDecorationImage", "_BlendedDecorationImage") + "(" + stringOf(d.a) + ", " + stringOf(d.b) +
		", " + dart.DoubleToString(d.t) + ")"
}

type blendedDecorationImagePainter struct {
	a, b DecorationImagePainter
	t    float64
}

func (p *blendedDecorationImagePainter) Paint(canvas ui.Canvas, rect ui.Rect, clipPath *ui.Path,
	configuration ImageConfiguration, opts ...DecorationImagePainterPaintOptions) {
	blend, blendMode := firstPaintOptions(opts).resolve()
	canvas.SaveLayer(nil, ui.NewPaint())
	if p.a != nil {
		ba := float64(blend * (1.0 - p.t))
		p.a.Paint(canvas, rect, clipPath, configuration, DecorationImagePainterPaintOptions{Blend: &ba, BlendMode: &blendMode})
	}
	if p.b != nil {
		bb := float64(blend * p.t)
		mode := blendMode
		if p.a != nil {
			mode = ui.BlendModePlus
		}
		p.b.Paint(canvas, rect, clipPath, configuration, DecorationImagePainterPaintOptions{Blend: &bb, BlendMode: &mode})
	}
	canvas.Restore()
}

func (p *blendedDecorationImagePainter) Dispose() {
	dartAssert(foundation.DebugMaybeDispatchDisposed(p))
	if p.a != nil {
		p.a.Dispose()
	}
	if p.b != nil {
		p.b.Dispose()
	}
}

func (p *blendedDecorationImagePainter) String() string {
	return runtimeTypeName("_BlendedDecorationImagePainter", "_BlendedDecorationImagePainter") + "(" + stringOf(p.a) + ", " +
		stringOf(p.b) + ", " + dart.DoubleToString(p.t) + ")"
}

// runtimeTypeName is objectRuntimeType for a type whose Dart name differs
// from its Go name (a private Dart class): debugName in debug builds,
// optimizedValue otherwise.
func runtimeTypeName(debugName, optimizedValue string) string {
	if foundation.KDebugMode {
		return debugName
	}
	return optimizedValue
}
