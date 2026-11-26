package services

// Port of predictive_back_event.dart (Flutter 3.50.1).

import (
	"reflect"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/src/dart"
)

// SwipeEdge is the edge of the screen a predictive back swipe starts
// from.
type SwipeEdge int

const (
	// SwipeEdgeLeft: the swipe starts at the left edge.
	SwipeEdgeLeft SwipeEdge = iota
	// SwipeEdgeRight: the swipe starts at the right edge.
	SwipeEdgeRight
)

// SwipeEdgeValues is SwipeEdge.values.
var SwipeEdgeValues = []SwipeEdge{SwipeEdgeLeft, SwipeEdgeRight}

// Name is the enum value's Dart name.
func (e SwipeEdge) Name() string { return [...]string{"left", "right"}[e] }

func (e SwipeEdge) String() string { return "SwipeEdge." + e.Name() }

// PredictiveBackEvent is an object that represents the state of a
// predictive back gesture (Android). Compare with Equal (Dart's ==).
type PredictiveBackEvent struct {
	// TouchOffset is the global position of the touch (nil for a button
	// press).
	TouchOffset *ui.Offset
	// Progress is a value between 0.0 and 1.0 indicating how far the back
	// gesture has progressed.
	Progress float64
	// SwipeEdge is the screen edge from which the swipe gesture starts.
	SwipeEdge SwipeEdge
}

// NewPredictiveBackEventFromMap is PredictiveBackEvent.fromMap (m is a
// decoded message map, a *dart.LinkedMap, or a Go map or []dart.MapEntry).
func NewPredictiveBackEventFromMap(m any) PredictiveBackEvent {
	var touchOffset *ui.Offset
	if raw, _ := MapLookup(m, "touchOffset"); raw != nil {
		// A List<Object?> (decoded: []any, or a typed list such as
		// []float64).
		list := reflect.ValueOf(raw)
		if list.Kind() != reflect.Slice {
			panic(dart.TypeError{Message: "type '" + dart.ToString(raw) + "' is not a subtype of type 'List<Object?>?' in type cast"})
		}
		if list.Len() < 2 {
			panic(dart.IndexError{Index: list.Len(), Length: list.Len(), Name: "index"})
		}
		o := ui.NewOffset(numToDoubleNonNull(list.Index(0).Interface()), numToDoubleNonNull(list.Index(1).Interface()))
		touchOffset = &o
	}
	progress, _ := MapLookup(m, "progress")
	swipeEdge, _ := MapLookup(m, "swipeEdge")
	if progress == nil || swipeEdge == nil {
		panic(dart.TypeError{Message: "Null check operator used on a null value"})
	}
	p := numToDoubleNonNull(progress)
	if !(p >= 0.0 && p <= 1.0) {
		panic(dart.NewAssertionError(nil))
	}
	edge := castElement[int](swipeEdge)
	if edge < 0 || edge >= len(SwipeEdgeValues) {
		panic(dart.IndexError{Index: edge, Length: len(SwipeEdgeValues), Name: "index"})
	}
	return PredictiveBackEvent{TouchOffset: touchOffset, Progress: p, SwipeEdge: SwipeEdgeValues[edge]}
}

// numToDouble is Dart's `(x! as num).toDouble()`.
func numToDoubleNonNull(v any) float64 {
	switch x := v.(type) {
	case int:
		return float64(x)
	case float64:
		return x
	case nil:
		panic(dart.TypeError{Message: "Null check operator used on a null value"})
	}
	panic(dart.TypeError{Message: "type '" + dart.ToString(v) + "' is not a subtype of type 'num' in type cast"})
}

// IsButtonEvent is whether this event was triggered by a system back
// button press rather than a swipe.
func (e PredictiveBackEvent) IsButtonEvent() bool {
	return e.TouchOffset == nil || (e.Progress == 0.0 && *e.TouchOffset == ui.OffsetZero)
}

// Equal is PredictiveBackEvent's ==.
func (e PredictiveBackEvent) Equal(other PredictiveBackEvent) bool {
	if (e.TouchOffset == nil) != (other.TouchOffset == nil) {
		return false
	}
	if e.TouchOffset != nil && *e.TouchOffset != *other.TouchOffset {
		return false
	}
	return e.Progress == other.Progress && e.SwipeEdge == other.SwipeEdge
}

func (e PredictiveBackEvent) String() string {
	offset := "null"
	if e.TouchOffset != nil {
		offset = e.TouchOffset.String()
	}
	return "PredictiveBackEvent{touchOffset: " + offset + ", progress: " + dart.DoubleToString(e.Progress) +
		", swipeEdge: " + e.SwipeEdge.String() + "}"
}
