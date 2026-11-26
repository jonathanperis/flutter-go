package ui

// Views, displays and their metrics, from engine/src/flutter/lib/ui/
// platform_dispatcher.dart and hooks.dart (Flutter 3.50.1):
// PlatformDispatcher's views/view/implicitView/displays/onMetricsChanged/
// onViewFocusChange/requestViewFocusChange/engineId, _ViewConfiguration,
// ViewPadding (WindowPadding), ViewConstraints, DisplayFeature (with its
// type and state), DisplayCornerRadii, ViewFocusEvent/State/Direction; and
// the engine entry points _addView, _removeView, _updateWindowMetrics
// (with _buildViewConfiguration, _decodeDisplayFeatures,
// _decodeDisplayCornerRadii), _updateDisplays, _sendViewFocusEvent and
// _setEngineId, which an embedder reaches through PlatformConfiguration
// (window/platform_configuration.cc: AddView, RemoveView,
// UpdateViewMetrics, UpdateDisplays, SendFocusEvent, SetEngineId), with
// the engine's ViewportMetrics and DisplayData.
//
// The implicit view (id 0) exists from the start, as the engine adds it
// before main() runs (RuntimeController::LaunchRootIsolate), with the
// default metrics until the embedder reports its own.

import (
	"errors"
	"iter"
	"math"
	"strconv"

	"github.com/jonathanperis/flutter-go/src/dart"
)

// kImplicitViewID is the implicit view's id (_implicitViewId, the
// engine's kFlutterImplicitViewId).
const kImplicitViewID = 0

// kUnsetGestureSetting is _kUnsetGestureSetting.
const kUnsetGestureSetting = -1.0

// kUnsetDisplayCornerRadius is _kUnsetDisplayCornerRadius.
const kUnsetDisplayCornerRadius = -1.0

// viewConfiguration is _ViewConfiguration.
type viewConfiguration struct {
	displayID           int
	viewConstraints     ViewConstraints
	devicePixelRatio    float64
	size                Size
	viewInsets          ViewPadding
	viewPadding         ViewPadding
	systemGestureInsets ViewPadding
	padding             ViewPadding
	gestureSettings     GestureSettings
	displayFeatures     []DisplayFeature
	displayCornerRadii  *DisplayCornerRadii
}

// newViewConfiguration is const _ViewConfiguration() with its defaults.
func newViewConfiguration() *viewConfiguration {
	return &viewConfiguration{
		devicePixelRatio: 1.0,
		displayFeatures:  []DisplayFeature{},
		viewConstraints:  ViewConstraints{MaxWidth: 0, MaxHeight: 0},
	}
}

func (c *viewConfiguration) String() string {
	return "_ViewConfiguration[size: " + c.size.String() + "]"
}

// platformDispatcherViews is the state of PlatformDispatcher's views and
// displays (Dart's maps keep insertion order: so do ids and displayIDs).
type platformDispatcherViews struct {
	byID              map[int]*FlutterView
	ids               []int
	displays          map[int]*Display
	displayIDs        []int
	onMetricsChanged  VoidCallback
	onViewFocusChange ViewFocusChangeCallback
	engineID          *int
}

func newPlatformDispatcherViews(p *PlatformDispatcher) platformDispatcherViews {
	v := platformDispatcherViews{byID: map[int]*FlutterView{}, displays: map[int]*Display{}}
	v.byID[kImplicitViewID] = &FlutterView{viewID: kImplicitViewID, platformDispatcher: p,
		configuration: newViewConfiguration()}
	v.ids = []int{kImplicitViewID}
	return v
}

// Displays are the displays the engine reported (Dart's Iterable).
func (p *PlatformDispatcher) Displays() iter.Seq[*Display] {
	return func(yield func(*Display) bool) {
		for _, id := range p.views.displayIDs {
			if !yield(p.views.displays[id]) {
				return
			}
		}
	}
}

// Views are the current views, in the order they were added (Dart's
// Iterable).
func (p *PlatformDispatcher) Views() iter.Seq[*FlutterView] {
	return func(yield func(*FlutterView) bool) {
		for _, id := range p.views.ids {
			if !yield(p.views.byID[id]) {
				return
			}
		}
	}
}

// View returns the view with the given id (Dart's required named `id`),
// or nil when there is none.
func (p *PlatformDispatcher) View(id int) *FlutterView { return p.views.byID[id] }

// ImplicitView is the FlutterView provided by the engine on platforms with
// an implicit view (flutter-go's embedders and flutter_tester all have one,
// id 0, so it is never nil).
func (p *PlatformDispatcher) ImplicitView() *FlutterView { return p.views.byID[kImplicitViewID] }

// OnMetricsChanged is called when a view's metrics change, or a view or
// display is added or removed.
func (p *PlatformDispatcher) OnMetricsChanged() VoidCallback { return p.views.onMetricsChanged }

// SetOnMetricsChanged sets OnMetricsChanged.
func (p *PlatformDispatcher) SetOnMetricsChanged(callback VoidCallback) {
	p.views.onMetricsChanged = callback
}

// addView is _addView: called from the engine, via hooks.
func (p *PlatformDispatcher) addView(id int, configuration *viewConfiguration) {
	_, exists := p.views.byID[id]
	dartAssert(!exists, "View ID "+strconv.Itoa(id)+" already exists.")
	if !exists {
		p.views.ids = append(p.views.ids, id)
	}
	p.views.byID[id] = &FlutterView{viewID: id, platformDispatcher: p, configuration: configuration}
	invoke(p.views.onMetricsChanged)
}

// removeView is _removeView: called from the engine, via hooks.
func (p *PlatformDispatcher) removeView(id int) {
	dartAssert(id != kImplicitViewID, "The implicit view #"+strconv.Itoa(id)+" can not be removed.")
	_, exists := p.views.byID[id]
	dartAssert(exists, "View ID "+strconv.Itoa(id)+" does not exist.")
	delete(p.views.byID, id)
	for i, v := range p.views.ids {
		if v == id {
			p.views.ids = append(p.views.ids[:i:i], p.views.ids[i+1:]...)
			break
		}
	}
	invoke(p.views.onMetricsChanged)
}

// sendViewFocusEvent is _sendViewFocusEvent.
func (p *PlatformDispatcher) sendViewFocusEvent(event ViewFocusEvent) {
	if p.views.onViewFocusChange != nil {
		p.views.onViewFocusChange(event)
	}
}

// EngineID is the identifier of the engine running this isolate (nil
// until the engine sets it).
func (p *PlatformDispatcher) EngineID() *int { return p.views.engineID }

// updateDisplays is _updateDisplays (PlatformDispatcher's).
func (p *PlatformDispatcher) updateDisplays(displays []*Display) {
	p.views.displays = map[int]*Display{}
	p.views.displayIDs = nil
	for _, d := range displays {
		if _, ok := p.views.displays[d.ID]; !ok {
			p.views.displayIDs = append(p.views.displayIDs, d.ID)
		}
		p.views.displays[d.ID] = d
	}
	invoke(p.views.onMetricsChanged)
}

// updateWindowMetrics is _updateWindowMetrics (PlatformDispatcher's).
func (p *PlatformDispatcher) updateWindowMetrics(viewID int, configuration *viewConfiguration) {
	view, ok := p.views.byID[viewID]
	dartAssert(ok, "View "+strconv.Itoa(viewID)+" does not exist.")
	view.configuration = configuration
	invoke(p.views.onMetricsChanged)
}

// OnViewFocusChange is called when a view's focus changes (the platform
// moved focus into or out of a view).
func (p *PlatformDispatcher) OnViewFocusChange() ViewFocusChangeCallback {
	return p.views.onViewFocusChange
}

// SetOnViewFocusChange sets OnViewFocusChange.
func (p *PlatformDispatcher) SetOnViewFocusChange(callback ViewFocusChangeCallback) {
	p.views.onViewFocusChange = callback
}

// RequestViewFocusChange asks the platform to change the focus state of
// the view viewID (Dart's required named parameters are positional). The
// request goes to the embedder's PlatformConfigurationViewFocusClient.
func (p *PlatformDispatcher) RequestViewFocusChange(viewID int, state ViewFocusState, direction ViewFocusDirection) {
	if pc := currentPlatformConfiguration; pc != nil {
		if c, ok := pc.client.(PlatformConfigurationViewFocusClient); ok {
			c.RequestViewFocusChange(viewID, state, direction)
		}
	}
}

// ---- ViewPadding

// ViewPadding is a representation of distances for each of the four edges
// of a rectangle, used to encode the view insets and padding that
// applications should place around their user interface (in physical
// pixels). Its constructor is private in Dart: the engine creates them.
type ViewPadding struct {
	// Left is the distance from the left edge to the first unpadded pixel.
	Left float64
	// Top is the distance from the top edge to the first unpadded pixel.
	Top float64
	// Right is the distance from the right edge to the first unpadded pixel.
	Right float64
	// Bottom is the distance from the bottom edge to the first unpadded
	// pixel.
	Bottom float64
}

// WindowPadding is the deprecated name of ViewPadding.
type WindowPadding = ViewPadding

// ViewPaddingZero is ViewPadding.zero: no padding on any side.
var ViewPaddingZero = ViewPadding{}

func (p ViewPadding) String() string {
	return "ViewPadding(left: " + dart.DoubleToString(p.Left) + ", top: " + dart.DoubleToString(p.Top) +
		", right: " + dart.DoubleToString(p.Right) + ", bottom: " + dart.DoubleToString(p.Bottom) + ")"
}

// ---- ViewConstraints

// ViewConstraints are the immutable layout constraints for a FlutterView:
// its width and height must stay within them. Use NewViewConstraints (the
// struct's zero maxima are not Dart's defaults).
type ViewConstraints struct {
	MinWidth, MaxWidth, MinHeight, MaxHeight float64
}

// ViewConstraintsOptions are ViewConstraints' named parameters (nil:
// Dart's default, 0 for the minima and infinity for the maxima).
type ViewConstraintsOptions struct {
	MinWidth, MaxWidth, MinHeight, MaxHeight *float64
}

// NewViewConstraints is ViewConstraints({minWidth, maxWidth, minHeight,
// maxHeight}).
func NewViewConstraints(opts ...ViewConstraintsOptions) ViewConstraints {
	c := ViewConstraints{MaxWidth: math.Inf(1), MaxHeight: math.Inf(1)}
	if len(opts) > 0 {
		o := opts[0]
		if o.MinWidth != nil {
			c.MinWidth = *o.MinWidth
		}
		if o.MaxWidth != nil {
			c.MaxWidth = *o.MaxWidth
		}
		if o.MinHeight != nil {
			c.MinHeight = *o.MinHeight
		}
		if o.MaxHeight != nil {
			c.MaxHeight = *o.MaxHeight
		}
	}
	return c
}

// NewViewConstraintsTight is ViewConstraints.tight(size): exactly size.
func NewViewConstraintsTight(size Size) ViewConstraints {
	return ViewConstraints{MinWidth: size.Width, MaxWidth: size.Width, MinHeight: size.Height, MaxHeight: size.Height}
}

// IsSatisfiedBy is whether size satisfies the constraints.
func (c ViewConstraints) IsSatisfiedBy(size Size) bool {
	return c.MinWidth <= size.Width && size.Width <= c.MaxWidth &&
		c.MinHeight <= size.Height && size.Height <= c.MaxHeight
}

// IsTight is whether exactly one size satisfies the constraints.
func (c ViewConstraints) IsTight() bool {
	return c.MinWidth >= c.MaxWidth && c.MinHeight >= c.MaxHeight
}

// Div is the operator /: every value divided by factor.
func (c ViewConstraints) Div(factor float64) ViewConstraints {
	return ViewConstraints{MinWidth: c.MinWidth / factor, MaxWidth: c.MaxWidth / factor,
		MinHeight: c.MinHeight / factor, MaxHeight: c.MaxHeight / factor}
}

// Equal is ==.
func (c ViewConstraints) Equal(other ViewConstraints) bool { return c == other }

func (c ViewConstraints) String() string {
	if c.MinWidth == math.Inf(1) && c.MinHeight == math.Inf(1) {
		return "ViewConstraints(biggest)"
	}
	if c.MinWidth == 0 && c.MaxWidth == math.Inf(1) && c.MinHeight == 0 && c.MaxHeight == math.Inf(1) {
		return "ViewConstraints(unconstrained)"
	}
	describe := func(min, max float64, dim string) string {
		if min == max {
			return dim + "=" + dart.ToStringAsFixed(min, 1)
		}
		return dart.ToStringAsFixed(min, 1) + "<=" + dim + "<=" + dart.ToStringAsFixed(max, 1)
	}
	return "ViewConstraints(" + describe(c.MinWidth, c.MaxWidth, "w") + ", " +
		describe(c.MinHeight, c.MaxHeight, "h") + ")"
}

// ---- DisplayFeature

// DisplayFeature is an area of a view's display obstructed by a hardware
// feature (a fold, hinge or cutout), in logical pixels.
type DisplayFeature struct {
	// Bounds is the area of the feature.
	Bounds Rect
	// Type is the kind of feature.
	Type DisplayFeatureType
	// State is the posture of the feature (always unknown for cutouts).
	State DisplayFeatureState
}

// NewDisplayFeature is DisplayFeature({bounds, type, state}) (required
// named parameters are positional); a cutout's state must be unknown.
func NewDisplayFeature(bounds Rect, typ DisplayFeatureType, state DisplayFeatureState) DisplayFeature {
	assertNoMessage(typ != DisplayFeatureTypeCutout || state == DisplayFeatureStateUnknown)
	return DisplayFeature{Bounds: bounds, Type: typ, State: state}
}

// Equal is ==.
func (f DisplayFeature) Equal(other DisplayFeature) bool { return f == other }

func (f DisplayFeature) String() string {
	return "DisplayFeature(rect: " + f.Bounds.String() + ", type: " + f.Type.String() +
		", state: " + f.State.String() + ")"
}

// DisplayFeatureType is the type of a DisplayFeature.
type DisplayFeatureType int

const (
	// DisplayFeatureTypeUnknown: the feature isn't known to Flutter.
	DisplayFeatureTypeUnknown DisplayFeatureType = iota
	// DisplayFeatureTypeFold: a fold in a flexible screen, or a seam
	// between two screens.
	DisplayFeatureTypeFold
	// DisplayFeatureTypeHinge: a physical separation with a hinge between
	// two displays.
	DisplayFeatureTypeHinge
	// DisplayFeatureTypeCutout: a non-functional area of the screen (a
	// notch or punch hole).
	DisplayFeatureTypeCutout
)

// DisplayFeatureTypeValues is DisplayFeatureType.values.
var DisplayFeatureTypeValues = []DisplayFeatureType{DisplayFeatureTypeUnknown, DisplayFeatureTypeFold,
	DisplayFeatureTypeHinge, DisplayFeatureTypeCutout}

// Name is the enum value's name ("fold").
func (t DisplayFeatureType) Name() string {
	return [...]string{"unknown", "fold", "hinge", "cutout"}[t]
}

func (t DisplayFeatureType) String() string { return "DisplayFeatureType." + t.Name() }

// DisplayFeatureState is the posture of a DisplayFeature.
type DisplayFeatureState int

const (
	// DisplayFeatureStateUnknown: the posture is unknown (or the feature has
	// none).
	DisplayFeatureStateUnknown DisplayFeatureState = iota
	// DisplayFeatureStatePostureFlat: the device is fully open.
	DisplayFeatureStatePostureFlat
	// DisplayFeatureStatePostureHalfOpened: the device is partially open
	// (tabletop or book posture).
	DisplayFeatureStatePostureHalfOpened
)

// DisplayFeatureStateValues is DisplayFeatureState.values.
var DisplayFeatureStateValues = []DisplayFeatureState{DisplayFeatureStateUnknown,
	DisplayFeatureStatePostureFlat, DisplayFeatureStatePostureHalfOpened}

// Name is the enum value's name ("postureFlat").
func (s DisplayFeatureState) Name() string {
	return [...]string{"unknown", "postureFlat", "postureHalfOpened"}[s]
}

func (s DisplayFeatureState) String() string { return "DisplayFeatureState." + s.Name() }

// ---- DisplayCornerRadii

// DisplayCornerRadii are the radii of a display's rounded corners, in
// physical pixels.
type DisplayCornerRadii struct {
	TopLeft, TopRight, BottomRight, BottomLeft float64
}

// NewDisplayCornerRadii is DisplayCornerRadii({topLeft, topRight,
// bottomRight, bottomLeft}) (positional); every radius must be >= 0.
func NewDisplayCornerRadii(topLeft, topRight, bottomRight, bottomLeft float64) DisplayCornerRadii {
	assertNoMessage(topLeft >= 0)
	assertNoMessage(topRight >= 0)
	assertNoMessage(bottomRight >= 0)
	assertNoMessage(bottomLeft >= 0)
	return DisplayCornerRadii{TopLeft: topLeft, TopRight: topRight, BottomRight: bottomRight, BottomLeft: bottomLeft}
}

// Equal is ==.
func (r DisplayCornerRadii) Equal(other DisplayCornerRadii) bool { return r == other }

func (r DisplayCornerRadii) String() string {
	return "DisplayCornerRadii(topLeft: " + dart.DoubleToString(r.TopLeft) + ", topRight: " +
		dart.DoubleToString(r.TopRight) + ", bottomRight: " + dart.DoubleToString(r.BottomRight) +
		", bottomLeft: " + dart.DoubleToString(r.BottomLeft) + ")"
}

// ---- View focus

// ViewFocusChangeCallback is the signature of
// PlatformDispatcher.onViewFocusChange.
type ViewFocusChangeCallback func(viewFocusEvent ViewFocusEvent)

// ViewFocusEvent is a focus change of a view, sent by the platform.
type ViewFocusEvent struct {
	// ViewID is the view whose focus changed.
	ViewID int
	// State is the view's new focus state.
	State ViewFocusState
	// Direction is the direction in which focus entered or left the view.
	Direction ViewFocusDirection
}

func (e ViewFocusEvent) String() string {
	return "ViewFocusEvent(viewId: " + strconv.Itoa(e.ViewID) + ", state: " + e.State.String() +
		", direction: " + e.Direction.String() + ")"
}

// ViewFocusState is whether a view has input focus.
type ViewFocusState int

const (
	// ViewFocusStateUnfocused: the view has no input focus.
	ViewFocusStateUnfocused ViewFocusState = iota
	// ViewFocusStateFocused: the view has input focus.
	ViewFocusStateFocused
)

// ViewFocusStateValues is ViewFocusState.values.
var ViewFocusStateValues = []ViewFocusState{ViewFocusStateUnfocused, ViewFocusStateFocused}

// Name is the enum value's name ("focused").
func (s ViewFocusState) Name() string { return [...]string{"unfocused", "focused"}[s] }

func (s ViewFocusState) String() string { return "ViewFocusState." + s.Name() }

// ViewFocusDirection is the direction in which focus moved into or out of a
// view.
type ViewFocusDirection int

const (
	// ViewFocusDirectionUndefined: no direction (a click, or a programmatic
	// change).
	ViewFocusDirectionUndefined ViewFocusDirection = iota
	// ViewFocusDirectionForward: forward traversal (tab).
	ViewFocusDirectionForward
	// ViewFocusDirectionBackward: backward traversal (shift-tab).
	ViewFocusDirectionBackward
)

// ViewFocusDirectionValues is ViewFocusDirection.values.
var ViewFocusDirectionValues = []ViewFocusDirection{ViewFocusDirectionUndefined,
	ViewFocusDirectionForward, ViewFocusDirectionBackward}

// Name is the enum value's name ("forward").
func (d ViewFocusDirection) Name() string { return [...]string{"undefined", "forward", "backward"}[d] }

func (d ViewFocusDirection) String() string { return "ViewFocusDirection." + d.Name() }

// ---- Engine side

// ViewportMetrics are a view's metrics as the platform reports them, in
// physical pixels (the engine's flutter::ViewportMetrics,
// lib/ui/window/viewport_metrics.h). NewViewportMetrics gives the
// engine's defaults; a PhysicalTouchSlop or corner radius of -1 is unset.
type ViewportMetrics struct {
	DevicePixelRatio float64
	PhysicalWidth    float64
	PhysicalHeight   float64

	PhysicalMinWidthConstraint  float64
	PhysicalMaxWidthConstraint  float64
	PhysicalMinHeightConstraint float64
	PhysicalMaxHeightConstraint float64

	PhysicalPaddingTop, PhysicalPaddingRight, PhysicalPaddingBottom, PhysicalPaddingLeft         float64
	PhysicalViewInsetTop, PhysicalViewInsetRight, PhysicalViewInsetBottom, PhysicalViewInsetLeft float64

	PhysicalSystemGestureInsetTop, PhysicalSystemGestureInsetRight   float64
	PhysicalSystemGestureInsetBottom, PhysicalSystemGestureInsetLeft float64

	PhysicalTouchSlop float64

	// The display features: four bounds (left, top, right, bottom) per
	// feature, its DisplayFeatureType index and DisplayFeatureState index.
	PhysicalDisplayFeaturesBounds []float64
	PhysicalDisplayFeaturesType   []int
	PhysicalDisplayFeaturesState  []int

	DisplayID int

	PhysicalDisplayCornerRadiusTopLeft     float64
	PhysicalDisplayCornerRadiusTopRight    float64
	PhysicalDisplayCornerRadiusBottomRight float64
	PhysicalDisplayCornerRadiusBottomLeft  float64
}

// NewViewportMetrics is ViewportMetrics() with the engine's defaults
// (device pixel ratio 1, everything else 0, touch slop and corner radii
// unset).
func NewViewportMetrics() ViewportMetrics {
	return ViewportMetrics{
		DevicePixelRatio:                       1.0,
		PhysicalTouchSlop:                      -1.0,
		PhysicalDisplayCornerRadiusTopLeft:     -1.0,
		PhysicalDisplayCornerRadiusTopRight:    -1.0,
		PhysicalDisplayCornerRadiusBottomRight: -1.0,
		PhysicalDisplayCornerRadiusBottomLeft:  -1.0,
	}
}

// DisplayData is a display as the platform reports it (the engine's
// flutter::DisplayData): its id, size in physical pixels, device pixel
// ratio and refresh rate.
type DisplayData struct {
	ID          int
	Width       float64
	Height      float64
	PixelRatio  float64
	RefreshRate float64
}

// decodeDisplayFeatures is hooks.dart's _decodeDisplayFeatures.
func decodeDisplayFeatures(bounds []float64, typ []int, state []int, devicePixelRatio float64) []DisplayFeature {
	dartAssert(float64(len(bounds))/4 == float64(len(typ)), "Bounds are rectangles, requiring 4 measurements each")
	assertNoMessage(len(typ) == len(state))
	result := []DisplayFeature{}
	for i := range typ {
		rectOffset := i * 4
		s := DisplayFeatureStateUnknown
		if state[i] < len(DisplayFeatureStateValues) {
			s = DisplayFeatureStateValues[state[i]]
		}
		result = append(result, NewDisplayFeature(
			NewRectFromLTRB(bounds[rectOffset]/devicePixelRatio, bounds[rectOffset+1]/devicePixelRatio,
				bounds[rectOffset+2]/devicePixelRatio, bounds[rectOffset+3]/devicePixelRatio),
			DisplayFeatureTypeValues[typ[i]], s))
	}
	return result
}

// decodeDisplayCornerRadii is hooks.dart's _decodeDisplayCornerRadii.
func decodeDisplayCornerRadii(topLeft, topRight, bottomRight, bottomLeft float64) *DisplayCornerRadii {
	if debugMode {
		isTopLeftSet := topLeft != kUnsetDisplayCornerRadius
		isConsistent := (topRight != kUnsetDisplayCornerRadius) == isTopLeftSet &&
			(bottomRight != kUnsetDisplayCornerRadius) == isTopLeftSet &&
			(bottomLeft != kUnsetDisplayCornerRadius) == isTopLeftSet
		if !isConsistent {
			panic(dart.ArgumentError{Message: "The display corner radii must be either all set or all unset.\n" +
				"Provided values were inconsistent:\n" +
				"  TopLeft: " + dart.DoubleToString(topLeft) + "\n" +
				"  TopRight: " + dart.DoubleToString(topRight) + "\n" +
				"  BottomRight: " + dart.DoubleToString(bottomRight) + "\n" +
				"  BottomLeft: " + dart.DoubleToString(bottomLeft)})
		}
	}
	if topLeft == kUnsetDisplayCornerRadius || topRight == kUnsetDisplayCornerRadius ||
		bottomRight == kUnsetDisplayCornerRadius || bottomLeft == kUnsetDisplayCornerRadius {
		return nil
	}
	r := NewDisplayCornerRadii(topLeft, topRight, bottomRight, bottomLeft)
	return &r
}

// buildViewConfiguration is hooks.dart's _buildViewConfiguration, from the
// arguments the engine passes (PlatformConfiguration::AddView and
// UpdateViewMetrics).
func buildViewConfiguration(m ViewportMetrics) *viewConfiguration {
	var touchSlop *float64
	if m.PhysicalTouchSlop != kUnsetGestureSetting {
		v := m.PhysicalTouchSlop
		touchSlop = &v
	}
	return &viewConfiguration{
		devicePixelRatio: m.DevicePixelRatio,
		size:             Size{Width: m.PhysicalWidth, Height: m.PhysicalHeight},
		viewPadding: ViewPadding{Top: m.PhysicalPaddingTop, Right: m.PhysicalPaddingRight,
			Bottom: m.PhysicalPaddingBottom, Left: m.PhysicalPaddingLeft},
		viewInsets: ViewPadding{Top: m.PhysicalViewInsetTop, Right: m.PhysicalViewInsetRight,
			Bottom: m.PhysicalViewInsetBottom, Left: m.PhysicalViewInsetLeft},
		padding: ViewPadding{
			Top:    mathMax(0.0, m.PhysicalPaddingTop-m.PhysicalViewInsetTop),
			Right:  mathMax(0.0, m.PhysicalPaddingRight-m.PhysicalViewInsetRight),
			Bottom: mathMax(0.0, m.PhysicalPaddingBottom-m.PhysicalViewInsetBottom),
			Left:   mathMax(0.0, m.PhysicalPaddingLeft-m.PhysicalViewInsetLeft),
		},
		systemGestureInsets: ViewPadding{
			Top:    mathMax(0.0, m.PhysicalSystemGestureInsetTop),
			Right:  mathMax(0.0, m.PhysicalSystemGestureInsetRight),
			Bottom: mathMax(0.0, m.PhysicalSystemGestureInsetBottom),
			Left:   mathMax(0.0, m.PhysicalSystemGestureInsetLeft),
		},
		gestureSettings: GestureSettings{PhysicalTouchSlop: touchSlop},
		displayFeatures: decodeDisplayFeatures(m.PhysicalDisplayFeaturesBounds, m.PhysicalDisplayFeaturesType,
			m.PhysicalDisplayFeaturesState, m.DevicePixelRatio),
		displayID: m.DisplayID,
		viewConstraints: ViewConstraints{MinWidth: m.PhysicalMinWidthConstraint, MaxWidth: m.PhysicalMaxWidthConstraint,
			MinHeight: m.PhysicalMinHeightConstraint, MaxHeight: m.PhysicalMaxHeightConstraint},
		displayCornerRadii: decodeDisplayCornerRadii(m.PhysicalDisplayCornerRadiusTopLeft,
			m.PhysicalDisplayCornerRadiusTopRight, m.PhysicalDisplayCornerRadiusBottomRight,
			m.PhysicalDisplayCornerRadiusBottomLeft),
	}
}

// GetMetrics is the metrics the engine holds for the view viewID
// (PlatformConfiguration::GetMetrics), or nil when there is no such view.
func (pc *PlatformConfiguration) GetMetrics(viewID int) *ViewportMetrics {
	if m, ok := pc.metrics[viewID]; ok {
		return &m
	}
	return nil
}

// AddView adds a view with the given metrics (PlatformConfiguration::
// AddView, then hooks.dart's _addView); false, with the engine's log
// message, when the view already exists. The implicit view exists from
// the start: report its metrics with UpdateViewMetrics.
func (pc *PlatformConfiguration) AddView(viewID int, metrics ViewportMetrics) bool {
	if _, ok := pc.metrics[viewID]; ok {
		engineLog("View #" + strconv.Itoa(viewID) + " already exists.")
		return false
	}
	pc.metrics[viewID] = metrics
	PlatformDispatcherInstance().addView(viewID, buildViewConfiguration(metrics))
	return true
}

// RemoveView removes the view viewID (PlatformConfiguration::RemoveView,
// then hooks.dart's _removeView). The implicit view can't be removed (the
// engine's FML_LOG(FATAL): a panic).
func (pc *PlatformConfiguration) RemoveView(viewID int) bool {
	if viewID == kImplicitViewID {
		panic(errors.New("The implicit view #" + strconv.Itoa(viewID) + " cannot be removed."))
	}
	if _, ok := pc.metrics[viewID]; !ok {
		engineLog("View #" + strconv.Itoa(viewID) + " doesn't exist.")
		return false
	}
	delete(pc.metrics, viewID)
	PlatformDispatcherInstance().removeView(viewID)
	return true
}

// UpdateViewMetrics reports new metrics for the view viewID
// (PlatformConfiguration::UpdateViewMetrics, then hooks.dart's
// _updateWindowMetrics); false when there is no such view.
func (pc *PlatformConfiguration) UpdateViewMetrics(viewID int, metrics ViewportMetrics) bool {
	if _, ok := pc.metrics[viewID]; !ok {
		return false
	}
	pc.metrics[viewID] = metrics
	PlatformDispatcherInstance().updateWindowMetrics(viewID, buildViewConfiguration(metrics))
	return true
}

// UpdateDisplays reports the available displays (PlatformConfiguration::
// UpdateDisplays, then hooks.dart's _updateDisplays).
func (pc *PlatformConfiguration) UpdateDisplays(displays []DisplayData) {
	out := make([]*Display, 0, len(displays))
	for _, d := range displays {
		out = append(out, &Display{ID: d.ID, Size: Size{Width: d.Width, Height: d.Height},
			DevicePixelRatio: d.PixelRatio, RefreshRate: d.RefreshRate})
	}
	PlatformDispatcherInstance().updateDisplays(out)
}

// SendViewFocusEvent reports a view's focus change
// (PlatformConfiguration::SendFocusEvent, hooks.dart's
// _sendViewFocusEvent).
func (pc *PlatformConfiguration) SendViewFocusEvent(event ViewFocusEvent) bool {
	PlatformDispatcherInstance().sendViewFocusEvent(ViewFocusEvent{ViewID: event.ViewID,
		State: ViewFocusStateValues[event.State], Direction: ViewFocusDirectionValues[event.Direction]})
	return true
}

// SetEngineID sets the engine's identifier (PlatformConfiguration::
// SetEngineId, hooks.dart's _setEngineId).
func (pc *PlatformConfiguration) SetEngineID(engineID int) bool {
	PlatformDispatcherInstance().views.engineID = &engineID
	return true
}

// PlatformConfigurationViewFocusClient is the part of the engine's
// PlatformConfigurationClient that receives
// PlatformDispatcher.requestViewFocusChange; an embedder's client
// implements it to get them (otherwise the request is dropped, as on
// platforms without view focus).
type PlatformConfigurationViewFocusClient interface {
	RequestViewFocusChange(viewID int, state ViewFocusState, direction ViewFocusDirection)
}
