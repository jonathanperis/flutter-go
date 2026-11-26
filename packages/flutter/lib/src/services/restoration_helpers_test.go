package services_test

// Port of packages/flutter/test/services/restoration.dart (Flutter 3.50.1):
// MockRestorationManager and the raw data keys; plus the memory event
// helper of leak_tracker's memoryEvents/areCreateAndDispose.

import (
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/scheduler"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/services"
	flutter_test "github.com/jonathanperis/flutter-go/packages/flutter_test/lib/src"
	"github.com/jonathanperis/flutter-go/src/async"
	"github.com/jonathanperis/flutter-go/src/dart"
	"time"
)

const (
	childrenMapKey = "c"
	valuesMapKey   = "v"
)

type mockRestorationManager struct {
	flutter_test.TestRestorationManager
	enableChannels  bool
	updateScheduled bool
	buckets         []*services.RestorationBucket

	rootBucketAccessed int
	rootBucket         async.Future[*services.RestorationBucket]
	isRestoring        bool
}

func newMockRestorationManager(enableChannels ...bool) *mockRestorationManager {
	m := &mockRestorationManager{enableChannels: len(enableChannels) > 0 && enableChannels[0]}
	m.Construct(m)
	return m
}

func (m *mockRestorationManager) InitChannels() {
	if m.enableChannels {
		m.TestRestorationManager.InitChannels()
	}
}

func (m *mockRestorationManager) ScheduleSerializationFor(bucket *services.RestorationBucket) {
	m.updateScheduled = true
	m.buckets = append(m.buckets, bucket)
}

// UnscheduleSerializationFor (Dart's override also returns whether the
// bucket was removed; the base method's Go signature has no result).
func (m *mockRestorationManager) UnscheduleSerializationFor(bucket *services.RestorationBucket) {
	m.updateScheduled = true
	for i, b := range m.buckets {
		if b == bucket {
			m.buckets = append(m.buckets[:i:i], m.buckets[i+1:]...)
			return
		}
	}
}

func (m *mockRestorationManager) doSerialization() {
	m.updateScheduled = false
	for _, bucket := range m.buckets {
		bucket.Finalize()
	}
	m.buckets = nil
}

func (m *mockRestorationManager) RestoreFrom(flutter_test.TestRestorationData) {
	// Ignore in mock.
}

func (m *mockRestorationManager) RootBucket() async.Future[*services.RestorationBucket] {
	m.rootBucketAccessed++
	return m.rootBucket
}

func (m *mockRestorationManager) setRootBucket(value async.Future[*services.RestorationBucket]) {
	m.rootBucket = value
	m.isRestoring = true
	scheduler.SchedulerBindingInstance().AddPostFrameCallback(func(time.Duration) { m.isRestoring = false })
	m.NotifyListeners()
}

func (m *mockRestorationManager) IsReplacing() bool { return m.isRestoring }

func (m *mockRestorationManager) SendToEngine([]byte) async.Future[async.Void] {
	panic(dart.UnimplementedError{Message: "unimplemented in mock"})
}

func (m *mockRestorationManager) String() string { return "MockManager" }

// createRawDataSet is _createRawDataSet.
func createRawDataSet() *dart.LinkedMap {
	return dart.NewLinkedMap(
		dart.MapEntry[any, any]{Key: valuesMapKey, Value: dart.NewLinkedMap(
			dart.MapEntry[any, any]{Key: "value1", Value: 10}, dart.MapEntry[any, any]{Key: "value2", Value: "Hello"})},
		dart.MapEntry[any, any]{Key: childrenMapKey, Value: dart.NewLinkedMap(
			dart.MapEntry[any, any]{Key: "child1", Value: dart.NewLinkedMap(
				dart.MapEntry[any, any]{Key: valuesMapKey, Value: dart.NewLinkedMap(dart.MapEntry[any, any]{Key: "foo", Value: 22})})})},
	)
}

// sub is `(m[key] as Map)` for raw restoration data.
func sub(m *dart.LinkedMap, keys ...string) *dart.LinkedMap {
	for _, k := range keys {
		m = m.Lookup(k).(*dart.LinkedMap)
	}
	return m
}

// memoryEvents runs body and returns the created/disposed events
// dispatched for objects whose type is typeName (leak_tracker's
// memoryEvents).
func memoryEvents(body func(), typeName string) []foundation.ObjectEvent {
	var events []foundation.ObjectEvent
	listener := func(event foundation.ObjectEvent) {
		if foundation.ObjectRuntimeType(event.Object(), "") == typeName {
			events = append(events, event)
		}
	}
	foundation.FlutterMemoryAllocationsInstance.AddListener(listener)
	defer foundation.FlutterMemoryAllocationsInstance.RemoveListener(listener)
	body()
	return events
}

// areCreateAndDispose is leak_tracker's matcher: one creation, then one
// disposal.
func areCreateAndDispose(events []foundation.ObjectEvent) bool {
	if len(events) != 2 {
		return false
	}
	_, created := events[0].(*foundation.ObjectCreated)
	_, disposed := events[1].(*foundation.ObjectDisposed)
	return created && disposed
}
