package services

// Port of packages/flutter/lib/src/services/mouse_tracking.dart (Flutter
// 3.50.1).

import (
	"strings"

	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/gestures"
	"github.com/jonathanperis/flutter-go/src/dart"
)

// PointerEnterEventListener is the signature for listening to
// PointerEnterEvent events.
type PointerEnterEventListener func(event *gestures.PointerEnterEvent)

// PointerExitEventListener is the signature for listening to
// PointerExitEvent events.
type PointerExitEventListener func(event *gestures.PointerExitEvent)

// PointerHoverEventListener is the signature for listening to
// PointerHoverEvent events.
type PointerHoverEventListener func(event *gestures.PointerHoverEvent)

// MouseTrackerAnnotation is the annotation object used to annotate regions
// that are interested in mouse movements. Dart's class is concrete and also
// implemented (rendering's RenderMouseRegion), so it is an interface; the
// concrete class is MouseTrackerAnnotationBase (NewMouseTrackerAnnotation),
// which a type implementing the interface may embed (SetSelf).
type MouseTrackerAnnotation interface {
	foundation.Diagnosticable
	// OnEnter is triggered when a mouse pointer enters the region (nil:
	// none).
	OnEnter() PointerEnterEventListener
	// OnExit is triggered when a mouse pointer exits the region (nil:
	// none).
	OnExit() PointerExitEventListener
	// Cursor is the mouse cursor for mouse pointers hovering over the
	// region.
	Cursor() MouseCursor
	// ValidForMouseTracker is whether this is included when MouseTracker
	// collects the annotations.
	ValidForMouseTracker() bool
}

// MouseTrackerAnnotationOptions are MouseTrackerAnnotation's named
// parameters (Cursor nil: MouseCursorDefer; ValidForMouseTracker nil:
// true).
type MouseTrackerAnnotationOptions struct {
	OnEnter              PointerEnterEventListener
	OnExit               PointerExitEventListener
	Cursor               MouseCursor
	ValidForMouseTracker *bool
}

// MouseTrackerAnnotationBase is the concrete MouseTrackerAnnotation.
type MouseTrackerAnnotationBase struct {
	foundation.DiagnosticableMixin
	self                 MouseTrackerAnnotation
	onEnter              PointerEnterEventListener
	onExit               PointerExitEventListener
	cursor               MouseCursor
	validForMouseTracker bool
}

// NewMouseTrackerAnnotation is MouseTrackerAnnotation({onEnter, onExit,
// cursor, validForMouseTracker}).
func NewMouseTrackerAnnotation(opts ...MouseTrackerAnnotationOptions) *MouseTrackerAnnotationBase {
	a := &MouseTrackerAnnotationBase{}
	a.Init(a, opts...)
	return a
}

// Init sets up an embedded base (Dart's constructor) for the annotation
// self that embeds it.
func (a *MouseTrackerAnnotationBase) Init(self MouseTrackerAnnotation, opts ...MouseTrackerAnnotationOptions) {
	var o MouseTrackerAnnotationOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	a.SetSelf(self)
	a.onEnter, a.onExit, a.cursor, a.validForMouseTracker = o.OnEnter, o.OnExit, o.Cursor, true
	if a.cursor == nil {
		a.cursor = MouseCursorDefer
	}
	if o.ValidForMouseTracker != nil {
		a.validForMouseTracker = *o.ValidForMouseTracker
	}
}

// SetSelf points the base at the annotation it is part of.
func (a *MouseTrackerAnnotationBase) SetSelf(self MouseTrackerAnnotation) {
	a.self = self
	a.DiagnosticableMixin.SetSelf(self)
}

// OnEnter is triggered when a mouse pointer enters the region.
func (a *MouseTrackerAnnotationBase) OnEnter() PointerEnterEventListener { return a.onEnter }

// OnExit is triggered when a mouse pointer exits the region.
func (a *MouseTrackerAnnotationBase) OnExit() PointerExitEventListener { return a.onExit }

// Cursor is the mouse cursor for mouse pointers hovering over the region.
func (a *MouseTrackerAnnotationBase) Cursor() MouseCursor { return a.cursor }

// ValidForMouseTracker is whether this is included when MouseTracker
// collects the annotations.
func (a *MouseTrackerAnnotationBase) ValidForMouseTracker() bool { return a.validForMouseTracker }

// ToStringShort prints the Dart class name for the base itself.
func (a *MouseTrackerAnnotationBase) ToStringShort() string {
	if a.self == MouseTrackerAnnotation(a) {
		return strings.TrimSuffix(foundation.ObjectRuntimeType(a, "<optimized out>"), "Base") + "#" + foundation.ShortHash(a)
	}
	return a.DiagnosticableMixin.ToStringShort()
}

// DebugFillProperties adds the callbacks and the cursor.
func (a *MouseTrackerAnnotationBase) DebugFillProperties(properties *foundation.DiagnosticPropertiesBuilder) {
	a.DiagnosticableMixin.DebugFillProperties(properties)
	var enter, exit any
	if a.self.OnEnter() != nil {
		enter = a.self.OnEnter()
	}
	if a.self.OnExit() != nil {
		exit = a.self.OnExit()
	}
	ifEmpty := "<none>"
	properties.Add(foundation.NewFlagsSummary("callbacks", []dart.MapEntry[string, any]{
		{Key: "enter", Value: enter}, {Key: "exit", Value: exit},
	}, foundation.FlagsSummaryOptions{IfEmpty: &ifEmpty}))
	properties.Add(foundation.NewDiagnosticsProperty("cursor", a.self.Cursor(),
		foundation.DiagnosticsPropertyOptions{DefaultValue: MouseCursorDefer}))
}
