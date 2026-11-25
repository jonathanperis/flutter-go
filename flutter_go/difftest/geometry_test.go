package difftest

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"

	. "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/flutter_go/dart"
)

func unsupported(f func() any) (s string) {
	defer func() {
		if r := recover(); r != nil {
			if _, ok := r.(dart.UnsupportedError); ok {
				s = "UnsupportedError"
				return
			}
			panic(r)
		}
	}()
	return fmt.Sprint(f())
}

// Same cases as flutter/test/geometry_test.dart.
func geometryCases() map[string]any {
	o, s, r := NewOffset(5.25, -3.5), NewSize(3, 4), NewRectFromLTRB(1, 2, 11, 22)
	rr := NewRRectFromRectAndCorners(r, RRectCorners{TopLeft: NewRadiusCircular(2), TopRight: NewRadiusElliptical(3, 4),
		BottomRight: NewRadiusCircular(9), BottomLeft: NewRadiusCircular(12)})
	uni := NewRRectFromLTRBXY(0, 0, 10, 10, 2, 3)
	one := NewOffset(1, 1)
	str := func(v any) string { return fmt.Sprint(v) }
	b := func(v bool) string { return fmt.Sprint(v) }
	size10 := NewSize(10, 0)
	rz := RectZero
	r1, r39 := NewRadiusCircular(1), NewRadiusElliptical(3, 9)
	r4 := NewRadiusCircular(4)
	rse := NewRSuperellipseFromLTRBXY(0, 0, 10, 10, 2, 3)
	var ts []float64
	for _, t := range []RSTransform{NewRSTransform(0.1, 0.2, 0.3, 1e-8),
		NewRSTransformFromComponents(RSTransformComponents{Rotation: 0.7, Scale: 1.3, AnchorX: 2, AnchorY: 3, TranslateX: 10, TranslateY: -4})} {
		ts = append(ts, t.Scos(), t.Ssin(), t.Tx(), t.Ty())
	}
	cases := map[string]any{
		"offset.toString":            str(o),
		"offset.neg":                 str(o.Neg()),
		"offset.mod":                 str(o.Mod(2)),
		"offset.mod.neg":             str(NewOffset(-7.5, -0.25).Mod(2)),
		"offset.truncdiv":            str(o.TruncDiv(2)),
		"offset.truncdiv.zero":       unsupported(func() any { return o.TruncDiv(0) }),
		"offset.distance":            o.Distance(),
		"offset.and":                 str(o.And(s)),
		"offset.lerp.a.null":         str(*OffsetLerp(nil, &o, 0.25)),
		"offset.lerp.b.null":         str(*OffsetLerp(&o, nil, 0.25)),
		"offset.lerp":                str(*OffsetLerp(&o, &one, 0.3)),
		"offset.lerp.x":              OffsetLerp(&o, &one, 0.3).Dx,
		"size.toString":              str(s),
		"size.minus.size":            str(s.Sub(NewSize(1, 1))),
		"size.minus.offset":          str(s.Sub(NewOffset(1, 1))),
		"size.mod":                   str(NewSize(-5, 7.5).Mod(4)),
		"size.flipped":               str(s.Flipped()),
		"size.center":                str(s.Center(o)),
		"size.lerp":                  str(*SizeLerp(&s, &size10, 0.7)),
		"size.lerp.null":             str(*SizeLerp(nil, &s, 0.5)),
		"rect.toString":              str(r),
		"rect.inflate":               str(r.Inflate(1.5)),
		"rect.overlaps":              b(r.Overlaps(NewRectFromLTRB(11, 0, 12, 30))),
		"rect.contains.edge":         b(r.Contains(NewOffset(11, 5))),
		"rect.fromPoints":            str(NewRectFromPoints(NewOffset(5, 1), NewOffset(-1, 9))),
		"rect.fromCircle":            str(NewRectFromCircle(o, 2.5)),
		"rect.lerp":                  str(*RectLerp(&r, &rz, 0.25)),
		"rect.lerp.null":             str(*RectLerp(nil, &r, 0.25)),
		"rect.isEmpty":               b(NewRectFromLTRB(1, 1, 1, 5).IsEmpty()),
		"radius.toString.circular":   str(NewRadiusCircular(2.25)),
		"radius.toString.elliptical": str(NewRadiusElliptical(2, 3)),
		"radius.mod":                 str(NewRadiusElliptical(-5, 5).Mod(3)),
		"radius.lerp":                str(*RadiusLerp(&r1, &r39, 0.5)),
		"radius.lerp.null":           str(*RadiusLerp(nil, &r4, 0.5)),
		"rrect.toString.corners":     str(rr),
		"rrect.toString.xy":          str(uni),
		"rrect.toString.r":           str(NewRRectFromLTRBR(0, 0, 4, 4, NewRadiusCircular(1))),
		"rrect.safeInnerRect":        str(rr.SafeInnerRect()),
		"rrect.safeInnerRect.left":   rr.SafeInnerRect().Left,
		"rrect.middleRect":           str(rr.MiddleRect()),
		"rrect.wideMiddleRect":       str(rr.WideMiddleRect()),
		"rrect.tallMiddleRect":       str(rr.TallMiddleRect()),
		"rrect.scaleRadii":           str(rr.ScaleRadii()),
		"rrect.isStadium":            b(NewRRectFromLTRBR(0, 0, 10, 4, NewRadiusCircular(2)).IsStadium()),
		"rrect.isEllipse":            b(NewRRectFromLTRBXY(0, 0, 10, 4, 5, 2).IsEllipse()),
		"rrect.isCircle":             b(NewRRectFromLTRBR(0, 0, 4, 4, NewRadiusCircular(2)).IsCircle()),
		"rrect.isRect":               b(NewRRectFromLTRBXY(0, 0, 4, 4, 0, 3).IsRect()),
		"rrect.lerp":                 str(*RRectLerp(&rr, &uni, 0.4)),
		"rrect.lerp.null":            str(*RRectLerp(nil, &rr, 0.4)),
		"rrect.lerp.null.b":          str(*RRectLerp(&rr, nil, 0.4)),
		"rrect.contains.corner":      b(rr.Contains(NewOffset(1.5, 2.5))),
		"rrect.contains.inside":      b(rr.Contains(NewOffset(6, 12))),
		"rrect.contains.grid":        containsGrid(rr),
		"rse.toString":               str(NewRSuperellipseFromRectAndCorners(r, RRectCorners{TopLeft: NewRadiusCircular(2)})),
		"rse.lerp":                   str(*RSuperellipseLerp(&rse, nil, 0.5)),
		"rstransform":                ts,
	}
	for name, rse := range map[string]RSuperellipse{
		"uniform":    NewRSuperellipseFromLTRBXY(0, 0, 40, 30, 9, 9),
		"elliptical": NewRSuperellipseFromLTRBXY(0, 0, 40, 30, 6, 13),
		"mixed": NewRSuperellipseFromLTRBAndCorners(0, 0, 40, 30, RRectCorners{TopLeft: NewRadiusCircular(2),
			TopRight: NewRadiusElliptical(15, 6), BottomRight: NewRadiusCircular(25), BottomLeft: RadiusZero}),
		"inverted": NewRSuperellipseFromLTRBXY(40, 30, 0, 0, 7, 7),
		"ratio27":  NewRSuperellipseFromLTRBXY(0, 0, 30, 30, 11, 11),
		"ratio15":  NewRSuperellipseFromLTRBXY(0, 0, 40, 30, 2, 2),
	} {
		var rows []string
		for y := -0.75; y <= 30.75; y += 0.5 {
			row := ""
			for x := -0.75; x <= 40.75; x += 0.5 {
				if rse.Contains(NewOffset(x, y)) {
					row += "1"
				} else {
					row += "0"
				}
			}
			rows = append(rows, row)
		}
		cases["rse.contains."+name] = strings.Join(rows, "|")
	}
	fine := NewRSuperellipseFromLTRBXY(0, 0, 30, 30, 11, 11)
	var rows []string
	for j := 0; j < 160; j++ {
		row := ""
		for i := 0; i < 160; i++ {
			if fine.Contains(NewOffset(22.025+float64(i)*0.05, 0.025+float64(j)*0.05)) {
				row += "1"
			} else {
				row += "0"
			}
		}
		rows = append(rows, row)
	}
	cases["rse.contains.ratio27.corner.fine"] = strings.Join(rows, "|")
	return cases
}

func containsGrid(rr RRect) string {
	var rows []string
	for y := 1.75; y <= 22.25; y += 0.5 {
		row := ""
		for x := 0.75; x <= 11.25; x += 0.5 {
			if rr.Contains(NewOffset(x, y)) {
				row += "1"
			} else {
				row += "0"
			}
		}
		rows = append(rows, row)
	}
	return strings.Join(rows, "|")
}

func TestGeometryMatchesFlutter(t *testing.T) {
	data, err := os.ReadFile("flutter/build/geometry.json")
	if err != nil {
		t.Skipf("no reference (%v); run scripts/difftest.sh", err)
	}
	var ref map[string]any
	if err := json.Unmarshal(data, &ref); err != nil {
		t.Fatal(err)
	}
	cases := geometryCases()
	if len(ref) != len(cases) {
		t.Fatalf("reference has %d cases, Go has %d", len(ref), len(cases))
	}
	for name, got := range cases {
		want := ref[name]
		switch g := got.(type) {
		case string:
			if g != want {
				t.Errorf("%s: Go %q, Flutter %q", name, g, want)
			}
		case float64:
			if w, ok := want.(float64); !ok || !closeTo(g, w) {
				t.Errorf("%s: Go %v, Flutter %v", name, g, want)
			}
		case []float64:
			w, _ := want.([]any)
			if len(w) != len(g) {
				t.Fatalf("%s: length %d vs %d", name, len(g), len(w))
			}
			for i := range g {
				if g[i] != w[i].(float64) {
					t.Errorf("%s[%d]: Go %v, Flutter %v (must match exactly: float32 storage)", name, i, g[i], w[i])
				}
			}
		}
	}
}
