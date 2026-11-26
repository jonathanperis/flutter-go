package widgets

// Partial port of packages/flutter/lib/src/widgets/actions.dart (Flutter
// 3.50.1): Intent, Action (isEnabled, invoke, consumesKey,
// toKeyEventResult), CallbackAction, DoNothingAction and the intents
// declared here that the text-editing shortcuts use. The Actions widget,
// ActionDispatcher and the context-based lookup are not ported yet: the
// editing bridge (editable_text.go) keeps EditableText's action map and
// looks actions up in it, which is what Actions.maybeFind finds under an
// EditableText.

import "reflect"

// Intent is an intent bound to an action. An intent type embeds IntentBase.
// Actions are found by the intent's dynamic type (Dart's runtimeType).
type Intent interface{ intent() }

// IntentBase is embedded by every Intent.
type IntentBase struct{}

func (IntentBase) intent() {}

// IntentType is an intent's Type (the key of an action map).
func IntentType(i Intent) reflect.Type { return reflect.TypeOf(i) }

// IntentTypeOf is the Type of the intent type T (Dart: a type literal).
func IntentTypeOf[T Intent]() reflect.Type { return reflect.TypeFor[T]() }

// Action is an action that can be invoked with an Intent. An action type
// embeds ActionBase (the defaults) and overrides what it needs.
type Action interface {
	// IsEnabled is whether the action is enabled for intent (Dart's
	// isEnabled(intent), whose default is isActionEnabled).
	IsEnabled(intent Intent) bool
	// ConsumesKey is whether the key event that invoked the action is
	// consumed (default true).
	ConsumesKey(intent Intent) bool
	// ToKeyEventResult converts the invocation result into the key event
	// result reported to the focus system.
	ToKeyEventResult(intent Intent, invokeResult any) KeyEventResult
	// Invoke performs the action.
	Invoke(intent Intent) any
}

// ActionBase provides Action's defaults; ToKeyEventResult calls
// ConsumesKey through self (SetSelf).
type ActionBase struct{ self Action }

// SetSelf points the base at the action that embeds it.
func (a *ActionBase) SetSelf(self Action) { a.self = self }

// IsEnabled is Action.isEnabled's default (isActionEnabled: true).
func (a *ActionBase) IsEnabled(Intent) bool { return true }

// ConsumesKey is Action.consumesKey's default.
func (a *ActionBase) ConsumesKey(Intent) bool { return true }

// ToKeyEventResult is Action.toKeyEventResult: handled when the key is
// consumed, skipRemainingHandlers otherwise.
func (a *ActionBase) ToKeyEventResult(intent Intent, _ any) KeyEventResult {
	consumes := true
	if a.self != nil {
		consumes = a.self.ConsumesKey(intent)
	}
	if consumes {
		return KeyEventResultHandled
	}
	return KeyEventResultSkipRemainingHandlers
}

// CallbackAction is an Action that invokes a callback.
type CallbackAction[T Intent] struct {
	ActionBase
	// OnInvoke is called when the action is invoked.
	OnInvoke func(intent T) any
}

// NewCallbackAction is CallbackAction(onInvoke:).
func NewCallbackAction[T Intent](onInvoke func(intent T) any) *CallbackAction[T] {
	a := &CallbackAction[T]{OnInvoke: onInvoke}
	a.SetSelf(a)
	return a
}

// Invoke calls OnInvoke.
func (a *CallbackAction[T]) Invoke(intent Intent) any { return a.OnInvoke(intent.(T)) }

// DoNothingAction is an Action that does nothing; it may or may not
// consume the key.
type DoNothingAction struct {
	ActionBase
	consumesKey bool
}

// DoNothingActionOptions are DoNothingAction's named parameters (nil
// ConsumesKey: true).
type DoNothingActionOptions struct{ ConsumesKey *bool }

// NewDoNothingAction is DoNothingAction({consumesKey = true}).
func NewDoNothingAction(opts ...DoNothingActionOptions) *DoNothingAction {
	a := &DoNothingAction{consumesKey: true}
	if len(opts) > 0 && opts[0].ConsumesKey != nil {
		a.consumesKey = *opts[0].ConsumesKey
	}
	a.SetSelf(a)
	return a
}

// ConsumesKey is the constructor's consumesKey.
func (a *DoNothingAction) ConsumesKey(Intent) bool { return a.consumesKey }

// Invoke does nothing.
func (a *DoNothingAction) Invoke(Intent) any { return nil }

// DoNothingIntent is an intent bound to DoNothingAction.
type DoNothingIntent struct{ IntentBase }

// DoNothingAndStopPropagationIntent is an intent that does nothing and
// stops the key event's propagation.
type DoNothingAndStopPropagationIntent struct{ IntentBase }

// DismissIntent is an intent that dismisses the currently focused widget.
type DismissIntent struct{ IntentBase }
