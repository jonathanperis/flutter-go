package services_test

// Port of packages/flutter/test/services/restoration_bucket_test.dart
// (Flutter 3.50.1), 24/24. The raw data maps are *dart.LinkedMap (Dart's
// maps, mutated in place by the buckets); `read<P>` is Read (or the typed
// RestorationBucketRead).

import (
	"errors"
	"testing"

	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/services"
	"github.com/jonathanperis/flutter-go/src/dart"
)

func check(t *testing.T, cond bool, format string, args ...any) {
	t.Helper()
	if !cond {
		t.Fatalf(format, args...)
	}
}

// throwsFlutterError runs f and reports whether it panicked with a
// FlutterError (returned).
func throwsFlutterError(f func()) (err *foundation.FlutterError) {
	defer func() {
		if r := recover(); r != nil {
			if e, ok := r.(error); ok {
				errors.As(e, &err)
			}
		}
	}()
	f()
	return nil
}

func TestRestorationRootBucketValues(t *testing.T) {
	ensureInitialized()
	manager := newMockRestorationManager()
	rawData := createRawDataSet()
	bucket := services.NewRestorationBucketRoot(&manager.RestorationManager, rawData)

	check(t, bucket.RestorationID() == "root", "id")
	check(t, bucket.DebugOwner() == any(manager), "owner %v", bucket.DebugOwner())

	// Bucket contains expected values from rawData.
	check(t, services.RestorationBucketRead[int](bucket, "value1") == 10, "value1")
	check(t, services.RestorationBucketRead[string](bucket, "value2") == "Hello", "value2")
	check(t, bucket.Read("value3") == nil, "value3")
	check(t, !manager.updateScheduled, "scheduled")

	// Can overwrite existing value.
	bucket.Write("value1", 22)
	check(t, manager.updateScheduled, "scheduled")
	check(t, bucket.Read("value1") == 22, "value1")
	manager.doSerialization()
	check(t, sub(rawData, valuesMapKey).Lookup("value1") == 22, "raw value1")
	check(t, !manager.updateScheduled, "scheduled")

	// Writing the same value is a no-op (1 == 1.0 in Dart, too).
	bucket.Write("value1", 22.0)
	check(t, !manager.updateScheduled, "same value scheduled")

	// Can add a new value.
	bucket.Write("value3", true)
	check(t, manager.updateScheduled, "scheduled")
	check(t, bucket.Read("value3") == true, "value3")
	manager.doSerialization()
	check(t, sub(rawData, valuesMapKey).Lookup("value3") == true, "raw value3")
	check(t, !manager.updateScheduled, "scheduled")

	// Can remove existing value.
	check(t, bucket.Remove("value1") == 22, "remove")
	check(t, manager.updateScheduled, "scheduled")
	check(t, bucket.Read("value1") == nil, "value1 removed")
	manager.doSerialization()
	check(t, !sub(rawData, valuesMapKey).ContainsKey("value1"), "raw value1 removed")
	check(t, !manager.updateScheduled, "scheduled")

	// Removing non-existing value is no-op.
	check(t, bucket.Remove("value4") == nil, "remove value4")
	check(t, !manager.updateScheduled, "scheduled")

	// Can store null.
	bucket.Write("value4", nil)
	check(t, manager.updateScheduled, "scheduled")
	check(t, bucket.Read("value4") == nil, "value4")
	manager.doSerialization()
	check(t, sub(rawData, valuesMapKey).ContainsKey("value4"), "raw value4")
	check(t, sub(rawData, valuesMapKey).Lookup("value4") == nil, "raw value4 null")
	check(t, !manager.updateScheduled, "scheduled")
}

func TestRestorationChildBucketValues(t *testing.T) {
	ensureInitialized()
	manager := newMockRestorationManager()
	rootRawData := createRawDataSet()
	debugOwner := new(int)
	root := services.NewRestorationBucketRoot(&manager.RestorationManager, rootRawData)
	child := services.NewRestorationBucketChild("child1", root, debugOwner)

	check(t, child.RestorationID() == "child1", "id")
	check(t, child.DebugOwner() == any(debugOwner), "owner")
	check(t, child.Read("foo") == 22, "foo")
	check(t, child.Read("bar") == nil, "bar")
	check(t, !manager.updateScheduled, "scheduled")

	child.Write("foo", 44)
	check(t, manager.updateScheduled, "scheduled")
	check(t, child.Read("foo") == 44, "foo")
	manager.doSerialization()
	check(t, sub(rootRawData, childrenMapKey, "child1", valuesMapKey).Lookup("foo") == 44, "raw foo")
	check(t, !manager.updateScheduled, "scheduled")

	child.Write("value3", true)
	check(t, manager.updateScheduled, "scheduled")
	check(t, child.Read("value3") == true, "value3")
	manager.doSerialization()
	check(t, sub(rootRawData, childrenMapKey, "child1", valuesMapKey).Lookup("value3") == true, "raw value3")
	check(t, !manager.updateScheduled, "scheduled")

	check(t, child.Remove("foo") == 44, "remove")
	check(t, manager.updateScheduled, "scheduled")
	check(t, child.Read("foo") == nil, "foo removed")
	manager.doSerialization()
	check(t, !sub(rootRawData, childrenMapKey, "child1").ContainsKey("foo"), "raw foo removed")
	check(t, !manager.updateScheduled, "scheduled")

	check(t, child.Remove("value4") == nil, "remove value4")
	check(t, !manager.updateScheduled, "scheduled")

	child.Write("value4", nil)
	check(t, manager.updateScheduled, "scheduled")
	check(t, child.Read("value4") == nil, "value4")
	manager.doSerialization()
	check(t, sub(rootRawData, childrenMapKey, "child1", valuesMapKey).ContainsKey("value4"), "raw value4")
	check(t, sub(rootRawData, childrenMapKey, "child1", valuesMapKey).Lookup("value4") == nil, "raw value4 null")
	check(t, !manager.updateScheduled, "scheduled")
}

func TestRestorationClaimChildWithExistingData(t *testing.T) {
	ensureInitialized()
	manager := newMockRestorationManager()
	rawData := createRawDataSet()
	bucket := services.NewRestorationBucketRoot(&manager.RestorationManager, rawData)
	debugOwner := new(int)
	child := bucket.ClaimChild("child1", debugOwner)
	check(t, !manager.updateScheduled, "scheduled")
	check(t, child.RestorationID() == "child1" && child.DebugOwner() == any(debugOwner), "child")
	check(t, child.Read("foo") == 22, "foo")
	child.Write("bar", 44)
	check(t, manager.updateScheduled, "scheduled")
	manager.doSerialization()
	check(t, sub(rawData, childrenMapKey, "child1", valuesMapKey).Lookup("bar") == 44, "raw bar")
	check(t, !manager.updateScheduled, "scheduled")
}

func TestRestorationClaimChildWithNoExistingData(t *testing.T) {
	ensureInitialized()
	manager := newMockRestorationManager()
	rawData := createRawDataSet()
	bucket := services.NewRestorationBucketRoot(&manager.RestorationManager, rawData)
	check(t, !sub(rawData, childrenMapKey).ContainsKey("child2"), "child2")
	debugOwner := new(int)
	child := bucket.ClaimChild("child2", debugOwner)
	check(t, manager.updateScheduled, "scheduled")
	check(t, child.RestorationID() == "child2" && child.DebugOwner() == any(debugOwner), "child")
	child.Write("foo", 55)
	check(t, child.Read("foo") == 55, "foo")
	manager.doSerialization()
	check(t, !manager.updateScheduled, "scheduled")
	check(t, sub(rawData, childrenMapKey).ContainsKey("child2"), "child2")
	check(t, sub(rawData, childrenMapKey, "child2", valuesMapKey).Lookup("foo") == 55, "raw foo")
}

func TestRestorationClaimClaimedChildThrowsIfNotGivenUp(t *testing.T) {
	ensureInitialized()
	manager := newMockRestorationManager()
	bucket := services.NewRestorationBucketRoot(&manager.RestorationManager, createRawDataSet())
	child1 := bucket.ClaimChild("child1", "FirstClaim")
	check(t, !manager.updateScheduled, "scheduled")
	check(t, child1.RestorationID() == "child1" && child1.Read("foo") == 22, "child1")
	manager.doSerialization()
	check(t, !manager.updateScheduled, "scheduled")

	child2 := bucket.ClaimChild("child1", "SecondClaim")
	check(t, child2.RestorationID() == "child1" && child2.Read("foo") == nil, "child2")

	// child1 is not given up before running finalizers.
	err := throwsFlutterError(manager.doSerialization)
	check(t, err != nil, "no FlutterError")
	want := "Multiple owners claimed child RestorationBuckets with the same IDs.\n" +
		"The following IDs were claimed multiple times from the parent RestorationBucket(restorationId: root, owner: MockManager):\n" +
		" * \"child1\" was claimed by:\n" +
		"   * SecondClaim\n" +
		"   * FirstClaim (current owner)"
	check(t, err.Message() == want, "message %q", err.Message())
}

func TestRestorationClaimClaimedChildGivenUp(t *testing.T) {
	ensureInitialized()
	manager := newMockRestorationManager()
	rawData := createRawDataSet()
	bucket := services.NewRestorationBucketRoot(&manager.RestorationManager, rawData)
	child1 := bucket.ClaimChild("child1", "FirstClaim")
	check(t, child1.Read("foo") == 22, "child1")
	manager.doSerialization()
	child2 := bucket.ClaimChild("child1", "SecondClaim")
	check(t, child2.RestorationID() == "child1" && child2.Read("foo") == nil, "child2")
	child2.Write("bar", 55)
	// give up child1.
	child1.Dispose()
	manager.doSerialization()
	check(t, !manager.updateScheduled, "scheduled")
	check(t, !sub(rawData, childrenMapKey, "child1", valuesMapKey).ContainsKey("foo"), "foo")
	check(t, sub(rawData, childrenMapKey, "child1", valuesMapKey).Lookup("bar") == 55, "bar")
}

func TestRestorationClaimTwiceGiveUpOnceThrows(t *testing.T) {
	ensureInitialized()
	manager := newMockRestorationManager()
	bucket := services.NewRestorationBucketRoot(&manager.RestorationManager, createRawDataSet())
	child1 := bucket.ClaimChild("child1", "FirstClaim")
	check(t, child1.RestorationID() == "child1", "child1")
	child2 := bucket.ClaimChild("child1", "SecondClaim")
	check(t, child2.RestorationID() == "child1", "child2")
	child1.Dispose()
	child3 := bucket.ClaimChild("child1", "ThirdClaim")
	check(t, child3.RestorationID() == "child1", "child3")
	check(t, manager.updateScheduled, "scheduled")
	check(t, throwsFlutterError(manager.doSerialization) != nil, "no FlutterError")
}

func TestRestorationUnclaimAndClaimGivesFreshBucket(t *testing.T) {
	ensureInitialized()
	manager := newMockRestorationManager()
	bucket := services.NewRestorationBucketRoot(&manager.RestorationManager, createRawDataSet())
	child1 := bucket.ClaimChild("child1", "FirstClaim")
	check(t, !manager.updateScheduled && child1.Read("foo") == 22, "child1")
	child1.Dispose()
	check(t, manager.updateScheduled, "scheduled")
	child2 := bucket.ClaimChild("child1", "SecondClaim")
	check(t, child2.Read("foo") == nil, "child2")
}

func TestRestorationCleansUpRawData(t *testing.T) {
	ensureInitialized()
	manager := newMockRestorationManager()
	rawData := createRawDataSet()
	root := services.NewRestorationBucketRoot(&manager.RestorationManager, rawData)
	check(t, rawData.ContainsKey(childrenMapKey), "children")
	child := root.ClaimChild("child1", "owner")
	child.Dispose()
	check(t, manager.updateScheduled, "scheduled")
	manager.doSerialization()
	check(t, !rawData.ContainsKey(childrenMapKey), "children left")

	check(t, rawData.ContainsKey(valuesMapKey), "values")
	check(t, root.Remove("value1") == 10, "value1")
	check(t, root.Remove("value2") == "Hello", "value2")
	check(t, manager.updateScheduled, "scheduled")
	manager.doSerialization()
	check(t, !rawData.ContainsKey(valuesMapKey), "values left")
}

func TestRestorationDisposeDeletesData(t *testing.T) {
	ensureInitialized()
	manager := newMockRestorationManager()
	rawData := createRawDataSet()
	root := services.NewRestorationBucketRoot(&manager.RestorationManager, rawData)
	child1 := root.ClaimChild("child1", "owner1")
	child1.ClaimChild("child1OfChild1", "owner1.1")
	child1.ClaimChild("child2OfChild1", "owner1.2")
	child2 := root.ClaimChild("child2", "owner2")
	check(t, manager.updateScheduled, "scheduled")
	manager.doSerialization()
	check(t, !manager.updateScheduled, "scheduled")
	check(t, sub(rawData, childrenMapKey).ContainsKey("child1") && sub(rawData, childrenMapKey).ContainsKey("child2"), "children")

	child1.Dispose()
	check(t, manager.updateScheduled, "scheduled")
	manager.doSerialization()
	check(t, !manager.updateScheduled, "scheduled")
	check(t, !sub(rawData, childrenMapKey).ContainsKey("child1"), "child1 left")

	child2.Dispose()
	check(t, manager.updateScheduled, "scheduled")
	manager.doSerialization()
	check(t, !manager.updateScheduled, "scheduled")
	check(t, !rawData.ContainsKey(childrenMapKey), "children left")
}

func TestRestorationRenameSameID(t *testing.T) {
	ensureInitialized()
	manager := newMockRestorationManager()
	rawData := createRawDataSet()
	root := services.NewRestorationBucketRoot(&manager.RestorationManager, rawData)
	child := root.ClaimChild("child1", "owner1")
	check(t, !manager.updateScheduled && child.RestorationID() == "child1", "child")
	child.Rename("child1")
	check(t, !manager.updateScheduled && child.RestorationID() == "child1", "renamed")
	check(t, sub(rawData, childrenMapKey).ContainsKey("child1"), "child1")
}

func TestRestorationRenameToUnusedID(t *testing.T) {
	ensureInitialized()
	manager := newMockRestorationManager()
	rawData := createRawDataSet()
	root := services.NewRestorationBucketRoot(&manager.RestorationManager, rawData)
	child := root.ClaimChild("child1", "owner1")
	rawChildData := sub(rawData, childrenMapKey).Lookup("child1")
	check(t, rawChildData != nil, "raw child")
	check(t, !manager.updateScheduled && child.RestorationID() == "child1", "child")
	child.Rename("new-name")
	check(t, child.RestorationID() == "new-name", "renamed")
	check(t, manager.updateScheduled, "scheduled")
	manager.doSerialization()
	check(t, !manager.updateScheduled, "scheduled")
	check(t, !sub(rawData, childrenMapKey).ContainsKey("child1"), "child1")
	check(t, sub(rawData, childrenMapKey).Lookup("new-name") == rawChildData, "new-name")
}

func TestRestorationRenameToUsedIDThrows(t *testing.T) {
	ensureInitialized()
	manager := newMockRestorationManager()
	root := services.NewRestorationBucketRoot(&manager.RestorationManager, createRawDataSet())
	child1 := root.ClaimChild("child1", "owner1")
	child2 := root.ClaimChild("child2", "owner1")
	manager.doSerialization()
	check(t, child1.RestorationID() == "child1" && child2.RestorationID() == "child2", "ids")
	child2.Rename("child1")
	check(t, child2.RestorationID() == "child1", "renamed")
	check(t, manager.updateScheduled, "scheduled")
	check(t, throwsFlutterError(manager.doSerialization) != nil, "no FlutterError")
}

func TestRestorationRenameToUsedIDGivenUp(t *testing.T) {
	ensureInitialized()
	manager := newMockRestorationManager()
	rawData := createRawDataSet()
	root := services.NewRestorationBucketRoot(&manager.RestorationManager, rawData)
	child1 := root.ClaimChild("child1", "owner1")
	child2 := root.ClaimChild("child2", "owner1")
	manager.doSerialization()
	rawChild1Data := sub(rawData, childrenMapKey).Lookup("child1")
	rawChild2Data := sub(rawData, childrenMapKey).Lookup("child2")
	check(t, rawChild1Data != nil && rawChild2Data != nil, "raw")
	child2.Rename("child1")
	check(t, child2.RestorationID() == "child1" && child1.RestorationID() == "child1", "ids")
	child1.Dispose()
	check(t, manager.updateScheduled, "scheduled")
	manager.doSerialization()
	check(t, !manager.updateScheduled, "scheduled")
	check(t, sub(rawData, childrenMapKey).Lookup("child1") == rawChild2Data, "child1 data")
	check(t, !sub(rawData, childrenMapKey).ContainsKey("child2"), "child2")
}

func TestRestorationRenameToBeAddedChild(t *testing.T) {
	ensureInitialized()
	manager := newMockRestorationManager()
	rawData := createRawDataSet()
	root := services.NewRestorationBucketRoot(&manager.RestorationManager, rawData)
	rawChild1Data := sub(rawData, childrenMapKey).Lookup("child1")
	check(t, rawChild1Data != nil, "raw")
	child1 := root.ClaimChild("child1", "owner1")
	child2 := root.ClaimChild("child1", "owner1")
	child2.Rename("foo")
	check(t, manager.updateScheduled, "scheduled")
	manager.doSerialization()
	check(t, !manager.updateScheduled, "scheduled")
	check(t, child1.RestorationID() == "child1" && child2.RestorationID() == "foo", "ids")
	check(t, sub(rawData, childrenMapKey).Lookup("child1") == rawChild1Data, "child1")
	check(t, sub(rawData, childrenMapKey, "foo").IsEmpty(), "foo") // new bucket
}

func TestRestorationAdoptSameParent(t *testing.T) {
	ensureInitialized()
	manager := newMockRestorationManager()
	rawData := createRawDataSet()
	root := services.NewRestorationBucketRoot(&manager.RestorationManager, rawData)
	child1 := root.ClaimChild("child1", "owner1")
	root.AdoptChild(child1)
	check(t, !manager.updateScheduled, "scheduled")
	check(t, sub(rawData, childrenMapKey).ContainsKey("child1"), "child1")
}

func TestRestorationAdoptFreshChild(t *testing.T) {
	ensureInitialized()
	manager := newMockRestorationManager()
	rawData := createRawDataSet()
	root := services.NewRestorationBucketRoot(&manager.RestorationManager, rawData)
	child := services.NewRestorationBucketEmpty("fresh-child", "owner1")
	root.AdoptChild(child)
	check(t, manager.updateScheduled, "scheduled")
	child.Write("value", 22)
	manager.doSerialization()
	check(t, !manager.updateScheduled, "scheduled")
	check(t, sub(rawData, childrenMapKey).ContainsKey("fresh-child"), "fresh-child")
	check(t, sub(rawData, childrenMapKey, "fresh-child", valuesMapKey).Lookup("value") == 22, "value")
	child.Write("bar", "blabla")
	check(t, manager.updateScheduled, "scheduled")
}

func TestRestorationAdoptChildWithParent(t *testing.T) {
	ensureInitialized()
	manager := newMockRestorationManager()
	rawData := createRawDataSet()
	root := services.NewRestorationBucketRoot(&manager.RestorationManager, rawData)
	child := root.ClaimChild("child1", "owner1")
	childOfChild := child.ClaimChild("childOfChild", "owner2")
	childOfChild.Write("foo", "bar")
	check(t, manager.updateScheduled, "scheduled")
	manager.doSerialization()
	check(t, !manager.updateScheduled, "scheduled")
	childOfChildData := sub(rawData, childrenMapKey, "child1", childrenMapKey, "childOfChild")
	check(t, !childOfChildData.IsEmpty(), "data")
	root.AdoptChild(childOfChild)
	check(t, manager.updateScheduled, "scheduled")
	manager.doSerialization()
	check(t, !manager.updateScheduled, "scheduled")
	check(t, !sub(rawData, childrenMapKey, "child1").ContainsKey(childrenMapKey), "child1 has children")
	check(t, sub(rawData, childrenMapKey).Lookup("childOfChild") == childOfChildData, "moved")
}

func TestRestorationAdoptUsedIDThrows(t *testing.T) {
	ensureInitialized()
	manager := newMockRestorationManager()
	root := services.NewRestorationBucketRoot(&manager.RestorationManager, createRawDataSet())
	child := root.ClaimChild("child1", "owner1")
	childOfChild := child.ClaimChild("child1", "owner2")
	childOfChild.Write("foo", "bar")
	root.AdoptChild(childOfChild)
	check(t, manager.updateScheduled, "scheduled")
	check(t, throwsFlutterError(manager.doSerialization) != nil, "no FlutterError")
}

func TestRestorationAdoptUsedIDGivenUp(t *testing.T) {
	ensureInitialized()
	manager := newMockRestorationManager()
	rawData := createRawDataSet()
	root := services.NewRestorationBucketRoot(&manager.RestorationManager, rawData)
	child := root.ClaimChild("child1", "owner1")
	childOfChild := child.ClaimChild("child1", "owner2")
	childOfChild.Write("foo", "bar")
	childOfChildData := sub(rawData, childrenMapKey, "child1", childrenMapKey, "child1")
	check(t, !childOfChildData.IsEmpty(), "data")
	check(t, manager.updateScheduled, "scheduled")
	manager.doSerialization()
	check(t, !manager.updateScheduled, "scheduled")
	root.AdoptChild(childOfChild)
	check(t, manager.updateScheduled, "scheduled")
	child.Dispose()
	manager.doSerialization()
	check(t, !manager.updateScheduled, "scheduled")
	check(t, sub(rawData, childrenMapKey).Lookup("child1") == childOfChildData, "moved")
}

func TestRestorationAdoptToBeAddedChildUnderUsedID(t *testing.T) {
	ensureInitialized()
	manager := newMockRestorationManager()
	rawData := createRawDataSet()
	root := services.NewRestorationBucketRoot(&manager.RestorationManager, rawData)
	child1 := root.ClaimChild("child1", "owner1")
	child2 := root.ClaimChild("child2", "owner1")
	check(t, manager.updateScheduled, "scheduled")
	manager.doSerialization()
	check(t, !manager.updateScheduled, "scheduled")

	child1OfChild1 := child1.ClaimChild("child2", "owner2")
	child1OfChild1.Write("hello", "world")
	child2OfChild1 := child1.ClaimChild("child2", "owner2")
	child2OfChild1.Write("foo", "bar")

	root.AdoptChild(child2OfChild1)
	child2.Dispose()
	check(t, manager.updateScheduled, "scheduled")
	manager.doSerialization()
	check(t, !manager.updateScheduled, "scheduled")
	check(t, sub(rawData, childrenMapKey, "child2", valuesMapKey).Lookup("foo") == "bar", "foo")
	check(t, sub(rawData, childrenMapKey, "child1", childrenMapKey, "child2", valuesMapKey).Lookup("hello") == "world", "hello")
}

func TestRestorationBucketThrowsWhenUsedAfterDispose(t *testing.T) {
	bucket := services.NewRestorationBucketEmpty("foo", nil)
	bucket.Dispose()
	child := services.NewRestorationBucketEmpty("child", nil)
	for name, f := range map[string]func(){
		"debugOwner":    func() { bucket.DebugOwner() },
		"restorationId": func() { bucket.RestorationID() },
		"read":          func() { bucket.Read("foo") },
		"write":         func() { bucket.Write("foo", 10) },
		"remove":        func() { bucket.Remove("foo") },
		"contains":      func() { bucket.Contains("foo") },
		"claimChild":    func() { bucket.ClaimChild("child", nil) },
		"adoptChild":    func() { bucket.AdoptChild(child) },
		"rename":        func() { bucket.Rename("bar") },
		"dispose":       func() { bucket.Dispose() },
	} {
		if throwsFlutterError(f) == nil {
			t.Errorf("%s after dispose doesn't throw a FlutterError", name)
		}
	}
}

func TestRestorationBucketDispatchesMemoryEvents(t *testing.T) {
	ensureInitialized()
	if !areCreateAndDispose(memoryEvents(func() { services.NewRestorationBucketEmpty("child1", nil).Dispose() }, "RestorationBucket")) {
		t.Error("empty")
	}
	manager1 := newMockRestorationManager()
	defer manager1.Dispose()
	if !areCreateAndDispose(memoryEvents(func() { services.NewRestorationBucketRoot(&manager1.RestorationManager, nil).Dispose() }, "RestorationBucket")) {
		t.Error("root")
	}
	manager2 := newMockRestorationManager()
	defer manager2.Dispose()
	parent := services.NewRestorationBucketRoot(&manager2.RestorationManager, createRawDataSet())
	defer parent.Dispose()
	if !areCreateAndDispose(memoryEvents(func() { services.NewRestorationBucketChild("child1", parent, nil).Dispose() }, "RestorationBucket")) {
		t.Error("child")
	}
}

var _ = dart.NewLinkedMap
