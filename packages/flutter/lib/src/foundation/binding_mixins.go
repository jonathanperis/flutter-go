package foundation

// Binding mixins: how Go composes `class B extends BindingBase with M1, M2,
// ...` (see API_MAPPING.md, "Bindings").
//
// A binding mixin (`mixin SchedulerBinding on BindingBase`) is an interface
// (scheduler.SchedulerBinding, embedding BindingBase) plus an embeddable
// struct (scheduler.SchedulerBindingMixin) holding the mixin's state and
// methods. The struct embeds nothing: it keeps `self`, the binding it is
// mixed into (typed as its own interface), and calls every overridable
// method through it. A binding embeds BindingBaseBase and the mixin structs
// in Dart's `with` order, points each mixin at itself with SetSelf and
// calls Construct with the mixins in that order:
//
//	type MyBinding struct {
//		foundation.BindingBaseBase
//		gestures.GestureBindingMixin
//		scheduler.SchedulerBindingMixin
//	}
//
//	func NewMyBinding() *MyBinding {
//		b := &MyBinding{}
//		b.GestureBindingMixin.SetSelf(b)
//		b.SchedulerBindingMixin.SetSelf(b)
//		b.Construct(b, &b.GestureBindingMixin, &b.SchedulerBindingMixin)
//		return b
//	}
//
//	// Methods several mixins (or a mixin and BindingBase) define are
//	// ambiguous in Go; the binding defines them, calling the last mixin in
//	// `with` order that defines it (Dart's most-derived implementation).
//	func (b *MyBinding) InitInstances()         { b.SchedulerBindingMixin.InitInstances() }
//	func (b *MyBinding) InitServiceExtensions() { b.SchedulerBindingMixin.InitServiceExtensions() }
//	func (b *MyBinding) Unlocked()              { b.SchedulerBindingMixin.Unlocked() }
//
// Dart's `super.m()` inside a mixin is the closest mixin before it in the
// `with` list that implements m, or BindingBase: the mixin calls
// BindingSuper (any method) or BindingBaseSuper (BindingBase's methods),
// passing its own pointer, so the chain follows the binding's actual
// composition (Dart's linearization) whatever mixins it combines. A
// binding class overriding a method calls the most-derived mixin's method
// as its super (or BindingSuper with a nil mixin).

import (
	"fmt"
	"reflect"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/src/async"
)

func (b *BindingBaseBase) setMixins(mixins []any) {
	seen := map[any]bool{}
	for i, m := range mixins {
		v := reflect.ValueOf(m)
		if m == nil || v.Kind() != reflect.Pointer || v.IsNil() {
			panic(fmt.Sprintf("foundation: mixin %d passed to Construct is not a non-nil pointer to an embedded mixin struct", i))
		}
		if seen[m] {
			panic(fmt.Sprintf("foundation: mixin %T passed to Construct twice", m))
		}
		seen[m] = true
	}
	b.mixins = mixins
}

// BindingMixins returns the mixins passed to Construct, in `with` order.
func (b *BindingBaseBase) BindingMixins() []any { return b.mixins }

// BindingSuper is Dart's `super` inside a binding mixin: the implementation
// of I that `super.m()` reaches from mixin (a pointer to one of binding's
// mixins, as passed to Construct): the closest mixin before it in `with`
// order that implements I, else binding's BindingBaseBase if it does. A nil
// mixin starts after the last mixin (`super` in the binding class itself).
// It panics if nothing implements I (Dart rejects such a call at compile
// time).
func BindingSuper[I any](binding BindingBase, mixin any) I {
	if v, ok := bindingSuper[I](binding.bindingBase(), mixin); ok {
		return v
	}
	panic(fmt.Sprintf("foundation: no super implementation of %v below %T in %T", reflect.TypeFor[I](), mixin, binding))
}

func bindingSuper[I any](base *BindingBaseBase, mixin any) (I, bool) {
	i := len(base.mixins)
	if mixin != nil {
		i = -1
		for j, m := range base.mixins {
			if m == mixin {
				i = j
				break
			}
		}
		if i < 0 {
			panic(fmt.Sprintf("foundation: %T is not one of the binding's mixins (pass it to Construct)", mixin))
		}
	}
	for j := i - 1; j >= 0; j-- {
		if v, ok := base.mixins[j].(I); ok {
			return v, true
		}
	}
	if v, ok := any(base).(I); ok {
		return v, true
	}
	var zero I
	return zero, false
}

// BindingBaseSuper is BindingSuper for BindingBase's own methods: a
// BindingBase whose every method is the one `super` reaches from mixin
// (`BindingBaseSuper(m.self, m).InitInstances()` is Dart's
// `super.initInstances()`).
func BindingBaseSuper(binding BindingBase, mixin any) BindingBase {
	return bindingSuperView{binding: binding, from: mixin}
}

// bindingSuperView dispatches each BindingBase method to BindingSuper.
type bindingSuperView struct {
	binding BindingBase
	from    any
}

func superOf[I any](v bindingSuperView) I { return BindingSuper[I](v.binding, v.from) }

func (v bindingSuperView) Window() *ui.SingletonFlutterWindow {
	return superOf[interface {
		Window() *ui.SingletonFlutterWindow
	}](v).Window()
}

func (v bindingSuperView) PlatformDispatcher() *ui.PlatformDispatcher {
	return superOf[interface{ PlatformDispatcher() *ui.PlatformDispatcher }](v).PlatformDispatcher()
}

func (v bindingSuperView) InitInstances() { superOf[interface{ InitInstances() }](v).InitInstances() }

func (v bindingSuperView) InitServiceExtensions() {
	superOf[interface{ InitServiceExtensions() }](v).InitServiceExtensions()
}

func (v bindingSuperView) DebugCheckZone(entryPoint string) bool {
	return superOf[interface{ DebugCheckZone(string) bool }](v).DebugCheckZone(entryPoint)
}

func (v bindingSuperView) Locked() bool { return superOf[interface{ Locked() bool }](v).Locked() }

func (v bindingSuperView) LockEvents(callback func() async.Future[async.Void]) async.Future[async.Void] {
	return superOf[interface {
		LockEvents(func() async.Future[async.Void]) async.Future[async.Void]
	}](v).LockEvents(callback)
}

func (v bindingSuperView) Unlocked() { superOf[interface{ Unlocked() }](v).Unlocked() }

func (v bindingSuperView) ReassembleApplication() async.Future[async.Void] {
	return superOf[interface {
		ReassembleApplication() async.Future[async.Void]
	}](v).ReassembleApplication()
}

func (v bindingSuperView) PerformReassemble() async.Future[async.Void] {
	return superOf[interface {
		PerformReassemble() async.Future[async.Void]
	}](v).PerformReassemble()
}

func (v bindingSuperView) RegisterSignalServiceExtension(name string, callback AsyncCallback) {
	superOf[interface {
		RegisterSignalServiceExtension(string, AsyncCallback)
	}](v).RegisterSignalServiceExtension(name, callback)
}

func (v bindingSuperView) RegisterBoolServiceExtension(name string, getter AsyncValueGetter[bool], setter AsyncValueSetter[bool]) {
	superOf[interface {
		RegisterBoolServiceExtension(string, AsyncValueGetter[bool], AsyncValueSetter[bool])
	}](v).RegisterBoolServiceExtension(name, getter, setter)
}

func (v bindingSuperView) RegisterNumericServiceExtension(name string, getter AsyncValueGetter[float64], setter AsyncValueSetter[float64]) {
	superOf[interface {
		RegisterNumericServiceExtension(string, AsyncValueGetter[float64], AsyncValueSetter[float64])
	}](v).RegisterNumericServiceExtension(name, getter, setter)
}

func (v bindingSuperView) RegisterStringServiceExtension(name string, getter AsyncValueGetter[string], setter AsyncValueSetter[string]) {
	superOf[interface {
		RegisterStringServiceExtension(string, AsyncValueGetter[string], AsyncValueSetter[string])
	}](v).RegisterStringServiceExtension(name, getter, setter)
}

func (v bindingSuperView) RegisterServiceExtension(name string, callback ServiceExtensionCallback) {
	superOf[interface {
		RegisterServiceExtension(string, ServiceExtensionCallback)
	}](v).RegisterServiceExtension(name, callback)
}

func (v bindingSuperView) PostEvent(eventKind string, eventData map[string]any) {
	superOf[interface{ PostEvent(string, map[string]any) }](v).PostEvent(eventKind, eventData)
}

func (v bindingSuperView) String() string { return superOf[interface{ String() string }](v).String() }

func (v bindingSuperView) bindingBase() *BindingBaseBase { return v.binding.bindingBase() }
