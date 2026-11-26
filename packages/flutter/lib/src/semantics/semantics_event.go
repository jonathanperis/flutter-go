package semantics

// Port of packages/flutter/lib/src/semantics/semantics_event.dart (Flutter
// 3.50.1): SemanticsEvent and the events Flutter sends.
//
// Dart's Map<String, dynamic> results (toMap, getDataMap) are ordered maps,
// []dart.MapEntry[string, any], since their order reaches the platform
// (StandardMessageCodec writes entries in order).

import (
	"strconv"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	"github.com/jonathanperis/flutter-go/src/dart"
)

// Assertiveness determines the assertiveness level of the accessibility
// announcement.
type Assertiveness int

const (
	// AssertivenessPolite: the assistive technology will speak changes
	// whenever the user is idle.
	AssertivenessPolite Assertiveness = iota
	// AssertivenessAssertive: the assistive technology will interrupt any
	// announcement that it is currently making.
	AssertivenessAssertive
)

// AssertivenessValues is Assertiveness.values.
var AssertivenessValues = []Assertiveness{AssertivenessPolite, AssertivenessAssertive}

// Name is the enum value's name.
func (a Assertiveness) Name() string { return [...]string{"polite", "assertive"}[a] }

func (a Assertiveness) String() string { return "Assertiveness." + a.Name() }

// SemanticsEvent is an event sent by the application to notify interested
// listeners that something happened to the user interface (e.g. a view
// scrolled). Implement it by embedding SemanticsEventBase (NewSemanticsEventBase
// with the type, then SetSelf) and defining GetDataMap.
type SemanticsEvent interface {
	// Type is the type of this event; the platform uses it to tell events
	// apart.
	Type() string
	// ToMap converts this event to a map suitable for encoding with
	// StandardMessageCodec ({type, data, nodeId}).
	ToMap(opts ...SemanticsEventToMapOptions) []dart.MapEntry[string, any]
	// GetDataMap returns the event's data object.
	GetDataMap() []dart.MapEntry[string, any]
	String() string
}

// SemanticsEventToMapOptions are toMap's named parameters (NodeID nil: no
// nodeId entry).
type SemanticsEventToMapOptions struct{ NodeID *int }

// SemanticsEventBase is the abstract class SemanticsEvent's state and
// default methods. A type embedding it calls SetSelf(outer).
type SemanticsEventBase struct {
	typ  string
	self SemanticsEvent
}

// NewSemanticsEventBase is SemanticsEvent(type), for embedding.
func NewSemanticsEventBase(typ string) SemanticsEventBase { return SemanticsEventBase{typ: typ} }

// SetSelf points the base at the event that embeds it.
func (e *SemanticsEventBase) SetSelf(self SemanticsEvent) { e.self = self }

func (e *SemanticsEventBase) outer() SemanticsEvent {
	if e.self == nil {
		panic("semantics: SemanticsEventBase used before SetSelf; call SetSelf with the outer event when creating it")
	}
	return e.self
}

// Type is the type of this event.
func (e *SemanticsEventBase) Type() string { return e.typ }

// ToMap is {type, data, nodeId?}.
func (e *SemanticsEventBase) ToMap(opts ...SemanticsEventToMapOptions) []dart.MapEntry[string, any] {
	event := []dart.MapEntry[string, any]{
		{Key: "type", Value: e.typ},
		{Key: "data", Value: e.outer().GetDataMap()},
	}
	if len(opts) > 0 && opts[0].NodeID != nil {
		event = append(event, dart.MapEntry[string, any]{Key: "nodeId", Value: *opts[0].NodeID})
	}
	return event
}

// String is "Type(key: value, ...)" with the data map's keys sorted.
func (e *SemanticsEventBase) String() string {
	self := e.outer()
	dataMap := append([]dart.MapEntry[string, any](nil), self.GetDataMap()...)
	dart.ListSort(dataMap, func(a, b dart.MapEntry[string, any]) int { return dart.CompareStrings(a.Key, b.Key) })
	pairs := ""
	for i, entry := range dataMap {
		if i > 0 {
			pairs += ", "
		}
		pairs += entry.Key + ": " + valueString(entry.Value)
	}
	return foundation.ObjectRuntimeType(self, "SemanticsEvent") + "(" + pairs + ")"
}

// valueString is Dart's toString of the values events carry.
func valueString(v any) string {
	switch x := v.(type) {
	case nil:
		return "null"
	case int:
		return strconv.Itoa(x)
	case float64:
		return dart.DoubleToString(x)
	case bool:
		return strconv.FormatBool(x)
	case string:
		return x
	case interface{ String() string }:
		return x.String()
	}
	return foundation.DescribeIdentity(v)
}

// AnnounceSemanticsEvent is an event for a semantic announcement.
type AnnounceSemanticsEvent struct {
	SemanticsEventBase
	// ViewID is the view that this announcement is on.
	ViewID int
	// Message is the message to announce.
	Message string
	// TextDirection is the text direction of the message.
	TextDirection ui.TextDirection
	// Assertiveness determines whether the announcement should interrupt
	// any existing announcement, or queue after it.
	Assertiveness Assertiveness
}

// AnnounceSemanticsEventOptions are AnnounceSemanticsEvent's optional named
// parameters (Assertiveness defaults to polite).
type AnnounceSemanticsEventOptions struct{ Assertiveness Assertiveness }

// NewAnnounceSemanticsEvent is AnnounceSemanticsEvent(message,
// textDirection, viewId, {assertiveness}).
func NewAnnounceSemanticsEvent(message string, textDirection ui.TextDirection, viewID int, opts ...AnnounceSemanticsEventOptions) *AnnounceSemanticsEvent {
	var o AnnounceSemanticsEventOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	e := &AnnounceSemanticsEvent{SemanticsEventBase: NewSemanticsEventBase("announce"), ViewID: viewID,
		Message: message, TextDirection: textDirection, Assertiveness: o.Assertiveness}
	e.SetSelf(e)
	return e
}

// GetDataMap is {viewId, message, textDirection (index), assertiveness
// (index, when not polite)}.
func (e *AnnounceSemanticsEvent) GetDataMap() []dart.MapEntry[string, any] {
	m := []dart.MapEntry[string, any]{
		{Key: "viewId", Value: e.ViewID},
		{Key: "message", Value: e.Message},
		{Key: "textDirection", Value: int(e.TextDirection)},
	}
	if e.Assertiveness != AssertivenessPolite {
		m = append(m, dart.MapEntry[string, any]{Key: "assertiveness", Value: int(e.Assertiveness)})
	}
	return m
}

// TooltipSemanticsEvent is an event for a semantic announcement of a
// tooltip.
type TooltipSemanticsEvent struct {
	SemanticsEventBase
	// Message is the text content of the tooltip.
	Message string
}

// NewTooltipSemanticsEvent is TooltipSemanticsEvent(message).
func NewTooltipSemanticsEvent(message string) *TooltipSemanticsEvent {
	e := &TooltipSemanticsEvent{SemanticsEventBase: NewSemanticsEventBase("tooltip"), Message: message}
	e.SetSelf(e)
	return e
}

// GetDataMap is {message}.
func (e *TooltipSemanticsEvent) GetDataMap() []dart.MapEntry[string, any] {
	return []dart.MapEntry[string, any]{{Key: "message", Value: e.Message}}
}

// LongPressSemanticsEvent is an event which triggers long press semantic
// feedback.
type LongPressSemanticsEvent struct{ SemanticsEventBase }

// NewLongPressSemanticsEvent is LongPressSemanticsEvent().
func NewLongPressSemanticsEvent() *LongPressSemanticsEvent {
	e := &LongPressSemanticsEvent{SemanticsEventBase: NewSemanticsEventBase("longPress")}
	e.SetSelf(e)
	return e
}

// GetDataMap is empty.
func (e *LongPressSemanticsEvent) GetDataMap() []dart.MapEntry[string, any] {
	return []dart.MapEntry[string, any]{}
}

// TapSemanticEvent is an event which triggers tap semantic feedback.
type TapSemanticEvent struct{ SemanticsEventBase }

// NewTapSemanticEvent is TapSemanticEvent().
func NewTapSemanticEvent() *TapSemanticEvent {
	e := &TapSemanticEvent{SemanticsEventBase: NewSemanticsEventBase("tap")}
	e.SetSelf(e)
	return e
}

// GetDataMap is empty.
func (e *TapSemanticEvent) GetDataMap() []dart.MapEntry[string, any] {
	return []dart.MapEntry[string, any]{}
}

// FocusSemanticEvent is an event to move the accessibility focus.
type FocusSemanticEvent struct{ SemanticsEventBase }

// NewFocusSemanticEvent is FocusSemanticEvent().
func NewFocusSemanticEvent() *FocusSemanticEvent {
	e := &FocusSemanticEvent{SemanticsEventBase: NewSemanticsEventBase("focus")}
	e.SetSelf(e)
	return e
}

// GetDataMap is empty.
func (e *FocusSemanticEvent) GetDataMap() []dart.MapEntry[string, any] {
	return []dart.MapEntry[string, any]{}
}
