package painting

// Port of text_scaler.dart (Flutter 3.50.1): TextScaler, its linear and
// clamped implementations. Pure values: nothing here needs dart:ui's
// Paragraph.

import (
	"math"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/src/dart"
)

// TextScaler computes the scaled font size (in logical pixels) for a given
// unscaled font size. Implementations embed TextScalerBase for the default
// Clamp (and call SetSelf), as Dart subclasses inherit it.
type TextScaler interface {
	// Scale computes the scaled font size given the original unscaled
	// font size specified by app developers.
	Scale(fontSize float64) float64
	// TextScaleFactor is the estimated number of font pixels for each
	// logical pixel (deprecated in Flutter).
	TextScaleFactor() float64
	// Clamp returns a new TextScaler that restricts the scaling factor to
	// [MinScaleFactor, MaxScaleFactor] (Dart's defaults: 0 and infinity).
	Clamp(opts ...TextScalerClampOptions) TextScaler
}

// TextScalerClampOptions are TextScaler.clamp's named parameters; nil is
// the Dart default (minScaleFactor 0, maxScaleFactor infinity).
type TextScalerClampOptions struct {
	MinScaleFactor *float64
	MaxScaleFactor *float64
}

func (o TextScalerClampOptions) resolve() (lo, hi float64) {
	lo, hi = 0, math.Inf(1)
	if o.MinScaleFactor != nil {
		lo = *o.MinScaleFactor
	}
	if o.MaxScaleFactor != nil {
		hi = *o.MaxScaleFactor
	}
	if !(hi >= lo) {
		panic(dart.NewAssertionError("maxScaleFactor >= minScaleFactor"))
	}
	if math.IsNaN(hi) {
		panic(dart.NewAssertionError("!maxScaleFactor.isNaN"))
	}
	if math.IsInf(lo, 0) || math.IsNaN(lo) {
		panic(dart.NewAssertionError("minScaleFactor.isFinite"))
	}
	if !(lo >= 0) {
		panic(dart.NewAssertionError("minScaleFactor >= 0"))
	}
	return lo, hi
}

func clampOptions(opts []TextScalerClampOptions) TextScalerClampOptions {
	if len(opts) > 0 {
		return opts[0]
	}
	return TextScalerClampOptions{}
}

// TextScalerBase is the embeddable base of a user-defined TextScaler: it
// provides TextScaler.clamp's default implementation, calling the outer
// scaler (set with SetSelf).
type TextScalerBase struct{ self TextScaler }

// SetSelf records the TextScaler embedding the base.
func (b *TextScalerBase) SetSelf(self TextScaler) { b.self = self }

// Clamp is TextScaler.clamp's default implementation.
func (b *TextScalerBase) Clamp(opts ...TextScalerClampOptions) TextScaler {
	if b.self == nil {
		panic("painting.TextScalerBase used before SetSelf")
	}
	lo, hi := clampOptions(opts).resolve()
	if lo == 0 && math.IsInf(hi, 1) {
		return b.self
	}
	if lo == hi {
		return NewTextScalerLinear(lo)
	}
	return clampedTextScaler{b.self, lo, hi}
}

// NewTextScalerLinear is TextScaler.linear(textScaleFactor): a TextScaler
// that scales font sizes linearly. It panics if textScaleFactor < 0, as
// Dart asserts.
func NewTextScalerLinear(textScaleFactor float64) TextScaler {
	if !(textScaleFactor >= 0) {
		panic(dart.NewAssertionError("textScaleFactor >= 0"))
	}
	return linearTextScaler{textScaleFactor}
}

// TextScalerNoScaling is TextScaler.noScaling: a TextScaler that doesn't
// scale the input font size.
var TextScalerNoScaling TextScaler = linearTextScaler{1.0}

// linearTextScaler is Dart's _LinearTextScaler (compare with ==).
type linearTextScaler struct{ factor float64 }

func (s linearTextScaler) TextScaleFactor() float64 { return s.factor }

func (s linearTextScaler) Scale(fontSize float64) float64 {
	checkFontSize(fontSize)
	return fontSize * s.factor
}

func (s linearTextScaler) Clamp(opts ...TextScalerClampOptions) TextScaler {
	lo, hi := clampOptions(opts).resolve()
	newScaleFactor := ui.ClampDouble(s.factor, lo, hi)
	if newScaleFactor == s.factor {
		return s
	}
	return linearTextScaler{newScaleFactor}
}

func (s linearTextScaler) String() string {
	if s.factor == 1.0 {
		return "no scaling"
	}
	return "linear (" + dart.DoubleToString(s.factor) + "x)"
}

// clampedTextScaler is Dart's _ClampedTextScaler. Go's == compares the
// wrapped scaler with == (Dart: the scaler's ==), so it needs a comparable
// scaler, as every TextScaler flutter-go creates is.
type clampedTextScaler struct {
	scaler             TextScaler
	minScale, maxScale float64
}

func (s clampedTextScaler) TextScaleFactor() float64 {
	return ui.ClampDouble(s.scaler.TextScaleFactor(), s.minScale, s.maxScale)
}

func (s clampedTextScaler) Scale(fontSize float64) float64 {
	checkFontSize(fontSize)
	return ui.ClampDouble(s.scaler.Scale(fontSize), float64(s.minScale*fontSize), float64(s.maxScale*fontSize))
}

func (s clampedTextScaler) Clamp(opts ...TextScalerClampOptions) TextScaler {
	lo, hi := clampOptions(opts).resolve()
	newMinScale := math.Max(s.minScale, lo)
	newMaxScale := math.Min(s.maxScale, hi)
	if newMaxScale <= newMinScale {
		// Ranges don't overlap or collapse to a single point.
		return NewTextScalerLinear(ui.ClampDouble(s.minScale, lo, hi))
	}
	return clampedTextScaler{s.scaler, newMinScale, newMaxScale}
}

func (s clampedTextScaler) String() string {
	return dart.ToString(s.scaler) + " clamped [" + dart.DoubleToString(s.minScale) + ", " +
		dart.DoubleToString(s.maxScale) + "]"
}

func checkFontSize(fontSize float64) {
	if !(fontSize >= 0) {
		panic(dart.NewAssertionError("fontSize >= 0"))
	}
	if math.IsInf(fontSize, 0) {
		panic(dart.NewAssertionError("fontSize.isFinite"))
	}
}
