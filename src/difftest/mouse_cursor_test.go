package difftest

// Go twin of flutter/test/mouse_cursor_test.dart: MouseCursorManager's
// sessions and messages, SystemMouseCursors' descriptions and
// MouseTrackerAnnotation's diagnostics.

import (
	"regexp"
	"slices"
	"strconv"
	"testing"

	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/gestures"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/services"
	"github.com/jonathanperis/flutter-go/src/async"
)

func TestMouseCursorMatchFlutter(t *testing.T) {
	runChannelScenes(t, "mouse_cursor", mouseCursorScenes)
}

var hashPattern = regexp.MustCompile(`#[0-9a-f]{5}`)

func maskHash(s string) string { return hashPattern.ReplaceAllString(s, "#HASH") }

type _LoggingSession struct {
	services.MouseCursorSessionBase
	l *channelLog
}

func (s *_LoggingSession) Activate() async.Future[async.Void] {
	return async.AsyncVoid(func() {
		s.l.add("activate " + s.Cursor().DebugDescription() + " on " + strconv.Itoa(s.Device()))
	})
}

func (s *_LoggingSession) Dispose() {
	s.l.add("dispose " + s.Cursor().DebugDescription() + " on " + strconv.Itoa(s.Device()))
}

type _LoggingCursor struct {
	services.MouseCursorBase
	name string
	l    *channelLog
}

func newLoggingCursor(name string, l *channelLog) *_LoggingCursor {
	c := &_LoggingCursor{name: name, l: l}
	c.SetSelf(c)
	return c
}

func (c *_LoggingCursor) CreateSession(device int) services.MouseCursorSession {
	return &_LoggingSession{services.NewMouseCursorSessionBase(c, device), c.l}
}

func (c *_LoggingCursor) DebugDescription() string { return "logging(" + c.name + ")" }

func mouseCursorScenes(l *channelLog) {
	codec := services.StandardMethodCodec{}
	testMessenger().SetMockMessageHandler("flutter/mousecursor", func(message []byte) async.Future[[]byte] {
		return async.Async(func() []byte {
			call := codec.DecodeMethodCall(message)
			l.add(call.Method + " [" + bytesList(message) + "]")
			return codec.EncodeSuccessEnvelope(nil)
		})
	})
	showCursor := func(c services.MouseCursor) string {
		if c == nil {
			return "null"
		}
		return c.String()
	}

	l.add("scene manager")
	c := services.SystemMouseCursors
	manager := services.NewMouseCursorManager(c.Basic)
	a := newLoggingCursor("a", l)
	b := newLoggingCursor("b", l)
	update := func(what string, device int, event gestures.PointerEvent, cursors ...services.MouseCursor) {
		l.add("update " + what)
		manager.HandleDeviceCursorUpdate(device, event, slices.Values(cursors))
		l.add("active " + strconv.Itoa(device) + " " + showCursor(manager.DebugDeviceActiveCursor(device)))
	}
	update("defer first", 1, nil, services.MouseCursorDefer, c.Click, c.Text)
	update("same", 1, nil, c.Click)
	update("all defer", 1, nil, services.MouseCursorDefer, services.MouseCursorDefer)
	update("empty", 1, nil)
	update("uncontrolled", 2, nil, services.MouseCursorUncontrolled, c.Grab)
	update("grab", 2, nil, c.Grab)
	update("custom", 2, nil, a)
	update("custom same", 2, gestures.NewPointerHoverEvent(gestures.PointerHoverEventOptions{Device: 2}), a)
	update("custom other", 2, nil, b)
	update("back to system", 2, nil, c.ZoomOut)
	update("removed", 1, gestures.NewPointerRemovedEvent(gestures.PointerRemovedEventOptions{Device: 1}), c.Wait)
	update("after removed", 1, nil, c.Wait)
	update("removed custom", 2, gestures.NewPointerRemovedEvent(gestures.PointerRemovedEventOptions{Device: 2}))
	update("device 7", 7, gestures.NewPointerAddedEvent(gestures.PointerAddedEventOptions{Device: 7}), services.MouseCursorDefer, a, b)
	async.FutureDelayed[any](0, nil).Await()

	l.add("scene uncontrolled fallback")
	uncontrolled := services.NewMouseCursorManager(services.MouseCursorUncontrolled)
	uncontrolled.HandleDeviceCursorUpdate(0, nil, slices.Values([]services.MouseCursor{}))
	l.add("active " + showCursor(uncontrolled.DebugDeviceActiveCursor(0)))
	uncontrolled.HandleDeviceCursorUpdate(0, nil, slices.Values([]services.MouseCursor{c.Help}))
	async.FutureDelayed[any](0, nil).Await()

	l.add("scene system cursors")
	all := []*services.SystemMouseCursor{c.None, c.Basic, c.Click, c.Forbidden, c.Wait, c.Progress, c.ContextMenu, c.Help,
		c.Text, c.VerticalText, c.Cell, c.Precise, c.Move, c.Grab, c.Grabbing, c.NoDrop, c.Alias, c.Copy, c.Disappearing,
		c.AllScroll, c.ResizeLeftRight, c.ResizeUpDown, c.ResizeUpLeftDownRight, c.ResizeUpRightDownLeft, c.ResizeUp,
		c.ResizeDown, c.ResizeLeft, c.ResizeRight, c.ResizeUpLeft, c.ResizeUpRight, c.ResizeDownLeft, c.ResizeDownRight,
		c.ResizeColumn, c.ResizeRow, c.ZoomIn, c.ZoomOut}
	debug, fine, hidden := foundation.DiagnosticLevelDebug, foundation.DiagnosticLevelFine, foundation.DiagnosticLevelHidden
	for _, cursor := range all {
		l.add(cursor.Kind + " | " + cursor.String() + " | " + cursor.DebugDescription() + " | " +
			maskHash(cursor.ToString(foundation.DiagnosticableToStringOptions{MinLevel: &debug})) + " | " +
			maskHash(cursor.ToString(foundation.DiagnosticableToStringOptions{MinLevel: &fine})) + " | " +
			maskHash(cursor.ToStringShort()) + " | " + strconv.FormatBool(cursor.Equal(c.Click)) + " | " +
			strconv.FormatBool(cursor.Equal(c.Text)))
	}
	info := foundation.DiagnosticLevelInfo
	for _, cursor := range []*services.SystemMouseCursor{c.Basic, c.ZoomIn} {
		for _, level := range []*foundation.DiagnosticLevel{&debug, &info} {
			l.add(maskHash(cursor.ToDiagnosticsNode().ToStringDeep(foundation.DiagnosticsNodeToStringDeepOptions{MinLevel: level})))
		}
	}
	for _, cursor := range []services.MouseCursor{services.MouseCursorDefer, services.MouseCursorUncontrolled, a} {
		l.add(cursor.String() + " | " + cursor.DebugDescription() + " | " +
			maskHash(cursor.ToString(foundation.DiagnosticableToStringOptions{MinLevel: &debug})) + " | " +
			maskHash(cursor.ToStringShort()))
	}

	l.add("scene annotations")
	f := false
	for _, annotation := range []*services.MouseTrackerAnnotationBase{
		services.NewMouseTrackerAnnotation(),
		services.NewMouseTrackerAnnotation(services.MouseTrackerAnnotationOptions{OnEnter: func(*gestures.PointerEnterEvent) {}}),
		services.NewMouseTrackerAnnotation(services.MouseTrackerAnnotationOptions{OnExit: func(*gestures.PointerExitEvent) {}}),
		services.NewMouseTrackerAnnotation(services.MouseTrackerAnnotationOptions{OnEnter: func(*gestures.PointerEnterEvent) {},
			OnExit: func(*gestures.PointerExitEvent) {}, Cursor: c.Click}),
		services.NewMouseTrackerAnnotation(services.MouseTrackerAnnotationOptions{Cursor: services.MouseCursorUncontrolled,
			ValidForMouseTracker: &f}),
		services.NewMouseTrackerAnnotation(services.MouseTrackerAnnotationOptions{Cursor: services.MouseCursorDefer}),
		services.NewMouseTrackerAnnotation(services.MouseTrackerAnnotationOptions{Cursor: a}),
	} {
		l.add(maskHash(annotation.String()) + " | " + strconv.FormatBool(annotation.ValidForMouseTracker()) + " | " +
			annotation.Cursor().String())
		l.add(maskHash(annotation.ToStringShort()))
		l.add(maskHash(annotation.ToString(foundation.DiagnosticableToStringOptions{MinLevel: &hidden})))
	}
}
