package flutter_test

// Port of packages/flutter_test/lib/src/recording_canvas.dart (Flutter
// 3.50.1): RecordedInvocation and TestRecordingCanvas.
//
// Not ported: TestRecordingPaintingContext (it implements rendering's
// PaintingContext, which is not ported yet; rendering still has its
// interim Canvas).
//
// Dart's Invocation (a method call captured by noSuchMethod) is the
// struct Invocation; a Symbol (#drawRect) is the method's Dart name
// ("drawRect").

import (
	"fmt"
	"reflect"
	"strings"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	"github.com/jonathanperis/flutter-go/src/dart"
)

// Invocation is a recorded method call: dart:core's Invocation for a
// method (TestRecordingCanvas records nothing else). MemberName is the
// Dart method name; NamedArguments keep the order they were given in.
type Invocation struct {
	MemberName          string
	PositionalArguments []any
	NamedArguments      []dart.MapEntry[string, any]
}

// IsMethod is Invocation.isMethod (always true for recorded canvas calls).
func (i Invocation) IsMethod() bool { return true }

// RecordedInvocation is an Invocation and the stack trace that led to it.
type RecordedInvocation struct {
	// Invocation is the method that was called and its arguments.
	Invocation Invocation
	// Stack is the stack trace at the time of the method call.
	Stack dart.StackTrace
}

// NewRecordedInvocation is RecordedInvocation(invocation, stack: stack).
func NewRecordedInvocation(invocation Invocation, stack dart.StackTrace) RecordedInvocation {
	return RecordedInvocation{Invocation: invocation, Stack: stack}
}

func (r RecordedInvocation) String() string { return describeInvocation(r.Invocation) }

// StackToStringOptions are StackToString's named parameters.
type StackToStringOptions struct {
	Indent string
}

// StackToString is the recorded stack trace, filtered by
// FlutterError.defaultStackFilter, each line prefixed by Indent.
func (r RecordedInvocation) StackToString(opts ...StackToStringOptions) string {
	indent := ""
	if len(opts) > 0 {
		indent = opts[0].Indent
	}
	stack := "null"
	if r.Stack != nil {
		stack = r.Stack.String()
	}
	lines := strings.Split(strings.TrimRight(stack, " \t\n\r"), "\n")
	var out []string
	for l := range foundation.FlutterErrorDefaultStackFilter(func(yield func(string) bool) {
		for _, l := range lines {
			if !yield(l) {
				return
			}
		}
	}) {
		out = append(out, l)
	}
	return indent + strings.Join(out, "\n"+indent)
}

// TestRecordingCanvas is a Canvas for tests that records its method calls
// (Invocations). Every Canvas method is recorded; those Dart's class
// leaves to noSuchMethod record their positional arguments and the named
// arguments actually given (non-nil options), as Dart's noSuchMethod
// does. Getters that Dart's noSuchMethod would answer with null
// (getTransform, getLocalClipBounds, getDestinationClipBounds) are
// recorded and return Go's zero values. Paints are recorded as copies
// (Paint.from).
type TestRecordingCanvas struct {
	// Invocations are all the calls made to this canvas, in order.
	Invocations []RecordedInvocation
	saveCount   int
}

var _ ui.Canvas = (*TestRecordingCanvas)(nil)

// NewTestRecordingCanvas is TestRecordingCanvas().
func NewTestRecordingCanvas() *TestRecordingCanvas { return &TestRecordingCanvas{} }

func (c *TestRecordingCanvas) record(name string, args ...any) {
	c.Invocations = append(c.Invocations, RecordedInvocation{
		Invocation: Invocation{MemberName: name, PositionalArguments: args},
		Stack:      dart.StackTraceCurrent(),
	})
}

func (c *TestRecordingCanvas) recordNamed(name string, named []dart.MapEntry[string, any], args ...any) {
	c.Invocations = append(c.Invocations, RecordedInvocation{
		Invocation: Invocation{MemberName: name, PositionalArguments: args, NamedArguments: named},
		Stack:      dart.StackTraceCurrent(),
	})
}

func paintCopy(p *ui.Paint) *ui.Paint { return ui.NewPaintFrom(p) }

// GetSaveCount is the number of saves minus restores.
func (c *TestRecordingCanvas) GetSaveCount() int { return c.saveCount }

// Save records save.
func (c *TestRecordingCanvas) Save() {
	c.saveCount++
	c.record("save")
}

// SaveLayer records saveLayer(bounds, paint) (nil bounds: null).
func (c *TestRecordingCanvas) SaveLayer(bounds *ui.Rect, paint *ui.Paint) {
	c.saveCount++
	var b any
	if bounds != nil {
		b = *bounds
	}
	c.record("saveLayer", b, paintCopy(paint))
}

// Restore records restore.
func (c *TestRecordingCanvas) Restore() {
	c.saveCount--
	if c.saveCount < 0 {
		panic(dart.NewAssertionError("_saveCount >= 0"))
	}
	c.record("restore")
}

// RestoreToCount records restoreToCount(count) (noSuchMethod in Dart: the
// save count is not changed).
func (c *TestRecordingCanvas) RestoreToCount(count int) { c.record("restoreToCount", count) }

// Translate records translate(dx, dy).
func (c *TestRecordingCanvas) Translate(dx, dy float64) { c.record("translate", dx, dy) }

// Scale records scale(sx, [sy]) with the arguments given.
func (c *TestRecordingCanvas) Scale(sx float64, sy ...float64) {
	if len(sy) > 0 {
		c.record("scale", sx, sy[0])
		return
	}
	c.record("scale", sx)
}

// Rotate records rotate(radians).
func (c *TestRecordingCanvas) Rotate(radians float64) { c.record("rotate", radians) }

// Skew records skew(sx, sy).
func (c *TestRecordingCanvas) Skew(sx, sy float64) { c.record("skew", sx, sy) }

// Transform records transform(matrix4).
func (c *TestRecordingCanvas) Transform(matrix4 []float64) { c.record("transform", matrix4) }

// GetTransform records getTransform and returns nil.
func (c *TestRecordingCanvas) GetTransform() []float64 {
	c.record("getTransform")
	return nil
}

func clipOptions(opts []ui.CanvasClipOptions) []dart.MapEntry[string, any] {
	if len(opts) > 0 && opts[0].DoAntiAlias != nil {
		return []dart.MapEntry[string, any]{{Key: "doAntiAlias", Value: *opts[0].DoAntiAlias}}
	}
	return nil
}

// ClipRect records clipRect(rect, {clipOp, doAntiAlias}).
func (c *TestRecordingCanvas) ClipRect(rect ui.Rect, opts ...ui.CanvasClipRectOptions) {
	var named []dart.MapEntry[string, any]
	if len(opts) > 0 {
		if opts[0].ClipOp != nil {
			named = append(named, dart.MapEntry[string, any]{Key: "clipOp", Value: *opts[0].ClipOp})
		}
		if opts[0].DoAntiAlias != nil {
			named = append(named, dart.MapEntry[string, any]{Key: "doAntiAlias", Value: *opts[0].DoAntiAlias})
		}
	}
	c.recordNamed("clipRect", named, rect)
}

// ClipRRect records clipRRect(rrect, {doAntiAlias}).
func (c *TestRecordingCanvas) ClipRRect(rrect ui.RRect, opts ...ui.CanvasClipOptions) {
	c.recordNamed("clipRRect", clipOptions(opts), rrect)
}

// ClipRSuperellipse records clipRSuperellipse(rsuperellipse, {doAntiAlias}).
func (c *TestRecordingCanvas) ClipRSuperellipse(rsuperellipse ui.RSuperellipse, opts ...ui.CanvasClipOptions) {
	c.recordNamed("clipRSuperellipse", clipOptions(opts), rsuperellipse)
}

// ClipPath records clipPath(path, {doAntiAlias}).
func (c *TestRecordingCanvas) ClipPath(path *ui.Path, opts ...ui.CanvasClipOptions) {
	c.recordNamed("clipPath", clipOptions(opts), path)
}

// GetLocalClipBounds records getLocalClipBounds and returns Rect.zero.
func (c *TestRecordingCanvas) GetLocalClipBounds() ui.Rect {
	c.record("getLocalClipBounds")
	return ui.Rect{}
}

// GetDestinationClipBounds records getDestinationClipBounds and returns
// Rect.zero.
func (c *TestRecordingCanvas) GetDestinationClipBounds() ui.Rect {
	c.record("getDestinationClipBounds")
	return ui.Rect{}
}

// DrawColor records drawColor(color, blendMode).
func (c *TestRecordingCanvas) DrawColor(color ui.Color, blendMode ui.BlendMode) {
	c.record("drawColor", color, blendMode)
}

// DrawLine records drawLine(p1, p2, paint).
func (c *TestRecordingCanvas) DrawLine(p1, p2 ui.Offset, paint *ui.Paint) {
	c.record("drawLine", p1, p2, paintCopy(paint))
}

// DrawPaint records drawPaint(paint).
func (c *TestRecordingCanvas) DrawPaint(paint *ui.Paint) { c.record("drawPaint", paintCopy(paint)) }

// DrawRect records drawRect(rect, paint).
func (c *TestRecordingCanvas) DrawRect(rect ui.Rect, paint *ui.Paint) {
	c.record("drawRect", rect, paintCopy(paint))
}

// DrawRRect records drawRRect(rrect, paint).
func (c *TestRecordingCanvas) DrawRRect(rrect ui.RRect, paint *ui.Paint) {
	c.record("drawRRect", rrect, paintCopy(paint))
}

// DrawDRRect records drawDRRect(outer, inner, paint).
func (c *TestRecordingCanvas) DrawDRRect(outer, inner ui.RRect, paint *ui.Paint) {
	c.record("drawDRRect", outer, inner, paintCopy(paint))
}

// DrawRSuperellipse records drawRSuperellipse(rsuperellipse, paint).
func (c *TestRecordingCanvas) DrawRSuperellipse(rsuperellipse ui.RSuperellipse, paint *ui.Paint) {
	c.record("drawRSuperellipse", rsuperellipse, paintCopy(paint))
}

// DrawOval records drawOval(rect, paint).
func (c *TestRecordingCanvas) DrawOval(rect ui.Rect, paint *ui.Paint) {
	c.record("drawOval", rect, paintCopy(paint))
}

// DrawCircle records drawCircle(c, radius, paint).
func (c *TestRecordingCanvas) DrawCircle(center ui.Offset, radius float64, paint *ui.Paint) {
	c.record("drawCircle", center, radius, paintCopy(paint))
}

// DrawArc records drawArc(rect, startAngle, sweepAngle, useCenter, paint).
func (c *TestRecordingCanvas) DrawArc(rect ui.Rect, startAngle, sweepAngle float64, useCenter bool, paint *ui.Paint) {
	c.record("drawArc", rect, startAngle, sweepAngle, useCenter, paintCopy(paint))
}

// DrawPath records drawPath(path, paint).
func (c *TestRecordingCanvas) DrawPath(path *ui.Path, paint *ui.Paint) {
	c.record("drawPath", path, paintCopy(paint))
}

// DrawImage records drawImage(image, offset, paint).
func (c *TestRecordingCanvas) DrawImage(image *ui.Image, offset ui.Offset, paint *ui.Paint) {
	c.record("drawImage", image, offset, paintCopy(paint))
}

// DrawImageRect records drawImageRect(image, src, dst, paint).
func (c *TestRecordingCanvas) DrawImageRect(image *ui.Image, src, dst ui.Rect, paint *ui.Paint) {
	c.record("drawImageRect", image, src, dst, paintCopy(paint))
}

// DrawImageNine records drawImageNine(image, center, dst, paint).
func (c *TestRecordingCanvas) DrawImageNine(image *ui.Image, center, dst ui.Rect, paint *ui.Paint) {
	c.record("drawImageNine", image, center, dst, paintCopy(paint))
}

// DrawPicture records drawPicture(picture).
func (c *TestRecordingCanvas) DrawPicture(picture *ui.Picture) { c.record("drawPicture", picture) }

// DrawParagraph records drawParagraph(paragraph, offset). It is not part
// of ui.Canvas yet (text.dart's Paragraph is being ported); painters that
// draw text can call it on a TestRecordingCanvas.
func (c *TestRecordingCanvas) DrawParagraph(paragraph any, offset ui.Offset) {
	c.record("drawParagraph", paragraph, offset)
}

// DrawPoints records drawPoints(pointMode, points, paint).
func (c *TestRecordingCanvas) DrawPoints(pointMode ui.PointMode, points []ui.Offset, paint *ui.Paint) {
	c.record("drawPoints", pointMode, points, paintCopy(paint))
}

// DrawRawPoints records drawRawPoints(pointMode, points, paint).
func (c *TestRecordingCanvas) DrawRawPoints(pointMode ui.PointMode, points []float32, paint *ui.Paint) {
	c.record("drawRawPoints", pointMode, points, paintCopy(paint))
}

// DrawVertices records drawVertices(vertices, blendMode, paint).
func (c *TestRecordingCanvas) DrawVertices(vertices *ui.Vertices, blendMode ui.BlendMode, paint *ui.Paint) {
	c.record("drawVertices", vertices, blendMode, paintCopy(paint))
}

func nullable[T any](p *T) any {
	if p == nil {
		return nil
	}
	return *p
}

// DrawAtlas records drawAtlas(atlas, transforms, rects, colors, blendMode,
// cullRect, paint).
func (c *TestRecordingCanvas) DrawAtlas(atlas *ui.Image, transforms []ui.RSTransform, rects []ui.Rect, colors []ui.Color,
	blendMode *ui.BlendMode, cullRect *ui.Rect, paint *ui.Paint) {
	var cs any
	if colors != nil {
		cs = colors
	}
	c.record("drawAtlas", atlas, transforms, rects, cs, nullable(blendMode), nullable(cullRect), paintCopy(paint))
}

// DrawRawAtlas records drawRawAtlas(atlas, rstTransforms, rects, colors,
// blendMode, cullRect, paint).
func (c *TestRecordingCanvas) DrawRawAtlas(atlas *ui.Image, rstTransforms, rects []float32, colors []int32,
	blendMode *ui.BlendMode, cullRect *ui.Rect, paint *ui.Paint) {
	var cs any
	if colors != nil {
		cs = colors
	}
	c.record("drawRawAtlas", atlas, rstTransforms, rects, cs, nullable(blendMode), nullable(cullRect), paintCopy(paint))
}

// DrawShadow records drawShadow(path, color, elevation,
// transparentOccluder).
func (c *TestRecordingCanvas) DrawShadow(path *ui.Path, color ui.Color, elevation float64, transparentOccluder bool) {
	c.record("drawShadow", path, color, elevation, transparentOccluder)
}

func (c *TestRecordingCanvas) String() string { return "Instance of 'TestRecordingCanvas'" }

// valueName is _valueName: doubles with one decimal, everything else by
// its Dart toString.
func valueName(value any) string {
	if v, ok := value.(float64); ok {
		return dart.ToStringAsFixed(v, 1)
	}
	return dartToString(value)
}

// dartToString renders a recorded argument as Dart's toString would.
func dartToString(value any) string {
	switch v := value.(type) {
	case nil:
		return "null"
	case float64:
		return dart.DoubleToString(v)
	case float32:
		return dart.DoubleToString(float64(v))
	case string:
		return v
	case fmt.Stringer:
		rv := reflect.ValueOf(v)
		if rv.Kind() == reflect.Pointer && rv.IsNil() {
			return "null"
		}
		return v.String()
	}
	rv := reflect.ValueOf(value)
	switch rv.Kind() {
	case reflect.Pointer:
		if rv.IsNil() {
			return "null"
		}
	case reflect.Slice:
		if rv.IsNil() {
			return "null"
		}
		parts := make([]string, rv.Len())
		for i := range parts {
			parts[i] = dartToString(rv.Index(i).Interface())
		}
		return "[" + strings.Join(parts, ", ") + "]"
	}
	return fmt.Sprint(value)
}

// describeInvocation is _describeInvocation.
func describeInvocation(call Invocation) string {
	var b strings.Builder
	b.WriteString(call.MemberName)
	b.WriteString("(")
	for i, a := range call.PositionalArguments {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(valueName(a))
	}
	separator := ""
	if len(call.PositionalArguments) > 0 {
		separator = ", "
	}
	for _, e := range call.NamedArguments {
		b.WriteString(separator)
		b.WriteString(e.Key)
		b.WriteString(": ")
		b.WriteString(valueName(e.Value))
		separator = ", "
	}
	b.WriteString(")")
	return b.String()
}
