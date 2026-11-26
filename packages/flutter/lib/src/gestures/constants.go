// Package gestures is Flutter's gestures library: pointer events and their
// conversion from dart:ui's PointerData, hit testing, the pointer router,
// the gesture arena, velocity tracking and the gesture recognizers.
package gestures

// Port of constants.dart (Flutter 3.50.1). Modeled after Android's
// ViewConfiguration.

import "time"

// KPressTimeout is the time that must elapse before a tap gesture sends
// onTapDown, if there's any doubt that the gesture is a tap.
const KPressTimeout = 100 * time.Millisecond

// KHoverTapTimeout is the maximum time from the start of the first tap to
// the start of the second tap in a double-tap gesture (hover).
const KHoverTapTimeout = 150 * time.Millisecond

// KHoverTapSlop is the maximum distance between the first touch and the
// second touch in a hover-tap gesture (logical pixels).
const KHoverTapSlop = 20.0

// KLongPressTimeout is the time before a long press gesture attempts to
// win.
const KLongPressTimeout = 500 * time.Millisecond

// KDoubleTapTimeout is the maximum time from the start of the first tap to
// the start of the second tap in a double-tap gesture.
const KDoubleTapTimeout = 300 * time.Millisecond

// KDoubleTapMinTime is the minimum time from the end of the first tap to
// the start of the second tap in a double-tap gesture.
const KDoubleTapMinTime = 40 * time.Millisecond

// KDoubleTapTouchSlop is the maximum distance that the first touch in a
// double-tap gesture can travel before deciding that it is not part of a
// double-tap gesture (logical pixels).
const KDoubleTapTouchSlop = KTouchSlop

// KDoubleTapSlop is the maximum distance between the first touch and the
// second touch in a double-tap gesture (logical pixels).
const KDoubleTapSlop = 100.0

// KZoomControlsTimeout is the time for which zoom controls are visible.
const KZoomControlsTimeout = 3000 * time.Millisecond

// KTouchSlop is the distance a touch can travel before the framework
// becomes confident that it is not a tap (logical pixels).
const KTouchSlop = 18.0

// KPagingTouchSlop is the distance a touch can travel before the framework
// becomes confident that it is not a paging tap (logical pixels).
const KPagingTouchSlop = KTouchSlop * 2.0

// KPanSlop is the distance a touch can travel before the framework becomes
// confident that it is not a tap (pan; logical pixels).
const KPanSlop = KTouchSlop * 2.0

// KScaleSlop is the absolute change in span before a scale is recognized
// (logical pixels).
const KScaleSlop = KTouchSlop

// KWindowTouchSlop is the margin around a dialog, popup menu or other
// window-like widget inside which a tap does not dismiss it (logical
// pixels).
const KWindowTouchSlop = 16.0

// KMinFlingVelocity is the minimum velocity for a touch to be considered a
// fling (logical pixels per second).
const KMinFlingVelocity = 50.0

// KMaxFlingVelocity is the maximum velocity of a fling (logical pixels per
// second).
const KMaxFlingVelocity = 8000.0

// KJumpTapTimeout is the maximum time from the start of the first tap to
// the start of the second tap in a jump-tap gesture.
const KJumpTapTimeout = 500 * time.Millisecond

// KPrecisePointerHitSlop is like KTouchSlop, for more precise pointers
// (mice, trackpads; logical pixels).
const KPrecisePointerHitSlop = 1.0

// KPrecisePointerPanSlop is like KPanSlop, for more precise pointers.
const KPrecisePointerPanSlop = KPrecisePointerHitSlop * 2.0

// KPrecisePointerScaleSlop is like KScaleSlop, for more precise pointers.
const KPrecisePointerScaleSlop = KPrecisePointerHitSlop
