package flutter_test

// Port of packages/flutter_test/lib/src/restoration.dart (Flutter
// 3.50.1): TestRestorationManager and TestRestorationData.

import (
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/services"
	"github.com/jonathanperis/flutter-go/src/async"
)

// TestRestorationManagerOverridable is what a type extending
// TestRestorationManager may override (RestorationManager's members and
// RestoreFrom).
type TestRestorationManagerOverridable interface {
	services.RestorationManagerOverridable
	RestoreFrom(data TestRestorationData)
}

// TestRestorationManager is the RestorationManager used in tests: it
// doesn't send data to the engine; it records it (RestorationData) and can
// restore from it (RestoreFrom). It is extended by embedding: call
// Construct(self).
type TestRestorationManager struct {
	services.RestorationManager
	restorationData         TestRestorationData
	debugRootBucketAccessed bool
}

// NewTestRestorationManager is TestRestorationManager().
func NewTestRestorationManager() *TestRestorationManager {
	m := &TestRestorationManager{}
	m.Construct(m)
	return m
}

// Construct is the constructor for a type extending
// TestRestorationManager: RestorationManager's, then RestoreFrom(empty)
// through self, so that RootBucket always returns a synchronous future (to
// avoid extra pumps in tests).
func (m *TestRestorationManager) Construct(self TestRestorationManagerOverridable) {
	m.RestorationManager.Construct(self)
	self.RestoreFrom(TestRestorationDataEmpty)
}

// RootBucket records that the root bucket was accessed.
func (m *TestRestorationManager) RootBucket() async.Future[*services.RestorationBucket] {
	m.debugRootBucketAccessed = true
	return m.RestorationManager.RootBucket()
}

// RestorationData is the restoration data the manager last sent to the
// engine (or restored from).
func (m *TestRestorationManager) RestorationData() TestRestorationData { return m.restorationData }

// DebugRootBucketAccessed is whether RootBucket was accessed since the
// manager was created.
func (m *TestRestorationManager) DebugRootBucketAccessed() bool { return m.debugRootBucketAccessed }

// RestoreFrom restores the state from data, as if the engine had sent it.
func (m *TestRestorationManager) RestoreFrom(data TestRestorationData) {
	m.restorationData = data
	m.HandleRestorationUpdateFromEngine(true, data.Binary())
}

// DisableRestoration disables state restoration (as if the engine had
// turned it off).
func (m *TestRestorationManager) DisableRestoration() {
	m.restorationData = TestRestorationDataEmpty
	m.HandleRestorationUpdateFromEngine(false, nil)
}

// SendToEngine records the data instead of sending it.
func (m *TestRestorationManager) SendToEngine(encodedData []byte) async.Future[async.Void] {
	return async.AsyncVoid(func() {
		m.restorationData = TestRestorationData{binary: encodedData}
	})
}

// TestRestorationData is the restoration data used in tests (opaque).
type TestRestorationData struct {
	binary []byte
}

// TestRestorationDataEmpty is TestRestorationData.empty: no data.
var TestRestorationDataEmpty = TestRestorationData{}

// Binary is the serialized data (@protected; nil for empty data).
func (d TestRestorationData) Binary() []byte { return d.binary }
