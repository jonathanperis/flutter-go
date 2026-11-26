package flutter_test

// Tests of TestRestorationManager and the binding's restoration manager
// (flutter_test's restoration.dart has no test file of its own upstream;
// these check what the restoration tests rely on).

import (
	"testing"

	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/services"
	"github.com/jonathanperis/flutter-go/src/async"
)

func syncBucket(t *testing.T, f async.Future[*services.RestorationBucket]) *services.RestorationBucket {
	t.Helper()
	var bucket *services.RestorationBucket
	called := false
	async.OnComplete(f, func(b *services.RestorationBucket) { bucket, called = b, true }, nil)
	if !called {
		t.Fatal("the root bucket is not synchronous")
	}
	return bucket
}

var firstManager *TestRestorationManager

func TestTestRestorationManagerRecordsAndRestores(t *testing.T) {
	tester := New(t)
	m := tester.Binding().TestRestorationManager()
	firstManager = m
	if tester.Binding().RestorationManager() != &m.RestorationManager {
		t.Fatal("RestorationManager isn't the test manager")
	}
	if m.DebugRootBucketAccessed() {
		t.Fatal("accessed before use")
	}
	root := syncBucket(t, m.RootBucket())
	if root == nil || !m.DebugRootBucketAccessed() || m.RestorationData().Binary() != nil {
		t.Fatal("empty data", root)
	}
	root.Write("a", 1)
	child := root.ClaimChild("c", nil)
	child.Write("b", "x")
	tester.Binding().ScheduleFrame()
	tester.Pump()
	data := m.RestorationData()
	if data.Binary() == nil {
		t.Fatal("nothing sent")
	}

	m.RestoreFrom(TestRestorationDataEmpty)
	empty := syncBucket(t, m.RootBucket())
	if empty == root || empty.Contains("a") {
		t.Fatal("restored empty", empty)
	}
	m.RestoreFrom(data)
	restored := syncBucket(t, m.RootBucket())
	restoredChild := restored.ClaimChild("c", nil)
	if restored.Read("a") != 1 || restoredChild.Read("b") != "x" {
		t.Fatal("restored", restored)
	}
	// (Disabling restoration with a claimed child would leave a
	// serialization scheduled for a null root, which throws in Flutter too.)
	restoredChild.Dispose()
	if !m.IsReplacing() {
		t.Fatal("not replacing after a restore")
	}
	tester.Binding().ScheduleFrame()
	tester.Pump()
	if m.IsReplacing() {
		t.Fatal("still replacing after a frame")
	}

	m.DisableRestoration()
	if syncBucket(t, m.RootBucket()) != nil || m.RestorationData().Binary() != nil {
		t.Fatal("not disabled")
	}
}

func TestTestRestorationManagerIsPerTest(t *testing.T) {
	tester := New(t)
	m := tester.Binding().TestRestorationManager()
	if m == firstManager && firstManager != nil {
		t.Fatal("the manager outlived its test")
	}
	if syncBucket(t, m.RootBucket()) == nil {
		t.Fatal("no root bucket")
	}
}
