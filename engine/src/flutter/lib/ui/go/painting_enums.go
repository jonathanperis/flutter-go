package ui

// Port of painting.dart's drawing enums (Flutter 3.50.1): BlendMode,
// FilterQuality, StrokeCap, StrokeJoin, PaintingStyle, Clip, BlurStyle,
// TileMode, VertexMode, PointMode and ClipOp.

// BlendMode is an algorithm for blending pixels when drawing on a canvas.
// Values are Dart's indices (also Impeller's ImpellerBlendMode values).
type BlendMode int

const (
	BlendModeClear BlendMode = iota
	BlendModeSrc
	BlendModeDst
	BlendModeSrcOver
	BlendModeDstOver
	BlendModeSrcIn
	BlendModeDstIn
	BlendModeSrcOut
	BlendModeDstOut
	BlendModeSrcATop
	BlendModeDstATop
	BlendModeXor
	BlendModePlus
	BlendModeModulate
	BlendModeScreen // The last coeff mode.
	BlendModeOverlay
	BlendModeDarken
	BlendModeLighten
	BlendModeColorDodge
	BlendModeColorBurn
	BlendModeHardLight
	BlendModeSoftLight
	BlendModeDifference
	BlendModeExclusion
	BlendModeMultiply // The last separable mode.
	BlendModeHue
	BlendModeSaturation
	BlendModeColor
	BlendModeLuminosity
)

var blendModeNames = [...]string{"clear", "src", "dst", "srcOver", "dstOver", "srcIn", "dstIn",
	"srcOut", "dstOut", "srcATop", "dstATop", "xor", "plus", "modulate", "screen", "overlay",
	"darken", "lighten", "colorDodge", "colorBurn", "hardLight", "softLight", "difference",
	"exclusion", "multiply", "hue", "saturation", "color", "luminosity"}

// BlendModeValues is BlendMode.values.
func BlendModeValues() []BlendMode {
	v := make([]BlendMode, len(blendModeNames))
	for i := range v {
		v[i] = BlendMode(i)
	}
	return v
}

func (m BlendMode) Name() string   { return blendModeNames[m] }
func (m BlendMode) String() string { return "BlendMode." + m.Name() }

// FilterQuality is the quality of sampling an image (none, low, medium,
// high).
type FilterQuality int

const (
	FilterQualityNone FilterQuality = iota
	FilterQualityLow
	FilterQualityMedium
	FilterQualityHigh
)

func (q FilterQuality) Name() string   { return [...]string{"none", "low", "medium", "high"}[q] }
func (q FilterQuality) String() string { return "FilterQuality." + q.Name() }

// StrokeCap is how the ends of open lines are drawn.
type StrokeCap int

const (
	StrokeCapButt StrokeCap = iota
	StrokeCapRound
	StrokeCapSquare
)

func (c StrokeCap) Name() string   { return [...]string{"butt", "round", "square"}[c] }
func (c StrokeCap) String() string { return "StrokeCap." + c.Name() }

// StrokeJoin is how segments of a stroked path are joined.
type StrokeJoin int

const (
	StrokeJoinMiter StrokeJoin = iota
	StrokeJoinRound
	StrokeJoinBevel
)

func (j StrokeJoin) Name() string   { return [...]string{"miter", "round", "bevel"}[j] }
func (j StrokeJoin) String() string { return "StrokeJoin." + j.Name() }

// PaintingStyle is whether shapes are filled or stroked.
type PaintingStyle int

const (
	PaintingStyleFill PaintingStyle = iota
	PaintingStyleStroke
)

func (s PaintingStyle) Name() string   { return [...]string{"fill", "stroke"}[s] }
func (s PaintingStyle) String() string { return "PaintingStyle." + s.Name() }

// Clip is how content is clipped (none, hardEdge, antiAlias,
// antiAliasWithSaveLayer).
type Clip int

const (
	ClipNone Clip = iota
	ClipHardEdge
	ClipAntiAlias
	ClipAntiAliasWithSaveLayer
)

func (c Clip) Name() string {
	return [...]string{"none", "hardEdge", "antiAlias", "antiAliasWithSaveLayer"}[c]
}
func (c Clip) String() string { return "Clip." + c.Name() }

// BlurStyle is the style of a MaskFilter.blur.
type BlurStyle int

const (
	BlurStyleNormal BlurStyle = iota
	BlurStyleSolid
	BlurStyleOuter
	BlurStyleInner
)

func (s BlurStyle) Name() string   { return [...]string{"normal", "solid", "outer", "inner"}[s] }
func (s BlurStyle) String() string { return "BlurStyle." + s.Name() }

// TileMode is how a shader or filter treats the area outside its source.
type TileMode int

const (
	TileModeClamp TileMode = iota
	TileModeRepeated
	TileModeMirror
	TileModeDecal
)

func (m TileMode) Name() string   { return [...]string{"clamp", "repeated", "mirror", "decal"}[m] }
func (m TileMode) String() string { return "TileMode." + m.Name() }

// VertexMode is how a Vertices list is interpreted.
type VertexMode int

const (
	VertexModeTriangles VertexMode = iota
	VertexModeTriangleStrip
	VertexModeTriangleFan
)

func (m VertexMode) Name() string {
	return [...]string{"triangles", "triangleStrip", "triangleFan"}[m]
}
func (m VertexMode) String() string { return "VertexMode." + m.Name() }

// PointMode is how Canvas.drawPoints interprets its points.
type PointMode int

const (
	PointModePoints PointMode = iota
	PointModeLines
	PointModePolygon
)

func (m PointMode) Name() string   { return [...]string{"points", "lines", "polygon"}[m] }
func (m PointMode) String() string { return "PointMode." + m.Name() }

// ClipOp is how a new clip combines with the current one.
type ClipOp int

const (
	ClipOpDifference ClipOp = iota
	ClipOpIntersect
)

func (o ClipOp) Name() string   { return [...]string{"difference", "intersect"}[o] }
func (o ClipOp) String() string { return "ClipOp." + o.Name() }
