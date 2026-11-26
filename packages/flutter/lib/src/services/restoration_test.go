package services_test

// Port of packages/flutter/test/services/restoration_test.dart (Flutter
// 3.50.1), 9/9. testWidgets' tester.pump/idle are this package's pump and
// idle (see system_helpers_test.go); `debugIsSerializableForRestoration
// (Container())` is a Go struct the codec can't encode.

import (
	"testing"

	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/scheduler"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/services"
	"github.com/jonathanperis/flutter-go/src/async"
	"github.com/jonathanperis/flutter-go/src/dart"
)

func pushDataFromEngine(data any) {
	messenger().HandlePlatformMessage("flutter/restoration",
		services.NewStandardMethodCodec().EncodeMethodCall(services.NewMethodCall("push", data)), func([]byte) {}).Await()
}

func packageRestorationData(enabled bool, data any) map[any]any {
	var encoded []byte
	if data != nil {
		encoded = services.NewStandardMessageCodec().EncodeMessage(data)
	}
	var dataValue any
	if encoded != nil {
		dataValue = encoded
	}
	return map[any]any{"enabled": enabled, "data": dataValue}
}

func createEncodedRestorationData1() map[any]any {
	return packageRestorationData(true, []dart.MapEntry[string, any]{
		{Key: valuesMapKey, Value: []dart.MapEntry[string, any]{{Key: "value1", Value: 10}, {Key: "value2", Value: "Hello"}}},
		{Key: childrenMapKey, Value: []dart.MapEntry[string, any]{
			{Key: "child1", Value: []dart.MapEntry[string, any]{{Key: valuesMapKey, Value: []dart.MapEntry[string, any]{{Key: "another value", Value: 22}}}}},
		}},
	})
}

func createEncodedRestorationData2() map[any]any {
	return packageRestorationData(true, []dart.MapEntry[string, any]{
		{Key: valuesMapKey, Value: []dart.MapEntry[string, any]{{Key: "foo", Value: 33}}},
		{Key: childrenMapKey, Value: []dart.MapEntry[string, any]{
			{Key: "childFoo", Value: []dart.MapEntry[string, any]{{Key: valuesMapKey, Value: []dart.MapEntry[string, any]{{Key: "bar", Value: "Hello"}}}}},
		}},
	})
}

// mockRestorationChannel makes the engine side of flutter/restoration
// answer every call with result's future, recording the calls.
func mockRestorationChannel(result func() async.Future[any]) (*[]services.MethodCall, func()) {
	calls := &[]services.MethodCall{}
	messenger().SetMockMethodCallHandler(services.SystemChannels.Restoration, func(call services.MethodCall) async.Future[any] {
		*calls = append(*calls, call)
		return result()
	})
	return calls, func() { messenger().SetMockMethodCallHandler(services.SystemChannels.Restoration, nil) }
}

func thenBucket(f async.Future[*services.RestorationBucket], set func(*services.RestorationBucket)) {
	async.OnComplete(f, set, nil)
}

func TestRestorationManagerDispatchesMemoryEvents(t *testing.T) {
	ensureInitialized()
	events := memoryEvents(func() { services.NewRestorationManager().Dispose() }, "RestorationManager")
	if !areCreateAndDispose(events) {
		t.Fatalf("events %v", events)
	}
}

func TestRestorationManagerRootBucketRetrieval(t *testing.T) {
	ensureInitialized()
	var result *async.Completer[any] // created on the test's loop
	callsToEngine, done := mockRestorationChannel(func() async.Future[any] { return result.Future() })
	defer done()
	runAsync(t, func() {
		result = async.NewCompleter[any]()
		manager := services.NewRestorationManager()
		defer manager.Dispose()
		rootBucketFuture := manager.RootBucket()
		var rootBucket *services.RestorationBucket
		thenBucket(rootBucketFuture, func(b *services.RestorationBucket) { rootBucket = b })
		expect(rootBucketFuture != nil && rootBucket == nil, "root bucket")

		// Accessing rootBucket again gives same future.
		expect(manager.RootBucket() == rootBucketFuture, "same future")

		// Engine has only been contacted once.
		expect(len(*callsToEngine) == 1 && (*callsToEngine)[0].Method == "get", "calls %v", *callsToEngine)

		// Complete the engine request.
		result.Complete(createEncodedRestorationData1())
		pump()

		// Root bucket future completed.
		expect(rootBucket != nil, "root bucket")
		expect(rootBucket.Read("value1") == 10 && rootBucket.Read("value2") == "Hello", "values")
		child := rootBucket.ClaimChild("child1", nil)
		defer child.Dispose()
		expect(child.Read("another value") == 22, "child value")

		// Accessing the root bucket again completes synchronously with same
		// bucket.
		var synchronousBucket *services.RestorationBucket
		thenBucket(manager.RootBucket(), func(b *services.RestorationBucket) { synchronousBucket = b })
		expect(synchronousBucket != nil && synchronousBucket == rootBucket, "synchronous bucket")
		postTestFrame()
	})
}

func TestRestorationManagerRootBucketReceivedBeforeRetrieval(t *testing.T) {
	ensureInitialized()
	services.SystemChannels.Restoration.SetMethodCallHandler(nil)
	callsToEngine, done := mockRestorationChannel(func() async.Future[any] { return async.FutureValue[any](nil) })
	defer done()
	runAsync(t, func() {
		manager := services.NewRestorationManager()
		defer manager.Dispose()
		pushDataFromEngine(createEncodedRestorationData1())
		var rootBucket *services.RestorationBucket
		thenBucket(manager.RootBucket(), func(b *services.RestorationBucket) { rootBucket = b })
		// Root bucket is available synchronously.
		expect(rootBucket != nil, "root bucket")
		// Engine was never asked.
		expect(len(*callsToEngine) == 0, "calls %v", *callsToEngine)
		postTestFrame()
	})
}

func TestRestorationManagerRootBucketReceivedWhileRetrievalPending(t *testing.T) {
	ensureInitialized()
	services.SystemChannels.Restoration.SetMethodCallHandler(nil)
	var result *async.Completer[any] // created on the test's loop
	callsToEngine, done := mockRestorationChannel(func() async.Future[any] { return result.Future() })
	defer done()
	runAsync(t, func() {
		result = async.NewCompleter[any]()
		manager := services.NewRestorationManager()
		defer manager.Dispose()
		var rootBucket *services.RestorationBucket
		thenBucket(manager.RootBucket(), func(b *services.RestorationBucket) { rootBucket = b })
		expect(rootBucket == nil, "root bucket")
		expect(len(*callsToEngine) == 1 && (*callsToEngine)[0].Method == "get", "calls")

		pushDataFromEngine(createEncodedRestorationData1())
		expect(rootBucket != nil && rootBucket.Read("value1") == 10, "root bucket")

		result.Complete(createEncodedRestorationData2())
		pump()

		var rootBucket2 *services.RestorationBucket
		thenBucket(manager.RootBucket(), func(b *services.RestorationBucket) { rootBucket2 = b })
		expect(rootBucket2 != nil && rootBucket2 == rootBucket, "same bucket")
		expect(rootBucket2.Read("value1") == 10 && !rootBucket2.Contains("foo"), "engine response ignored")
		postTestFrame()
	})
}

func TestRestorationManagerRootBucketReplaced(t *testing.T) {
	ensureInitialized()
	_, done := mockRestorationChannel(func() async.Future[any] { return async.FutureValue[any](createEncodedRestorationData1()) })
	defer done()
	runAsync(t, func() {
		manager := services.NewRestorationManager()
		defer manager.Dispose()
		var rootBucket *services.RestorationBucket
		thenBucket(manager.RootBucket(), func(b *services.RestorationBucket) { rootBucket = b })
		pump()
		expect(rootBucket != nil && rootBucket.Read("value1") == 10, "root bucket")
		child := rootBucket.ClaimChild("child1", nil)
		expect(child.Read("another value") == 22, "child")

		rootReplaced := false
		var newRoot *services.RestorationBucket
		manager.AddListener(func() {
			rootReplaced = true
			thenBucket(manager.RootBucket(), func(b *services.RestorationBucket) { newRoot = b })
			// The new bucket is available synchronously.
			expect(newRoot != nil, "new root")
		})

		// Send new Data.
		pushDataFromEngine(createEncodedRestorationData2())
		expect(rootReplaced && newRoot != rootBucket, "replaced")

		child.Dispose()
		expect(newRoot.Read("foo") == 33 && newRoot.Read("value1") == nil, "new values")
		newChild := newRoot.ClaimChild("childFoo", nil)
		defer newChild.Dispose()
		expect(newChild.Read("bar") == "Hello", "new child")
		postTestFrame()
	})
}

func TestRestorationManagerDisabled(t *testing.T) {
	ensureInitialized()
	var result *async.Completer[any] // created on the test's loop
	_, done := mockRestorationChannel(func() async.Future[any] { return result.Future() })
	defer done()
	runAsync(t, func() {
		result = async.NewCompleter[any]()
		listenerCount := 0
		manager := services.NewRestorationManager()
		manager.AddListener(func() { listenerCount++ })
		defer manager.Dispose()
		var rootBucket *services.RestorationBucket
		rootBucketResolved := false
		thenBucket(manager.RootBucket(), func(b *services.RestorationBucket) { rootBucketResolved = true; rootBucket = b })
		pump()
		expect(!rootBucketResolved && listenerCount == 0, "resolved early")

		result.Complete(packageRestorationData(false, nil))
		pump()
		expect(rootBucketResolved && rootBucket == nil, "disabled")

		// Switch to non-null.
		pushDataFromEngine(createEncodedRestorationData1())
		expect(listenerCount == 1, "listeners %d", listenerCount)
		thenBucket(manager.RootBucket(), func(b *services.RestorationBucket) { rootBucket = b })
		expect(rootBucket != nil, "enabled")

		// Switch to null again.
		pushDataFromEngine(packageRestorationData(false, nil))
		expect(listenerCount == 2, "listeners %d", listenerCount)
		thenBucket(manager.RootBucket(), func(b *services.RestorationBucket) { rootBucket = b })
		expect(rootBucket == nil, "disabled again")
		postTestFrame()
	})
}

func TestRestorationManagerFlushData(t *testing.T) {
	ensureInitialized()
	var result *async.Completer[any] // created on the test's loop
	callsToEngine, done := mockRestorationChannel(func() async.Future[any] { return result.Future() })
	defer done()
	runAsync(t, func() {
		result = async.NewCompleter[any]()
		manager := services.NewRestorationManager()
		defer manager.Dispose()
		var rootBucket *services.RestorationBucket
		thenBucket(manager.RootBucket(), func(b *services.RestorationBucket) { rootBucket = b })
		result.Complete(createEncodedRestorationData1())
		pump()
		expect(rootBucket != nil, "root bucket")
		*callsToEngine = nil

		// Schedule a frame.
		scheduler.SchedulerBindingInstance().EnsureVisualUpdate()
		rootBucket.Write("foo", 1)
		// flushData is no-op because frame is scheduled.
		manager.FlushData()
		expect(len(*callsToEngine) == 0, "calls %v", *callsToEngine)
		// Data is flushed at the end of the frame.
		pump()
		expect(len(*callsToEngine) == 1, "calls %v", *callsToEngine)
		*callsToEngine = nil

		// flushData without frame sends data directly.
		rootBucket.Write("foo", 2)
		manager.FlushData()
		expect(len(*callsToEngine) == 1, "calls %v", *callsToEngine)
		expect((*callsToEngine)[0].Method == "put", "put")
		postTestFrame()
	})
}

// receivingRestorationManager is restoration_test's
// TestRestorationManager (receiveDataFromEngine).
type receivingRestorationManager struct {
	services.RestorationManager
}

func TestRestorationManagerIsReplacing(t *testing.T) {
	ensureInitialized()
	var result *async.Completer[any] // created on the test's loop
	_, done := mockRestorationChannel(func() async.Future[any] { return result.Future() })
	defer done()
	runAsync(t, func() {
		result = async.NewCompleter[any]()
		manager := &receivingRestorationManager{}
		manager.Construct(manager)
		defer manager.Dispose()
		expect(!manager.IsReplacing(), "replacing")

		var rootBucket *services.RestorationBucket
		thenBucket(manager.RootBucket(), func(b *services.RestorationBucket) { rootBucket = b })
		result.Complete(createEncodedRestorationData1())
		idle()
		expect(rootBucket != nil && !rootBucket.IsReplacing() && !manager.IsReplacing(), "first data")
		b := ensureInitialized()
		b.ScheduleFrame()
		pump()
		expect(!manager.IsReplacing() && !rootBucket.IsReplacing(), "after frame")

		manager.HandleRestorationUpdateFromEngine(true, nil)
		var rootBucket2 *services.RestorationBucket
		thenBucket(manager.RootBucket(), func(b *services.RestorationBucket) { rootBucket2 = b })
		expect(rootBucket2 != nil && rootBucket2 != rootBucket, "new bucket")
		expect(manager.IsReplacing() && rootBucket2.IsReplacing(), "replacing")
		idle()
		expect(manager.IsReplacing() && rootBucket2.IsReplacing(), "replacing after idle")
		b.ScheduleFrame()
		pump()
		expect(!manager.IsReplacing() && !rootBucket2.IsReplacing(), "replaced after frame")

		manager.HandleRestorationUpdateFromEngine(false, nil)
		var rootBucket3 *services.RestorationBucket
		rootBucket3 = rootBucket2
		thenBucket(manager.RootBucket(), func(b *services.RestorationBucket) { rootBucket3 = b })
		expect(rootBucket3 == nil && !manager.IsReplacing(), "disabled")
		idle()
		expect(!manager.IsReplacing(), "disabled after idle")
		b.ScheduleFrame()
		pump()
		expect(!manager.IsReplacing(), "disabled after frame")
		postTestFrame()
	})
}

type notSerializable struct{ x int }

func TestDebugIsSerializableForRestoration(t *testing.T) {
	cases := []struct {
		value any
		want  bool
	}{
		{&struct{}{}, false},
		{notSerializable{1}, false},
		{nil, true},
		{147823, true},
		{12.43, true},
		{"Hello World", true},
		{[]int{12, 13, 14}, true},
		{map[string]int{"v1": 10, "v2": 23}, true},
		{map[string]any{"hello": []int{12, 12, 12}, "world": map[int]bool{1: true, 2: false, 4: true}}, true},
	}
	for _, c := range cases {
		if got := services.DebugIsSerializableForRestoration(c.value); got != c.want {
			t.Errorf("%#v: %v", c.value, got)
		}
	}
}

// postTestFrame is the frame testWidgets pumps after each test body (before
// the tear-downs): post-frame callbacks the test left scheduled run then.
func postTestFrame() {
	ensureInitialized().ScheduleFrame()
	pump()
}
