// hello opens a window and draws a rounded rectangle and text with Impeller,
// driven entirely from Go.
//
// Run with -frames=N to exit automatically after N frames.
package main

import (
	"flag"
	"log"
	"runtime"

	"github.com/go-gl/glfw/v3.3/glfw"

	"github.com/jonathanperis/flutter-go/flutter_go/impeller"
	"github.com/jonathanperis/flutter-go/flutter_go/platform/metal"
)

// GLFW and Cocoa must run on the process's main thread.
func init() { runtime.LockOSThread() }

func main() {
	maxFrames := flag.Int("frames", 0, "exit after this many frames (0 = run until closed)")
	flag.Parse()

	if err := glfw.Init(); err != nil {
		log.Fatal(err)
	}
	defer glfw.Terminate()

	glfw.WindowHint(glfw.ClientAPI, glfw.NoAPI)
	window, err := glfw.CreateWindow(800, 600, "flutter-go: hello", nil, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer window.Destroy()

	ctx, err := impeller.NewMetalContext()
	if err != nil {
		log.Fatal(err)
	}
	defer ctx.Release()

	layer := metal.Attach(window.GetCocoaWindow())
	defer layer.Release()

	typography := impeller.NewTypographyContext()
	defer typography.Release()

	dl, err := buildScene(typography, layer.Scale())
	if err != nil {
		log.Fatal(err)
	}
	defer dl.Release()

	log.Printf("Impeller C API version %#x, scale %.1f, window %d",
		impeller.Version(), layer.Scale(), metal.WindowNumber(window.GetCocoaWindow()))

	for frame := 1; !window.ShouldClose(); frame++ {
		if err := layer.Present(ctx.Handle(), dl.Handle()); err != nil {
			log.Fatal(err)
		}
		if *maxFrames > 0 && frame >= *maxFrames {
			log.Printf("presented %d frames, exiting", frame)
			return
		}
		glfw.WaitEventsTimeout(1.0 / 60)
	}
}

// buildScene records the display list once; logical pixels are multiplied by
// the backing scale so the result is crisp on Retina screens.
func buildScene(tc *impeller.TypographyContext, s float32) (*impeller.DisplayList, error) {
	b := impeller.NewDisplayListBuilder()
	defer b.Release()
	paint := impeller.NewPaint()
	defer paint.Release()

	paint.SetColor(impeller.Color{R: 1, G: 1, B: 1, A: 1})
	b.DrawPaint(paint)

	paint.SetColor(impeller.Color{R: 0.85, G: 0.15, B: 0.2, A: 1})
	b.DrawRoundedRect(impeller.Rect{X: 100 * s, Y: 100 * s, W: 600 * s, H: 300 * s}, 32*s, paint)

	text, err := impeller.NewParagraph(tc, "hello from Go", 56*s, impeller.Color{R: 1, G: 1, B: 1, A: 1}, 600*s)
	if err != nil {
		return nil, err
	}
	defer text.Release()
	b.DrawParagraph(text, 160*s, 210*s)

	return b.Build(), nil
}
