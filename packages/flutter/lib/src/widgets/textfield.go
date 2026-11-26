package widgets

import (
	"time"

	"github.com/jonathanperis/flutter-go/src/async"

	r "github.com/jonathanperis/flutter-go/packages/flutter/lib/src/rendering"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/services"
)

// CursorBlinkHalfPeriod is Flutter's cursor blink interval.
const CursorBlinkHalfPeriod = 500 * time.Millisecond

// KeyTarget receives keyboard input while focused.
type KeyTarget interface {
	HandleKey(ev services.KeyEvent) bool
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

// DispatchKey sends a key to the focused target.
func (o *BuildOwner) DispatchKey(ev services.KeyEvent) bool {
	if o.focus == nil {
		return false
	}
	return o.focus.HandleKey(ev)
}

// DispatchText sends committed text (typing, IME commit) to the focused target.
func (o *BuildOwner) DispatchText(s string) {
	if o.focus != nil {
		o.focus.InsertText(s)
	}
}

// TextEditingController holds the value of a TextField.
type TextEditingController struct {
	value     services.Value
	listeners []func()
}

func NewTextEditingController(text string) *TextEditingController {
	n := services.Len16(text)
	return &TextEditingController{value: services.Value{Text: text, Selection: services.Collapsed(n), Composing: services.Selection{Base: -1, Extent: -1}}}
}

func (c *TextEditingController) Value() services.Value { return c.value }
func (c *TextEditingController) Text() string          { return c.value.Text }
func (c *TextEditingController) AddListener(f func())  { c.listeners = append(c.listeners, f) }

func (c *TextEditingController) SetValue(v services.Value) {
	if v == c.value {
		return
	}
	c.value = v
	for _, f := range c.listeners {
		f()
	}
}

// SetText replaces the text and puts the caret at the end.
func (c *TextEditingController) SetText(s string) {
	c.SetValue(services.Value{Text: s, Selection: services.Collapsed(services.Len16(s)), Composing: services.Selection{Base: -1, Extent: -1}})
}

// TextField is a single-line text input.
type TextField struct {
	Controller  *TextEditingController // nil: the field keeps its own
	Style       r.TextStyle
	Autofocus   bool
	OnChanged   func(text string)
	OnSubmitted func(text string)
}

func (TextField) CreateState() State { return &textFieldState{} }

type textFieldState struct {
	StateOf[TextField]
	own          *TextEditingController
	focused      bool
	caretVisible bool
	blinkTimer   *async.Timer
	render       *r.RenderEditable
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
	s.controller().AddListener(s.onValueChanged)
	if s.Widget().Autofocus {
		s.Context().owner.RequestFocus(s)
	}
}

func (s *textFieldState) Dispose() {
	o := s.Context().owner
	if o.focus == KeyTarget(s) {
		o.focus = nil
	}
	s.stopBlink()
}

func (s *textFieldState) onValueChanged() {
	if s.Context().mounted {
		s.restartBlink()
		s.SetState(nil)
	}
}

// --- KeyTarget

func (s *textFieldState) focusChanged(focused bool) {
	s.focused = focused
	if focused {
		c := s.controller()
		if c.value.Selection.Base < 0 { // like Flutter: focusing puts the caret at the end
			c.value.Selection = services.Collapsed(c.value.Len())
		}
		s.restartBlink()
	} else {
		s.stopBlink()
	}
	s.SetState(nil)
}

func (s *textFieldState) HandleKey(ev services.KeyEvent) bool {
	o := s.Context().owner
	if ev.Key == services.KeyEnter {
		if f := s.Widget().OnSubmitted; f != nil {
			f(s.controller().Text())
		}
		return true
	}
	v, ok := services.HandleKey(s.controller().Value(), ev, o.Platform, o.Clipboard)
	if ok {
		s.setValue(v)
	}
	return ok
}

func (s *textFieldState) InsertText(text string) {
	s.setValue(services.InsertText(s.controller().Value(), text))
}

func (s *textFieldState) setValue(v services.Value) {
	c := s.controller()
	changed := v.Text != c.value.Text
	c.SetValue(v)
	if changed {
		if f := s.Widget().OnChanged; f != nil {
			f(v.Text)
		}
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
		v := s.controller().Value()
		v.Selection = services.Collapsed(s.render.OffsetAt(local))
		s.controller().SetValue(v)
	}
}

func (s *textFieldState) Build(*Element) Widget {
	return GestureDetector{
		OnTapAt: s.tapAt,
		Child: editable{
			value: s.controller().Value(), style: s.Widget().Style,
			focused: s.focused, caretVisible: s.caretVisible,
			onCreate: func(re *r.RenderEditable) { s.render = re },
		},
	}
}

type editable struct {
	value                 services.Value
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
