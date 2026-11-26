package widgets

// Interim single-line text field (flutter-go's own widget, until
// EditableText/TextField are ported). Its state behaves as
// EditableTextState does for the parts ported in editable_text.go: the
// value is a services.TextEditingValue, keys go through the default text
// editing shortcuts to EditableText's actions, and it is a TextInputClient
// of the platform's text input (TextInput.attach on focus, setEditingState
// when the framework changes the value, updateEditingValue/performAction/
// performSelector from the platform).

import (
	"reflect"
	"time"

	"github.com/jonathanperis/flutter-go/src/async"
	"github.com/jonathanperis/flutter-go/src/dart"

	r "github.com/jonathanperis/flutter-go/packages/flutter/lib/src/rendering"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/services"
)

// CursorBlinkHalfPeriod is Flutter's cursor blink interval.
const CursorBlinkHalfPeriod = 500 * time.Millisecond

// KeyTarget is the interim primary focus: a widget receiving keyboard
// input while focused.
type KeyTarget interface {
	// KeyEventHandlers are the onKeyEvent handlers of the focused node and
	// its ancestors, leaf first (see BuildOwner.HandleKeyMessage).
	KeyEventHandlers() []OnKeyEventCallback
	// InsertText commits text as a platform text input plugin does (the
	// selection is replaced and the new value sent with
	// TextInputClient.updateEditingValue).
	InsertText(s string)
	focusChanged(focused bool)
}

// Focus returns the currently focused target, if any.
func (o *BuildOwner) Focus() KeyTarget { return o.focus }

// RequestFocus moves keyboard focus to t (nil clears it).
func (o *BuildOwner) RequestFocus(t KeyTarget) {
	if o.focus == t {
		return
	}
	if o.focus != nil {
		o.focus.focusChanged(false)
	}
	o.focus = t
	if t != nil {
		t.focusChanged(true)
	}
}

// DispatchText sends committed text (typing, IME commit) to the focused
// target, as the platform's text input plugin would.
func (o *BuildOwner) DispatchText(s string) {
	if o.focus != nil {
		o.focus.InsertText(s)
	}
}

// TextEditingController holds the value of a TextField.
type TextEditingController struct {
	value     services.TextEditingValue
	listeners []func()
}

// NewTextEditingController is TextEditingController(text:): the text with
// no selection.
func NewTextEditingController(text string) *TextEditingController {
	return &TextEditingController{value: services.NewTextEditingValue(services.TextEditingValueOptions{Text: text})}
}

func (c *TextEditingController) Value() services.TextEditingValue { return c.value }
func (c *TextEditingController) Text() string                     { return c.value.Text }
func (c *TextEditingController) AddListener(f func())             { c.listeners = append(c.listeners, f) }

// SetValue sets the value and notifies the listeners when it changed
// (ValueNotifier's ==).
func (c *TextEditingController) SetValue(v services.TextEditingValue) {
	if v.Equal(c.value) {
		return
	}
	c.value = v
	for _, f := range c.listeners {
		f()
	}
}

// SetText is the text setter: the selection and composing range are
// cleared.
func (c *TextEditingController) SetText(s string) {
	c.SetValue(services.NewTextEditingValue(services.TextEditingValueOptions{Text: s}))
}

// TextField is a single-line text input.
type TextField struct {
	Controller  *TextEditingController // nil: the field keeps its own
	Style       r.TextStyle
	Autofocus   bool
	ReadOnly    bool
	OnChanged   func(text string)
	OnSubmitted func(text string)
}

func (TextField) CreateState() State { return &textFieldState{} }

type textFieldState struct {
	StateOf[TextField]
	services.TextInputClientMixin
	own          *TextEditingController
	focused      bool
	caretVisible bool
	blinkTimer   *async.Timer
	render       *r.RenderEditable

	actions    map[reflect.Type]Action
	textLayout *editableTextLayout

	connection                      *services.TextInputConnection
	lastKnownRemoteTextEditingValue *services.TextEditingValue
	batchEditDepth                  int
}

func (s *textFieldState) controller() *TextEditingController {
	if c := s.Widget().Controller; c != nil {
		return c
	}
	if s.own == nil {
		s.own = NewTextEditingController("")
	}
	return s.own
}

func (s *textFieldState) InitState() {
	s.actions = s.buildActions()
	s.controller().AddListener(s.didChangeTextEditingValue)
	if s.Widget().Autofocus {
		s.Context().owner.RequestFocus(s)
	}
}

func (s *textFieldState) Dispose() {
	o := s.Context().owner
	if o.focus == KeyTarget(s) {
		o.focus = nil
	}
	s.closeInputConnectionIfNeeded()
	s.stopBlink()
}

// didChangeTextEditingValue is _didChangeTextEditingValue.
func (s *textFieldState) didChangeTextEditingValue() {
	s.updateRemoteEditingValueIfNeeded()
	if s.Context().mounted {
		s.restartBlink()
		s.SetState(nil)
	}
}

// --- KeyTarget

func (s *textFieldState) KeyEventHandlers() []OnKeyEventCallback {
	return DefaultTextEditingShortcutsHandlers(s.findAction)
}

func (s *textFieldState) InsertText(text string) {
	v := s.value()
	sel := v.Selection
	if !sel.IsValid() {
		return
	}
	caret := services.NewTextSelectionCollapsed(sel.Start + dart.StringLength(text))
	newValue := services.NewTextEditingValue(services.TextEditingValueOptions{
		Text:      dart.Concat(sel.TextBefore(v.Text), text, sel.TextAfter(v.Text)),
		Selection: &caret,
	})
	s.UpdateEditingValue(newValue)
}

// focusChanged is _handleFocusChanged.
func (s *textFieldState) focusChanged(focused bool) {
	s.focused = focused
	if focused {
		s.openInputConnection()
		s.restartBlink()
		if !s.value().Selection.IsValid() {
			// Place the cursor at the end if the selection is invalid when
			// the field receives focus.
			s.handleSelectionChanged(services.NewTextSelectionCollapsed(dart.StringLength(s.value().Text)))
		}
	} else {
		s.closeInputConnectionIfNeeded()
		s.stopBlink()
	}
	s.SetState(nil)
}

// handleSelectionChanged is _handleSelectionChanged's controller update.
func (s *textFieldState) handleSelectionChanged(selection services.TextSelection) {
	v := s.value()
	s.controller().SetValue(v.CopyWith(services.TextEditingValueCopyWithOptions{Selection: &selection}))
}

// --- the text input connection.

func (s *textFieldState) openInputConnection() {
	if s.Widget().ReadOnly {
		return
	}
	if s.connection == nil || !s.connection.Attached() {
		localValue := s.value()
		s.connection = services.TextInputAttach(s, services.NewTextInputConfiguration())
		s.connection.SetEditingState(localValue)
		s.connection.Show()
		s.lastKnownRemoteTextEditingValue = &localValue
	} else {
		s.connection.Show()
	}
}

func (s *textFieldState) closeInputConnectionIfNeeded() {
	if s.connection != nil && s.connection.Attached() {
		s.connection.Close()
	}
	s.connection = nil
	s.lastKnownRemoteTextEditingValue = nil
}

// updateRemoteEditingValueIfNeeded is _updateRemoteEditingValueIfNeeded.
func (s *textFieldState) updateRemoteEditingValueIfNeeded() {
	if s.batchEditDepth > 0 || s.connection == nil || !s.connection.Attached() {
		return
	}
	localValue := s.value()
	if s.lastKnownRemoteTextEditingValue != nil && localValue.Equal(*s.lastKnownRemoteTextEditingValue) {
		return
	}
	s.connection.SetEditingState(localValue)
	s.lastKnownRemoteTextEditingValue = &localValue
}

// TextEditingValue is TextSelectionDelegate.textEditingValue.
func (s *textFieldState) TextEditingValue() services.TextEditingValue { return s.value() }

// UserUpdateTextEditingValue is userUpdateTextEditingValue.
func (s *textFieldState) UserUpdateTextEditingValue(value services.TextEditingValue, cause services.SelectionChangedCause) {
	if value.Equal(s.value()) {
		if !s.focused {
			s.Context().owner.RequestFocus(s)
		}
		return
	}
	s.formatAndSetValue(value)
}

// formatAndSetValue is _formatAndSetValue (no input formatters): the value
// is set in a batch edit (one setEditingState at its end) and onChanged is
// called when the text changed.
func (s *textFieldState) formatAndSetValue(value services.TextEditingValue) {
	oldValue := s.value()
	s.batchEditDepth++
	s.controller().SetValue(value)
	if oldValue.Text != value.Text {
		if f := s.Widget().OnChanged; f != nil {
			f(value.Text)
		}
	}
	s.batchEditDepth--
	s.updateRemoteEditingValueIfNeeded()
}

// --- TextInputClient

func (s *textFieldState) CurrentTextEditingValue() *services.TextEditingValue {
	v := s.value()
	return &v
}

func (s *textFieldState) CurrentAutofillScope() services.AutofillScope { return nil }

// UpdateEditingValue is updateEditingValue: the platform's new state.
func (s *textFieldState) UpdateEditingValue(value services.TextEditingValue) {
	if s.Widget().ReadOnly {
		return
	}
	current := s.value()
	// _checkNeedsAdjustAffinity: trust the engine's affinity only if the
	// text or selection changes.
	if value.Text == current.Text && value.Selection.IsCollapsed() == current.Selection.IsCollapsed() &&
		value.Selection.Start == current.Selection.Start && value.Selection.Affinity != current.Selection.Affinity {
		sel := value.Selection.CopyWith(services.TextSelectionCopyWithOptions{Affinity: &current.Selection.Affinity})
		value = value.CopyWith(services.TextEditingValueCopyWithOptions{Selection: &sel})
	}
	s.lastKnownRemoteTextEditingValue = &value
	if value.Equal(current) {
		return
	}
	if value.Text == current.Text && value.Composing == current.Composing {
		// The selection is the only change.
		s.handleSelectionChanged(value.Selection)
	} else {
		s.formatAndSetValue(value)
	}
}

// PerformAction is performAction: a single-line field finalizes editing
// (onSubmitted) for every action.
func (s *textFieldState) PerformAction(action services.TextInputAction) {
	if f := s.Widget().OnSubmitted; f != nil {
		f(s.value().Text)
	}
}

func (s *textFieldState) PerformPrivateCommand(string, map[string]any)         {}
func (s *textFieldState) UpdateFloatingCursor(services.RawFloatingCursorPoint) {}
func (s *textFieldState) ShowAutocorrectionPromptRect(int, int)                {}
func (s *textFieldState) InsertContent(services.KeyboardInsertedContent)       {}
func (s *textFieldState) DidChangeInputControl(_, _ services.TextInputControl) {}

// ConnectionClosed is connectionClosed: the platform closed the input
// connection.
func (s *textFieldState) ConnectionClosed() {
	if s.connection != nil {
		s.connection.ConnectionClosedReceived()
		s.connection = nil
		s.lastKnownRemoteTextEditingValue = nil
	}
}

// --- caret blink: visible on focus/edit, then toggles every 500 ms.

func (s *textFieldState) restartBlink() {
	s.stopBlink()
	s.caretVisible = true
	if s.focused {
		s.scheduleBlink()
	}
}

func (s *textFieldState) scheduleBlink() {
	s.blinkTimer = async.NewTimer(CursorBlinkHalfPeriod, func() {
		s.blinkTimer = nil
		if !s.focused || !s.Context().mounted {
			return
		}
		s.caretVisible = !s.caretVisible
		s.SetState(nil)
		s.scheduleBlink()
	})
}

func (s *textFieldState) stopBlink() {
	if s.blinkTimer != nil {
		s.blinkTimer.Cancel()
		s.blinkTimer = nil
	}
}

func (s *textFieldState) tapAt(global r.Offset) {
	o := s.Context().owner
	o.RequestFocus(s)
	if s.render != nil {
		local := global.Sub(r.GlobalOffset(s.render))
		s.handleSelectionChanged(services.NewTextSelectionCollapsed(s.render.OffsetAt(local)))
	}
}

func (s *textFieldState) Build(*Element) Widget {
	return GestureDetector{
		OnTapAt: s.tapAt,
		Child: editable{
			value: s.value(), style: s.Widget().Style,
			focused: s.focused, caretVisible: s.caretVisible,
			onCreate: func(re *r.RenderEditable) { s.render = re },
		},
	}
}

type editable struct {
	value                 services.TextEditingValue
	style                 r.TextStyle
	focused, caretVisible bool
	onCreate              func(*r.RenderEditable)
}

func (w editable) CreateRenderObject() r.RenderBox {
	re := r.NewEditable(w.style)
	re.Set(w.value, w.style, w.focused, w.caretVisible)
	w.onCreate(re)
	return re
}

func (w editable) UpdateRenderObject(ro r.RenderBox) {
	ro.(*r.RenderEditable).Set(w.value, w.style, w.focused, w.caretVisible)
}
