package developer

import (
	"testing"

	"github.com/jonathanperis/flutter-go/flutter_go/async"
	"github.com/jonathanperis/flutter-go/flutter_go/dart"
)

func expectPanic(t *testing.T, want string, f func()) {
	t.Helper()
	defer func() {
		t.Helper()
		r := recover()
		err, ok := r.(error)
		if !ok || err.Error() != want {
			t.Fatalf("recovered %#v, want %q", r, want)
		}
	}()
	f()
}

func TestRegisterExtension(t *testing.T) {
	handler := func(string, map[string]string) async.Future[*ServiceExtensionResponse] { return nil }
	expectPanic(t, `Invalid argument (method): Must begin with ext.: "foo"`, func() { RegisterExtension("foo", handler) })
	RegisterExtension("ext.test.a", handler)
	if LookupExtension("ext.test.a") == nil || LookupExtension("ext.test.b") != nil {
		t.Fatal("lookup")
	}
	expectPanic(t, "Invalid argument(s): Extension already registered: ext.test.a", func() { RegisterExtension("ext.test.a", handler) })
}

func TestServiceExtensionResponse(t *testing.T) {
	r := NewServiceExtensionResponseResult("{}")
	if r.IsError() || *r.Result != "{}" {
		t.Fatal(r)
	}
	for _, code := range []int{ServiceExtensionResponseInvalidParams, -32000, -32016} {
		if e := NewServiceExtensionResponseError(code, "d"); !e.IsError() || *e.ErrorCode != code {
			t.Fatal(code)
		}
	}
	for _, code := range []int{-31999, -32017, 0} {
		func() {
			defer func() {
				if _, ok := recover().(dart.ArgumentError); !ok {
					t.Fatalf("code %d accepted", code)
				}
			}()
			NewServiceExtensionResponseError(code, "d")
		}()
	}
}

func TestPostEventStreams(t *testing.T) {
	PostEvent("kind", map[string]any{"a": 1})
	PostEvent("kind", nil, PostEventOptions{Stream: "Mine"})
	expectPanic(t, `Invalid argument (stream): Cannot be a protected stream.: "Timeline"`,
		func() { PostEvent("kind", nil, PostEventOptions{Stream: "Timeline"}) })
	expectPanic(t, `Invalid argument (stream): Cannot start with an underscore.: "_x"`,
		func() { PostEvent("kind", nil, PostEventOptions{Stream: "_x"}) })
}

func TestTimelineStacks(t *testing.T) {
	TimelineStartSync("a")
	TimelineStartSync("b")
	TimelineFinishSync()
	TimelineFinishSync()
	expectPanic(t, "Bad state: Uneven calls to startSync and finishSync", TimelineFinishSync)
	if v := TimelineTimeSync("t", func() int { return 3 }); v != 3 {
		t.Fatal(v)
	}
	task := NewTimelineTask()
	task.Start("x", nil)
	expectPanic(t, "Bad state: You cannot pass a TimelineTask without finishing all started operations", func() { task.Pass() })
	task.Finish(nil)
	expectPanic(t, "Bad state: Uneven calls to start and finish", func() { task.Finish(nil) })
	if task.Pass() <= 0 {
		t.Fatal("task id")
	}
	if FlowBegin(nil).ID == FlowBegin(nil).ID || FlowStep(7).ID != 7 || FlowEnd(8).ID != 8 {
		t.Fatal("flows")
	}
	if TimelineNow() < 0 {
		t.Fatal("now")
	}
}
