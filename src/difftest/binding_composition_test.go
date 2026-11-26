package difftest

// Go twin of flutter/test/binding_composition_test.dart: a binding composed
// like WidgetsFlutterBinding's lower layers (BindingBase with
// GestureBinding, SchedulerBinding, ServicesBinding, SemanticsBinding) with
// logging mixins interleaved, composed with foundation/binding_mixins.go's
// pattern; the order of initInstances, initServiceExtensions, unlocked,
// createBinaryMessenger and handleAppLifecycleStateChanged, the instances
// set at each step and the service extensions registered are compared
// line by line with Dart's linearization.

import (
	"encoding/json"
	"os"
	"strconv"
	"testing"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/gestures"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/scheduler"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/semantics"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/services"
	"github.com/jonathanperis/flutter-go/src/async"
)

var bcLog []string

func bclog(s string) { bcLog = append(bcLog, s) }

func bcInstances() string {
	probe := func(name string, get func()) (s string) {
		defer func() {
			if recover() != nil {
				s = "-"
			}
		}()
		get()
		return name
	}
	return probe("G", func() { gestures.GestureBindingInstance() }) +
		probe("S", func() { scheduler.SchedulerBindingInstance() }) +
		probe("V", func() { services.ServicesBindingInstance() }) +
		probe("M", func() { semantics.SemanticsBindingInstance() })
}

func bcLogInit(name string, superCall func()) {
	bclog(name + ".initInstances before " + bcInstances())
	superCall()
	bclog(name + ".initInstances after " + bcInstances())
}

// logMixin is the Dart test's L0, L1, L2 (`on BindingBase`).
type logMixin struct {
	name string
	self foundation.BindingBase
}

func (m *logMixin) InitInstances() {
	bcLogInit(m.name, foundation.BindingBaseSuper(m.self, m).InitInstances)
}
func (m *logMixin) InitServiceExtensions() {
	bclog(m.name + ".initServiceExtensions before")
	foundation.BindingBaseSuper(m.self, m).InitServiceExtensions()
	bclog(m.name + ".initServiceExtensions after")
}
func (m *logMixin) Unlocked() {
	bclog(m.name + ".unlocked before")
	foundation.BindingBaseSuper(m.self, m).Unlocked()
	bclog(m.name + ".unlocked after")
}

type lifecycleHandler interface {
	HandleAppLifecycleStateChanged(state ui.AppLifecycleState)
}

// l3Mixin is L3 (`on BindingBase, ServicesBinding`).
type l3Mixin struct{ self services.ServicesBinding }

func (m *l3Mixin) InitInstances() {
	bcLogInit("L3", foundation.BindingBaseSuper(m.self, m).InitInstances)
}
func (m *l3Mixin) InitServiceExtensions() {
	bclog("L3.initServiceExtensions before")
	foundation.BindingBaseSuper(m.self, m).InitServiceExtensions()
	bclog("L3.initServiceExtensions after")
}
func (m *l3Mixin) CreateBinaryMessenger() services.BinaryMessenger {
	bclog("L3.createBinaryMessenger " + bcInstances())
	return foundation.BindingSuper[interface {
		CreateBinaryMessenger() services.BinaryMessenger
	}](m.self, m).CreateBinaryMessenger()
}
func (m *l3Mixin) HandleAppLifecycleStateChanged(state ui.AppLifecycleState) {
	bclog("L3.handleAppLifecycleStateChanged " + state.Name() + " before")
	foundation.BindingSuper[lifecycleHandler](m.self, m).HandleAppLifecycleStateChanged(state)
	bclog("L3.handleAppLifecycleStateChanged " + state.Name() + " after " + m.self.LifecycleState().Name())
}

// l4Mixin is L4 (`on BindingBase, SchedulerBinding`).
type l4Mixin struct{ self scheduler.SchedulerBinding }

func (m *l4Mixin) InitInstances() {
	bcLogInit("L4", foundation.BindingBaseSuper(m.self, m).InitInstances)
}
func (m *l4Mixin) InitServiceExtensions() {
	bclog("L4.initServiceExtensions before")
	foundation.BindingBaseSuper(m.self, m).InitServiceExtensions()
	bclog("L4.initServiceExtensions after")
}
func (m *l4Mixin) Unlocked() {
	bclog("L4.unlocked before")
	foundation.BindingBaseSuper(m.self, m).Unlocked()
	bclog("L4.unlocked after")
}
func (m *l4Mixin) HandleAppLifecycleStateChanged(state ui.AppLifecycleState) {
	bclog("L4.handleAppLifecycleStateChanged " + state.Name())
	foundation.BindingSuper[lifecycleHandler](m.self, m).HandleAppLifecycleStateChanged(state)
}

// ComposedBinding is `BindingBase with L0, GestureBinding, L1,
// SchedulerBinding, L2, ServicesBinding, L3, SemanticsBinding, L4`.
type ComposedBinding struct {
	foundation.BindingBaseBase
	L0 logMixin
	gestures.GestureBindingMixin
	L1 logMixin
	scheduler.SchedulerBindingMixin
	L2 logMixin
	services.ServicesBindingMixin
	l3Mixin
	semantics.SemanticsBindingMixin
	l4Mixin
}

func newComposedBinding() *ComposedBinding {
	b := &ComposedBinding{}
	b.L0 = logMixin{name: "L0", self: b}
	b.GestureBindingMixin.SetSelf(b)
	b.L1 = logMixin{name: "L1", self: b}
	b.SchedulerBindingMixin.SetSelf(b)
	b.L2 = logMixin{name: "L2", self: b}
	b.ServicesBindingMixin.SetSelf(b)
	b.l3Mixin.self = b
	b.SemanticsBindingMixin.SetSelf(b)
	b.l4Mixin.self = b
	b.Construct(b, &b.L0, &b.GestureBindingMixin, &b.L1, &b.SchedulerBindingMixin, &b.L2,
		&b.ServicesBindingMixin, &b.l3Mixin, &b.SemanticsBindingMixin, &b.l4Mixin)
	return b
}

// The class's own overrides (`super` is the last mixin defining each).

func (b *ComposedBinding) InitInstances() {
	bcLogInit("ComposedBinding", b.l4Mixin.InitInstances)
}
func (b *ComposedBinding) InitServiceExtensions() { b.l4Mixin.InitServiceExtensions() }
func (b *ComposedBinding) Unlocked()              { b.l4Mixin.Unlocked() }
func (b *ComposedBinding) HandleAppLifecycleStateChanged(state ui.AppLifecycleState) {
	b.l4Mixin.HandleAppLifecycleStateChanged(state)
}
func (b *ComposedBinding) RegisterServiceExtension(name string, callback foundation.ServiceExtensionCallback) {
	bclog("registerServiceExtension " + name + " " + bcInstances())
	b.BindingBaseBase.RegisterServiceExtension(name, callback)
}
func (b *ComposedBinding) CreateBinaryMessenger() services.BinaryMessenger {
	bclog("ComposedBinding.createBinaryMessenger")
	return b.l3Mixin.CreateBinaryMessenger()
}
func (b *ComposedBinding) PerformSemanticsAction(ui.SemanticsActionEvent) {}

func TestBindingCompositionMatchesFlutter(t *testing.T) {
	data, err := os.ReadFile("flutter/build/binding_composition.json")
	if err != nil {
		t.Skipf("no reference (%v); run scripts/difftest.sh", err)
	}
	if !isolated(t) { // it constructs its own binding
		return
	}
	var ref struct{ Log []string }
	if err := json.Unmarshal(data, &ref); err != nil {
		t.Fatal(err)
	}
	loop := async.NewLoop()
	defer async.SetCurrent(loop)()

	b := newComposedBinding()
	loop.FlushMicrotasks()
	loop.RunPosted()
	bclog("constructed " + bcInstances())
	bclog("debugBindingType " + foundation.BindingBaseDebugBindingType().Name())
	bclog("toString " + b.String())
	bclog("same instances " + strconv.FormatBool(gestures.GestureBindingInstance() == gestures.GestureBinding(b)) + " " +
		strconv.FormatBool(scheduler.SchedulerBindingInstance() == scheduler.SchedulerBinding(b)) + " " +
		strconv.FormatBool(services.ServicesBindingInstance() == services.ServicesBinding(b)) + " " +
		strconv.FormatBool(semantics.SemanticsBindingInstance() == semantics.SemanticsBinding(b)))
	pd := b.PlatformDispatcher()
	bclog("dispatcher pointer=" + strconv.FormatBool(pd.OnPointerDataPacket() != nil) +
		" semanticsEnabled=" + strconv.FormatBool(pd.OnSemanticsEnabledChanged() != nil) +
		" semanticsAction=" + strconv.FormatBool(pd.OnSemanticsActionEvent() != nil) +
		" a11y=" + strconv.FormatBool(pd.OnAccessibilityFeaturesChanged() != nil) +
		" timings=" + strconv.FormatBool(pd.OnReportTimings() != nil) +
		" beginFrame=" + strconv.FormatBool(pd.OnBeginFrame() != nil))

	bclog("lockEvents")
	done := false
	async.OnComplete(b.LockEvents(func() async.Future[async.Void] {
		return async.AsyncVoid(func() { bclog("locked " + strconv.FormatBool(b.Locked())) })
	}), func(async.Void) { done = true }, nil)
	loop.FlushMicrotasks()
	if !done {
		t.Fatal("lockEvents did not complete")
	}
	bclog("after lockEvents locked=" + strconv.FormatBool(b.Locked()))

	bclog("lifecycle message")
	msg := services.StringCodec{}.EncodeMessage(ptr("AppLifecycleState.paused"))
	b.DefaultBinaryMessenger().HandlePlatformMessage("flutter/lifecycle", msg, func([]byte) {})
	loop.FlushMicrotasks()
	loop.RunPosted()
	loop.FlushMicrotasks()
	bclog("lifecycleState " + b.LifecycleState().Name() + " framesEnabled " + strconv.FormatBool(b.FramesEnabled()))
	b.HandleAppLifecycleStateChanged(ui.AppLifecycleStateResumed)
	bclog("lifecycleState " + b.LifecycleState().Name() + " framesEnabled " + strconv.FormatBool(b.FramesEnabled()))

	for i := 0; i < max(len(bcLog), len(ref.Log)); i++ {
		var w, g string
		if i < len(ref.Log) {
			w = ref.Log[i]
		}
		if i < len(bcLog) {
			g = bcLog[i]
		}
		if w != g {
			t.Fatalf("line %d:\nflutter: %s\ngo:      %s", i, w, g)
		}
	}
}
