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
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/gestures"
	r "github.com/jonathanperis/flutter-go/packages/flutter/lib/src/rendering"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/scheduler"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/services"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/widgets"
	"github.com/jonathanperis/flutter-go/src/async"
	"github.com/jonathanperis/flutter-go/src/platform/metal"
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
	// InjectText types this text into the focused widget after the first
	// frame, for unattended runs.
	InjectText string
	// InjectKeys delivers these GLFW key events (and committed characters)
	// after the first frame, as the key and char callbacks would, for
	// unattended runs.
	InjectKeys []InjectedKey
	// InjectTaps synthesizes this many taps at InjectTapAt (logical px), one
	// per frame, for unattended runs.
	InjectTaps  int
	InjectTapAt [2]float64
	// Window receives the GLFW window once created (for tests and tools).
	Window func(w *glfw.Window)
}

// InjectedKey is a GLFW key event (Char: the character GLFW's char
// callback reports after it, 0 for none).
type InjectedKey struct {
	Key      glfw.Key
	Scancode int
	Action   glfw.Action
	Mods     glfw.ModifierKey
	Char     rune
}

// appBinding is the app's binding (partial WidgetsFlutterBinding:
// BindingBase with GestureBinding, SchedulerBinding, ServicesBinding); its persistent frame
// callback draws the tree into the frame being produced, as
// WidgetsBinding.drawFrame does, and its hit test finds the tree's gesture
// targets, as RendererBinding.hitTestInView does.
type appBinding struct {
	foundation.BindingBaseBase
	gestures.GestureBindingMixin
	scheduler.SchedulerBindingMixin
	services.ServicesBindingMixin
	// drawFrame is the rendering pipeline for the frame in progress.
	drawFrame func()
	// owner is the tree being shown.
	owner *widgets.BuildOwner
}

func (b *appBinding) InitInstances()         { b.ServicesBindingMixin.InitInstances() }
func (b *appBinding) InitServiceExtensions() { b.ServicesBindingMixin.InitServiceExtensions() }
func (b *appBinding) Unlocked()              { b.SchedulerBindingMixin.Unlocked() }

func (b *appBinding) HitTestInView(result *gestures.HitTestResult, position ui.Offset, viewID int) {
	if b.owner != nil {
		b.owner.HitTestGestures(result, position)
	}
	b.GestureBindingMixin.HitTestInView(result, position, viewID)
}

var binding *appBinding

func ensureBinding() *appBinding {
	if binding == nil {
		b := &appBinding{}
		b.GestureBindingMixin.SetSelf(b)
		b.SchedulerBindingMixin.SetSelf(b)
		b.ServicesBindingMixin.SetSelf(b)
		b.Construct(b, &b.GestureBindingMixin, &b.SchedulerBindingMixin, &b.ServicesBindingMixin)
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
// needsFrame, and the run loop produces the frame at the next vsync; the
// scene FlutterView.render hands over (the layer tree) is presented after
// the frame.
type engine struct {
	needsFrame bool
	layerTree  *ui.EngineLayerTree
	window     *glfw.Window // for the platform plugins (platform_plugins.go)
}

func (e *engine) ScheduleFrame()             { e.needsFrame = true }
func (e *engine) EndWarmUpFrame()            {}
func (e *engine) SetNeedsReportTimings(bool) {} // ENGINE GAP: no frame timings are collected

// Render receives the scene of the implicit view (the only one).
func (e *engine) Render(viewID int, tree *ui.EngineLayerTree, devicePixelRatio float64) {
	e.layerTree = tree
}

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

	r.SetTextEngine(r.UITextEngine{})

	eng := &engine{window: win}
	pc := ui.NewPlatformConfiguration(eng)
	defer ui.SetPlatformConfiguration(pc)()
	// The platform's metrics: the display, and the implicit view's size and
	// device pixel ratio (the backing scale factor, 2.0 on Retina).
	reportDisplay := func() {
		if mon := glfw.GetPrimaryMonitor(); mon != nil {
			if mode := mon.GetVideoMode(); mode != nil {
				sx, _ := mon.GetContentScale()
				pc.UpdateDisplays([]ui.DisplayData{displayData(mode.Width, mode.Height, mode.RefreshRate, float64(sx))})
			}
		}
	}
	reportMetrics := func() {
		ww, wh := win.GetSize()
		fw, fh := win.GetFramebufferSize()
		pc.UpdateViewMetrics(0, viewportMetrics(ww, wh, fw, fh, float64(layer.Scale())))
	}
	reportDisplay()
	reportMetrics()
	view := ui.PlatformDispatcherInstance().ImplicitView()
	b := ensureBinding()
	loop := async.Current()
	// ServicesBinding's channel handlers and keyboard sync complete in
	// microtasks and posted engine replies.
	loop.FlushMicrotasks()
	loop.RunPosted()
	loop.FlushMicrotasks()
	owner := &widgets.BuildOwner{OnNeedsFrame: b.EnsureVisualUpdate}
	epoch := time.Now().Add(-loop.Elapsed()) // frame timestamps and the loop's clock are measured from here
	router := widgets.NewPointerRouter(owner)
	owner.Platform = foundation.TargetPlatformMacOS
	// Keys reach the framework as the macOS embedder sends them (key data,
	// then raw key messages); the focus system's handleKeyMessage hands them
	// to the focused widget, and the key downs it doesn't handle go to the
	// text input host (macOS's key bindings and text insertion).
	keyboard := &glfwKeyboard{manager: newKeyboardManager(pc), host: textInputHost{focused: owner.Focus}}
	win.SetInputMode(glfw.LockKeyMods, glfw.True)
	b.KeyEventManager().KeyMessageHandler = owner.HandleKeyMessage
	win.SetKeyCallback(func(_ *glfw.Window, k glfw.Key, scancode int, a glfw.Action, m glfw.ModifierKey) {
		keyboard.onKey(k, scancode, a, m)
	})
	// Committed text (typing, and IME text once composition finishes) is
	// the key down's characters; the text input host inserts it when the
	// framework doesn't handle the key.
	win.SetCharCallback(func(_ *glfw.Window, ch rune) { keyboard.onChar(ch) })
	b.owner = owner
	router.TimeStamp = func() time.Duration { return time.Since(epoch) }
	router.AfterEvent = func() { loop.FlushMicrotasks() }
	owner.DevicePixelRatio = view.DevicePixelRatio()
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
	// A resize (or a move to a screen of another scale) changes the view's
	// metrics and needs a frame at the new size even if frames are disabled
	// (Flutter's handleMetricsChanged: scheduleForcedFrame).
	win.SetSizeCallback(func(*glfw.Window, int, int) {
		reportMetrics()
		b.ScheduleForcedFrame()
	})
	win.SetContentScaleCallback(func(*glfw.Window, float32, float32) {
		reportDisplay()
		reportMetrics()
		owner.DevicePixelRatio = view.DevicePixelRatio()
		b.ScheduleForcedFrame()
	})
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
			dpr := view.DevicePixelRatio()
			size := view.PhysicalSize()
			logical := ui.Size{Width: size.Width / dpr, Height: size.Height / dpr}
			// The frame is painted as RenderView composites it: the tree is
			// recorded with dart:ui's PictureRecorder/Canvas in logical
			// pixels, put in a scene under the root transform (device pixel
			// ratio), and shown with FlutterView.render.
			b.drawFrame = func() {
				rec := ui.NewPictureRecorder()
				canvas := ui.NewCanvas(rec)
				owner.DrawFrame(logical, NewCanvas(canvas))
				pic := rec.EndRecording()
				sb := ui.NewSceneBuilder()
				sb.PushTransform(rootTransform(dpr))
				sb.AddPicture(ui.Offset{}, pic)
				sb.Pop()
				scene := sb.Build()
				view.Render(scene)
				scene.Dispose()
				pic.Dispose()
			}
			frameNumber++
			b.EnsureFrameCallbacksRegistered() // a refresh can come before the framework asked for a frame
			eng.layerTree = nil
			pc.BeginFrame(start.Sub(epoch), frameNumber)
			b.drawFrame = nil
			cpu := time.Since(start)
			if tree := eng.layerTree; tree != nil {
				// The rasterizer: the layer tree flattened into a display
				// list, presented with Impeller.
				dl := tree.Flatten()
				if err := layer.Present(ctx.Handle(), dl.Handle()); err != nil {
					log.Print(err)
				}
				dl.Release()
				eng.layerTree = nil
			}
			w, h := int(logical.Width), int(logical.Height)
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
			for _, k := range o.InjectKeys {
				keyboard.onKey(k.Key, k.Scancode, k.Action, k.Mods)
				if k.Char != 0 {
					keyboard.onChar(k.Char)
				}
				keyboard.flush()
				loop.FlushMicrotasks()
			}
			o.InjectKeys = nil
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
		keyboard.flush()       // a key press GLFW's char callback didn't follow
		loop.FlushMicrotasks() // microtasks queued by input handlers
	}
	if rr := owner.RenderRoot(); rr != nil {
		r.Dispose(rr)
	}
	return nil
}
