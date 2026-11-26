package flutter_test

// The key event methods of flutter_test's controller.dart (Flutter 3.50.1)
// WidgetController: sendKeyEvent, sendKeyDownEvent, sendKeyUpEvent and
// sendKeyRepeatEvent, on the Tester (the rest of WidgetController is not
// ported).

import (
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/services"
	"github.com/jonathanperis/flutter-go/src/async"
)

// awaitBool runs the test's event loop (microtasks and posted events, no
// time) until f completes, as awaiting it in a test does; it fails the
// test if f never completes, and re-panics its error.
func (t *Tester) awaitBool(f async.Future[bool]) bool {
	t.t.Helper()
	var result bool
	var err error
	done := false
	async.OnComplete(f, func(v bool) { result, done = v, true }, func(e error) { err, done = e, true })
	for i := 0; !done && i < 1000; i++ {
		t.loop.FlushMicrotasks()
		t.loop.RunPosted()
	}
	t.loop.FlushMicrotasks()
	if !done {
		t.t.Fatal("flutter_test: a key event future never completed")
	}
	if err != nil {
		panic(err)
	}
	return result
}

// SendKeyEvent simulates a key down then a key up of key, and returns
// whether the key down was handled (WidgetController.sendKeyEvent; it
// doesn't pump).
func (t *Tester) SendKeyEvent(key services.LogicalKeyboardKey, opts ...SimulateKeyEventOptions) bool {
	t.t.Helper()
	handled := t.awaitBool(SimulateKeyDownEvent(key, opts...))
	t.awaitBool(SimulateKeyUpEvent(key, opts...))
	return handled
}

// SendKeyDownEvent simulates a key down of key.
func (t *Tester) SendKeyDownEvent(key services.LogicalKeyboardKey, opts ...SimulateKeyEventOptions) bool {
	t.t.Helper()
	return t.awaitBool(SimulateKeyDownEvent(key, opts...))
}

// SendKeyUpEvent simulates a key up of key.
func (t *Tester) SendKeyUpEvent(key services.LogicalKeyboardKey, opts ...SimulateKeyEventOptions) bool {
	t.t.Helper()
	return t.awaitBool(SimulateKeyUpEvent(key, opts...))
}

// SendKeyRepeatEvent simulates a key repeat of key.
func (t *Tester) SendKeyRepeatEvent(key services.LogicalKeyboardKey, opts ...SimulateKeyEventOptions) bool {
	t.t.Helper()
	return t.awaitBool(SimulateKeyRepeatEvent(key, opts...))
}
