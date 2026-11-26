package difftest

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"regexp"
	"slices"
	"testing"

	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	"github.com/jonathanperis/flutter-go/src/dart"
	"github.com/jonathanperis/flutter-go/src/pub/listen"
)

// listenDefaultOnError is package:listen's own Listenable.onError, read
// before any test can construct a binding (which replaces it).
var listenDefaultOnError = listen.ListenableOnError

// TestNotifier is the Dart test's `class TestNotifier extends ChangeNotifier`.
type TestNotifier struct{ foundation.ChangeNotifierMixin }

func newTestNotifier() *TestNotifier {
	n := &TestNotifier{}
	n.SetSelf(n)
	return n
}

func (n *TestNotifier) notify()        { n.NotifyListeners() }
func (n *TestNotifier) listened() bool { return n.HasListeners() }

// LoggingValueNotifier overrides notifyListeners.
type LoggingValueNotifier struct {
	foundation.ValueNotifier[int]
	log func(string)
}

func newLoggingValueNotifier(value int, log func(string)) *LoggingValueNotifier {
	n := &LoggingValueNotifier{log: log}
	n.Construct(n, value)
	return n
}

func (n *LoggingValueNotifier) NotifyListeners() {
	n.log(fmt.Sprintf("override notifyListeners(%d)", n.Value()))
	n.ValueNotifier.NotifyListeners()
}

// P's == compares x only.
type P struct{ x, y int }

func (p P) Equal(other P) bool { return p.x == other.x }
func (p P) String() string     { return fmt.Sprintf("P(%d, %d)", p.x, p.y) }

type listenException struct{ n int }

func (e listenException) String() string { return fmt.Sprintf("MyException(%d)", e.n) }

type listenHandler struct {
	name string
	log  func(string)
}

func (h *listenHandler) handle() { h.log("handle " + h.name) }

type ListenBinding struct{ foundation.BindingBaseBase }

func (b *ListenBinding) RegisterServiceExtension(string, foundation.ServiceExtensionCallback) {}

func listenDescribe(e any) string {
	switch e := e.(type) {
	case *foundation.FlutterError:
		return "FlutterError: " + e.ToStringDeep()
	case dart.AssertionFailure:
		return "AssertionError: " + fmt.Sprint(e.AssertionMessage())
	case dart.ArgumentError:
		return "ArgumentError: " + e.Error()
	case dart.StateError:
		return "StateError: " + e.Error()
	case string:
		return "String: " + e
	}
	return "Object: " + fmt.Sprint(e)
}

// Same steps as flutter/test/listen_test.dart.
func TestListenMatchFlutter(t *testing.T) {
	data, err := os.ReadFile("flutter/build/listen.json")
	if err != nil {
		t.Skipf("no reference (%v); run scripts/difftest.sh", err)
	}
	var ref struct{ Log []string }
	if err := json.Unmarshal(data, &ref); err != nil {
		t.Fatal(err)
	}
	savedOnError := listen.ListenableOnError
	t.Cleanup(func() { listen.ListenableOnError = savedOnError })
	hash := regexp.MustCompile(`#[0-9a-f]{5}`)
	var got []string
	log := func(s string) { got = append(got, hash.ReplaceAllString(s, "#HASH")) }
	catch := func(f func()) {
		defer func() {
			if r := recover(); r != nil {
				log("caught " + listenDescribe(r))
			}
		}()
		f()
	}
	loggingOnError := func(error any, stack dart.StackTrace, context listen.ErrorContext) {
		log(fmt.Sprintf("onError %s %s stack=%v", context, listenDescribe(error), stack != nil))
	}
	l := func(name string) func() { return func() { log(name) } }

	// basic
	log("--- basic")
	listen.ListenableOnError = loggingOnError
	basic := newTestNotifier()
	a, b, c := l("a"), l("b"), l("c")
	log(fmt.Sprintf("hasListeners=%v", basic.listened()))
	basic.AddListener(a)
	basic.AddListener(a)
	basic.AddListener(b)
	basic.notify()
	basic.RemoveListener(a)
	basic.notify()
	basic.RemoveListener(c)
	basic.notify()
	basic.RemoveListener(a)
	log(fmt.Sprintf("hasListeners=%v", basic.listened()))
	basic.RemoveListener(b)
	log(fmt.Sprintf("hasListeners=%v", basic.listened()))
	basic.notify()
	for i := 0; i < 9; i++ {
		if i%2 == 0 {
			basic.AddListener(a)
		} else {
			basic.AddListener(b)
		}
	}
	basic.RemoveListener(b)
	basic.RemoveListener(a)
	basic.RemoveListener(b)
	basic.notify()

	// mutating
	log("--- mutating")
	mut := newTestNotifier()
	d := l("d")
	var mutator func()
	mutator = func() {
		log("mutator")
		mut.RemoveListener(a)
		mut.RemoveListener(c)
		mut.AddListener(d)
		mut.AddListener(mutator)
		mut.RemoveListener(mutator)
	}
	mut.AddListener(a)
	mut.AddListener(mutator)
	mut.AddListener(c)
	mut.AddListener(b)
	for i := 0; i < 3; i++ {
		log(fmt.Sprintf("notify %d", i))
		mut.notify()
	}

	// duplicates
	log("--- duplicates")
	dup := newTestNotifier()
	removeA := func() {
		log("removeA")
		dup.RemoveListener(a)
	}
	dup.AddListener(a)
	dup.AddListener(removeA)
	dup.AddListener(a)
	dup.notify()
	dup.notify()
	dup2 := newTestNotifier()
	removeB := func() {
		log("removeB")
		dup2.RemoveListener(b)
	}
	dup2.AddListener(removeB)
	dup2.AddListener(b)
	dup2.AddListener(b)
	dup2.notify()
	dup2.notify()
	dup2.RemoveListener(removeB)
	dup2.notify()

	// self removal
	log("--- self removal")
	self := newTestNotifier()
	var selfRemoving func()
	selfRemoving = func() {
		log("selfRemoving")
		self.RemoveListener(selfRemoving)
	}
	self.AddListener(a)
	self.AddListener(selfRemoving)
	self.AddListener(b)
	self.notify()
	self.notify()
	clearAll := func() {
		log("clearAll")
		self.RemoveListener(a)
		self.RemoveListener(b)
	}
	self.AddListener(clearAll)
	self.notify()
	log(fmt.Sprintf("hasListeners=%v", self.listened()))
	self.RemoveListener(clearAll)
	log(fmt.Sprintf("hasListeners=%v", self.listened()))

	// nested
	log("--- nested")
	nest := newTestNotifier()
	depth := 0
	e := l("e")
	var nester func()
	nester = func() {
		log(fmt.Sprintf("nester depth=%d", depth))
		if depth < 2 {
			depth++
			nest.RemoveListener(a)
			nest.AddListener(e)
			nest.notify()
			depth--
			log(fmt.Sprintf("back to depth=%d", depth))
		} else {
			nest.RemoveListener(b)
			nest.RemoveListener(nester)
		}
	}
	nest.AddListener(a)
	nest.AddListener(nester)
	nest.AddListener(b)
	nest.AddListener(c)
	nest.notify()
	log("after nested")
	nest.notify()

	// errors
	log("--- errors")
	errN := newTestNotifier()
	errN.AddListener(func() { panic("boom") })
	errN.AddListener(a)
	errN.AddListener(func() { panic(dart.ArgumentError{}) })
	errN.AddListener(func() { panic(dart.StateError{Message: "state"}) })
	errN.AddListener(func() { panic(listenException{1}) })
	var throwsAndRemoves func()
	throwsAndRemoves = func() {
		errN.RemoveListener(throwsAndRemoves)
		panic(dart.ArgumentError{Message: "removed"})
	}
	errN.AddListener(throwsAndRemoves)
	errN.AddListener(b)
	errN.notify()
	errN.notify()

	listen.ListenableOnError = func(error any, _ dart.StackTrace, _ listen.ErrorContext) {
		log("rethrowing onError " + listenDescribe(error))
		panic(listenException{2})
	}
	escape := newTestNotifier()
	escape.AddListener(func() { panic("first") })
	escape.AddListener(a)
	catch(escape.notify)
	listen.ListenableOnError = loggingOnError

	// dispose
	log("--- dispose")
	disp := newTestNotifier()
	finished := false
	disp.AddListener(func() {
		disp.Dispose()
		finished = true
	})
	disp.AddListener(a)
	disp.notify()
	log(fmt.Sprintf("finished=%v hasListeners=%v", finished, disp.listened()))
	disp2 := newTestNotifier()
	disp2.AddListener(a)
	disp2.Dispose()
	log(fmt.Sprintf("hasListeners=%v", disp2.listened()))
	disp2.RemoveListener(a)
	disp2.AddListener(b)
	log(fmt.Sprintf("hasListeners=%v", disp2.listened()))
	disp2.notify()
	log(fmt.Sprintf("debugAssertNotDisposed=%v", foundation.ChangeNotifierDebugAssertNotDisposed(disp2)))
	disp2.Dispose()
	log(fmt.Sprintf("hasListeners=%v", disp2.listened()))
	plain := foundation.NewChangeNotifier()
	plain.Dispose()
	plain.NotifyListeners()

	// default onError
	log("--- default onError")
	listen.ListenableOnError = listenDefaultOnError
	def := newTestNotifier()
	def.AddListener(func() { panic(dart.StateError{Message: "default"}) })
	def.AddListener(a)
	catch(def.notify)
	def.RemoveListener(a)
	log(fmt.Sprintf("hasListeners=%v", def.listened()))
	removeFirst := func() { log("never") }
	def.AddListener(removeFirst)
	catch(def.notify)
	catch(def.Dispose)
	disposed := newTestNotifier()
	disposed.Dispose()
	catch(disposed.notify)
	listen.ListenableOnError = loggingOnError

	// ValueNotifier
	log("--- ValueNotifier")
	vd := foundation.NewValueNotifier(2.0)
	vd.AddListener(func() { log("vd " + dart.DoubleToString(vd.Value())) })
	for _, v := range []float64{3.0, 3.0, math.NaN(), math.NaN(), math.Copysign(0, -1), 0.0, math.Inf(1), 1e21, 0.1} {
		vd.SetValue(v)
	}
	log(vd.String())
	vs := foundation.NewValueNotifier("a")
	vs.AddListener(func() { log("vs " + vs.Value()) })
	vs.SetValue("a")
	vs.SetValue("b")
	vs.SetValue("b")
	log(vs.String())
	list := []int{1, 2}
	vl := foundation.NewValueNotifier(list)
	vl.AddListener(func() { log(fmt.Sprintf("vl [%d, %d]", vl.Value()[0], vl.Value()[1])) })
	vl.SetValue(list)
	vl.SetValue([]int{1, 2})
	log(vl.String())
	vo := foundation.NewValueNotifier[any](nil)
	vo.AddListener(func() {
		v := vo.Value()
		if v == nil {
			v = "null"
		}
		log(fmt.Sprintf("vo %v", v))
	})
	vo.SetValue(nil)
	vo.SetValue("x")
	vo.SetValue("x")
	vo.SetValue(1)
	vo.SetValue(1)
	vo.SetValue(nil)
	vp := foundation.NewValueNotifier(P{1, 2})
	vp.AddListener(func() { log("vp " + vp.Value().String()) })
	vp.SetValue(P{1, 3})
	vp.SetValue(P{2, 3})
	log(vp.String())
	vb := foundation.NewValueNotifier(false)
	log(vb.String())
	lv := newLoggingValueNotifier(0, log)
	lv.AddListener(func() { log(fmt.Sprintf("lv listener %d", lv.Value())) })
	lv.SetValue(0)
	lv.SetValue(5)
	log(lv.String())
	log(foundation.NewChangeNotifier().String())
	log(newTestNotifier().String())

	// merge
	log("--- merge")
	s1 := newTestNotifier()
	s2 := newTestNotifier()
	vm := foundation.NewValueNotifier(0)
	merged := foundation.ListenableMerge(slices.Values([]foundation.Listenable{s1, nil, s2, vm, s1}))
	log(fmt.Sprint(merged))
	merged.AddListener(a)
	s1.notify()
	s2.notify()
	vm.SetValue(1)
	log(fmt.Sprint(merged))
	merged.RemoveListener(a)
	log(fmt.Sprintf("s1=%v s2=%v", s1.listened(), s2.listened()))
	s1.notify()
	merged.AddListener(a)
	s1.RemoveListener(a)
	s1.notify()
	log(fmt.Sprint(foundation.ListenableMerge(slices.Values([]foundation.Listenable{}))) + " " +
		fmt.Sprint(foundation.ListenableMerge(slices.Values([]foundation.Listenable{nil}))))
	log(fmt.Sprint(foundation.ListenableMerge(slices.Values([]foundation.Listenable{
		foundation.ListenableMerge(slices.Values([]foundation.Listenable{s1})), foundation.NewChangeNotifier(),
	}))))

	// tear-offs
	log("--- tear-offs")
	h1 := &listenHandler{"h1", log}
	h2 := &listenHandler{"h2", log}
	tear := newTestNotifier()
	tear.AddListener(h1.handle)
	tear.AddListener(h2.handle)
	tear.AddListener(h1.handle)
	tear.notify()
	tear.RemoveListener(h1.handle)
	tear.notify()
	tear.RemoveListener(h2.handle)
	tear.RemoveListener(h1.handle)
	log(fmt.Sprintf("hasListeners=%v", tear.listened()))

	// binding
	log("--- binding")
	if foundation.BindingBaseDebugBindingType() == nil {
		binding := &ListenBinding{}
		binding.Construct(binding)
	} else {
		listen.ListenableOnError = savedOnError // the existing binding's
	}
	originalOnError := foundation.FlutterErrorOnError
	foundation.FlutterErrorOnError = func(details *foundation.FlutterErrorDetails) {
		log(fmt.Sprintf("reported %s library=%s context=%s stack=%v",
			listenDescribe(details.Exception), *details.Library, details.Context, details.Stack != nil))
	}
	bind := newTestNotifier()
	bind.AddListener(func() { panic(dart.StateError{Message: "in listener"}) })
	bind.AddListener(func() { panic("plain") })
	bind.AddListener(a)
	bind.notify()
	bind.AddListener(func() { bind.Dispose() })
	bind.notify()
	bind.Dispose()
	for _, use := range []func(){
		func() { bind.AddListener(a) },
		func() { bind.notify() },
		func() { bind.Dispose() },
		func() { foundation.ChangeNotifierDebugAssertNotDisposed(bind) },
	} {
		catch(func() {
			use()
			log("no error")
		})
	}
	vdisp := foundation.NewValueNotifier(3)
	vdisp.Dispose()
	catch(func() { vdisp.SetValue(4) })
	log(fmt.Sprintf("value=%d", vdisp.Value()))
	foundation.FlutterErrorOnError = originalOnError

	// memory allocations
	log("--- memory allocations")
	ma := foundation.FlutterMemoryAllocationsInstance
	maListener := func(event foundation.ObjectEvent) {
		var fields map[string]any
		for _, f := range event.ToMap() {
			fields = f
		}
		kind := "ObjectDisposed"
		if _, ok := event.(*foundation.ObjectCreated); ok {
			kind = "ObjectCreated"
		}
		var objectType string
		switch event.Object().(type) {
		case *TestNotifier:
			objectType = "TestNotifier"
		case *foundation.ValueNotifier[int]:
			objectType = "ValueNotifier<int>"
		case *LoggingValueNotifier:
			objectType = "LoggingValueNotifier"
		case *foundation.ChangeNotifierMixin:
			objectType = "ChangeNotifier"
		default:
			objectType = fmt.Sprintf("%T", event.Object())
		}
		str := func(v any) string {
			if v == nil {
				return "null"
			}
			return fmt.Sprint(v)
		}
		log(fmt.Sprintf("%s %s %s %s %s", kind, objectType, str(fields["libraryName"]), str(fields["className"]), str(fields["eventType"])))
	}
	ma.AddListener(maListener)
	m1 := newTestNotifier()
	log("m1 created")
	m1.AddListener(a)
	m1.AddListener(b)
	m2 := foundation.NewValueNotifier(1)
	m3 := foundation.NewChangeNotifier()
	m4 := newTestNotifier()
	m5 := newLoggingValueNotifier(2, log)
	log("all created")
	m1.Dispose()
	m2.Dispose()
	m3.Dispose()
	m4.Dispose()
	m5.Dispose()
	ma.RemoveListener(maListener)

	if !slices.Equal(got, ref.Log) {
		for i := range max(len(got), len(ref.Log)) {
			var g, r string
			if i < len(got) {
				g = got[i]
			}
			if i < len(ref.Log) {
				r = ref.Log[i]
			}
			if g != r {
				t.Fatalf("first difference at entry %d:\nGo:   %q\nDart: %q", i, g, r)
			}
		}
	}
	t.Logf("%d log entries identical", len(got))
}
