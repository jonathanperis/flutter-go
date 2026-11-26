package ui

// From platform_dispatcher.dart and hooks.dart (Flutter 3.50.1): the
// engine's hit test requests (PlatformDispatcher.onHitTest, HitTestRequest,
// HitTestResponse, hooks.dart's _hitTest as PlatformConfiguration.HitTest).

import (
	"strconv"

	"github.com/jonathanperis/flutter-go/src/dart"
)

// HitTestCallback is the signature of PlatformDispatcher.onHitTest.
type HitTestCallback func(request HitTestRequest) HitTestResponse

// HitTestRequest is a request to hit test a view at a position.
type HitTestRequest struct {
	// View is the view to hit test.
	View *FlutterView
	// Offset is the position in the view, in logical pixels.
	Offset Offset
}

// HitTestResponse is the result of a hit test request.
type HitTestResponse struct {
	// HasPlatformView reports whether a platform view was hit.
	HasPlatformView bool
}

// HitTestResponseEmpty is HitTestResponse.empty.
var HitTestResponseEmpty = HitTestResponse{}

// onHitTest holds each dispatcher's onHitTest (PlatformDispatcher's
// fields are in other files).
var onHitTest = map[*PlatformDispatcher]HitTestCallback{}

// OnHitTest is called when the platform asks whether a position of a view
// hits a platform view.
func (p *PlatformDispatcher) OnHitTest() HitTestCallback { return onHitTest[p] }

// SetOnHitTest sets OnHitTest.
func (p *PlatformDispatcher) SetOnHitTest(callback HitTestCallback) {
	if callback == nil {
		delete(onHitTest, p)
		return
	}
	onHitTest[p] = callback
}

func (p *PlatformDispatcher) hitTest(request HitTestRequest) HitTestResponse {
	if cb := p.OnHitTest(); cb != nil {
		return cb(request)
	}
	return HitTestResponseEmpty
}

// HitTest is the engine's hit test (hooks.dart's _hitTest): a request for
// the position (x, y) of the view viewID. Only the implicit view exists.
func (pc *PlatformConfiguration) HitTest(viewID int, x, y float64) HitTestResponse {
	pd := PlatformDispatcherInstance()
	view := pd.ImplicitView()
	if view.ViewID() != viewID {
		panic(dart.NewAssertionError("View " + strconv.Itoa(viewID) + " does not exist."))
	}
	return pd.hitTest(HitTestRequest{View: view, Offset: NewOffset(x, y)})
}
