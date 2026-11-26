package painting

// Port of decoration.dart (Flutter 3.50.1): Decoration and BoxPainter.
//
// Both are abstract classes users extend: interfaces plus the embeddable
// DecorationBase (Diagnosticable, with the defaults; call SetSelf) and
// BoxPainterBase (onChanged and the default dispose). Dart's named
// textDirection of hitTest is a positional *ui.TextDirection (nil is
// null); createBoxPainter's optional onChanged is a func() (nil is null).

import (
	"reflect"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	"github.com/jonathanperis/flutter-go/src/dart"
)

// Decoration is a description of a box decoration (a decoration applied
// to a Rect).
type Decoration interface {
	foundation.Diagnosticable
	// DebugAssertIsValid asserts that the decoration is valid (it is only
	// called in debug builds).
	DebugAssertIsValid() bool
	// Padding is the space the decoration's border takes up inside the
	// box (EdgeInsets.zero by default).
	Padding() EdgeInsetsGeometry
	// IsComplex reports whether the decoration is complex enough to
	// benefit from caching its painting.
	IsComplex() bool
	// LerpFrom is Dart's @protected lerpFrom (a nil: null); nil is "can't
	// lerp".
	LerpFrom(a Decoration, t float64) Decoration
	// LerpTo is Dart's @protected lerpTo.
	LerpTo(b Decoration, t float64) Decoration
	// HitTest tests whether the given point, on a rectangle of the given
	// size, would be considered to hit the decoration.
	HitTest(size ui.Size, position ui.Offset, textDirection *ui.TextDirection) bool
	// CreateBoxPainter returns a BoxPainter that will paint this
	// decoration (onChanged nil: null).
	CreateBoxPainter(onChanged func()) BoxPainter
	// GetClipPath returns a closed Path that describes the outer edge of
	// the decoration.
	GetClipPath(rect ui.Rect, textDirection ui.TextDirection) *ui.Path
}

// DecorationBase is the embeddable base of Dart's abstract Decoration:
// Diagnosticable, with Decoration's default members. A type embedding it
// calls SetSelf(outer) when created.
type DecorationBase struct {
	foundation.DiagnosticableMixin
	self Decoration
}

// SetSelf points the base (and its Diagnosticable mixin) at the outer
// decoration.
func (d *DecorationBase) SetSelf(self Decoration) {
	d.self = self
	d.DiagnosticableMixin.SetSelf(self)
}

func (d *DecorationBase) outerDecoration() Decoration {
	if d.self == nil {
		panic("painting: DecorationBase used before SetSelf; call SetSelf with the outer decoration when creating it")
	}
	return d.self
}

// ToStringShort is the decoration's runtime type.
func (d *DecorationBase) ToStringShort() string {
	return foundation.ObjectRuntimeType(d.outerDecoration(), "Decoration")
}

// DebugAssertIsValid returns true.
func (d *DecorationBase) DebugAssertIsValid() bool { return true }

// Padding is EdgeInsets.zero.
func (d *DecorationBase) Padding() EdgeInsetsGeometry { return EdgeInsetsZero }

// IsComplex is false.
func (d *DecorationBase) IsComplex() bool { return false }

// LerpFrom returns nil (can't lerp).
func (d *DecorationBase) LerpFrom(a Decoration, t float64) Decoration { return nil }

// LerpTo returns nil (can't lerp).
func (d *DecorationBase) LerpTo(b Decoration, t float64) Decoration { return nil }

// HitTest returns true.
func (d *DecorationBase) HitTest(size ui.Size, position ui.Offset, textDirection *ui.TextDirection) bool {
	return true
}

// GetClipPath panics with UnsupportedError: this decoration does not
// expect to be used for clipping.
func (d *DecorationBase) GetClipPath(rect ui.Rect, textDirection ui.TextDirection) *ui.Path {
	panic(dart.UnsupportedError{Message: foundation.ObjectRuntimeType(d.outerDecoration(), "This Decoration subclass") +
		" does not expect to be used for clipping."})
}

// isNilValue reports whether v is nil or a nil pointer in an interface
// (Dart's null).
func isNilValue(v any) bool {
	if v == nil {
		return true
	}
	r := reflect.ValueOf(v)
	switch r.Kind() {
	case reflect.Pointer, reflect.Interface, reflect.Map, reflect.Slice, reflect.Func:
		return r.IsNil()
	}
	return false
}

func orDecoration(d, otherwise Decoration) Decoration {
	if isNilValue(d) {
		return otherwise
	}
	return d
}

// DecorationLerp is Decoration.lerp: linearly interpolates between two
// decorations (nil is null).
func DecorationLerp(a, b Decoration, t float64) Decoration {
	if isNilValue(a) {
		a = nil
	}
	if isNilValue(b) {
		b = nil
	}
	if sameObject(a, b) {
		return a
	}
	if a == nil {
		return orDecoration(b.LerpFrom(nil, t), b)
	}
	if b == nil {
		return orDecoration(a.LerpTo(nil, t), a)
	}
	if t == 0.0 {
		return a
	}
	if t == 1.0 {
		return b
	}
	if r := b.LerpFrom(a, t); !isNilValue(r) {
		return r
	}
	if r := a.LerpTo(b, t); !isNilValue(r) {
		return r
	}
	if t < 0.5 {
		return orDecoration(a.LerpTo(nil, t*2.0), a)
	}
	return orDecoration(b.LerpFrom(nil, (t-0.5)*2.0), b)
}

// BoxPainter is a stateful class that can paint a particular Decoration.
type BoxPainter interface {
	// Paint paints the decoration at offset; configuration.Size is the
	// size of the box.
	Paint(canvas ui.Canvas, offset ui.Offset, configuration ImageConfiguration)
	// OnChanged is the callback called when the painter needs repainting
	// (nil: null).
	OnChanged() func()
	// Dispose discards any resources being held by the object.
	Dispose()
}

// BoxPainterBase is the embeddable base of Dart's abstract BoxPainter: the
// onChanged callback and the default Dispose.
type BoxPainterBase struct{ onChanged func() }

// NewBoxPainterBase is BoxPainter([onChanged]), for embedding.
func NewBoxPainterBase(onChanged func()) BoxPainterBase { return BoxPainterBase{onChanged} }

// OnChanged is the callback the painter calls when it needs repainting.
func (p *BoxPainterBase) OnChanged() func() { return p.onChanged }

// Dispose does nothing; overrides must call it (Dart's @mustCallSuper).
func (p *BoxPainterBase) Dispose() {}
