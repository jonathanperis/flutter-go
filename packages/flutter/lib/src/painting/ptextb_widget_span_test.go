package painting

// A test stand-in for widgets' WidgetSpan (widget_span.dart, 3.50.1), the
// one PlaceholderSpan Flutter's painting tests use: the same visit,
// position, code unit, build, compareTo and == rules, with its child
// widget replaced by a comparable value (Flutter's tests use const
// widgets, which Dart canonicalizes: equal consts are identical).

import (
	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/src/dart"
)

type ptextbWidgetSpan struct {
	PlaceholderSpanBase
	child any
}

func newPtextbWidgetSpan(child any, opts ...PlaceholderSpanOptions) *ptextbWidgetSpan {
	s := &ptextbWidgetSpan{child: child}
	s.InitPlaceholderSpan(s, opts...)
	return s
}

func (s *ptextbWidgetSpan) Build(builder ui.ParagraphBuilder, opts ...InlineSpanBuildOptions) {
	var o InlineSpanBuildOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	if o.Dimensions == nil {
		panic(dart.NewAssertionError(nil))
	}
	if s.Style() != nil {
		builder.PushStyle(s.Style().GetTextStyle(TextStyleGetTextStyleOptions{TextScaler: o.TextScaler}))
	}
	d := o.Dimensions[builder.PlaceholderCount()]
	builder.AddPlaceholder(d.Size.Width, d.Size.Height, s.Alignment(),
		ui.AddPlaceholderOptions{Baseline: d.Baseline, BaselineOffset: d.BaselineOffset})
	if s.Style() != nil {
		builder.Pop()
	}
}

func (s *ptextbWidgetSpan) VisitChildren(visitor InlineSpanVisitor) bool { return visitor(s) }

func (s *ptextbWidgetSpan) VisitDirectChildren(InlineSpanVisitor) bool { return true }

func (s *ptextbWidgetSpan) GetSpanForPositionVisitor(position ui.TextPosition, offset *Accumulator) InlineSpan {
	if position.Offset == offset.Value() {
		return s
	}
	offset.Increment(1)
	return nil
}

func (s *ptextbWidgetSpan) GetSpanForPosition(ui.TextPosition) InlineSpan { return nil }

func (s *ptextbWidgetSpan) CodeUnitAtVisitor(index int, offset *Accumulator) *int {
	localOffset := index - offset.Value()
	offset.Increment(1)
	if localOffset == 0 {
		v := PlaceholderSpanPlaceholderCodeUnit
		return &v
	}
	return nil
}

func (s *ptextbWidgetSpan) CompareTo(other InlineSpan) RenderComparison {
	if InlineSpan(s) == other {
		return RenderComparisonIdentical
	}
	o, ok := other.(*ptextbWidgetSpan)
	if !ok {
		return RenderComparisonLayout
	}
	if (s.Style() == nil) != (o.Style() == nil) {
		return RenderComparisonLayout
	}
	if s.child != o.child || s.Alignment() != o.Alignment() {
		return RenderComparisonLayout
	}
	result := RenderComparisonIdentical
	if s.Style() != nil {
		if c := s.Style().CompareTo(o.Style()); c > result {
			result = c
		}
	}
	return result
}

func (s *ptextbWidgetSpan) Equal(other InlineSpan) bool {
	if InlineSpan(s) == other {
		return true
	}
	o, ok := other.(*ptextbWidgetSpan)
	return ok && s.InlineSpanBase.Equal(other) && o.child == s.child && o.Alignment() == s.Alignment() &&
		ptrEq(o.Baseline(), s.Baseline())
}

func (s *ptextbWidgetSpan) DebugAssertIsValid() bool { return true }

// ptextbSizedBox and ptextbTextRich stand for the const widgets the tests
// use as children.
type ptextbSizedBox struct{ w, h float64 }

type ptextbTextRich struct{ description string }
