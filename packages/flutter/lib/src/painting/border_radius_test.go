package painting

// Port of packages/flutter/test/painting/border_radius_test.dart (Flutter
// 3.50.1). Dart's == is Go's == (same type and radii); identical on const
// values is ==.
//
// Skipped: hasOneLineDescription is checked as "String() is one non-empty
// line"; hashCode (Go values have none).

import (
	"strings"
	"testing"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/src/dart"
)

func brEll(x, y float64) ui.Radius { return ui.NewRadiusElliptical(x, y) }
func brCirc(r float64) ui.Radius   { return ui.NewRadiusCircular(r) }

func brOneLine(t *testing.T, s brStringer) {
	t.Helper()
	if str := s.String(); str == "" || strings.Contains(str, "\n") {
		t.Errorf("not a one-line description: %q", str)
	}
}

type brStringer interface{ String() string }

func eqBR(t *testing.T, got, want any) {
	t.Helper()
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestBorderRadiusControl(t *testing.T) {
	rect := ui.NewRectFromLTRB(19, 23, 29, 31)
	br := NewBorderRadiusAll(brEll(5, 7))
	brOneLine(t, br)
	eqBR(t, br, BorderRadius{brEll(5, 7), brEll(5, 7), brEll(5, 7), brEll(5, 7)})
	eqBR(t, br.ToRRect(rect), ui.NewRRectFromRectXY(rect, 5, 7))

	br = NewBorderRadiusCircular(3)
	brOneLine(t, br)
	eqBR(t, br, BorderRadius{brEll(3, 3), brEll(3, 3), brEll(3, 3), brEll(3, 3)})
	eqBR(t, br.ToRRect(rect), ui.NewRRectFromRectXY(rect, 3, 3))

	r1, r2 := brEll(89, 87), brEll(103, 107)
	br = NewBorderRadiusVertical(BorderRadiusVerticalOptions{Top: r1, Bottom: r2})
	brOneLine(t, br)
	eqBR(t, br, BorderRadius{r1, r1, r2, r2})
	eqBR(t, br.ToRRect(rect), ui.NewRRectFromRectAndCorners(rect, ui.RRectCorners{TopLeft: r1, TopRight: r1, BottomLeft: r2, BottomRight: r2}))

	br = NewBorderRadiusHorizontal(BorderRadiusHorizontalOptions{Left: r1, Right: r2})
	brOneLine(t, br)
	eqBR(t, br, BorderRadius{r1, r2, r1, r2})
	eqBR(t, br.ToRRect(rect), ui.NewRRectFromRectAndCorners(rect, ui.RRectCorners{TopLeft: r1, TopRight: r2, BottomLeft: r1, BottomRight: r2}))

	br = BorderRadiusZero
	brOneLine(t, br)
	eqBR(t, br, BorderRadius{})
	eqBR(t, br.ToRRect(rect), ui.NewRRectFromRectAndCorners(rect, ui.RRectCorners{}))

	br = BorderRadius{TopRight: r1, BottomRight: r2}
	brOneLine(t, br)
	eqBR(t, br.ToRRect(rect), ui.NewRRectFromRectAndCorners(rect, ui.RRectCorners{TopRight: r1, BottomRight: r2}))

	eqBR(t, BorderRadius{TopLeft: brEll(1, 2)}.Subtract(BorderRadius{TopLeft: brEll(3, 5)}), BorderRadiusGeometry(BorderRadius{TopLeft: brEll(-2, -3)}))
	eqBR(t, BorderRadius{TopRight: brEll(1, 2)}.AddGeometry(BorderRadius{TopLeft: brEll(3, 5)}),
		BorderRadiusGeometry(BorderRadius{TopLeft: brEll(3, 5), TopRight: brEll(1, 2)}))
	eqBR(t, BorderRadius{TopLeft: brEll(1, 2)}.Sub(BorderRadius{TopLeft: brEll(3, 5)}), BorderRadius{TopLeft: brEll(-2, -3)})
	eqBR(t, BorderRadius{TopRight: brEll(1, 2)}.Add(BorderRadius{TopLeft: brEll(3, 5)}), BorderRadius{TopLeft: brEll(3, 5), TopRight: brEll(1, 2)})
	eqBR(t, BorderRadius{TopLeft: brEll(1, 2)}.Neg(), BorderRadius{TopLeft: brEll(-1, -2)})
	eqBR(t, BorderRadius{r1, r2, r2, r1}.Mul(0), BorderRadiusZero)
	eqBR(t, NewBorderRadiusAll(brCirc(15)).Div(10), NewBorderRadiusAll(brCirc(1.5)))
	eqBR(t, NewBorderRadiusAll(brCirc(15)).TruncDiv(10), NewBorderRadiusAll(brCirc(1)))
	eqBR(t, NewBorderRadiusAll(brCirc(15)).Mod(10), NewBorderRadiusAll(brCirc(5)))
}

func TestBorderRadiusLerp(t *testing.T) {
	a, b := NewBorderRadiusAll(brCirc(10)), NewBorderRadiusAll(brCirc(20))
	got := *BorderRadiusLerp(&a, &b, 0.25)
	eqBR(t, got, a.Mul(1.25))
	eqBR(t, got, b.Mul(0.625))
	eqBR(t, got, a.Add(NewBorderRadiusAll(brCirc(2.5))))
	eqBR(t, got, b.Sub(NewBorderRadiusAll(brCirc(7.5))))
	if BorderRadiusLerp(nil, nil, 0.25) != nil {
		t.Error("null")
	}
	eqBR(t, *BorderRadiusLerp(nil, &b, 0.25), b.Mul(0.25))
	eqBR(t, *BorderRadiusLerp(&a, nil, 0.25), a.Mul(0.75))
	z := BorderRadiusZero
	if BorderRadiusLerp(nil, nil, 0) != nil || BorderRadiusLerp(&z, &z, 0.5) != &z {
		t.Error("identical")
	}
	// crazy
	ca := BorderRadius{TopLeft: brEll(10, 20), TopRight: brEll(30, 40), BottomLeft: brEll(50, 60)}
	cb := BorderRadius{TopRight: brEll(100, 110), BottomLeft: brEll(120, 130), BottomRight: brEll(140, 150)}
	eqBR(t, *BorderRadiusLerp(&ca, &cb, 0.5), BorderRadius{brEll(5, 10), brEll(65, 75), brEll(85, 95), brEll(70, 75)})
}

func TestBorderRadiusDirectionalControl(t *testing.T) {
	rect := ui.NewRectFromLTRB(19, 23, 29, 31)
	res := func(b BorderRadiusDirectional, d ui.TextDirection) ui.RRect { return b.Resolve(&d).ToRRect(rect) }
	br := NewBorderRadiusDirectionalAll(brEll(5, 7))
	brOneLine(t, br)
	eqBR(t, br, BorderRadiusDirectional{brEll(5, 7), brEll(5, 7), brEll(5, 7), brEll(5, 7)})
	eqBR(t, res(br, ui.TextDirectionLtr), ui.NewRRectFromRectXY(rect, 5, 7))
	eqBR(t, res(br, ui.TextDirectionRtl), ui.NewRRectFromRectXY(rect, 5, 7))

	br = NewBorderRadiusDirectionalAll(brCirc(3))
	brOneLine(t, br)
	eqBR(t, br, BorderRadiusDirectional{brEll(3, 3), brEll(3, 3), brEll(3, 3), brEll(3, 3)})
	eqBR(t, res(br, ui.TextDirectionLtr), ui.NewRRectFromRectXY(rect, 3, 3))
	eqBR(t, res(br, ui.TextDirectionRtl), ui.NewRRectFromRectXY(rect, 3, 3))

	r1, r2 := brEll(89, 87), brEll(103, 107)
	br = NewBorderRadiusDirectionalVertical(BorderRadiusVerticalOptions{Top: r1, Bottom: r2})
	brOneLine(t, br)
	eqBR(t, br, BorderRadiusDirectional{r1, r1, r2, r2})
	want := ui.NewRRectFromRectAndCorners(rect, ui.RRectCorners{TopLeft: r1, TopRight: r1, BottomLeft: r2, BottomRight: r2})
	eqBR(t, res(br, ui.TextDirectionLtr), want)
	eqBR(t, res(br, ui.TextDirectionRtl), want)

	br = NewBorderRadiusDirectionalHorizontal(BorderRadiusDirectionalHorizontalOptions{Start: r1, End: r2})
	brOneLine(t, br)
	eqBR(t, br, BorderRadiusDirectional{r1, r2, r1, r2})
	eqBR(t, res(br, ui.TextDirectionLtr), ui.NewRRectFromRectAndCorners(rect, ui.RRectCorners{TopLeft: r1, TopRight: r2, BottomLeft: r1, BottomRight: r2}))
	eqBR(t, res(br, ui.TextDirectionRtl), ui.NewRRectFromRectAndCorners(rect, ui.RRectCorners{TopLeft: r2, TopRight: r1, BottomLeft: r2, BottomRight: r1}))

	br = BorderRadiusDirectionalZero
	brOneLine(t, br)
	eqBR(t, res(br, ui.TextDirectionLtr), ui.NewRRectFromRectAndCorners(rect, ui.RRectCorners{}))
	eqBR(t, res(br, ui.TextDirectionRtl), ui.NewRRectFromRectAndCorners(rect, ui.RRectCorners{}))

	br = BorderRadiusDirectional{TopEnd: r1, BottomEnd: r2}
	brOneLine(t, br)
	eqBR(t, res(br, ui.TextDirectionLtr), ui.NewRRectFromRectAndCorners(rect, ui.RRectCorners{TopRight: r1, BottomRight: r2}))
	eqBR(t, res(br, ui.TextDirectionRtl), ui.NewRRectFromRectAndCorners(rect, ui.RRectCorners{TopLeft: r1, BottomLeft: r2}))

	eqBR(t, BorderRadiusDirectional{TopStart: brEll(1, 2)}.Subtract(BorderRadiusDirectional{TopStart: brEll(3, 5)}),
		BorderRadiusGeometry(BorderRadiusDirectional{TopStart: brEll(-2, -3)}))
	eqBR(t, BorderRadiusDirectional{TopEnd: brEll(1, 2)}.AddGeometry(BorderRadiusDirectional{TopStart: brEll(3, 5)}),
		BorderRadiusGeometry(BorderRadiusDirectional{TopStart: brEll(3, 5), TopEnd: brEll(1, 2)}))
	eqBR(t, BorderRadiusDirectional{TopStart: brEll(1, 2)}.Sub(BorderRadiusDirectional{TopStart: brEll(3, 5)}), BorderRadiusDirectional{TopStart: brEll(-2, -3)})
	eqBR(t, BorderRadiusDirectional{TopEnd: brEll(1, 2)}.Add(BorderRadiusDirectional{TopStart: brEll(3, 5)}),
		BorderRadiusDirectional{TopStart: brEll(3, 5), TopEnd: brEll(1, 2)})
	eqBR(t, BorderRadiusDirectional{TopStart: brEll(1, 2)}.Neg(), BorderRadiusDirectional{TopStart: brEll(-1, -2)})
	eqBR(t, BorderRadiusDirectional{r1, r2, r2, r1}.Mul(0), BorderRadiusDirectionalZero)
	eqBR(t, NewBorderRadiusDirectionalAll(brCirc(15)).Div(10), NewBorderRadiusDirectionalAll(brCirc(1.5)))
	eqBR(t, NewBorderRadiusDirectionalAll(brCirc(15)).TruncDiv(10), NewBorderRadiusDirectionalAll(brCirc(1)))
	eqBR(t, NewBorderRadiusDirectionalAll(brCirc(15)).Mod(10), NewBorderRadiusDirectionalAll(brCirc(5)))
}

var (
	brA = BorderRadius{TopLeft: brEll(10, 20), TopRight: brEll(30, 40), BottomLeft: brEll(50, 60)}
	brB = BorderRadiusDirectional{TopEnd: brEll(100, 110), BottomStart: brEll(120, 130), BottomEnd: brEll(140, 150)}
)

func TestBorderRadiusResolveNull(t *testing.T) {
	expectNoTextDirection(t, func() { NewBorderRadiusDirectionalAll(brCirc(1)).Resolve(nil) })
	expectNoTextDirection(t, func() { brA.AddGeometry(brB).Resolve(nil) })
	expectNoTextDirection(t, func() { brB.Subtract(brA).Resolve(nil) })
}

func TestBorderRadiusDirectionalLerp(t *testing.T) {
	a, b := NewBorderRadiusDirectionalAll(brCirc(10)), NewBorderRadiusDirectionalAll(brCirc(20))
	got := *BorderRadiusDirectionalLerp(&a, &b, 0.25)
	eqBR(t, got, a.Mul(1.25))
	eqBR(t, got, b.Mul(0.625))
	eqBR(t, got, a.Add(NewBorderRadiusDirectionalAll(brCirc(2.5))))
	eqBR(t, got, b.Sub(NewBorderRadiusDirectionalAll(brCirc(7.5))))
	if BorderRadiusDirectionalLerp(nil, nil, 0.25) != nil {
		t.Error("null")
	}
	eqBR(t, *BorderRadiusDirectionalLerp(nil, &b, 0.25), b.Mul(0.25))
	eqBR(t, *BorderRadiusDirectionalLerp(&a, nil, 0.25), a.Mul(0.75))
	z := BorderRadiusDirectionalZero
	if BorderRadiusDirectionalLerp(&z, &z, 0.5) != &z {
		t.Error("identical")
	}
	ca := BorderRadiusDirectional{TopStart: brEll(10, 20), TopEnd: brEll(30, 40), BottomStart: brEll(50, 60)}
	cb := BorderRadiusDirectional{TopEnd: brEll(100, 110), BottomStart: brEll(120, 130), BottomEnd: brEll(140, 150)}
	eqBR(t, *BorderRadiusDirectionalLerp(&ca, &cb, 0.5), BorderRadiusDirectional{brEll(5, 10), brEll(65, 75), brEll(85, 95), brEll(70, 75)})
}

var (
	brLTR = BorderRadius{brEll(5, 10), brEll(65, 75), brEll(85, 95), brEll(70, 75)}
	brRTL = BorderRadius{brEll(55, 65), brEll(15, 20), brEll(95, 105), brEll(60, 65)}
)

func TestBorderRadiusGeometryLerp(t *testing.T) {
	l, r := ui.TextDirectionLtr, ui.TextDirectionRtl
	eqBR(t, BorderRadiusGeometryLerp(brA, brB, 0.5).Resolve(&l), brLTR)
	eqBR(t, BorderRadiusGeometryLerp(brA, brB, 0.5).Resolve(&r), brRTL)
	eqBR(t, BorderRadiusGeometryLerp(brA, brB, 0.0).Resolve(&l), brA)
	eqBR(t, BorderRadiusGeometryLerp(brA, brB, 1.0).Resolve(&r), brB.Resolve(&r))
	var z BorderRadiusGeometry = BorderRadiusZero
	if BorderRadiusGeometryLerp(z, z, 0.5) != z || BorderRadiusGeometryLerp(nil, nil, 0) != nil {
		t.Error("identical")
	}
}

func TestBorderRadiusGeometryAddSubtract(t *testing.T) {
	l, r := ui.TextDirectionLtr, ui.TextDirectionRtl
	z := ui.RadiusZero
	eqBR(t, brA.Subtract(brB).Resolve(&l), BorderRadius{brEll(10, 20).Sub(z), brEll(30, 40).Sub(brEll(100, 110)),
		brEll(50, 60).Sub(brEll(120, 130)), z.Sub(brEll(140, 150))})
	eqBR(t, brA.Subtract(brB).Resolve(&r), BorderRadius{brEll(10, 20).Sub(brEll(100, 110)), brEll(30, 40).Sub(z),
		brEll(50, 60).Sub(brEll(140, 150)), z.Sub(brEll(120, 130))})
	eqBR(t, brA.AddGeometry(brB).Resolve(&l), BorderRadius{brEll(10, 20).Add(z), brEll(30, 40).Add(brEll(100, 110)),
		brEll(50, 60).Add(brEll(120, 130)), z.Add(brEll(140, 150))})
	eqBR(t, brA.AddGeometry(brB).Resolve(&r), BorderRadius{brEll(10, 20).Add(brEll(100, 110)), brEll(30, 40).Add(z),
		brEll(50, 60).Add(brEll(140, 150)), z.Add(brEll(120, 130))})
	// add and multiply
	eqBR(t, brA.AddGeometry(brB).MulGeometry(0.5).Resolve(&l), BorderRadius{brEll(10, 20).Add(z).Div(2), brEll(30, 40).Add(brEll(100, 110)).Div(2),
		brEll(50, 60).Add(brEll(120, 130)).Div(2), z.Add(brEll(140, 150)).Div(2)})
	eqBR(t, brA.AddGeometry(brB).MulGeometry(0.5).Resolve(&r), BorderRadius{brEll(10, 20).Add(brEll(100, 110)).Div(2), brEll(30, 40).Add(z).Div(2),
		brEll(50, 60).Add(brEll(140, 150)).Div(2), z.Add(brEll(120, 130)).Div(2)})

	// add and subtract
	a := BorderRadius{TopLeft: brEll(300, 500)}
	b := BorderRadiusDirectional{TopEnd: brEll(30, 50)}
	c := BorderRadius{BottomLeft: brEll(3, 5)}
	eqBR(t, a.AddGeometry(b.Subtract(c)).Resolve(&l), BorderRadius{TopLeft: brEll(300, 500), TopRight: brEll(30, 50), BottomLeft: brEll(-3, -5)})
	eqBR(t, a.AddGeometry(b.Subtract(c)).Resolve(&r), BorderRadius{TopLeft: brEll(330, 550), BottomLeft: brEll(-3, -5)})

	// more
	a = BorderRadius{brEll(300, 300), brEll(500, 500), brEll(700, 700), brEll(900, 900)}
	b = BorderRadiusDirectional{brEll(30, 30), brEll(50, 50), brEll(70, 70), brEll(90, 90)}
	c = BorderRadius{brEll(3, 3), brEll(5, 5), brEll(7, 7), brEll(9, 9)}
	eqBR(t, a.AddGeometry(b.Subtract(c)).Resolve(&l), BorderRadius{brEll(327, 327), brEll(545, 545), brEll(763, 763), brEll(981, 981)})
	eqBR(t, a.AddGeometry(b.Subtract(c)).Resolve(&r), BorderRadius{brEll(347, 347), brEll(525, 525), brEll(783, 783), brEll(961, 961)})
}

func TestBorderRadiusGeometryOperators(t *testing.T) {
	l, r := ui.TextDirectionLtr, ui.TextDirectionRtl
	eqBR(t, brA.AddGeometry(brB.Subtract(brA).MulGeometry(0.5)).Resolve(&l), brLTR)
	eqBR(t, brA.AddGeometry(brB.Subtract(brA).MulGeometry(0.5)).Resolve(&r), brRTL)
	eqBR(t, brA.AddGeometry(brB.Subtract(brA).MulGeometry(0.0)).Resolve(&l), brA)
	eqBR(t, brA.AddGeometry(brB.Subtract(brA).MulGeometry(1.0)).Resolve(&r), brB.Resolve(&r))
}

func TestBorderRadiusCopyWith(t *testing.T) {
	first := NewBorderRadiusAll(brCirc(5))
	eqBR(t, first.CopyWith(), first)
	radius := brCirc(10)
	br := NewBorderRadiusAll(radius)
	z := ui.RadiusZero
	eqBR(t, br.CopyWith(BorderRadiusCopyWithOptions{TopLeft: &z}).TopLeft, z)
	eqBR(t, br.CopyWith(BorderRadiusCopyWithOptions{TopLeft: &z}).CopyWith(BorderRadiusCopyWithOptions{TopLeft: &radius}), br)
	eqBR(t, br.CopyWith(BorderRadiusCopyWithOptions{TopRight: &z}).TopRight, z)
	eqBR(t, br.CopyWith(BorderRadiusCopyWithOptions{TopRight: &z}).CopyWith(BorderRadiusCopyWithOptions{TopRight: &radius}), br)
	eqBR(t, br.CopyWith(BorderRadiusCopyWithOptions{BottomLeft: &z}).BottomLeft, z)
	eqBR(t, br.CopyWith(BorderRadiusCopyWithOptions{BottomLeft: &z}).CopyWith(BorderRadiusCopyWithOptions{BottomLeft: &radius}), br)
	eqBR(t, br.CopyWith(BorderRadiusCopyWithOptions{BottomRight: &z}).BottomRight, z)
	eqBR(t, br.CopyWith(BorderRadiusCopyWithOptions{BottomRight: &z}).CopyWith(BorderRadiusCopyWithOptions{BottomRight: &radius}), br)
}

func TestBorderRadiusGeometryFactories(t *testing.T) {
	r5, r10, r15, r20 := brCirc(5), brCirc(10), brCirc(15), brCirc(20)
	eqBR(t, NewBorderRadiusGeometryAll(r10), BorderRadiusGeometry(NewBorderRadiusAll(r10)))
	eqBR(t, NewBorderRadiusGeometryCircular(10), BorderRadiusGeometry(NewBorderRadiusCircular(10)))
	eqBR(t, NewBorderRadiusGeometryHorizontal(BorderRadiusGeometryHorizontalOptions{Left: &r5, Right: &r10}),
		BorderRadiusGeometry(NewBorderRadiusHorizontal(BorderRadiusHorizontalOptions{Left: r5, Right: r10})))
	eqBR(t, NewBorderRadiusGeometryHorizontal(BorderRadiusGeometryHorizontalOptions{Start: &r5, End: &r10}),
		BorderRadiusGeometry(NewBorderRadiusDirectionalHorizontal(BorderRadiusDirectionalHorizontalOptions{Start: r5, End: r10})))
	for _, o := range []BorderRadiusGeometryHorizontalOptions{
		{Start: &r5, Left: &r10}, {End: &r5, Right: &r10}, {End: &r5, Right: &r10, Start: &r5, Left: &r10},
	} {
		func() {
			defer func() {
				if _, ok := recover().(*dart.AssertionError); !ok {
					t.Error("want an AssertionError")
				}
			}()
			NewBorderRadiusGeometryHorizontal(o)
		}()
	}
	eqBR(t, NewBorderRadiusGeometryOnly(BorderRadius{r5, r10, r15, r20}), BorderRadiusGeometry(BorderRadius{TopLeft: r5, TopRight: r10, BottomLeft: r15, BottomRight: r20}))
	eqBR(t, NewBorderRadiusGeometryDirectional(BorderRadiusDirectional{r5, r10, r15, r20}),
		BorderRadiusGeometry(BorderRadiusDirectional{TopStart: r5, TopEnd: r10, BottomStart: r15, BottomEnd: r20}))
	eqBR(t, NewBorderRadiusGeometryVertical(BorderRadiusVerticalOptions{Top: r5, Bottom: r10}),
		BorderRadiusGeometry(NewBorderRadiusVertical(BorderRadiusVerticalOptions{Top: r5, Bottom: r10})))
	eqBR(t, BorderRadiusGeometryZero, BorderRadiusGeometry(BorderRadiusZero))
}

// Not in Flutter's test file: toString's forms (also in the difftest).
func TestBorderRadiusToString(t *testing.T) {
	for _, c := range []struct {
		g    BorderRadiusGeometry
		want string
	}{
		{BorderRadiusZero, "BorderRadius.zero"},
		{BorderRadiusDirectionalZero, "BorderRadius.zero"},
		{NewBorderRadiusCircular(3), "BorderRadius.circular(3.0)"},
		{NewBorderRadiusAll(brEll(5, 7)), "BorderRadius.all(Radius.elliptical(5.0, 7.0))"},
		{BorderRadius{TopRight: brCirc(1), BottomRight: brEll(2, 3)}, "BorderRadius.only(topRight: Radius.circular(1.0), bottomRight: Radius.elliptical(2.0, 3.0))"},
		{NewBorderRadiusDirectionalCircular(3), "BorderRadiusDirectional.circular(3.0)"},
		{BorderRadiusDirectional{TopEnd: brCirc(1), BottomStart: brCirc(2)}, "BorderRadiusDirectional.only(topEnd: Radius.circular(1.0), bottomStart: Radius.circular(2.0))"},
		{brA.AddGeometry(brB), "BorderRadius.only(topLeft: Radius.elliptical(10.0, 20.0), topRight: Radius.elliptical(30.0, 40.0), bottomLeft: Radius.elliptical(50.0, 60.0)) + BorderRadiusDirectional.only(topEnd: Radius.elliptical(100.0, 110.0), bottomStart: Radius.elliptical(120.0, 130.0), bottomEnd: Radius.elliptical(140.0, 150.0))"},
	} {
		if got := c.g.String(); got != c.want {
			t.Errorf("got %q, want %q", got, c.want)
		}
	}
}
