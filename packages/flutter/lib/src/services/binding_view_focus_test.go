package services_test

// binding_test.dart's "Default handleViewFocusChanged propagates event"
// (Flutter 3.50.1): the binding installs handleViewFocusChanged as
// PlatformDispatcher.onViewFocusChange, and an override sees the event.

import (
	"testing"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/src/async"
)

type viewFocusBinding struct {
	testBinding
	lastFocusEvent *ui.ViewFocusEvent
}

func (b *viewFocusBinding) HandleViewFocusChanged(event ui.ViewFocusEvent) {
	b.testBinding.HandleViewFocusChanged(event)
	b.lastFocusEvent = &event
}

func TestDefaultHandleViewFocusChangedPropagatesEvent(t *testing.T) {
	if !isolated(t) {
		return
	}
	loop := async.NewLoop()
	defer async.SetCurrent(loop)()
	b := &viewFocusBinding{}
	b.SchedulerBindingMixin.SetSelf(b)
	b.ServicesBindingMixin.SetSelf(b)
	b.TestDefaultBinaryMessengerBindingMixin.SetSelf(b)
	b.Construct(b, &b.SchedulerBindingMixin, &b.ServicesBindingMixin, &b.TestDefaultBinaryMessengerBindingMixin)
	loop.FlushMicrotasks()
	event := ui.ViewFocusEvent{ViewID: 0, Direction: ui.ViewFocusDirectionForward, State: ui.ViewFocusStateFocused}
	if cb := ui.PlatformDispatcherInstance().OnViewFocusChange(); cb != nil {
		cb(event)
	}
	if b.lastFocusEvent == nil || *b.lastFocusEvent != event {
		t.Fatalf("lastFocusEvent = %v, want %v", b.lastFocusEvent, event)
	}
}
