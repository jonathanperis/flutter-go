package flutter_test

// Port of packages/flutter/test/services/fake_platform_views.dart (Flutter
// 3.50.1), the platform view fakes of Flutter's services and widgets tests:
// fake controllers and fake platform sides of the platform_views channel
// for Android, iOS and macOS. As with gesture_tester.dart, the helper lives
// in flutter_test so every library's tests can use it.

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/gestures"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/services"
	"github.com/jonathanperis/flutter-go/src/async"
	"github.com/jonathanperis/flutter-go/src/dart"
)

// FakePlatformViewController is a PlatformViewController that records what
// it is asked to do.
type FakePlatformViewController struct {
	services.PlatformViewControllerBase
	viewID                  int
	Disposed                bool
	FocusCleared            bool
	DispatchedPointerEvents []gestures.PointerEvent
	RejectGestureCount      int
	LastRejectGestureID     *int
}

// NewFakePlatformViewController is FakePlatformViewController(viewId).
func NewFakePlatformViewController(viewID int) *FakePlatformViewController {
	return &FakePlatformViewController{viewID: viewID}
}

// ViewID is the controller's view id.
func (c *FakePlatformViewController) ViewID() int { return c.viewID }

// DispatchPointerEvent records the event.
func (c *FakePlatformViewController) DispatchPointerEvent(event gestures.PointerEvent) async.Future[async.Void] {
	return async.AsyncVoid(func() { c.DispatchedPointerEvents = append(c.DispatchedPointerEvents, event) })
}

// ClearTestingVariables resets what was recorded.
func (c *FakePlatformViewController) ClearTestingVariables() {
	c.DispatchedPointerEvents = c.DispatchedPointerEvents[:0]
	c.Disposed = false
	c.FocusCleared = false
	c.RejectGestureCount = 0
	c.LastRejectGestureID = nil
}

// Dispose records the disposal.
func (c *FakePlatformViewController) Dispose() async.Future[async.Void] {
	return async.AsyncVoid(func() { c.Disposed = true })
}

// ClearFocus records the call.
func (c *FakePlatformViewController) ClearFocus() async.Future[async.Void] {
	return async.AsyncVoid(func() { c.FocusCleared = true })
}

// RejectGesture counts the call and records the gesture id.
func (c *FakePlatformViewController) RejectGesture(opts ...services.PlatformViewControllerRejectGestureOptions) async.Future[async.Void] {
	return async.AsyncVoid(func() {
		c.RejectGestureCount++
		c.LastRejectGestureID = nil
		if len(opts) > 0 {
			c.LastRejectGestureID = opts[0].GestureID
		}
	})
}

// FakeAndroidViewController is an AndroidViewController that records what
// it is asked to do.
type FakeAndroidViewController struct {
	viewID int
	// Disposed, FocusCleared and Created record the calls.
	Disposed, FocusCleared, Created bool
	// RequiresSize: if true, Create isn't considered successful unless it
	// includes a size.
	RequiresSize             bool
	createCalledSuccessfully bool
	CreatePosition           *ui.Offset
	createdCallbacks         []services.PlatformViewCreatedCallback
	DispatchedPointerEvents  []gestures.PointerEvent
	Offsets                  []ui.Offset
	Sizes                    []ui.Size
	pointTransformer         services.PointTransformer
	requiresViewComposition  bool
	RejectGestureCount       int
	LastRejectGestureID      *int
}

// FakeAndroidViewControllerOptions are FakeAndroidViewController's named
// parameters.
type FakeAndroidViewControllerOptions struct {
	RequiresSize            bool
	RequiresViewComposition bool
}

// NewFakeAndroidViewController is FakeAndroidViewController(viewId,
// {requiresSize, requiresViewComposition}).
func NewFakeAndroidViewController(viewID int, opts ...FakeAndroidViewControllerOptions) *FakeAndroidViewController {
	var o FakeAndroidViewControllerOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	return &FakeAndroidViewController{viewID: viewID, RequiresSize: o.RequiresSize, requiresViewComposition: o.RequiresViewComposition}
}

var _ services.AndroidViewController = (*FakeAndroidViewController)(nil)

// ViewID is the controller's view id.
func (c *FakeAndroidViewController) ViewID() int { return c.viewID }

// PointTransformer is the transformer set with SetPointTransformer.
func (c *FakeAndroidViewController) PointTransformer() services.PointTransformer {
	return c.pointTransformer
}

// SetPointTransformer sets PointTransformer.
func (c *FakeAndroidViewController) SetPointTransformer(t services.PointTransformer) {
	c.pointTransformer = t
}

// DispatchPointerEvent records the event.
func (c *FakeAndroidViewController) DispatchPointerEvent(event gestures.PointerEvent) async.Future[async.Void] {
	return async.AsyncVoid(func() { c.DispatchedPointerEvents = append(c.DispatchedPointerEvents, event) })
}

// ClearTestingVariables resets what was recorded.
func (c *FakeAndroidViewController) ClearTestingVariables() {
	c.DispatchedPointerEvents = c.DispatchedPointerEvents[:0]
	c.Offsets = c.Offsets[:0]
	c.Sizes = c.Sizes[:0]
	c.Disposed = false
	c.FocusCleared = false
}

// Dispose records the disposal.
func (c *FakeAndroidViewController) Dispose() async.Future[async.Void] {
	return async.AsyncVoid(func() { c.Disposed = true })
}

// ClearFocus records the call.
func (c *FakeAndroidViewController) ClearFocus() async.Future[async.Void] {
	return async.AsyncVoid(func() { c.FocusCleared = true })
}

// SetSize records the size and completes with it.
func (c *FakeAndroidViewController) SetSize(size ui.Size) async.Future[ui.Size] {
	c.Sizes = append(c.Sizes, size)
	return async.FutureValue(size)
}

// SetOffset records the offset.
func (c *FakeAndroidViewController) SetOffset(off ui.Offset) async.Future[async.Void] {
	return async.AsyncVoid(func() { c.Offsets = append(c.Offsets, off) })
}

// TextureID is 0.
func (c *FakeAndroidViewController) TextureID() *int {
	zero := 0
	return &zero
}

// AwaitingCreation is true until Create has been called successfully.
func (c *FakeAndroidViewController) AwaitingCreation() bool { return !c.createCalledSuccessfully }

// IsCreated is true once a listener has been added.
func (c *FakeAndroidViewController) IsCreated() bool { return c.Created }

// AddOnPlatformViewCreatedListener marks the view created and records the
// listener.
func (c *FakeAndroidViewController) AddOnPlatformViewCreatedListener(listener services.PlatformViewCreatedCallback) {
	c.Created = true
	c.createdCallbacks = append(c.createdCallbacks, listener)
}

// RemoveOnPlatformViewCreatedListener removes the listener.
func (c *FakeAndroidViewController) RemoveOnPlatformViewCreatedListener(listener services.PlatformViewCreatedCallback) {
	for i, l := range c.createdCallbacks {
		if dart.FunctionsEqual(l, listener) {
			c.createdCallbacks = slices.Delete(c.createdCallbacks, i, i+1)
			return
		}
	}
}

// SendMotionEvent is not implemented.
func (c *FakeAndroidViewController) SendMotionEvent(*services.AndroidMotionEvent) async.Future[async.Void] {
	panic(dart.UnimplementedError{})
}

// SetLayoutDirection is not implemented.
func (c *FakeAndroidViewController) SetLayoutDirection(ui.TextDirection) async.Future[async.Void] {
	panic(dart.UnimplementedError{})
}

// Create records the position; it succeeds if a size and a position are
// given or no size is required.
func (c *FakeAndroidViewController) Create(opts ...services.PlatformViewControllerCreateOptions) async.Future[async.Void] {
	var o services.PlatformViewControllerCreateOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	return async.AsyncVoid(func() {
		if c.createCalledSuccessfully {
			panic(dart.NewAssertionError(nil))
		}
		if c.RequiresSize && o.Size != nil && o.Size.IsEmpty() {
			panic(dart.NewAssertionError(nil))
		}
		c.createCalledSuccessfully = o.Size != nil && o.Position != nil || !c.RequiresSize
		c.CreatePosition = o.Position
	})
}

// CreatedCallbacks returns the listeners.
func (c *FakeAndroidViewController) CreatedCallbacks() []services.PlatformViewCreatedCallback {
	return c.createdCallbacks
}

// RequiresViewComposition is the value given at construction (or set).
func (c *FakeAndroidViewController) RequiresViewComposition() bool { return c.requiresViewComposition }

// SetRequiresViewComposition sets RequiresViewComposition.
func (c *FakeAndroidViewController) SetRequiresViewComposition(v bool) { c.requiresViewComposition = v }

// RejectGesture counts the call and records the gesture id.
func (c *FakeAndroidViewController) RejectGesture(opts ...services.PlatformViewControllerRejectGestureOptions) async.Future[async.Void] {
	return async.AsyncVoid(func() {
		c.RejectGestureCount++
		c.LastRejectGestureID = nil
		if len(opts) > 0 {
			c.LastRejectGestureID = opts[0].GestureID
		}
	})
}

func testMessenger() *TestDefaultBinaryMessenger {
	return TestDefaultBinaryMessengerBindingInstance().DefaultBinaryMessenger().(*TestDefaultBinaryMessenger)
}

func platformError(message string) *services.PlatformException {
	return services.NewPlatformException("error", services.PlatformExceptionOptions{Message: &message})
}

// invokeViewFocused sends a viewFocused call from the platform.
func invokeViewFocused(viewID int) {
	codec := services.SystemChannels.PlatformViews.Codec
	data := codec.EncodeMethodCall(services.MethodCall{Method: "viewFocused", Arguments: viewID})
	testMessenger().HandlePlatformMessage(services.SystemChannels.PlatformViews.Name, data, func([]byte) {})
}

func argOf[T any](args any, key string) T {
	v, _ := services.MapLookup(args, key)
	if v == nil {
		var zero T
		return zero
	}
	return v.(T)
}

func optionalArg[T any](args any, key string) *T {
	v, _ := services.MapLookup(args, key)
	if v == nil {
		return nil
	}
	t := v.(T)
	return &t
}

// FakeAndroidPlatformViewsController is a fake of the Android side of the
// platform_views channel.
type FakeAndroidPlatformViewsController struct {
	views                  map[int]FakeAndroidPlatformView
	order                  []int // views' keys in insertion order
	MotionEvents           map[int][]FakeAndroidMotionEvent
	registeredViewTypes    map[string]bool
	textureCounter         int
	ResizeCompleter        *async.Completer[async.Void]
	CreateCompleter        *async.Completer[async.Void]
	LastClearedFocusViewID *int
	Offsets                map[int]ui.Offset
	AllowTextureLayerMode  bool
}

// NewFakeAndroidPlatformViewsController installs the fake on
// SystemChannels.platform_views.
func NewFakeAndroidPlatformViewsController() *FakeAndroidPlatformViewsController {
	c := &FakeAndroidPlatformViewsController{
		views: map[int]FakeAndroidPlatformView{}, MotionEvents: map[int][]FakeAndroidMotionEvent{},
		registeredViewTypes: map[string]bool{}, Offsets: map[int]ui.Offset{}, AllowTextureLayerMode: true,
	}
	testMessenger().SetMockMethodCallHandler(services.SystemChannels.PlatformViews, c.onMethodCall)
	return c
}

// Views are the views created and not disposed, in creation order.
func (c *FakeAndroidPlatformViewsController) Views() []FakeAndroidPlatformView {
	out := make([]FakeAndroidPlatformView, 0, len(c.order))
	for _, id := range c.order {
		out = append(out, c.views[id])
	}
	return out
}

func (c *FakeAndroidPlatformViewsController) put(id int, v FakeAndroidPlatformView) {
	if _, ok := c.views[id]; !ok {
		c.order = append(c.order, id)
	}
	c.views[id] = v
}

// RegisterViewType registers a view type the fake can create.
func (c *FakeAndroidPlatformViewsController) RegisterViewType(viewType string) {
	c.registeredViewTypes[viewType] = true
}

// InvokeViewFocused sends viewFocused for viewID from the platform.
func (c *FakeAndroidPlatformViewsController) InvokeViewFocused(viewID int) { invokeViewFocused(viewID) }

func (c *FakeAndroidPlatformViewsController) onMethodCall(call services.MethodCall) async.Future[any] {
	switch call.Method {
	case "create":
		return c.create(call)
	case "dispose":
		return c.dispose(call)
	case "resize":
		return c.resize(call)
	case "touch":
		return c.touch(call)
	case "setDirection":
		return c.setDirection(call)
	case "clearFocus":
		return c.clearFocus(call)
	case "offset":
		return c.offset(call)
	}
	return async.FutureValue[any](nil)
}

func (c *FakeAndroidPlatformViewsController) create(call services.MethodCall) async.Future[any] {
	return async.Async(func() any {
		args := call.Arguments
		id := argOf[int](args, "id")
		viewType := argOf[string](args, "viewType")
		width := optionalArg[float64](args, "width")
		height := optionalArg[float64](args, "height")
		layoutDirection := argOf[int](args, "direction")
		hybrid := optionalArg[bool](args, "hybrid")
		hybridFallback := optionalArg[bool](args, "hybridFallback")
		creationParams := argOf[[]byte](args, "params")
		top := optionalArg[float64](args, "top")
		left := optionalArg[float64](args, "left")
		if _, ok := c.views[id]; ok {
			panic(platformError("Trying to create an already created platform view, view id: " + strconv.Itoa(id)))
		}
		if !c.registeredViewTypes[viewType] {
			panic(platformError("Trying to create a platform view of unregistered type: " + viewType))
		}
		if c.CreateCompleter != nil {
			c.CreateCompleter.Future().Await()
		}
		var size *ui.Size
		if width != nil && height != nil {
			s := ui.NewSize(*width, *height)
			size = &s
		}
		var position *ui.Offset
		if left != nil && top != nil {
			p := ui.NewOffset(*left, *top)
			position = &p
		}
		c.put(id, FakeAndroidPlatformView{ID: id, Type: viewType, Size: size, LayoutDirection: layoutDirection,
			Hybrid: hybrid, HybridFallback: hybridFallback, CreationParams: creationParams, Position: position})
		// Return a hybrid result (null rather than a texture ID) if hybrid
		// was explicitly requested, or hybrid fallback was requested and
		// simulated.
		hybridResult := (hybrid != nil && *hybrid) || (!c.AllowTextureLayerMode && hybridFallback != nil && *hybridFallback)
		if hybridResult {
			return nil
		}
		textureID := c.textureCounter
		c.textureCounter++
		return textureID
	})
}

func nullCheck[T any](p *T) T {
	if p == nil {
		panic(dart.TypeError{Message: "Null check operator used on a null value"})
	}
	return *p
}

func (c *FakeAndroidPlatformViewsController) viewOrNull(id int) *FakeAndroidPlatformView {
	if v, ok := c.views[id]; ok {
		return &v
	}
	return nil
}

func (c *FakeAndroidPlatformViewsController) dispose(call services.MethodCall) async.Future[any] {
	args := call.Arguments
	id := argOf[int](args, "id")
	hybrid := argOf[bool](args, "hybrid")
	if hybrid && !nullCheck(nullCheck(c.viewOrNull(id)).Hybrid) {
		panic(dart.ArgumentError{Message: "An AndroidViewController using hybrid composition must pass `hybrid: true`"})
	} else if !hybrid {
		if h := nullCheck(c.viewOrNull(id)).Hybrid; h != nil && *h {
			panic(dart.ArgumentError{Message: "An AndroidViewController not using hybrid composition must pass `hybrid: false`"})
		}
	}
	if _, ok := c.views[id]; !ok {
		panic(platformError("Trying to dispose a platform view with unknown id: " + strconv.Itoa(id)))
	}
	delete(c.views, id)
	c.order = slices.DeleteFunc(c.order, func(i int) bool { return i == id })
	return async.FutureValue[any](nil)
}

func (c *FakeAndroidPlatformViewsController) resize(call services.MethodCall) async.Future[any] {
	return async.Async(func() any {
		args := call.Arguments
		id := argOf[int](args, "id")
		width := argOf[float64](args, "width")
		height := argOf[float64](args, "height")
		if _, ok := c.views[id]; !ok {
			panic(platformError("Trying to resize a platform view with unknown id: " + strconv.Itoa(id)))
		}
		if c.ResizeCompleter != nil {
			c.ResizeCompleter.Future().Await()
		}
		size := ui.NewSize(width, height)
		c.put(id, c.views[id].CopyWith(&size, nil))
		return []dart.MapEntry[any, any]{{Key: "width", Value: width}, {Key: "height", Value: height}}
	})
}

func (c *FakeAndroidPlatformViewsController) offset(call services.MethodCall) async.Future[any] {
	return async.Async(func() any {
		args := call.Arguments
		id := argOf[int](args, "id")
		top := argOf[float64](args, "top")
		left := argOf[float64](args, "left")
		c.Offsets[id] = ui.NewOffset(left, top)
		return nil
	})
}

func (c *FakeAndroidPlatformViewsController) touch(call services.MethodCall) async.Future[any] {
	args := call.Arguments.([]any)
	id := args[0].(int)
	action := args[3].(int)
	pointerProperties := args[5].([]any)
	pointerCoords := args[6].([]any)
	var pointerOffsets []ui.Offset
	var pointerIDs []int
	for i := range pointerCoords {
		pointerIDs = append(pointerIDs, pointerProperties[i].([]any)[0].(int))
		x := pointerCoords[i].([]any)[7].(float64)
		y := pointerCoords[i].([]any)[8].(float64)
		pointerOffsets = append(pointerOffsets, ui.NewOffset(x, y))
	}
	c.MotionEvents[id] = append(c.MotionEvents[id], FakeAndroidMotionEvent{Action: action, PointerIDs: pointerIDs, Pointers: pointerOffsets})
	return async.FutureValue[any](nil)
}

func (c *FakeAndroidPlatformViewsController) setDirection(call services.MethodCall) async.Future[any] {
	return async.Async(func() any {
		args := call.Arguments
		id := argOf[int](args, "id")
		layoutDirection := argOf[int](args, "direction")
		if _, ok := c.views[id]; !ok {
			panic(platformError("Trying to resize a platform view with unknown id: " + strconv.Itoa(id)))
		}
		c.put(id, c.views[id].CopyWith(nil, &layoutDirection))
		return nil
	})
}

func (c *FakeAndroidPlatformViewsController) clearFocus(call services.MethodCall) async.Future[any] {
	id := call.Arguments.(int)
	if _, ok := c.views[id]; !ok {
		panic(platformError("Trying to clear the focus on a platform view with unknown id: " + strconv.Itoa(id)))
	}
	c.LastClearedFocusViewID = &id
	return async.FutureValue[any](nil)
}

// FakeDarwinPlatformViewsController is the shared implementation of
// FakeIosPlatformViewsController and FakeMacosPlatformViewsController
// (identical in Dart but for their view type).
type fakeDarwinPlatformViewsController struct {
	views               map[int][]byte
	types               map[int]string
	order               []int
	registeredViewTypes map[string]bool
	// CreationDelay: when non nil, the 'create' call waits for it.
	CreationDelay *async.Completer[async.Void]
	// GesturesAccepted maps a view id to the number of gestures it accepted.
	GesturesAccepted map[int]int
	// GesturesRejected maps a view id to the number of gestures it rejected.
	GesturesRejected map[int]int
}

func newFakeDarwinPlatformViewsController() fakeDarwinPlatformViewsController {
	return fakeDarwinPlatformViewsController{views: map[int][]byte{}, types: map[int]string{},
		registeredViewTypes: map[string]bool{}, GesturesAccepted: map[int]int{}, GesturesRejected: map[int]int{}}
}

// RegisterViewType registers a view type the fake can create.
func (c *fakeDarwinPlatformViewsController) RegisterViewType(viewType string) {
	c.registeredViewTypes[viewType] = true
}

// InvokeViewFocused sends viewFocused for viewID from the platform.
func (c *fakeDarwinPlatformViewsController) InvokeViewFocused(viewID int) { invokeViewFocused(viewID) }

func (c *fakeDarwinPlatformViewsController) onMethodCall(call services.MethodCall) async.Future[any] {
	switch call.Method {
	case "create":
		return c.create(call)
	case "dispose":
		return c.dispose(call)
	case "acceptGesture":
		return async.Async(func() any {
			id := argOf[int](call.Arguments, "id")
			c.GesturesAccepted[id]++
			return nil
		})
	case "rejectGesture":
		return async.Async(func() any {
			id := argOf[int](call.Arguments, "id")
			c.GesturesRejected[id]++
			return nil
		})
	}
	return async.FutureValue[any](nil)
}

func (c *fakeDarwinPlatformViewsController) create(call services.MethodCall) async.Future[any] {
	return async.Async(func() any {
		if c.CreationDelay != nil {
			c.CreationDelay.Future().Await()
		}
		args := call.Arguments
		id := argOf[int](args, "id")
		viewType := argOf[string](args, "viewType")
		creationParams := argOf[[]byte](args, "params")
		if _, ok := c.views[id]; ok {
			panic(platformError("Trying to create an already created platform view, view id: " + strconv.Itoa(id)))
		}
		if !c.registeredViewTypes[viewType] {
			panic(platformError("Trying to create a platform view of unregistered type: " + viewType))
		}
		c.views[id] = creationParams
		c.types[id] = viewType
		c.order = append(c.order, id)
		c.GesturesAccepted[id] = 0
		c.GesturesRejected[id] = 0
		return nil
	})
}

func (c *fakeDarwinPlatformViewsController) dispose(call services.MethodCall) async.Future[any] {
	id := call.Arguments.(int)
	if _, ok := c.views[id]; !ok {
		panic(platformError("Trying to dispose a platform view with unknown id: " + strconv.Itoa(id)))
	}
	delete(c.views, id)
	delete(c.types, id)
	c.order = slices.DeleteFunc(c.order, func(i int) bool { return i == id })
	return async.FutureValue[any](nil)
}

// FakeIosPlatformViewsController is a fake of the iOS side of the
// platform_views channel.
type FakeIosPlatformViewsController struct {
	fakeDarwinPlatformViewsController
}

// NewFakeIosPlatformViewsController installs the fake on
// SystemChannels.platform_views.
func NewFakeIosPlatformViewsController() *FakeIosPlatformViewsController {
	c := &FakeIosPlatformViewsController{newFakeDarwinPlatformViewsController()}
	testMessenger().SetMockMethodCallHandler(services.SystemChannels.PlatformViews, c.onMethodCall)
	return c
}

// Views are the views created and not disposed, in creation order.
func (c *FakeIosPlatformViewsController) Views() []FakeUiKitView {
	out := make([]FakeUiKitView, 0, len(c.order))
	for _, id := range c.order {
		out = append(out, FakeUiKitView{ID: id, Type: c.types[id], CreationParams: c.views[id]})
	}
	return out
}

// FakeMacosPlatformViewsController is a fake of the macOS side of the
// platform_views channel.
type FakeMacosPlatformViewsController struct {
	fakeDarwinPlatformViewsController
}

// NewFakeMacosPlatformViewsController installs the fake on
// SystemChannels.platform_views.
func NewFakeMacosPlatformViewsController() *FakeMacosPlatformViewsController {
	c := &FakeMacosPlatformViewsController{newFakeDarwinPlatformViewsController()}
	testMessenger().SetMockMethodCallHandler(services.SystemChannels.PlatformViews, c.onMethodCall)
	return c
}

// Views are the views created and not disposed, in creation order.
func (c *FakeMacosPlatformViewsController) Views() []FakeAppKitView {
	out := make([]FakeAppKitView, 0, len(c.order))
	for _, id := range c.order {
		out = append(out, FakeAppKitView{ID: id, Type: c.types[id], CreationParams: c.views[id]})
	}
	return out
}

// FakeAndroidPlatformView is a view the Android fake created. Compare with
// Equal (Dart's ==).
type FakeAndroidPlatformView struct {
	ID              int
	Type            string
	CreationParams  []byte // nil: null
	Size            *ui.Size
	LayoutDirection int
	Hybrid          *bool
	HybridFallback  *bool
	Position        *ui.Offset
}

// CopyWith replaces the size and the layout direction (nil: keep).
func (v FakeAndroidPlatformView) CopyWith(size *ui.Size, layoutDirection *int) FakeAndroidPlatformView {
	if size != nil {
		v.Size = size
	}
	if layoutDirection != nil {
		v.LayoutDirection = *layoutDirection
	}
	return v
}

func ptrEqual[T comparable](a, b *T) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

// Equal is FakeAndroidPlatformView's ==.
func (v FakeAndroidPlatformView) Equal(o FakeAndroidPlatformView) bool {
	return o.ID == v.ID && o.Type == v.Type && bytesListEquals(o.CreationParams, v.CreationParams) &&
		ptrEqual(o.Size, v.Size) && ptrEqual(o.Hybrid, v.Hybrid) && ptrEqual(o.HybridFallback, v.HybridFallback) &&
		o.LayoutDirection == v.LayoutDirection && ptrEqual(o.Position, v.Position)
}

func bytesListEquals(a, b []byte) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return slices.Equal(a, b)
}

func nullableString[T fmt.Stringer](p *T) string {
	if p == nil {
		return "null"
	}
	return (*p).String()
}

func nullableBool(p *bool) string {
	if p == nil {
		return "null"
	}
	return strconv.FormatBool(*p)
}

func bytesString(b []byte) string {
	if b == nil {
		return "null"
	}
	parts := make([]string, len(b))
	for i, x := range b {
		parts[i] = strconv.Itoa(int(x))
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

func (v FakeAndroidPlatformView) String() string {
	return "FakeAndroidPlatformView(id: " + strconv.Itoa(v.ID) + ", type: " + v.Type + ", size: " + nullableString(v.Size) +
		", layoutDirection: " + strconv.Itoa(v.LayoutDirection) + ", hybrid: " + nullableBool(v.Hybrid) +
		", hybridFallback: " + nullableBool(v.HybridFallback) + ", creationParams: " + bytesString(v.CreationParams) +
		", position: " + nullableString(v.Position) + ")"
}

// FakeAndroidMotionEvent is a motion event the Android fake received.
// Compare with Equal.
type FakeAndroidMotionEvent struct {
	Action     int
	PointerIDs []int
	Pointers   []ui.Offset
}

// Equal is FakeAndroidMotionEvent's ==.
func (e FakeAndroidMotionEvent) Equal(o FakeAndroidMotionEvent) bool {
	return slices.Equal(o.PointerIDs, e.PointerIDs) && o.Action == e.Action && slices.Equal(o.Pointers, e.Pointers)
}

func (e FakeAndroidMotionEvent) String() string {
	ids := make([]string, len(e.PointerIDs))
	for i, id := range e.PointerIDs {
		ids[i] = strconv.Itoa(id)
	}
	pointers := make([]string, len(e.Pointers))
	for i, p := range e.Pointers {
		pointers[i] = p.String()
	}
	return "FakeAndroidMotionEvent(action: " + strconv.Itoa(e.Action) + ", pointerIds: [" + strings.Join(ids, ", ") +
		"], pointers: [" + strings.Join(pointers, ", ") + "])"
}

// FakeUiKitView is a view the iOS fake created. Dart's == compares
// creationParams by identity, so two views with params are never equal;
// Equal reproduces that.
type FakeUiKitView struct {
	ID             int
	Type           string
	CreationParams []byte
}

// Equal is FakeUiKitView's ==.
func (v FakeUiKitView) Equal(o FakeUiKitView) bool {
	return o.ID == v.ID && o.Type == v.Type && o.CreationParams == nil && v.CreationParams == nil
}

func (v FakeUiKitView) String() string {
	return "FakeUiKitView(id: " + strconv.Itoa(v.ID) + ", type: " + v.Type + ", creationParams: " + bytesString(v.CreationParams) + ")"
}

// FakeAppKitView is a view the macOS fake created (see FakeUiKitView).
type FakeAppKitView struct {
	ID             int
	Type           string
	CreationParams []byte
}

// Equal is FakeAppKitView's ==.
func (v FakeAppKitView) Equal(o FakeAppKitView) bool {
	return o.ID == v.ID && o.Type == v.Type && o.CreationParams == nil && v.CreationParams == nil
}

func (v FakeAppKitView) String() string {
	return "FakeAppKitView(id: " + strconv.Itoa(v.ID) + ", type: " + v.Type + ", creationParams: " + bytesString(v.CreationParams) + ")"
}
