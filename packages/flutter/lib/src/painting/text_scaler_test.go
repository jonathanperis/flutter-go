package painting

// Port of packages/flutter/test/painting/text_scaler_test.dart (3.50.1).
//
// Skipped / adapted:
//   - 'SystemTextScaler equality' needs widgets' MediaQueryData.fromView and
//     the test platform dispatcher's textScaleFactorTestValue (not ported).
//   - 'Reclamping' and 'ClampedScaler asserts' start from
//     MediaQueryData.fromView(tester.view).textScaler, which in a test is
//     TextScaler.noScaling (textScaleFactor 1.0): ported with
//     TextScalerNoScaling in its place.
// Go-only additions (marked): scale/textScaleFactor values, toString, the
// enums of text_painter.dart ported so far, and a TextScalerBase subclass.

import (
	"math"
	"strings"
	"testing"

	"github.com/jonathanperis/flutter-go/src/dart"
)

func tsF(v float64) *float64 { return &v }

func tsClamp(lo, hi *float64) TextScalerClampOptions {
	return TextScalerClampOptions{MinScaleFactor: lo, MaxScaleFactor: hi}
}

func tsExpectAssert(t *testing.T, contains string, f func()) {
	t.Helper()
	defer func() {
		t.Helper()
		r := recover()
		e, ok := r.(*dart.AssertionError)
		if !ok || !strings.Contains(e.Error(), contains) {
			t.Errorf("got %v, want an AssertionError containing %q", r, contains)
		}
	}()
	f()
}

func TestLinearTextScalerEquality(t *testing.T) {
	a := NewTextScalerLinear(3.0)
	b := TextScalerNoScaling.Clamp(tsClamp(tsF(3.0), nil))
	c := NewTextScalerLinear(3.0)
	d := TextScalerNoScaling.Clamp(tsClamp(tsF(1), tsF(5))).Clamp(tsClamp(tsF(3), tsF(6)))
	list := []TextScaler{a, b, c, d}
	for _, lhs := range list {
		for _, x := range list {
			if x != lhs {
				t.Errorf("%v != %v", x, lhs)
			}
		}
	}
}

func TestLinearTextScalerClamping(t *testing.T) {
	for _, c := range [][2]TextScaler{
		{TextScalerNoScaling.Clamp(tsClamp(tsF(3.0), nil)), NewTextScalerLinear(3.0)},
		{NewTextScalerLinear(5.0).Clamp(tsClamp(nil, tsF(3.0))), NewTextScalerLinear(3.0)},
		{NewTextScalerLinear(5.0).Clamp(tsClamp(tsF(3.0), tsF(3.0))), NewTextScalerLinear(3.0)},
	} {
		if c[0] != c[1] {
			t.Errorf("%v != %v", c[0], c[1])
		}
	}
	tsExpectAssert(t, "maxScaleFactor >= minScaleFactor", func() {
		TextScalerNoScaling.Clamp(tsClamp(tsF(5.0), tsF(4.0)))
	})
}

func TestTextScalerReclamping(t *testing.T) {
	defaultScaler := TextScalerNoScaling
	// Does not raise maxScale > minScale.
	scaler1 := defaultScaler.Clamp(tsClamp(tsF(1), tsF(1.5)))
	if s := scaler1.Clamp(tsClamp(tsF(0.5), tsF(1))); s != TextScalerNoScaling {
		t.Errorf("scaler2 %v", s)
	}
	// No overlap, first scaler < second scaler: uses new minScale of last clamp.
	if s := defaultScaler.Clamp(tsClamp(tsF(1), tsF(2))).Clamp(tsClamp(tsF(3), tsF(4))); s != NewTextScalerLinear(3) {
		t.Errorf("scaler4 %v", s)
	}
	// No overlap, first scaler > second scaler: uses new maxScale of last clamp.
	if s := defaultScaler.Clamp(tsClamp(tsF(5), tsF(6))).Clamp(tsClamp(tsF(3), tsF(4))); s != NewTextScalerLinear(4) {
		t.Errorf("scaler6 %v", s)
	}
	// Overlap, combine clamps.
	if s := defaultScaler.Clamp(tsClamp(tsF(1), tsF(3))).Clamp(tsClamp(tsF(2), tsF(4))); s != defaultScaler.Clamp(tsClamp(tsF(2), tsF(3))) {
		t.Errorf("scaler8 %v", s)
	}
}

func TestClampedScalerAsserts(t *testing.T) {
	clamped := TextScalerNoScaling.Clamp(tsClamp(tsF(1), tsF(5)))
	tsExpectAssert(t, "maxScaleFactor >= minScaleFactor", func() { clamped.Clamp(tsClamp(tsF(3), tsF(2))) })
	tsExpectAssert(t, "maxScaleFactor >= minScaleFactor", func() { clamped.Clamp(tsClamp(tsF(3), tsF(math.NaN()))) })
	tsExpectAssert(t, "minScaleFactor >= 0", func() { clamped.Clamp(tsClamp(tsF(-1), tsF(2))) })
	tsExpectAssert(t, "minScaleFactor.isFinite", func() { clamped.Clamp(tsClamp(tsF(math.Inf(1)), nil)) })
}

// customScaler is a user-defined TextScaler (Go-only test).
type customScaler struct {
	TextScalerBase
	k float64
}

func (s *customScaler) Scale(fontSize float64) float64 { return fontSize*s.k + 1 }
func (s *customScaler) TextScaleFactor() float64       { return s.k }
func (s *customScaler) String() string                 { return "custom" }

// Go-only: values, toString and the inherited clamp.
func TestTextScalerValues(t *testing.T) {
	if got := NewTextScalerLinear(1.5).Scale(10); got != 15 {
		t.Errorf("linear scale %v", got)
	}
	c := NewTextScalerLinear(4).Clamp(tsClamp(tsF(1), tsF(3)))
	if c != NewTextScalerLinear(3) {
		t.Errorf("linear clamp %v", c)
	}
	s := &customScaler{k: 4}
	s.SetSelf(s)
	if s.Clamp() != TextScaler(s) {
		t.Error("unbounded clamp returns the scaler itself")
	}
	if s.Clamp(tsClamp(tsF(2), tsF(2))) != NewTextScalerLinear(2) {
		t.Error("collapsed clamp is linear")
	}
	cl := s.Clamp(tsClamp(tsF(1), tsF(3)))
	if got := cl.Scale(10); got != 30 { // 41 clamped to [10, 30]
		t.Errorf("clamped scale %v", got)
	}
	if got := s.Clamp(tsClamp(tsF(1), tsF(5))).Scale(10); got != 41 {
		t.Errorf("clamped scale within range %v", got)
	}
	if got := s.Clamp(tsClamp(tsF(5), tsF(6))).Scale(1); got != 5 { // 5 clamped to [5, 6]
		t.Errorf("clamped scale below %v", got)
	}
	if got := cl.TextScaleFactor(); got != 3 {
		t.Errorf("clamped textScaleFactor %v", got)
	}
	for _, c := range []struct{ got, want string }{
		{dart.ToString(TextScalerNoScaling), "no scaling"},
		{dart.ToString(NewTextScalerLinear(2)), "linear (2.0x)"},
		{dart.ToString(NewTextScalerLinear(0.25)), "linear (0.25x)"},
		{dart.ToString(cl), "custom clamped [1.0, 3.0]"},
		{dart.ToString(TextScalerNoScaling.Clamp(tsClamp(tsF(0.5), tsF(2)))), "no scaling"},
		{dart.ToString(clampedTextScaler{NewTextScalerLinear(2), 0.5, math.Inf(1)}), "linear (2.0x) clamped [0.5, Infinity]"},
		{TextOverflowEllipsis.String(), "TextOverflow.ellipsis"},
		{TextOverflowVisible.String(), "TextOverflow.visible"},
		{TextOverflowClip.String(), "TextOverflow.clip"},
		{TextOverflowFade.String(), "TextOverflow.fade"},
		{TextWidthBasisParent.String(), "TextWidthBasis.parent"},
		{TextWidthBasisLongestLine.String(), "TextWidthBasis.longestLine"},
	} {
		if c.got != c.want {
			t.Errorf("got %q, want %q", c.got, c.want)
		}
	}
	tsExpectAssert(t, "textScaleFactor >= 0", func() { NewTextScalerLinear(-1) })
	tsExpectAssert(t, "fontSize >= 0", func() { NewTextScalerLinear(1).Scale(-1) })
	tsExpectAssert(t, "fontSize.isFinite", func() { cl.Scale(math.Inf(1)) })
	if KDefaultFontSize != 14.0 {
		t.Error("kDefaultFontSize")
	}
}
