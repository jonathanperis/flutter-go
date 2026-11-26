package textedit

import "github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"

// Key identifies a non-character key (typed characters arrive as text).
type Key int

const (
	KeyNone Key = iota
	KeyBackspace
	KeyDelete
	KeyLeft
	KeyRight
	KeyUp
	KeyDown
	KeyHome
	KeyEnd
	KeyEnter
	KeyTab
	KeyA
	KeyC
	KeyV
	KeyX
)

type KeyEvent struct {
	Key                    Key
	Shift, Ctrl, Alt, Meta bool
}

// The platform selects the shortcut map, as Flutter's DefaultTextEditingShortcuts
// does (flutter_test defaults to Android).

// Clipboard is the system clipboard.
type Clipboard interface {
	Get() string
	Set(string)
}

// HandleKey applies the default text-editing shortcut for ev, if any.
func HandleKey(v Value, ev KeyEvent, p foundation.TargetPlatform, clip Clipboard) (Value, bool) {
	word, line := ev.Ctrl, false // modifier for word moves / line moves
	command := ev.Ctrl           // select all, copy, paste...
	if p == foundation.TargetPlatformMacOS || p == foundation.TargetPlatformIOS {
		word, line, command = ev.Alt, ev.Meta, ev.Meta
	}
	extend := ev.Shift
	switch ev.Key {
	case KeyBackspace, KeyDelete:
		fwd := ev.Key == KeyDelete
		switch {
		case line:
			return Delete(v, fwd, ByLine), true
		case word:
			return Delete(v, fwd, ByWord), true
		}
		return Delete(v, fwd, ByCharacter), true
	case KeyLeft, KeyRight:
		fwd := ev.Key == KeyRight
		switch {
		case line:
			return Move(v, fwd, ByLine, !extend), true
		case word:
			return Move(v, fwd, ByWord, !extend), true
		}
		return Move(v, fwd, ByCharacter, !extend), true
	case KeyUp, KeyDown:
		return MoveVertically(v, ev.Key == KeyDown, !extend), true
	case KeyHome, KeyEnd:
		if p == foundation.TargetPlatformMacOS || p == foundation.TargetPlatformIOS {
			return v, false // on Apple platforms Home/End scroll; they don't move the caret
		}
		return Move(v, ev.Key == KeyEnd, ByLine, !extend), true
	case KeyA:
		if command {
			return SelectAll(v), true
		}
	case KeyC, KeyX:
		if command && clip != nil && !v.Selection.IsCollapsed() {
			clip.Set(v.SelectedText())
			if ev.Key == KeyX {
				return Delete(v, false, ByCharacter), true
			}
			return v, true
		}
	case KeyV:
		if command && clip != nil {
			return InsertText(v, clip.Get()), true
		}
	}
	return v, false
}
