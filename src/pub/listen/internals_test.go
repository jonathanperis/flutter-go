// Own tests: the listener list's length and layout (Dart's private
// _listeners, checked against the lengths Dart's algorithm gives), listener
// identity, the creation/disposal hooks, ValueNotifier's equality and
// toString, and the Go-specific checks (nil listener, unbound self).
package listen

import (
	"fmt"
	"math"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/jonathanperis/flutter-go/src/dart"
)

// layout describes n.listeners: a name per entry ("-" for nil), and count.
func layout(n *ChangeNotifierMixin, names map[string]VoidCallback) string {
	var parts []string
	for _, l := range n.listeners {
		name := "-"
		if l != nil {
			name = "?"
			for k, v := range names {
				if dart.FunctionsEqual(v, l) {
					name = k
				}
			}
		}
		parts = append(parts, name)
	}
	return fmt.Sprintf("[%s] count=%d", strings.Join(parts, " "), n.count)
}

func expectLayout(t *testing.T, n *ChangeNotifierMixin, names map[string]VoidCallback, want string) {
	t.Helper()
	if got := layout(n, names); got != want {
		t.Fatalf("layout = %s, want %s", got, want)
	}
}

func namedListeners(log *[]string, names ...string) map[string]VoidCallback {
	m := map[string]VoidCallback{}
	for _, name := range names {
		m[name] = func() { *log = append(*log, name) }
	}
	return m
}

func TestListenerListGrowsAndShrinksAsDart(t *testing.T) {
	setUp(t)
	var log []string
	l := namedListeners(&log, "a", "b", "c", "d", "e")
	n := NewTestNotifier()
	m := &n.ChangeNotifierMixin
	expectLayout(t, m, l, "[] count=0")
	n.AddListener(l["a"])
	expectLayout(t, m, l, "[a] count=1")
	n.AddListener(l["b"])
	expectLayout(t, m, l, "[a b] count=2")
	n.AddListener(l["c"])
	expectLayout(t, m, l, "[a b c -] count=3")
	n.AddListener(l["d"])
	n.AddListener(l["e"])
	expectLayout(t, m, l, "[a b c d e - - -] count=5")

	// count 4, 4*2 <= 8: reallocated to exactly count.
	n.RemoveListener(l["c"])
	expectLayout(t, m, l, "[a b d e] count=4")
	// count 3, 6 > 4: shifted in place.
	n.RemoveListener(l["a"])
	expectLayout(t, m, l, "[b d e -] count=3")
	// count 2, 4 <= 4: reallocated.
	n.RemoveListener(l["e"])
	expectLayout(t, m, l, "[b d] count=2")
	n.RemoveListener(l["b"])
	expectLayout(t, m, l, "[d] count=1")
	n.RemoveListener(l["d"])
	expectLayout(t, m, l, "[] count=0")
	n.RemoveListener(l["d"]) // not registered: ignored
	expectLayout(t, m, l, "[] count=0")

	// Removing the first of two registrations.
	n.AddListener(l["a"])
	n.AddListener(l["b"])
	n.AddListener(l["a"])
	expectLayout(t, m, l, "[a b a -] count=3")
	n.RemoveListener(l["a"])
	expectLayout(t, m, l, "[b a] count=2")
	n.NotifyListeners()
	if !slices.Equal(log, []string{"b", "a"}) {
		t.Fatal(log)
	}
}

func TestCompactionAfterNotification(t *testing.T) {
	setUp(t)
	var log []string
	l := namedListeners(&log, "a", "b", "c", "d", "e", "f", "g")
	n := NewTestNotifier()
	m := &n.ChangeNotifierMixin
	wantCount := 8
	remover := func(names ...string) VoidCallback {
		return func() {
			log = append(log, "r")
			for _, name := range names {
				n.RemoveListener(l[name])
			}
			if m.notificationCallStackDepth != 1 || m.reentrantlyRemovedListeners != len(names) || m.count != wantCount {
				t.Errorf("during: depth=%d removed=%d count=%d", m.notificationCallStackDepth, m.reentrantlyRemovedListeners, m.count)
			}
		}
	}

	// 3 of 8 removed: 5*2 > 8, so nil entries are swapped to the end.
	l["r"] = remover("a", "c", "f")
	for _, name := range []string{"a", "b", "r", "c", "d", "e", "f", "g"} {
		n.AddListener(l[name])
	}
	expectLayout(t, m, l, "[a b r c d e f g] count=8")
	n.NotifyListeners()
	if !slices.Equal(log, []string{"a", "b", "r", "d", "e", "g"}) {
		t.Fatal(log)
	}
	log = nil
	expectLayout(t, m, l, "[b r d e g - - -] count=5")
	if m.reentrantlyRemovedListeners != 0 || m.notificationCallStackDepth != 0 {
		t.Fatal(m.reentrantlyRemovedListeners, m.notificationCallStackDepth)
	}

	// Now 4 of 8 removed: 4*2 <= 8, so the list is reallocated.
	n.RemoveListener(l["r"])
	n.RemoveListener(l["b"])
	expectLayout(t, m, l, "[d e g -] count=3")
	n.AddListener(l["a"])
	n.AddListener(l["b"])
	n.AddListener(l["c"])
	n.AddListener(l["f"])
	l["r"] = remover("d", "g", "b")
	n.AddListener(l["r"])
	expectLayout(t, m, l, "[d e g a b c f r] count=8")
	n.NotifyListeners()
	if !slices.Equal(log, []string{"d", "e", "g", "a", "b", "c", "f", "r"}) {
		t.Fatal(log)
	}
	log = nil
	expectLayout(t, m, l, "[e a c f r - - -] count=5")
	n.RemoveListener(l["r"])
	n.RemoveListener(l["a"])
	expectLayout(t, m, l, "[e c f -] count=3")
	// Removing 2 of 4 entries where count is 4: 2*2 <= 4.
	n.AddListener(l["g"])
	l["r"] = remover("c")
	n.AddListener(l["r"])
	n.AddListener(l["b"])
	n.AddListener(l["d"])
	expectLayout(t, m, l, "[e c f g r b d -] count=7")
	wantCount = 7
	n.NotifyListeners()
	// count 7 - 1 = 6, 12 > 8: swapped in place.
	expectLayout(t, m, l, "[e f g r b d - -] count=6")
}

func TestCompactionWaitsForTheOutermostNotification(t *testing.T) {
	setUp(t)
	var log []string
	l := namedListeners(&log, "a", "b", "c")
	n := NewTestNotifier()
	m := &n.ChangeNotifierMixin
	depth := 0
	l["nest"] = func() {
		log = append(log, fmt.Sprintf("nest%d", depth))
		if depth == 0 {
			depth++
			n.RemoveListener(l["a"])
			n.NotifyListeners() // nested: a is skipped, b removed, nothing compacted
			expectLayout(t, m, l, "[- nest - c] count=4")
			if m.reentrantlyRemovedListeners != 2 || m.notificationCallStackDepth != 1 {
				t.Errorf("removed=%d depth=%d", m.reentrantlyRemovedListeners, m.notificationCallStackDepth)
			}
		} else {
			n.RemoveListener(l["b"])
		}
	}
	for _, name := range []string{"a", "nest", "b", "c"} {
		n.AddListener(l[name])
	}
	n.NotifyListeners()
	if !slices.Equal(log, []string{"a", "nest0", "nest1", "c", "c"}) {
		t.Fatal(log)
	}
	// 2 of 4: 2*2 <= 4, reallocated.
	expectLayout(t, m, l, "[nest c] count=2")
}

type handler struct {
	name string
	log  *[]string
}

func (h *handler) handle() { *h.log = append(*h.log, h.name) }

type valueHandler struct{ log *[]string }

func (h valueHandler) handle() { *h.log = append(*h.log, "value") }

func TestMethodValuesRemoveAsDartTearOffs(t *testing.T) {
	setUp(t)
	var log []string
	h1 := &handler{"h1", &log}
	h2 := &handler{"h2", &log}
	n := NewTestNotifier()
	n.AddListener(h1.handle)
	n.AddListener(h2.handle)
	n.AddListener(h1.handle)
	n.NotifyListeners()
	expectLog(t, &log, "h1", "h2", "h1")
	n.RemoveListener(h1.handle) // a new method value of the same receiver
	n.NotifyListeners()
	expectLog(t, &log, "h2", "h1")
	n.RemoveListener(h1.handle)
	n.RemoveListener(h2.handle)
	if n.HasListeners() {
		t.Fatal("not removed")
	}

	// A value receiver is copied: only the same closure value matches.
	v := valueHandler{&log}
	n.AddListener(v.handle)
	n.RemoveListener(v.handle)
	if !n.HasListeners() {
		t.Fatal("a value-receiver method value matched another")
	}
	cb := v.handle
	n.AddListener(cb)
	n.RemoveListener(cb)
	n.NotifyListeners()
	expectLog(t, &log, "value")
}

func TestAddNilListenerPanics(t *testing.T) {
	n := NewTestNotifier()
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("no panic")
		}
	}()
	n.AddListener(nil)
}

// RemoveListener(nil) (Dart can't pass null) is ignored, also during a
// notification, where removed entries are nil.
func TestRemoveNilListenerIsIgnored(t *testing.T) {
	setUp(t)
	var log []string
	l := namedListeners(&log, "a", "b")
	n := NewTestNotifier()
	l["r"] = func() {
		n.RemoveListener(l["a"])
		n.RemoveListener(nil)
		n.RemoveListener(nil)
	}
	n.AddListener(l["a"])
	n.AddListener(l["r"])
	n.AddListener(l["b"])
	n.RemoveListener(nil)
	n.NotifyListeners()
	expectLayout(t, &n.ChangeNotifierMixin, l, "[r b] count=2")
	expectLog(t, &log, "a", "b")
}

func TestUnboundMixinPanics(t *testing.T) {
	n := &TestNotifier{}
	defer func() {
		if r := recover(); r == nil || !strings.Contains(fmt.Sprint(r), "SetSelf") {
			t.Fatalf("recovered %v", r)
		}
	}()
	n.AddListener(func() {})
}

func TestNewChangeNotifier(t *testing.T) {
	s := setUp(t)
	n := NewChangeNotifier()
	if got := toString(n); got != "Instance of 'ChangeNotifier'" {
		t.Fatal(got)
	}
	calls := 0
	n.AddListener(func() { calls++ })
	n.NotifyListeners()
	n.Dispose()
	n.NotifyListeners()
	if calls != 1 || !strings.Contains(errorString(s.lastError), "A ChangeNotifier was used after being disposed.\nOnce you have called dispose() on a ChangeNotifier, it can no longer be used.") {
		t.Fatal(calls, s.lastError)
	}
	s.lastError = nil
}

func TestDisposeDuringNotificationMessage(t *testing.T) {
	s := setUp(t)
	n := NewTestNotifier()
	n.AddListener(func() { n.Dispose() })
	n.NotifyListeners()
	e, ok := s.lastError.(*dart.AssertionError)
	if !ok || e.AssertionMessage() != `The "dispose()" method on Instance of 'TestNotifier' was called during the call to "notifyListeners()". This is likely to cause errors since it modifies the list of listeners while the list is being used.` {
		t.Fatalf("%#v", s.lastError)
	}
	s.lastError = nil
	if !n.HasListeners() {
		t.Fatal("the failed dispose cleared the listeners")
	}
}

func TestDefaultOnErrorAbandonsTheNotification(t *testing.T) {
	// No setUp: the default onError rethrows out of NotifyListeners, and
	// (no finally in Dart) the call depth stays incremented.
	var log []string
	n := NewTestNotifier()
	ok := func() { log = append(log, "ok") }
	n.AddListener(func() { panic("boom") })
	n.AddListener(ok)
	func() {
		defer func() {
			if r := recover(); r != "boom" {
				t.Fatalf("recovered %v", r)
			}
		}()
		n.NotifyListeners()
	}()
	if len(log) != 0 || n.notificationCallStackDepth != 1 {
		t.Fatal(log, n.notificationCallStackDepth)
	}
	n.RemoveListener(ok) // nulled, not removed
	if !n.HasListeners() || n.count != 2 || n.reentrantlyRemovedListeners != 1 {
		t.Fatal(n.count, n.reentrantlyRemovedListeners)
	}
}

func TestCreationAndDisposalDispatch(t *testing.T) {
	setUp(t)
	var events []string
	created, disposed := ListenableDebugMaybeDispatchCreated, ListenableDebugMaybeDispatchDisposed
	t.Cleanup(func() { ListenableDebugMaybeDispatchCreated, ListenableDebugMaybeDispatchDisposed = created, disposed })
	ListenableDebugMaybeDispatchCreated = func(className string, object any) {
		events = append(events, "created "+className+" "+runtimeTypeName(object))
	}
	ListenableDebugMaybeDispatchDisposed = func(object any) {
		events = append(events, "disposed "+runtimeTypeName(object))
	}

	n := NewTestNotifier() // no constructor dispatch
	n.Dispose()            // never dispatched as created: no disposal event
	if len(events) != 0 {
		t.Fatal(events)
	}

	n = NewTestNotifier()
	n.AddListener(func() {}) // first addListener dispatches the creation
	n.AddListener(func() {})
	n.Dispose()
	c := NewCounter() // dispatches in the constructor
	c.AddListener(func() {})
	c.Dispose()
	v := NewHasListenersTester("x")
	v.Dispose()
	want := []string{
		"created ChangeNotifier TestNotifier", "disposed TestNotifier",
		"created ChangeNotifier Counter", "disposed Counter",
		"created ChangeNotifier HasListenersTester<String>", "disposed HasListenersTester<String>",
	}
	if !slices.Equal(events, want) {
		t.Fatalf("events = %q", events)
	}
	if !debugMode {
		t.Skip("release build")
	}
}

type point struct{ x, y float64 }

func (p point) Equal(o point) bool { return p.x == o.x } // y ignored, to show Equal is used

func TestValueNotifierEquality(t *testing.T) {
	setUp(t)
	count := func(n interface{ AddListener(VoidCallback) }) *int {
		c := 0
		n.AddListener(func() { c++ })
		return &c
	}

	p := NewValueNotifier(point{1, 2})
	pc := count(p)
	p.SetValue(point{1, 3}) // Equal: no notification
	p.SetValue(point{2, 3})
	if *pc != 1 || p.Value() != (point{2, 3}) {
		t.Fatal(*pc, p.Value())
	}

	nan := NewValueNotifier(math.NaN())
	nc := count(nan)
	nan.SetValue(math.NaN()) // NaN != NaN in Dart too
	nan.SetValue(0.0)
	nan.SetValue(math.Copysign(0, -1)) // -0.0 == 0.0
	if *nc != 2 {
		t.Fatal(*nc)
	}

	s := []int{1}
	sl := NewValueNotifier(s)
	sc := count(sl)
	sl.SetValue(s)        // the same list
	sl.SetValue([]int{1}) // an equal but different list: Dart's List == is identity
	if *sc != 1 {
		t.Fatal(*sc)
	}

	var a any = []int{1}
	an := NewValueNotifier[any](nil)
	ac := count(an)
	an.SetValue(nil)
	an.SetValue(a)
	an.SetValue(a)
	an.SetValue("x")
	an.SetValue("x")
	an.SetValue(1)
	an.SetValue(1.0) // a different type: Dart's 1 == 1.0 is true, Go's typed values differ
	if *ac != 4 {
		t.Fatal(*ac)
	}

	pp := NewValueNotifier[*int](nil)
	pc2 := count(pp)
	one, other := 1, 1
	pp.SetValue(&one)
	pp.SetValue(&one)
	pp.SetValue(&other) // pointers compare by identity
	pp.SetValue(nil)
	if *pc2 != 3 {
		t.Fatal(*pc2)
	}
}

type fruit int

func (f fruit) String() string { return [...]string{"apple", "pear"}[f] }

func TestValueNotifierToString(t *testing.T) {
	setUp(t)
	hash := regexp.MustCompile(`#[0-9a-f]{5}\(`)
	for _, c := range []struct {
		n    fmt.Stringer
		want string
	}{
		{NewValueNotifier(3.0), "ValueNotifier<double>#HASH(3.0)"},
		{NewValueNotifier(7), "ValueNotifier<int>#HASH(7)"},
		{NewValueNotifier("s"), "ValueNotifier<String>#HASH(s)"},
		{NewValueNotifier[*float64](nil), "ValueNotifier<double>#HASH(null)"},
		{NewValueNotifier(fruit(1)), "ValueNotifier<fruit>#HASH(fruit.pear)"},
		{NewValueNotifier([]int{1, 2}), "ValueNotifier<List<int>>#HASH([1, 2])"},
		{NewHasListenersTester(true), "HasListenersTester<bool>#HASH(true)"},
	} {
		got := hash.ReplaceAllString(c.n.String(), "#HASH(")
		if got != c.want {
			t.Errorf("got %q, want %q", got, c.want)
		}
	}
	n := NewValueNotifier(1)
	want := fmt.Sprintf("ValueNotifier<int>#%05x(1)", dart.IdentityHashCode(n)&0xFFFFF)
	if n.String() != want {
		t.Fatal(n.String(), want)
	}
}

// overridingNotifier overrides NotifyListeners and Value, which
// ValueNotifier's SetValue and String reach through self.
type overridingNotifier struct {
	ValueNotifier[int]
	log []string
}

func (o *overridingNotifier) NotifyListeners() {
	o.log = append(o.log, "override")
	o.ValueNotifier.NotifyListeners()
}

func (o *overridingNotifier) Value() int { return o.ValueNotifier.Value() * 10 }

func TestValueNotifierCallsOverrides(t *testing.T) {
	setUp(t)
	o := &overridingNotifier{}
	o.Construct(o, 1)
	o.AddListener(func() { o.log = append(o.log, "listener") })
	o.SetValue(2)
	if !slices.Equal(o.log, []string{"override", "listener"}) {
		t.Fatal(o.log)
	}
	if got := regexp.MustCompile(`#[0-9a-f]{5}`).ReplaceAllString(o.String(), "#H"); got != "overridingNotifier#H(20)" {
		t.Fatal(got)
	}
}

func TestErrorContextString(t *testing.T) {
	if ErrorContextListener.String() != "listener" || ErrorContextAssertion.String() != "assertion" {
		t.Fatal("names")
	}
}
