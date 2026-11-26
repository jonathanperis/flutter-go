package widgets

// The bridge from the flutter-go render tree to the gestures library, until
// widgets' GestureDetector/RawGestureDetector/Listener and rendering's
// hit testing are ported: BuildOwner.HitTestGestures is the part of
// RendererBinding.hitTestInView that finds the gesture targets (each
// GestureDetector's TapGestureRecognizer, each list's
// VerticalDragGestureRecognizer, as ScrollableState configures it), and
// PointerRouter turns pointer positions into the PointerEvents a test or
// an embedder sends to GestureBinding.handlePointerEvent.

import (
	"time"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/gestures"
	r "github.com/jonathanperis/flutter-go/packages/flutter/lib/src/rendering"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/services"
)

// Fling velocity limits and the touch slop (gestures' kMinFlingVelocity,
// kMaxFlingVelocity and kTouchSlop).
const (
	MinFlingVelocity = gestures.KMinFlingVelocity
	MaxFlingVelocity = gestures.KMaxFlingVelocity
	TouchSlop        = gestures.KTouchSlop
)

// tapTarget is a GestureDetector's RawGestureDetector: a TapGestureRecognizer
// for a RenderTapRegion.
type tapTarget struct {
	region *r.RenderTapRegion
	tap    *gestures.TapGestureRecognizer
}

func newTapTarget(region *r.RenderTapRegion) *tapTarget {
	t := &tapTarget{region: region, tap: gestures.NewTapGestureRecognizer()}
	t.tap.OnTapUp = func(details *gestures.TapUpDetails) {
		if t.region.OnTapAt != nil {
			t.region.OnTapAt(details.GlobalPosition())
		}
	}
	t.tap.OnTap = func() {
		if t.region.OnTap != nil {
			t.region.OnTap()
		}
	}
	return t
}

// HandleEvent is RenderPointerListener's: a down adds the pointer to the
// recognizer.
func (t *tapTarget) HandleEvent(event gestures.PointerEvent, entry gestures.HitTestEntry) {
	if down, ok := event.(*gestures.PointerDownEvent); ok {
		t.tap.AddPointer(down)
	}
}

// scrollTarget is a Scrollable's RawGestureDetector: a
// VerticalDragGestureRecognizer configured from the list's physics, and
// the pointer signal handler (Scrollable's Listener) for wheel scrolls.
type scrollTarget struct {
	list     *r.RenderFixedExtentList
	drag     *gestures.VerticalDragGestureRecognizer
	platform services.Platform
}

func newScrollTarget(list *r.RenderFixedExtentList) *scrollTarget {
	t := &scrollTarget{list: list, drag: gestures.NewVerticalDragGestureRecognizer()}
	t.drag.OnDown = func(*gestures.DragDownDetails) {
		if t.list.Hold != nil {
			t.list.Hold()
		}
	}
	t.drag.OnStart = func(details *gestures.DragStartDetails) {
		if t.list.DragStart != nil {
			t.list.DragStart(timeStampOf(details.SourceTimeStamp))
		}
	}
	t.drag.OnUpdate = func(details *gestures.DragUpdateDetails) {
		if t.list.DragUpdate != nil && details.PrimaryDelta != nil {
			t.list.DragUpdate(*details.PrimaryDelta, timeStampOf(details.SourceTimeStamp))
		}
	}
	t.drag.OnEnd = func(details *gestures.DragEndDetails) {
		if t.list.DragEnd != nil {
			v := 0.0
			if details.PrimaryVelocity != nil {
				v = *details.PrimaryVelocity
			}
			t.list.DragEnd(v)
		}
	}
	t.drag.OnCancel = func() {
		if t.list.DragCancel != nil {
			t.list.DragCancel()
		}
	}
	t.drag.VelocityTrackerBuilder = func(event gestures.PointerEvent) *gestures.VelocityTracker {
		switch t.platform { // ScrollBehavior.velocityTrackerBuilder
		case services.IOS:
			return &gestures.NewIOSScrollViewFlingVelocityTracker(event.Kind()).VelocityTracker
		case services.MacOS:
			return &gestures.NewMacOSScrollViewFlingVelocityTracker(event.Kind()).VelocityTracker
		}
		return gestures.NewVelocityTrackerWithKind(event.Kind())
	}
	return t
}

func timeStampOf(ts *time.Duration) time.Duration {
	if ts == nil {
		return 0
	}
	return *ts
}

// configure is ScrollableState's recognizer setup from the physics.
func (t *scrollTarget) configure(platform services.Platform) {
	t.platform = platform
	if t.list.FlingLimits != nil {
		minDistance, minVelocity, maxVelocity := t.list.FlingLimits()
		t.drag.MinFlingDistance, t.drag.MinFlingVelocity, t.drag.MaxFlingVelocity = &minDistance, &minVelocity, &maxVelocity
	}
}

func (t *scrollTarget) HandleEvent(event gestures.PointerEvent, entry gestures.HitTestEntry) {
	switch e := event.(type) {
	case *gestures.PointerDownEvent:
		t.drag.AddPointer(e)
	case *gestures.PointerScrollEvent:
		// Scrollable._receivedPointerSignal: one scrollable handles a wheel
		// event (the deepest, registered first).
		gestures.GestureBindingInstance().PointerSignalResolver().Register(e, func(gestures.PointerSignalEvent) {
			if t.list.PointerScroll != nil {
				t.list.PointerScroll(e.ScrollDelta().Dy)
			} else {
				t.list.ScrollBy(e.ScrollDelta().Dy)
			}
		})
	}
}

// HitTestGestures adds the gesture targets under position (logical pixels,
// in the root's coordinates) to result, deepest first: a TapGestureRecognizer
// per tap region and a VerticalDragGestureRecognizer for the deepest list.
// A list whose scroll activity ignores pointers (a fling or animation is
// running) hides its items, as Scrollable's IgnorePointer does.
func (o *BuildOwner) HitTestGestures(result *gestures.HitTestResult, position ui.Offset) {
	if o.rootRender == nil {
		return
	}
	if o.gestureTargets == nil {
		o.gestureTargets = map[r.RenderBox]gestures.HitTestTarget{}
	}
	path := r.HitTest(o.rootRender, position)
	listAt := -1
	for i, ro := range path {
		if _, ok := ro.(*r.RenderFixedExtentList); ok {
			listAt = i
			break
		}
	}
	ignoreItems := false
	if listAt >= 0 {
		list := path[listAt].(*r.RenderFixedExtentList)
		ignoreItems = list.IgnoringPointer != nil && list.IgnoringPointer()
	}
	for i, ro := range path {
		switch ro := ro.(type) {
		case *r.RenderTapRegion:
			if ignoreItems && i < listAt {
				continue
			}
			t, ok := o.gestureTargets[ro].(*tapTarget)
			if !ok {
				t = newTapTarget(ro)
				o.gestureTargets[ro] = t
			}
			result.Add(gestures.NewHitTestEntry(t))
		case *r.RenderFixedExtentList:
			if i != listAt {
				continue
			}
			t, ok := o.gestureTargets[ro].(*scrollTarget)
			if !ok {
				t = newScrollTarget(ro)
				o.gestureTargets[ro] = t
			}
			t.configure(o.Platform)
			result.Add(gestures.NewHitTestEntry(t))
		}
	}
}

// PointerRouter sends one pointer's events (as flutter_test's TestPointer
// builds them) to GestureBinding.handlePointerEvent.
type PointerRouter struct {
	owner       *BuildOwner
	pointer     int
	isDown      bool
	last        ui.Offset
	mouseAdded  bool
	nextPointer int
	// TimeStamp returns the time stamp of the next event (zero if nil).
	TimeStamp func() time.Duration
	// AfterEvent, if set, runs after each event is handled (flutter_test
	// awaits each event, which runs the microtasks it scheduled).
	AfterEvent func()
}

// NewPointerRouter creates a router for o's tree.
func NewPointerRouter(o *BuildOwner) *PointerRouter { return &PointerRouter{owner: o} }

func (p *PointerRouter) now() time.Duration {
	if p.TimeStamp == nil {
		return 0
	}
	return p.TimeStamp()
}

func (p *PointerRouter) send(event gestures.PointerEvent) {
	gestures.GestureBindingInstance().HandlePointerEvent(event)
	if p.AfterEvent != nil {
		p.AfterEvent()
	}
}

// PointerDown puts a new touch pointer down at pos.
func (p *PointerRouter) PointerDown(pos ui.Offset) {
	p.nextPointer++
	p.pointer = p.nextPointer
	p.isDown = true
	p.last = pos
	p.send(gestures.NewPointerDownEvent(gestures.PointerDownEventOptions{
		TimeStamp: p.now(), Pointer: p.pointer, Position: pos}))
}

// PointerMove moves the pointer to pos.
func (p *PointerRouter) PointerMove(pos ui.Offset) {
	if !p.isDown {
		return
	}
	delta := pos.Sub(p.last)
	p.last = pos
	p.send(gestures.NewPointerMoveEvent(gestures.PointerMoveEventOptions{
		TimeStamp: p.now(), Pointer: p.pointer, Position: pos, Delta: delta}))
}

// PointerUp lifts the pointer at pos (moving it there first if needed).
func (p *PointerRouter) PointerUp(pos ui.Offset) {
	if !p.isDown {
		return
	}
	if pos != p.last {
		p.PointerMove(pos)
	}
	p.isDown = false
	p.send(gestures.NewPointerUpEvent(gestures.PointerUpEventOptions{
		TimeStamp: p.now(), Pointer: p.pointer, Position: pos}))
}

// Scroll sends a mouse-wheel scroll of dy (positive scrolls content up) at
// pos.
func (p *PointerRouter) Scroll(pos ui.Offset, dy float64) {
	mouse := ui.PointerDeviceKindMouse
	if !p.mouseAdded {
		p.mouseAdded = true
		p.send(gestures.NewPointerAddedEvent(gestures.PointerAddedEventOptions{
			TimeStamp: p.now(), Kind: mouse, Device: 1, Position: pos}))
	}
	p.send(gestures.NewPointerScrollEvent(gestures.PointerScrollEventOptions{
		PointerSignalEventOptions: gestures.PointerSignalEventOptions{TimeStamp: p.now(), Kind: &mouse, Device: 1, Position: pos},
		ScrollDelta:               ui.NewOffset(0, dy)}))
}
