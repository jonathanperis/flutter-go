package difftest

// Pixel differential test of painting's borders over the Impeller SDK
// against Flutter's own Impeller renderer (flutter_tester
// --enable-impeller): the same 64×64 scenes as
// flutter/test/impeller/painting_borders_pixels_test.dart, compared with
// painting_pixels_test.go's per-scene tolerance scheme (the GPUs' MSAA
// sample positions make anti-aliased edges differ; see pixelTolerance
// there).

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"sort"
	"testing"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	p "github.com/jonathanperis/flutter-go/packages/flutter/lib/src/painting"
)

var bpBox = ui.NewRectFromLTRB(10, 12, 54, 50)

func bpSide(color int, width float64, strokeAlign ...float64) p.BorderSide {
	a := p.BorderSideStrokeAlignInside
	if len(strokeAlign) > 0 {
		a = strokeAlign[0]
	}
	c := ui.NewColor(color)
	return p.NewBorderSide(p.BorderSideOptions{Color: &c, Width: &width, StrokeAlign: &a})
}

func bordersPixelScenes() map[string]func(c ui.Canvas) {
	red, green, blue, black := bpSide(0xFFE53935, 4), bpSide(0xFF43A047, 6), bpSide(0xFF1E88E5, 2), bpSide(0xFF000000, 1)
	ltr, rtl := ui.TextDirectionLtr, ui.TextDirectionRtl
	shape := func(c ui.Canvas, s p.ShapeBorder, td ui.TextDirection, fill bool) {
		if fill {
			q := ui.NewPaint()
			q.SetColor(ui.NewColor(0x8000BCD4))
			s.PaintInterior(c, bpBox, q, &td)
		}
		s.Paint(c, bpBox, &td)
	}
	all := func(color int, width float64, align ...float64) p.Border {
		return p.NewBorderFromBorderSide(bpSide(color, width, align...))
	}
	purple := func(w float64, a ...float64) *p.BorderSide { s := bpSide(0xFF8E24AA, w, a...); return &s }
	r := func(v float64) ui.Radius { return ui.NewRadiusCircular(v) }
	rad := func(v float64) p.BorderRadius { return p.NewBorderRadiusAll(r(v)) }
	return map[string]func(c ui.Canvas){
		"border.uniform":         func(c ui.Canvas) { shape(c, all(0xFFE53935, 4), ltr, false) },
		"border.uniform.center":  func(c ui.Canvas) { shape(c, all(0xFFE53935, 6, p.BorderSideStrokeAlignCenter), ltr, false) },
		"border.uniform.outside": func(c ui.Canvas) { shape(c, all(0xFFE53935, 5, p.BorderSideStrokeAlignOutside), ltr, false) },
		"border.uniform.radius": func(c ui.Canvas) {
			radius := rad(12)
			all(0xFF1E88E5, 5).PaintBox(c, bpBox, p.BoxBorderPaintOptions{BorderRadius: &radius})
		},
		"border.uniform.circle": func(c ui.Canvas) {
			all(0xFF1E88E5, 5).PaintBox(c, bpBox, p.BoxBorderPaintOptions{Shape: p.BoxShapeCircle})
		},
		"border.hairline": func(c ui.Canvas) { shape(c, all(0xFF000000, 0), ltr, false) },
		"border.nonuniform": func(c ui.Canvas) {
			shape(c, p.NewBorder(p.BorderOptions{Top: &red, Right: &green, Bottom: &blue, Left: &black}), ltr, false)
		},
		"border.nonuniform.radius": func(c ui.Canvas) {
			radius := p.BorderRadius{TopLeft: r(16), BottomRight: ui.NewRadiusElliptical(20, 8)}
			p.NewBorder(p.BorderOptions{Top: purple(2), Right: purple(8), Bottom: purple(4), Left: purple(1)}).
				PaintBox(c, bpBox, p.BoxBorderPaintOptions{BorderRadius: &radius})
		},
		"border.nonuniform.circle": func(c ui.Canvas) {
			p.NewBorder(p.BorderOptions{Top: purple(2), Right: purple(8, p.BorderSideStrokeAlignCenter), Bottom: purple(4), Left: purple(1)}).
				PaintBox(c, bpBox, p.BoxBorderPaintOptions{Shape: p.BoxShapeCircle})
		},
		"border.someNone": func(c ui.Canvas) {
			none := p.BorderStyleNone
			g := green.CopyWith(p.BorderSideCopyWithOptions{Style: &none})
			shape(c, p.NewBorder(p.BorderOptions{Top: &red, Left: &g, Bottom: &blue}), ltr, false)
		},
		"directional.ltr": func(c ui.Canvas) {
			shape(c, p.NewBorderDirectional(p.BorderDirectionalOptions{Top: &red, Start: &green, End: &blue, Bottom: &black}), ltr, false)
		},
		"directional.rtl": func(c ui.Canvas) {
			shape(c, p.NewBorderDirectional(p.BorderDirectionalOptions{Top: &red, Start: &green, End: &blue, Bottom: &black}), rtl, false)
		},
		"circle": func(c ui.Canvas) { shape(c, p.NewCircleBorder(p.CircleBorderOptions{Side: &red}), ltr, true) },
		"circle.eccentric": func(c ui.Canvas) {
			shape(c, p.NewCircleBorder(p.CircleBorderOptions{Side: &green, Eccentricity: 0.5}), ltr, true)
		},
		"circle.outside": func(c ui.Canvas) {
			s := bpSide(0xFF1E88E5, 5, p.BorderSideStrokeAlignOutside)
			shape(c, p.NewCircleBorder(p.CircleBorderOptions{Side: &s}), ltr, false)
		},
		"oval": func(c ui.Canvas) { shape(c, p.NewOvalBorder(p.OvalBorderOptions{Side: &blue}), ltr, true) },
		"rrect": func(c ui.Canvas) {
			shape(c, p.NewRoundedRectangleBorder(p.RRectBorderOptions{Side: &red, BorderRadius: rad(10)}), ltr, true)
		},
		"rrect.hairline": func(c ui.Canvas) {
			s := bpSide(0xFF000000, 0)
			shape(c, p.NewRoundedRectangleBorder(p.RRectBorderOptions{Side: &s, BorderRadius: rad(10)}), ltr, false)
		},
		"rrect.directional.rtl": func(c ui.Canvas) {
			s := bpSide(0xFF43A047, 3, p.BorderSideStrokeAlignCenter)
			shape(c, p.NewRoundedRectangleBorder(p.RRectBorderOptions{Side: &s,
				BorderRadius: p.BorderRadiusDirectional{TopStart: r(18), BottomEnd: r(6)}}), rtl, false)
		},
		"superellipse": func(c ui.Canvas) {
			shape(c, p.NewRoundedSuperellipseBorder(p.RRectBorderOptions{Side: &red, BorderRadius: rad(16)}), ltr, true)
		},
		"superellipse.zero": func(c ui.Canvas) {
			shape(c, p.NewRoundedSuperellipseBorder(p.RRectBorderOptions{Side: &green}), ltr, true)
		},
		"stadium": func(c ui.Canvas) { shape(c, p.NewStadiumBorder(p.StadiumBorderOptions{Side: &blue}), ltr, true) },
		"stadium.outside": func(c ui.Canvas) {
			s := bpSide(0xFFE53935, 4, p.BorderSideStrokeAlignOutside)
			shape(c, p.NewStadiumBorder(p.StadiumBorderOptions{Side: &s}), ltr, false)
		},
		"beveled": func(c ui.Canvas) {
			shape(c, p.NewBeveledRectangleBorder(p.RRectBorderOptions{Side: &red, BorderRadius: rad(12)}), ltr, false)
		},
		"continuous": func(c ui.Canvas) {
			shape(c, p.NewContinuousRectangleBorder(p.RRectBorderOptions{Side: &green, BorderRadius: rad(20)}), ltr, false)
		},
		"star": func(c ui.Canvas) { shape(c, p.NewStarBorder(p.StarBorderOptions{Side: &blue}), ltr, false) },
		"star.rounded": func(c ui.Canvas) {
			shape(c, p.NewStarBorder(p.StarBorderOptions{Side: &red, Points: ptr(6.0), InnerRadiusRatio: ptr(0.5),
				PointRounding: 0.4, ValleyRounding: 0.3, Rotation: 15, Squash: 0.5}), ltr, false)
		},
		"star.polygon": func(c ui.Canvas) {
			shape(c, p.NewStarBorderPolygon(p.StarBorderPolygonOptions{Side: &green, Sides: ptr(5.0), PointRounding: 0.2, Squash: 1}), ltr, false)
		},
		"linear.start": func(c ui.Canvas) {
			shape(c, p.NewLinearBorderStart(p.LinearBorderEdgeConstructorOptions{Side: &red, Size: ptr(0.5), Alignment: -1}), ltr, false)
		},
		"linear.start.rtl": func(c ui.Canvas) {
			shape(c, p.NewLinearBorderStart(p.LinearBorderEdgeConstructorOptions{Side: &red, Size: ptr(0.5)}), rtl, false)
		},
		"linear.topBottom": func(c ui.Canvas) {
			shape(c, p.NewLinearBorder(p.LinearBorderOptions{Side: &blue,
				Top: ptr(p.NewLinearBorderEdge(p.LinearBorderEdgeOptions{Size: ptr(0.75), Alignment: 1})), Bottom: ptr(p.NewLinearBorderEdge())}), ltr, false)
		},
		"compound": func(c ui.Canvas) { shape(c, all(0xFFE53935, 3).Add(all(0xFF1E88E5, 4)), ltr, false) },
		"lerp.rrect.circle": func(c ui.Canvas) {
			shape(c, p.ShapeBorderLerp(p.NewRoundedRectangleBorder(p.RRectBorderOptions{Side: &red, BorderRadius: rad(4)}),
				p.NewCircleBorder(p.CircleBorderOptions{Side: &blue}), 0.4), ltr, true)
		},
		"lerp.superellipse.oval": func(c ui.Canvas) {
			shape(c, p.ShapeBorderLerp(p.NewRoundedSuperellipseBorder(p.RRectBorderOptions{Side: &red, BorderRadius: rad(8)}),
				p.NewOvalBorder(p.OvalBorderOptions{Side: &blue}), 0.6), ltr, true)
		},
		"lerp.stadium.circle": func(c ui.Canvas) {
			shape(c, p.ShapeBorderLerp(p.NewStadiumBorder(p.StadiumBorderOptions{Side: &green}),
				p.NewCircleBorder(p.CircleBorderOptions{Eccentricity: 0.3}), 0.5), ltr, true)
		},
		"lerp.stadium.rrect": func(c ui.Canvas) {
			shape(c, p.ShapeBorderLerp(p.NewStadiumBorder(p.StadiumBorderOptions{Side: &green}),
				p.NewRoundedRectangleBorder(p.RRectBorderOptions{Side: &red}), 0.3), ltr, true)
		},
		"lerp.star.rrect": func(c ui.Canvas) {
			shape(c, p.ShapeBorderLerp(p.NewStarBorder(p.StarBorderOptions{Side: &blue}),
				p.NewRoundedRectangleBorder(p.RRectBorderOptions{BorderRadius: rad(8)}), 0.5), ltr, false)
		},
	}
}

// bpEdgePixel reports whether the reference pixel at (x, y) is on an edge:
// some pixel of its 3×3 neighbourhood differs from it by more than 16 in a
// channel. Every pixel differing by more than 2 must be one: the GPUs
// only disagree on anti-aliased coverage.
func bpEdgePixel(px []byte, x, y int) bool {
	i := (y*pixelSize + x) * 4
	for dy := -1; dy <= 1; dy++ {
		for dx := -1; dx <= 1; dx++ {
			xx, yy := x+dx, y+dy
			if xx < 0 || yy < 0 || xx >= pixelSize || yy >= pixelSize {
				continue
			}
			j := (yy*pixelSize + xx) * 4
			for k := 0; k < 4; k++ {
				if absInt(int(px[j+k])-int(px[i+k])) > 16 {
					return true
				}
			}
		}
	}
	return false
}

// bordersPixelTolerance: see pixelTolerance in painting_pixels_test.go.
// Scenes not listed allow max 2 and no pixel over 2 (rectangles, DRRect
// rings at pixel-aligned edges, paintBorder trapezoids, linear borders,
// compound borders: exact or ≤1). Measured on Apple Metal vs SwiftShader:
//   - curved or diagonal anti-aliased edges differ by one 4x MSAA sample
//     (64), two on a few pixels of the rounded star (128); with the
//     translucent interior fill (alpha 0x80) under the stroke a sample is
//     worth ~32-37;
//   - drawCircle (border.uniform.circle, circle, circle.outside) is
//     Impeller's circle geometry: ≤8 on edge pixels (as in
//     painting_pixels' "circle").
// Every pixel differing by more than 2 must be an edge pixel (bpEdgePixel).
var bordersPixelTolerance = map[string]struct{ max, over2 int }{
	"border.nonuniform.circle": {65, 5},
	"border.nonuniform.radius": {65, 9},
	"border.uniform.circle":    {5, 44},
	"circle":                   {8, 92},
	"circle.eccentric":         {65, 9},
	"circle.outside":           {7, 140},
	"continuous":               {65, 4},
	"lerp.rrect.circle":        {37, 27},
	"lerp.stadium.circle":      {33, 14},
	"lerp.stadium.rrect":       {65, 36},
	"lerp.star.rrect":          {32, 10},
	"lerp.superellipse.oval":   {65, 54},
	"oval":                     {33, 20},
	"rrect":                    {57, 4},
	"rrect.directional.rtl":    {65, 3},
	"rrect.hairline":           {65, 8},
	"stadium":                  {65, 28},
	"stadium.outside":          {65, 6},
	"star":                     {65, 12},
	"star.polygon":             {65, 15},
	"star.rounded":             {129, 22},
	"superellipse":             {65, 31},
}

func TestPaintingBordersPixelsMatchFlutter(t *testing.T) {
	data, err := os.ReadFile("flutter/build/painting_borders_pixels.json")
	if err != nil {
		t.Skipf("no reference (%v); run scripts/difftest.sh", err)
	}
	var ref map[string]string
	if err := json.Unmarshal(data, &ref); err != nil {
		t.Fatal(err)
	}
	if ui.EngineImpellerContext() == nil {
		t.Fatal("no Metal device: the pixel difftest needs one")
	}
	scenes := bordersPixelScenes()
	if len(ref) != len(scenes) {
		t.Errorf("reference has %d scenes, Go has %d", len(ref), len(scenes))
	}
	names := make([]string, 0, len(scenes))
	for n := range scenes {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, name := range names {
		want, err := base64.StdEncoding.DecodeString(ref[name])
		if err != nil || len(want) != pixelSize*pixelSize*4 {
			t.Errorf("%s: bad reference", name)
			continue
		}
		rec := ui.NewPictureRecorder()
		scenes[name](ui.NewCanvas(rec))
		got := awaitF(t, awaitF(t, rec.EndRecording().ToImage(pixelSize, pixelSize)).ToByteData())
		maxDiff, over2, over16, interior := 0, 0, 0, 0
		for i := 0; i < len(want); i += 4 {
			d := 0
			for k := 0; k < 4; k++ {
				d = max(d, absInt(int(got[i+k])-int(want[i+k])))
			}
			maxDiff = max(maxDiff, d)
			if d > 2 {
				over2++
				if !bpEdgePixel(want, i/4%pixelSize, i/4/pixelSize) {
					interior++
				}
			}
			if d > 16 {
				over16++
			}
		}
		if interior > 0 {
			t.Errorf("%s: %d pixels away from any edge differ by more than 2", name, interior)
		}
		tol, ok := bordersPixelTolerance[name]
		if !ok {
			tol.max, tol.over2 = 2, 0
		}
		t.Logf("%-28s max %3d, pixels >2: %4d, >16: %4d", name, maxDiff, over2, over16)
		if maxDiff > tol.max || over2 > tol.over2 {
			t.Errorf("%s: max channel difference %d (%d pixels > 2), tolerance %+v", name, maxDiff, over2, tol)
		}
	}
}
