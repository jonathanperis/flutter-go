// Package app runs a widget tree in a desktop window (macOS/Metal for now).
package app

import (
	"log"
	"runtime"
	"time"

	"github.com/go-gl/glfw/v3.3/glfw"

	"github.com/jonathanperis/flutter-go/impeller"
	"github.com/jonathanperis/flutter-go/platform/metal"
	r "github.com/jonathanperis/flutter-go/rendering"
	"github.com/jonathanperis/flutter-go/widgets"
)

func init() { runtime.LockOSThread() } // Cocoa needs the main thread.

// WheelPixelsPerLine converts GLFW scroll units to logical pixels.
const WheelPixelsPerLine = 20.0

type Options struct {
	Title         string
	Width, Height int
	// MaxFrames stops after this many drawn frames (0 = run until closed).
	MaxFrames int
	// OnFrame is called after each frame with its build+layout+paint time.
	OnFrame func(n int, cpu time.Duration)
	// InjectScroll synthesizes a wheel scroll of this many pixels at the
	// window center after the first frame, for unattended runs.
	InjectScroll float64
	// InjectTaps synthesizes this many taps at InjectTapAt (logical px), one
	// per frame, for unattended runs.
	InjectTaps  int
	InjectTapAt [2]float64
	// Window receives the GLFW window once created (for tests and tools).
	Window func(w *glfw.Window)
}

// Run shows root in a window until it is closed.
func Run(root widgets.Widget, o Options) error {
	if err := glfw.Init(); err != nil {
		return err
	}
	defer glfw.Terminate()
	glfw.WindowHint(glfw.ClientAPI, glfw.NoAPI)
	win, err := glfw.CreateWindow(o.Width, o.Height, o.Title, nil, nil)
	if err != nil {
		return err
	}
	defer win.Destroy()
	if o.Window != nil {
		o.Window(win)
	}

	ctx, err := impeller.NewMetalContext()
	if err != nil {
		return err
	}
	defer ctx.Release()
	layer := metal.Attach(win.GetCocoaWindow())
	defer layer.Release()
	tc := impeller.NewTypographyContext()
	defer tc.Release()
	r.SetTextEngine(impellerText{tc})

	needsFrame := true
	owner := &widgets.BuildOwner{OnNeedsFrame: func() { needsFrame = true }}
	router := widgets.NewPointerRouter(owner)
	owner.RunApp(root)

	// GLFW reports cursor positions in logical points, which is what layout uses.
	win.SetMouseButtonCallback(func(w *glfw.Window, b glfw.MouseButton, a glfw.Action, _ glfw.ModifierKey) {
		if b != glfw.MouseButtonLeft {
			return
		}
		x, y := w.GetCursorPos()
		pos := r.Offset{X: x, Y: y}
		if a == glfw.Press {
			router.PointerDown(pos)
		} else if a == glfw.Release {
			router.PointerUp(pos)
		}
	})
	win.SetCursorPosCallback(func(w *glfw.Window, x, y float64) {
		if w.GetMouseButton(glfw.MouseButtonLeft) == glfw.Press {
			router.PointerMove(r.Offset{X: x, Y: y})
		}
	})
	// Wheel/trackpad: GLFW reports lines (or fine trackpad deltas); positive y
	// means "scroll up", i.e. move content down.
	win.SetScrollCallback(func(w *glfw.Window, _, yoff float64) {
		x, y := w.GetCursorPos()
		router.Scroll(r.Offset{X: x, Y: y}, -yoff*WheelPixelsPerLine)
	})
	win.SetSizeCallback(func(*glfw.Window, int, int) { needsFrame = true })

	paint := impeller.NewPaint()
	defer paint.Release()
	for frames := 0; !win.ShouldClose(); {
		if needsFrame {
			needsFrame = false
			start := time.Now()
			w, h := win.GetSize()
			scale := layer.Scale()
			b := impeller.NewDisplayListBuilder()
			b.Scale(scale, scale)
			owner.DrawFrame(r.Size{W: float64(w), H: float64(h)}, &impellerCanvas{b, paint})
			dl := b.Build()
			b.Release()
			cpu := time.Since(start)
			err := layer.Present(ctx.Handle(), dl.Handle())
			dl.Release()
			if err != nil {
				log.Print(err)
			}
			frames++
			if o.OnFrame != nil {
				o.OnFrame(frames, cpu)
			}
			if o.InjectScroll != 0 {
				router.Scroll(r.Offset{X: float64(w) / 2, Y: float64(h) / 2}, o.InjectScroll)
				o.InjectScroll = 0
			}
			if o.InjectTaps > 0 {
				o.InjectTaps--
				pos := r.Offset{X: o.InjectTapAt[0], Y: o.InjectTapAt[1]}
				router.PointerDown(pos)
				router.PointerUp(pos)
			}
			if o.MaxFrames > 0 && frames >= o.MaxFrames {
				break
			}
		}
		glfw.WaitEventsTimeout(0.25)
	}
	if rr := owner.RenderRoot(); rr != nil {
		r.Dispose(rr)
	}
	return nil
}
