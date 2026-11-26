// Package app runs a widget tree in a desktop window (macOS/Metal for now).
package app

import (
	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"log"
	"runtime"
	"time"

	"github.com/go-gl/glfw/v3.3/glfw"

	r "github.com/jonathanperis/flutter-go/packages/flutter/lib/src/rendering"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/services"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/widgets"
	"github.com/jonathanperis/flutter-go/src/impeller"
	"github.com/jonathanperis/flutter-go/src/platform/metal"
)

func init() { runtime.LockOSThread() } // Cocoa needs the main thread.

var keyMap = map[glfw.Key]services.Key{
	glfw.KeyBackspace: services.KeyBackspace, glfw.KeyDelete: services.KeyDelete,
	glfw.KeyLeft: services.KeyLeft, glfw.KeyRight: services.KeyRight,
	glfw.KeyUp: services.KeyUp, glfw.KeyDown: services.KeyDown,
	glfw.KeyHome: services.KeyHome, glfw.KeyEnd: services.KeyEnd,
	glfw.KeyEnter: services.KeyEnter, glfw.KeyKPEnter: services.KeyEnter,
	glfw.KeyTab: services.KeyTab, glfw.KeyA: services.KeyA, glfw.KeyC: services.KeyC,
	glfw.KeyV: services.KeyV, glfw.KeyX: services.KeyX,
}

type glfwClipboard struct{ w *glfw.Window }

func (c glfwClipboard) Get() string  { return c.w.GetClipboardString() }
func (c glfwClipboard) Set(s string) { c.w.SetClipboardString(s) }

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
	// InjectText types this text into the focused widget after the first
	// frame, for unattended runs.
	InjectText string
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
	epoch := time.Now() // frame timestamps are measured from here
	router := widgets.NewPointerRouter(owner)
	owner.Platform = services.MacOS
	owner.Clipboard = glfwClipboard{win}
	owner.Scheduler().Clock = func() time.Duration { return time.Since(epoch) }
	// Committed text (typing, and IME text once composition finishes).
	win.SetCharCallback(func(_ *glfw.Window, ch rune) { owner.DispatchText(string(ch)) })
	win.SetKeyCallback(func(_ *glfw.Window, k glfw.Key, _ int, a glfw.Action, m glfw.ModifierKey) {
		if a == glfw.Release {
			return
		}
		if key, ok := keyMap[k]; ok {
			owner.DispatchKey(services.KeyEvent{Key: key,
				Shift: m&glfw.ModShift != 0, Ctrl: m&glfw.ModControl != 0,
				Alt: m&glfw.ModAlt != 0, Meta: m&glfw.ModSuper != 0})
		}
	})
	router.Now = func() float64 { return time.Since(epoch).Seconds() }
	owner.DevicePixelRatio = float64(layer.Scale())
	ww, wh := win.GetSize()
	owner.SetViewSize(ui.Size{Width: float64(ww), Height: float64(wh)})
	owner.RunApp(root)

	// GLFW reports cursor positions in logical points, which is what layout uses.
	win.SetMouseButtonCallback(func(w *glfw.Window, b glfw.MouseButton, a glfw.Action, _ glfw.ModifierKey) {
		if b != glfw.MouseButtonLeft {
			return
		}
		x, y := w.GetCursorPos()
		pos := ui.Offset{Dx: x, Dy: y}
		if a == glfw.Press {
			router.PointerDown(pos)
		} else if a == glfw.Release {
			router.PointerUp(pos)
		}
	})
	win.SetCursorPosCallback(func(w *glfw.Window, x, y float64) {
		if w.GetMouseButton(glfw.MouseButtonLeft) == glfw.Press {
			router.PointerMove(ui.Offset{Dx: x, Dy: y})
		}
	})
	// Wheel/trackpad: GLFW reports lines (or fine trackpad deltas); positive y
	// means "scroll up", i.e. move content down.
	win.SetScrollCallback(func(w *glfw.Window, _, yoff float64) {
		x, y := w.GetCursorPos()
		router.Scroll(ui.Offset{Dx: x, Dy: y}, -yoff*WheelPixelsPerLine)
	})
	win.SetSizeCallback(func(*glfw.Window, int, int) { needsFrame = true })
	// The window system asks for a redraw when the window is first shown,
	// uncovered, or changes screens. Without this, a frame drawn before the
	// window was visible is never replaced.
	win.SetRefreshCallback(func(*glfw.Window) { needsFrame = true })

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
			owner.DrawFrameAt(start.Sub(epoch), ui.Size{Width: float64(w), Height: float64(h)}, &impellerCanvas{b, paint})
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
				router.Scroll(ui.Offset{Dx: float64(w) / 2, Dy: float64(h) / 2}, o.InjectScroll)
				o.InjectScroll = 0
			}
			if o.InjectText != "" {
				owner.DispatchText(o.InjectText)
				o.InjectText = ""
			}
			if o.InjectTaps > 0 {
				o.InjectTaps--
				pos := ui.Offset{Dx: o.InjectTapAt[0], Dy: o.InjectTapAt[1]}
				router.PointerDown(pos)
				router.PointerUp(pos)
			}
			if o.MaxFrames > 0 && frames >= o.MaxFrames {
				break
			}
		}
		if needsFrame {
			glfw.PollEvents() // animating: don't sleep; presenting paces us to the display
		} else if next, ok := owner.Scheduler().NextTimer(); ok {
			// Sleep until input or the next timer (e.g. cursor blink).
			glfw.WaitEventsTimeout(max(0.001, (next - time.Since(epoch)).Seconds()))
		} else {
			glfw.WaitEvents()
		}
		owner.Scheduler().RunTimers(time.Since(epoch))
	}
	if rr := owner.RenderRoot(); rr != nil {
		r.Dispose(rr)
	}
	return nil
}
