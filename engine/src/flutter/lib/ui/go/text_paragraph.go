package ui

// Port of engine/src/flutter/lib/ui/text.dart (Flutter 3.50.1), part 4:
// Paragraph (_NativeParagraph), ParagraphBuilder (_NativeParagraphBuilder)
// and loadFontFromList, over the engine's text stack in the Impeller SDK
// (src/impeller/typography_bridge.cc ports lib/ui/text's natives).

import (
	"math"
	"sync"
	"unicode/utf16"

	"github.com/jonathanperis/flutter-go/src/async"
	"github.com/jonathanperis/flutter-go/src/dart"
	"github.com/jonathanperis/flutter-go/src/impeller"
)

// ---- The engine's font collection -----------------------------------------

var (
	engineFontsMu   sync.Mutex
	engineFonts     *impeller.TxtFontCollection
	engineTestFonts bool
)

// EngineUseTestFonts makes the engine lay out text with flutter_tester's
// test fonts (its --use-test-fonts, which `flutter test` passes): FlutterTest
// for every family but "Ahem" and "Cough", no font fallback. flutter_test's
// binding calls it; call it before any text is built (fonts loaded before
// are dropped).
func EngineUseTestFonts() {
	engineFontsMu.Lock()
	defer engineFontsMu.Unlock()
	if !engineTestFonts {
		engineTestFonts = true
		engineFonts = nil
	}
}

// engineFontCollection is the engine's FontCollection (the platform
// configuration client's), created on first use.
func engineFontCollection() *impeller.TxtFontCollection {
	engineFontsMu.Lock()
	defer engineFontsMu.Unlock()
	if engineFonts == nil {
		var test []impeller.TestFont
		if engineTestFonts {
			for i, name := range testFontFamilyNames {
				test = append(test, impeller.TestFont{Family: name, Data: []byte(testFontData[i])})
			}
		}
		engineFonts = impeller.NewTxtFontCollection(test)
	}
	return engineFonts
}

// ---- Paragraph ---------------------------------------------------------------

// Paragraph is a paragraph of text: laid out with Layout, then queried and
// drawn with Canvas.DrawParagraph. Build one with a ParagraphBuilder.
type Paragraph interface {
	// Width is the amount of horizontal space this paragraph occupies.
	Width() float64
	// Height is the amount of vertical space this paragraph occupies.
	Height() float64
	// LongestLine is the distance from the left edge of the leftmost glyph
	// to the right edge of the rightmost glyph.
	LongestLine() float64
	// MinIntrinsicWidth is the minimum width beyond which the height
	// doesn't decrease.
	MinIntrinsicWidth() float64
	// MaxIntrinsicWidth is the width beyond which the paragraph's height
	// doesn't decrease.
	MaxIntrinsicWidth() float64
	// AlphabeticBaseline is the distance from the top to the alphabetic
	// baseline of the first line.
	AlphabeticBaseline() float64
	// IdeographicBaseline is the distance from the top to the ideographic
	// baseline of the first line.
	IdeographicBaseline() float64
	// DidExceedMaxLines reports whether maxLines truncated the paragraph.
	DidExceedMaxLines() bool
	// Layout computes the size and position of each glyph.
	Layout(constraints ParagraphConstraints)
	// GetBoxesForRange is the boxes enclosing the given UTF-16 range.
	GetBoxesForRange(start, end int, opts ...GetBoxesForRangeOptions) []TextBox
	// GetBoxesForPlaceholders is the boxes of the placeholders, in order.
	GetBoxesForPlaceholders() []TextBox
	// GetPositionForOffset is the text position closest to the offset.
	GetPositionForOffset(offset Offset) TextPosition
	// GetClosestGlyphInfoForOffset is the glyph closest to the offset (nil
	// when the paragraph is empty).
	GetClosestGlyphInfoForOffset(offset Offset) *GlyphInfo
	// GetGlyphInfoAt is the glyph at the UTF-16 offset (nil when out of
	// range or not visible).
	GetGlyphInfoAt(codeUnitOffset int) *GlyphInfo
	// GetWordBoundary is the word around the position (UAX #29).
	GetWordBoundary(position TextPosition) TextRange
	// GetLineBoundary is the line around the position.
	GetLineBoundary(position TextPosition) TextRange
	// ComputeLineMetrics is the metrics of each line.
	ComputeLineMetrics() []LineMetrics
	// GetLineMetricsAt is the metrics of a line (nil when out of range).
	GetLineMetricsAt(lineNumber int) *LineMetrics
	// NumberOfLines is the number of visual lines.
	NumberOfLines() int
	// GetLineNumberAt is the line holding the UTF-16 offset (nil when out
	// of range).
	GetLineNumberAt(codeUnitOffset int) *int
	// Dispose releases the paragraph's resources.
	Dispose()
	// DebugDisposed reports whether Dispose was called.
	DebugDisposed() bool
	String() string
}

// GetBoxesForRangeOptions are getBoxesForRange's named arguments (the zero
// values are Dart's defaults: tight).
type GetBoxesForRangeOptions struct {
	BoxHeightStyle BoxHeightStyle
	BoxWidthStyle  BoxWidthStyle
}

// nativeParagraph is _NativeParagraph. p is nil once disposed (the native
// peer is cleared).
type nativeParagraph struct {
	p           *impeller.TxtParagraph
	needsLayout bool
	disposed    bool
}

// native is the engine paragraph; calling a native method after dispose
// throws, as the cleared Dart wrapper does.
func (p *nativeParagraph) native() *impeller.TxtParagraph {
	if p.p == nil {
		panic(errNativePeerCollected)
	}
	return p.p
}

func (p *nativeParagraph) metrics() impeller.TxtParagraphMetrics { return p.native().Metrics() }

func (p *nativeParagraph) Width() float64               { return p.metrics().Width }
func (p *nativeParagraph) Height() float64              { return p.metrics().Height }
func (p *nativeParagraph) LongestLine() float64         { return p.metrics().LongestLine }
func (p *nativeParagraph) MinIntrinsicWidth() float64   { return p.metrics().MinIntrinsicWidth }
func (p *nativeParagraph) MaxIntrinsicWidth() float64   { return p.metrics().MaxIntrinsicWidth }
func (p *nativeParagraph) AlphabeticBaseline() float64  { return p.metrics().AlphabeticBaseline }
func (p *nativeParagraph) IdeographicBaseline() float64 { return p.metrics().IdeographicBaseline }
func (p *nativeParagraph) DidExceedMaxLines() bool      { return p.metrics().DidExceedMaxLines }

func (p *nativeParagraph) Layout(constraints ParagraphConstraints) {
	p.native().Layout(constraints.Width)
	p.needsLayout = false
}

// uint32Arg is how an int reaches a Uint32 FFI parameter (mod 2^32).
func uint32Arg(v int) uint32 { return uint32(v) }

// decodeTextBoxes is _NativeParagraph._decodeTextBoxes.
func decodeTextBoxes(encoded []float32) []TextBox {
	count := len(encoded) / 5
	boxes := []TextBox{}
	position := 0
	for index := 0; index < count; index++ {
		boxes = append(boxes, NewTextBoxFromLTRBD(
			float64(encoded[position]),
			float64(encoded[position+1]),
			float64(encoded[position+2]),
			float64(encoded[position+3]),
			TextDirectionValues[int(encoded[position+4])],
		))
		position += 5
	}
	return boxes
}

func (p *nativeParagraph) GetBoxesForRange(start, end int, opts ...GetBoxesForRangeOptions) []TextBox {
	var o GetBoxesForRangeOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	return decodeTextBoxes(p.native().RectsForRange(uint32Arg(start), uint32Arg(end), uint32(o.BoxHeightStyle), uint32(o.BoxWidthStyle)))
}

func (p *nativeParagraph) GetBoxesForPlaceholders() []TextBox {
	return decodeTextBoxes(p.native().RectsForPlaceholders())
}

func (p *nativeParagraph) GetPositionForOffset(offset Offset) TextPosition {
	position, affinity := p.native().PositionForOffset(offset.Dx, offset.Dy)
	return NewTextPosition(position, TextAffinityValues[affinity])
}

func glyphInfoFrom(g impeller.TxtGlyphInfo, ok bool) *GlyphInfo {
	if !ok {
		return nil
	}
	info := newGlyphInfo(g.Left, g.Top, g.Right, g.Bottom, g.Start, g.End, g.IsLTR)
	return &info
}

func (p *nativeParagraph) GetGlyphInfoAt(codeUnitOffset int) *GlyphInfo {
	return glyphInfoFrom(p.native().GlyphInfoAt(uint32Arg(codeUnitOffset)))
}

func (p *nativeParagraph) GetClosestGlyphInfoForOffset(offset Offset) *GlyphInfo {
	return glyphInfoFrom(p.native().ClosestGlyphInfo(offset.Dx, offset.Dy))
}

func (p *nativeParagraph) GetWordBoundary(position TextPosition) TextRange {
	var characterPosition int
	switch position.Affinity {
	case TextAffinityUpstream:
		characterPosition = position.Offset - 1
	case TextAffinityDownstream:
		characterPosition = position.Offset
	}
	start, end := p.native().WordBoundary(uint32Arg(characterPosition))
	return NewTextRange(start, end)
}

func (p *nativeParagraph) GetLineBoundary(position TextPosition) TextRange {
	s, e := p.native().LineBoundary(uint32Arg(position.Offset))
	line := NewTextRange(s, e)
	ns, ne := p.native().LineBoundary(uint32Arg(position.Offset + 1))
	nextLine := NewTextRange(ns, ne)
	// If there is no next line, because we're at the end of the field,
	// return line.
	if !nextLine.IsValid() {
		return line
	}
	// _getLineBoundary only considers the offset and assumes that the
	// TextAffinity is upstream. In the case that TextPosition is just after a
	// word wrap (downstream), we need to return the line for the next offset.
	if position.Affinity == TextAffinityDownstream && line != nextLine &&
		position.Offset == line.End && line.End == nextLine.Start {
		return NewTextRange(ns, ne)
	}
	return line
}

func lineMetricsFrom(v []float64) LineMetrics {
	return LineMetrics{
		HardBreak:      v[0] != 0,
		Ascent:         v[1],
		Descent:        v[2],
		UnscaledAscent: v[3],
		Height:         v[4],
		Width:          v[5],
		Left:           v[6],
		Baseline:       v[7],
		LineNumber:     dartToInt(v[8]),
	}
}

// dartToInt is double.toInt (truncation).
func dartToInt(v float64) int { return int(math.Trunc(v)) }

func (p *nativeParagraph) ComputeLineMetrics() []LineMetrics {
	encoded := p.native().ComputeLineMetrics()
	count := len(encoded) / 9
	metrics := make([]LineMetrics, 0, count)
	for i := 0; i < count; i++ {
		metrics = append(metrics, lineMetricsFrom(encoded[i*9:i*9+9]))
	}
	return metrics
}

func (p *nativeParagraph) GetLineMetricsAt(lineNumber int) *LineMetrics {
	// Uint32 to the native int parameter.
	v, ok := p.native().LineMetricsAt(int32(uint32Arg(lineNumber)))
	if !ok {
		return nil
	}
	// The engine passes lineNumber as Dart_NewInteger(size_t).
	m := lineMetricsFrom(v[:])
	return &m
}

func (p *nativeParagraph) NumberOfLines() int { return int(uint32(p.native().NumberOfLines())) }

func (p *nativeParagraph) GetLineNumberAt(codeUnitOffset int) *int {
	lineNumber := int(p.native().LineNumberAt(uint64(uint32Arg(codeUnitOffset))))
	if lineNumber < 0 {
		return nil
	}
	return &lineNumber
}

func (p *nativeParagraph) Dispose() {
	textAssert(!p.disposed)
	p.disposed = true
	p.native().Dispose() // Paragraph::dispose: resets the paragraph, clears the wrapper
	p.p = nil
}

func (p *nativeParagraph) DebugDisposed() bool { return p.disposed }

func (p *nativeParagraph) String() string {
	if p.disposed && p.needsLayout {
		return "Paragraph(DISPOSED while dirty)"
	}
	if p.disposed && !p.needsLayout {
		return "Paragraph(DISPOSED)"
	}
	if p.needsLayout {
		return "Paragraph(dirty)"
	}
	return "Paragraph()"
}

// DrawParagraph is Canvas.drawParagraph: draws the laid-out paragraph with
// its top-left corner at offset.
func (c *nativeCanvas) DrawParagraph(paragraph Paragraph, offset Offset) {
	np := paragraph.(*nativeParagraph)
	textAssert(!np.DebugDisposed())
	offsetIsValid(offset)
	textAssert(!np.needsLayout)
	b := c.builder()
	np.native().Paint(b, offset.Dx, offset.Dy)
	c.ops++
}

// ---- ParagraphBuilder --------------------------------------------------------

// ParagraphBuilder builds a Paragraph from styled text and placeholders.
type ParagraphBuilder interface {
	// PlaceholderCount is the number of placeholders added.
	PlaceholderCount() int
	// PlaceholderScales is the scale of each placeholder added.
	PlaceholderScales() []float64
	// PushStyle applies style to the text added until the matching Pop.
	PushStyle(style *TextStyle)
	// Pop ends the effect of the last PushStyle.
	Pop()
	// AddText adds text with the current style.
	AddText(text string)
	// AddPlaceholder adds an inline placeholder box of the given size.
	AddPlaceholder(width, height float64, alignment PlaceholderAlignment, opts ...AddPlaceholderOptions)
	// Build returns the paragraph; the builder can't be used afterwards.
	Build() Paragraph
	String() string
}

// AddPlaceholderOptions are addPlaceholder's named arguments (Scale nil:
// 1.0; BaselineOffset nil: the height; Baseline nil: null, required for the
// baseline alignments).
type AddPlaceholderOptions struct {
	Scale          *float64
	BaselineOffset *float64
	Baseline       *TextBaseline
}

// nativeParagraphBuilder is _NativeParagraphBuilder.
type nativeParagraphBuilder struct {
	b                          *impeller.TxtParagraphBuilder
	placeholderCount           int
	placeholderScales          []float64
	defaultLeadingDistribution TextLeadingDistribution
}

// native is the engine builder; after build it is gone and native calls
// throw, as the cleared Dart wrapper does.
func (b *nativeParagraphBuilder) native() *impeller.TxtParagraphBuilder {
	if b.b == nil {
		panic(errNativePeerCollected)
	}
	return b.b
}

// encodeLocale is _NativeParagraphBuilder._encodeLocale.
func encodeLocale(l *Locale) string {
	if l == nil {
		return ""
	}
	return l.String()
}

// NewParagraphBuilder is ParagraphBuilder(style).
func NewParagraphBuilder(style *ParagraphStyle) ParagraphBuilder {
	pb := &nativeParagraphBuilder{defaultLeadingDistribution: style.leadingDistribution, placeholderScales: []float64{}}
	var strutFontFamilies []string
	var encodedStrutStyle []byte
	if s := style.strutStyle; s != nil && s.enabled() {
		strutFontFamilies = []string{}
		if s.fontFamily != nil {
			strutFontFamilies = append(strutFontFamilies, *s.fontFamily)
		}
		strutFontFamilies = append(strutFontFamilies, s.fontFamilyFallback...)
		leadingDistribution := style.leadingDistribution
		if s.leadingDistribution != nil {
			leadingDistribution = *s.leadingDistribution
		}
		// Dart sets the bit in the strut style's own buffer.
		bitmask := getInt32(s.buffer)
		bitmask |= int32(leadingDistribution) << 3
		putInt32(s.buffer, bitmask)
		encodedStrutStyle = s.buffer[:s.encodedLength]
	}
	fontFamily, fontSize, height, ellipsis := "", 0.0, 0.0, ""
	if style.fontFamily != nil {
		fontFamily = *style.fontFamily
	}
	if style.fontSize != nil {
		fontSize = *style.fontSize
	}
	if style.height != nil {
		height = *style.height
	}
	if style.ellipsis != nil {
		ellipsis = *style.ellipsis
	}
	pb.b = impeller.NewTxtParagraphBuilder(engineFontCollection(), impeller.TxtParagraphBuilderArgs{
		Encoded:         style.encoded,
		StrutData:       encodedStrutStyle,
		FontFamily:      fontFamily,
		StrutFamilies:   strutFontFamilies,
		FontSize:        fontSize,
		Height:          height,
		Ellipsis:        dart.UTF16CodeUnits(ellipsis),
		Locale:          encodeLocale(style.locale),
		ImpellerEnabled: true,
	})
	return pb
}

func (b *nativeParagraphBuilder) PlaceholderCount() int        { return b.placeholderCount }
func (b *nativeParagraphBuilder) PlaceholderScales() []float64 { return b.placeholderScales }

// encodeShadows is painting.dart's Shadow._encodeShadows: per shadow, the
// color XOR opaque black, then dx, dy and the blur sigma as float32.
func encodeShadows(shadows []Shadow) []byte {
	if shadows == nil {
		return []byte{}
	}
	const bytesPerShadow = 16
	data := make([]byte, len(shadows)*bytesPerShadow)
	for i, s := range shadows {
		o := i * bytesPerShadow
		putInt32(data[o:], int32(uint32(s.Color.Value())^0xFF000000))
		putInt32(data[o+4:], int32(math.Float32bits(float32(s.Offset.Dx))))
		putInt32(data[o+8:], int32(math.Float32bits(float32(s.Offset.Dy))))
		putInt32(data[o+12:], int32(math.Float32bits(float32(ShadowConvertRadiusToSigma(s.BlurRadius)))))
	}
	return data
}

func (b *nativeParagraphBuilder) PushStyle(style *TextStyle) {
	fullFontFamilies := []string{style.fontFamily}
	fullFontFamilies = append(fullFontFamilies, style.fontFamilyFallback...)
	encoded := style.encoded
	finalLeadingDistribution := b.defaultLeadingDistribution
	if style.leadingDistribution != nil {
		finalLeadingDistribution = *style.leadingDistribution
	}
	// Use the leading distribution from the paragraph's style if it's not
	// explicitly set in `style` (Dart sets the bit in the style's own
	// encoding).
	encoded[0] |= int32(finalLeadingDistribution) << 0
	var encodedFontFeatures []byte
	if style.fontFeatures != nil {
		encodedFontFeatures = make([]byte, len(style.fontFeatures)*fontFeatureEncodedSize)
		for i, f := range style.fontFeatures {
			f.encode(encodedFontFeatures[i*fontFeatureEncodedSize:])
		}
	}
	var encodedFontVariations []byte
	if style.fontVariations != nil {
		encodedFontVariations = make([]byte, len(style.fontVariations)*fontVariationEncodedSize)
		for i, v := range style.fontVariations {
			v.encode(encodedFontVariations[i*fontVariationEncodedSize:])
		}
	}
	orZero := func(v *float64) float64 {
		if v == nil {
			return 0
		}
		return *v
	}
	// Paint::paint(dl_paint, kDrawParagraphFlags, kDecal).
	var background, foreground *impeller.Paint
	if style.background != nil {
		background = style.background.impellerPaint(flagsDrawGeometry)
		defer background.Release()
	}
	if style.foreground != nil {
		foreground = style.foreground.impellerPaint(flagsDrawGeometry)
		defer foreground.Release()
	}
	b.native().PushStyle(impeller.TxtTextStyleArgs{
		Encoded:             encoded,
		FontFamilies:        fullFontFamilies,
		FontSize:            orZero(style.fontSize),
		LetterSpacing:       orZero(style.letterSpacing),
		WordSpacing:         orZero(style.wordSpacing),
		Height:              orZero(style.height),
		DecorationThickness: orZero(style.decorationThickness),
		Locale:              encodeLocale(style.locale),
		Background:          background,
		Foreground:          foreground,
		Shadows:             encodeShadows(style.shadows),
		FontFeatures:        encodedFontFeatures,
		FontVariations:      encodedFontVariations,
	})
}

func (b *nativeParagraphBuilder) Pop() { b.native().Pop() }

// wellFormedUTF16 is the engine's check (ICU's u_strToUTF8): no unpaired
// surrogate.
func wellFormedUTF16(units []uint16) bool {
	for i := 0; i < len(units); i++ {
		c := rune(units[i])
		switch {
		case utf16.IsSurrogate(c) && c < 0xDC00:
			if i+1 >= len(units) || units[i+1] < 0xDC00 || units[i+1] > 0xDFFF {
				return false
			}
			i++
		case utf16.IsSurrogate(c):
			return false
		}
	}
	return true
}

func (b *nativeParagraphBuilder) AddText(text string) {
	// ParagraphBuilder::addText.
	nb := b.native()
	units := dart.UTF16CodeUnits(text)
	if len(units) == 0 {
		return
	}
	if !wellFormedUTF16(units) {
		panic(argumentError("string is not well-formed UTF-16"))
	}
	nb.AddText(units)
}

func (b *nativeParagraphBuilder) AddPlaceholder(width, height float64, alignment PlaceholderAlignment, opts ...AddPlaceholderOptions) {
	var o AddPlaceholderOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	scale := 1.0
	if o.Scale != nil {
		scale = *o.Scale
	}
	// Require a baseline to be specified if using a baseline-based alignment.
	textAssert(!(alignment == PlaceholderAlignmentAboveBaseline ||
		alignment == PlaceholderAlignmentBelowBaseline ||
		alignment == PlaceholderAlignmentBaseline) || o.Baseline != nil)
	// Default the baselineOffset to height if null. This will place the
	// placeholder fully above the baseline, similar to
	// PlaceholderAlignment.aboveBaseline.
	baselineOffset := height
	if o.BaselineOffset != nil {
		baselineOffset = *o.BaselineOffset
	}
	baseline := TextBaselineAlphabetic
	if o.Baseline != nil {
		baseline = *o.Baseline
	}
	b.native().AddPlaceholder(width*scale, height*scale, uint32(alignment), baselineOffset*scale, uint32(baseline))
	b.placeholderCount++
	b.placeholderScales = append(b.placeholderScales, scale)
}

func (b *nativeParagraphBuilder) Build() Paragraph {
	// ParagraphBuilder::build: the builder's engine side is released.
	p := &nativeParagraph{p: b.native().Build(), needsLayout: true}
	b.b = nil
	return p
}

func (b *nativeParagraphBuilder) String() string { return "ParagraphBuilder" }

// ---- Fonts -------------------------------------------------------------------

// LoadFontFromListOptions are loadFontFromList's named arguments
// (FontFamily nil: the font's own family name).
type LoadFontFromListOptions struct {
	FontFamily *string
}

// fontChangeMessage is _fontChangeMessage.
var fontChangeMessage = dart.UTF8Encode(dart.JSONEncode(map[string]any{"type": "fontsChange"}))

// LoadFontFromList loads a font from a buffer and makes it available for
// rendering text; the platform is told with a fontsChange system message.
func LoadFontFromList(list []byte, opts ...LoadFontFromListOptions) async.Future[async.Void] {
	family := ""
	if len(opts) > 0 && opts[0].FontFamily != nil {
		family = *opts[0].FontFamily
	}
	f := futurize(func(callback func(async.Void, bool)) string {
		// FontCollection::LoadFontFromList: a buffer that isn't a font
		// registers nothing, and the callback is called all the same.
		engineFontCollection().LoadFont(list, family)
		callback(async.Void{}, true)
		return ""
	})
	return async.Then(f, func(async.Void) async.Void {
		sendFontChangeMessage()
		return async.Void{}
	})
}

// sendFontChangeMessage is _sendFontChangeMessage.
func sendFontChangeMessage() {
	const kSystemChannelName = "flutter/system"
	if handler := PlatformDispatcherInstance().OnPlatformMessage(); handler != nil {
		handler(kSystemChannelName, fontChangeMessage, func([]byte) {})
	} else {
		ChannelBuffersInstance().Push(kSystemChannelName, fontChangeMessage, func([]byte) {})
	}
}
