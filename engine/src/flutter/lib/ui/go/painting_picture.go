package ui

// Port of painting.dart's PictureRecorder, Picture, Vertices and Shadow
// (Flutter 3.50.1). A Picture holds the Impeller SDK display list the
// canvas recorded; Picture.toImage renders it offscreen with the engine's
// Impeller context and reads the pixels back (src/platform/metal), as the
// engine rasterizes a picture into an image.

import (
	"errors"
	"math"
	"strconv"

	"github.com/jonathanperis/flutter-go/src/async"
	"github.com/jonathanperis/flutter-go/src/dart"
	"github.com/jonathanperis/flutter-go/src/impeller"
	"github.com/jonathanperis/flutter-go/src/platform/metal"
)

// ---- PictureRecorder ----

// PictureRecorder records a Picture from the operations of a Canvas.
type PictureRecorder struct {
	canvas *nativeCanvas
	cull   dlRect
}

// NewPictureRecorder is PictureRecorder().
func NewPictureRecorder() *PictureRecorder { return &PictureRecorder{} }

// IsRecording is whether a Canvas is recording into this recorder.
func (r *PictureRecorder) IsRecording() bool { return r.canvas != nil }

// EndRecording finishes recording and returns the Picture.
func (r *PictureRecorder) EndRecording() *Picture {
	if r.canvas == nil {
		panic(dart.StateError{Message: "PictureRecorder did not start recording."})
	}
	c := r.canvas
	pic := &Picture{dl: c.b.Build(), ops: c.ops, cull: r.cull}
	c.b.Release()
	c.recorder = nil
	r.canvas = nil
	if PictureOnCreate != nil {
		PictureOnCreate(pic)
	}
	return pic
}

func (r *PictureRecorder) String() string {
	return "PictureRecorder(recording: " + strconv.FormatBool(r.IsRecording()) + ")"
}

// ---- Picture ----

// PictureEventCallback is called with a Picture when it is created or
// disposed.
type PictureEventCallback func(picture *Picture)

// PictureOnCreate and PictureOnDispose are Picture.onCreate and
// Picture.onDispose.
var PictureOnCreate, PictureOnDispose PictureEventCallback

// Picture is a sequence of recorded graphical operations.
type Picture struct {
	dl       *impeller.DisplayList
	ops      int
	cull     dlRect
	disposed bool
}

// Dispose releases the picture's resources.
func (p *Picture) Dispose() {
	dartAssert(!p.disposed, "")
	if debugMode {
		p.disposed = true
	}
	if PictureOnDispose != nil {
		PictureOnDispose(p)
	}
	if p.dl != nil {
		p.dl.Release()
		p.dl = nil
	}
}

// DebugDisposed is Picture.debugDisposed (debug builds; it panics with a
// StateError in release builds, as Dart throws).
func (p *Picture) DebugDisposed() bool {
	if !debugMode {
		panic(dart.StateError{Message: "Picture.debugDisposed is only available when asserts are enabled."})
	}
	return p.disposed
}

// ApproximateBytesUsed is Picture.approximateBytesUsed. ENGINE GAP: the
// SDK doesn't report a display list's size; this estimates the engine's
// DisplayList::bytes() + sizeof(Picture) from the number of recorded
// operations.
func (p *Picture) ApproximateBytesUsed() int {
	const pictureSize, opSize, displayListBase = 48, 32, 176
	return pictureSize + displayListBase + opSize*p.ops
}

func (p *Picture) String() string { return "Picture" }

// EnginePictureDisplayList is the Impeller display list a picture holds
// (the engine's sk_sp<DisplayList>), for the embedder to present: nil once
// the picture is disposed. The picture keeps ownership.
func EnginePictureDisplayList(p *Picture) *impeller.DisplayList { return p.dl }

// rasterize renders the picture into a width×height premultiplied RGBA8
// image (DoRasterizeToImage).
func (p *Picture) rasterize(width, height int) (*Image, error) {
	ctx := engineContext()
	if ctx == nil {
		return nil, errors.New("no GPU context")
	}
	b := impeller.NewDisplayListBuilder()
	if p.dl != nil {
		b.DrawDisplayList(p.dl, 1)
	}
	dl := b.Build()
	b.Release()
	defer dl.Release()
	px, err := metal.RenderOffscreen(ctx.Handle(), dl.Handle(), width, height)
	if err != nil {
		return nil, err
	}
	return newImageFromRGBA8(width, height, px), nil
}

// ToImage is Picture.toImage: the picture rendered into a width×height
// image. The future completes in a later event of the current loop (the
// engine completes it from the raster thread), with an error when the
// picture can't be rasterized.
func (p *Picture) ToImage(width, height int) async.Future[*Image] {
	dartAssert(!p.disposed, "")
	if width <= 0 || height <= 0 {
		panic(dart.NewException("Invalid image dimensions."))
	}
	c := async.NewSyncCompleter[*Image]()
	img, err := p.rasterize(width, height)
	async.Current().Post(func() {
		if err != nil || img == nil {
			c.CompleteError(dart.NewException("operation failed"))
			return
		}
		c.Complete(img)
	})
	return c.Future()
}

// PictureToImageSyncOptions are toImageSync's named arguments.
type PictureToImageSyncOptions struct {
	TargetFormat TargetPixelFormat // default dontCare
}

// ToImageSync is Picture.toImageSync. ENGINE GAP: the engine defers the
// rasterization (the image is a texture filled later); here the picture is
// rendered now, and targetFormat is ignored (the SDK renders RGBA8).
func (p *Picture) ToImageSync(width, height int, opts ...PictureToImageSyncOptions) *Image {
	dartAssert(!p.disposed, "")
	if width <= 0 || height <= 0 {
		panic(dart.NewException("Invalid image dimensions."))
	}
	img, err := p.rasterize(width, height)
	if err != nil {
		panic(dart.NewException(err.Error()))
	}
	return img
}

// PictureRasterizationException is thrown when drawing an image fails to
// rasterize (the image's texture is gone).
type PictureRasterizationException struct {
	Message string
	Stack   dart.StackTrace
}

func (e PictureRasterizationException) Error() string {
	s := "Failed to rasterize a picture: " + e.Message + "."
	if e.Stack != nil {
		s += "\nThe callstack when the image was created was:\n" + e.Stack.String()
	}
	return s
}

// ---- Vertices ----

// Vertices is a set of vertex data for Canvas.drawVertices.
type Vertices struct {
	mode      VertexMode
	positions []float32
	texCoords []float32
	colors    []int32
	indices   []uint16
	disposed  bool
}

// VerticesOptions are Vertices' named arguments.
type VerticesOptions struct {
	Colors             []Color
	TextureCoordinates []Offset
	Indices            []int
}

// NewVertices is Vertices(mode, positions, colors:, textureCoordinates:,
// indices:).
func NewVertices(mode VertexMode, positions []Offset, opts ...VerticesOptions) *Vertices {
	var o VerticesOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	if o.Colors != nil && len(o.Colors) != len(positions) {
		panic(argumentError(`"positions" and "colors" lengths must match.`))
	}
	if o.TextureCoordinates != nil && len(o.TextureCoordinates) != len(positions) {
		panic(argumentError(`"positions" and "textureCoordinates" lengths must match.`))
	}
	if debugMode && o.Indices != nil {
		for i, idx := range o.Indices {
			if idx >= len(positions) {
				panic(argumentError(`"indices" values must be valid indices in the positions list (i.e. numbers in the range 0..` +
					strconv.Itoa(len(positions)-1) + `), but indices[` + strconv.Itoa(i) + `] is ` + strconv.Itoa(idx) +
					`, which is too big.`))
			}
		}
	}
	enc := func(pts []Offset) []float32 {
		out := make([]float32, 0, 2*len(pts))
		for _, p := range pts {
			offsetIsValid(p)
			out = append(out, float32(p.Dx), float32(p.Dy))
		}
		return out
	}
	v := &Vertices{mode: mode, positions: enc(positions)}
	if o.TextureCoordinates != nil {
		v.texCoords = enc(o.TextureCoordinates)
	}
	if o.Colors != nil {
		v.colors = make([]int32, len(o.Colors))
		for i, c := range o.Colors {
			v.colors[i] = int32(uint32(c.Value()))
		}
	}
	if o.Indices != nil {
		v.indices = make([]uint16, len(o.Indices))
		for i, idx := range o.Indices {
			v.indices[i] = uint16(idx) // Uint16List.fromList truncates
		}
	}
	return v
}

// VerticesRawOptions are Vertices.raw's named arguments.
type VerticesRawOptions struct {
	Colors             []int32
	TextureCoordinates []float32
	Indices            []uint16
}

// NewVerticesRaw is Vertices.raw(mode, positions, colors:,
// textureCoordinates:, indices:).
func NewVerticesRaw(mode VertexMode, positions []float32, opts ...VerticesRawOptions) *Vertices {
	var o VerticesRawOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	if len(positions)%2 != 0 {
		panic(argumentError(`"positions" must have an even number of entries (each coordinate is an x,y pair).`))
	}
	if o.Colors != nil && len(o.Colors)*2 != len(positions) {
		panic(argumentError(`"positions" and "colors" lengths must match.`))
	}
	if o.TextureCoordinates != nil && len(o.TextureCoordinates) != len(positions) {
		panic(argumentError(`"positions" and "textureCoordinates" lengths must match.`))
	}
	if debugMode && o.Indices != nil {
		for i, idx := range o.Indices {
			if int(idx)*2 >= len(positions) {
				panic(argumentError(`"indices" values must be valid indices in the positions list (i.e. numbers in the range 0..` +
					strconv.Itoa(len(positions)/2-1) + `), but indices[` + strconv.Itoa(i) + `] is ` + strconv.Itoa(int(idx)) +
					`, which is too big.`))
			}
		}
	}
	return &Vertices{mode: mode, positions: positions, texCoords: o.TextureCoordinates, colors: o.Colors, indices: o.Indices}
}

// Dispose releases the vertices.
func (v *Vertices) Dispose() {
	dartAssert(!v.disposed, "")
	if debugMode {
		v.disposed = true
	}
}

// DebugDisposed is Vertices.debugDisposed (debug builds only).
func (v *Vertices) DebugDisposed() bool {
	if !debugMode {
		panic(dart.StateError{Message: "Vertices.debugDisposed is only available when asserts are enabled."})
	}
	return v.disposed
}

// triangles expands the vertices into triangles (x0, y0, x1, y1, x2, y2)
// by mode, through the indices when there are some (DlVertices).
func (v *Vertices) triangles() [][6]float32 {
	n := len(v.positions) / 2
	idx := func(i int) int { return i }
	count := n
	if len(v.indices) > 0 {
		idx = func(i int) int { return int(v.indices[i]) }
		count = len(v.indices)
	}
	at := func(i int) (float32, float32) {
		j := idx(i)
		if j < 0 || j >= n {
			return 0, 0
		}
		return v.positions[2*j], v.positions[2*j+1]
	}
	tri := func(a, b, c int) [6]float32 {
		ax, ay := at(a)
		bx, by := at(b)
		cx, cy := at(c)
		return [6]float32{ax, ay, bx, by, cx, cy}
	}
	var out [][6]float32
	switch v.mode {
	case VertexModeTriangles:
		for i := 0; i+2 < count; i += 3 {
			out = append(out, tri(i, i+1, i+2))
		}
	case VertexModeTriangleStrip:
		for i := 0; i+2 < count; i++ {
			out = append(out, tri(i, i+1, i+2))
		}
	case VertexModeTriangleFan:
		for i := 1; i+1 < count; i++ {
			out = append(out, tri(0, i, i+1))
		}
	}
	return out
}

// ---- Shadow ----

// Shadow is a single shadow (used by text styles and box shadows).
// Comparable: == is Dart's ==.
type Shadow struct {
	Color      Color
	Offset     Offset
	BlurRadius float64
}

// ShadowOptions are Shadow's named arguments.
type ShadowOptions struct {
	Color      *Color // nil: opaque black
	Offset     Offset
	BlurRadius float64
}

// NewShadow is Shadow(color:, offset:, blurRadius:).
func NewShadow(opts ...ShadowOptions) Shadow {
	var o ShadowOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	c := NewColor(0xFF000000)
	if o.Color != nil {
		c = *o.Color
	}
	dartAssert(o.BlurRadius >= 0.0, "Text shadow blur radius should be non-negative.")
	return Shadow{Color: c, Offset: o.Offset, BlurRadius: o.BlurRadius}
}

// ShadowConvertRadiusToSigma is Shadow.convertRadiusToSigma.
func ShadowConvertRadiusToSigma(radius float64) float64 {
	if radius > 0 {
		return float64(radius*0.57735) + 0.5 // no FMA, as in Dart
	}
	return 0
}

// BlurSigma is the standard deviation of the shadow's Gaussian blur.
func (s Shadow) BlurSigma() float64 { return ShadowConvertRadiusToSigma(s.BlurRadius) }

// ToPaint is a paint that draws the shadow (color and blur mask filter).
func (s Shadow) ToPaint() *Paint {
	p := NewPaint()
	p.SetColor(s.Color)
	m := NewMaskFilterBlur(BlurStyleNormal, s.BlurSigma())
	p.SetMaskFilter(&m)
	return p
}

// Scale is the shadow with offset and blur radius scaled by factor.
func (s Shadow) Scale(factor float64) Shadow {
	return NewShadow(ShadowOptions{Color: &s.Color, Offset: s.Offset.Mul(factor), BlurRadius: s.BlurRadius * factor})
}

// ShadowLerp is Shadow.lerp (nil is null).
func ShadowLerp(a, b *Shadow, t float64) *Shadow {
	if b == nil {
		if a == nil {
			return nil
		}
		r := a.Scale(1.0 - t)
		return &r
	}
	if a == nil {
		r := b.Scale(t)
		return &r
	}
	c := ColorLerp(&a.Color, &b.Color, t)
	o := OffsetLerp(&a.Offset, &b.Offset, t)
	r := NewShadow(ShadowOptions{Color: c, Offset: *o, BlurRadius: lerpDouble(a.BlurRadius, b.BlurRadius, t)})
	return &r
}

// ShadowLerpList is Shadow.lerpList (nil is null).
func ShadowLerpList(a, b []Shadow, t float64) []Shadow {
	if a == nil && b == nil {
		return nil
	}
	commonLength := int(math.Min(float64(len(a)), float64(len(b))))
	result := []Shadow{}
	for i := 0; i < commonLength; i++ {
		result = append(result, *ShadowLerp(&a[i], &b[i], t))
	}
	for i := commonLength; i < len(a); i++ {
		result = append(result, a[i].Scale(1.0-t))
	}
	for i := commonLength; i < len(b); i++ {
		result = append(result, b[i].Scale(t))
	}
	return result
}

func (s Shadow) String() string {
	return "TextShadow(" + s.Color.String() + ", " + s.Offset.String() + ", " + dart.DoubleToString(s.BlurRadius) + ")"
}
