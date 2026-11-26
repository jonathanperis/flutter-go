package widgets

// Partial port of packages/flutter/lib/src/widgets/shortcuts.dart (Flutter
// 3.50.1): LockState, ShortcutActivator, SingleActivator, CharacterActivator
// and ShortcutManager's lookup (handleKeypress). The Shortcuts widget,
// LogicalKeySet/KeySet, the registry and the menu serialization are not
// ported yet; the editing bridge calls ShortcutManager.HandleKeypress where
// a Shortcuts widget's focus node would.

import (
	"slices"
	"strings"

	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/services"
	"github.com/jonathanperis/flutter-go/src/dart"
)

func assertionFailed(message string) *dart.AssertionError { return dart.NewAssertionError(message) }

// LockState is the state of a lock key (Num Lock) a SingleActivator
// requires.
type LockState int

const (
	// LockStateIgnored: the lock key state is not checked.
	LockStateIgnored LockState = iota
	// LockStateLocked: the lock key must be locked.
	LockStateLocked
	// LockStateUnlocked: the lock key must be unlocked.
	LockStateUnlocked
)

// LockStateValues is LockState.values.
var LockStateValues = []LockState{LockStateIgnored, LockStateLocked, LockStateUnlocked}

// Name is the enum value's name.
func (s LockState) Name() string { return [...]string{"ignored", "locked", "unlocked"}[s] }

func (s LockState) String() string { return "LockState." + s.Name() }

// ShortcutActivator is an interface to define the keyboard key combination
// to trigger a shortcut.
type ShortcutActivator interface {
	// Triggers are the keys whose key down events may trigger the shortcut
	// (nil: any key, as Dart's null).
	Triggers() []services.LogicalKeyboardKey
	// Accepts is whether the triggering event and the keyboard state
	// trigger the shortcut.
	Accepts(event services.KeyEvent, state *services.HardwareKeyboard) bool
	// DebugDescribeKeys is a description of the key set for debugging.
	DebugDescribeKeys() string
}

var (
	controlSynonyms = services.LogicalKeyboardKeyExpandSynonyms([]services.LogicalKeyboardKey{services.LogicalKeyboardKeyControl})
	shiftSynonyms   = services.LogicalKeyboardKeyExpandSynonyms([]services.LogicalKeyboardKey{services.LogicalKeyboardKeyShift})
	altSynonyms     = services.LogicalKeyboardKeyExpandSynonyms([]services.LogicalKeyboardKey{services.LogicalKeyboardKeyAlt})
	metaSynonyms    = services.LogicalKeyboardKeyExpandSynonyms([]services.LogicalKeyboardKey{services.LogicalKeyboardKeyMeta})
)

func intersects(pressed, set []services.LogicalKeyboardKey) bool {
	for _, k := range pressed {
		if slices.Contains(set, k) {
			return true
		}
	}
	return false
}

// SingleActivator is a shortcut key combination of a single key and
// modifiers. It is a comparable value: two equal activators are Dart's one
// canonicalized const instance (see ShortcutMap for the non-const ones).
type SingleActivator struct {
	// Trigger is the non-modifier key of the shortcut.
	Trigger services.LogicalKeyboardKey
	// Control, Shift, Alt, Meta: whether the modifier (either side) must
	// be held; a modifier that isn't required must not be held.
	Control, Shift, Alt, Meta bool
	// NumLock is whether Num Lock must be locked, unlocked or is ignored.
	NumLock LockState
	// ExcludeRepeats is Dart's includeRepeats negated (so the zero value
	// is the default, true): whether repeat events are ignored.
	ExcludeRepeats bool
}

// SingleActivatorOptions are SingleActivator's named parameters (nil
// IncludeRepeats: true).
type SingleActivatorOptions struct {
	Control, Shift, Alt, Meta bool
	NumLock                   LockState
	IncludeRepeats            *bool
}

// NewSingleActivator is SingleActivator(trigger, {control, shift, alt,
// meta, numLock, includeRepeats}); it asserts that trigger is not a
// modifier key.
func NewSingleActivator(trigger services.LogicalKeyboardKey, opts ...SingleActivatorOptions) SingleActivator {
	var o SingleActivatorOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	switch trigger {
	case services.LogicalKeyboardKeyControl, services.LogicalKeyboardKeyControlLeft, services.LogicalKeyboardKeyControlRight,
		services.LogicalKeyboardKeyShift, services.LogicalKeyboardKeyShiftLeft, services.LogicalKeyboardKeyShiftRight,
		services.LogicalKeyboardKeyAlt, services.LogicalKeyboardKeyAltLeft, services.LogicalKeyboardKeyAltRight,
		services.LogicalKeyboardKeyMeta, services.LogicalKeyboardKeyMetaLeft, services.LogicalKeyboardKeyMetaRight:
		panic(assertionFailed("a SingleActivator's trigger can't be a modifier key"))
	}
	return SingleActivator{Trigger: trigger, Control: o.Control, Shift: o.Shift, Alt: o.Alt, Meta: o.Meta,
		NumLock: o.NumLock, ExcludeRepeats: o.IncludeRepeats != nil && !*o.IncludeRepeats}
}

// IncludeRepeats is whether this activator accepts repeat events.
func (a SingleActivator) IncludeRepeats() bool { return !a.ExcludeRepeats }

// Triggers is the trigger key alone.
func (a SingleActivator) Triggers() []services.LogicalKeyboardKey {
	return []services.LogicalKeyboardKey{a.Trigger}
}

func (a SingleActivator) shouldAcceptModifiers(pressed []services.LogicalKeyboardKey) bool {
	return a.Control == intersects(pressed, controlSynonyms) &&
		a.Shift == intersects(pressed, shiftSynonyms) &&
		a.Alt == intersects(pressed, altSynonyms) &&
		a.Meta == intersects(pressed, metaSynonyms)
}

func (a SingleActivator) shouldAcceptNumLock(state *services.HardwareKeyboard) bool {
	switch a.NumLock {
	case LockStateLocked:
		return slices.Contains(state.LockModesEnabled(), services.KeyboardLockModeNumLock)
	case LockStateUnlocked:
		return !slices.Contains(state.LockModesEnabled(), services.KeyboardLockModeNumLock)
	}
	return true
}

func isDownOrRepeat(event services.KeyEvent, includeRepeats bool) bool {
	switch event.(type) {
	case *services.KeyDownEvent:
		return true
	case *services.KeyRepeatEvent:
		return includeRepeats
	}
	return false
}

// Accepts is whether event (a key down, or a repeat when included) of the
// trigger, with exactly the required modifiers pressed and the Num Lock
// state, triggers the shortcut.
func (a SingleActivator) Accepts(event services.KeyEvent, state *services.HardwareKeyboard) bool {
	return isDownOrRepeat(event, a.IncludeRepeats()) &&
		event.LogicalKey() == a.Trigger &&
		a.shouldAcceptModifiers(state.LogicalKeysPressed()) &&
		a.shouldAcceptNumLock(state)
}

// DebugDescribeKeys is "Control + Alt + Meta + Shift + <key>" (debug
// builds; "" otherwise).
func (a SingleActivator) DebugDescribeKeys() string {
	var keys []string
	if a.Control {
		keys = append(keys, "Control")
	}
	if a.Alt {
		keys = append(keys, "Alt")
	}
	if a.Meta {
		keys = append(keys, "Meta")
	}
	if a.Shift {
		keys = append(keys, "Shift")
	}
	if name := a.Trigger.DebugName(); name != nil {
		keys = append(keys, *name)
	} else {
		keys = append(keys, a.Trigger.ToStringShort())
	}
	return strings.Join(keys, " + ")
}

// CharacterActivator is a shortcut combination that is triggered by a key
// event that produces a specific character.
type CharacterActivator struct {
	// Character is the character the triggering event produces.
	Character string
	// Alt, Control, Meta: whether the modifier must be held (Shift is
	// encoded in the character).
	Alt, Control, Meta bool
	// ExcludeRepeats is Dart's includeRepeats negated.
	ExcludeRepeats bool
}

// Triggers is nil: any key may produce the character.
func (a CharacterActivator) Triggers() []services.LogicalKeyboardKey { return nil }

// Accepts is whether event produces the character with exactly the
// required modifiers.
func (a CharacterActivator) Accepts(event services.KeyEvent, state *services.HardwareKeyboard) bool {
	c := event.Character()
	pressed := state.LogicalKeysPressed()
	return c != nil && *c == a.Character && isDownOrRepeat(event, !a.ExcludeRepeats) &&
		a.Control == intersects(pressed, controlSynonyms) &&
		a.Alt == intersects(pressed, altSynonyms) &&
		a.Meta == intersects(pressed, metaSynonyms)
}

// DebugDescribeKeys is "Alt + Control + Meta + '<character>'".
func (a CharacterActivator) DebugDescribeKeys() string {
	var keys []string
	if a.Alt {
		keys = append(keys, "Alt")
	}
	if a.Control {
		keys = append(keys, "Control")
	}
	if a.Meta {
		keys = append(keys, "Meta")
	}
	return strings.Join(append(keys, "'"+a.Character+"'"), " + ")
}

// ShortcutEntry is one entry of a ShortcutMap.
type ShortcutEntry struct {
	Activator ShortcutActivator
	Intent    Intent
	// NonConst marks an activator Dart builds with `new` (a non-const
	// SingleActivator inside a collection-for): Dart's map keys compare by
	// identity (SingleActivator has no ==), so such a key never replaces
	// an equal one; equal const keys are one canonical instance and do.
	NonConst bool
}

// ShortcutMap is a Dart Map<ShortcutActivator, Intent> literal: ordered,
// a later const key equal to an earlier const key replacing its intent in
// place.
type ShortcutMap []ShortcutEntry

// With is the map literal `{...m, entries}`: a copy of m with entries added
// in order.
func (m ShortcutMap) With(entries ...ShortcutEntry) ShortcutMap {
	out := slices.Clone(m)
	for _, e := range entries {
		replaced := false
		if !e.NonConst {
			for i := range out {
				if !out[i].NonConst && out[i].Activator == e.Activator {
					out[i].Intent = e.Intent
					replaced = true
					break
				}
			}
		}
		if !replaced {
			out = append(out, e)
		}
	}
	return out
}

// ShortcutManager is a ChangeNotifier-less port of Flutter's
// ShortcutManager's lookup: it finds the intent of a key event and invokes
// its action.
type ShortcutManager struct {
	// Shortcuts is the map of activators to intents.
	Shortcuts ShortcutMap
	// Modal: if true, unhandled events stop propagating
	// (skipRemainingHandlers).
	Modal bool
}

// candidates is _getCandidates: the entries indexed under the key (in map
// order), then those whose activator has no triggers.
func (m *ShortcutManager) candidates(key services.LogicalKeyboardKey) []ShortcutEntry {
	var withKey, any []ShortcutEntry
	for _, e := range m.Shortcuts {
		triggers := e.Activator.Triggers()
		if triggers == nil {
			any = append(any, e)
			continue
		}
		if slices.Contains(triggers, key) {
			withKey = append(withKey, e)
		}
	}
	return append(withKey, any...)
}

// Find is _find: the intent of the first candidate accepting the event.
func (m *ShortcutManager) Find(event services.KeyEvent, state *services.HardwareKeyboard) Intent {
	for _, e := range m.candidates(event.LogicalKey()) {
		if e.Activator.Accepts(event, state) {
			return e.Intent
		}
	}
	return nil
}

// HandleKeypress is handleKeypress: the intent the event triggers is
// invoked with the action findAction gives for it (Actions.maybeFind
// from the primary focus), when enabled; the action converts the result.
// Otherwise the event is ignored (skipRemainingHandlers when modal).
func (m *ShortcutManager) HandleKeypress(event services.KeyEvent, findAction func(Intent) Action) KeyEventResult {
	intent := m.Find(event, services.HardwareKeyboardInstance())
	if intent != nil {
		if action := findAction(intent); action != nil && action.IsEnabled(intent) {
			result := action.Invoke(intent)
			return action.ToKeyEventResult(intent, result)
		}
	}
	if m.Modal {
		return KeyEventResultSkipRemainingHandlers
	}
	return KeyEventResultIgnored
}
