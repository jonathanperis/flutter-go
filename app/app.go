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
	"github.com/jonathanperis/flutter-go/textedit"
	"github.com/jonathanperis/flutter-go/widgets"
)

func init() { runtime.LockOSThread() } // Cocoa needs the main thread.

var keyMap = map[glfw.Key]textedit.Key{
	glfw.KeyBackspace: textedit.KeyBackspace, glfw.KeyDelete: textedit.KeyDelete,
	glfw.KeyLeft: textedit.KeyLeft, glfw.KeyRight: textedit.KeyRight,
	glfw.KeyUp: textedit.KeyUp, glfw.KeyDown: textedit.KeyDown,
	glfw.KeyHome: textedit.KeyHome, glfw.KeyEnd: textedit.KeyEnd,
	glfw.KeyEnter: textedit.KeyEnter, glfw.KeyKPEnter: textedit.KeyEnter,
	glfw.KeyTab: textedit.KeyTab, glfw.KeyA: textedit.KeyA, glfw.KeyC: textedit.KeyC,
	glfw.KeyV: textedit.KeyV, glfw.KeyX: textedit.KeyX,
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
	owner.Platform = textedit.MacOS
	owner.Clipboard = glfwClipboard{win}
	owner.Scheduler().Clock = func() time.Duration { return time.Since(epoch) }
	// Committed text (typing, and IME text once composition finishes).
	win.SetCharCallback(func(_ *glfw.Window, ch rune) { owner.DispatchText(string(ch)) })
	win.SetKeyCallback(func(_ *glfw.Window, k glfw.Key, _ int, a glfw.Action, m glfw.ModifierKey) {
		if a == glfw.Release {
			return
		}
		if key, ok := keyMap[k]; ok {
			owner.DispatchKey(textedit.KeyEvent{Key: key,
				Shift: m&glfw.ModShift != 0, Ctrl: m&glfw.ModControl != 0,
				Alt: m&glfw.ModAlt != 0, Meta: m&glfw.ModSuper != 0})
		}
	})
	router.Now = func() float64 { return time.Since(epoch).Seconds() }
	owner.DevicePixelRatio = float64(layer.Scale())
	ww, wh := win.GetSize()
	owner.SetViewSize(r.Size{W: float64(ww), H: float64(wh)})
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
			owner.DrawFrameAt(start.Sub(epoch), r.Size{W: float64(w), H: float64(h)}, &impellerCanvas{b, paint})
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
			if o.InjectText != "" {
				owner.DispatchText(o.InjectText)
				o.InjectText = ""
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
