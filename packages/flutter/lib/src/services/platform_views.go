package services

// Port of packages/flutter/lib/src/services/platform_views.dart (Flutter
// 3.50.1): the platform views registry and service, the Android, UIKit and
// AppKit view controllers, and the Android motion event conversion.

import (
	"slices"
	"strconv"
	"strings"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/gestures"
	"github.com/jonathanperis/flutter-go/src/async"
	"github.com/jonathanperis/flutter-go/src/dart"
)

// PointTransformer converts a given point from the global coordinate system
// to a local one.
type PointTransformer func(position ui.Offset) ui.Offset

// PlatformViewsRegistry is a registry for platform view IDs.
type PlatformViewsRegistry struct {
	nextPlatformViewID int
}

var platformViewsRegistry = &PlatformViewsRegistry{}

// PlatformViewsRegistryInstance is the top-level platformViewsRegistry
// (the type has the variable's Go name).
func PlatformViewsRegistryInstance() *PlatformViewsRegistry { return platformViewsRegistry }

// GetNextPlatformViewID allocates a unique identifier for a platform view.
func (r *PlatformViewsRegistry) GetNextPlatformViewID() int {
	// On the Android side, the interface exposed to users uses 32-bit
	// integers.
	const maxInt32 = 0x7FFFFFFF
	if r.nextPlatformViewID > maxInt32 {
		panic(dart.NewAssertionError(nil))
	}
	id := r.nextPlatformViewID
	r.nextPlatformViewID++
	return id
}

// PlatformViewCreatedCallback is a callback invoked when a platform view
// has been created.
type PlatformViewCreatedCallback func(id int)

// platformViewsServiceInstance is PlatformViewsService's private instance.
type platformViewsServiceInstance struct {
	focusCallbacks map[int]func()
}

var platformViewsServiceSingleton *platformViewsServiceInstance

// platformViewsServiceGet is PlatformViewsService._instance, created (and
// registering its method call handler) on first use, as Dart's lazy static
// final is.
func platformViewsServiceGet() *platformViewsServiceInstance {
	if platformViewsServiceSingleton == nil {
		s := &platformViewsServiceInstance{focusCallbacks: map[int]func(){}}
		platformViewsServiceSingleton = s
		SystemChannels.PlatformViews.SetMethodCallHandler(s.onMethodCall)
	}
	return platformViewsServiceSingleton
}

func (s *platformViewsServiceInstance) onMethodCall(call MethodCall) async.Future[any] {
	switch call.Method {
	case "viewFocused":
		id := call.Arguments.(int)
		if callback, ok := s.focusCallbacks[id]; ok {
			callback()
		}
	default:
		panic(dart.UnimplementedError{Message: call.Method + " was invoked but isn't implemented by PlatformViewsService"})
	}
	return async.FutureValue[any](nil)
}

// platformViewsService is the type of PlatformViewsService (whose public
// API is static).
type platformViewsService struct{}

// PlatformViewsService provides access to the platform views service
// (PlatformViewsService.initAndroidView is
// PlatformViewsService.InitAndroidView...).
var PlatformViewsService platformViewsService

// PlatformViewsServiceInitOptions are the optional named parameters of
// PlatformViewsService's init methods (CreationParams nil: null; a non-nil
// CreationParams needs a CreationParamsCodec; OnFocus nil: none).
type PlatformViewsServiceInitOptions struct {
	CreationParams      any
	CreationParamsCodec MessageCodec[any]
	OnFocus             func()
}

func initOptions(opts []PlatformViewsServiceInitOptions) PlatformViewsServiceInitOptions {
	if len(opts) > 0 {
		return opts[0]
	}
	return PlatformViewsServiceInitOptions{}
}

func (o PlatformViewsServiceInitOptions) onFocusOrNoop() func() {
	if o.OnFocus != nil {
		return o.OnFocus
	}
	return func() {}
}

// InitAndroidView creates a controller for a new Android view (texture
// layer, falling back to virtual display); id, viewType and
// layoutDirection are Dart's required named parameters.
func (platformViewsService) InitAndroidView(id int, viewType string, layoutDirection ui.TextDirection,
	opts ...PlatformViewsServiceInitOptions) *TextureAndroidViewController {
	o := initOptions(opts)
	if !(o.CreationParams == nil || o.CreationParamsCodec != nil) {
		panic(dart.NewAssertionError(nil))
	}
	controller := &TextureAndroidViewController{internals: newTextureInternals()}
	controller.init(controller, id, viewType, layoutDirection, o.CreationParams, o.CreationParamsCodec)
	platformViewsServiceGet().focusCallbacks[id] = o.onFocusOrNoop()
	return controller
}

// InitSurfaceAndroidView creates a controller for a new Android view
// (texture layer, falling back to hybrid composition).
func (platformViewsService) InitSurfaceAndroidView(id int, viewType string, layoutDirection ui.TextDirection,
	opts ...PlatformViewsServiceInitOptions) *SurfaceAndroidViewController {
	o := initOptions(opts)
	if !(o.CreationParams == nil || o.CreationParamsCodec != nil) {
		panic(dart.NewAssertionError(nil))
	}
	controller := &SurfaceAndroidViewController{internals: newTextureInternals()}
	controller.init(controller, id, viewType, layoutDirection, o.CreationParams, o.CreationParamsCodec)
	platformViewsServiceGet().focusCallbacks[id] = o.onFocusOrNoop()
	return controller
}

// InitExpensiveAndroidView creates a controller for a new Android view
// that always uses hybrid composition.
func (platformViewsService) InitExpensiveAndroidView(id int, viewType string, layoutDirection ui.TextDirection,
	opts ...PlatformViewsServiceInitOptions) *ExpensiveAndroidViewController {
	o := initOptions(opts)
	controller := &ExpensiveAndroidViewController{internals: hybridInternals{}}
	controller.init(controller, id, viewType, layoutDirection, o.CreationParams, o.CreationParamsCodec)
	platformViewsServiceGet().focusCallbacks[id] = o.onFocusOrNoop()
	return controller
}

// InitHybridAndroidView creates a controller for a new Android view using
// the newer hybrid composition (platform_views_2).
func (platformViewsService) InitHybridAndroidView(id int, viewType string, layoutDirection ui.TextDirection,
	opts ...PlatformViewsServiceInitOptions) *HybridAndroidViewController {
	o := initOptions(opts)
	controller := &HybridAndroidViewController{internals: hybrid2Internals{}}
	controller.init(controller, id, viewType, layoutDirection, o.CreationParams, o.CreationParamsCodec)
	platformViewsServiceGet().focusCallbacks[id] = o.onFocusOrNoop()
	return controller
}

// InitUiKitViewOptions are InitUiKitView's optional named parameters
// (GestureBlockingPolicy nil: fallbackToPluginDefault).
type InitUiKitViewOptions struct {
	GestureBlockingPolicy *UiKitViewGestureBlockingPolicy
	CreationParams        any
	CreationParamsCodec   MessageCodec[any]
	OnFocus               func()
}

// InitUiKitView creates a UIKit view controller, creating the view on the
// platform.
func (platformViewsService) InitUiKitView(id int, viewType string, layoutDirection ui.TextDirection,
	opts ...InitUiKitViewOptions) async.Future[*UiKitViewController] {
	var o InitUiKitViewOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	return async.Async(func() *UiKitViewController {
		if !(o.CreationParams == nil || o.CreationParamsCodec != nil) {
			panic(dart.NewAssertionError(nil))
		}
		policy := UiKitViewGestureBlockingPolicyFallbackToPluginDefault
		if o.GestureBlockingPolicy != nil {
			policy = *o.GestureBlockingPolicy
		}
		args := []dart.MapEntry[string, any]{
			{Key: "id", Value: id},
			{Key: "viewType", Value: viewType},
			{Key: "gestureBlockingPolicy", Value: policy.Name()},
		}
		if o.CreationParams != nil {
			args = append(args, dart.MapEntry[string, any]{Key: "params", Value: o.CreationParamsCodec.EncodeMessage(o.CreationParams)})
		}
		SystemChannels.PlatformViews.InvokeMethod("create", args).Await()
		if o.OnFocus != nil {
			platformViewsServiceGet().focusCallbacks[id] = o.OnFocus
		}
		return &UiKitViewController{DarwinPlatformViewController: *NewDarwinPlatformViewController(id, layoutDirection)}
	})
}

// InitAppKitView creates an AppKit view controller, creating the view on
// the platform.
func (platformViewsService) InitAppKitView(id int, viewType string, layoutDirection ui.TextDirection,
	opts ...PlatformViewsServiceInitOptions) async.Future[*AppKitViewController] {
	o := initOptions(opts)
	return async.Async(func() *AppKitViewController {
		if !(o.CreationParams == nil || o.CreationParamsCodec != nil) {
			panic(dart.NewAssertionError(nil))
		}
		args := []dart.MapEntry[string, any]{{Key: "id", Value: id}, {Key: "viewType", Value: viewType}}
		if o.CreationParams != nil {
			args = append(args, dart.MapEntry[string, any]{Key: "params", Value: o.CreationParamsCodec.EncodeMessage(o.CreationParams)})
		}
		SystemChannels.PlatformViews.InvokeMethod("create", args).Await()
		if o.OnFocus != nil {
			platformViewsServiceGet().focusCallbacks[id] = o.OnFocus
		}
		return &AppKitViewController{DarwinPlatformViewController: *NewDarwinPlatformViewController(id, layoutDirection)}
	})
}

// AndroidPointerProperties are the properties of an Android pointer (a
// comparable value).
type AndroidPointerProperties struct {
	// ID is the unique identifier of the pointer.
	ID int
	// ToolType is the type of tool used to make contact (the
	// AndroidPointerPropertiesKToolType constants).
	ToolType int
}

// Tool types (AndroidPointerProperties.kToolType...).
const (
	AndroidPointerPropertiesKToolTypeUnknown = 0
	AndroidPointerPropertiesKToolTypeFinger  = 1
	AndroidPointerPropertiesKToolTypeStylus  = 2
	AndroidPointerPropertiesKToolTypeMouse   = 3
	AndroidPointerPropertiesKToolTypeEraser  = 4
)

func (p AndroidPointerProperties) asList() []any { return []any{p.ID, p.ToolType} }

func (p AndroidPointerProperties) String() string {
	return foundation.ObjectRuntimeType(p, "AndroidPointerProperties") + "(id: " + strconv.Itoa(p.ID) +
		", toolType: " + strconv.Itoa(p.ToolType) + ")"
}

// AndroidPointerCoords are the position of an Android pointer and its
// contact (a comparable value).
type AndroidPointerCoords struct {
	Orientation, Pressure, Size, ToolMajor, ToolMinor, TouchMajor, TouchMinor, X, Y float64
}

func (c AndroidPointerCoords) asList() []any {
	return []any{c.Orientation, c.Pressure, c.Size, c.ToolMajor, c.ToolMinor, c.TouchMajor, c.TouchMinor, c.X, c.Y}
}

func (c AndroidPointerCoords) String() string {
	d := dart.DoubleToString
	return foundation.ObjectRuntimeType(c, "AndroidPointerCoords") + "(orientation: " + d(c.Orientation) +
		", pressure: " + d(c.Pressure) + ", size: " + d(c.Size) + ", toolMajor: " + d(c.ToolMajor) +
		", toolMinor: " + d(c.ToolMinor) + ", touchMajor: " + d(c.TouchMajor) + ", touchMinor: " +
		d(c.TouchMinor) + ", x: " + d(c.X) + ", y: " + d(c.Y) + ")"
}

// AndroidMotionEvent is a Dart version of Android's MotionEvent. Every
// parameter is required, so it is built as a struct literal; Dart's
// assertions (PointerCount equals the lengths of the lists) run when it is
// sent.
type AndroidMotionEvent struct {
	DownTime          int
	EventTime         int
	Action            int
	PointerCount      int
	PointerProperties []AndroidPointerProperties
	PointerCoords     []AndroidPointerCoords
	MetaState         int
	ButtonState       int
	XPrecision        float64
	YPrecision        float64
	DeviceID          int
	EdgeFlags         int
	Source            int
	Flags             int
	MotionEventID     int
}

func (e *AndroidMotionEvent) asList(viewID int) []any {
	if len(e.PointerProperties) != e.PointerCount || len(e.PointerCoords) != e.PointerCount {
		panic(dart.NewAssertionError(nil))
	}
	properties := make([]any, len(e.PointerProperties))
	for i, p := range e.PointerProperties {
		properties[i] = p.asList()
	}
	coords := make([]any, len(e.PointerCoords))
	for i, c := range e.PointerCoords {
		coords[i] = c.asList()
	}
	return []any{viewID, e.DownTime, e.EventTime, e.Action, e.PointerCount, properties, coords, e.MetaState,
		e.ButtonState, e.XPrecision, e.YPrecision, e.DeviceID, e.EdgeFlags, e.Source, e.Flags, e.MotionEventID}
}

func (e *AndroidMotionEvent) String() string {
	properties := make([]string, len(e.PointerProperties))
	for i, p := range e.PointerProperties {
		properties[i] = p.String()
	}
	coords := make([]string, len(e.PointerCoords))
	for i, c := range e.PointerCoords {
		coords[i] = c.String()
	}
	i := strconv.Itoa
	return "AndroidPointerEvent(downTime: " + i(e.DownTime) + ", eventTime: " + i(e.EventTime) + ", action: " +
		i(e.Action) + ", pointerCount: " + i(e.PointerCount) + ", pointerProperties: [" +
		strings.Join(properties, ", ") + "], pointerCoords: [" + strings.Join(coords, ", ") + "], metaState: " +
		i(e.MetaState) + ", buttonState: " + i(e.ButtonState) + ", xPrecision: " + dart.DoubleToString(e.XPrecision) +
		", yPrecision: " + dart.DoubleToString(e.YPrecision) + ", deviceId: " + i(e.DeviceID) + ", edgeFlags: " +
		i(e.EdgeFlags) + ", source: " + i(e.Source) + ", flags: " + i(e.Flags) + ", motionEventId: " +
		i(e.MotionEventID) + ")"
}

type androidViewState int

const (
	androidViewStateWaitingForSize androidViewState = iota
	androidViewStateCreating
	androidViewStateCreated
	androidViewStateDisposed
)

// androidMotionEventConverter is _AndroidMotionEventConverter.
type androidMotionEventConverter struct {
	pointerPositions      map[int]AndroidPointerCoords
	pointerProperties     map[int]AndroidPointerProperties
	pointerOrder          []int // the maps' keys in insertion order (Dart's LinkedHashMap)
	usedAndroidPointerIDs map[int]bool
	pointTransformer      PointTransformer
	downTimeMillis        *int
}

func newAndroidMotionEventConverter() *androidMotionEventConverter {
	return &androidMotionEventConverter{
		pointerPositions:      map[int]AndroidPointerCoords{},
		pointerProperties:     map[int]AndroidPointerProperties{},
		usedAndroidPointerIDs: map[int]bool{},
	}
}

func (c *androidMotionEventConverter) handlePointerDownEvent(event gestures.PointerEvent) {
	if len(c.pointerProperties) == 0 {
		ms := int(event.TimeStamp().Milliseconds())
		c.downTimeMillis = &ms
	}
	androidPointerID := 0
	for c.usedAndroidPointerIDs[androidPointerID] {
		androidPointerID++
	}
	c.usedAndroidPointerIDs[androidPointerID] = true
	c.pointerProperties[event.Pointer()] = c.propertiesFor(event, androidPointerID)
}

func (c *androidMotionEventConverter) updatePointerPositions(event gestures.PointerEvent) {
	position := c.pointTransformer(event.Position())
	if _, ok := c.pointerPositions[event.Pointer()]; !ok {
		c.pointerOrder = append(c.pointerOrder, event.Pointer())
	}
	c.pointerPositions[event.Pointer()] = AndroidPointerCoords{
		Orientation: event.Orientation(),
		Pressure:    event.Pressure(),
		Size:        event.Size(),
		ToolMajor:   event.RadiusMajor(),
		ToolMinor:   event.RadiusMinor(),
		TouchMajor:  event.RadiusMajor(),
		TouchMinor:  event.RadiusMinor(),
		X:           position.Dx,
		Y:           position.Dy,
	}
}

func (c *androidMotionEventConverter) remove(pointer int) {
	if _, ok := c.pointerPositions[pointer]; ok {
		delete(c.pointerPositions, pointer)
		c.pointerOrder = slices.DeleteFunc(c.pointerOrder, func(p int) bool { return p == pointer })
	}
	props, ok := c.pointerProperties[pointer]
	if !ok {
		panic(dart.TypeError{Message: "Null check operator used on a null value"})
	}
	delete(c.usedAndroidPointerIDs, props.ID)
	delete(c.pointerProperties, pointer)
	if len(c.pointerProperties) == 0 {
		c.downTimeMillis = nil
	}
}

func (c *androidMotionEventConverter) toAndroidMotionEvent(event gestures.PointerEvent) *AndroidMotionEvent {
	pointers := slices.Clone(c.pointerOrder)
	if len(pointers) >= 2 {
		// Dart's comparator reads pointerProperties[x]! for every element.
		for _, p := range pointers {
			if _, ok := c.pointerProperties[p]; !ok {
				panic(dart.TypeError{Message: "Null check operator used on a null value"})
			}
		}
	}
	slices.SortStableFunc(pointers, func(a, b int) int {
		return c.pointerProperties[a].ID - c.pointerProperties[b].ID
	})
	pointerIdx := slices.Index(pointers, event.Pointer())
	numPointers := len(pointers)

	// This value must match the value in engine's FlutterView.java.
	// This flag indicates whether the original Android pointer events were
	// batched together.
	const kPointerDataFlagBatched = 1
	// This value must match the value in engine's FlutterView.java.
	// This flag indicates whether the original Android pointer events were
	// multiple pointers.
	const kPointerDataFlagMultiple = 2
	const kPointerDataFlagMask = 0xff
	const kPointerDataMultiplePointerCountShift = 8

	platformDataFlag := event.PlatformData() & kPointerDataFlagMask
	// Android MotionEvent objects can batch information on multiple
	// pointers. Flutter breaks these such batched events into multiple
	// PointerEvent objects. When there are multiple active pointers we
	// accumulate the information for all pointers before we dispatch the
	// MotionEvent to the embedded view.
	if platformDataFlag == kPointerDataFlagBatched {
		return nil
	}
	if platformDataFlag == kPointerDataFlagMultiple {
		originalPointerCount := event.PlatformData() >> kPointerDataMultiplePointerCountShift
		if pointerIdx != originalPointerCount-1 {
			return nil
		}
	}

	var action int
	switch event.(type) {
	case *gestures.PointerDownEvent:
		if numPointers == 1 {
			action = AndroidViewControllerKActionDown
		} else {
			action = AndroidViewControllerPointerAction(pointerIdx, AndroidViewControllerKActionPointerDown)
		}
	case *gestures.PointerUpEvent:
		if numPointers == 1 {
			action = AndroidViewControllerKActionUp
		} else {
			action = AndroidViewControllerPointerAction(pointerIdx, AndroidViewControllerKActionPointerUp)
		}
	case *gestures.PointerMoveEvent:
		action = AndroidViewControllerKActionMove
	case *gestures.PointerCancelEvent:
		action = AndroidViewControllerKActionCancel
	default:
		return nil
	}

	if c.downTimeMillis == nil {
		panic(dart.TypeError{Message: "Null check operator used on a null value"})
	}
	properties := make([]AndroidPointerProperties, len(pointers))
	coords := make([]AndroidPointerCoords, len(pointers))
	for i, p := range pointers {
		props, ok := c.pointerProperties[p]
		if !ok {
			panic(dart.TypeError{Message: "Null check operator used on a null value"})
		}
		properties[i] = props
		coords[i] = c.pointerPositions[p]
	}
	return &AndroidMotionEvent{
		DownTime:          *c.downTimeMillis,
		EventTime:         int(event.TimeStamp().Milliseconds()),
		Action:            action,
		PointerCount:      len(c.pointerPositions),
		PointerProperties: properties,
		PointerCoords:     coords,
		MetaState:         0,
		ButtonState:       0,
		XPrecision:        1.0,
		YPrecision:        1.0,
		DeviceID:          0,
		EdgeFlags:         0,
		Source:            androidMotionEventConverterSourceFor(event),
		Flags:             0,
		MotionEventID:     event.EmbedderID(),
	}
}

func androidMotionEventConverterSourceFor(event gestures.PointerEvent) int {
	switch event.Kind() {
	case ui.PointerDeviceKindTouch:
		return AndroidViewControllerKInputDeviceSourceTouchScreen
	case ui.PointerDeviceKindTrackpad:
		return AndroidViewControllerKInputDeviceSourceTouchPad
	case ui.PointerDeviceKindMouse:
		return AndroidViewControllerKInputDeviceSourceMouse
	case ui.PointerDeviceKindStylus, ui.PointerDeviceKindInvertedStylus:
		return AndroidViewControllerKInputDeviceSourceStylus
	}
	return AndroidViewControllerKInputDeviceSourceUnknown
}

func (c *androidMotionEventConverter) propertiesFor(event gestures.PointerEvent, pointerID int) AndroidPointerProperties {
	toolType := AndroidPointerPropertiesKToolTypeUnknown
	switch event.Kind() {
	case ui.PointerDeviceKindTouch, ui.PointerDeviceKindTrackpad:
		toolType = AndroidPointerPropertiesKToolTypeFinger
	case ui.PointerDeviceKindMouse:
		toolType = AndroidPointerPropertiesKToolTypeMouse
	case ui.PointerDeviceKindStylus:
		toolType = AndroidPointerPropertiesKToolTypeStylus
	case ui.PointerDeviceKindInvertedStylus:
		toolType = AndroidPointerPropertiesKToolTypeEraser
	}
	return AndroidPointerProperties{ID: pointerID, ToolType: toolType}
}

type creationParams struct {
	data  any
	codec MessageCodec[any]
}

// Android MotionEvent actions, layout directions and input device sources
// (AndroidViewController's static constants).
const (
	AndroidViewControllerKActionDown        = 0
	AndroidViewControllerKActionUp          = 1
	AndroidViewControllerKActionMove        = 2
	AndroidViewControllerKActionCancel      = 3
	AndroidViewControllerKActionPointerDown = 5
	AndroidViewControllerKActionPointerUp   = 6

	AndroidViewControllerKAndroidLayoutDirectionLtr = 0
	AndroidViewControllerKAndroidLayoutDirectionRtl = 1

	AndroidViewControllerKInputDeviceSourceUnknown     = 0
	AndroidViewControllerKInputDeviceSourceTouchScreen = 4098
	AndroidViewControllerKInputDeviceSourceMouse       = 8194
	AndroidViewControllerKInputDeviceSourceStylus      = 16386
	AndroidViewControllerKInputDeviceSourceTouchPad    = 1048584
)

func androidViewControllerGetAndroidDirection(direction ui.TextDirection) int {
	if direction == ui.TextDirectionRtl {
		return AndroidViewControllerKAndroidLayoutDirectionRtl
	}
	return AndroidViewControllerKAndroidLayoutDirectionLtr
}

// AndroidViewControllerPointerAction is AndroidViewController.pointerAction:
// the action for a pointer at index pointerID.
func AndroidViewControllerPointerAction(pointerID, action int) int {
	return ((pointerID << 8) & 0xff00) | (action & 0xff)
}

// PlatformViewControllerCreateOptions are create's named parameters (nil:
// null).
type PlatformViewControllerCreateOptions struct {
	Size     *ui.Size
	Position *ui.Offset
}

// PlatformViewControllerRejectGestureOptions are rejectGesture's named
// parameters (GestureID nil: null).
type PlatformViewControllerRejectGestureOptions struct{ GestureID *int }

// PlatformViewController is an interface for controlling a single platform
// view. Implementations may embed PlatformViewControllerBase for the
// default AwaitingCreation, Create and RejectGesture.
type PlatformViewController interface {
	// ViewID is the viewId associated with this controller.
	ViewID() int
	// AwaitingCreation is true if Create has not been successfully called.
	AwaitingCreation() bool
	// DispatchPointerEvent dispatches the event to the platform view.
	DispatchPointerEvent(event gestures.PointerEvent) async.Future[async.Void]
	// Create creates the platform view with the initial size and position.
	Create(opts ...PlatformViewControllerCreateOptions) async.Future[async.Void]
	// Dispose disposes the platform view.
	Dispose() async.Future[async.Void]
	// ClearFocus clears the view's focus on the platform side.
	ClearFocus() async.Future[async.Void]
	// RejectGesture tells the platform view that the gesture arena rejected
	// it.
	RejectGesture(opts ...PlatformViewControllerRejectGestureOptions) async.Future[async.Void]
}

// PlatformViewControllerBase holds PlatformViewController's default
// methods.
type PlatformViewControllerBase struct{}

// AwaitingCreation is false by default.
func (PlatformViewControllerBase) AwaitingCreation() bool { return false }

// Create does nothing by default.
func (PlatformViewControllerBase) Create(...PlatformViewControllerCreateOptions) async.Future[async.Void] {
	return async.AsyncVoid(func() {})
}

// RejectGesture does nothing by default.
func (PlatformViewControllerBase) RejectGesture(...PlatformViewControllerRejectGestureOptions) async.Future[async.Void] {
	return async.AsyncVoid(func() {})
}

// AndroidViewController controls an Android view (one of the controllers
// PlatformViewsService creates, or a fake implementing it).
type AndroidViewController interface {
	PlatformViewController
	// SetSize sizes the Android view (creating it first if needed); the
	// future's value is the size the view was given.
	SetSize(size ui.Size) async.Future[ui.Size]
	// SetOffset sets the offset of the platform view.
	SetOffset(off ui.Offset) async.Future[async.Void]
	// TextureID returns the texture entry id the view is rendered into (nil
	// if none).
	TextureID() *int
	// RequiresViewComposition is true if the view requires native view
	// composition rather than a texture.
	RequiresViewComposition() bool
	// SendMotionEvent sends an Android MotionEvent to the view.
	SendMotionEvent(event *AndroidMotionEvent) async.Future[async.Void]
	// PointTransformer converts points from the global to the view's
	// coordinates.
	PointTransformer() PointTransformer
	// SetPointTransformer sets PointTransformer.
	SetPointTransformer(transformer PointTransformer)
	// IsCreated is whether the platform view has already been created.
	IsCreated() bool
	// AddOnPlatformViewCreatedListener adds a callback invoked when the view
	// is created.
	AddOnPlatformViewCreatedListener(listener PlatformViewCreatedCallback)
	// RemoveOnPlatformViewCreatedListener removes a callback added with
	// AddOnPlatformViewCreatedListener.
	RemoveOnPlatformViewCreatedListener(listener PlatformViewCreatedCallback)
	// CreatedCallbacks returns the listeners (@visibleForTesting).
	CreatedCallbacks() []PlatformViewCreatedCallback
	// SetLayoutDirection sets the layout direction of the view.
	SetLayoutDirection(layoutDirection ui.TextDirection) async.Future[async.Void]
}

// androidViewControllerImpl is what the controllers override (Dart's
// abstract and overridden members of AndroidViewController).
type androidViewControllerImpl interface {
	AndroidViewController
	sendDisposeMessage() async.Future[async.Void]
	createRequiresSize() bool
	sendCreateMessage(size *ui.Size, position *ui.Offset) async.Future[async.Void]
	sendResizeMessage(size ui.Size) async.Future[ui.Size]
}

// AndroidViewControllerBase is the state and behavior the Android
// controllers share (Dart's abstract AndroidViewController).
type AndroidViewControllerBase struct {
	PlatformViewControllerBase
	self                         androidViewControllerImpl
	viewID                       int
	viewType                     string
	motionEventConverter         *androidMotionEventConverter
	layoutDirection              ui.TextDirection
	state                        androidViewState
	creationParams               *creationParams
	platformViewCreatedCallbacks []PlatformViewCreatedCallback
}

func (c *AndroidViewControllerBase) init(self androidViewControllerImpl, viewID int, viewType string,
	layoutDirection ui.TextDirection, params any, codec MessageCodec[any]) {
	if !(params == nil || codec != nil) {
		panic(dart.NewAssertionError(nil))
	}
	c.self, c.viewID, c.viewType, c.layoutDirection = self, viewID, viewType, layoutDirection
	c.motionEventConverter = newAndroidMotionEventConverter()
	if params != nil {
		c.creationParams = &creationParams{data: params, codec: codec}
	}
}

// ViewID is the unique identifier of the Android view controlled by this
// controller.
func (c *AndroidViewControllerBase) ViewID() int { return c.viewID }

// AwaitingCreation is true until the view has been sized and created.
func (c *AndroidViewControllerBase) AwaitingCreation() bool {
	return c.state == androidViewStateWaitingForSize
}

// Create creates the Android view (doing nothing when this kind of
// controller needs a size and none is given).
func (c *AndroidViewControllerBase) Create(opts ...PlatformViewControllerCreateOptions) async.Future[async.Void] {
	var o PlatformViewControllerCreateOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	return async.AsyncVoid(func() {
		if c.state == androidViewStateDisposed {
			panic(dart.NewAssertionError("trying to create a disposed Android view"))
		}
		if c.state != androidViewStateWaitingForSize {
			panic(dart.NewAssertionError("Android view is already sized. View id: " + strconv.Itoa(c.viewID)))
		}
		if c.self.createRequiresSize() && o.Size == nil {
			// Wait for a setSize call.
			return
		}
		c.state = androidViewStateCreating
		c.self.sendCreateMessage(o.Size, o.Position).Await()
		c.state = androidViewStateCreated
		for _, callback := range c.platformViewCreatedCallbacks {
			callback(c.viewID)
		}
	})
}

// SetSize sizes the view, creating it if it is still waiting for a size.
func (c *AndroidViewControllerBase) SetSize(size ui.Size) async.Future[ui.Size] {
	return async.Async(func() ui.Size {
		if c.state == androidViewStateDisposed {
			panic(dart.NewAssertionError("Android view is disposed. View id: " + strconv.Itoa(c.viewID)))
		}
		if c.state == androidViewStateWaitingForSize {
			// Either `create` hasn't been called, or it couldn't run due to
			// missing size information, so create the view now.
			c.self.Create(PlatformViewControllerCreateOptions{Size: &size}).Await()
			return size
		}
		return c.self.sendResizeMessage(size).Await()
	})
}

// RequiresViewComposition is false by default.
func (c *AndroidViewControllerBase) RequiresViewComposition() bool { return false }

// SendMotionEvent sends an Android MotionEvent on SystemChannels
// .platform_views.
func (c *AndroidViewControllerBase) SendMotionEvent(event *AndroidMotionEvent) async.Future[async.Void] {
	return async.AsyncVoid(func() {
		SystemChannels.PlatformViews.InvokeMethod("touch", event.asList(c.viewID)).Await()
	})
}

// PointTransformer converts points from the global to the view's
// coordinates.
func (c *AndroidViewControllerBase) PointTransformer() PointTransformer {
	if c.motionEventConverter.pointTransformer == nil {
		panic(dart.StateError{Message: "Field 'pointTransformer' has not been initialized."})
	}
	return c.motionEventConverter.pointTransformer
}

// SetPointTransformer sets PointTransformer.
func (c *AndroidViewControllerBase) SetPointTransformer(transformer PointTransformer) {
	c.motionEventConverter.pointTransformer = transformer
}

// IsCreated is whether the platform view has already been created.
func (c *AndroidViewControllerBase) IsCreated() bool { return c.state == androidViewStateCreated }

// AddOnPlatformViewCreatedListener adds a callback invoked when the view is
// created.
func (c *AndroidViewControllerBase) AddOnPlatformViewCreatedListener(listener PlatformViewCreatedCallback) {
	if c.state == androidViewStateDisposed {
		panic(dart.NewAssertionError(nil))
	}
	c.platformViewCreatedCallbacks = append(c.platformViewCreatedCallbacks, listener)
}

// RemoveOnPlatformViewCreatedListener removes the first occurrence of
// listener (dart.FunctionsEqual).
func (c *AndroidViewControllerBase) RemoveOnPlatformViewCreatedListener(listener PlatformViewCreatedCallback) {
	if c.state == androidViewStateDisposed {
		panic(dart.NewAssertionError(nil))
	}
	for i, l := range c.platformViewCreatedCallbacks {
		if dart.FunctionsEqual(l, listener) {
			c.platformViewCreatedCallbacks = slices.Delete(c.platformViewCreatedCallbacks, i, i+1)
			return
		}
	}
}

// CreatedCallbacks returns the listeners (@visibleForTesting).
func (c *AndroidViewControllerBase) CreatedCallbacks() []PlatformViewCreatedCallback {
	return c.platformViewCreatedCallbacks
}

// SetLayoutDirection sets the layout direction of the Android view (sent to
// the platform once the view exists).
func (c *AndroidViewControllerBase) SetLayoutDirection(layoutDirection ui.TextDirection) async.Future[async.Void] {
	return async.AsyncVoid(func() {
		if c.state == androidViewStateDisposed {
			panic(dart.NewAssertionError("trying to set a layout direction for a disposed Android view. View id: " +
				strconv.Itoa(c.viewID)))
		}
		if layoutDirection == c.layoutDirection {
			return
		}
		c.layoutDirection = layoutDirection
		// If the view was not yet created we just update _layoutDirection and
		// return, as the new direction will be used in _create.
		if c.state == androidViewStateWaitingForSize {
			return
		}
		SystemChannels.PlatformViews.InvokeMethod("setDirection", []dart.MapEntry[string, any]{
			{Key: "id", Value: c.viewID},
			{Key: "direction", Value: androidViewControllerGetAndroidDirection(layoutDirection)},
		}).Await()
	})
}

// DispatchPointerEvent converts the event to an Android MotionEvent and
// sends it to the view (hover events are ignored).
func (c *AndroidViewControllerBase) DispatchPointerEvent(event gestures.PointerEvent) async.Future[async.Void] {
	return async.AsyncVoid(func() {
		if _, hover := event.(*gestures.PointerHoverEvent); hover {
			return
		}
		if _, down := event.(*gestures.PointerDownEvent); down {
			c.motionEventConverter.handlePointerDownEvent(event)
		}
		c.motionEventConverter.updatePointerPositions(event)
		androidEvent := c.motionEventConverter.toAndroidMotionEvent(event)
		switch event.(type) {
		case *gestures.PointerUpEvent, *gestures.PointerCancelEvent:
			c.motionEventConverter.remove(event.Pointer())
		}
		if androidEvent != nil {
			c.self.SendMotionEvent(androidEvent).Await()
		}
	})
}

// ClearFocus clears the focus from the Android view if it is focused.
func (c *AndroidViewControllerBase) ClearFocus() async.Future[async.Void] {
	if c.state != androidViewStateCreated {
		return async.FutureValue(async.Void{})
	}
	return async.CastFuture[async.Void](SystemChannels.PlatformViews.InvokeMethod("clearFocus", c.viewID))
}

// Dispose disposes the Android view (sending the dispose message if it was
// created or is being created).
func (c *AndroidViewControllerBase) Dispose() async.Future[async.Void] {
	return async.AsyncVoid(func() {
		state := c.state
		c.state = androidViewStateDisposed
		c.platformViewCreatedCallbacks = nil
		delete(platformViewsServiceGet().focusCallbacks, c.viewID)
		if state == androidViewStateCreating || state == androidViewStateCreated {
			c.self.sendDisposeMessage().Await()
		}
	})
}

// SurfaceAndroidViewController controls an Android view composed with
// texture layer composition, falling back to hybrid composition.
type SurfaceAndroidViewController struct {
	AndroidViewControllerBase
	internals androidViewControllerInternals
}

func (c *SurfaceAndroidViewController) createRequiresSize() bool { return true }

func (c *SurfaceAndroidViewController) sendCreateMessage(size *ui.Size, position *ui.Offset) async.Future[async.Void] {
	return async.AsyncVoid(func() {
		if size.IsEmpty() {
			panic(dart.NewAssertionError("trying to create TextureAndroidViewController without setting a valid size."))
		}
		response := androidViewControllerInternalsSendCreateMessage(c.viewID, c.viewType, c.layoutDirection, false,
			true, false, c.creationParams, size, position).Await()
		if id, ok := response.(int); ok {
			c.internals.(*textureInternals).textureID = &id
		} else {
			c.internals = hybridInternals{}
		}
	})
}

// TextureID is the texture entry id (nil if none; panics with
// UnimplementedError under hybrid composition).
func (c *SurfaceAndroidViewController) TextureID() *int { return c.internals.getTextureID() }

// RequiresViewComposition is true when it fell back to hybrid composition.
func (c *SurfaceAndroidViewController) RequiresViewComposition() bool {
	return c.internals.requiresViewComposition()
}

func (c *SurfaceAndroidViewController) sendDisposeMessage() async.Future[async.Void] {
	return c.internals.sendDisposeMessage(c.viewID)
}

func (c *SurfaceAndroidViewController) sendResizeMessage(size ui.Size) async.Future[ui.Size] {
	return c.internals.setSize(size, c.viewID, c.state)
}

// SetOffset sets the offset of the platform view.
func (c *SurfaceAndroidViewController) SetOffset(off ui.Offset) async.Future[async.Void] {
	return c.internals.setOffset(off, c.viewID, c.state)
}

// RejectGesture tells the platform that the gesture arena rejected it.
func (c *SurfaceAndroidViewController) RejectGesture(opts ...PlatformViewControllerRejectGestureOptions) async.Future[async.Void] {
	return c.internals.rejectGesture(c.viewID, gestureIDOf(opts))
}

// ExpensiveAndroidViewController controls an Android view that always uses
// hybrid composition.
type ExpensiveAndroidViewController struct {
	AndroidViewControllerBase
	internals androidViewControllerInternals
}

func (c *ExpensiveAndroidViewController) createRequiresSize() bool { return false }

func (c *ExpensiveAndroidViewController) sendCreateMessage(_ *ui.Size, position *ui.Offset) async.Future[async.Void] {
	return async.AsyncVoid(func() {
		androidViewControllerInternalsSendCreateMessage(c.viewID, c.viewType, c.layoutDirection, true,
			false, false, c.creationParams, nil, position).Await()
	})
}

// TextureID panics with UnimplementedError (hybrid composition).
func (c *ExpensiveAndroidViewController) TextureID() *int { return c.internals.getTextureID() }

// RequiresViewComposition is true.
func (c *ExpensiveAndroidViewController) RequiresViewComposition() bool {
	return c.internals.requiresViewComposition()
}

func (c *ExpensiveAndroidViewController) sendDisposeMessage() async.Future[async.Void] {
	return c.internals.sendDisposeMessage(c.viewID)
}

func (c *ExpensiveAndroidViewController) sendResizeMessage(size ui.Size) async.Future[ui.Size] {
	return c.internals.setSize(size, c.viewID, c.state)
}

// SetOffset panics with UnimplementedError (hybrid composition).
func (c *ExpensiveAndroidViewController) SetOffset(off ui.Offset) async.Future[async.Void] {
	return c.internals.setOffset(off, c.viewID, c.state)
}

// RejectGesture tells the platform that the gesture arena rejected it.
func (c *ExpensiveAndroidViewController) RejectGesture(opts ...PlatformViewControllerRejectGestureOptions) async.Future[async.Void] {
	return c.internals.rejectGesture(c.viewID, gestureIDOf(opts))
}

// HybridAndroidViewController controls an Android view composed with the
// newer hybrid composition (platform_views_2).
type HybridAndroidViewController struct {
	AndroidViewControllerBase
	internals androidViewControllerInternals
}

// HybridAndroidViewControllerCheckIfSupported is
// HybridAndroidViewController.checkIfSupported: whether the platform
// supports SurfaceControl hybrid composition.
func HybridAndroidViewControllerCheckIfSupported() async.Future[bool] {
	return hybrid2CheckIfSurfaceControlEnabled()
}

func (c *HybridAndroidViewController) createRequiresSize() bool { return false }

func (c *HybridAndroidViewController) sendCreateMessage(_ *ui.Size, position *ui.Offset) async.Future[async.Void] {
	return async.AsyncVoid(func() {
		androidViewControllerInternalsSendCreateMessage(c.viewID, c.viewType, c.layoutDirection, true,
			false, true, c.creationParams, nil, position).Await()
	})
}

// TextureID panics with UnimplementedError (hybrid composition).
func (c *HybridAndroidViewController) TextureID() *int { return c.internals.getTextureID() }

// RequiresViewComposition is true.
func (c *HybridAndroidViewController) RequiresViewComposition() bool {
	return c.internals.requiresViewComposition()
}

func (c *HybridAndroidViewController) sendDisposeMessage() async.Future[async.Void] {
	return c.internals.sendDisposeMessage(c.viewID)
}

func (c *HybridAndroidViewController) sendResizeMessage(size ui.Size) async.Future[ui.Size] {
	return c.internals.setSize(size, c.viewID, c.state)
}

// SetOffset panics with UnimplementedError (hybrid composition).
func (c *HybridAndroidViewController) SetOffset(off ui.Offset) async.Future[async.Void] {
	return c.internals.setOffset(off, c.viewID, c.state)
}

// SendMotionEvent sends the MotionEvent on SystemChannels.platform_views_2.
func (c *HybridAndroidViewController) SendMotionEvent(event *AndroidMotionEvent) async.Future[async.Void] {
	return async.AsyncVoid(func() {
		SystemChannels.PlatformViews2.InvokeMethod("touch", event.asList(c.viewID)).Await()
	})
}

// RejectGesture tells the platform that the gesture arena rejected it.
func (c *HybridAndroidViewController) RejectGesture(opts ...PlatformViewControllerRejectGestureOptions) async.Future[async.Void] {
	return c.internals.rejectGesture(c.viewID, gestureIDOf(opts))
}

// TextureAndroidViewController controls an Android view rendered into a
// texture (falling back to hybrid composition, platform_views_2).
type TextureAndroidViewController struct {
	AndroidViewControllerBase
	internals androidViewControllerInternals
}

func (c *TextureAndroidViewController) createRequiresSize() bool { return true }

func (c *TextureAndroidViewController) sendCreateMessage(size *ui.Size, position *ui.Offset) async.Future[async.Void] {
	return async.AsyncVoid(func() {
		if size.IsEmpty() {
			panic(dart.NewAssertionError("trying to create TextureAndroidViewController without setting a valid size."))
		}
		response := androidViewControllerInternalsSendCreateMessage(c.viewID, c.viewType, c.layoutDirection, false,
			false, false, c.creationParams, size, position).Await()
		if id, ok := response.(int); ok {
			c.internals.(*textureInternals).textureID = &id
		} else {
			c.internals = hybrid2Internals{}
		}
	})
}

// TextureID is the texture entry id (nil when it fell back to view
// composition).
func (c *TextureAndroidViewController) TextureID() *int {
	if c.internals.requiresViewComposition() {
		return nil
	}
	return c.internals.getTextureID()
}

// RequiresViewComposition is true when it fell back to hybrid composition.
func (c *TextureAndroidViewController) RequiresViewComposition() bool {
	return c.internals.requiresViewComposition()
}

func (c *TextureAndroidViewController) sendDisposeMessage() async.Future[async.Void] {
	return c.internals.sendDisposeMessage(c.viewID)
}

func (c *TextureAndroidViewController) sendResizeMessage(size ui.Size) async.Future[ui.Size] {
	if c.internals.requiresViewComposition() {
		return async.FutureValue(size)
	}
	return c.internals.setSize(size, c.viewID, c.state)
}

// SetOffset sets the offset of the platform view (nothing under view
// composition).
func (c *TextureAndroidViewController) SetOffset(off ui.Offset) async.Future[async.Void] {
	if c.internals.requiresViewComposition() {
		return async.FutureValue(async.Void{})
	}
	return c.internals.setOffset(off, c.viewID, c.state)
}

// RejectGesture tells the platform that the gesture arena rejected it.
func (c *TextureAndroidViewController) RejectGesture(opts ...PlatformViewControllerRejectGestureOptions) async.Future[async.Void] {
	return c.internals.rejectGesture(c.viewID, gestureIDOf(opts))
}

func gestureIDOf(opts []PlatformViewControllerRejectGestureOptions) *int {
	if len(opts) > 0 {
		return opts[0].GestureID
	}
	return nil
}

// androidViewControllerInternals is _AndroidViewControllerInternals.
type androidViewControllerInternals interface {
	getTextureID() *int
	requiresViewComposition() bool
	setSize(size ui.Size, viewID int, viewState androidViewState) async.Future[ui.Size]
	setOffset(offset ui.Offset, viewID int, viewState androidViewState) async.Future[async.Void]
	sendDisposeMessage(viewID int) async.Future[async.Void]
	rejectGesture(viewID int, gestureID *int) async.Future[async.Void]
}

func androidViewControllerInternalsSendCreateMessage(viewID int, viewType string, layoutDirection ui.TextDirection,
	hybrid, hybridFallback, useNewController bool, params *creationParams, size *ui.Size, position *ui.Offset) async.Future[any] {
	args := []dart.MapEntry[string, any]{
		{Key: "id", Value: viewID},
		{Key: "viewType", Value: viewType},
		{Key: "direction", Value: androidViewControllerGetAndroidDirection(layoutDirection)},
	}
	if hybrid {
		args = append(args, dart.MapEntry[string, any]{Key: "hybrid", Value: hybrid})
	}
	if size != nil {
		args = append(args, dart.MapEntry[string, any]{Key: "width", Value: size.Width},
			dart.MapEntry[string, any]{Key: "height", Value: size.Height})
	}
	if hybridFallback {
		args = append(args, dart.MapEntry[string, any]{Key: "hybridFallback", Value: hybridFallback})
	}
	if position != nil {
		args = append(args, dart.MapEntry[string, any]{Key: "left", Value: position.Dx},
			dart.MapEntry[string, any]{Key: "top", Value: position.Dy})
	}
	if params != nil {
		args = append(args, dart.MapEntry[string, any]{Key: "params", Value: params.codec.EncodeMessage(params.data)})
	}
	if useNewController {
		return SystemChannels.PlatformViews2.InvokeMethod("create", args)
	}
	return SystemChannels.PlatformViews.InvokeMethod("create", args)
}

func rejectGestureArgs(viewID int, gestureID *int) []dart.MapEntry[string, any] {
	args := []dart.MapEntry[string, any]{{Key: "id", Value: viewID}}
	if gestureID != nil {
		args = append(args, dart.MapEntry[string, any]{Key: "gestureId", Value: *gestureID})
	}
	return args
}

// defaultRejectGesture is _AndroidViewControllerInternals.rejectGesture.
func defaultRejectGesture(viewID int, gestureID *int) async.Future[async.Void] {
	return async.CastFuture[async.Void](SystemChannels.PlatformViews.InvokeMethod("rejectGesture", rejectGestureArgs(viewID, gestureID)))
}

// textureInternals is _TextureAndroidViewControllerInternals.
type textureInternals struct {
	offset    ui.Offset
	textureID *int
}

func newTextureInternals() *textureInternals { return &textureInternals{offset: ui.OffsetZero} }

func (t *textureInternals) getTextureID() *int            { return t.textureID }
func (t *textureInternals) requiresViewComposition() bool { return false }

func (t *textureInternals) setSize(size ui.Size, viewID int, viewState androidViewState) async.Future[ui.Size] {
	return async.Async(func() ui.Size {
		if viewState == androidViewStateWaitingForSize {
			panic(dart.NewAssertionError("Android view must have an initial size. View id: " + strconv.Itoa(viewID)))
		}
		if size.IsEmpty() {
			panic(dart.NewAssertionError(nil))
		}
		meta := MethodChannelInvokeMapMethod[any, any](SystemChannels.PlatformViews, "resize", []dart.MapEntry[string, any]{
			{Key: "id", Value: viewID},
			{Key: "width", Value: size.Width},
			{Key: "height", Value: size.Height},
		}).Await()
		if meta == nil {
			panic(dart.NewAssertionError(nil))
		}
		width, hasWidth := meta["width"]
		height, hasHeight := meta["height"]
		if !hasWidth || !hasHeight {
			panic(dart.NewAssertionError(nil))
		}
		return ui.NewSize(castElement[float64](width), castElement[float64](height))
	})
}

func (t *textureInternals) setOffset(offset ui.Offset, viewID int, viewState androidViewState) async.Future[async.Void] {
	return async.AsyncVoid(func() {
		if offset == t.offset {
			return
		}
		// Don't set the offset unless the Android view has been created.
		// The implementation of this method channel throws if the Android
		// view for this viewId isn't addressable.
		if viewState != androidViewStateCreated {
			return
		}
		t.offset = offset
		SystemChannels.PlatformViews.InvokeMethod("offset", []dart.MapEntry[string, any]{
			{Key: "id", Value: viewID},
			{Key: "top", Value: offset.Dy},
			{Key: "left", Value: offset.Dx},
		}).Await()
	})
}

func (t *textureInternals) sendDisposeMessage(viewID int) async.Future[async.Void] {
	return async.CastFuture[async.Void](SystemChannels.PlatformViews.InvokeMethod("dispose", []dart.MapEntry[string, any]{
		{Key: "id", Value: viewID},
		{Key: "hybrid", Value: false},
	}))
}

func (t *textureInternals) rejectGesture(viewID int, gestureID *int) async.Future[async.Void] {
	return defaultRejectGesture(viewID, gestureID)
}

const notSupportedForHybrid = "Not supported for hybrid composition."

// hybridInternals is _HybridAndroidViewControllerInternals.
type hybridInternals struct{}

func (hybridInternals) getTextureID() *int {
	panic(dart.UnimplementedError{Message: notSupportedForHybrid})
}
func (hybridInternals) requiresViewComposition() bool { return true }
func (hybridInternals) setSize(ui.Size, int, androidViewState) async.Future[ui.Size] {
	panic(dart.UnimplementedError{Message: notSupportedForHybrid})
}
func (hybridInternals) setOffset(ui.Offset, int, androidViewState) async.Future[async.Void] {
	panic(dart.UnimplementedError{Message: notSupportedForHybrid})
}
func (hybridInternals) sendDisposeMessage(viewID int) async.Future[async.Void] {
	return async.CastFuture[async.Void](SystemChannels.PlatformViews.InvokeMethod("dispose", []dart.MapEntry[string, any]{
		{Key: "id", Value: viewID},
		{Key: "hybrid", Value: true},
	}))
}
func (hybridInternals) rejectGesture(viewID int, gestureID *int) async.Future[async.Void] {
	return defaultRejectGesture(viewID, gestureID)
}

// hybrid2Internals is _Hybrid2AndroidViewControllerInternals.
type hybrid2Internals struct{}

func hybrid2CheckIfSurfaceControlEnabled() async.Future[bool] {
	return async.Async(func() bool {
		result := SystemChannels.PlatformViews2.InvokeMethod("isSurfaceControlEnabled", map[string]any{}).Await()
		if result == nil {
			panic(dart.TypeError{Message: "Null check operator used on a null value"})
		}
		return result.(bool)
	})
}

func (hybrid2Internals) getTextureID() *int {
	panic(dart.UnimplementedError{Message: notSupportedForHybrid})
}
func (hybrid2Internals) requiresViewComposition() bool { return true }
func (hybrid2Internals) setSize(ui.Size, int, androidViewState) async.Future[ui.Size] {
	panic(dart.UnimplementedError{Message: notSupportedForHybrid})
}
func (hybrid2Internals) setOffset(ui.Offset, int, androidViewState) async.Future[async.Void] {
	panic(dart.UnimplementedError{Message: notSupportedForHybrid})
}
func (hybrid2Internals) sendDisposeMessage(viewID int) async.Future[async.Void] {
	return async.CastFuture[async.Void](SystemChannels.PlatformViews2.InvokeMethod("dispose", []dart.MapEntry[string, any]{
		{Key: "id", Value: viewID},
		{Key: "hybrid", Value: true},
	}))
}
func (hybrid2Internals) rejectGesture(viewID int, gestureID *int) async.Future[async.Void] {
	return async.CastFuture[async.Void](SystemChannels.PlatformViews2.InvokeMethod("rejectGesture", rejectGestureArgs(viewID, gestureID)))
}

// DarwinPlatformViewController is the base of the iOS (UiKit) and macOS
// (AppKit) platform view controllers.
type DarwinPlatformViewController struct {
	// ID is the unique identifier of the platform view.
	ID              int
	debugDisposed   bool
	layoutDirection ui.TextDirection
}

// NewDarwinPlatformViewController is DarwinPlatformViewController(id,
// layoutDirection).
func NewDarwinPlatformViewController(id int, layoutDirection ui.TextDirection) *DarwinPlatformViewController {
	return &DarwinPlatformViewController{ID: id, layoutDirection: layoutDirection}
}

// SetLayoutDirection sets the layout direction (not sent to the platform:
// the engine has no message for it).
func (c *DarwinPlatformViewController) SetLayoutDirection(layoutDirection ui.TextDirection) async.Future[async.Void] {
	return async.AsyncVoid(func() {
		if c.debugDisposed {
			panic(dart.NewAssertionError("trying to set a layout direction for a disposed iOS UIView. View id: " + strconv.Itoa(c.ID)))
		}
		if layoutDirection == c.layoutDirection {
			return
		}
		c.layoutDirection = layoutDirection
		// TODO(amirh): invoke the iOS platform views channel direction method once available.
	})
}

// AcceptGesture accepts an active gesture.
func (c *DarwinPlatformViewController) AcceptGesture() async.Future[async.Void] {
	return async.CastFuture[async.Void](SystemChannels.PlatformViews.InvokeMethod("acceptGesture",
		[]dart.MapEntry[string, any]{{Key: "id", Value: c.ID}}))
}

// RejectGesture rejects an active gesture.
func (c *DarwinPlatformViewController) RejectGesture() async.Future[async.Void] {
	return async.CastFuture[async.Void](SystemChannels.PlatformViews.InvokeMethod("rejectGesture",
		[]dart.MapEntry[string, any]{{Key: "id", Value: c.ID}}))
}

// Dispose disposes the view.
func (c *DarwinPlatformViewController) Dispose() async.Future[async.Void] {
	return async.AsyncVoid(func() {
		c.debugDisposed = true
		SystemChannels.PlatformViews.InvokeMethod("dispose", c.ID).Await()
		delete(platformViewsServiceGet().focusCallbacks, c.ID)
	})
}

// UiKitViewGestureBlockingPolicy is how an iOS UIView should block
// gestures.
type UiKitViewGestureBlockingPolicy int

const (
	// UiKitViewGestureBlockingPolicyEager: blocks gestures right away.
	UiKitViewGestureBlockingPolicyEager UiKitViewGestureBlockingPolicy = iota
	// UiKitViewGestureBlockingPolicyWaitUntilTouchesEnded: blocks once all
	// touches end.
	UiKitViewGestureBlockingPolicyWaitUntilTouchesEnded
	// UiKitViewGestureBlockingPolicyDoNotBlockGesture: never blocks.
	UiKitViewGestureBlockingPolicyDoNotBlockGesture
	// UiKitViewGestureBlockingPolicyFallbackToPluginDefault: the plugin's
	// default.
	UiKitViewGestureBlockingPolicyFallbackToPluginDefault
)

// UiKitViewGestureBlockingPolicyValues is UiKitViewGestureBlockingPolicy.values.
var UiKitViewGestureBlockingPolicyValues = []UiKitViewGestureBlockingPolicy{
	UiKitViewGestureBlockingPolicyEager, UiKitViewGestureBlockingPolicyWaitUntilTouchesEnded,
	UiKitViewGestureBlockingPolicyDoNotBlockGesture, UiKitViewGestureBlockingPolicyFallbackToPluginDefault,
}

// Name is the enum value's name.
func (p UiKitViewGestureBlockingPolicy) Name() string {
	return [...]string{"eager", "waitUntilTouchesEnded", "doNotBlockGesture", "fallbackToPluginDefault"}[p]
}

func (p UiKitViewGestureBlockingPolicy) String() string {
	return "UiKitViewGestureBlockingPolicy." + p.Name()
}

// UiKitViewController controls an embedded iOS UIView.
type UiKitViewController struct{ DarwinPlatformViewController }

// AppKitViewController controls an embedded macOS NSView.
type AppKitViewController struct{ DarwinPlatformViewController }
