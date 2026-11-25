// Package widgets is the declarative layer: immutable widget descriptions,
// long-lived elements that reconcile them, and State for stateful widgets.
//
// A widget is any Go value. Its element kind is chosen by which interface it
// implements:
//
//	StatelessWidget   Build(ctx) Widget
//	StatefulWidget    CreateState() State
//	RenderWidget      CreateRenderObject() / UpdateRenderObject(r)
//	ParentDataWidget  ApplyParentData(b) + ChildWidget()
package widgets

import (
	"fmt"
	"reflect"
	"sort"
	"time"

	"github.com/jonathanperis/flutter-go/animation"
	r "github.com/jonathanperis/flutter-go/rendering"
	"github.com/jonathanperis/flutter-go/textedit"
)

type Widget any

type StatelessWidget interface {
	Build(ctx *Element) Widget
}

type StatefulWidget interface {
	CreateState() State
}

// State holds mutable state for a StatefulWidget. Embed StateOf[W].
type State interface {
	Build(ctx *Element) Widget
	base() *stateBase
}

// Optional State lifecycle hooks.
type (
	initStater   interface{ InitState() }
	didUpdater   interface{ DidUpdateWidget(old Widget) }
	disposer     interface{ Dispose() }
	singleChild  interface{ ChildWidget() Widget }
	multiChild   interface{ ChildWidgets() []Widget }
	keyedWidget  interface{ WidgetKey() any }
	parentDataer interface{ ApplyParentData(b *r.Box) }
)

// RenderWidget creates and configures a render object. It may also implement
// ChildWidget() or ChildWidgets().
type RenderWidget interface {
	CreateRenderObject() r.RenderBox
	UpdateRenderObject(ro r.RenderBox)
}

// ParentDataWidget configures its child's parent data (Expanded, Flexible).
type ParentDataWidget interface {
	ApplyParentData(b *r.Box)
	ChildWidget() Widget
}

type stateBase struct {
	el     *Element
	widget Widget
}

func (s *stateBase) base() *stateBase { return s }

// StateOf is embedded in a State struct for widget type W.
type StateOf[W Widget] struct{ stateBase }

// Widget returns the current configuration.
func (s *StateOf[W]) Widget() W { return s.widget.(W) }

// Context returns the element this state is mounted at.
func (s *StateOf[W]) Context() *Element { return s.el }

// SetState runs fn and schedules a rebuild.
func (s *StateOf[W]) SetState(fn func()) {
	if s.el == nil || !s.el.mounted {
		panic("SetState called on an unmounted State")
	}
	if fn != nil {
		fn()
	}
	s.el.MarkNeedsBuild()
}

// ---------------------------------------------------------------------------

type kind int

const (
	kindStateless kind = iota
	kindStateful
	kindRender
	kindParentData
	kindLazy      // ListViewBuilder: children built on demand during layout
	kindInherited // InheritedWidget: provides data to descendants
)

// Element is a widget's instantiation at a location in the tree. It doubles
// as Flutter's BuildContext.
type Element struct {
	widget    Widget
	kind      kind
	parent    *Element
	children  []*Element
	state     State
	render    r.RenderBox       // own render object (kindRender and kindLazy)
	lazy      map[int]*Element  // live items by index (kindLazy only)
	ownScroll *ScrollController // used when a ListViewBuilder has none
	// inherited maps each InheritedWidget type to the nearest ancestor
	// element providing it (shared with the parent unless e provides one).
	inherited map[reflect.Type]*Element
	// dependents of an inherited element; dependencies of any element.
	dependents   map[*Element]struct{}
	dependencies map[*Element]struct{}
	owner        *BuildOwner
	depth        int
	dirty        bool
	mounted      bool
}

func (e *Element) Widget() Widget   { return e.widget }
func (e *Element) Parent() *Element { return e.parent }

// Children returns child elements.
func (e *Element) Children() []*Element { return e.children }

// RenderObject returns the nearest render object at or below e.
func (e *Element) RenderObject() r.RenderBox {
	for e != nil {
		if e.render != nil {
			return e.render
		}
		if len(e.children) == 0 {
			return nil
		}
		e = e.children[0]
	}
	return nil
}

func kindOf(w Widget) kind {
	switch w.(type) {
	case ListViewBuilder:
		return kindLazy
	case InheritedWidget:
		return kindInherited
	case StatelessWidget:
		return kindStateless
	case StatefulWidget:
		return kindStateful
	case RenderWidget:
		return kindRender
	case ParentDataWidget:
		return kindParentData
	}
	panic(fmt.Sprintf("widgets: %T is not a widget (implement Build, CreateState or CreateRenderObject)", w))
}

func keyOf(w Widget) any {
	if k, ok := w.(keyedWidget); ok {
		return k.WidgetKey()
	}
	return nil
}

// canUpdate is Flutter's Widget.canUpdate: same runtime type and key.
func canUpdate(old, new Widget) bool {
	return reflect.TypeOf(old) == reflect.TypeOf(new) && keyOf(old) == keyOf(new)
}

func (e *Element) MarkNeedsBuild() {
	if e.dirty || !e.mounted {
		return
	}
	e.dirty = true
	e.owner.scheduleBuild(e)
}

func inflate(w Widget, parent *Element, owner *BuildOwner) *Element {
	e := &Element{widget: w, kind: kindOf(w), parent: parent, owner: owner}
	if parent != nil {
		e.depth = parent.depth + 1
		e.inherited = parent.inherited
	}
	if e.kind == kindInherited {
		m := make(map[reflect.Type]*Element, len(e.inherited)+1)
		for k, v := range e.inherited {
			m[k] = v
		}
		m[reflect.TypeOf(w)] = e
		e.inherited = m
	}
	e.mount()
	return e
}

func (e *Element) mount() {
	e.mounted = true
	switch e.kind {
	case kindStateful:
		e.state = e.widget.(StatefulWidget).CreateState()
		sb := e.state.base()
		sb.el, sb.widget = e, e.widget
		if s, ok := e.state.(initStater); ok {
			s.InitState()
		}
		e.rebuild()
	case kindStateless:
		e.rebuild()
	case kindRender:
		e.render = e.widget.(RenderWidget).CreateRenderObject()
		e.updateChildWidgets()
		e.syncRenderChildren()
	case kindParentData:
		e.children = []*Element{e.updateChild(nil, e.widget.(ParentDataWidget).ChildWidget())}
	case kindLazy:
		e.mountLazy()
	case kindInherited:
		e.children = []*Element{e.updateChild(nil, e.widget.(InheritedWidget).ChildWidget())}
	}
}

func (e *Element) update(w Widget) {
	old := e.widget
	e.widget = w
	switch e.kind {
	case kindStateful:
		e.state.base().widget = w
		if s, ok := e.state.(didUpdater); ok {
			s.DidUpdateWidget(old)
		}
		e.rebuild()
	case kindStateless:
		e.rebuild()
	case kindRender:
		w.(RenderWidget).UpdateRenderObject(e.render)
		e.updateChildWidgets()
		e.syncRenderChildren()
	case kindParentData:
		e.children = []*Element{e.updateChild(e.children[0], w.(ParentDataWidget).ChildWidget())}
		e.nearestRenderAncestor().syncRenderChildren()
	case kindLazy:
		e.updateLazy(old.(ListViewBuilder))
	case kindInherited:
		if w.(InheritedWidget).UpdateShouldNotify(old) {
			for d := range e.dependents {
				d.didChangeDependencies()
			}
		}
		e.children = []*Element{e.updateChild(e.children[0], w.(InheritedWidget).ChildWidget())}
		if ra := e.nearestRenderAncestor(); ra != nil {
			ra.syncRenderChildren()
		}
	}
}

func (e *Element) didChangeDependencies() {
	if s, ok := e.state.(interface{ DidChangeDependencies() }); ok {
		s.DidChangeDependencies()
	}
	e.MarkNeedsBuild()
}

// rebuild re-runs Build for component (stateless/stateful) elements.
func (e *Element) rebuild() {
	e.dirty = false
	e.owner.Builds++
	var built Widget
	if e.kind == kindStateful {
		built = e.state.Build(e)
	} else {
		built = e.widget.(StatelessWidget).Build(e)
	}
	var old *Element
	if len(e.children) > 0 {
		old = e.children[0]
	}
	oldRender := old.RenderObject()
	child := e.updateChild(old, built)
	e.children = nil
	if child != nil {
		e.children = []*Element{child}
	}
	// If our subtree's top render object changed, the render parent must
	// re-collect its children.
	if child.RenderObject() != oldRender {
		if ra := e.nearestRenderAncestor(); ra != nil {
			ra.syncRenderChildren()
		}
	}
}

// sameWidget reports whether a and b are equal values. It is flutter-go's
// stand-in for Flutter's identical(child.widget, newWidget) check (which
// const widgets and the `widget.child` pattern rely on): Go structs have no
// identity, but comparable ones have equality. Widgets holding slices, maps
// or funcs are not comparable and always update.
func sameWidget(a, b Widget) (same bool) {
	defer func() {
		if recover() != nil {
			same = false // a field (e.g. a Child) holds an uncomparable value
		}
	}()
	return reflect.TypeOf(a).Comparable() && a == b
}

// updateChild is Flutter's Element.updateChild.
func (e *Element) updateChild(old *Element, w Widget) *Element {
	if w == nil {
		if old != nil {
			old.unmount()
		}
		return nil
	}
	if old != nil && !old.dirty && sameWidget(old.widget, w) {
		return old // nothing to do: same configuration, not dirty
	}
	if old != nil && canUpdate(old.widget, w) {
		old.update(w)
		return old
	}
	if old != nil {
		old.unmount()
	}
	return inflate(w, e, e.owner)
}

// updateChildWidgets reconciles a render element's child list. Keyed children
// are matched by key; unkeyed ones in order. This is a simplification of
// Flutter's updateChildren, with the same results for the common cases.
func (e *Element) updateChildWidgets() {
	switch w := e.widget.(type) {
	case multiChild:
		e.updateChildList(w.ChildWidgets())
	case singleChild:
		var old *Element
		if len(e.children) > 0 {
			old = e.children[0]
		}
		switch c := e.updateChild(old, w.ChildWidget()); {
		case c == nil:
			e.children = e.children[:0]
		case len(e.children) == 1:
			e.children[0] = c
		default:
			e.children = []*Element{c}
		}
	}
}

func (e *Element) updateChildList(ws []Widget) {
	// Fast path: same length, no keys, every pair updatable. No allocation.
	if len(ws) == len(e.children) {
		fast := true
		for i, c := range e.children {
			if keyOf(c.widget) != nil || !canUpdate(c.widget, ws[i]) {
				fast = false
				break
			}
		}
		if fast {
			for i, c := range e.children {
				if c.dirty || !sameWidget(c.widget, ws[i]) {
					c.update(ws[i])
				}
			}
			return
		}
	}
	keyed := map[any]*Element{}
	var unkeyed []*Element
	for _, c := range e.children {
		if k := keyOf(c.widget); k != nil {
			keyed[k] = c
		} else {
			unkeyed = append(unkeyed, c)
		}
	}
	next := make([]*Element, 0, len(ws))
	for _, w := range ws {
		var old *Element
		if k := keyOf(w); k != nil {
			old = keyed[k]
			delete(keyed, k)
		} else if len(unkeyed) > 0 && canUpdate(unkeyed[0].widget, w) {
			old, unkeyed = unkeyed[0], unkeyed[1:]
		}
		next = append(next, e.updateChild(old, w))
	}
	for _, c := range keyed {
		c.unmount()
	}
	for _, c := range unkeyed {
		c.unmount()
	}
	e.children = next
}

func (e *Element) nearestRenderAncestor() *Element {
	p := e.parent
	for p != nil && p.kind != kindRender && p.kind != kindLazy {
		p = p.parent
	}
	return p
}

// childRender returns the top render object in c's subtree, after applying
// the parent data of any ParentDataWidgets above it.
func childRender(c *Element) r.RenderBox {
	top := c
	for c.kind != kindRender && c.kind != kindLazy {
		if len(c.children) == 0 {
			return nil
		}
		c = c.children[0]
	}
	b := c.render.Base()
	oldFlex, oldFit := b.Flex, b.Fit
	b.Flex, b.Fit = 0, 0
	for p := top; p != c; p = p.children[0] {
		if p.kind == kindParentData {
			p.widget.(parentDataer).ApplyParentData(b)
		}
	}
	if b.Flex != oldFlex || b.Fit != oldFit {
		r.MarkNeedsLayout(c.render)
	}
	return c.render
}

// syncRenderChildren gives a render element's render object the render
// objects of its child subtrees. It allocates only when they changed.
func (e *Element) syncRenderChildren() {
	if e.kind == kindLazy {
		e.syncLazy()
		return
	}
	cur := e.render.Children()
	n, same := 0, true
	for _, c := range e.children {
		if ro := childRender(c); ro != nil {
			if n >= len(cur) || cur[n] != ro {
				same = false
			}
			n++
		}
	}
	if same && n == len(cur) {
		return
	}
	ros := make([]r.RenderBox, 0, n)
	for _, c := range e.children {
		if ro := childRender(c); ro != nil {
			ros = append(ros, ro)
		}
	}
	switch ro := e.render.(type) {
	case interface {
		SetChildren(self r.RenderBox, cs []r.RenderBox)
	}:
		ro.SetChildren(e.render, ros)
	case interface {
		SetChild(self, child r.RenderBox)
	}:
		var c r.RenderBox
		if len(ros) > 0 {
			c = ros[0]
		}
		ro.SetChild(e.render, c)
	}
}

func (e *Element) unmount() {
	for _, c := range e.children {
		c.unmount()
	}
	for p := range e.dependencies {
		delete(p.dependents, e)
	}
	e.dependencies = nil
	if s, ok := e.state.(disposer); ok {
		s.Dispose()
	}
	if d, ok := e.render.(r.Disposer); ok {
		d.Dispose()
	}
	e.mounted, e.children = false, nil
}

// Walk visits e and its descendants depth-first.
func (e *Element) Walk(visit func(*Element)) {
	visit(e)
	for _, c := range e.children {
		c.Walk(visit)
	}
}

// ---------------------------------------------------------------------------

// BuildOwner tracks dirty elements and the root of the tree.
type BuildOwner struct {
	dirty        []*Element
	root         *Element
	rootRender   *r.RenderView
	OnNeedsFrame func()
	// Builds counts Build calls, for tests and benchmarks.
	Builds int
	// Repainted is how many repaint boundaries the last frame repainted.
	Repainted int
	sched     *animation.Scheduler
	// DevicePixelRatio is physical pixels per logical pixel (sets scroll
	// physics tolerances). 0 means 1.
	DevicePixelRatio float64
	// Platform selects keyboard shortcuts (flutter_test's default is Android).
	Platform textedit.Platform
	// Clipboard backs copy/cut/paste; nil disables them.
	Clipboard textedit.Clipboard
	focus     KeyTarget
	app       Widget
	viewSize  r.Size
	// pendingRoot is a RunApp widget waiting for the next build phase.
	pendingRoot *rootWidget
}

// Scheduler returns the frame scheduler that drives tickers.
func (o *BuildOwner) Scheduler() *animation.Scheduler {
	if o.sched == nil {
		o.sched = animation.NewScheduler()
		o.sched.RequestFrame = func() {
			if o.OnNeedsFrame != nil {
				o.OnNeedsFrame()
			}
		}
	}
	return o.sched
}

// Scheduler returns the frame scheduler (Flutter's TickerProvider/vsync).
func (e *Element) Scheduler() *animation.Scheduler { return e.owner.Scheduler() }

func (o *BuildOwner) scheduleBuild(e *Element) {
	o.dirty = append(o.dirty, e)
	if o.OnNeedsFrame != nil {
		o.OnNeedsFrame()
	}
}

// Root returns the root element.
func (o *BuildOwner) Root() *Element { return o.root }

// RenderRoot returns the root of the render tree.
func (o *BuildOwner) RenderRoot() r.RenderBox { return o.rootRender }

// RunApp mounts w as the root widget, replacing any previous root.
func (o *BuildOwner) RunApp(w Widget) {
	if o.rootRender == nil {
		o.rootRender = r.NewView()
		r.SetRoot(o.rootRender, func() {
			if o.OnNeedsFrame != nil {
				o.OnNeedsFrame()
			}
		})
	}
	o.app = w
	root := o.rootFor(w)
	if o.root == nil {
		// First mount builds synchronously, like Flutter's attachRootWidget.
		o.root = inflate(root, nil, o)
		return
	}
	// Later calls are applied inside the next frame's build phase, as
	// Flutter's RootWidget.attach does (markNeedsBuild on the root).
	o.pendingRoot = &root
	if o.OnNeedsFrame != nil {
		o.OnNeedsFrame()
	}
}

// rootFor wraps the app in a MediaQuery describing the view, as Flutter's
// View widget does.
func (o *BuildOwner) rootFor(app Widget) rootWidget {
	dpr := o.DevicePixelRatio
	if dpr == 0 {
		dpr = 1
	}
	return rootWidget{child: MediaQuery{Size: o.viewSize, DevicePixelRatio: dpr, Child: app}, render: o.rootRender}
}

// SetViewSize sets the logical size of the view before the first frame,
// so the first build already sees the right MediaQuery.
func (o *BuildOwner) SetViewSize(size r.Size) { o.viewSize = size }

// NeedsBuild reports whether any element is dirty.
func (o *BuildOwner) NeedsBuild() bool { return len(o.dirty) > 0 || o.pendingRoot != nil }

// BuildScope rebuilds dirty elements, parents before children.
func (o *BuildOwner) BuildScope() {
	if p := o.pendingRoot; p != nil {
		o.pendingRoot = nil
		o.root.update(*p)
	}
	for len(o.dirty) > 0 {
		dirty := o.dirty
		o.dirty = nil
		sort.SliceStable(dirty, func(i, j int) bool { return dirty[i].depth < dirty[j].depth })
		for _, e := range dirty {
			if e.dirty && e.mounted {
				e.rebuild()
			}
		}
	}
}

// DrawFrame draws a frame at the current timestamp (no time passes).
func (o *BuildOwner) DrawFrame(size r.Size, cv r.Canvas) {
	o.DrawFrameAt(o.Scheduler().Now(), size, cv)
}

// DrawFrameAt runs one frame at timestamp ts: tickers, then build, layout
// and paint, in Flutter's order.
func (o *BuildOwner) DrawFrameAt(ts time.Duration, size r.Size, cv r.Canvas) {
	s := o.Scheduler()
	s.BeginFrame(ts)
	if size != o.viewSize {
		// The view changed size: update MediaQuery; its dependents rebuild.
		o.viewSize = size
		if o.root != nil && o.app != nil && o.pendingRoot == nil {
			root := o.rootFor(o.app)
			o.pendingRoot = &root
		}
	}
	o.BuildScope()
	r.Layout(o.rootRender, r.Tight(size))
	o.Repainted = r.Composite(o.rootRender, cv, r.Offset{})
	s.EndFrame()
}

// rootWidget hosts the app's widget under the owner's root render object.
type rootWidget struct {
	child  Widget
	render r.RenderBox
}

func (w rootWidget) CreateRenderObject() r.RenderBox { return w.render }
func (w rootWidget) UpdateRenderObject(r.RenderBox)  {}
func (w rootWidget) ChildWidget() Widget             { return w.child }

// StateOfElement returns the State of a stateful element (Flutter's
// tester.state), or nil.
func StateOfElement(e *Element) State { return e.state }

// InheritedWidget provides data to descendants that depend on it with
// DependOn. When it is rebuilt with UpdateShouldNotify(old) true, its
// dependents rebuild, even if the widgets in between don't.
type InheritedWidget interface {
	ChildWidget() Widget
	UpdateShouldNotify(old Widget) bool
}

// DependOn returns the nearest ancestor InheritedWidget of type T and
// registers ctx to rebuild when it changes (Flutter's
// dependOnInheritedWidgetOfExactType).
func DependOn[T InheritedWidget](ctx *Element) (T, bool) {
	var zero T
	p, ok := ctx.inherited[reflect.TypeOf(zero)]
	if !ok {
		return zero, false
	}
	if p.dependents == nil {
		p.dependents = map[*Element]struct{}{}
	}
	p.dependents[ctx] = struct{}{}
	if ctx.dependencies == nil {
		ctx.dependencies = map[*Element]struct{}{}
	}
	ctx.dependencies[p] = struct{}{}
	return p.widget.(T), true
}

// Find returns the nearest ancestor InheritedWidget of type T without
// registering a dependency (Flutter's getInheritedWidgetOfExactType).
func Find[T InheritedWidget](ctx *Element) (T, bool) {
	var zero T
	if p, ok := ctx.inherited[reflect.TypeOf(zero)]; ok {
		return p.widget.(T), true
	}
	return zero, false
}
