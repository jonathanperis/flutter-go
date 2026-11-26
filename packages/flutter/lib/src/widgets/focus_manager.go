package widgets

// Partial port of packages/flutter/lib/src/widgets/focus_manager.dart
// (Flutter 3.50.1): KeyEventResult, combineKeyEventResults and
// FocusManager.handleKeyMessage's walk, on the interim focus of BuildOwner
// (one focused KeyTarget; FocusNode/FocusManager are not ported yet).

import "github.com/jonathanperis/flutter-go/packages/flutter/lib/src/services"

// KeyEventResult is the result of a key event handler.
type KeyEventResult int

const (
	// KeyEventResultHandled: the key event has been handled; stop
	// propagating it.
	KeyEventResultHandled KeyEventResult = iota
	// KeyEventResultIgnored: the key event has not been handled; continue
	// propagating it.
	KeyEventResultIgnored
	// KeyEventResultSkipRemainingHandlers: the key event has not been
	// handled, but stop propagating it (the platform may handle it).
	KeyEventResultSkipRemainingHandlers
)

// KeyEventResultValues is KeyEventResult.values.
var KeyEventResultValues = []KeyEventResult{KeyEventResultHandled, KeyEventResultIgnored, KeyEventResultSkipRemainingHandlers}

// Name is the enum value's name.
func (r KeyEventResult) Name() string {
	return [...]string{"handled", "ignored", "skipRemainingHandlers"}[r]
}

func (r KeyEventResult) String() string { return "KeyEventResult." + r.Name() }

// CombineKeyEventResults combines the results of several handlers: handled
// if any is handled, else skipRemainingHandlers if any is, else ignored.
func CombineKeyEventResults(results []KeyEventResult) KeyEventResult {
	hasSkipRemainingHandlers := false
	for _, result := range results {
		switch result {
		case KeyEventResultHandled:
			return KeyEventResultHandled
		case KeyEventResultSkipRemainingHandlers:
			hasSkipRemainingHandlers = true
		}
	}
	if hasSkipRemainingHandlers {
		return KeyEventResultSkipRemainingHandlers
	}
	return KeyEventResultIgnored
}

// OnKeyEventCallback is a focus node's onKeyEvent handler.
type OnKeyEventCallback func(event services.KeyEvent) KeyEventResult

// HandleKeyMessage is FocusManager.handleKeyMessage, the
// KeyEventManager.keyMessageHandler a binding installs: it walks the
// primary focus and its ancestors (the interim focused KeyTarget's
// KeyEventHandlers, leaf first: e.g. the Shortcuts widgets above an
// EditableText), giving each node every event of the message, until one
// handles the message or stops its propagation. It reports whether the
// framework handled it.
func (o *BuildOwner) HandleKeyMessage(message *services.KeyMessage) bool {
	if o.focus == nil {
		return false
	}
	handled := false
	for _, onKeyEvent := range o.focus.KeyEventHandlers() {
		results := make([]KeyEventResult, 0, len(message.Events))
		for _, event := range message.Events {
			results = append(results, onKeyEvent(event))
		}
		result := CombineKeyEventResults(results)
		if result == KeyEventResultIgnored {
			continue
		}
		handled = result == KeyEventResultHandled
		break
	}
	return handled
}
