// Package app runs a widget tree in a desktop window (macOS/Metal for now).
package app

import (
	"errors"
	"log"
	"runtime"
	"time"

	"github.com/go-gl/glfw/v3.3/glfw"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	r "github.com/jonathanperis/flutter-go/packages/flutter/lib/src/rendering"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/scheduler"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/services"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/widgets"
	"github.com/jonathanperis/flutter-go/src/async"
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

// appBinding is the app's binding (partial WidgetsFlutterBinding:
// BindingBase and SchedulerBinding); its persistent frame callback draws
// the tree into the frame being produced, as WidgetsBinding.drawFrame
// does.
type appBinding struct {
	scheduler.SchedulerBindingMixin
	// drawFrame is the rendering pipeline for the frame in progress.
	drawFrame func()
}

var binding *appBinding

func ensureBinding() *appBinding {
	if binding == nil {
		b := &appBinding{}
		base := &foundation.BindingBaseBase{}
		b.SchedulerBindingMixin.BindingBase = base
		b.SetSelf(b)
		base.Construct(b)
		b.AddPersistentFrameCallback(func(time.Duration) {
			if b.drawFrame != nil {
				b.drawFrame()
			}
		})
		binding = b
	}
	return binding
}

// engine is the embedder's side of dart:ui's PlatformConfiguration: the
// framework's frame requests (PlatformDispatcher.scheduleFrame) set
// needsFrame, and the run loop produces the frame at the next vsync.
type engine struct{ needsFrame bool }

func (e *engine) ScheduleFrame()             { e.needsFrame = true }
func (e *engine) EndWarmUpFrame()            {}
func (e *engine) SetNeedsReportTimings(bool) {} // ENGINE GAP: no frame timings are collected

// Run shows root in a window until it is closed.
//
// Frames follow Flutter's engine protocol: the framework requests one
// through SchedulerBinding.ScheduleFrame (PlatformDispatcher.scheduleFrame),
// and the loop produces it with PlatformConfiguration.BeginFrame (begin
// frame: transient callbacks; microtasks; draw frame: build, layout, paint)
// before presenting, paced by Metal presentation (vsync). Between frames,
// the event loop (src/async's current Loop, its clock following the
// wall clock) runs timers and microtasks.
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

	// dart:ui's engine context: images decoded or rasterized by dart:ui are
	// textures in it, so frames are presented with it too.
	ctx := ui.EngineImpellerContext()
	if ctx == nil {
		return errors.New("app: could not create an Impeller (Metal) context")
	}
	layer := metal.Attach(win.GetCocoaWindow())
	defer layer.Release()
	tc := impeller.NewTypographyContext()
	defer tc.Release()
	r.SetTextEngine(impellerText{tc})

	eng := &engine{}
	pc := ui.NewPlatformConfiguration(eng)
	defer ui.SetPlatformConfiguration(pc)()
	b := ensureBinding()
	loop := async.Current()
	owner := &widgets.BuildOwner{OnNeedsFrame: b.EnsureVisualUpdate}
	epoch := time.Now().Add(-loop.Elapsed()) // frame timestamps and the loop's clock are measured from here
	router := widgets.NewPointerRouter(owner)
	owner.Platform = services.MacOS
	owner.Clipboard = glfwClipboard{win}
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
	b.ScheduleFrame() // Flutter's runApp schedules a warm-up frame; here the first frame waits for the loop

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
	// A resize needs a frame at the new size even if frames are disabled
	// (Flutter's handleMetricsChanged: scheduleForcedFrame).
	win.SetSizeCallback(func(*glfw.Window, int, int) { b.ScheduleForcedFrame() })
	// The window system asks for a redraw when the window is first shown,
	// uncovered, or changes screens. Without this, a frame drawn before the
	// window was visible is never replaced. The tree itself is unchanged, so
	// this is an engine-side frame (the framework repaints from its layers).
	win.SetRefreshCallback(func(*glfw.Window) { eng.needsFrame = true })

	var frameNumber uint64
	for frames := 0; !win.ShouldClose(); {
		loop.ElapseTo(time.Since(epoch)) // timers due by now, with their microtasks
		loop.RunPosted()
		if eng.needsFrame {
			eng.needsFrame = false
			start := time.Now()
			w, h := win.GetSize()
			scale := float64(layer.Scale())
			ui.EngineDevicePixelRatio = scale
			// The frame is recorded with dart:ui's PictureRecorder/Canvas
			// (an Impeller display list), in physical pixels.
			rec := ui.NewPictureRecorder()
			canvas := ui.NewCanvas(rec)
			canvas.Scale(scale, scale)
			b.drawFrame = func() {
				owner.DrawFrame(ui.Size{Width: float64(w), Height: float64(h)}, NewCanvas(canvas))
			}
			frameNumber++
			b.EnsureFrameCallbacksRegistered() // a refresh can come before the framework asked for a frame
			pc.BeginFrame(start.Sub(epoch), frameNumber)
			b.drawFrame = nil
			pic := rec.EndRecording()
			cpu := time.Since(start)
			err := layer.Present(ctx.Handle(), ui.EnginePictureDisplayList(pic).Handle())
			pic.Dispose()
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
		if eng.needsFrame || loop.MicrotaskCount() > 0 {
			glfw.PollEvents() // animating: don't sleep; presenting paces us to the display
		} else if next, ok := loop.NextDeadline(); ok {
			// Sleep until input or the next timer (e.g. cursor blink).
			glfw.WaitEventsTimeout(max(0.001, (next - time.Since(epoch)).Seconds()))
		} else {
			glfw.WaitEvents()
		}
		loop.FlushMicrotasks() // microtasks queued by input handlers
	}
	if rr := owner.RenderRoot(); rr != nil {
		r.Dispose(rr)
	}
	return nil
}
