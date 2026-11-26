package services

// Port of restoration.dart (Flutter 3.50.1).
//
// Restoration data is a tree of Dart maps whose insertion order reaches the
// engine (the StandardMessageCodec writes a map in its order), so the raw
// data is *dart.LinkedMap (Dart's LinkedHashMap): the data decoded from the
// engine, the buckets' "v"/"c" maps, and every map value restored from the
// engine. Values written by the app are kept as given.

import (
	"reflect"
	"time"

	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/scheduler"
	"github.com/jonathanperis/flutter-go/src/async"
	"github.com/jonathanperis/flutter-go/src/dart"
)

// orderedStandardMessageCodec is the StandardMessageCodec reading maps as
// Dart's do: *dart.LinkedMap in the encoded order (a key equal to an
// earlier one replaces its value and keeps its place); it writes a
// *dart.LinkedMap as a map in its order.
type orderedStandardMessageCodec struct {
	StandardMessageCodec
}

func newOrderedStandardMessageCodec() *orderedStandardMessageCodec {
	c := &orderedStandardMessageCodec{}
	c.SetSelf(c)
	return c
}

func (c *orderedStandardMessageCodec) WriteValue(buffer *foundation.WriteBuffer, value any) {
	if m, ok := value.(*dart.LinkedMap); ok && m != nil {
		buffer.PutUint8(valueMap)
		c.WriteSize(buffer, m.Len())
		for _, e := range m.Entries() {
			c.WriteValue(buffer, e.Key)
			c.WriteValue(buffer, e.Value)
		}
		return
	}
	c.StandardMessageCodec.WriteValue(buffer, value)
}

func (c *orderedStandardMessageCodec) ReadValueOfType(typ int, buffer *foundation.ReadBuffer) any {
	if typ == valueMap {
		length := c.ReadSize(buffer)
		result := &dart.LinkedMap{}
		for range length {
			key := c.ReadValue(buffer)
			result.Set(key, c.ReadValue(buffer))
		}
		return result
	}
	return c.StandardMessageCodec.ReadValueOfType(typ, buffer)
}

// RestorationManagerOverridable is what a type extending
// RestorationManager may override (it calls Construct with itself).
type RestorationManagerOverridable interface {
	foundation.ChangeNotifier
	InitChannels()
	RootBucket() async.Future[*RestorationBucket]
	IsReplacing() bool
	HandleRestorationUpdateFromEngine(enabled bool, data []byte)
	SendToEngine(encodedData []byte) async.Future[async.Void]
	ScheduleSerializationFor(bucket *RestorationBucket)
	UnscheduleSerializationFor(bucket *RestorationBucket)
	FlushData()
	Dispose()
	String() string
	restorationManager() *RestorationManager
}

// RestorationManager manages the restoration data in the framework and
// synchronizes it with the engine. It is a ChangeNotifier that notifies
// when the root bucket is replaced. It is used by pointer; a type extending
// it embeds it and calls Construct(self) (Dart's super constructor, which
// calls InitChannels through self).
type RestorationManager struct {
	foundation.ChangeNotifierMixin
	self RestorationManagerOverridable

	rootBucket        *RestorationBucket // nil: restoration is turned off
	pendingRootBucket *async.Completer[*RestorationBucket]
	rootBucketIsValid bool
	isReplacing       bool

	debugDoingUpdate            bool
	serializationScheduled      bool
	bucketsNeedingSerialization []*RestorationBucket // a LinkedHashSet
}

// NewRestorationManager is RestorationManager().
func NewRestorationManager() *RestorationManager {
	m := &RestorationManager{}
	m.Construct(m)
	return m
}

// Construct is the RestorationManager constructor for a type extending
// it: it points the manager (and its ChangeNotifier) at self, dispatches
// the creation event and calls self.InitChannels().
func (m *RestorationManager) Construct(self RestorationManagerOverridable) {
	m.self = self
	m.ChangeNotifierMixin.SetSelf(self)
	if foundation.KFlutterMemoryAllocationsEnabled {
		foundation.ChangeNotifierMaybeDispatchObjectCreation(self)
	}
	self.InitChannels()
}

func (m *RestorationManager) restorationManager() *RestorationManager { return m }

func (m *RestorationManager) outer() RestorationManagerOverridable {
	if m.self == nil {
		panic("services: RestorationManager used before Construct; create it with NewRestorationManager or call Construct")
	}
	return m.self
}

// InitChannels sets up the method call handler for
// SystemChannels.Restoration (@protected; tests override it).
func (m *RestorationManager) InitChannels() {
	SystemChannels.Restoration.SetMethodCallHandler(m.methodHandler)
}

// RootBucket is the root of the RestorationBucket hierarchy containing the
// restoration data; the future completes with nil when restoration is
// disabled (synchronously, as a SynchronousFuture, once known).
func (m *RestorationManager) RootBucket() async.Future[*RestorationBucket] {
	if m.rootBucketIsValid {
		return foundation.NewSynchronousFuture(m.rootBucket)
	}
	if m.pendingRootBucket == nil {
		m.pendingRootBucket = async.NewCompleter[*RestorationBucket]()
		m.getRootBucketFromEngine()
	}
	return m.pendingRootBucket.Future()
}

// IsReplacing is true while the restoration data is being replaced by new
// data from the engine (until the end of the next frame).
func (m *RestorationManager) IsReplacing() bool { return m.isReplacing }

func (m *RestorationManager) getRootBucketFromEngine() async.Future[async.Void] {
	return async.AsyncVoid(func() {
		config := SystemChannels.Restoration.InvokeMethod("get").Await()
		if m.pendingRootBucket == nil {
			// The restoration data was obtained via other means (e.g. by
			// calling handleRestorationDataUpdate while the request to the
			// engine was outstanding. Ignore the engine's response.
			return
		}
		if m.rootBucket != nil {
			panic(dart.NewAssertionError(nil))
		}
		m.parseAndHandleRestorationUpdateFromEngine(config)
	})
}

func (m *RestorationManager) parseAndHandleRestorationUpdateFromEngine(update any) {
	if update == nil {
		m.outer().HandleRestorationUpdateFromEngine(false, nil)
		return
	}
	enabled, ok := MapLookup(update, "enabled")
	if !ok || enabled == nil {
		if !isMap(update) {
			panic(dart.TypeError{Message: "type '" + foundation.ObjectRuntimeType(update, "Object") +
				"' is not a subtype of type 'Map<Object?, Object?>?' in type cast"})
		}
		panic(dart.TypeError{Message: "Null check operator used on a null value"})
	}
	data, _ := MapLookup(update, "data")
	m.outer().HandleRestorationUpdateFromEngine(castElement[bool](enabled), castElement[[]byte](data))
}

func isMap(v any) bool {
	switch v.(type) {
	case []dart.MapEntry[any, any], []dart.MapEntry[string, any], *dart.LinkedMap:
		return true
	}
	return v != nil && reflect.TypeOf(v).Kind() == reflect.Map
}

// HandleRestorationUpdateFromEngine is called by the engine (and tests)
// with new restoration data (@protected): it replaces the root bucket,
// notifying listeners if it changed.
func (m *RestorationManager) HandleRestorationUpdateFromEngine(enabled bool, data []byte) {
	if !(enabled || data == nil) {
		panic(dart.NewAssertionError(nil))
	}
	m.isReplacing = m.rootBucketIsValid && enabled
	if m.isReplacing {
		scheduler.SchedulerBindingInstance().AddPostFrameCallback(func(time.Duration) {
			m.isReplacing = false
		}, scheduler.AddPostFrameCallbackOptions{DebugLabel: "RestorationManager.resetIsReplacing"})
	}
	oldRoot := m.rootBucket
	if enabled {
		m.rootBucket = NewRestorationBucketRoot(m, m.decodeRestorationData(data))
	} else {
		m.rootBucket = nil
	}
	m.rootBucketIsValid = true
	if m.pendingRootBucket != nil && m.pendingRootBucket.IsCompleted() {
		panic(dart.NewAssertionError(nil))
	}
	if m.pendingRootBucket != nil {
		m.pendingRootBucket.Complete(m.rootBucket)
	}
	m.pendingRootBucket = nil
	if m.rootBucket != oldRoot {
		m.NotifyListeners()
		if oldRoot != nil {
			oldRoot.Dispose()
		}
	}
}

// SendToEngine sends the encoded restoration data to the engine
// (@protected; tests override it).
func (m *RestorationManager) SendToEngine(encodedData []byte) async.Future[async.Void] {
	return asVoid(SystemChannels.Restoration.InvokeMethod("put", encodedData))
}

func (m *RestorationManager) methodHandler(call MethodCall) async.Future[any] {
	return async.Async(func() any {
		switch call.Method {
		case "push":
			if !isMap(call.Arguments) {
				panic(dart.TypeError{Message: "type '" + foundation.ObjectRuntimeType(call.Arguments, "Null") +
					"' is not a subtype of type 'Map<Object?, Object?>' in type cast"})
			}
			m.parseAndHandleRestorationUpdateFromEngine(call.Arguments)
		default:
			panic(dart.UnimplementedError{Message: call.Method + " was invoked but isn't implemented by " +
				foundation.ObjectRuntimeType(m.outer(), "RestorationManager")})
		}
		return nil
	})
}

func (m *RestorationManager) decodeRestorationData(data []byte) *dart.LinkedMap {
	if data == nil {
		return nil
	}
	decoded := newOrderedStandardMessageCodec().DecodeMessage(data)
	if decoded == nil {
		return nil
	}
	result, ok := decoded.(*dart.LinkedMap)
	if !ok {
		panic(dart.TypeError{Message: "type '" + foundation.ObjectRuntimeType(decoded, "Object") +
			"' is not a subtype of type 'Map<Object?, Object?>?' in type cast"})
	}
	return result
}

func encodeRestorationData(data *dart.LinkedMap) []byte {
	return newOrderedStandardMessageCodec().EncodeMessage(data)
}

// ScheduleSerializationFor is called by a RestorationBucket to request
// serialization for that bucket at the end of the frame
// (@protected, @visibleForTesting).
func (m *RestorationManager) ScheduleSerializationFor(bucket *RestorationBucket) {
	if bucket.manager != m {
		panic(dart.NewAssertionError(nil))
	}
	if m.debugDoingUpdate {
		panic(dart.NewAssertionError(nil))
	}
	if !m.needsSerialization(bucket) {
		m.bucketsNeedingSerialization = append(m.bucketsNeedingSerialization, bucket)
	}
	if !m.serializationScheduled {
		m.serializationScheduled = true
		scheduler.SchedulerBindingInstance().AddPostFrameCallback(func(time.Duration) { m.doSerialization() },
			scheduler.AddPostFrameCallbackOptions{DebugLabel: "RestorationManager.doSerialization"})
	}
}

func (m *RestorationManager) needsSerialization(bucket *RestorationBucket) bool {
	for _, b := range m.bucketsNeedingSerialization {
		if b == bucket {
			return true
		}
	}
	return false
}

// UnscheduleSerializationFor is called by a RestorationBucket to unschedule
// a request for serialization (@protected, @visibleForTesting).
func (m *RestorationManager) UnscheduleSerializationFor(bucket *RestorationBucket) {
	if bucket.manager != m {
		panic(dart.NewAssertionError(nil))
	}
	if m.debugDoingUpdate {
		panic(dart.NewAssertionError(nil))
	}
	for i, b := range m.bucketsNeedingSerialization {
		if b == bucket {
			m.bucketsNeedingSerialization = append(m.bucketsNeedingSerialization[:i:i], m.bucketsNeedingSerialization[i+1:]...)
			return
		}
	}
}

func (m *RestorationManager) doSerialization() {
	if !m.serializationScheduled {
		return
	}
	if foundation.KDebugMode {
		m.debugDoingUpdate = true
	}
	m.serializationScheduled = false
	for _, bucket := range m.bucketsNeedingSerialization {
		bucket.Finalize()
	}
	m.bucketsNeedingSerialization = nil
	if m.rootBucket == nil {
		panic(dart.TypeError{Message: "Null check operator used on a null value"}) // _rootBucket!
	}
	m.outer().SendToEngine(encodeRestorationData(m.rootBucket.rawData))
	if foundation.KDebugMode {
		m.debugDoingUpdate = false
	}
}

// FlushData sends the current restoration data to the engine now, unless
// a frame is scheduled (the data is then sent at its end).
func (m *RestorationManager) FlushData() {
	if m.debugDoingUpdate {
		panic(dart.NewAssertionError(nil))
	}
	if scheduler.SchedulerBindingInstance().HasScheduledFrame() {
		return
	}
	m.doSerialization()
	if m.serializationScheduled {
		panic(dart.NewAssertionError(nil))
	}
}

// Dispose disposes the root bucket and the ChangeNotifier.
func (m *RestorationManager) Dispose() {
	if m.rootBucket != nil {
		m.rootBucket.Dispose()
	}
	m.ChangeNotifierMixin.Dispose()
}

func (m *RestorationManager) String() string { return m.ChangeNotifierMixin.String() }

// RestorationBucket is a bucket in the restoration hierarchy: it stores
// values (Read/Write/Remove) and child buckets (ClaimChild/AdoptChild). It
// is used by pointer and compared by identity.
type RestorationBucket struct {
	rawData       *dart.LinkedMap
	debugOwner    any
	manager       *RestorationManager
	parent        *RestorationBucket
	restorationID string

	claimedChildren    *orderedBuckets                 // _claimedChildren
	childrenToAdd      map[string][]*RestorationBucket // _childrenToAdd
	childrenToAddOrder []string                        // its keys in insertion order

	needsSerialization bool
	debugDisposed      bool
}

const (
	childrenMapKey = "c"
	valuesMapKey   = "v"
)

// orderedBuckets is a Map<String, RestorationBucket> in insertion order.
type orderedBuckets struct {
	ids     []string
	buckets map[string]*RestorationBucket
}

func newOrderedBuckets() *orderedBuckets {
	return &orderedBuckets{buckets: map[string]*RestorationBucket{}}
}

func (o *orderedBuckets) get(id string) *RestorationBucket { return o.buckets[id] }

func (o *orderedBuckets) contains(id string) bool {
	_, ok := o.buckets[id]
	return ok
}

func (o *orderedBuckets) set(id string, b *RestorationBucket) {
	if _, ok := o.buckets[id]; !ok {
		o.ids = append(o.ids, id)
	}
	o.buckets[id] = b
}

func (o *orderedBuckets) remove(id string) *RestorationBucket {
	b, ok := o.buckets[id]
	if !ok {
		return nil
	}
	delete(o.buckets, id)
	for i, x := range o.ids {
		if x == id {
			o.ids = append(o.ids[:i:i], o.ids[i+1:]...)
			break
		}
	}
	return b
}

func (o *orderedBuckets) values() []*RestorationBucket {
	out := make([]*RestorationBucket, len(o.ids))
	for i, id := range o.ids {
		out[i] = o.buckets[id]
	}
	return out
}

func (o *orderedBuckets) clear() {
	o.ids = nil
	o.buckets = map[string]*RestorationBucket{}
}

// NewRestorationBucketEmpty is RestorationBucket.empty(restorationId:,
// debugOwner:): an empty bucket with the given ID, to be adopted into the
// hierarchy (the required named parameters are positional).
func NewRestorationBucketEmpty(restorationID string, debugOwner any) *RestorationBucket {
	b := &RestorationBucket{restorationID: restorationID, rawData: &dart.LinkedMap{},
		claimedChildren: newOrderedBuckets(), childrenToAdd: map[string][]*RestorationBucket{}}
	if foundation.KDebugMode {
		b.debugOwner = debugOwner
	}
	foundation.DebugMaybeDispatchCreated("services", "RestorationBucket", b)
	return b
}

// NewRestorationBucketRoot is RestorationBucket.root(manager:, rawData:):
// the root bucket of the hierarchy, wrapping rawData (nil: a new empty
// map), which it mutates in place.
func NewRestorationBucketRoot(manager *RestorationManager, rawData *dart.LinkedMap) *RestorationBucket {
	if rawData == nil {
		rawData = &dart.LinkedMap{}
	}
	b := &RestorationBucket{restorationID: "root", rawData: rawData, manager: manager,
		claimedChildren: newOrderedBuckets(), childrenToAdd: map[string][]*RestorationBucket{}}
	if foundation.KDebugMode && manager != nil {
		b.debugOwner = manager.outer()
	}
	foundation.DebugMaybeDispatchCreated("services", "RestorationBucket", b)
	return b
}

// NewRestorationBucketChild is RestorationBucket.child(restorationId:,
// parent:, debugOwner:): a bucket wrapping the data parent holds for the
// child restorationID (which must exist).
func NewRestorationBucketChild(restorationID string, parent *RestorationBucket, debugOwner any) *RestorationBucket {
	raw, _ := parent.rawChildren().Lookup(restorationID).(*dart.LinkedMap)
	if raw == nil {
		panic(dart.NewAssertionError(nil))
	}
	b := &RestorationBucket{restorationID: restorationID, rawData: raw, manager: parent.manager, parent: parent,
		claimedChildren: newOrderedBuckets(), childrenToAdd: map[string][]*RestorationBucket{}}
	if foundation.KDebugMode {
		b.debugOwner = debugOwner
	}
	foundation.DebugMaybeDispatchCreated("services", "RestorationBucket", b)
	return b
}

// DebugOwner is the owner of the bucket that was provided when the bucket
// was claimed via ClaimChild (debug builds; nil otherwise).
func (b *RestorationBucket) DebugOwner() any {
	b.debugAssertNotDisposed()
	return b.debugOwner
}

// IsReplacing is whether the manager is currently replacing the
// restoration data.
func (b *RestorationBucket) IsReplacing() bool {
	if b.manager == nil {
		return false
	}
	return b.manager.outer().IsReplacing()
}

// RestorationID is the identifier under which the bucket is stored in its
// parent.
func (b *RestorationBucket) RestorationID() string {
	b.debugAssertNotDisposed()
	return b.restorationID
}

// rawMap is `_rawData.putIfAbsent(key, () => {}) as Map`.
func (b *RestorationBucket) rawMap(key string) *dart.LinkedMap {
	v := b.rawData.PutIfAbsent(key, func() any { return &dart.LinkedMap{} })
	m, ok := v.(*dart.LinkedMap)
	if !ok || m == nil {
		panic(dart.TypeError{Message: "type '" + foundation.ObjectRuntimeType(v, "Null") +
			"' is not a subtype of type 'Map<Object?, Object?>' in type cast"})
	}
	return m
}

// rawChildren maps a restoration ID to the raw map representation of a
// child bucket.
func (b *RestorationBucket) rawChildren() *dart.LinkedMap { return b.rawMap(childrenMapKey) }

// rawValues maps a restoration ID to a value that is stored in this
// bucket.
func (b *RestorationBucket) rawValues() *dart.LinkedMap { return b.rawMap(valuesMapKey) }

// Read returns the value of type P that is currently stored in the bucket
// under restorationID (nil, Dart's null, if there is none).
func (b *RestorationBucket) Read(restorationID string) any {
	b.debugAssertNotDisposed()
	return b.rawValues().Lookup(restorationID)
}

// RestorationBucketRead is the typed read<P>: it panics with Dart's
// TypeError if the stored value is not a P.
func RestorationBucketRead[P any](bucket *RestorationBucket, restorationID string) P {
	return castElement[P](bucket.Read(restorationID))
}

// Write stores value under restorationID in the bucket (any value the
// StandardMessageCodec can encode); the bucket is serialized at the end of
// the frame if the value changed.
func (b *RestorationBucket) Write(restorationID string, value any) {
	b.debugAssertNotDisposed()
	if foundation.KDebugMode && !DebugIsSerializableForRestoration(value) {
		panic(dart.NewAssertionError(nil))
	}
	values := b.rawValues()
	current, ok := values.Get(restorationID)
	if !dartValuesEqual(current, value) || !ok {
		values.Set(restorationID, value)
		b.markNeedsSerialization()
	}
}

// Remove deletes the value stored under restorationID from the bucket and
// returns it (nil if there was none).
func (b *RestorationBucket) Remove(restorationID string) any {
	b.debugAssertNotDisposed()
	values := b.rawValues()
	needsUpdate := values.ContainsKey(restorationID)
	result, _ := values.Remove(restorationID)
	if values.IsEmpty() {
		b.rawData.Remove(valuesMapKey)
	}
	if needsUpdate {
		b.markNeedsSerialization()
	}
	return result
}

// Contains is whether a value is stored under restorationID.
func (b *RestorationBucket) Contains(restorationID string) bool {
	b.debugAssertNotDisposed()
	return b.rawValues().ContainsKey(restorationID)
}

// ClaimChild claims ownership of the child with the given restorationID
// from this bucket: a bucket wrapping its stored data, or a new empty
// bucket if there is none or the ID is already claimed this frame.
func (b *RestorationBucket) ClaimChild(restorationID string, debugOwner any) *RestorationBucket {
	b.debugAssertNotDisposed()
	// Case 1+2: Adopt and return an empty bucket.
	if b.claimedChildren.contains(restorationID) || !b.rawChildren().ContainsKey(restorationID) {
		child := NewRestorationBucketEmpty(restorationID, debugOwner)
		b.AdoptChild(child)
		return child
	}
	// Case 3: Return bucket wrapping the existing data.
	if b.rawChildren().Lookup(restorationID) == nil {
		panic(dart.NewAssertionError(nil))
	}
	child := NewRestorationBucketChild(restorationID, b, debugOwner)
	b.claimedChildren.set(restorationID, child)
	return child
}

// AdoptChild adopts child into this bucket (moving it from its current
// parent, if any); its data is stored under its RestorationID.
func (b *RestorationBucket) AdoptChild(child *RestorationBucket) {
	b.debugAssertNotDisposed()
	if child.parent != b {
		if child.parent != nil {
			child.parent.removeChildData(child)
		}
		child.parent = b
		b.addChildData(child)
		if child.manager != b.manager {
			b.recursivelyUpdateManager(child)
		}
	}
	if child.parent != b || child.manager != b.manager {
		panic(dart.NewAssertionError(nil))
	}
}

func (b *RestorationBucket) dropChild(child *RestorationBucket) {
	if child.parent != b {
		panic(dart.NewAssertionError(nil))
	}
	b.removeChildData(child)
	child.parent = nil
	if child.manager != nil {
		child.updateManager(nil)
		child.visitChildren(b.recursivelyUpdateManager, false)
	}
}

func (b *RestorationBucket) markNeedsSerialization() {
	if !b.needsSerialization {
		b.needsSerialization = true
		if b.manager != nil {
			b.manager.outer().ScheduleSerializationFor(b)
		}
	}
}

// Finalize is called by the RestorationManager at the end of a frame to
// serialize the bucket (@visibleForTesting); it checks that no ID was
// claimed twice.
func (b *RestorationBucket) Finalize() {
	b.debugAssertNotDisposed()
	if !b.needsSerialization {
		panic(dart.NewAssertionError(nil))
	}
	b.needsSerialization = false
	b.debugAssertIntegrity()
}

func (b *RestorationBucket) recursivelyUpdateManager(bucket *RestorationBucket) {
	bucket.updateManager(b.manager)
	bucket.visitChildren(b.recursivelyUpdateManager, false)
}

func (b *RestorationBucket) updateManager(newManager *RestorationManager) {
	if b.manager == newManager {
		return
	}
	if b.needsSerialization && b.manager != nil {
		b.manager.outer().UnscheduleSerializationFor(b)
	}
	b.manager = newManager
	if b.needsSerialization && b.manager != nil {
		b.needsSerialization = false
		b.markNeedsSerialization()
	}
}

func (b *RestorationBucket) debugAssertIntegrity() {
	if !foundation.KDebugMode || len(b.childrenToAddOrder) == 0 {
		return
	}
	errors := []foundation.DiagnosticsNode{
		foundation.NewErrorSummary("Multiple owners claimed child RestorationBuckets with the same IDs."),
		foundation.NewErrorDescription("The following IDs were claimed multiple times from the parent " + b.String() + ":"),
	}
	for _, id := range b.childrenToAddOrder {
		buckets := b.childrenToAdd[id]
		errors = append(errors, foundation.NewErrorDescription(` * "`+id+`" was claimed by:`))
		for _, bucket := range buckets {
			errors = append(errors, foundation.NewErrorDescription("   * "+dart.ToString(bucket.DebugOwner())))
		}
		errors = append(errors, foundation.NewErrorDescription("   * "+dart.ToString(b.claimedChildren.get(id).DebugOwner())+" (current owner)"))
	}
	panic(foundation.NewFlutterErrorFromParts(errors))
}

func (b *RestorationBucket) removeChildrenToAdd(id string) {
	delete(b.childrenToAdd, id)
	for i, x := range b.childrenToAddOrder {
		if x == id {
			b.childrenToAddOrder = append(b.childrenToAddOrder[:i:i], b.childrenToAddOrder[i+1:]...)
			return
		}
	}
}

func (b *RestorationBucket) removeChildData(child *RestorationBucket) {
	if child.parent != b {
		panic(dart.NewAssertionError(nil))
	}
	if b.claimedChildren.get(child.RestorationID()) == child {
		b.claimedChildren.remove(child.RestorationID())
		b.rawChildren().Remove(child.RestorationID())
		if pendingChildren, ok := b.childrenToAdd[child.RestorationID()]; ok {
			toAdd := pendingChildren[len(pendingChildren)-1]
			pendingChildren = pendingChildren[:len(pendingChildren)-1]
			b.childrenToAdd[child.RestorationID()] = pendingChildren
			b.finalizeAddChildData(toAdd)
			if len(pendingChildren) == 0 {
				b.removeChildrenToAdd(child.RestorationID())
			}
		}
		if b.rawChildren().IsEmpty() {
			b.rawData.Remove(childrenMapKey)
		}
		b.markNeedsSerialization()
		return
	}
	if pending, ok := b.childrenToAdd[child.RestorationID()]; ok {
		for i, x := range pending {
			if x == child {
				pending = append(pending[:i:i], pending[i+1:]...)
				break
			}
		}
		b.childrenToAdd[child.RestorationID()] = pending
		if len(pending) == 0 {
			b.removeChildrenToAdd(child.RestorationID())
		}
	}
}

func (b *RestorationBucket) addChildData(child *RestorationBucket) {
	if child.parent != b {
		panic(dart.NewAssertionError(nil))
	}
	if b.claimedChildren.contains(child.RestorationID()) {
		// Delay addition until the end of the frame in the hopes that the
		// current owner of the child with the same ID will have given up
		// that child by then.
		id := child.RestorationID()
		if _, ok := b.childrenToAdd[id]; !ok {
			b.childrenToAddOrder = append(b.childrenToAddOrder, id)
		}
		b.childrenToAdd[id] = append(b.childrenToAdd[id], child)
		b.markNeedsSerialization()
		return
	}
	b.finalizeAddChildData(child)
	b.markNeedsSerialization()
}

func (b *RestorationBucket) finalizeAddChildData(child *RestorationBucket) {
	if b.claimedChildren.get(child.RestorationID()) != nil || b.rawChildren().Lookup(child.RestorationID()) != nil {
		panic(dart.NewAssertionError(nil))
	}
	b.claimedChildren.set(child.RestorationID(), child)
	b.rawChildren().Set(child.RestorationID(), child.rawData)
}

func (b *RestorationBucket) visitChildren(visitor func(*RestorationBucket), concurrentModification bool) {
	// Dart iterates _claimedChildren.values.followedBy(...) lazily unless a
	// concurrent modification is expected; the visitors used without it
	// don't modify these maps.
	children := b.claimedChildren.values()
	for _, id := range b.childrenToAddOrder {
		children = append(children, b.childrenToAdd[id]...)
	}
	for _, child := range children {
		visitor(child)
	}
}

// Rename changes the restoration ID under which the bucket (or its
// owner) stores its data in its parent.
func (b *RestorationBucket) Rename(newRestorationID string) {
	b.debugAssertNotDisposed()
	if newRestorationID == b.RestorationID() {
		return
	}
	if b.parent != nil {
		b.parent.removeChildData(b)
	}
	b.restorationID = newRestorationID
	if b.parent != nil {
		b.parent.addChildData(b)
	}
}

// Dispose deletes the bucket and all the data stored in it from the
// hierarchy (its children are dropped and keep their data).
func (b *RestorationBucket) Dispose() {
	b.debugAssertNotDisposed()
	foundation.DebugMaybeDispatchDisposed(b)
	b.visitChildren(b.dropChild, true)
	b.claimedChildren.clear()
	b.childrenToAdd = map[string][]*RestorationBucket{}
	b.childrenToAddOrder = nil
	if b.parent != nil {
		b.parent.removeChildData(b)
	}
	b.parent = nil
	b.updateManager(nil)
	b.debugDisposed = true
}

func (b *RestorationBucket) String() string {
	return foundation.ObjectRuntimeType(b, "RestorationBucket") + "(restorationId: " + b.RestorationID() +
		", owner: " + dart.ToString(b.DebugOwner()) + ")"
}

func (b *RestorationBucket) debugAssertNotDisposed() {
	if foundation.KDebugMode && b.debugDisposed {
		panic(foundation.NewFlutterError("A RestorationBucket was used after being disposed.\n" +
			"Once you have called dispose() on a RestorationBucket, it can no longer be used."))
	}
}

// DebugIsSerializableForRestoration is debugIsSerializableForRestoration:
// whether object can be encoded by the StandardMessageCodec (debug
// builds; always false otherwise, as in Dart).
func DebugIsSerializableForRestoration(object any) (result bool) {
	if !foundation.KDebugMode {
		return false
	}
	defer func() {
		if recover() != nil {
			result = false
		}
	}()
	newOrderedStandardMessageCodec().EncodeMessage(object)
	return true
}

// dartValuesEqual is Dart's == on stored values: numbers by value (1 ==
// 1.0), comparable values with Go's ==, lists and maps by identity.
func dartValuesEqual(a, b any) bool {
	if isNull(a) || isNull(b) {
		return isNull(a) && isNull(b)
	}
	if ia, ok := a.(int); ok {
		if ib, ok := b.(int); ok {
			return ia == ib
		}
	}
	if fa, ok := numberValue(a); ok {
		fb, ok := numberValue(b)
		return ok && fa == fb
	}
	ta, tb := reflect.TypeOf(a), reflect.TypeOf(b)
	if ta != tb {
		return false
	}
	if ta.Comparable() {
		return comparableEqual(a, b)
	}
	va, vb := reflect.ValueOf(a), reflect.ValueOf(b)
	switch ta.Kind() {
	case reflect.Slice:
		return va.Len() == vb.Len() && (va.Len() == 0 && va.IsNil() == vb.IsNil() || va.Pointer() == vb.Pointer())
	case reflect.Map:
		return va.Pointer() == vb.Pointer()
	}
	return false
}

func numberValue(v any) (float64, bool) {
	switch x := v.(type) {
	case int:
		return float64(x), true
	case float64:
		return x, true
	}
	return 0, false
}
