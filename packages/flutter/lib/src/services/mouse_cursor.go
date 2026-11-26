package services

// Port of packages/flutter/lib/src/services/mouse_cursor.dart (Flutter
// 3.50.1): MouseCursorManager, MouseCursor, MouseCursorSession,
// SystemMouseCursor and SystemMouseCursors.

import (
	"iter"

	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/gestures"
	"github.com/jonathanperis/flutter-go/src/async"
	"github.com/jonathanperis/flutter-go/src/dart"
)

// MouseCursorManager maintains the relationship between mouse devices and
// their cursor sessions.
type MouseCursorManager struct {
	// FallbackMouseCursor is the mouse cursor used when no candidate
	// specifies one (it must not be MouseCursorDefer).
	FallbackMouseCursor MouseCursor

	lastSession map[int]MouseCursorSession
}

// NewMouseCursorManager is MouseCursorManager(fallbackMouseCursor).
func NewMouseCursorManager(fallbackMouseCursor MouseCursor) *MouseCursorManager {
	if fallbackMouseCursor == MouseCursorDefer {
		panic(dart.NewAssertionError(nil))
	}
	return &MouseCursorManager{FallbackMouseCursor: fallbackMouseCursor, lastSession: map[int]MouseCursorSession{}}
}

// DebugDeviceActiveCursor returns the active mouse cursor of a device (nil
// when it has none, or outside debug builds).
func (m *MouseCursorManager) DebugDeviceActiveCursor(device int) MouseCursor {
	if !foundation.KDebugMode {
		return nil
	}
	if s, ok := m.lastSession[device]; ok {
		return s.Cursor()
	}
	return nil
}

// HandleDeviceCursorUpdate changes the mouse cursor of a device to the
// first non-deferring candidate (or the fallback), creating and activating
// a new session if it differs from the device's current cursor. A
// PointerRemovedEvent ends the device's session.
func (m *MouseCursorManager) HandleDeviceCursorUpdate(device int, triggeringEvent gestures.PointerEvent, cursorCandidates iter.Seq[MouseCursor]) {
	if _, removed := triggeringEvent.(*gestures.PointerRemovedEvent); removed {
		delete(m.lastSession, device)
		return
	}
	lastSession, hasLast := m.lastSession[device]
	nextCursor := deferringMouseCursorFirstNonDeferred(cursorCandidates)
	if nextCursor == nil {
		nextCursor = m.FallbackMouseCursor
	}
	if _, deferring := nextCursor.(*deferringMouseCursor); deferring {
		panic(dart.NewAssertionError(nil))
	}
	if hasLast && mouseCursorEquals(lastSession.Cursor(), nextCursor) {
		return
	}
	nextSession := nextCursor.CreateSession(device)
	m.lastSession[device] = nextSession
	if hasLast {
		lastSession.Dispose()
	}
	nextSession.Activate()
}

// mouseCursorEquals is MouseCursor's ==: a cursor with an Equal method
// (SystemMouseCursor) compares by value, others by identity.
func mouseCursorEquals(a, b MouseCursor) bool {
	if e, ok := a.(interface{ Equal(MouseCursor) bool }); ok {
		return e.Equal(b)
	}
	return a == b
}

// MouseCursorSession manages the duration that a mouse cursor is active on
// a device. Implementations embed MouseCursorSessionBase (Cursor, Device)
// and implement Activate and Dispose.
type MouseCursorSession interface {
	// Cursor is the cursor that created this session.
	Cursor() MouseCursor
	// Device is the device ID where the cursor is being displayed.
	Device() int
	// Activate is called once when the session starts; it returns once the
	// platform has activated the cursor.
	Activate() async.Future[async.Void]
	// Dispose is called once when the session ends.
	Dispose()
}

// MouseCursorSessionBase holds a session's cursor and device (Dart's
// MouseCursorSession constructor).
type MouseCursorSessionBase struct {
	cursor MouseCursor
	device int
}

// NewMouseCursorSessionBase is MouseCursorSession(cursor, device), for
// embedding.
func NewMouseCursorSessionBase(cursor MouseCursor, device int) MouseCursorSessionBase {
	return MouseCursorSessionBase{cursor: cursor, device: device}
}

// Cursor is the cursor that created this session.
func (s *MouseCursorSessionBase) Cursor() MouseCursor { return s.cursor }

// Device is the device ID where the cursor is being displayed.
func (s *MouseCursorSessionBase) Device() int { return s.device }

// MouseCursor is an interface for mouse cursor definitions. Implementations
// embed MouseCursorBase (and call SetSelf), and implement CreateSession and
// DebugDescription.
type MouseCursor interface {
	foundation.Diagnosticable
	// CreateSession associates this cursor with the given device (protected
	// in Dart).
	CreateSession(device int) MouseCursorSession
	// DebugDescription is a very short description of the mouse cursor.
	DebugDescription() string
}

// MouseCursorBase is MouseCursor's Diagnosticable part and its toString.
type MouseCursorBase struct {
	foundation.DiagnosticableMixin
	self MouseCursor
}

// SetSelf points the base at the cursor it is part of.
func (c *MouseCursorBase) SetSelf(self MouseCursor) {
	c.self = self
	c.DiagnosticableMixin.SetSelf(self)
}

// String is toString().
func (c *MouseCursorBase) String() string { return c.self.ToString() }

// ToString is the debug description at minLevel info or above, the
// Diagnosticable description otherwise.
func (c *MouseCursorBase) ToString(opts ...foundation.DiagnosticableToStringOptions) string {
	minLevel := foundation.DiagnosticLevelInfo
	if len(opts) > 0 && opts[0].MinLevel != nil {
		minLevel = *opts[0].MinLevel
	}
	debugDescription := c.self.DebugDescription()
	if minLevel >= foundation.DiagnosticLevelInfo {
		return debugDescription
	}
	return c.DiagnosticableMixin.ToString(opts...)
}

// MouseCursorDefer is MouseCursor.defer: a special value that defers to the
// next region (it can't create sessions).
var MouseCursorDefer MouseCursor = newDeferringMouseCursor()

// MouseCursorUncontrolled is MouseCursor.uncontrolled: a cursor that
// doesn't change the cursor when activated.
var MouseCursorUncontrolled MouseCursor = newNoopMouseCursor()

type deferringMouseCursor struct{ MouseCursorBase }

func newDeferringMouseCursor() *deferringMouseCursor {
	c := &deferringMouseCursor{}
	c.SetSelf(c)
	return c
}

func (c *deferringMouseCursor) CreateSession(int) MouseCursorSession {
	panic(dart.NewAssertionError("_DeferringMouseCursor can not create a session"))
}

func (c *deferringMouseCursor) DebugDescription() string { return "defer" }

// ToStringShort is describeIdentity with the Dart class name.
func (c *deferringMouseCursor) ToStringShort() string {
	return privateIdentity(c, "_DeferringMouseCursor")
}

// privateIdentity is describeIdentity of an object of a private Dart class
// (Go's unexported type names differ from Dart's).
func privateIdentity(object any, dartName string) string {
	if !foundation.KDebugMode {
		return foundation.DescribeIdentity(object)
	}
	return dartName + "#" + foundation.ShortHash(object)
}

func deferringMouseCursorFirstNonDeferred(cursors iter.Seq[MouseCursor]) MouseCursor {
	for cursor := range cursors {
		if cursor != MouseCursorDefer {
			return cursor
		}
	}
	return nil
}

type noopMouseCursorSession struct{ MouseCursorSessionBase }

func (s *noopMouseCursorSession) Activate() async.Future[async.Void] {
	return async.AsyncVoid(func() { /* Nothing */ })
}

func (s *noopMouseCursorSession) Dispose() { /* Nothing */ }

type noopMouseCursor struct{ MouseCursorBase }

func newNoopMouseCursor() *noopMouseCursor {
	c := &noopMouseCursor{}
	c.SetSelf(c)
	return c
}

func (c *noopMouseCursor) CreateSession(device int) MouseCursorSession {
	return &noopMouseCursorSession{NewMouseCursorSessionBase(c, device)}
}

func (c *noopMouseCursor) DebugDescription() string { return "uncontrolled" }

// ToStringShort is describeIdentity with the Dart class name.
func (c *noopMouseCursor) ToStringShort() string { return privateIdentity(c, "_NoopMouseCursor") }

type systemMouseCursorSession struct{ MouseCursorSessionBase }

func (s *systemMouseCursorSession) Activate() async.Future[async.Void] {
	return async.CastFuture[async.Void](SystemChannels.MouseCursor.InvokeMethod("activateSystemCursor",
		[]dart.MapEntry[string, any]{
			{Key: "device", Value: s.Device()},
			{Key: "kind", Value: s.Cursor().(*SystemMouseCursor).Kind},
		}))
}

func (s *systemMouseCursorSession) Dispose() { /* Nothing */ }

// SystemMouseCursor is a mouse cursor that is natively supported on the
// platform the application is running on. Its instances are the fields of
// SystemMouseCursors; == is Equal (by kind).
type SystemMouseCursor struct {
	MouseCursorBase
	// Kind is a string that identifies the kind of the cursor.
	Kind string
}

func newSystemMouseCursor(kind string) *SystemMouseCursor {
	c := &SystemMouseCursor{Kind: kind}
	c.SetSelf(c)
	return c
}

// DebugDescription is "SystemMouseCursor(kind)".
func (c *SystemMouseCursor) DebugDescription() string {
	return foundation.ObjectRuntimeType(c, "SystemMouseCursor") + "(" + c.Kind + ")"
}

// CreateSession creates a session that activates the cursor through
// SystemChannels.MouseCursor.
func (c *SystemMouseCursor) CreateSession(device int) MouseCursorSession {
	return &systemMouseCursorSession{NewMouseCursorSessionBase(c, device)}
}

// Equal is SystemMouseCursor's ==.
func (c *SystemMouseCursor) Equal(other MouseCursor) bool {
	o, ok := other.(*SystemMouseCursor)
	return ok && o != nil && o.Kind == c.Kind
}

// DebugFillProperties adds the kind (at debug level).
func (c *SystemMouseCursor) DebugFillProperties(properties *foundation.DiagnosticPropertiesBuilder) {
	c.MouseCursorBase.DebugFillProperties(properties)
	level := foundation.DiagnosticLevelDebug
	properties.Add(foundation.NewDiagnosticsProperty("kind", c.Kind, foundation.DiagnosticsPropertyOptions{Level: &level}))
}

// systemMouseCursors is the type of SystemMouseCursors (Dart's abstract
// final class of static constants).
type systemMouseCursors struct {
	None *SystemMouseCursor

	// Status.
	Basic, Click, Forbidden, Wait, Progress, ContextMenu, Help *SystemMouseCursor

	// Selection.
	Text, VerticalText, Cell, Precise *SystemMouseCursor

	// Drag-and-drop.
	Move, Grab, Grabbing, NoDrop, Alias, Copy, Disappearing *SystemMouseCursor

	// Resizing and scrolling.
	AllScroll, ResizeLeftRight, ResizeUpDown, ResizeUpLeftDownRight, ResizeUpRightDownLeft,
	ResizeUp, ResizeDown, ResizeLeft, ResizeRight, ResizeUpLeft, ResizeUpRight, ResizeDownLeft,
	ResizeDownRight, ResizeColumn, ResizeRow *SystemMouseCursor

	// Other operations.
	ZoomIn, ZoomOut *SystemMouseCursor
}

// SystemMouseCursors is the collection of system mouse cursors
// (SystemMouseCursors.basic is SystemMouseCursors.Basic...).
var SystemMouseCursors = systemMouseCursors{
	None: newSystemMouseCursor("none"),

	Basic:       newSystemMouseCursor("basic"),
	Click:       newSystemMouseCursor("click"),
	Forbidden:   newSystemMouseCursor("forbidden"),
	Wait:        newSystemMouseCursor("wait"),
	Progress:    newSystemMouseCursor("progress"),
	ContextMenu: newSystemMouseCursor("contextMenu"),
	Help:        newSystemMouseCursor("help"),

	Text:         newSystemMouseCursor("text"),
	VerticalText: newSystemMouseCursor("verticalText"),
	Cell:         newSystemMouseCursor("cell"),
	Precise:      newSystemMouseCursor("precise"),

	Move:         newSystemMouseCursor("move"),
	Grab:         newSystemMouseCursor("grab"),
	Grabbing:     newSystemMouseCursor("grabbing"),
	NoDrop:       newSystemMouseCursor("noDrop"),
	Alias:        newSystemMouseCursor("alias"),
	Copy:         newSystemMouseCursor("copy"),
	Disappearing: newSystemMouseCursor("disappearing"),

	AllScroll:             newSystemMouseCursor("allScroll"),
	ResizeLeftRight:       newSystemMouseCursor("resizeLeftRight"),
	ResizeUpDown:          newSystemMouseCursor("resizeUpDown"),
	ResizeUpLeftDownRight: newSystemMouseCursor("resizeUpLeftDownRight"),
	ResizeUpRightDownLeft: newSystemMouseCursor("resizeUpRightDownLeft"),
	ResizeUp:              newSystemMouseCursor("resizeUp"),
	ResizeDown:            newSystemMouseCursor("resizeDown"),
	ResizeLeft:            newSystemMouseCursor("resizeLeft"),
	ResizeRight:           newSystemMouseCursor("resizeRight"),
	ResizeUpLeft:          newSystemMouseCursor("resizeUpLeft"),
	ResizeUpRight:         newSystemMouseCursor("resizeUpRight"),
	ResizeDownLeft:        newSystemMouseCursor("resizeDownLeft"),
	ResizeDownRight:       newSystemMouseCursor("resizeDownRight"),
	ResizeColumn:          newSystemMouseCursor("resizeColumn"),
	ResizeRow:             newSystemMouseCursor("resizeRow"),

	ZoomIn:  newSystemMouseCursor("zoomIn"),
	ZoomOut: newSystemMouseCursor("zoomOut"),
}
