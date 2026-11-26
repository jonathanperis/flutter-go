package difftest

// Pixel differential test of BoxDecoration and ShapeDecoration painting
// against Flutter's Impeller renderer (flutter_tester --enable-impeller):
// the same scenes as flutter/test/impeller/painting_box_decorations_pixels_test.dart,
// each decoration painted with createBoxPainter().paint at (10, 12), size
// 44×38, into a 64×64 picture read back as rawRgba. The comparison scheme
// is painting_borders_pixels_test.go's: a per-scene tolerance, and every
// pixel differing by more than 2 must be on an edge of the reference
// (bpEdgePixel) — the GPUs only disagree on anti-aliased coverage.

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"sort"
	"testing"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	p "github.com/jonathanperis/flutter-go/packages/flutter/lib/src/painting"
	flutter_test "github.com/jonathanperis/flutter-go/packages/flutter_test/lib/src"
	"github.com/jonathanperis/flutter-go/src/async"
)

// bxImg is the Dart side's Img.
type bxImg struct {
	p.ImageProviderBase
	image *ui.Image
}

func newBxImg(image *ui.Image) *bxImg { x := &bxImg{image: image}; x.SetSelf(x); return x }

func (x *bxImg) ObtainKey(p.ImageConfiguration) async.Future[any] {
	return foundation.NewSynchronousFuture[any](x)
}

func (x *bxImg) LoadImage(any, p.ImageDecoderCallback) p.ImageStreamCompleter {
	return p.NewOneFrameImageStreamCompleter(foundation.NewSynchronousFuture(p.NewImageInfo(x.image.Clone())))
}

type bxScene struct {
	dec p.Decoration
	td  ui.TextDirection
}

func boxDecorationPixelScenes(img *ui.Image) map[string]bxScene {
	ltr, rtl := ui.TextDirectionLtr, ui.TextDirectionRtl
	nb := p.NewBoxDecoration
	ns := p.NewShapeDecoration
	type bo = p.BoxDecorationOptions
	type so = p.ShapeDecorationOptions
	circ := func(r float64) ui.Radius { return ui.NewRadiusCircular(r) }
	side := func(color int, width float64, align ...float64) *p.BorderSide {
		s := deBS(color, width, align...)
		return &s
	}
	all := func(color int, width float64) p.BoxBorder {
		return p.NewBorderAll(p.BorderSideOptions{Color: deCol(color), Width: ptr(width)})
	}
	lin := p.NewLinearGradient(p.LinearGradientOptions{Begin: p.AlignmentDirectionalTopStart, End: p.AlignmentBottomRight,
		Colors: deColors(0xFFE53935, 0xFF43A047, 0xFF1E88E5), Stops: []float64{0, 0.4, 1}})
	rad := p.NewRadialGradient(p.RadialGradientOptions{Center: p.NewAlignment(0.2, -0.3), Radius: ptr(0.7),
		Colors: deColors(0xFFFFEB3B, 0xFF8E24AA)})
	swp := p.NewSweepGradient(p.SweepGradientOptions{Colors: deColors(0xFF00BCD4, 0xFFE91E63, 0xFF00BCD4),
		Transform: p.NewGradientRotation(0.5)})
	shadows := []p.BoxShadow{
		deShadow(0x80000000, 2, 3, 4, 1, ui.BlurStyleNormal),
		deShadow(0x60E53935, -2, -1, 3, 0, ui.BlurStyleNormal),
	}
	image := func(o p.DecorationImageOptions) p.DecorationImage {
		o.Image = newBxImg(img)
		return p.NewDecorationImage(o)
	}
	cover, fill, repeat := p.BoxFitCover, p.BoxFitFill, p.ImageRepeatRepeat
	return map[string]bxScene{
		"box.color.border.radius": {nb(bo{Color: deCol(0xFF43A047), Border: all(0xFF1E88E5, 4),
			BorderRadius: p.NewBorderRadiusAll(circ(10))}), ltr},
		"box.circle":                 {nb(bo{Color: deCol(0xFFE53935), Shape: p.BoxShapeCircle, Border: p.NewBorderAll(p.BorderSideOptions{Width: ptr(3.0)})}), ltr},
		"box.gradient.linear":        {nb(bo{Gradient: lin}), ltr},
		"box.gradient.linear.rtl":    {nb(bo{Gradient: lin, BorderRadius: p.BorderRadiusDirectional{TopStart: circ(16)}}), rtl},
		"box.gradient.radial.circle": {nb(bo{Gradient: rad, Shape: p.BoxShapeCircle}), ltr},
		"box.gradient.sweep":         {nb(bo{Gradient: swp, BorderRadius: p.NewBorderRadiusAll(circ(8))}), ltr},
		"box.shadows":                {nb(bo{Color: deCol(0xFFFFFFFF), BoxShadow: shadows}), ltr},
		"box.shadow.spread": {nb(bo{Color: deCol(0xFF1E88E5), BorderRadius: p.NewBorderRadiusAll(circ(6)),
			BoxShadow: []p.BoxShadow{deShadow(0xFF000000, 0, 0, 5, 4, ui.BlurStyleNormal)}}), ltr},
		"box.shadow.solid": {nb(bo{Color: deCol(0xFF8E24AA),
			BoxShadow: []p.BoxShadow{deShadow(0xA0000000, 3, 3, 4, 0, ui.BlurStyleSolid)}}), ltr},
		"box.shadow.outer.circle": {nb(bo{Color: deCol(0x4000BCD4), Shape: p.BoxShapeCircle,
			BoxShadow: []p.BoxShadow{deShadow(0xFF000000, 0, 0, 6, 0, ui.BlurStyleOuter)}}), ltr},
		"box.shadow.inner": {nb(bo{Color: deCol(0xFFFFEB3B),
			BoxShadow: []p.BoxShadow{deShadow(0xFF000000, 0, 0, 4, -3, ui.BlurStyleInner)}}), ltr},
		"box.blendMode": {nb(bo{Color: deCol(0xFFFFCC00), BackgroundBlendMode: ptr(ui.BlendModeDifference)}), ltr},
		"box.border.nonuniform": {nb(bo{Color: deCol(0xFFFFEB3B), Border: p.NewBorder(p.BorderOptions{
			Top: side(0xFFE53935, 6), Left: side(0xFF1E88E5, 3), Bottom: side(0x80000000, 2)})}), ltr},
		"box.border.directional.rtl": {nb(bo{Color: deCol(0xFF00BCD4), Border: p.NewBorderDirectional(p.BorderDirectionalOptions{
			Start: side(0xFF000000, 5), End: side(0xFFE53935, 2)})}), rtl},
		"box.image":        {nb(bo{Image: image(p.DecorationImageOptions{Fit: &cover})}), ltr},
		"box.image.circle": {nb(bo{Color: deCol(0xFF000000), Shape: p.BoxShapeCircle, Image: image(p.DecorationImageOptions{Fit: &fill})}), ltr},
		"box.image.radius.border": {nb(bo{Image: image(p.DecorationImageOptions{Repeat: &repeat, Alignment: p.AlignmentTopLeft}),
			BorderRadius: p.NewBorderRadiusAll(circ(12)), Border: all(0xFF43A047, 3)}), ltr},
		"shape.star.gradient.shadows": {ns(so{Gradient: lin, Shadows: shadows, Shape: p.NewStarBorder(p.StarBorderOptions{
			Side: side(0xFF000000, 2), InnerRadiusRatio: ptr(0.5)})}), ltr},
		"shape.stadium.color.shadows": {ns(so{Color: deCol(0xFF1E88E5), Shadows: shadows,
			Shape: p.NewStadiumBorder(p.StadiumBorderOptions{Side: side(0xFFFFFFFF, 3)})}), ltr},
		"shape.rrect.gradient": {ns(so{Gradient: rad, Shape: p.NewRoundedRectangleBorder(p.RRectBorderOptions{
			Side: side(0xFF8E24AA, 4), BorderRadius: p.NewBorderRadiusAll(circ(14))})}), ltr},
		"shape.superellipse.shadows": {ns(so{Color: deCol(0xFF43A047), Shadows: shadows,
			Shape: p.NewRoundedSuperellipseBorder(p.RRectBorderOptions{Side: side(0xFF000000, 2), BorderRadius: p.NewBorderRadiusAll(circ(18))})}), ltr},
		"shape.circle.image": {ns(so{Image: image(p.DecorationImageOptions{Fit: &cover}),
			Shape: p.NewCircleBorder(p.CircleBorderOptions{Side: side(0xFFE53935, 3)})}), ltr},
		"shape.border.shadows": {ns(so{Color: deCol(0xFFFFEB3B), Shadows: shadows, Shape: all(0xFF1E88E5, 4)}), ltr},
		"shape.beveled.sweep": {ns(so{Gradient: swp, Shape: p.NewBeveledRectangleBorder(p.RRectBorderOptions{
			BorderRadius: p.NewBorderRadiusAll(circ(12))})}), ltr},
		"shape.oval.translucent": {ns(so{Color: deCol(0x8000BCD4), Shape: p.NewOvalBorder(p.OvalBorderOptions{
			Side: side(0xC0000000, 4, p.BorderSideStrokeAlignCenter)})}), ltr},
		"lerp.box.shape": {p.DecorationLerp(
			nb(bo{Color: deCol(0xFFE53935), BorderRadius: p.NewBorderRadiusAll(circ(4))}),
			ns(so{Color: deCol(0xFF1E88E5), Shape: p.NewCircleBorder(p.CircleBorderOptions{Side: side(0xFF000000, 2)})}),
			0.4), ltr},
	}
}

// boxDecorationPixelTolerance: see bordersPixelTolerance. Scenes not
// listed allow max 2 and no pixel over 2 (fills of pixel-aligned rects,
// non-uniform and directional borders, linear/sweep gradients, every box
// shadow, image cover): exact or within 1. Measured on Apple Metal vs
// SwiftShader:
//   - curved or diagonal anti-aliased edges (rounded rects, circles, the
//     image clips, stadium/star/superellipse/oval shapes, the lerped
//     circle) differ by one 4x MSAA sample (≤64 per channel), two on a few
//     pixels of the translucent rrect gradient (127); under translucent
//     colors or blurred shadows a sample is worth less (29-50);
//   - drawCircle (box.circle, shape.circle.image) is Impeller's circle
//     geometry: ≤9 on edge pixels (as in painting_pixels' "circle");
//   - box.blendMode: the reference has a seam along the rect's diagonal
//     (42 pixels darker by 63) where flutter_tester's Vulkan/SwiftShader
//     applies the advanced blend (difference) to both triangles of the
//     rect; Metal's is uniform. The seam is not in Flutter's code.
//
// Every pixel differing by more than 2 must be an edge pixel of the
// reference (bpEdgePixel; the seam qualifies).
var boxDecorationPixelTolerance = map[string]struct{ max, over2 int }{
	"box.blendMode":               {63, 42},
	"box.circle":                  {4, 74},
	"box.color.border.radius":     {64, 4},
	"box.gradient.linear.rtl":     {64, 2},
	"box.gradient.radial.circle":  {64, 8},
	"box.image.circle":            {64, 14},
	"box.image.radius.border":     {64, 12},
	"box.shadow.outer.circle":     {29, 8},
	"lerp.box.shape":              {64, 98},
	"shape.circle.image":          {9, 54},
	"shape.oval.translucent":      {48, 18},
	"shape.rrect.gradient":        {127, 11},
	"shape.stadium.color.shadows": {62, 16},
	"shape.star.gradient.shadows": {50, 7},
	"shape.superellipse.shadows":  {40, 140},
}

func TestPaintingBoxDecorationsPixelsMatchFlutter(t *testing.T) {
	data, err := os.ReadFile("flutter/build/painting_box_decorations_pixels.json")
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
	flutter_test.New(t) // image providers resolve through PaintingBinding.instance
	// As the Dart side: the test binding disables shadows; paint them.
	defer func(old bool) { p.DebugDisableShadows = old }(p.DebugDisableShadows)
	p.DebugDisableShadows = false
	ui.NewPlatformConfiguration(nil).UpdateViewGestureMetrics(0, 3.0, ui.GestureSettings{})
	defer ui.NewPlatformConfiguration(nil).UpdateViewGestureMetrics(0, 1.0, ui.GestureSettings{})
	defer func() {
		p.ImageCacheInstance().Clear()
		p.ImageCacheInstance().ClearLiveImages()
	}()
	img := checkerImage(t)
	scenes := boxDecorationPixelScenes(img)
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
		sc := scenes[name]
		painter := sc.dec.CreateBoxPainter(func() {})
		size := ui.NewSize(44, 38)
		td := sc.td
		painter.Paint(ui.NewCanvas(rec), ui.NewOffset(10, 12), p.ImageConfiguration{Size: &size, TextDirection: &td})
		painter.Dispose()
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
		tol, ok := boxDecorationPixelTolerance[name]
		if !ok {
			tol.max, tol.over2 = 2, 0
		}
		t.Logf("%-30s max %3d, pixels >2: %4d, >16: %4d", name, maxDiff, over2, over16)
		if maxDiff > tol.max || over2 > tol.over2 {
			t.Errorf("%s: max channel difference %d (%d pixels > 2), tolerance %+v", name, maxDiff, over2, tol)
		}
	}
}
