package ui

// Port of engine/src/flutter/testing/dart/channel_buffers_test.dart
// (Flutter 3.50.1), all 16 tests, plus tests of our own (below the port).
//
// Adapted, not skipped:
//   - "ChannelBuffers uses the right zones": Go has no zones; the port checks
//     that the listener and the response callback are called, in that order,
//     in the microtask after the push.
//   - _TestChannelBuffers (a subclass overriding resize/allowOverflow) is a
//     type embedding ChannelBuffers that calls SetSelf.
//   - "push rejects names with nulls" checks the AssertionError text.
// Dart's `await null` is awaitNull (one microtask turn); each test body runs
// as an async function on its own loop.

import (
	"strings"
	"testing"
	"time"

	"github.com/jonathanperis/flutter-go/src/async"
	"github.com/jonathanperis/flutter-go/src/dart"
)

func makeByteData(s string) []byte { return []byte(s) }

func resizeLegacy(b *ChannelBuffers, name string, newSize int) {
	b.HandleMessage(makeByteData("resize\r" + name + "\r" + itoa(newSize)))
}

func awaitNull() { async.FutureValue(async.Void{}).Await() }

// runAsync runs body as a Dart async test body on a fresh loop, flushing
// microtasks and timers, and fails if it panics or never completes.
func runAsync(t *testing.T, body func()) {
	t.Helper()
	l := async.NewLoop()
	restore := async.SetCurrent(l)
	defer restore()
	f := async.AsyncVoid(body)
	var err error
	done := false
	async.OnComplete(f, func(async.Void) { done = true }, func(e error) { err = e; done = true })
	l.FlushMicrotasks()
	l.FlushTimers(time.Hour, true)
	l.FlushMicrotasks()
	if err != nil {
		t.Fatal(err)
	}
	if !done {
		t.Fatal("test body did not complete")
	}
}

func panics(f func()) (r any) {
	defer func() { r = recover() }()
	f()
	return nil
}

func eqLog(t *testing.T, got, want []string) {
	t.Helper()
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Errorf("log:\n got  %q\n want %q", got, want)
	}
}

func voidFuture() async.Future[async.Void] { return async.FutureValue(async.Void{}) }

func TestChannelBuffersPushDrain(t *testing.T) {
	runAsync(t, func() {
		data := makeByteData("bar")
		buffers := NewChannelBuffers()
		called := false
		buffers.Push("foo", data, func([]byte) { called = true })
		buffers.Drain("foo", func(drainedData []byte, drainedCallback PlatformMessageResponseCallback) async.Future[async.Void] {
			if string(drainedData) != "bar" || called {
				t.Errorf("drained %q called=%v", drainedData, called)
			}
			drainedCallback(drainedData)
			if !called {
				t.Error("callback not called")
			}
			return voidFuture()
		}).Await()
	})
}

func TestChannelBuffersDrainIsSync(t *testing.T) {
	runAsync(t, func() {
		buffers := NewChannelBuffers()
		buffers.Push("foo", makeByteData("message"), func([]byte) {})
		var log []string
		completer := async.NewCompleter[async.Void]()
		async.ScheduleMicrotask(func() { log = append(log, "before drain, microtask") })
		log = append(log, "before drain")
		buffers.Drain("foo", func([]byte, PlatformMessageResponseCallback) async.Future[async.Void] {
			return async.AsyncVoid(func() {
				log = append(log, "callback")
				completer.Complete(async.Void{})
			})
		}).Ignore()
		log = append(log, "after drain, before await")
		completer.Future().Await()
		log = append(log, "after await")
		eqLog(t, log, []string{"before drain", "callback", "after drain, before await", "before drain, microtask", "after await"})
	})
}

func TestChannelBuffersPushDrainZero(t *testing.T) {
	runAsync(t, func() {
		buffers := NewChannelBuffers()
		resizeLegacy(buffers, "foo", 0)
		buffers.Push("foo", makeByteData("bar"), func([]byte) {})
		didCall := false
		buffers.Drain("foo", func([]byte, PlatformMessageResponseCallback) async.Future[async.Void] {
			didCall = true
			return voidFuture()
		}).Await()
		if didCall {
			t.Error("drained a message pushed to a zero-capacity channel")
		}
	})
}

func TestChannelBuffersDrainWhenEmpty(t *testing.T) {
	runAsync(t, func() {
		buffers := NewChannelBuffers()
		didCall := false
		buffers.Drain("foo", func([]byte, PlatformMessageResponseCallback) async.Future[async.Void] {
			didCall = true
			return voidFuture()
		}).Await()
		if didCall {
			t.Error("called")
		}
	})
}

func drainAll(buffers *ChannelBuffers, name string) []string {
	var got []string
	buffers.Drain(name, func(d []byte, _ PlatformMessageResponseCallback) async.Future[async.Void] {
		got = append(got, string(d))
		return voidFuture()
	}).Await()
	return got
}

func TestChannelBuffersOverflow(t *testing.T) {
	runAsync(t, func() {
		buffers := NewChannelBuffers()
		resizeLegacy(buffers, "foo", 3)
		for _, s := range []string{"one", "two", "three", "four"} {
			buffers.Push("foo", makeByteData(s), func([]byte) {})
		}
		eqLog(t, drainAll(buffers, "foo"), []string{"two", "three", "four"})
	})
}

func TestChannelBuffersResizeDrop(t *testing.T) {
	runAsync(t, func() {
		buffers := NewChannelBuffers()
		resizeLegacy(buffers, "foo", 100)
		buffers.Push("foo", makeByteData("one"), func([]byte) {})
		buffers.Push("foo", makeByteData("two"), func([]byte) {})
		resizeLegacy(buffers, "foo", 1)
		eqLog(t, drainAll(buffers, "foo"), []string{"two"})
	})
}

func TestChannelBuffersResizeDroppingCallsCallback(t *testing.T) {
	buffers := NewChannelBuffers()
	didCallCallback := false
	resizeLegacy(buffers, "foo", 100)
	buffers.Push("foo", makeByteData("one"), func(d []byte) {
		if d != nil {
			t.Error("non-null reply")
		}
		didCallCallback = true
	})
	buffers.Push("foo", makeByteData("two"), func([]byte) { t.Error("wrong callback called") })
	if didCallCallback {
		t.Fatal("called too early")
	}
	resizeLegacy(buffers, "foo", 1)
	if !didCallCallback {
		t.Fatal("not called")
	}
}

func TestChannelBuffersOverflowCallsCallback(t *testing.T) {
	buffers := NewChannelBuffers()
	didCallCallback := false
	resizeLegacy(buffers, "foo", 1)
	buffers.Push("foo", makeByteData("one"), func(d []byte) {
		if d != nil {
			t.Error("non-null reply")
		}
		didCallCallback = true
	})
	buffers.Push("foo", makeByteData("two"), func([]byte) { t.Error("wrong callback called") })
	if !didCallCallback {
		t.Fatal("not called")
	}
}

func TestChannelBuffersHandleGarbage(t *testing.T) {
	buffers := NewChannelBuffers()
	if _, ok := panics(func() { buffers.HandleMessage(makeByteData("asdfasdf")) }).(dart.Exception); !ok {
		t.Error("no Exception")
	}
}

func TestChannelBuffersHandleResizeGarbage(t *testing.T) {
	buffers := NewChannelBuffers()
	if _, ok := panics(func() { buffers.HandleMessage(makeByteData("resize\rfoo\rbar")) }).(dart.FormatException); !ok {
		t.Error("no FormatException")
	}
}

func TestChannelBuffersSetListener(t *testing.T) {
	runAsync(t, func() {
		var log []string
		buffers := NewChannelBuffers()
		listener := func(prefix string) ChannelCallback {
			return func(data []byte, _ PlatformMessageResponseCallback) {
				if data == nil {
					t.Error("null data")
				}
				log = append(log, prefix+": "+string(data))
			}
		}
		none := func([]byte) {}
		buffers.Push("a", makeByteData("one"), none)
		buffers.Push("b", makeByteData("two"), none)
		buffers.Push("a", makeByteData("three"), none)
		log = append(log, "top")
		buffers.SetListener("a", listener("a1"))
		log = append(log, "-1")
		awaitNull()
		log = append(log, "-2")
		buffers.SetListener("a", listener("a2"))
		log = append(log, "-3")
		awaitNull()
		log = append(log, "-4")
		buffers.SetListener("b", listener("b"))
		log = append(log, "-5")
		awaitNull() // first microtask after setting listener drains the first message
		awaitNull() // second microtask ends the draining.
		log = append(log, "-6")
		buffers.Push("b", makeByteData("four"), none)
		buffers.Push("a", makeByteData("five"), none)
		log = append(log, "-7")
		awaitNull()
		log = append(log, "-8")
		buffers.ClearListener("a")
		buffers.Push("a", makeByteData("six"), none)
		buffers.Push("b", makeByteData("seven"), none)
		awaitNull()
		log = append(log, "-9")
		eqLog(t, log, []string{"top", "-1", "a1: three", "-2", "-3", "-4", "-5", "b: two", "-6",
			"b: four", "a2: five", "-7", "-8", "b: seven", "-9"})
	})
}

func TestChannelBuffersClearListener(t *testing.T) {
	runAsync(t, func() {
		var log []string
		buffers := NewChannelBuffers()
		none := func([]byte) {}
		buffers.HandleMessage(makeByteData("resize\ra\r10"))
		buffers.Push("a", makeByteData("one"), none)
		buffers.Push("a", makeByteData("two"), none)
		buffers.Push("a", makeByteData("three"), none)
		log = append(log, "-1")
		buffers.SetListener("a", func(data []byte, _ PlatformMessageResponseCallback) {
			log = append(log, "a1: "+string(data))
		})
		awaitNull() // handles one
		log = append(log, "-2")
		buffers.ClearListener("a")
		awaitNull()
		log = append(log, "-3")
		buffers.SetListener("a", func(data []byte, _ PlatformMessageResponseCallback) {
			log = append(log, "a2: "+string(data))
		})
		log = append(log, "-4")
		awaitNull()
		buffers.Push("a", makeByteData("four"), none)
		log = append(log, "-5")
		awaitNull()
		log = append(log, "-6")
		awaitNull()
		log = append(log, "-7")
		awaitNull()
		eqLog(t, log, []string{"-1", "a1: one", "-2", "-3", "-4", "a2: two", "-5", "a2: three", "-6", "a2: four", "-7"})
	})
}

// testChannelBuffers is the Dart test's _TestChannelBuffers.
type testChannelBuffers struct {
	ChannelBuffers
	log []string
}

func newTestChannelBuffers() *testChannelBuffers {
	b := &testChannelBuffers{}
	b.SetSelf(b)
	return b
}

func (b *testChannelBuffers) Resize(name string, newSize int) {
	b.log = append(b.log, "resize "+name+" "+itoa(newSize))
}

func (b *testChannelBuffers) AllowOverflow(name string, allowed bool) {
	s := "false"
	if allowed {
		s = "true"
	}
	b.log = append(b.log, "allowOverflow "+name+" "+s)
}

func TestChannelBuffersHandleMessageForResize(t *testing.T) {
	buffers := newTestChannelBuffers()
	bytes := []byte{255, 255, 255, 7, 6, 114, 101, 115, 105, 122, 101, 12, 2, 7, 6, 97, 98, 99, 100, 101, 102, 3,
		57, 48, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 255, 255, 255}
	buffers.HandleMessage(bytes[3 : 3+27])
	eqLog(t, buffers.log, []string{"resize abcdef 12345"})
}

func TestChannelBuffersHandleMessageForOverflow(t *testing.T) {
	buffers := newTestChannelBuffers()
	bytes := []byte{255, 255, 255, 7, 8, 111, 118, 101, 114, 102, 108, 111, 119, 12, 2, 7, 6, 97, 98, 99, 100, 101,
		102, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 255, 255, 255}
	buffers.HandleMessage(bytes[3 : 3+24])
	eqLog(t, buffers.log, []string{"allowOverflow abcdef false"})
}

func TestChannelBuffersCallsListenerThenResponse(t *testing.T) {
	runAsync(t, func() {
		var log []string
		buffers := NewChannelBuffers()
		buffers.SetListener("a", func(data []byte, callback PlatformMessageResponseCallback) {
			log = append(log, "callback1")
			callback(data)
		})
		buffers.Push("a", []byte{}, func(data []byte) {
			if data == nil {
				t.Error("an empty ByteData became null")
			}
			log = append(log, "callback2")
		})
		awaitNull()
		eqLog(t, log, []string{"callback1", "callback2"})
	})
}

func TestChannelBuffersPushRejectsNamesWithNulls(t *testing.T) {
	buffers := NewChannelBuffers()
	for _, f := range []func(){
		func() { buffers.Push("foo\x00bar", makeByteData("blabla"), func([]byte) {}) },
		func() { buffers.SetListener("foo\x00bar", func([]byte, PlatformMessageResponseCallback) {}) },
	} {
		e, ok := panics(f).(*dart.AssertionError)
		if !ok || !strings.Contains(e.String(), "U+0000 NULL") {
			t.Errorf("got %v", e)
		}
	}
}

// ---- Own tests ----

func capturePrintDebug(t *testing.T) *[]string {
	var printed []string
	prev := printDebug
	printDebug = func(s string) { printed = append(printed, s) }
	t.Cleanup(func() { printDebug = prev })
	return &printed
}

func discardMessage(name string, capacity string) string {
	return "A message on the " + name + " channel was discarded before it could be handled.\n" +
		"This happens when a plugin sends messages to the framework side before the framework has had an opportunity to register a listener. See the ChannelBuffers API documentation for details on how to configure the channel to expect more messages, or to expect messages to get discarded:\n" +
		"  https://api.flutter.dev/flutter/dart-ui/ChannelBuffers-class.html\n" +
		"The capacity of the " + name + " channel is " + capacity + "."
}

func TestChannelBuffersDiscardWarning(t *testing.T) {
	if !debugMode {
		t.Skip("debug builds only: allowOverflow and the discard warning are debug-only")
	}
	printed := capturePrintDebug(t)
	b := NewChannelBuffers()
	b.Push("c", nil, nil)
	if len(*printed) != 0 {
		t.Fatalf("warned without a discard: %q", *printed)
	}
	b.Push("c", nil, nil)
	b.Resize("d", 2)
	for range 3 {
		b.Push("d", nil, nil)
	}
	b.Resize("z", 0)
	b.Push("z", nil, nil)
	want := []string{discardMessage("c", "1 message"), discardMessage("d", "2 messages"), discardMessage("z", "0 messages")}
	eqLog(t, *printed, want)

	// allowOverflow silences the warning of a zero-capacity channel only (debug
	// builds): a full channel with capacity warns anyway (Dart's quirk);
	// allowOverflow(false) restores it.
	*printed = nil
	b.AllowOverflow("c", true)
	b.Push("c", nil, nil)
	b.AllowOverflow("z", true)
	b.Push("z", nil, nil)
	b.Push("d", nil, nil)
	eqLog(t, *printed, []string{discardMessage("c", "1 message"), discardMessage("d", "2 messages")})
	*printed = nil
	b.AllowOverflow("z", false)
	b.Push("z", nil, nil)
	eqLog(t, *printed, []string{discardMessage("z", "0 messages")})

	// allowOverflow(true) creates the channel; allowOverflow(false) doesn't.
	b.AllowOverflow("new", false)
	if _, ok := b.channels["new"]; ok {
		t.Error("allowOverflow(false) created a channel")
	}
	b.AllowOverflow("new2", true)
	if c := b.channels["new2"]; c == nil || c.debugEnableDiscardWarnings || c.capacity != 1 {
		t.Errorf("allowOverflow(true) channel: %+v", c)
	}
	if r, ok := panics(func() { b.AllowOverflow("x\x00", true) }).(*dart.AssertionError); !ok || !strings.Contains(r.String(), "U+0000") {
		t.Errorf("allowOverflow null name: %v", r)
	}
	if r, ok := panics(func() { b.Resize("x\x00", 3) }).(*dart.AssertionError); !ok || !strings.Contains(r.String(), "U+0000") {
		t.Errorf("resize null name: %v", r)
	}
}

func TestChannelBuffersOverflowOrder(t *testing.T) {
	b := NewChannelBuffers()
	var log []string
	cb := func(name string) PlatformMessageResponseCallback {
		return func(d []byte) {
			if d != nil {
				t.Error("non-null")
			}
			log = append(log, name)
		}
	}
	b.Resize("a", 4)
	for _, n := range []string{"1", "2", "3", "4", "5"} {
		b.Push("a", []byte(n), cb(n))
	}
	b.Resize("a", 1)
	b.Resize("a", 0)
	eqLog(t, log, []string{"1", "2", "3", "4", "5"})
	if len(b.channels["a"].queue) != 0 {
		t.Error("queue not empty")
	}
	// zero capacity: the message is neither queued nor answered.
	called := false
	b.Push("a", []byte("6"), func([]byte) { called = true })
	if called || len(b.channels["a"].queue) != 0 {
		t.Error("zero capacity kept or answered the message")
	}
	// growing keeps the queue.
	b.Resize("a", 3)
	b.Push("a", []byte("x"), nil)
	b.Push("a", []byte("y"), nil)
	b.Resize("a", 5)
	if len(b.channels["a"].queue) != 2 {
		t.Error("growing dropped messages")
	}
}

type updateClient struct {
	testClient
	log []string
}

type testClient struct{}

func (testClient) ScheduleFrame()             {}
func (testClient) EndWarmUpFrame()            {}
func (testClient) SetNeedsReportTimings(bool) {}

func (c *updateClient) SendChannelUpdate(name string, listening bool) {
	s := "false"
	if listening {
		s = "true"
	}
	c.log = append(c.log, name+" "+s)
}

func TestChannelBuffersSendChannelUpdate(t *testing.T) {
	l := async.NewLoop()
	defer async.SetCurrent(l)()
	b := NewChannelBuffers()
	b.SetListener("a", func([]byte, PlatformMessageResponseCallback) {}) // no configuration: nothing
	c := &updateClient{}
	defer SetPlatformConfiguration(NewPlatformConfiguration(c))()
	b.SetListener("a", func([]byte, PlatformMessageResponseCallback) {})
	b.ClearListener("a")
	b.ClearListener("never") // unknown channel: no update
	b.SetListener("b", func([]byte, PlatformMessageResponseCallback) {})
	eqLog(t, c.log, []string{"a true", "a false", "b true"})
	// A configuration whose client doesn't take updates.
	defer SetPlatformConfiguration(NewPlatformConfiguration(testClient{}))()
	b.SetListener("c", func([]byte, PlatformMessageResponseCallback) {})
	l.FlushMicrotasks()
}

// Pushes while a listener drains go to the end of the queue and are
// delivered by the drain, in order; replacing the listener mid-drain makes
// the new one receive the rest; a direct push while not draining is
// synchronous.
func TestChannelBuffersPushDuringDrain(t *testing.T) {
	runAsync(t, func() {
		var log []string
		b := NewChannelBuffers()
		b.Resize("a", 10)
		b.Push("a", []byte("1"), nil)
		b.Push("a", []byte("2"), nil)
		b.SetListener("a", func(d []byte, _ PlatformMessageResponseCallback) {
			log = append(log, "L1 "+string(d))
			if string(d) == "1" {
				b.Push("a", []byte("3"), nil)
			}
		})
		awaitNull()
		log = append(log, "-")
		b.SetListener("a", func(d []byte, _ PlatformMessageResponseCallback) { log = append(log, "L2 "+string(d)) })
		awaitNull()
		awaitNull()
		awaitNull()
		log = append(log, "-")
		b.Push("a", []byte("4"), nil)
		log = append(log, "end")
		eqLog(t, log, []string{"L1 1", "-", "L2 2", "L2 3", "-", "L2 4", "end"})
	})
}

func TestChannelBuffersHandleMessageErrors(t *testing.T) {
	str := func(s string) []byte { return append([]byte{7, byte(len(s))}, s...) }
	cat := func(parts ...[]byte) []byte {
		var out []byte
		for _, p := range parts {
			out = append(out, p...)
		}
		return out
	}
	const ctl = "dev.flutter/channel-buffers"
	for _, c := range []struct {
		data []byte
		want string
	}{
		{cat([]byte{7, 254}), "Exception: Unrecognized message sent to " + ctl + " (method name too long)"},
		{cat([]byte{7, 253}, []byte(strings.Repeat("x", 253))), "Exception: Unrecognized method '" + strings.Repeat("x", 253) + "' sent to " + ctl},
		{str("foo"), "Exception: Unrecognized method 'foo' sent to " + ctl},
		{cat(str("resize"), []byte{13}), "Exception: Invalid arguments for 'resize' method sent to " + ctl + " (arguments must be a two-element list, channel name and new capacity)"},
		{cat(str("resize"), []byte{12, 1}), "Exception: Invalid arguments for 'resize' method sent to " + ctl + " (arguments must be a two-element list, channel name and new capacity)"},
		{cat(str("resize"), []byte{12, 2, 3}), "Exception: Invalid arguments for 'resize' method sent to " + ctl + " (first argument must be a string)"},
		{cat(str("resize"), []byte{12, 2, 7, 254}), "Exception: Invalid arguments for 'resize' method sent to " + ctl + " (channel name must be less than 254 characters long)"},
		{cat(str("resize"), []byte{12, 2}, str("a\x00b")), "Exception: Invalid arguments for 'resize' method sent to " + ctl + " (channel name must not contain any null bytes)"},
		{cat(str("resize"), []byte{12, 2}, str("ab"), []byte{4}), "Exception: Invalid arguments for 'resize' method sent to " + ctl + " (second argument must be an integer in the range 0 to 2147483647)"},
		{cat(str("overflow"), []byte{13}), "Exception: Invalid arguments for 'overflow' method sent to " + ctl + " (arguments must be a two-element list, channel name and flag state)"},
		{cat(str("overflow"), []byte{12, 1}), "Exception: Invalid arguments for 'overflow' method sent to " + ctl + " (arguments must be a two-element list, channel name and flag state)"},
		{cat(str("overflow"), []byte{12, 2, 3}), "Exception: Invalid arguments for 'overflow' method sent to " + ctl + " (first argument must be a string)"},
		{cat(str("overflow"), []byte{12, 2, 7, 254}), "Exception: Invalid arguments for 'overflow' method sent to " + ctl + " (channel name must be less than 254 characters long)"},
		{cat(str("overflow"), []byte{12, 2}, str("ab"), []byte{0}), "Exception: Invalid arguments for 'overflow' method sent to " + ctl + " (second argument must be a boolean)"},
		{cat(str("overflow"), []byte{12, 2}, str("ab"), []byte{3}), "Exception: Invalid arguments for 'overflow' method sent to " + ctl + " (second argument must be a boolean)"},
		{[]byte("resize\ra"), "Exception: Unrecognized message [resize, a] sent to " + ctl + "."},
		{[]byte("resize\ra\r1\r2"), "Exception: Unrecognized message [resize, a, 1, 2] sent to " + ctl + "."},
		{[]byte("resiz\ra\r1"), "Exception: Unrecognized message [resiz, a, 1] sent to " + ctl + "."},
		{[]byte("resize\ra\rbar"), "FormatException: Invalid radix-10 number (at character 1)\nbar\n^\n"},
		{[]byte("resize\ra\r"), "FormatException: Invalid number (at character 1)\n\n^\n"},
		{[]byte("resize\ra\r  "), "FormatException: Invalid number (at character 3)\n  \n  ^\n"},
		{[]byte("resize\ra\r-"), "FormatException: Invalid number (at character 2)\n-\n ^\n"},
		{[]byte("resize\ra\r0x"), "FormatException: Invalid number (at character 3)\n0x\n  ^\n"},
		{[]byte("resize\ra\r 0x1g"), "FormatException: Invalid radix-16 number (at character 4)\n 0x1g\n   ^\n"},
		{[]byte("resize\ra\r00012a"), "FormatException: Invalid radix-10 number (at character 4)\n00012a\n   ^\n"},
		{[]byte("resize\ra\r1234567890123456789x"), "FormatException: Invalid radix-10 number (at character 3)\n1234567890123456789x\n  ^\n"},
		{[]byte("resize\ra\r12345678901234567890"), "FormatException: Positive input exceeds the limit of integer\n12345678901234567890"},
		{[]byte("resize\ra\r-9223372036854775809"), "FormatException: Negative input exceeds the limit of integer\n-9223372036854775809"},
		{[]byte("resize\ra\r0x10000000000000000"), "FormatException: Positive input exceeds the limit of integer\n0x10000000000000000"},
		{[]byte{0xff, 'a'}, "FormatException: Invalid UTF-8 byte (at offset 0)"},
	} {
		r := panics(func() { NewChannelBuffers().HandleMessage(c.data) })
		e, ok := r.(error)
		if !ok || e.Error() != c.want {
			t.Errorf("%q:\n got  %v\n want %q", c.data, r, c.want)
		}
	}
}

func TestDartIntParse(t *testing.T) {
	for s, want := range map[string]int{
		"0": 0, "12345": 12345, " \t+42\n": 42, "-17": -17, "0x1F": 31, "-0xff": -255, "000": 0,
		"9223372036854775807": 9223372036854775807, "-9223372036854775808": -9223372036854775808,
		"123456789012345678": 123456789012345678, "1234567890123456789": 1234567890123456789,
		"0xFFFFFFFFFFFFFFFF": -1, "0x8000000000000000": -9223372036854775808, "\u3000\ufeff7\u2028": 7,
		"-0": 0, "0X10": 16,
	} {
		if got := dartIntParse(s); got != want {
			t.Errorf("int.parse(%q) = %d, want %d", s, got, want)
		}
	}
}

func TestChannelBuffersLegacyResize(t *testing.T) {
	b := newTestChannelBuffers()
	b.HandleMessage([]byte("resize\rchan\r 0x10 "))
	b.HandleMessage([]byte("resize\r\r-3"))
	eqLog(t, b.log, []string{"resize chan 16", "resize  -3"})
}

func TestChannelBuffersResizeLittleEndian(t *testing.T) {
	b := newTestChannelBuffers()
	b.HandleMessage(append([]byte{7, 6}, append([]byte("resize"), 12, 3, 7, 1, 'q', 3, 1, 2, 3, 0x80, 9)...))
	eqLog(t, b.log, []string{"resize q " + itoa(0x80030201)})
}

func TestChannelBuffersInstance(t *testing.T) {
	if ChannelBuffersInstance() == nil || ChannelBuffersInstance() != ChannelBuffersInstance() {
		t.Error("not a singleton")
	}
	var zero ChannelBuffers // the zero value works, as its own self
	zero.Resize("a", 2)
	if zero.channels["a"].capacity != 2 {
		t.Error("zero value")
	}
}

func TestChannelBuffersDrainNilFutureAndPop(t *testing.T) {
	runAsync(t, func() {
		b := NewChannelBuffers()
		b.Resize("a", 3)
		var replies []string
		for _, s := range []string{"1", "2"} {
			b.Push("a", []byte(s), func(d []byte) { replies = append(replies, string(d)) })
		}
		b.Drain("a", func(d []byte, cb PlatformMessageResponseCallback) async.Future[async.Void] {
			cb(append([]byte("r"), d...))
			return nil
		}).Await()
		eqLog(t, replies, []string{"r1", "r2"})
		if len(b.channels["a"].queue) != 0 {
			t.Error("drain left messages")
		}
		if _, ok := panics(func() { b.channels["a"].pop() }).(dart.StateError); !ok {
			t.Error("pop on empty")
		}
	})
}
