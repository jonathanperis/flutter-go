package services_test

// binding_test.dart's "initInstances synchronizes keyboard state" (Flutter
// 3.50.1): a binding whose messenger answers getKeyboardState with {1: 1}.

import (
	"testing"

	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/scheduler"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/services"
	flutter_test "github.com/jonathanperis/flutter-go/packages/flutter_test/lib/src"
	"github.com/jonathanperis/flutter-go/src/async"
)

type keyboardStateBinding struct{ testBinding }

func (b *keyboardStateBinding) CreateBinaryMessenger() services.BinaryMessenger {
	keyboardHandler := func([]byte) async.Future[[]byte] {
		return async.Async(func() []byte {
			return services.StandardMethodCodec{}.EncodeSuccessEnvelope(map[int]int{1: 1})
		})
	}
	return flutter_test.NewTestDefaultBinaryMessenger(b.ServicesBindingMixin.CreateBinaryMessenger(),
		flutter_test.TestDefaultBinaryMessengerOptions{OutboundHandlers: map[string]services.MessageHandler{"flutter/keyboard": keyboardHandler}})
}

func TestInitInstancesSynchronizesKeyboardState(t *testing.T) {
	if !isolated(t) {
		return
	}
	loop := async.NewLoop()
	defer async.SetCurrent(loop)()
	b := &keyboardStateBinding{}
	b.SchedulerBindingMixin.SetSelf(b)
	b.ServicesBindingMixin.SetSelf(b)
	b.TestDefaultBinaryMessengerBindingMixin.SetSelf(b)
	b.Construct(b, &b.SchedulerBindingMixin, &b.ServicesBindingMixin, &b.TestDefaultBinaryMessengerBindingMixin)
	for i := 0; i < 5; i++ {
		loop.FlushMicrotasks()
		loop.RunPosted()
	}
	physical := services.HardwareKeyboardInstance().PhysicalKeysPressed()
	logical := services.HardwareKeyboardInstance().LogicalKeysPressed()
	if len(physical) != 1 || len(logical) != 1 || physical[0] != services.NewPhysicalKeyboardKey(1) ||
		logical[0] != services.NewLogicalKeyboardKey(1) {
		t.Errorf("pressed %v %v", physical, logical)
	}
	// Once the state is synchronized, the key data and raw key handlers are
	// installed.
	if b.PlatformDispatcher().OnKeyData() == nil {
		t.Error("onKeyData not set")
	}
	var _ foundation.BindingBase = b
	var _ scheduler.SchedulerBinding = b
}
