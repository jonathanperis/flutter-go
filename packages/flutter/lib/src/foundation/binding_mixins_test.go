// Tests of binding_mixins.go: a binding combining mixins runs their
// overrides in Dart's linearization order (each mixin's super is the mixin
// before it in `with` order, then BindingBase), for BindingBase's methods
// and for a mixin's own methods overridden by a later mixin. The
// difftest src/difftest/binding_composition_test.go checks the real
// bindings against Dart.
package foundation

import (
	"strings"
	"testing"

	"github.com/jonathanperis/flutter-go/src/async"
)

var mixinLog []string

func logm(s string) { mixinLog = append(mixinLog, s) }

// greeter is a method mixin A declares and mixin C overrides (Dart: `mixin
// C on BindingBase, A`).
type greeter interface{ Greet(name string) string }

type mixinA struct{ self BindingBase }

func (m *mixinA) InitInstances() {
	logm("A.initInstances before super")
	BindingBaseSuper(m.self, m).InitInstances()
	logm("A.initInstances after super")
}
func (m *mixinA) Unlocked() {
	BindingBaseSuper(m.self, m).Unlocked()
	logm("A.unlocked")
}
func (m *mixinA) Greet(name string) string { return "A(" + name + ")" }

type mixinB struct{ self BindingBase }

func (m *mixinB) InitInstances() {
	logm("B.initInstances before super")
	BindingBaseSuper(m.self, m).InitInstances()
	logm("B.initInstances after super")
}
func (m *mixinB) InitServiceExtensions() {
	BindingBaseSuper(m.self, m).InitServiceExtensions()
	logm("B.initServiceExtensions")
	m.self.RegisterServiceExtension("b", nil)
}

type mixinC struct{ self BindingBase }

func (m *mixinC) InitInstances() {
	BindingBaseSuper(m.self, m).InitInstances()
	logm("C.initInstances")
}
func (m *mixinC) Unlocked() {
	logm("C.unlocked before super")
	BindingBaseSuper(m.self, m).Unlocked()
}
func (m *mixinC) Greet(name string) string {
	return "C[" + BindingSuper[greeter](m.self, m).Greet(name) + "]"
}

// abcBinding is `BindingBase with A, B, C`.
type abcBinding struct {
	BindingBaseBase
	mixinA
	mixinB
	mixinC
	registered []string
}

func (b *abcBinding) InitInstances()         { b.mixinC.InitInstances() }
func (b *abcBinding) InitServiceExtensions() { b.mixinB.InitServiceExtensions() }
func (b *abcBinding) Unlocked()              { b.mixinC.Unlocked() }
func (b *abcBinding) Greet(name string) string {
	return "class{" + BindingSuper[greeter](b, nil).Greet(name) + "}"
}
func (b *abcBinding) RegisterServiceExtension(name string, callback ServiceExtensionCallback) {
	b.registered = append(b.registered, name)
}

func TestBindingMixinsLinearization(t *testing.T) {
	resetBindingStatics(t)
	mixinLog = nil
	b := &abcBinding{}
	b.mixinA.self, b.mixinB.self, b.mixinC.self = b, b, b
	b.Construct(b, &b.mixinA, &b.mixinB, &b.mixinC)

	want := []string{
		"B.initInstances before super",
		"A.initInstances before super",
		"A.initInstances after super",
		"B.initInstances after super",
		"C.initInstances",
		"B.initServiceExtensions",
	}
	if got := strings.Join(mixinLog, "\n"); got != strings.Join(want, "\n") {
		t.Errorf("construction order:\n%s\nwant:\n%s", got, strings.Join(want, "\n"))
	}
	// BindingBase's own extensions come first (super), then B's.
	if n := len(b.registered); n == 0 || b.registered[n-1] != "b" {
		t.Errorf("registered %v, want B's extension last", b.registered)
	}
	if bindingDebugInitializedType == nil || bindingDebugInitializedType.Name() != "abcBinding" {
		t.Errorf("initialized type %v", bindingDebugInitializedType)
	}

	// A mixin's own method overridden by a later mixin, and by the class.
	if got := b.Greet("x"); got != "class{C[A(x)]}" {
		t.Errorf("Greet = %q", got)
	}
	if got := BindingSuper[greeter](b, &b.mixinB).Greet("y"); got != "A(y)" {
		t.Errorf("super of B = %q", got)
	}

	// unlocked: C, then A, then BindingBase (which asserts it is unlocked).
	mixinLog = nil
	loop := async.NewLoop()
	defer async.SetCurrent(loop)()
	completer := async.NewCompleter[async.Void]()
	b.LockEvents(func() async.Future[async.Void] { return completer.Future() })
	if !b.Locked() {
		t.Fatal("not locked")
	}
	completer.Complete(async.Void{})
	loop.FlushMicrotasks()
	if got := strings.Join(mixinLog, ","); got != "C.unlocked before super,A.unlocked" {
		t.Errorf("unlocked order: %s", got)
	}
}

func TestBindingSuperErrors(t *testing.T) {
	resetBindingStatics(t)
	b := &abcBinding{}
	b.mixinA.self, b.mixinB.self, b.mixinC.self = b, b, b
	b.Construct(b, &b.mixinA, &b.mixinB, &b.mixinC)

	expectPanic := func(name, substr string, f func()) {
		t.Helper()
		defer func() {
			r := recover()
			if r == nil || !strings.Contains(r.(string), substr) {
				t.Errorf("%s: panic %v, want %q", name, r, substr)
			}
		}()
		f()
	}
	// Nothing below A implements Greet (Dart: a compile-time error).
	expectPanic("no super", "no super implementation", func() { BindingSuper[greeter](b, &b.mixinA) })
	// Not one of the binding's mixins.
	expectPanic("foreign mixin", "is not one of the binding's mixins", func() { BindingSuper[greeter](b, &mixinA{}) })
	// Construct validates its arguments.
	expectPanic("nil mixin", "not a non-nil pointer", func() { (&BindingBaseBase{}).setMixins([]any{nil}) })
	expectPanic("value mixin", "not a non-nil pointer", func() { (&BindingBaseBase{}).setMixins([]any{mixinA{}}) })
	expectPanic("twice", "twice", func() { (&BindingBaseBase{}).setMixins([]any{&b.mixinA, &b.mixinA}) })
	other := &abcBinding{}
	expectPanic("wrong base", "must be called on the BindingBaseBase embedded in self", func() {
		(&BindingBaseBase{}).Construct(other)
	})
	if got := b.BindingMixins(); len(got) != 3 || got[0] != any(&b.mixinA) || got[2] != any(&b.mixinC) {
		t.Errorf("BindingMixins = %v", got)
	}
}
