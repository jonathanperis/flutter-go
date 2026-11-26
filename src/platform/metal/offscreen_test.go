package metal

import (
	"testing"

	"github.com/jonathanperis/flutter-go/src/impeller"
)

// The readback must see everything Impeller drew, even after heavy work
// (synchronization between Impeller's queue and the blit's).
func TestRenderOffscreenSeesFinishedFrame(t *testing.T) {
	ctx, err := impeller.NewMetalContext()
	if err != nil {
		t.Skip(err)
	}
	defer ctx.Release()
	for round := 0; round < 5; round++ {
		b := impeller.NewDisplayListBuilder()
		p := impeller.NewPaint()
		blur := impeller.NewBlurMaskFilter(0, 20)
		p.SetMaskFilter(blur)
		for i := 0; i < 3000; i++ {
			p.SetColor(impeller.Color{R: 0.2, G: float32(i%7) / 7, B: 0.5, A: 0.3})
			b.DrawOval(impeller.Rect{X: float32(i % 500), Y: float32(i % 300), W: 200, H: 200}, p)
		}
		q := impeller.NewPaint()
		q.SetColor(impeller.Color{R: float32(round % 2), G: float32(1 - round%2), B: 0, A: 1})
		b.DrawRect(impeller.Rect{X: 900, Y: 900, W: 100, H: 100}, q)
		dl := b.Build()
		px, err := RenderOffscreen(ctx.Handle(), dl.Handle(), 1000, 1000)
		if err != nil {
			t.Fatal(err)
		}
		i := (950*1000 + 950) * 4
		if got := px[i : i+4]; int(got[0]) != 255*(round%2) || int(got[1]) != 255*(1-round%2) || got[2] != 0 || got[3] != 255 {
			t.Fatalf("round %d: pixel %v, want opaque red or green by round", round, got)
		}
		dl.Release()
		b.Release()
		p.Release()
		q.Release()
		blur.Release()
	}
}
