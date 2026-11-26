package gestures

// Port of gesture_settings.dart (Flutter 3.50.1).

import (
	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/src/dart"
)

// DeviceGestureSettings are the device-specific gesture settings scaled
// into logical pixels. TouchSlop nil is null (use the default). Compare
// with Equal (Dart's ==).
type DeviceGestureSettings struct {
	// TouchSlop is the touch slop value in logical pixels, or nil.
	TouchSlop *float64
}

// NewDeviceGestureSettingsFromView is DeviceGestureSettings.fromView: the
// view's physical touch slop divided by its device pixel ratio.
func NewDeviceGestureSettingsFromView(view *ui.FlutterView) DeviceGestureSettings {
	physicalTouchSlop := view.GestureSettings().PhysicalTouchSlop
	if physicalTouchSlop == nil {
		return DeviceGestureSettings{}
	}
	touchSlop := *physicalTouchSlop / view.DevicePixelRatio()
	return DeviceGestureSettings{TouchSlop: &touchSlop}
}

// PanSlop is the touch slop value for pan gestures (twice TouchSlop), or
// nil.
func (s DeviceGestureSettings) PanSlop() *float64 {
	if s.TouchSlop == nil {
		return nil
	}
	v := float64(*s.TouchSlop * 2)
	return &v
}

// Equal is ==.
func (s DeviceGestureSettings) Equal(other DeviceGestureSettings) bool {
	if s.TouchSlop == nil || other.TouchSlop == nil {
		return s.TouchSlop == nil && other.TouchSlop == nil
	}
	return *s.TouchSlop == *other.TouchSlop
}

func (s DeviceGestureSettings) String() string {
	slop := "null"
	if s.TouchSlop != nil {
		slop = dart.DoubleToString(*s.TouchSlop)
	}
	return "DeviceGestureSettings(touchSlop: " + slop + ")"
}
