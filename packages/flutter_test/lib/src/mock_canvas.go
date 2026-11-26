package flutter_test

// Port of packages/flutter_test/lib/src/mock_canvas.dart (Flutter 3.50.1):
// the paints, paintsNothing, paintsAssertion and paintsExactlyCountTimes
// matchers and isPathThat.
//
// The object under test is a painter function, func(ui.Canvas), run on a
// TestRecordingCanvas. Not ported (they need rendering, which is not
// ported yet): RenderObject and Finder targets and painters taking a
// PaintingContext and an Offset (Dart's _ContextPainterFunction); such
// objects are reported as "not one of the supported objects".
//
// Go usage (Dart's cascades are method chains):
//
//	Expect(t, func(c ui.Canvas) { border.Paint(c, rect) },
//		Paints().Path(PaintPatternPathOptions{Color: &color}).Rect(PaintPatternRectOptions{Rect: &r}))
//	Expect(t, painter, PaintsNothing())
//	Expect(t, painter, IsNot(Paints().Image()))

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	"github.com/jonathanperis/flutter-go/src/dart"
)

// PaintPatternPredicate is the signature of the predicates given to
// PaintPattern.Something and Everything: the method's Dart name and its
// positional arguments. A predicate may panic with a string to explain a
// mismatch (Dart: throw a String).
type PaintPatternPredicate func(methodName string, arguments []any) bool

func colorsMatch(x ui.Color, y *ui.Color) bool {
	if y == nil {
		return false
	}
	const limit = 1.0 / 255
	return x.ColorSpace == y.ColorSpace && abs(x.A-y.A) < limit && abs(x.R-y.R) < limit &&
		abs(x.G-y.G) < limit && abs(x.B-y.B) < limit
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

// ---- isPathThat ----

// IsPathThatOptions are isPathThat's named parameters.
type IsPathThatOptions struct {
	Includes []ui.Offset
	Excludes []ui.Offset
}

type pathMatcher struct{ includes, excludes []ui.Offset }

// IsPathThat matches a *ui.Path that contains every point of Includes and
// none of Excludes.
func IsPathThat(opts ...IsPathThatOptions) Matcher {
	var o IsPathThatOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	return &pathMatcher{o.Includes, o.Excludes}
}

func (m *pathMatcher) Matches(object any, matchState map[any]any) bool {
	path, ok := object.(*ui.Path)
	if !ok || path == nil {
		matchState[m] = "The given object (" + dartToString(object) + ") was not a Path."
		return false
	}
	var errs []string
	for _, offset := range m.includes {
		if !path.Contains(offset) {
			errs = append(errs, "Offset "+offset.String()+" should be inside the path, but is not.")
		}
	}
	for _, offset := range m.excludes {
		if path.Contains(offset) {
			errs = append(errs, "Offset "+offset.String()+" should be outside the path, but is not.")
		}
	}
	if len(errs) == 0 {
		return true
	}
	matchState[m] = "Not all the given points were inside or outside the path as expected:\n  " + strings.Join(errs, "\n  ")
	return false
}

func (m *pathMatcher) Describe() string {
	points := func(list []ui.Offset) string {
		if len(list) == 1 {
			return "one particular point"
		}
		return fmt.Sprintf("%d particular points", len(list))
	}
	return "A Path that contains " + points(m.includes) + " but does not contain " + points(m.excludes) + "."
}

func (m *pathMatcher) DescribeMismatch(_ any, matchState map[any]any) string {
	s, _ := matchState[m].(string)
	return s
}

// ---- the recording-canvas matchers ----

// mismatchedCall is _MismatchedCall.
type mismatchedCall struct {
	message, callIntroduction string
	call                      RecordedInvocation
}

// evaluatePainter runs a supported object on the canvas (Dart's
// _evaluatePainter); false: not supported.
func evaluatePainter(object any, canvas *TestRecordingCanvas) bool {
	switch f := object.(type) {
	case func(ui.Canvas):
		f(canvas)
	case func(*TestRecordingCanvas):
		f(canvas)
	default:
		return false
	}
	return true
}

// errorString is Dart's error.toString() for a recovered panic value.
func errorString(r any) string {
	switch e := r.(type) {
	case error:
		return e.Error()
	case string:
		return e
	}
	return dartToString(r)
}

// recordingCanvasMatch is _TestRecordingCanvasMatcher.matches.
func recordingCanvasMatch(key any, object any, matchState map[any]any,
	evaluate func(calls []RecordedInvocation, description *strings.Builder) bool) bool {
	canvas := NewTestRecordingCanvas()
	var description strings.Builder
	prefixMessage := "unexpectedly failed."
	result := false
	supported := true
	func() {
		defer func() {
			if r := recover(); r != nil {
				stack := dart.StackTraceFromPanic()
				prefixMessage = "threw the following exception:"
				description.WriteString(errorString(r) + "\n")
				if stack != nil {
					description.WriteString(stack.String())
				}
				result = false
			}
		}()
		if !evaluatePainter(object, canvas) {
			supported = false
			return
		}
		result = evaluate(canvas.Invocations, &description)
		if !result {
			prefixMessage = "did not match the pattern."
		}
	}()
	if !supported {
		matchState[key] = `was not one of the supported objects for the "paints" matcher.`
		return false
	}
	if !result {
		if len(canvas.Invocations) > 0 {
			description.WriteString("The complete display list was:")
			for _, call := range canvas.Invocations {
				description.WriteString("\n  * " + call.String())
			}
		}
		matchState[key] = prefixMessage + "\n" + description.String()
	}
	return result
}

func describeMismatchFromState(key any, matchState map[any]any) string {
	s, _ := matchState[key].(string)
	return s
}

type paintsCountMatcher struct {
	methodName string
	count      int
}

// PaintsExactlyCountTimes matches a painter that calls the canvas method
// methodName (its Dart name, Dart's Symbol) exactly count times.
func PaintsExactlyCountTimes(methodName string, count int) Matcher {
	return &paintsCountMatcher{methodName, count}
}

func (m *paintsCountMatcher) Describe() string {
	return fmt.Sprintf("Object or closure painting Symbol(%q) exactly %d times", m.methodName, m.count)
}

func (m *paintsCountMatcher) Matches(object any, matchState map[any]any) bool {
	return recordingCanvasMatch(m, object, matchState, func(calls []RecordedInvocation, description *strings.Builder) bool {
		count := 0
		for _, call := range calls {
			if call.Invocation.IsMethod() && call.Invocation.MemberName == m.methodName {
				count++
			}
		}
		if count != m.count {
			fmt.Fprintf(description, "It painted Symbol(%q) %d times instead of %d times.", m.methodName, count, m.count)
		}
		return count == m.count
	})
}

func (m *paintsCountMatcher) DescribeMismatch(_ any, matchState map[any]any) string {
	return describeMismatchFromState(m, matchState)
}

type paintsNothingMatcher struct{}

// PaintsNothing matches a painter that paints nothing (save and restore
// don't count).
func PaintsNothing() Matcher { return &paintsNothingMatcher{} }

func (m *paintsNothingMatcher) Describe() string { return "An object or closure that paints nothing." }

func (m *paintsNothingMatcher) Matches(object any, matchState map[any]any) bool {
	return recordingCanvasMatch(m, object, matchState, func(calls []RecordedInvocation, description *strings.Builder) bool {
		for _, call := range calls {
			if name := call.Invocation.MemberName; name == "save" || name == "restore" {
				continue
			}
			description.WriteString("painted something, the first call having the following stack:\n" +
				call.StackToString(StackToStringOptions{Indent: "  "}) + "\n")
			return false
		}
		return true
	})
}

func (m *paintsNothingMatcher) DescribeMismatch(_ any, matchState map[any]any) string {
	return describeMismatchFromState(m, matchState)
}

type paintsAssertionMatcher struct{}

// PaintsAssertion matches a painter that fails an assertion (panics with
// a dart.AssertionFailure, such as a FlutterError) while painting.
func PaintsAssertion() Matcher { return &paintsAssertionMatcher{} }

func (m *paintsAssertionMatcher) Describe() string {
	return "An object or closure that asserts when it tries to paint."
}

func (m *paintsAssertionMatcher) Matches(object any, matchState map[any]any) bool {
	canvas := NewTestRecordingCanvas()
	var description strings.Builder
	prefixMessage := "unexpectedly failed."
	result := false
	supported := true
	func() {
		defer func() {
			if r := recover(); r != nil {
				var af dart.AssertionFailure
				if e, ok := r.(error); ok && errors.As(e, &af) {
					result = true
					return
				}
				stack := dart.StackTraceFromPanic()
				prefixMessage = "threw the following exception:"
				description.WriteString(errorString(r) + "\n")
				if stack != nil {
					description.WriteString(stack.String())
				}
				result = false
			}
		}()
		if !evaluatePainter(object, canvas) {
			supported = false
			return
		}
		prefixMessage = "did not assert."
	}()
	if !supported {
		matchState[m] = `was not one of the supported objects for the "paints" matcher.`
		return false
	}
	if !result {
		if len(canvas.Invocations) > 0 {
			description.WriteString("The complete display list was:")
			for _, call := range canvas.Invocations {
				description.WriteString("\n  * " + call.String())
			}
		}
		matchState[m] = prefixMessage + "\n" + description.String()
	}
	return result
}

func (m *paintsAssertionMatcher) DescribeMismatch(_ any, matchState map[any]any) string {
	return describeMismatchFromState(m, matchState)
}

// ---- the paints pattern ----

// PaintPattern is the paint pattern the "paints" matcher checks: a list
// of predicates matched in order against the canvas calls (other calls in
// between are allowed, and so are calls after the last predicate). Build
// it with Paints() and chain the methods (Dart's cascade); it is a
// Matcher.
type PaintPattern struct {
	predicates []paintPredicate
}

// Paints is the "paints" matcher: an empty pattern (which fails, asking
// for a pattern, when the painter paints something).
func Paints() *PaintPattern { return &PaintPattern{} }

func (p *PaintPattern) add(pred paintPredicate) *PaintPattern {
	p.predicates = append(p.predicates, pred)
	return p
}

func optF(v *float64) any {
	if v == nil {
		return nil
	}
	return *v
}

// PaintPatternTransformOptions are transform's named parameters: Matrix4
// is the expected []float64 (compared element-wise) or a Matcher.
type PaintPatternTransformOptions struct{ Matrix4 any }

// Transform expects a call to transform.
func (p *PaintPattern) Transform(opts ...PaintPatternTransformOptions) *PaintPattern {
	var m any
	if len(opts) > 0 {
		m = opts[0].Matrix4
	}
	return p.add(&functionPaintPredicate{"transform", []any{m}})
}

// PaintPatternTranslateOptions are translate's named parameters.
type PaintPatternTranslateOptions struct{ X, Y *float64 }

// Translate expects a call to translate.
func (p *PaintPattern) Translate(opts ...PaintPatternTranslateOptions) *PaintPattern {
	var o PaintPatternTranslateOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	return p.add(&functionPaintPredicate{"translate", []any{optF(o.X), optF(o.Y)}})
}

// PaintPatternScaleOptions are scale's named parameters.
type PaintPatternScaleOptions struct{ X, Y *float64 }

// Scale expects a call to scale with two arguments.
func (p *PaintPattern) Scale(opts ...PaintPatternScaleOptions) *PaintPattern {
	var o PaintPatternScaleOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	return p.add(&functionPaintPredicate{"scale", []any{optF(o.X), optF(o.Y)}})
}

// PaintPatternRotateOptions are rotate's named parameters.
type PaintPatternRotateOptions struct{ Angle *float64 }

// Rotate expects a call to rotate.
func (p *PaintPattern) Rotate(opts ...PaintPatternRotateOptions) *PaintPattern {
	var o PaintPatternRotateOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	return p.add(&functionPaintPredicate{"rotate", []any{optF(o.Angle)}})
}

// Save expects a call to save.
func (p *PaintPattern) Save() *PaintPattern { return p.add(&functionPaintPredicate{"save", nil}) }

// Restore expects a call to restore.
func (p *PaintPattern) Restore() *PaintPattern {
	return p.add(&functionPaintPredicate{"restore", nil})
}

// SaveRestore expects a save and skips to its matching restore.
func (p *PaintPattern) SaveRestore() *PaintPattern { return p.add(&saveRestorePairPaintPredicate{}) }

// PaintPatternClipRectOptions are clipRect's named parameters.
type PaintPatternClipRectOptions struct{ Rect *ui.Rect }

// ClipRect expects a call to clipRect.
func (p *PaintPattern) ClipRect(opts ...PaintPatternClipRectOptions) *PaintPattern {
	var r any
	if len(opts) > 0 && opts[0].Rect != nil {
		r = *opts[0].Rect
	}
	return p.add(&functionPaintPredicate{"clipRect", []any{r}})
}

// PaintPatternClipPathOptions are clipPath's named parameters.
type PaintPatternClipPathOptions struct{ PathMatcher Matcher }

// ClipPath expects a call to clipPath whose path matches PathMatcher
// (IsPathThat).
func (p *PaintPattern) ClipPath(opts ...PaintPatternClipPathOptions) *PaintPattern {
	var m any
	if len(opts) > 0 && opts[0].PathMatcher != nil {
		m = opts[0].PathMatcher
	}
	return p.add(&functionPaintPredicate{"clipPath", []any{m}})
}

// PaintPatternClipRRectOptions are clipRRect's named parameters.
type PaintPatternClipRRectOptions struct{ RRect *ui.RRect }

// ClipRRect expects a call to clipRRect.
func (p *PaintPattern) ClipRRect(opts ...PaintPatternClipRRectOptions) *PaintPattern {
	var r any
	if len(opts) > 0 && opts[0].RRect != nil {
		r = *opts[0].RRect
	}
	return p.add(&functionPaintPredicate{"clipRRect", []any{r}})
}

// PaintPatternClipRSuperellipseOptions are clipRSuperellipse's named
// parameters.
type PaintPatternClipRSuperellipseOptions struct{ RSuperellipse *ui.RSuperellipse }

// ClipRSuperellipse expects a call to clipRSuperellipse.
func (p *PaintPattern) ClipRSuperellipse(opts ...PaintPatternClipRSuperellipseOptions) *PaintPattern {
	var r any
	if len(opts) > 0 && opts[0].RSuperellipse != nil {
		r = *opts[0].RSuperellipse
	}
	return p.add(&functionPaintPredicate{"clipRSuperellipse", []any{r}})
}

// PaintOptions are the paint checks shared by the drawing predicates
// (nil: not checked).
type PaintOptions struct {
	Color         *ui.Color
	StrokeWidth   *float64
	HasMaskFilter *bool
	Style         *ui.PaintingStyle
}

// PaintPatternRectOptions are rect's named parameters.
type PaintPatternRectOptions struct {
	Rect *ui.Rect
	PaintOptions
}

// Rect expects a call to drawRect.
func (p *PaintPattern) Rect(opts ...PaintPatternRectOptions) *PaintPattern {
	var o PaintPatternRectOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	var r any
	if o.Rect != nil {
		r = *o.Rect
	}
	return p.add(newOneParameterPredicate("drawRect", "a rectangle", "Rect", r, o.PaintOptions))
}

// PaintPatternRRectOptions are rrect's named parameters.
type PaintPatternRRectOptions struct {
	RRect *ui.RRect
	PaintOptions
}

// RRect expects a call to drawRRect (the rrect compared within 1e-4).
func (p *PaintPattern) RRect(opts ...PaintPatternRRectOptions) *PaintPattern {
	var o PaintPatternRRectOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	pred := &rrectPaintPredicate{rrect: o.RRect}
	pred.drawCommandPaintPredicate = newDrawCommand("drawRRect", "a rounded rectangle", 2, 1, o.PaintOptions, nil)
	pred.self = pred
	return p.add(pred)
}

// PaintPatternDRRectOptions are drrect's named parameters.
type PaintPatternDRRectOptions struct {
	Outer, Inner *ui.RRect
	PaintOptions
}

// DRRect expects a call to drawDRRect.
func (p *PaintPattern) DRRect(opts ...PaintPatternDRRectOptions) *PaintPattern {
	var o PaintPatternDRRectOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	var outer, inner any
	if o.Outer != nil {
		outer = *o.Outer
	}
	if o.Inner != nil {
		inner = *o.Inner
	}
	return p.add(newTwoParameterPredicate("drawDRRect", "a rounded rectangle outline", "RRect", "RRect", outer, inner, o.PaintOptions))
}

// PaintPatternRSuperellipseOptions are rsuperellipse's named parameters
// (Dart's has no style).
type PaintPatternRSuperellipseOptions struct {
	RSuperellipse *ui.RSuperellipse
	Color         *ui.Color
	StrokeWidth   *float64
	HasMaskFilter *bool
}

// RSuperellipse expects a call to drawRSuperellipse.
func (p *PaintPattern) RSuperellipse(opts ...PaintPatternRSuperellipseOptions) *PaintPattern {
	var o PaintPatternRSuperellipseOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	pred := &rsuperellipsePaintPredicate{rsuperellipse: o.RSuperellipse}
	pred.drawCommandPaintPredicate = newDrawCommand("drawRSuperellipse", "a rounded superellipse", 2, 1,
		PaintOptions{Color: o.Color, StrokeWidth: o.StrokeWidth, HasMaskFilter: o.HasMaskFilter}, nil)
	pred.self = pred
	return p.add(pred)
}

// PaintPatternCircleOptions are circle's named parameters.
type PaintPatternCircleOptions struct {
	X, Y, Radius *float64
	PaintOptions
}

// Circle expects a call to drawCircle.
func (p *PaintPattern) Circle(opts ...PaintPatternCircleOptions) *PaintPattern {
	var o PaintPatternCircleOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	pred := &circlePaintPredicate{x: o.X, y: o.Y, radius: o.Radius}
	pred.drawCommandPaintPredicate = newDrawCommand("drawCircle", "a circle", 3, 2, o.PaintOptions, nil)
	pred.self = pred
	return p.add(pred)
}

// PaintPatternPathOptions are path's named parameters: points the path
// must and must not contain (nil: not checked).
type PaintPatternPathOptions struct {
	Includes, Excludes []ui.Offset
	PaintOptions
}

// Path expects a call to drawPath.
func (p *PaintPattern) Path(opts ...PaintPatternPathOptions) *PaintPattern {
	var o PaintPatternPathOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	pred := &pathPaintPredicate{includes: o.Includes, excludes: o.Excludes}
	pred.drawCommandPaintPredicate = newDrawCommand("drawPath", "a path", 2, 1, o.PaintOptions, nil)
	pred.self = pred
	return p.add(pred)
}

// PaintPatternLineOptions are line's named parameters.
type PaintPatternLineOptions struct {
	P1, P2 *ui.Offset
	PaintOptions
}

// Line expects a call to drawLine.
func (p *PaintPattern) Line(opts ...PaintPatternLineOptions) *PaintPattern {
	var o PaintPatternLineOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	pred := &linePaintPredicate{p1: o.P1, p2: o.P2}
	pred.drawCommandPaintPredicate = newDrawCommand("drawLine", "a line", 3, 2, o.PaintOptions, nil)
	pred.self = pred
	return p.add(pred)
}

// PaintPatternArcOptions are arc's named parameters.
type PaintPatternArcOptions struct {
	Rect                   *ui.Rect
	StartAngle, SweepAngle *float64
	UseCenter              *bool
	PaintOptions
	StrokeCap *ui.StrokeCap
}

// Arc expects a call to drawArc.
func (p *PaintPattern) Arc(opts ...PaintPatternArcOptions) *PaintPattern {
	var o PaintPatternArcOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	pred := &arcPaintPredicate{rect: o.Rect, startAngle: o.StartAngle, sweepAngle: o.SweepAngle, useCenter: o.UseCenter}
	pred.drawCommandPaintPredicate = newDrawCommand("drawArc", "an arc", 5, 4, o.PaintOptions, o.StrokeCap)
	pred.self = pred
	return p.add(pred)
}

// PaintPatternParagraphOptions are paragraph's named parameters
// (Paragraph: whatever TestRecordingCanvas.DrawParagraph recorded; Offset:
// a ui.Offset or a Matcher).
type PaintPatternParagraphOptions struct {
	Paragraph any
	Offset    any
}

// Paragraph expects a call to drawParagraph (see
// TestRecordingCanvas.DrawParagraph: ui.Canvas has no drawParagraph until
// text.dart's Paragraph is ported).
func (p *PaintPattern) Paragraph(opts ...PaintPatternParagraphOptions) *PaintPattern {
	var o PaintPatternParagraphOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	return p.add(&functionPaintPredicate{"drawParagraph", []any{o.Paragraph, o.Offset}})
}

// PaintPatternShadowOptions are shadow's named parameters.
type PaintPatternShadowOptions struct {
	Includes, Excludes  []ui.Offset
	Color               *ui.Color
	Elevation           *float64
	TransparentOccluder *bool
}

// Shadow expects a call to drawShadow.
func (p *PaintPattern) Shadow(opts ...PaintPatternShadowOptions) *PaintPattern {
	var o PaintPatternShadowOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	return p.add(&shadowPredicate{o})
}

// PaintPatternImageOptions are image's named parameters.
type PaintPatternImageOptions struct {
	Image *ui.Image
	X, Y  *float64
	PaintOptions
}

// Image expects a call to drawImage. As in Flutter 3.50.1, the predicate
// casts the image argument to an Offset, so it always fails with a type
// error (Flutter's tests only use it as IsNot(Paints().Image())).
func (p *PaintPattern) Image(opts ...PaintPatternImageOptions) *PaintPattern {
	var o PaintPatternImageOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	pred := &drawImagePaintPredicate{image: o.Image, x: o.X, y: o.Y}
	pred.drawCommandPaintPredicate = newDrawCommand("drawImage", "an image", 3, 2, o.PaintOptions, nil)
	pred.self = pred
	return p.add(pred)
}

// PaintPatternDrawImageRectOptions are drawImageRect's named parameters.
type PaintPatternDrawImageRectOptions struct {
	Image               *ui.Image
	Source, Destination *ui.Rect
	PaintOptions
}

// DrawImageRect expects a call to drawImageRect.
func (p *PaintPattern) DrawImageRect(opts ...PaintPatternDrawImageRectOptions) *PaintPattern {
	var o PaintPatternDrawImageRectOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	pred := &drawImageRectPaintPredicate{image: o.Image, source: o.Source, destination: o.Destination}
	pred.drawCommandPaintPredicate = newDrawCommand("drawImageRect", "an image", 4, 3, o.PaintOptions, nil)
	pred.self = pred
	return p.add(pred)
}

// Something skips calls until predicate accepts one (and fails if none
// does).
func (p *PaintPattern) Something(predicate PaintPatternPredicate) *PaintPattern {
	return p.add(&somethingPaintPredicate{predicate})
}

// Everything requires predicate to accept every remaining call.
func (p *PaintPattern) Everything(predicate PaintPatternPredicate) *PaintPattern {
	return p.add(&everythingPaintPredicate{predicate})
}

// Describe is describe().
func (p *PaintPattern) Describe() string {
	if len(p.predicates) == 0 {
		return "An object or closure and a paint pattern."
	}
	parts := make([]string, len(p.predicates))
	for i, pred := range p.predicates {
		parts[i] = pred.String()
	}
	return "Object or closure painting:\n" + strings.Join(parts, "\n")
}

// Matches runs the painter on a TestRecordingCanvas and matches the
// pattern against its calls.
func (p *PaintPattern) Matches(object any, matchState map[any]any) bool {
	return recordingCanvasMatch(p, object, matchState, p.evaluatePredicates)
}

// DescribeMismatch is the failure explanation recorded by Matches.
func (p *PaintPattern) DescribeMismatch(_ any, matchState map[any]any) string {
	return describeMismatchFromState(p, matchState)
}

// callIterator is Dart's Iterator<RecordedInvocation> over the calls.
type callIterator struct {
	calls []RecordedInvocation
	i     int
}

func (c *callIterator) moveNext() bool {
	if c.i < len(c.calls) {
		c.i++
	}
	return c.i < len(c.calls)
}

// current is Iterator.current; past the end, Dart's list iterator yields
// null and the cast to RecordedInvocation throws a TypeError.
func (c *callIterator) current() RecordedInvocation {
	if c.i < 0 || c.i >= len(c.calls) {
		panic(typeError("type 'Null' is not a subtype of type 'RecordedInvocation' in type cast"))
	}
	return c.calls[c.i]
}

type typeError string

func (e typeError) Error() string { return string(e) }

func (p *PaintPattern) evaluatePredicates(calls []RecordedInvocation, description *strings.Builder) (ok bool) {
	if len(calls) == 0 {
		description.WriteString("It painted nothing.\n")
		return false
	}
	if len(p.predicates) == 0 {
		description.WriteString("It painted something, but you must now add a pattern to the paints " +
			"matcher in the test to verify that it matches the important parts of the following.\n")
		return false
	}
	call := &callIterator{calls: calls, i: -1}
	call.moveNext()
	defer func() {
		r := recover()
		if r == nil {
			return
		}
		switch e := r.(type) {
		case *mismatchedCall:
			description.WriteString(e.message + "\n")
			description.WriteString(e.callIntroduction + "\n")
			description.WriteString(e.call.StackToString(StackToStringOptions{Indent: "  "}) + "\n")
			ok = false
		case string:
			description.WriteString(e + "\n")
			if call.i >= 0 && call.i < len(call.calls) {
				description.WriteString("The stack of the offending call was:\n" +
					call.calls[call.i].StackToString(StackToStringOptions{Indent: "  "}) + "\n")
			}
			ok = false
		default:
			panic(r)
		}
	}()
	for _, pred := range p.predicates {
		pred.match(call)
	}
	// We allow painting more than expected.
	return true
}

// ---- predicates ----

type paintPredicate interface {
	match(call *callIterator)
	String() string
}

func symbolString(name string) string { return name }

// checkMethod is _PaintPredicate.checkMethod: skip to the next call of
// symbol, failing if there is none.
func checkMethod(self paintPredicate, call *callIterator, symbol string) {
	others := 0
	firstCall := call.current()
	for !call.current().Invocation.IsMethod() || call.current().Invocation.MemberName != symbol {
		others++
		if !call.moveNext() {
			plural := "s"
			if others == 1 {
				plural = ""
			}
			panic(&mismatchedCall{
				message: fmt.Sprintf("It called %d other method%s on the canvas, the first of which was %v, but did not call "+
					"%s() at the time where %s was expected.", others, plural, firstCall, symbol, self),
				callIntroduction: fmt.Sprintf("The first method that was called when the call to %s() was expected, %v, "+
					"was called with the following stack:", symbol, firstCall),
				call: firstCall,
			})
		}
	}
}

func flutterError(msg string) *foundation.FlutterError { return foundation.NewFlutterError(msg) }

// argsVerifier is the overridable verifyArguments/debugFillDescription
// pair of a draw-command predicate.
type argsVerifier interface {
	verifyArguments(arguments []any)
	debugFillDescription(description *[]string)
}

// drawCommandPaintPredicate is _DrawCommandPaintPredicate.
type drawCommandPaintPredicate struct {
	symbol             string
	name               string
	argumentCount      int
	paintArgumentIndex int
	PaintOptions
	strokeCap *ui.StrokeCap
	self      argsVerifier
}

func newDrawCommand(symbol, name string, argumentCount, paintArgumentIndex int, paint PaintOptions, strokeCap *ui.StrokeCap) drawCommandPaintPredicate {
	return drawCommandPaintPredicate{symbol: symbol, name: name, argumentCount: argumentCount,
		paintArgumentIndex: paintArgumentIndex, PaintOptions: paint, strokeCap: strokeCap}
}

func (d *drawCommandPaintPredicate) methodName() string { return symbolString(d.symbol) }

func (d *drawCommandPaintPredicate) match(call *callIterator) {
	checkMethod(d, call, d.symbol)
	actualArgumentCount := len(call.current().Invocation.PositionalArguments)
	if actualArgumentCount != d.argumentCount {
		plural := "s"
		if actualArgumentCount == 1 {
			plural = ""
		}
		panic(flutterError(fmt.Sprintf("It called %s with %d argument%s; expected %d.", d.methodName(),
			actualArgumentCount, plural, d.argumentCount)))
	}
	d.self.verifyArguments(call.current().Invocation.PositionalArguments)
	call.moveNext()
}

func (d *drawCommandPaintPredicate) verifyArguments(arguments []any) {
	paintArgument := arguments[d.paintArgumentIndex].(*ui.Paint)
	if d.Color != nil && !colorsMatch(paintArgument.Color(), d.Color) {
		panic(flutterError(fmt.Sprintf("It called %s with a paint whose color, %v, was not exactly the expected color (%v).",
			d.methodName(), paintArgument.Color(), *d.Color)))
	}
	if d.StrokeWidth != nil && paintArgument.StrokeWidth() != *d.StrokeWidth {
		panic(flutterError(fmt.Sprintf("It called %s with a paint whose strokeWidth, %s, was not exactly the expected strokeWidth (%s).",
			d.methodName(), dart.DoubleToString(paintArgument.StrokeWidth()), dart.DoubleToString(*d.StrokeWidth))))
	}
	if d.HasMaskFilter != nil && (paintArgument.MaskFilter() != nil) != *d.HasMaskFilter {
		if *d.HasMaskFilter {
			panic(flutterError("It called " + d.methodName() + " with a paint that did not have a mask filter, despite expecting one."))
		}
		panic(flutterError("It called " + d.methodName() + " with a paint that had a mask filter, despite not expecting one."))
	}
	if d.Style != nil && paintArgument.Style() != *d.Style {
		panic(flutterError(fmt.Sprintf("It called %s with a paint whose style, %v, was not exactly the expected style (%v).",
			d.methodName(), paintArgument.Style(), *d.Style)))
	}
	if d.strokeCap != nil && paintArgument.StrokeCap() != *d.strokeCap {
		panic(flutterError(fmt.Sprintf("It called %s with a paint whose strokeCap, %v, was not exactly the expected strokeCap (%v).",
			d.methodName(), paintArgument.StrokeCap(), *d.strokeCap)))
	}
}

func (d *drawCommandPaintPredicate) debugFillDescription(description *[]string) {
	if d.Color != nil {
		*description = append(*description, d.Color.String())
	}
	if d.StrokeWidth != nil {
		*description = append(*description, "strokeWidth: "+dart.DoubleToString(*d.StrokeWidth))
	}
	if d.HasMaskFilter != nil {
		if *d.HasMaskFilter {
			*description = append(*description, "a mask filter")
		} else {
			*description = append(*description, "no mask filter")
		}
	}
	if d.Style != nil {
		*description = append(*description, d.Style.String())
	}
}

func (d *drawCommandPaintPredicate) String() string {
	var description []string
	d.self.debugFillDescription(&description)
	result := d.name
	if len(description) > 0 {
		result += " with " + strings.Join(description, ", ")
	}
	return result
}

// dartEquals is Dart's == on recorded arguments: Go's == for comparable
// values of the same type, reflect.DeepEqual otherwise (Dart compares
// lists by identity; equal contents are accepted here).
func dartEquals(a, b any) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	ta, tb := reflect.TypeOf(a), reflect.TypeOf(b)
	if ta == tb && ta.Comparable() {
		return a == b
	}
	return reflect.DeepEqual(a, b)
}

// describeExpected is the shared "contains the type name" rule of the
// parameter predicates' debugFillDescription.
func describeExpected(expected any, typeName string) string {
	s := dartToString(expected)
	if strings.Contains(s, typeName) {
		return s
	}
	return typeName + ": " + s
}

// oneParameterPaintPredicate is _OneParameterPaintPredicate<T>.
type oneParameterPaintPredicate struct {
	drawCommandPaintPredicate
	typeName string
	expected any
}

func newOneParameterPredicate(symbol, name, typeName string, expected any, paint PaintOptions) *oneParameterPaintPredicate {
	p := &oneParameterPaintPredicate{typeName: typeName, expected: expected}
	p.drawCommandPaintPredicate = newDrawCommand(symbol, name, 2, 1, paint, nil)
	p.self = p
	return p
}

func (p *oneParameterPaintPredicate) verifyArguments(arguments []any) {
	p.drawCommandPaintPredicate.verifyArguments(arguments)
	actual := arguments[0]
	if p.expected != nil && !dartEquals(actual, p.expected) {
		panic(flutterError(fmt.Sprintf("It called %s with %s, %s, which was not exactly the expected %s (%s).",
			p.methodName(), p.typeName, dartToString(actual), p.typeName, dartToString(p.expected))))
	}
}

func (p *oneParameterPaintPredicate) debugFillDescription(description *[]string) {
	p.drawCommandPaintPredicate.debugFillDescription(description)
	if p.expected != nil {
		*description = append(*description, describeExpected(p.expected, p.typeName))
	}
}

// twoParameterPaintPredicate is _TwoParameterPaintPredicate<T1, T2>.
type twoParameterPaintPredicate struct {
	drawCommandPaintPredicate
	type1, type2         string
	expected1, expected2 any
}

func newTwoParameterPredicate(symbol, name, type1, type2 string, expected1, expected2 any, paint PaintOptions) *twoParameterPaintPredicate {
	p := &twoParameterPaintPredicate{type1: type1, type2: type2, expected1: expected1, expected2: expected2}
	p.drawCommandPaintPredicate = newDrawCommand(symbol, name, 3, 2, paint, nil)
	p.self = p
	return p
}

func (p *twoParameterPaintPredicate) verifyArguments(arguments []any) {
	p.drawCommandPaintPredicate.verifyArguments(arguments)
	if actual1 := arguments[0]; p.expected1 != nil && !dartEquals(actual1, p.expected1) {
		panic(flutterError(fmt.Sprintf("It called %s with its first argument (a %s), %s, which was not exactly the expected %s (%s).",
			p.methodName(), p.type1, dartToString(actual1), p.type1, dartToString(p.expected1))))
	}
	if actual2 := arguments[1]; p.expected2 != nil && !dartEquals(actual2, p.expected2) {
		panic(flutterError(fmt.Sprintf("It called %s with its second argument (a %s), %s, which was not exactly the expected %s (%s).",
			p.methodName(), p.type2, dartToString(actual2), p.type2, dartToString(p.expected2))))
	}
}

func (p *twoParameterPaintPredicate) debugFillDescription(description *[]string) {
	p.drawCommandPaintPredicate.debugFillDescription(description)
	if p.expected1 != nil {
		*description = append(*description, describeExpected(p.expected1, p.type1))
	}
	if p.expected2 != nil {
		*description = append(*description, describeExpected(p.expected2, p.type2))
	}
}

type rrectPaintPredicate struct {
	drawCommandPaintPredicate
	rrect *ui.RRect
}

func (p *rrectPaintPredicate) verifyArguments(arguments []any) {
	p.drawCommandPaintPredicate.verifyArguments(arguments)
	const eps = .0001
	actual := arguments[0].(ui.RRect)
	if r := p.rrect; r != nil && (abs(actual.Left-r.Left) > eps || abs(actual.Right-r.Right) > eps ||
		abs(actual.Top-r.Top) > eps || abs(actual.Bottom-r.Bottom) > eps ||
		abs(actual.BlRadiusX-r.BlRadiusX) > eps || abs(actual.BlRadiusY-r.BlRadiusY) > eps ||
		abs(actual.BrRadiusX-r.BrRadiusX) > eps || abs(actual.BrRadiusY-r.BrRadiusY) > eps ||
		abs(actual.TlRadiusX-r.TlRadiusX) > eps || abs(actual.TlRadiusY-r.TlRadiusY) > eps ||
		abs(actual.TrRadiusX-r.TrRadiusX) > eps || abs(actual.TrRadiusY-r.TrRadiusY) > eps) {
		panic(flutterError(fmt.Sprintf("It called %s with RRect, %v, which was not exactly the expected RRect (%v).",
			p.methodName(), actual, *r)))
	}
}

func (p *rrectPaintPredicate) debugFillDescription(description *[]string) {
	p.drawCommandPaintPredicate.debugFillDescription(description)
	if p.rrect != nil {
		*description = append(*description, "RRect: "+p.rrect.String())
	}
}

type rsuperellipsePaintPredicate struct {
	drawCommandPaintPredicate
	rsuperellipse *ui.RSuperellipse
}

func (p *rsuperellipsePaintPredicate) verifyArguments(arguments []any) {
	p.drawCommandPaintPredicate.verifyArguments(arguments)
	arg := arguments[0].(ui.RSuperellipse)
	if p.rsuperellipse != nil && arg != *p.rsuperellipse {
		panic(flutterError(fmt.Sprintf("It called %s with a rounded superellipse, %v, which was not exactly the expected "+
			"rounded superellipse (%v).", p.methodName(), arg, *p.rsuperellipse)))
	}
}

func (p *rsuperellipsePaintPredicate) debugFillDescription(description *[]string) {
	p.drawCommandPaintPredicate.debugFillDescription(description)
	if p.rsuperellipse != nil {
		*description = append(*description, "rounded superellipse "+p.rsuperellipse.String())
	}
}

func fixed1(v float64) string { return dart.ToStringAsFixed(v, 1) }

type circlePaintPredicate struct {
	drawCommandPaintPredicate
	x, y, radius *float64
}

func (p *circlePaintPredicate) verifyArguments(arguments []any) {
	p.drawCommandPaintPredicate.verifyArguments(arguments)
	pointArgument := arguments[0].(ui.Offset)
	if p.x != nil && p.y != nil {
		point := ui.NewOffset(*p.x, *p.y)
		if point != pointArgument {
			panic(flutterError(fmt.Sprintf("It called %s with a center coordinate, %v, which was not exactly the expected coordinate (%v).",
				p.methodName(), pointArgument, point)))
		}
	} else {
		if p.x != nil && pointArgument.Dx != *p.x {
			panic(flutterError(fmt.Sprintf("It called %s with a center coordinate, %v, whose x-coordinate was not exactly the expected coordinate (%s).",
				p.methodName(), pointArgument, fixed1(*p.x))))
		}
		if p.y != nil && pointArgument.Dy != *p.y {
			panic(flutterError(fmt.Sprintf("It called %s with a center coordinate, %v, whose y-coordinate was not exactly the expected coordinate (%s).",
				p.methodName(), pointArgument, fixed1(*p.y))))
		}
	}
	radiusArgument := arguments[1].(float64)
	if p.radius != nil && radiusArgument != *p.radius {
		panic(flutterError(fmt.Sprintf("It called %s with radius, %s, which was not exactly the expected radius (%s).",
			p.methodName(), fixed1(radiusArgument), fixed1(*p.radius))))
	}
}

func (p *circlePaintPredicate) debugFillDescription(description *[]string) {
	p.drawCommandPaintPredicate.debugFillDescription(description)
	if p.x != nil && p.y != nil {
		*description = append(*description, "point "+ui.NewOffset(*p.x, *p.y).String())
	} else {
		if p.x != nil {
			*description = append(*description, "x-coordinate "+fixed1(*p.x))
		}
		if p.y != nil {
			*description = append(*description, "y-coordinate "+fixed1(*p.y))
		}
	}
	if p.radius != nil {
		*description = append(*description, "radius "+fixed1(*p.radius))
	}
}

func checkContains(methodName string, path *ui.Path, includes, excludes []ui.Offset) {
	for _, offset := range includes {
		if !path.Contains(offset) {
			panic(flutterError("It called " + methodName + " with a path that unexpectedly did not contain " + offset.String() + "."))
		}
	}
	for _, offset := range excludes {
		if path.Contains(offset) {
			panic(flutterError("It called " + methodName + " with a path that unexpectedly contained " + offset.String() + "."))
		}
	}
}

func describeContains(includes, excludes []ui.Offset) []string {
	switch {
	case includes != nil && excludes != nil:
		return []string{"that contains " + dartToString(includes) + " and does not contain " + dartToString(excludes)}
	case includes != nil:
		return []string{"that contains " + dartToString(includes)}
	case excludes != nil:
		return []string{"that does not contain " + dartToString(excludes)}
	}
	return nil
}

type pathPaintPredicate struct {
	drawCommandPaintPredicate
	includes, excludes []ui.Offset
}

func (p *pathPaintPredicate) verifyArguments(arguments []any) {
	p.drawCommandPaintPredicate.verifyArguments(arguments)
	checkContains(p.methodName(), arguments[0].(*ui.Path), p.includes, p.excludes)
}

func (p *pathPaintPredicate) debugFillDescription(description *[]string) {
	p.drawCommandPaintPredicate.debugFillDescription(description)
	*description = append(*description, describeContains(p.includes, p.excludes)...)
}

type linePaintPredicate struct {
	drawCommandPaintPredicate
	p1, p2 *ui.Offset
}

func (p *linePaintPredicate) verifyArguments(arguments []any) {
	p.drawCommandPaintPredicate.verifyArguments(arguments) // Checks the 3rd argument, a Paint
	if len(arguments) != 3 {
		panic(flutterError(fmt.Sprintf("It called %s with %d arguments; expected 3.", p.methodName(), len(arguments))))
	}
	p1Argument, p2Argument := arguments[0].(ui.Offset), arguments[1].(ui.Offset)
	if p.p1 != nil && p1Argument != *p.p1 {
		panic(flutterError(fmt.Sprintf("It called %s with p1 endpoint, %v, which was not exactly the expected endpoint (%v).",
			p.methodName(), p1Argument, *p.p1)))
	}
	if p.p2 != nil && p2Argument != *p.p2 {
		panic(flutterError(fmt.Sprintf("It called %s with p2 endpoint, %v, which was not exactly the expected endpoint (%v).",
			p.methodName(), p2Argument, *p.p2)))
	}
}

func (p *linePaintPredicate) debugFillDescription(description *[]string) {
	p.drawCommandPaintPredicate.debugFillDescription(description)
	if p.p1 != nil {
		*description = append(*description, "end point p1: "+p.p1.String())
	}
	if p.p2 != nil {
		*description = append(*description, "end point p2: "+p.p2.String())
	}
}

type arcPaintPredicate struct {
	drawCommandPaintPredicate
	rect                   *ui.Rect
	startAngle, sweepAngle *float64
	useCenter              *bool
}

func (p *arcPaintPredicate) verifyArguments(arguments []any) {
	p.drawCommandPaintPredicate.verifyArguments(arguments)
	rectArgument := arguments[0].(ui.Rect)
	if p.rect != nil && rectArgument != *p.rect {
		panic(flutterError(fmt.Sprintf("It called %s with a paint whose rect, %v, was not exactly the expected rect (%v).",
			p.methodName(), rectArgument, *p.rect)))
	}
	if a := arguments[1].(float64); p.startAngle != nil && a != *p.startAngle {
		panic(flutterError(fmt.Sprintf("It called %s with a start angle, %s, which was not exactly the expected start angle (%s).",
			p.methodName(), dart.DoubleToString(a), dart.DoubleToString(*p.startAngle))))
	}
	if a := arguments[2].(float64); p.sweepAngle != nil && a != *p.sweepAngle {
		panic(flutterError(fmt.Sprintf("It called %s with a sweep angle, %s, which was not exactly the expected sweep angle (%s).",
			p.methodName(), dart.DoubleToString(a), dart.DoubleToString(*p.sweepAngle))))
	}
	if u := arguments[3].(bool); p.useCenter != nil && u != *p.useCenter {
		panic(flutterError(fmt.Sprintf("It called %s with a useCenter value, %v, which was not exactly the expected value (%v).",
			p.methodName(), u, *p.useCenter)))
	}
}

func (p *arcPaintPredicate) debugFillDescription(description *[]string) {
	p.drawCommandPaintPredicate.debugFillDescription(description)
	if p.rect != nil {
		*description = append(*description, "rect "+p.rect.String())
	}
	if p.startAngle != nil {
		*description = append(*description, "startAngle "+dart.DoubleToString(*p.startAngle))
	}
	if p.sweepAngle != nil {
		*description = append(*description, "sweepAngle "+dart.DoubleToString(*p.sweepAngle))
	}
	if p.useCenter != nil {
		*description = append(*description, fmt.Sprintf("useCenter %v", *p.useCenter))
	}
}

type shadowPredicate struct{ o PaintPatternShadowOptions }

func (p *shadowPredicate) verifyArguments(arguments []any) {
	const methodName = "drawShadow"
	if len(arguments) != 4 {
		panic(flutterError(fmt.Sprintf("It called %s with %d arguments; expected 4.", methodName, len(arguments))))
	}
	checkContains(methodName, arguments[0].(*ui.Path), p.o.Includes, p.o.Excludes)
	actualColor := arguments[1].(ui.Color)
	if p.o.Color != nil && !colorsMatch(actualColor, p.o.Color) {
		panic(flutterError(fmt.Sprintf("It called %s with a color, %v, which was not exactly the expected color (%v).",
			methodName, actualColor, *p.o.Color)))
	}
	if e := arguments[2].(float64); p.o.Elevation != nil && e != *p.o.Elevation {
		panic(flutterError(fmt.Sprintf("It called %s with an elevation, %s, which was not exactly the expected value (%s).",
			methodName, dart.DoubleToString(e), dart.DoubleToString(*p.o.Elevation))))
	}
	if o := arguments[3].(bool); p.o.TransparentOccluder != nil && o != *p.o.TransparentOccluder {
		panic(flutterError(fmt.Sprintf("It called %s with a transparentOccluder value, %v, which was not exactly the expected value (%v).",
			methodName, o, *p.o.TransparentOccluder)))
	}
}

func (p *shadowPredicate) match(call *callIterator) {
	checkMethod(p, call, "drawShadow")
	p.verifyArguments(call.current().Invocation.PositionalArguments)
	call.moveNext()
}

func (p *shadowPredicate) String() string {
	description := describeContains(p.o.Includes, p.o.Excludes)
	if p.o.Color != nil {
		description = append(description, p.o.Color.String())
	}
	if p.o.Elevation != nil {
		description = append(description, "elevation: "+dart.DoubleToString(*p.o.Elevation))
	}
	if p.o.TransparentOccluder != nil {
		description = append(description, fmt.Sprintf("transparentOccluder: %v", *p.o.TransparentOccluder))
	}
	result := "drawShadow"
	if len(description) > 0 {
		result += " with " + strings.Join(description, ", ")
	}
	return result
}

type drawImagePaintPredicate struct {
	drawCommandPaintPredicate
	image *ui.Image
	x, y  *float64
}

func (p *drawImagePaintPredicate) verifyArguments(arguments []any) {
	p.drawCommandPaintPredicate.verifyArguments(arguments)
	imageArgument := arguments[0].(*ui.Image)
	if p.image != nil && !p.image.IsCloneOf(imageArgument) {
		panic(flutterError(fmt.Sprintf("It called %s with an image, %v, which was not exactly the expected image (%v).",
			p.methodName(), imageArgument, p.image)))
	}
	// Flutter 3.50.1 reads the offset from the first argument (the image):
	// the cast always throws.
	pointArgument, ok := arguments[0].(ui.Offset)
	if !ok {
		panic(typeError("type 'Image' is not a subtype of type 'Offset' in type cast"))
	}
	if p.x != nil && p.y != nil {
		if point := ui.NewOffset(*p.x, *p.y); point != pointArgument {
			panic(flutterError(fmt.Sprintf("It called %s with an offset coordinate, %v, which was not exactly the expected coordinate (%v).",
				p.methodName(), pointArgument, point)))
		}
	}
}

func (p *drawImagePaintPredicate) debugFillDescription(description *[]string) {
	p.drawCommandPaintPredicate.debugFillDescription(description)
	if p.image != nil {
		*description = append(*description, "image "+p.image.String())
	}
	if p.x != nil && p.y != nil {
		*description = append(*description, "point "+ui.NewOffset(*p.x, *p.y).String())
	} else {
		if p.x != nil {
			*description = append(*description, "x-coordinate "+fixed1(*p.x))
		}
		if p.y != nil {
			*description = append(*description, "y-coordinate "+fixed1(*p.y))
		}
	}
}

type drawImageRectPaintPredicate struct {
	drawCommandPaintPredicate
	image               *ui.Image
	source, destination *ui.Rect
}

func (p *drawImageRectPaintPredicate) verifyArguments(arguments []any) {
	p.drawCommandPaintPredicate.verifyArguments(arguments)
	imageArgument := arguments[0].(*ui.Image)
	if p.image != nil && !p.image.IsCloneOf(imageArgument) {
		panic(flutterError(fmt.Sprintf("It called %s with an image, %v, which was not exactly the expected image (%v).",
			p.methodName(), imageArgument, p.image)))
	}
	if s := arguments[1].(ui.Rect); p.source != nil && s != *p.source {
		panic(flutterError(fmt.Sprintf("It called %s with a source rectangle, %v, which was not exactly the expected rectangle (%v).",
			p.methodName(), s, *p.source)))
	}
	if d := arguments[2].(ui.Rect); p.destination != nil && d != *p.destination {
		panic(flutterError(fmt.Sprintf("It called %s with a destination rectangle, %v, which was not exactly the expected rectangle (%v).",
			p.methodName(), d, *p.destination)))
	}
}

func (p *drawImageRectPaintPredicate) debugFillDescription(description *[]string) {
	p.drawCommandPaintPredicate.debugFillDescription(description)
	if p.image != nil {
		*description = append(*description, "image "+p.image.String())
	}
	if p.source != nil {
		*description = append(*description, "source "+p.source.String())
	}
	if p.destination != nil {
		*description = append(*description, "destination "+p.destination.String())
	}
}

func runPredicate(predicate PaintPatternPredicate, step string, methodName string, arguments []any) (ok bool) {
	defer func() {
		if r := recover(); r != nil {
			if s, isString := r.(string); isString {
				panic(flutterError("It painted something that the predicate passed to " + step +
					" step in the paint pattern considered incorrect:\n      " + s + "\n  "))
			}
			panic(r)
		}
	}()
	return predicate(methodName, arguments)
}

type somethingPaintPredicate struct{ predicate PaintPatternPredicate }

func (p *somethingPaintPredicate) match(call *callIterator) {
	testedAllCalls := false
	for {
		if testedAllCalls {
			panic(flutterError(`It painted methods that the predicate passed to a "something" step, ` +
				"in the paint pattern, none of which were considered correct."))
		}
		currentCall := call.current()
		testedAllCalls = !call.moveNext()
		if runPredicate(p.predicate, `a "something"`, currentCall.Invocation.MemberName, currentCall.Invocation.PositionalArguments) {
			return
		}
	}
}

func (p *somethingPaintPredicate) String() string { return `a "something" step` }

type everythingPaintPredicate struct{ predicate PaintPatternPredicate }

func (p *everythingPaintPredicate) match(call *callIterator) {
	for {
		currentCall := call.current()
		if !runPredicate(p.predicate, `an "everything"`, currentCall.Invocation.MemberName, currentCall.Invocation.PositionalArguments) {
			panic(flutterError(`It painted something that the predicate passed to an "everything" step in the paint pattern considered incorrect.` + "\n"))
		}
		if !call.moveNext() {
			return
		}
	}
}

func (p *everythingPaintPredicate) String() string { return `an "everything" step` }

// functionPaintPredicate is _FunctionPaintPredicate: a method name and
// expected positional arguments (nil: any; a Matcher: matched).
type functionPaintPredicate struct {
	symbol    string
	arguments []any
}

func (p *functionPaintPredicate) match(call *callIterator) {
	checkMethod(p, call, p.symbol)
	actual := call.current().Invocation.PositionalArguments
	if len(actual) != len(p.arguments) {
		panic(flutterError(fmt.Sprintf("It called %s with %d arguments; expected %d.", p.symbol, len(actual), len(p.arguments))))
	}
	for index, desired := range p.arguments {
		actualArgument := actual[index]
		if m, ok := desired.(Matcher); ok {
			expectOrThrow(actualArgument, m)
		} else if desired != nil && !dartEquals(desired, actualArgument) {
			panic(flutterError(fmt.Sprintf("It called %s with argument %d having value %s when %s was expected.",
				p.symbol, index, valueName(actualArgument), valueName(desired))))
		}
	}
	call.moveNext()
}

func (p *functionPaintPredicate) String() string {
	adjectives := make([]string, len(p.arguments))
	for i, a := range p.arguments {
		if a != nil {
			adjectives[i] = valueName(a)
		} else {
			adjectives[i] = "..."
		}
	}
	return p.symbol + "(" + strings.Join(adjectives, ", ") + ")"
}

type saveRestorePairPaintPredicate struct{}

func (p *saveRestorePairPaintPredicate) match(call *callIterator) {
	checkMethod(p, call, "save")
	depth := 1
	for depth > 0 {
		if !call.moveNext() {
			panic(flutterError("It did not have a matching restore() for the save() that was found where " + p.String() + " was expected."))
		}
		switch call.current().Invocation.MemberName {
		case "save":
			depth++
		case "restore":
			depth--
		}
	}
	call.moveNext()
}

func (p *saveRestorePairPaintPredicate) String() string { return "a matching save/restore pair" }
