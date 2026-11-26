package difftest

// Shared runner of the services difftests that log channel traffic
// (platform_views, mouse_cursor, undo_scribe_misc): each runs its scenes
// as one async test body in a child process with the services test
// binding, and compares its log with Flutter's build/<name>.json line by
// line.

import (
	"encoding/json"
	"errors"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/services"
	flutter_test "github.com/jonathanperis/flutter-go/packages/flutter_test/lib/src"
	"github.com/jonathanperis/flutter-go/src/async"
	"github.com/jonathanperis/flutter-go/src/dart"
)

// channelLog is the scenes' log and its helpers (Dart's top-level log,
// ticks, track, settle).
type channelLog struct{ lines []string }

func (l *channelLog) add(s string) { l.lines = append(l.lines, s) }

// ticks logs "tick i" from a chain of n microtasks.
func (l *channelLog) ticks(n int) {
	i := 0
	var step func()
	step = func() {
		l.add("tick " + strconv.Itoa(i))
		i++
		if i < n {
			async.ScheduleMicrotask(step)
		}
	}
	async.ScheduleMicrotask(step)
}

// showValue is the scenes' show(): PlatformExceptions with all fields,
// errors and values as Dart's toString.
func showValue(v any) string {
	if err, ok := v.(error); ok {
		var pe *services.PlatformException
		if errors.As(err, &pe) {
			return "PlatformException(" + pe.Code + ", " + codecNullable(pe.Message) + ", " + dart.ToString(pe.Details) + ", " +
				codecNullable(pe.Stacktrace) + ")"
		}
		if p, ok := err.(async.PanicError); ok {
			return dart.ToString(p.Value)
		}
	}
	return dart.ToString(v)
}

// track logs a future's value ("name -> v") or error ("name !! e").
func trackFuture[T any](l *channelLog, name string, f async.Future[T], show func(T) string) {
	async.OnComplete(f, func(v T) { l.add(name + " -> " + show(v)) }, func(err error) { l.add(name + " !! " + showValue(err)) })
}

func (l *channelLog) trackVoid(name string, f async.Future[async.Void]) {
	trackFuture(l, name, f, func(async.Void) string { return "null" })
}

func settle() { async.FutureDelayed(time.Millisecond, func() any { return nil }).Await() }

func bytesList(b []byte) string {
	if b == nil {
		return "null"
	}
	parts := make([]string, len(b))
	for i, x := range b {
		parts[i] = strconv.Itoa(int(x))
	}
	return strings.Join(parts, ",")
}

func testMessenger() *flutter_test.TestDefaultBinaryMessenger {
	return flutter_test.TestDefaultBinaryMessengerBindingInstance().DefaultBinaryMessenger().(*flutter_test.TestDefaultBinaryMessenger)
}

// runChannelScenes runs body (the scenes) and compares the log with
// flutter/build/<name>.json.
func runChannelScenes(t *testing.T, name string, body func(l *channelLog)) {
	data, err := os.ReadFile("flutter/build/" + name + ".json")
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
	newChannelsBinding()
	loop := async.NewLoop()
	defer async.SetCurrent(loop)()
	flutter_test.SetCurrentTest(t)
	l := &channelLog{}
	done := false
	var failure error
	async.OnComplete(async.AsyncVoid(func() { body(l) }), func(async.Void) { done = true }, func(err error) { failure = err; done = true })
	for i := 0; !done && i < 100000; i++ {
		loop.FlushMicrotasks()
		if done || loop.RunPosted() > 0 {
			continue
		}
		if d, ok := loop.NextDeadline(); ok {
			loop.ElapseTo(d)
		} else {
			break
		}
	}
	if failure != nil {
		t.Fatalf("scenes failed: %v (log so far: %q)", failure, l.lines[max(0, len(l.lines)-5):])
	}
	if !done {
		t.Fatal("scenes never completed")
	}
	log := l.lines
	for i := range max(len(log), len(ref.Log)) {
		var got, want string
		if i < len(log) {
			got = log[i]
		}
		if i < len(ref.Log) {
			want = ref.Log[i]
		}
		if got != want {
			t.Fatalf("line %d differs:\n Go:      %q\n Flutter: %q\ncontext (Go): %q", i, got, want, log[max(0, i-5):min(len(log), i+3)])
		}
	}
	t.Logf("%d lines identical", len(log))
}
